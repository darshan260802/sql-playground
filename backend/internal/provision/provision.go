// Package provision manages MySQL accounts and per-user database copies.
//
// Every signed-in learner gets:
//   - a MySQL account pgu_<id> with full rights on their own schema and
//     read-only access to the pristine core schema, and
//   - a schema <prefix><id> holding a copy of core, stored in its own general
//     tablespace so its disk use is easy to measure and to reclaim on reset.
//
// Anonymous visitors use the shared read-only account pg_reader.
//
// MySQL passwords for these accounts are random, kept only in memory and
// rotated every time the server starts.
package provision

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/darshan260802/sql-playground/backend/internal/config"
	"github.com/darshan260802/sql-playground/backend/internal/dataset"
	"github.com/darshan260802/sql-playground/backend/internal/store"
)

const ReaderAccount = "pg_reader"

type Provisioner struct {
	cfg   config.Config
	admin *sql.DB

	mu        sync.Mutex
	passwords map[string]string  // MySQL account -> password
	pools     map[string]*sql.DB // MySQL account -> pool
	progress  map[int]Progress   // user id -> copy progress
	copySem   chan struct{}
}

// Progress of a running copy of the core schema.
type Progress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

func New(cfg config.Config, admin *sql.DB) *Provisioner {
	return &Provisioner{
		cfg:       cfg,
		admin:     admin,
		passwords: map[string]string{},
		pools:     map[string]*sql.DB{},
		progress:  map[int]Progress{},
		copySem:   make(chan struct{}, 2),
	}
}

func (p *Provisioner) Schema(userID int) string  { return fmt.Sprintf("%s%d", p.cfg.UserPrefix, userID) }
func (p *Provisioner) Account(userID int) string { return fmt.Sprintf("pgu_%d", userID) }

// grantName escapes the _ and % wildcards that GRANT treats specially in schema names.
func grantName(schema string) string {
	return "`" + strings.NewReplacer("_", `\_`, "%", `\%`).Replace(schema) + "`"
}

// EnsureCore generates the core dataset when it is missing.
func (p *Provisioner) EnsureCore(ctx context.Context) error {
	var n int
	err := p.admin.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", p.cfg.CoreSchema).Scan(&n)
	if err != nil {
		return err
	}
	if n >= len(dataset.Tables) {
		return nil
	}
	log.Printf("core dataset missing (%d tables found), generating it now", n)
	return dataset.Generate(ctx, p.admin, dataset.Options{
		Schema: p.cfg.CoreSchema,
		Log:    func(f string, a ...any) { log.Printf(f, a...) },
	})
}

// EnsureReader creates or refreshes the anonymous read-only account.
func (p *Provisioner) EnsureReader(ctx context.Context) error {
	pw := store.RandomToken(24)
	core := grantName(p.cfg.CoreSchema)
	stmts := []string{
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", ReaderAccount, pw),
		fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s' WITH MAX_USER_CONNECTIONS %d", ReaderAccount, pw, p.cfg.MaxConcurrent*2+4),
		fmt.Sprintf("GRANT SELECT, SHOW VIEW ON %s.* TO '%s'@'%%'", core, ReaderAccount),
	}
	for _, s := range stmts {
		if _, err := p.admin.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("reader account: %w", err)
		}
	}
	p.setPassword(ReaderAccount, pw)
	return nil
}

// EnsureUserAccount creates or refreshes the MySQL account for a learner.
func (p *Provisioner) EnsureUserAccount(ctx context.Context, userID int) error {
	acct := p.Account(userID)
	pw := store.RandomToken(24)
	mine := grantName(p.Schema(userID))
	stmts := []string{
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", acct, pw),
		fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s' WITH MAX_USER_CONNECTIONS 6", acct, pw),
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE VIEW, SHOW VIEW, "+
			"CREATE ROUTINE, ALTER ROUTINE, EXECUTE, TRIGGER, CREATE TEMPORARY TABLES, LOCK TABLES ON %s.* TO '%s'@'%%'", mine, acct),
		fmt.Sprintf("GRANT SELECT, SHOW VIEW ON %s.* TO '%s'@'%%'", grantName(p.cfg.CoreSchema), acct),
	}
	for _, s := range stmts {
		if _, err := p.admin.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("user account: %w", err)
		}
	}
	p.setPassword(acct, pw)
	return nil
}

