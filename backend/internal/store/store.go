// Package store holds the application's own data: accounts, login sessions,
// saved queries, query history, share links and lesson progress.
package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	DB     *sql.DB
	schema string
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id            INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		email         VARCHAR(190) NOT NULL,
		name          VARCHAR(80) NOT NULL,
		password_hash VARCHAR(100) NOT NULL,
		db_status     ENUM('provisioning','ready','resetting','error') NOT NULL DEFAULT 'provisioning',
		db_error      VARCHAR(500) NULL,
		db_reset_at   DATETIME NULL,
		created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen_at  DATETIME NULL,
		UNIQUE KEY uq_users_email (email)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token_hash BINARY(32) NOT NULL PRIMARY KEY,
		user_id    INT UNSIGNED NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		KEY idx_sessions_user (user_id),
		CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	) ENGINE=InnoDB`,
	`CREATE TABLE IF NOT EXISTS saved_queries (
		id         INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		user_id    INT UNSIGNED NOT NULL,
		title      VARCHAR(120) NOT NULL,
		sql_text   MEDIUMTEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		KEY idx_saved_user (user_id, updated_at),
		CONSTRAINT fk_saved_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS query_history (
		id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		user_id     INT UNSIGNED NOT NULL,
		sql_text    MEDIUMTEXT NOT NULL,
		target      ENUM('mine','core') NOT NULL,
		ok          TINYINT(1) NOT NULL,
		error       VARCHAR(500) NULL,
		duration_ms INT UNSIGNED NOT NULL,
		row_count   INT UNSIGNED NULL,
		created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		KEY idx_history_user (user_id, id),
		CONSTRAINT fk_history_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS shared_queries (
		id         CHAR(10) NOT NULL PRIMARY KEY,
		user_id    INT UNSIGNED NULL,
		title      VARCHAR(120) NULL,
		sql_text   MEDIUMTEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		KEY idx_shared_user (user_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS exercise_progress (
		user_id     INT UNSIGNED NOT NULL,
		exercise_id VARCHAR(80) NOT NULL,
		attempts    INT UNSIGNED NOT NULL DEFAULT 0,
		solved_at   DATETIME NULL,
		last_sql    MEDIUMTEXT NULL,
		PRIMARY KEY (user_id, exercise_id),
		CONSTRAINT fk_progress_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
}

// Open creates the application schema if needed and returns a Store using it.
func Open(ctx context.Context, db *sql.DB, schema string) (*Store, error) {
	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+schema+"` CHARACTER SET utf8mb4"); err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "USE `"+schema+"`"); err != nil {
		return nil, err
	}
	for _, m := range migrations {
		if _, err := conn.ExecContext(ctx, m); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	return &Store{DB: db, schema: schema}, nil
}

func (s *Store) t(name string) string { return "`" + s.schema + "`.`" + name + "`" }

// ---------------------------------------------------------------- users

type User struct {
	ID           int        `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	DBStatus     string     `json:"db_status"`
	DBError      string     `json:"db_error,omitempty"`
	DBResetAt    *time.Time `json:"db_reset_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

const userCols = "id, email, name, password_hash, db_status, COALESCE(db_error,''), db_reset_at, created_at"

func scanUser(row interface{ Scan(...any) error }) (*User, error) {
	var u User
	var reset sql.NullTime
	if err := row.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.DBStatus, &u.DBError, &reset, &u.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if reset.Valid {
		u.DBResetAt = &reset.Time
	}
	return &u, nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+s.t("users")).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, email, name, hash string) (*User, error) {
	res, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("users")+" (email, name, password_hash) VALUES (?, ?, ?)", email, name, hash)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.UserByID(ctx, int(id))
}

func (s *Store) UserByID(ctx context.Context, id int) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, "SELECT "+userCols+" FROM "+s.t("users")+" WHERE id = ?", id))
}

func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, "SELECT "+userCols+" FROM "+s.t("users")+" WHERE email = ?", email))
}

func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+userCols+" FROM "+s.t("users")+" ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) SetDBStatus(ctx context.Context, id int, status, errMsg string) error {
	var e any
	if errMsg != "" {
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		e = errMsg
	}
	q := "UPDATE " + s.t("users") + " SET db_status = ?, db_error = ?"
	if status == "ready" {
		q += ", db_reset_at = NOW()"
	}
	_, err := s.DB.ExecContext(ctx, q+" WHERE id = ?", status, e, id)
	return err
}

