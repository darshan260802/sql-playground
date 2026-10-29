package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darshan260802/sql-playground/backend/internal/sqlexec"
	"github.com/darshan260802/sql-playground/backend/internal/store"
)

// ---------------------------------------------------------------- sticky SQL sessions

// sqlSession is a MySQL connection kept open for one signed-in user and target,
// so variables, temporary tables and open transactions survive between runs.
type sqlSession struct {
	mu       sync.Mutex // held while a run executes
	conn     *sql.Conn
	connID   int64
	lastUsed time.Time
}

type sessionManager struct {
	mu   sync.Mutex
	m    map[string]*sqlSession
	idle time.Duration
}

func newSessionManager(idle time.Duration) *sessionManager {
	sm := &sessionManager{m: map[string]*sqlSession{}, idle: idle}
	go func() {
		for range time.Tick(30 * time.Second) {
			sm.reapIdle()
		}
	}()
	return sm
}

func sessionKey(userID int, target string) string { return fmt.Sprintf("u%d:%s", userID, target) }

func (sm *sessionManager) get(key string) *sqlSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s := sm.m[key]
	if s == nil {
		s = &sqlSession{}
		sm.m[key] = s
	}
	return s
}

func (sm *sessionManager) reapIdle() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for key, s := range sm.m {
		if !s.mu.TryLock() {
			continue // running
		}
		if s.conn != nil && time.Since(s.lastUsed) > sm.idle {
			s.conn.Close()
			s.conn = nil
		}
		if s.conn == nil {
			delete(sm.m, key)
		}
		s.mu.Unlock()
	}
}

// closeUser drops every session of a user (logout, reset, account deletion).
// Running statements are killed by the caller via KillConnections when needed.
func (sm *sessionManager) closeUser(userID int) {
	prefix := fmt.Sprintf("u%d:", userID)
	sm.mu.Lock()
	var victims []*sqlSession
	for key, s := range sm.m {
		if strings.HasPrefix(key, prefix) {
			victims = append(victims, s)
			delete(sm.m, key)
		}
	}
	sm.mu.Unlock()
	for _, s := range victims {
		go func(s *sqlSession) {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.conn != nil {
				s.conn.Close()
				s.conn = nil
			}
		}(s)
	}
}

// ---------------------------------------------------------------- run registry (for Cancel)

type runRegistry struct {
	mu sync.Mutex
	m  map[string]runEntry
}

type runEntry struct {
	owner  string
	cancel context.CancelFunc
}

func newRunRegistry() *runRegistry { return &runRegistry{m: map[string]runEntry{}} }

func (rr *runRegistry) add(id, owner string, cancel context.CancelFunc) {
	if id == "" {
		return
	}
	rr.mu.Lock()
	rr.m[id] = runEntry{owner, cancel}
	rr.mu.Unlock()
}

func (rr *runRegistry) remove(id string) {
	rr.mu.Lock()
	delete(rr.m, id)
	rr.mu.Unlock()
}

func (rr *runRegistry) cancel(id, owner string) bool {
	rr.mu.Lock()
	e, ok := rr.m[id]
	rr.mu.Unlock()
	if !ok || e.owner != owner {
		return false
	}
	e.cancel()
	return true
}

func (s *Server) owner(r *http.Request) string {
	if u := currentUser(r); u != nil {
		return fmt.Sprintf("u%d", u.ID)
	}
	return "ip:" + s.clientIP(r)
}

// ---------------------------------------------------------------- /api/query

type queryRequest struct {
	SQL    string `json:"sql"`
	Target string `json:"target"` // "mine" (default for signed-in users) or "core"
	RunID  string `json:"run_id"`
}

type queryResponse struct {
	Results    []sqlexec.Result `json:"results"`
	DurationMs float64          `json:"duration_ms"`
	Target     string           `json:"target"`
	Database   string           `json:"database"`
	Session    *sessionInfo     `json:"session,omitempty"`
}