func (p *Provisioner) setPassword(acct, pw string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.passwords[acct] = pw
	for key, db := range p.pools {
		if strings.HasPrefix(key, acct+"/") {
			db.Close()
			delete(p.pools, key)
		}
	}
}

// Pool returns a connection pool for a MySQL account. Connections are not kept
// idle, so session state (variables, temp tables) never leaks between runs
// unless a caller deliberately holds on to a *sql.Conn.
func (p *Provisioner) Pool(acct, defaultSchema string) (*sql.DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key := acct + "/" + defaultSchema
	if db := p.pools[key]; db != nil {
		return db, nil
	}
	pw, ok := p.passwords[acct]
	if !ok {
		return nil, fmt.Errorf("no credentials for %s", acct)
	}
	mc := mysql.NewConfig()
	mc.User = acct
	mc.Passwd = pw
	mc.Net = "tcp"
	mc.Addr = p.cfg.MySQLAddr
	mc.DBName = defaultSchema
	mc.AllowNativePasswords = true
	mc.Collation = "utf8mb4_0900_ai_ci"
	mc.Params = map[string]string{"charset": "utf8mb4"}
	mc.MultiStatements = false
	mc.Timeout = 5 * time.Second
	db, err := sql.Open("mysql", mc.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(8)
	p.pools[key] = db
	return db, nil
}

// ReaderPool connects as the anonymous read-only account with core as default schema.
func (p *Provisioner) ReaderPool() (*sql.DB, error) { return p.Pool(ReaderAccount, p.cfg.CoreSchema) }

// UserPool connects as the learner's account with their own schema as default.
func (p *Provisioner) UserPool(userID int) (*sql.DB, error) {
	return p.Pool(p.Account(userID), p.Schema(userID))
}

// UserCorePool connects as the learner's account with core (read-only) as default.
func (p *Provisioner) UserCorePool(userID int) (*sql.DB, error) {
	return p.Pool(p.Account(userID), p.cfg.CoreSchema)
}

// Progress reports a running copy for a user, if any.
func (p *Provisioner) Progress(userID int) (Progress, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pr, ok := p.progress[userID]
	return pr, ok
}

func (p *Provisioner) setProgress(userID int, pr *Progress) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pr == nil {
		delete(p.progress, userID)
	} else {
		p.progress[userID] = *pr
	}
}

// KillConnections terminates every MySQL connection of an account.
func (p *Provisioner) KillConnections(ctx context.Context, acct string) {
	rows, err := p.admin.QueryContext(ctx, "SELECT ID FROM information_schema.PROCESSLIST WHERE USER = ?", acct)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		_, _ = p.admin.ExecContext(ctx, fmt.Sprintf("KILL %d", id))
	}
}

// KillQuery interrupts the statement running on a MySQL connection.
func (p *Provisioner) KillQuery(connID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = p.admin.ExecContext(ctx, fmt.Sprintf("KILL QUERY %d", connID))
}

