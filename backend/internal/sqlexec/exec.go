package sqlexec

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
)

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type SQLError struct {
	Code    uint16 `json:"code,omitempty"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	Line    int    `json:"line,omitempty"` // absolute line in the script
	Reason  string `json:"reason,omitempty"`
}

func (e *SQLError) Error() string { return e.Message }

// Result is the outcome of one statement.
type Result struct {
	Index        int         `json:"index"`
	SQL          string      `json:"sql"`
	Line         int         `json:"line"`
	Columns      []Column    `json:"columns,omitempty"`
	Rows         [][]*string `json:"rows,omitempty"`
	RowCount     int         `json:"row_count"`
	Truncated    bool        `json:"truncated,omitempty"`
	RowsAffected *int64      `json:"rows_affected,omitempty"`
	LastInsertID *int64      `json:"last_insert_id,omitempty"`
	Message      string      `json:"message,omitempty"`
	Warnings     []string    `json:"warnings,omitempty"`
	DurationMs   float64     `json:"duration_ms"`
	Error        *SQLError   `json:"error,omitempty"`
}

// Options control one script execution.
type Options struct {
	MaxRows int
	Timeout time.Duration
	// Kill interrupts the statement running on a MySQL connection id.
	Kill func(connID int64)
	// Guard may reject a statement before it runs.
	Guard func(stmt Statement) *SQLError
	// Monitor is polled every second while a statement runs; returning a
	// reason kills the statement (used to enforce the storage quota).
	Monitor func(stmt Statement) string
	// Hints adds context-specific hints to errors (e.g. "sign in to write").
	Hints func(e *SQLError)
	// StopOnError stops at the first failing statement.
	StopOnError bool
}

// Conn is the subset of *sql.Conn used by the executor.
type Conn interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// ConnectionID returns the MySQL thread id of a connection.
func ConnectionID(ctx context.Context, c Conn) (int64, error) {
	rows, err := c.QueryContext(ctx, "SELECT CONNECTION_ID()")
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var id int64
	if rows.Next() {
		err = rows.Scan(&id)
	}
	return id, err
}

// Run executes the statements of a script on one connection. ctx cancellation
// (for example a user pressing Cancel) interrupts the running statement with
// KILL QUERY instead of dropping the connection, so session state survives.
func Run(ctx context.Context, c Conn, connID int64, stmts []Statement, opt Options) []Result {
	deadline := time.Now().Add(opt.Timeout)
	var out []Result
	for i, st := range stmts {
		res := Result{Index: i, SQL: st.SQL, Line: st.Line}
		if opt.Guard != nil {
			if e := opt.Guard(st); e != nil {
				res.Error = e
				out = append(out, res)
				if opt.StopOnError {
					break
				}
				continue
			}
		}
		remaining := time.Until(deadline)
		if ctx.Err() != nil {
			res.Error = &SQLError{Message: "Not run: the run was cancelled.", Reason: "cancelled"}
			out = append(out, res)
			break
		}
		if remaining <= 0 {
			res.Error = &SQLError{Message: "Not run: the time limit for this run was reached.", Reason: "timeout"}
			out = append(out, res)
			break
		}
		reason := watch(ctx, connID, remaining, st, opt)
		results := execOne(c, st, opt.MaxRows)
		stop := reason()
		for k := range results {
			r := &results[k]
			r.Index, r.SQL, r.Line = i, st.SQL, st.Line
			if r.Error != nil {
				r.Error.Line = errorLine(r.Error, st.Line)
				if stop != "" {
					describeStop(r.Error, stop, opt.Timeout)
				}
				addHint(r.Error)
				if opt.Hints != nil {
					opt.Hints(r.Error)
				}
			}
		}
		if last := &results[len(results)-1]; stop != "" && last.Error == nil {
			// e.g. SLEEP() returns normally when interrupted
			last.Error = &SQLError{Line: st.Line}
			describeStop(last.Error, stop, opt.Timeout)
		}
		out = append(out, results...)
		failed := results[len(results)-1].Error != nil
		if failed && (opt.StopOnError || stop != "" || isConnLost(results[len(results)-1].Error)) {
			break
		}
	}
	return out
}

// watch starts a watchdog for one statement and returns a function that stops
// it and reports why (if) the statement was interrupted.
func watch(ctx context.Context, connID int64, limit time.Duration, st Statement, opt Options) func() string {
	done := make(chan struct{})
	var mu sync.Mutex
	why := ""
	kill := func(reason string) {
		mu.Lock()
		why = reason
		mu.Unlock()
		if opt.Kill != nil {
			opt.Kill(connID)
		}
	}
	go func() {
		timer := time.NewTimer(limit)
		defer timer.Stop()
		var tick <-chan time.Time
		if opt.Monitor != nil {
			t := time.NewTicker(time.Second)
			defer t.Stop()
			tick = t.C
		}
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				kill("timeout")
				return
			case <-ctx.Done():
				kill("cancelled")
				return
			case <-tick:
				if r := opt.Monitor(st); r != "" {
					kill(r)
					return
				}
			}
		}
	}()
	return func() string {
		close(done)
		mu.Lock()
		defer mu.Unlock()
		return why
	}
}

func describeStop(e *SQLError, reason string, limit time.Duration) {
	e.Reason = reason
	switch reason {
	case "timeout":
		e.Message = fmt.Sprintf("Query stopped: it ran longer than the %s limit.", limit.Round(time.Second))
		e.Hint = "Add a WHERE filter or LIMIT, or check for a join missing its ON condition (which multiplies rows)."
	case "cancelled":
		e.Message = "Query cancelled."
		e.Hint = ""
	case "quota":
		e.Message = "Query stopped: your database reached its storage limit."
		e.Hint = "Drop tables you no longer need, or reset your database to the original data."
	}
}

// returnsRows reports whether a statement produces a result set.
func returnsRows(sqlText string) bool {
	w := Keywords(sqlText, 1)
	if len(w) == 0 {
		return false
	}
	switch w[0] {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "TABLE", "VALUES", "CALL", "CHECK", "ANALYZE",
		"OPTIMIZE", "REPAIR", "CHECKSUM", "HELP":
		return true
	case "WITH":
		return Classify(sqlText) == KindRead
	}
	return false
}

func execOne(c Conn, st Statement, maxRows int) []Result {
	// Driver calls never see the request context: cancelling would drop the
	// connection mid-query. The watchdog interrupts with KILL QUERY instead.
	ctx := context.Background()
	start := time.Now()
	ms := func() float64 { return float64(time.Since(start).Microseconds()) / 1000 }

	if !returnsRows(st.SQL) {
		res, err := c.ExecContext(ctx, st.SQL)
		r := Result{DurationMs: ms()}
		if err != nil {
			r.Error = toSQLError(err)
			return []Result{r}
		}
		if n, err := res.RowsAffected(); err == nil {
			r.RowsAffected = &n
			r.Message = fmt.Sprintf("Query OK, %d %s affected", n, plural(n, "row", "rows"))
		}
		if id, err := res.LastInsertId(); err == nil && id > 0 {
			r.LastInsertID = &id
		}
		r.Warnings = warnings(c)
		return []Result{r}
	}

	rows, err := c.QueryContext(ctx, st.SQL)
	if err != nil {
		return []Result{{DurationMs: ms(), Error: toSQLError(err)}}
	}
	defer rows.Close()
	var out []Result
	for {
		r := readResultSet(rows, maxRows)
		r.DurationMs = ms()
		if r.Error == nil && len(r.Columns) == 0 {
			r.Message = "Query OK"
		}
		out = append(out, r)
		if r.Error != nil || !rows.NextResultSet() {
			break
		}
	}
	if err := rows.Err(); err != nil && out[len(out)-1].Error == nil {
		out[len(out)-1].Error = toSQLError(err)
	}
	// CALL ends with an empty status result; drop it when real results came first.
	if len(out) > 1 && len(out[len(out)-1].Columns) == 0 && out[len(out)-1].Error == nil {
		out = out[:len(out)-1]
	}
	rows.Close()
	if out[len(out)-1].Error == nil {
		out[len(out)-1].Warnings = warnings(c)
	}
	return out
}

var binaryTypes = map[string]bool{"BINARY": true, "VARBINARY": true, "BLOB": true, "TINYBLOB": true, "MEDIUMBLOB": true, "LONGBLOB": true, "BIT": true, "GEOMETRY": true}

func readResultSet(rows *sql.Rows, maxRows int) Result {
	var r Result
	cts, err := rows.ColumnTypes()
	if err != nil {
		r.Error = toSQLError(err)
		return r
	}
	r.Columns = make([]Column, len(cts))
	binary := make([]bool, len(cts))
	for i, ct := range cts {
		r.Columns[i] = Column{Name: ct.Name(), Type: ct.DatabaseTypeName()}
		binary[i] = binaryTypes[ct.DatabaseTypeName()]
	}
	if len(cts) == 0 {
		return r
	}
	raw := make([]sql.RawBytes, len(cts))
	dest := make([]any, len(cts))
	for i := range raw {
		dest[i] = &raw[i]
	}
	r.Rows = [][]*string{}
	for rows.Next() {
		if r.RowCount >= maxRows {
			r.Truncated = true
			r.RowCount++
			continue // keep counting (the driver drains the set anyway)
		}
		if err := rows.Scan(dest...); err != nil {
			r.Error = toSQLError(err)
			return r
		}
		row := make([]*string, len(raw))
		for i, b := range raw {
			if b == nil {
				continue
			}
			var s string
			if binary[i] || !utf8.Valid(b) {
				s = "0x" + strings.ToUpper(hex.EncodeToString(b))
			} else {
				s = string(b)
			}
			row[i] = &s
		}
		r.Rows = append(r.Rows, row)
		r.RowCount++
	}
	if err := rows.Err(); err != nil {
		r.Error = toSQLError(err)
	}
	return r
}

func warnings(c Conn) []string {
	rows, err := c.QueryContext(context.Background(), "SHOW WARNINGS LIMIT 10")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var level, msg string
		var code int
		if rows.Scan(&level, &code, &msg) == nil {
			out = append(out, fmt.Sprintf("%s %d: %s", level, code, msg))
		}
	}
	return out
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func toSQLError(err error) *SQLError {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return &SQLError{Code: me.Number, Message: me.Message}
	}
	if errors.Is(err, mysql.ErrInvalidConn) || errors.Is(err, sql.ErrConnDone) || strings.Contains(err.Error(), "bad connection") {
		return &SQLError{Message: "The database connection was lost.", Reason: "connection", Hint: "Run the query again; a new session will be opened."}
	}
	return &SQLError{Message: err.Error()}
}

func isConnLost(e *SQLError) bool { return e != nil && e.Reason == "connection" }

// IsConnLost reports whether a result failed because the connection broke.
func IsConnLost(results []Result) bool {
	for _, r := range results {
		if isConnLost(r.Error) {
			return true
		}
	}
	return false
}

var lineRE = regexp.MustCompile(`at line (\d+)\s*$`)

func errorLine(e *SQLError, stmtLine int) int {
	if m := lineRE.FindStringSubmatch(e.Message); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return stmtLine + n - 1
		}
	}
	return stmtLine
}

var hints = map[uint16]string{
	1064: "Check the SQL right before the quoted text: a missing comma, a misspelled keyword or an unclosed quote are the usual suspects.",
	1146: "That table doesn't exist here. Browse the schema panel to see the available tables.",
	1054: "Check the column name, and the table alias in front of it. The schema panel lists every column.",
	1052: "More than one table has a column with that name. Prefix it with a table name or alias, e.g. o.customer_id.",
	1055: "With ONLY_FULL_GROUP_BY (MySQL's default), every selected column must be listed in GROUP BY or wrapped in an aggregate such as MAX().",
	1140: "You're mixing aggregates (SUM, COUNT, ...) with plain columns. Add a GROUP BY for the plain columns.",
	1111: "Aggregates can't be used in WHERE. Filter groups with HAVING, or compute the aggregate in a subquery or CTE.",
	1242: "The subquery returned more than one row where a single value was expected. Use IN instead of =, or make it return one value (an aggregate or LIMIT 1).",
	1248: "Every derived table (a subquery in FROM) needs an alias: FROM (SELECT ...) AS t.",
	1062: "That value already exists in a PRIMARY KEY or UNIQUE column.",
	1451: "Other rows still reference this row through a foreign key. Update or delete those child rows first.",
	1452: "The referenced parent row doesn't exist, so the foreign key check failed.",
	1093: "MySQL can't read the table you're modifying in a subquery. Wrap it in a derived table: WHERE id IN (SELECT id FROM (SELECT ...) AS t).",
	1050: "Use CREATE TABLE IF NOT EXISTS, or DROP TABLE first.",
	1051: "Use DROP TABLE IF EXISTS to ignore missing tables.",
	1048: "This column is declared NOT NULL, so it needs a value.",
	1364: "This column has no default value, so the INSERT must provide one.",
	1406: "The value is longer than the column allows.",
	1366: "The value doesn't fit the column's type.",
	1205: "Another session holds a lock on these rows. Did you leave a transaction open? Run COMMIT or ROLLBACK.",
	3636: "The recursive CTE hit @@cte_max_recursion_depth (1000 by default). Check its stopping condition, or raise it: SET SESSION cte_max_recursion_depth = 10000;",
	1235: "This MySQL version doesn't support that syntax in this position.",
	1349: "A view's SELECT can't contain that construct.",
	1415: "Functions can't return result sets; use a procedure, or SELECT ... INTO a variable.",
}

func addHint(e *SQLError) {
	if e.Hint == "" {
		e.Hint = hints[e.Code]
	}
}

// IsPermissionError reports MySQL "access denied" style errors.
func IsPermissionError(e *SQLError) bool {
	switch e.Code {
	case 1142, 1143, 1044, 1045, 1227, 1370, 1410, 1449, 1095:
		return true
	}
	return false
}