type sessionInfo struct {
	InTransaction bool `json:"in_transaction"`
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if !readJSON(w, r, &req) {
		return
	}
	u := currentUser(r)
	target := "core"
	if u != nil && req.Target != "core" {
		target = "mine"
	}
	if u == nil {
		if !s.limits.anonQuery.allow(s.clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "You're running queries very quickly. Take a breath and try again in a minute.")
			return
		}
	} else if !s.limits.userQuery.allow(fmt.Sprint(u.ID)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "You're running queries very quickly. Try again in a minute.")
		return
	}
	if target == "mine" && u.DBStatus != "ready" {
		writeError(w, http.StatusConflict, "db_not_ready", "Your database copy is still being prepared. You can query the original data meanwhile.")
		return
	}
	stmts := sqlexec.Split(req.SQL)
	if len(stmts) == 0 {
		writeError(w, http.StatusBadRequest, "empty", "There's no SQL to run.")
		return
	}
	if len(stmts) > 200 {
		writeError(w, http.StatusBadRequest, "too_many", "A single run can contain at most 200 statements.")
		return
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-time.After(15 * time.Second):
		writeError(w, http.StatusServiceUnavailable, "busy", "The server is busy running other queries. Please try again.")
		return
	case <-r.Context().Done():
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	s.runs.add(req.RunID, s.owner(r), cancel)
	defer s.runs.remove(req.RunID)

	timeout := s.cfg.QueryTimeout
	if u == nil {
		timeout = s.cfg.AnonQueryTimeout
	}
	opt := sqlexec.Options{
		MaxRows:     s.cfg.MaxRows,
		Timeout:     timeout,
		Kill:        s.prov.KillQuery,
		Guard:       s.guard(u, target),
		Hints:       s.hinter(u, target),
		StopOnError: true,
	}
	if target == "mine" {
		opt.Monitor = s.quotaMonitor(u.ID)
	}

	start := time.Now()
	resp := queryResponse{Target: target, Database: s.cfg.CoreSchema}
	if u == nil {
		pool, err := s.prov.ReaderPool()
		if err != nil {
			internalError(w, err)
			return
		}
		conn, connID, err := s.openConn(r.Context(), pool, timeout)
		if err != nil {
			internalError(w, err)
			return
		}
		resp.Results = sqlexec.Run(ctx, conn, connID, stmts, opt)
		conn.Close()
	} else {
		if target == "mine" {
			resp.Database = s.prov.Schema(u.ID)
		}
		sess := s.sessions.get(sessionKey(u.ID, target))
		if !sess.mu.TryLock() {
			writeError(w, http.StatusConflict, "busy_session", "A query is already running in this session. Wait for it or cancel it.")
			return
		}
		defer sess.mu.Unlock()
		if sess.conn == nil {
			var pool *sql.DB
			var err error
			if target == "mine" {
				pool, err = s.prov.UserPool(u.ID)
			} else {
				pool, err = s.prov.UserCorePool(u.ID)
			}
			if err == nil {
				sess.conn, sess.connID, err = s.openConn(r.Context(), pool, timeout)
			}
			if err != nil {
				internalError(w, err)
				return
			}
		}
		sess.lastUsed = time.Now()
		resp.Results = sqlexec.Run(ctx, sess.conn, sess.connID, stmts, opt)
		if sqlexec.IsConnLost(resp.Results) {
			sess.conn.Close()
			sess.conn = nil
		} else {
			resp.Session = &sessionInfo{InTransaction: s.prov.InTransaction(context.Background(), sess.connID)}
		}
	}
	resp.DurationMs = float64(time.Since(start).Microseconds()) / 1000

	if u != nil {
		s.recordHistory(u, target, req.SQL, resp)
	}
	writeJSON(w, http.StatusOK, resp)
}

// openConn takes a dedicated connection and applies session limits.
func (s *Server) openConn(ctx context.Context, pool *sql.DB, timeout time.Duration) (*sql.Conn, int64, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := pool.Conn(cctx)
	if err != nil {
		return nil, 0, err
	}
	for _, q := range []string{
		fmt.Sprintf("SET SESSION max_execution_time = %d", timeout.Milliseconds()),
		"SET SESSION innodb_lock_wait_timeout = 10",
		"SET SESSION lock_wait_timeout = 10",
	} {
		if _, err := conn.ExecContext(cctx, q); err != nil {
			conn.Close()
			return nil, 0, err
		}
	}
	id, err := sqlexec.ConnectionID(cctx, conn)
	if err != nil {
		conn.Close()
		return nil, 0, err
	}
	return conn, id, nil
}