func (s *Store) SetPassword(ctx context.Context, id int, hash string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE "+s.t("users")+" SET password_hash = ? WHERE id = ?", hash, id)
	return err
}

func (s *Store) DeleteUser(ctx context.Context, id int) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("users")+" WHERE id = ?", id)
	return err
}

// ---------------------------------------------------------------- sessions

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// RandomToken returns a URL-safe random string with n bytes of entropy.
func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Store) CreateSession(ctx context.Context, userID int, ttl time.Duration) (string, error) {
	token := RandomToken(32)
	_, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("sessions")+" (token_hash, user_id, expires_at) VALUES (?, ?, ?)",
		hashToken(token), userID, time.Now().UTC().Add(ttl))
	return token, err
}

// UserBySession returns the user for a valid session token.
func (s *Store) UserBySession(ctx context.Context, token string) (*User, error) {
	q := "SELECT u.id, u.email, u.name, u.password_hash, u.db_status, COALESCE(u.db_error,''), u.db_reset_at, u.created_at FROM " +
		s.t("sessions") + " s JOIN " + s.t("users") + " u ON u.id = s.user_id WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP()"
	return scanUser(s.DB.QueryRowContext(ctx, q, hashToken(token)))
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("sessions")+" WHERE token_hash = ?", hashToken(token))
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int, except string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("sessions")+" WHERE user_id = ? AND token_hash <> ?", userID, hashToken(except))
	return err
}

func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("sessions")+" WHERE expires_at < UTC_TIMESTAMP()")
	return err
}

func (s *Store) TouchUser(ctx context.Context, id int) {
	_, _ = s.DB.ExecContext(ctx, "UPDATE "+s.t("users")+" SET last_seen_at = NOW() WHERE id = ?", id)
}

// ---------------------------------------------------------------- saved queries

type SavedQuery struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	SQL       string    `json:"sql"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) ListSaved(ctx context.Context, userID int) ([]SavedQuery, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id, title, sql_text, created_at, updated_at FROM "+s.t("saved_queries")+" WHERE user_id = ? ORDER BY updated_at DESC, id DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedQuery{}
	for rows.Next() {
		var q SavedQuery
		if err := rows.Scan(&q.ID, &q.Title, &q.SQL, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) SaveQuery(ctx context.Context, userID int, title, sqlText string) (int, error) {
	res, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("saved_queries")+" (user_id, title, sql_text) VALUES (?, ?, ?)", userID, title, sqlText)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func (s *Store) UpdateSaved(ctx context.Context, userID, id int, title, sqlText string) error {
	res, err := s.DB.ExecContext(ctx, "UPDATE "+s.t("saved_queries")+" SET title = ?, sql_text = ? WHERE id = ? AND user_id = ?", title, sqlText, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+s.t("saved_queries")+" WHERE id = ? AND user_id = ?", id, userID).Scan(&exists); err != nil || exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (s *Store) DeleteSaved(ctx context.Context, userID, id int) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("saved_queries")+" WHERE id = ? AND user_id = ?", id, userID)
	return err
}

// ---------------------------------------------------------------- history

type HistoryEntry struct {
	ID         int64     `json:"id"`
	SQL        string    `json:"sql"`
	Target     string    `json:"target"`
	OK         bool      `json:"ok"`
	Error      string    `json:"error,omitempty"`
	DurationMs int       `json:"duration_ms"`
	RowCount   *int      `json:"row_count,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

const historyCap = 500

func (s *Store) AddHistory(ctx context.Context, userID int, e HistoryEntry) error {
	if len(e.SQL) > 65000 {
		e.SQL = e.SQL[:65000]
	}
	if len(e.Error) > 500 {
		e.Error = e.Error[:500]
	}
	var errText any
	if e.Error != "" {
		errText = e.Error
	}
	res, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("query_history")+" (user_id, sql_text, target, ok, error, duration_ms, row_count) VALUES (?, ?, ?, ?, ?, ?, ?)",
		userID, e.SQL, e.Target, e.OK, errText, e.DurationMs, e.RowCount)
	if err != nil {
		return err
	}
	if id, _ := res.LastInsertId(); id%25 == 0 { // trim occasionally
		_, err = s.DB.ExecContext(ctx, "DELETE FROM "+s.t("query_history")+" WHERE user_id = ? AND id < (SELECT id FROM (SELECT id FROM "+s.t("query_history")+" WHERE user_id = ? ORDER BY id DESC LIMIT 1 OFFSET ?) x)", userID, userID, historyCap)
	}
	return err
}

