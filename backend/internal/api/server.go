// Package api implements the HTTP API used by the frontend.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/darshan260802/sql-playground/backend/internal/config"
	"github.com/darshan260802/sql-playground/backend/internal/lessons"
	"github.com/darshan260802/sql-playground/backend/internal/provision"
	"github.com/darshan260802/sql-playground/backend/internal/store"
)

const sessionCookie = "sqlpg_session"

type Server struct {
	cfg     config.Config
	store   *store.Store
	prov    *provision.Provisioner
	lessons *lessons.Catalog
	checker *lessons.Checker

	sessions *sessionManager
	runs     *runRegistry
	slots    chan struct{} // global concurrency limit for running SQL

	limits struct {
		anonQuery, userQuery, login, signup, share, check *limiter
	}

	schemaMu   sync.Mutex
	coreSchema *SchemaInfo
	usageMu    sync.Mutex
	usageCache map[int]usageEntry
}

type usageEntry struct {
	bytes int64
	at    time.Time
}

func New(cfg config.Config, st *store.Store, prov *provision.Provisioner, cat *lessons.Catalog) *Server {
	s := &Server{
		cfg:        cfg,
		store:      st,
		prov:       prov,
		lessons:    cat,
		checker:    lessons.NewChecker(prov.KillQuery),
		sessions:   newSessionManager(cfg.SessionIdle),
		runs:       newRunRegistry(),
		slots:      make(chan struct{}, cfg.MaxConcurrent),
		usageCache: map[int]usageEntry{},
	}
	s.limits.anonQuery = newLimiter(40, time.Minute)
	s.limits.userQuery = newLimiter(150, time.Minute)
	s.limits.check = newLimiter(60, time.Minute)
	s.limits.login = newLimiter(10, 10*time.Minute)
	s.limits.signup = newLimiter(5, time.Hour)
	s.limits.share = newLimiter(30, time.Hour)
	return s
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/config", s.handleConfig)

	mux.HandleFunc("POST /api/auth/signup", s.handleSignup)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.handleMe)
	mux.HandleFunc("POST /api/account/password", s.requireUser(s.handleChangePassword))
	mux.HandleFunc("POST /api/account/delete", s.requireUser(s.handleDeleteAccount))

	mux.HandleFunc("POST /api/query", s.handleQuery)
	mux.HandleFunc("POST /api/query/cancel", s.handleCancel)
	mux.HandleFunc("POST /api/session/reset", s.requireUser(s.handleSessionReset))
	mux.HandleFunc("GET /api/schema", s.handleSchema)

	mux.HandleFunc("GET /api/database", s.requireUser(s.handleDatabase))
	mux.HandleFunc("POST /api/database/reset", s.requireUser(s.handleDatabaseReset))

	mux.HandleFunc("GET /api/saved", s.requireUser(s.handleListSaved))
	mux.HandleFunc("POST /api/saved", s.requireUser(s.handleCreateSaved))
	mux.HandleFunc("PUT /api/saved/{id}", s.requireUser(s.handleUpdateSaved))
	mux.HandleFunc("DELETE /api/saved/{id}", s.requireUser(s.handleDeleteSaved))
	mux.HandleFunc("GET /api/history", s.requireUser(s.handleListHistory))
	mux.HandleFunc("DELETE /api/history", s.requireUser(s.handleClearHistory))
	mux.HandleFunc("POST /api/share", s.handleCreateShare)
	mux.HandleFunc("GET /api/share/{id}", s.handleGetShare)

	mux.HandleFunc("GET /api/lessons", s.handleLessons)
	mux.HandleFunc("GET /api/lessons/{slug}", s.handleLesson)
	mux.HandleFunc("POST /api/exercises/{id}/check", s.handleCheck)
	mux.HandleFunc("GET /api/exercises/{id}/solution", s.handleSolution)
	mux.HandleFunc("POST /api/progress/import", s.requireUser(s.handleImportProgress))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "No such API endpoint.")
	})
	if s.cfg.StaticDir != "" {
		mux.Handle("/", s.staticHandler())
	}
	return s.middleware(mux)
}

type ctxKey int

const userKey ctxKey = 1

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			// State-changing requests must carry a custom header, which a
			// cross-site form or image request cannot add (CSRF protection).
			if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("X-Requested-With") != "sqlpg" {
				writeError(w, http.StatusForbidden, "csrf", "Missing X-Requested-With header.")
				return
			}
			if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
				if u, err := s.store.UserBySession(r.Context(), c.Value); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), userKey, u))
				}
			}
		}
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" {
			log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func currentUser(r *http.Request) *store.User {
	u, _ := r.Context().Value(userKey).(*store.User)
	return u
}

func (s *Server) requireUser(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r) == nil {
			writeError(w, http.StatusUnauthorized, "auth_required", "Please sign in first.")
			return
		}
		h(w, r)
	}
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------------------------------------------------------------- JSON helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: msg}})
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong on our side. Please try again.")
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", "Request is too large (1 MB max).")
		} else {
			writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body.")
		}
		return false
	}
	return true
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.DB.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_down", "Database unreachable.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"max_rows":        s.cfg.MaxRows,
		"query_timeout_s": int(s.cfg.QueryTimeout.Seconds()),
		"anon_timeout_s":  int(s.cfg.AnonQueryTimeout.Seconds()),
		"user_quota_mb":   s.cfg.UserQuotaMB,
		"signup_open":     n < s.cfg.MaxUsers,
		"core_schema":     s.cfg.CoreSchema,
		"lesson_count":    len(s.lessons.Lessons),
		"exercise_count":  s.lessons.ExerciseCount(),
	})
}

// ---------------------------------------------------------------- static files

func (s *Server) staticHandler() http.Handler {
	root := s.cfg.StaticDir
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean("/" + r.URL.Path)
		full := filepath.Join(root, p)
		if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
			if strings.HasPrefix(p, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			fs.ServeHTTP(w, r)
			return
		}
		// SPA fallback: client-side routes render index.html.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	})
}

// ---------------------------------------------------------------- rate limiting

// limiter allows n events per window per key (a fixed-window counter).
type limiter struct {
	mu     sync.Mutex
	n      int
	window time.Duration
	hits   map[string]*window
}

type window struct {
	start time.Time
	count int
}

func newLimiter(n int, per time.Duration) *limiter {
	return &limiter{n: n, window: per, hits: map[string]*window{}}
}

func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	w := l.hits[key]
	if w == nil || now.Sub(w.start) > l.window {
		if len(l.hits) > 10000 {
			for k, v := range l.hits {
				if now.Sub(v.start) > l.window {
					delete(l.hits, k)
				}
			}
		}
		l.hits[key] = &window{start: now, count: 1}
		return true
	}
	if w.count >= l.n {
		return false
	}
	w.count++
	return true
}