func (s *Server) recordHistory(u *store.User, target, sqlText string, resp queryResponse) {
	e := store.HistoryEntry{SQL: sqlText, Target: target, OK: true, DurationMs: int(resp.DurationMs)}
	for _, res := range resp.Results {
		if res.Error != nil {
			e.OK = false
			e.Error = res.Error.Message
		}
	}
	if n := len(resp.Results); n > 0 && resp.Results[n-1].Columns != nil {
		rc := resp.Results[n-1].RowCount
		e.RowCount = &rc
	}
	if err := s.store.AddHistory(context.Background(), u.ID, e); err != nil {
		log.Printf("history: %v", err)
	}
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID string `json:"run_id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"cancelled": s.runs.cancel(req.RunID, s.owner(r))})
}

// handleSessionReset closes the user's sticky session (rolling back any open transaction).
func (s *Server) handleSessionReset(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	s.sessions.closeUser(u.ID)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ---------------------------------------------------------------- guards, quota, hints

// stripQuoted blanks out quoted strings and comments so keyword checks don't
// trip over data like 'tablespace'.
func stripQuoted(sqlText string) string {
	s := sqlexec.StripComments(sqlText)
	var b strings.Builder
	var q byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q != 0:
			if c == '\\' && q != '`' {
				i++
			} else if c == q {
				q = 0
			}
		case c == '\'' || c == '"' || c == '`':
			q = c
			b.WriteByte(' ')
		default:
			b.WriteByte(c)
		}
	}
	return strings.ToUpper(b.String())
}

func (s *Server) guard(u *store.User, target string) func(sqlexec.Statement) *sqlexec.SQLError {
	return func(st sqlexec.Statement) *sqlexec.SQLError {
		w := sqlexec.Keywords(st.SQL, 2)
		if len(w) == 2 && (w[0] == "CREATE" || w[0] == "DROP") && (w[1] == "DATABASE" || w[1] == "SCHEMA") {
			return &sqlexec.SQLError{Message: "Creating or dropping databases isn't available here.",
				Hint: "Your workspace is exactly one database. To start over, use Reset database in the account menu.", Reason: "blocked"}
		}
		bare := stripQuoted(st.SQL)
		if strings.Contains(bare, "TABLESPACE") || strings.Contains(bare, "DATA DIRECTORY") || strings.Contains(bare, "INDEX DIRECTORY") {
			return &sqlexec.SQLError{Message: "Tablespace and directory options are managed by the playground.",
				Hint: "Remove the TABLESPACE / DATA DIRECTORY clause; tables are stored automatically.", Reason: "blocked"}
		}
		if u != nil && target == "mine" && sqlexec.Classify(st.SQL) == sqlexec.KindWrite {
			if used := s.usage(u.ID, false); used >= s.quotaBytes() {
				return &sqlexec.SQLError{Message: fmt.Sprintf("Your database is using %s of its %d MB limit, so new data can't be added.", mb(used), s.cfg.UserQuotaMB),
					Hint: "DROP tables you no longer need, or reset your database to the original data. Deleting rows frees space for reuse but doesn't shrink the total.", Reason: "quota"}
			}
		}
		return nil
	}
}

func (s *Server) quotaMonitor(userID int) func(sqlexec.Statement) string {
	return func(st sqlexec.Statement) string {
		if sqlexec.Classify(st.SQL) == sqlexec.KindRead {
			return ""
		}
		// allow a little overshoot so a statement that crosses the line mid-way is stopped
		if s.usage(userID, true) > s.quotaBytes()+16<<20 {
			return "quota"
		}
		return ""
	}
}