func (s *Store) ListHistory(ctx context.Context, userID int, search string, limit int) ([]HistoryEntry, error) {
	q := "SELECT id, sql_text, target, ok, COALESCE(error,''), duration_ms, row_count, created_at FROM " + s.t("query_history") + " WHERE user_id = ?"
	args := []any{userID}
	if search = strings.TrimSpace(search); search != "" {
		q += " AND sql_text LIKE ?"
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(search)+"%")
	}
	q += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryEntry{}
	for rows.Next() {
		var e HistoryEntry
		var rc sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SQL, &e.Target, &e.OK, &e.Error, &e.DurationMs, &rc, &e.CreatedAt); err != nil {
			return nil, err
		}
		if rc.Valid {
			n := int(rc.Int64)
			e.RowCount = &n
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ClearHistory(ctx context.Context, userID int) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM "+s.t("query_history")+" WHERE user_id = ?", userID)
	return err
}

// ---------------------------------------------------------------- share links

type Shared struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	SQL       string    `json:"sql"`
	CreatedAt time.Time `json:"created_at"`
}

const shareAlphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (s *Store) CreateShare(ctx context.Context, userID int, title, sqlText string) (string, error) {
	var uid any
	if userID > 0 {
		uid = userID
	}
	for attempt := 0; attempt < 5; attempt++ {
		b := make([]byte, 10)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		for i := range b {
			b[i] = shareAlphabet[int(b[i])%len(shareAlphabet)]
		}
		id := string(b)
		_, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("shared_queries")+" (id, user_id, title, sql_text) VALUES (?, ?, ?, ?)", id, uid, title, sqlText)
		if err == nil {
			return id, nil
		}
		if !strings.Contains(err.Error(), "Duplicate") {
			return "", err
		}
	}
	return "", errors.New("could not allocate share id")
}

func (s *Store) GetShare(ctx context.Context, id string) (*Shared, error) {
	var sh Shared
	err := s.DB.QueryRowContext(ctx, "SELECT id, COALESCE(title,''), sql_text, created_at FROM "+s.t("shared_queries")+" WHERE id = ?", id).
		Scan(&sh.ID, &sh.Title, &sh.SQL, &sh.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &sh, err
}

// ---------------------------------------------------------------- lesson progress

type Progress struct {
	ExerciseID string     `json:"exercise_id"`
	Attempts   int        `json:"attempts"`
	SolvedAt   *time.Time `json:"solved_at,omitempty"`
	LastSQL    string     `json:"last_sql,omitempty"`
}

func (s *Store) ListProgress(ctx context.Context, userID int) ([]Progress, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT exercise_id, attempts, solved_at, COALESCE(last_sql,'') FROM "+s.t("exercise_progress")+" WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Progress{}
	for rows.Next() {
		var p Progress
		var solved sql.NullTime
		if err := rows.Scan(&p.ExerciseID, &p.Attempts, &solved, &p.LastSQL); err != nil {
			return nil, err
		}
		if solved.Valid {
			p.SolvedAt = &solved.Time
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RecordAttempt counts an attempt and marks the exercise solved the first time it passes.
func (s *Store) RecordAttempt(ctx context.Context, userID int, exerciseID, sqlText string, solved bool) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("exercise_progress")+" (user_id, exercise_id, attempts, solved_at, last_sql) VALUES (?, ?, 1, IF(?, NOW(), NULL), ?) AS new "+
		"ON DUPLICATE KEY UPDATE attempts = exercise_progress.attempts + 1, solved_at = COALESCE(exercise_progress.solved_at, new.solved_at), last_sql = new.last_sql",
		userID, exerciseID, solved, sqlText)
	return err
}

// ImportSolved marks exercises solved without counting attempts (progress made before signing up).
func (s *Store) ImportSolved(ctx context.Context, userID int, ids []string) error {
	for _, id := range ids {
		if _, err := s.DB.ExecContext(ctx, "INSERT INTO "+s.t("exercise_progress")+" (user_id, exercise_id, attempts, solved_at) VALUES (?, ?, 0, NOW()) "+
			"ON DUPLICATE KEY UPDATE solved_at = COALESCE(solved_at, NOW())", userID, id); err != nil {
			return err
		}
	}
	return nil
}