// CopyCore replaces a user's schema with a fresh copy of core.
func (p *Provisioner) CopyCore(ctx context.Context, userID int) error {
	p.copySem <- struct{}{}
	defer func() { <-p.copySem }()

	// progress is measured in rows so the big tables move the bar proportionally
	weights := map[string]int{}
	total := 0
	if rows, err := p.admin.QueryContext(ctx, "SELECT TABLE_NAME, COALESCE(TABLE_ROWS,0) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", p.cfg.CoreSchema); err == nil {
		for rows.Next() {
			var t string
			var n int
			if rows.Scan(&t, &n) == nil {
				weights[t] = n
			}
		}
		rows.Close()
	}
	for _, t := range dataset.Tables {
		weights[t] = max(weights[t], 1)
		total += weights[t]
	}
	p.setProgress(userID, &Progress{0, total})
	defer p.setProgress(userID, nil)

	p.KillConnections(ctx, p.Account(userID))
	conn, err := p.admin.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	schema := p.Schema(userID)
	ts := dataset.TablespaceName(schema)
	q := "`" + schema + "`"
	if _, err := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS "+q); err != nil {
		return err
	}
	if err := dataset.DropTablespace(ctx, conn, ts); err != nil {
		return err
	}
	for _, s := range []string{
		"CREATE TABLESPACE `" + ts + "` ADD DATAFILE '" + ts + ".ibd' ENGINE=InnoDB",
		"CREATE DATABASE " + q + " CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci",
		"USE " + q,
		"SET SESSION foreign_key_checks = 0",
		"SET SESSION unique_checks = 0",
	} {
		if _, err := conn.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	if err := dataset.ApplySchema(ctx, conn, ts); err != nil {
		return err
	}
	done := 0
	for _, t := range dataset.Tables {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s.`%s` SELECT * FROM `%s`.`%s`", q, t, p.cfg.CoreSchema, t)); err != nil {
			return fmt.Errorf("copy %s: %w", t, err)
		}
		done += weights[t]
		p.setProgress(userID, &Progress{done, total})
	}
	_, err = conn.ExecContext(ctx, "SET SESSION foreign_key_checks = 1")
	return err
}

// DropUser removes a learner's schema, tablespace and MySQL account.
func (p *Provisioner) DropUser(ctx context.Context, userID int) error {
	acct := p.Account(userID)
	p.KillConnections(ctx, acct)
	conn, err := p.admin.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+p.Schema(userID)+"`"); err != nil {
		return err
	}
	if err := dataset.DropTablespace(ctx, conn, dataset.TablespaceName(p.Schema(userID))); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%'", acct)); err != nil {
		return err
	}
	p.mu.Lock()
	for key, db := range p.pools {
		if strings.HasPrefix(key, acct+"/") {
			db.Close()
			delete(p.pools, key)
		}
	}
	delete(p.passwords, acct)
	p.mu.Unlock()
	return nil
}

// Usage returns the bytes a learner's database occupies on disk: their copy's
// tablespace plus any tables they created themselves.
func (p *Provisioner) Usage(ctx context.Context, userID int) (int64, error) {
	schema := p.Schema(userID)
	like := strings.NewReplacer(`\`, `\\`, "_", `\_`, "%", `\%`).Replace(schema) + "/%"
	var n int64
	err := p.admin.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(FILE_SIZE),0) FROM information_schema.INNODB_TABLESPACES WHERE NAME = ? OR NAME LIKE ?",
		dataset.TablespaceName(schema), like).Scan(&n)
	return n, err
}

// CoreSize returns the bytes the core dataset occupies on disk.
func (p *Provisioner) CoreSize(ctx context.Context) (int64, error) {
	var n int64
	err := p.admin.QueryRowContext(ctx, "SELECT COALESCE(SUM(FILE_SIZE),0) FROM information_schema.INNODB_TABLESPACES WHERE NAME = ?",
		dataset.TablespaceName(p.cfg.CoreSchema)).Scan(&n)
	return n, err
}

// InTransaction reports whether a connection has an open (uncommitted) transaction.
func (p *Provisioner) InTransaction(ctx context.Context, connID int64) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var n int
	err := p.admin.QueryRowContext(ctx, `SELECT COUNT(*) FROM performance_schema.events_transactions_current e
		JOIN performance_schema.threads t ON t.THREAD_ID = e.THREAD_ID
		WHERE t.PROCESSLIST_ID = ? AND e.STATE = 'ACTIVE'`, connID).Scan(&n)
	return err == nil && n > 0
}

// Admin exposes the admin pool for read-only metadata queries.
func (p *Provisioner) Admin() *sql.DB { return p.admin }