func (s *Server) hinter(u *store.User, target string) func(*sqlexec.SQLError, sqlexec.Statement) {
	return func(e *sqlexec.SQLError, stmt sqlexec.Statement) {
		if !sqlexec.IsPermissionError(e) {
			return
		}
		db, table := deniedDatabase(e, stmt.SQL)
		if db != "" && !s.usableDatabase(u, db) {
			e.Reason = "unknown_database"
			e.Hint = s.databaseHint(u, db, table)
			return
		}
		switch {
		case u == nil:
			e.Reason = "read_only_anon"
			e.Hint = "You're exploring the read-only original data. Create a free account to get your own full copy where INSERT, UPDATE, DELETE and CREATE all work."
		case target == "core":
			e.Reason = "read_only_core"
			e.Hint = "The original data is read-only. Switch the target to “My database” to make changes."
		case strings.EqualFold(db, s.cfg.CoreSchema):
			e.Hint = fmt.Sprintf("%[1]s is the original data, which is read-only. Leave out “%[1]s.” to change your own copy.", db)
		default:
			e.Hint = "You can only change objects inside your own database."
		}
	}
}

var (
	deniedTableRe = regexp.MustCompile(`(?i)command denied to user .* for table '([^']+)'`)
	deniedDBRe    = regexp.MustCompile(`(?i)access denied for user .* to database '([^']+)'`)
)

// deniedDatabase finds the database a permission error is about: named in the
// message (1044), or written in front of the denied table in the SQL (1142,
// whose message names only the table).
func deniedDatabase(e *sqlexec.SQLError, sql string) (db, table string) {
	if m := deniedDBRe.FindStringSubmatch(e.Message); m != nil {
		return m[1], ""
	}
	m := deniedTableRe.FindStringSubmatch(e.Message)
	if m == nil {
		return "", ""
	}
	table = m[1]
	if i := strings.LastIndexByte(table, '.'); i >= 0 {
		return table[:i], table[i+1:]
	}
	q := regexp.MustCompile("(?i)`?([A-Za-z0-9_$]+)`?\\s*\\.\\s*`?" + regexp.QuoteMeta(table) + "(?:`|\\b)")
	if m := q.FindStringSubmatch(sql); m != nil {
		return m[1], table
	}
	return "", table
}

// usableDatabase reports whether db is one this visitor can read at all.
func (s *Server) usableDatabase(u *store.User, db string) bool {
	switch {
	case strings.EqualFold(db, s.cfg.CoreSchema), strings.EqualFold(db, "information_schema"):
		return true
	case u != nil && strings.EqualFold(db, s.prov.Schema(u.ID)):
		return true
	}
	return false
}

// databaseHint explains that the domains in the schema browser (E-commerce,
// HR, Finance) are groups of tables in one database, not databases.
func (s *Server) databaseHint(u *store.User, db, table string) string {
	if table == "" {
		return fmt.Sprintf("There's no database called “%s” here. E-commerce, HR and Finance are groups of tables inside one database, which is already selected, so write table names on their own, e.g. SELECT * FROM jobs.", db)
	}
	core := s.cfg.CoreSchema
	names := fmt.Sprintf("%s.%s (original data)", core, table)
	if u != nil {
		names = fmt.Sprintf("%s.%s (your copy) or %s", s.prov.Schema(u.ID), table, names)
	}
	return fmt.Sprintf("There's no database called “%s” here. E-commerce, HR and Finance are groups of tables inside one database, so write just %s, or include the database name: %s.", db, table, names)
}

func (s *Server) quotaBytes() int64 { return int64(s.cfg.UserQuotaMB) << 20 }

// usage returns a user's storage use, cached for a few seconds unless fresh is set.
func (s *Server) usage(userID int, fresh bool) int64 {
	s.usageMu.Lock()
	e, ok := s.usageCache[userID]
	s.usageMu.Unlock()
	if ok && !fresh && time.Since(e.at) < 3*time.Second {
		return e.bytes
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	n, err := s.prov.Usage(ctx, userID)
	if err != nil {
		log.Printf("usage: %v", err)
		return e.bytes
	}
	s.usageMu.Lock()
	s.usageCache[userID] = usageEntry{n, time.Now()}
	s.usageMu.Unlock()
	return n
}

func (s *Server) forgetUsage(userID int) {
	s.usageMu.Lock()
	delete(s.usageCache, userID)
	s.usageMu.Unlock()
}

func mb(n int64) string { return fmt.Sprintf("%.0f MB", float64(n)/(1<<20)) }
