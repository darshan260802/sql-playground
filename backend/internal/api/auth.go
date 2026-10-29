package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/darshan260802/sql-playground/backend/internal/store"
)

var dummyHash = sync.OnceValue(func() string {
	h, _ := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), 12)
	return string(h)
})

type meResponse struct {
	User     *store.User `json:"user"`
	Database *dbStatus   `json:"database,omitempty"`
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int) error {
	ttl := time.Duration(s.cfg.SessionDays) * 24 * time.Hour
	token, err := s.store.CreateSession(r.Context(), userID, ttl)
	if err != nil {
		return err
	}
	s.setSessionCookie(w, token, int(ttl.Seconds()))
	return nil
}

func normalizeEmail(e string) (string, bool) {
	e = strings.ToLower(strings.TrimSpace(e))
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e || len(e) > 190 || !strings.Contains(e[strings.LastIndex(e, "@"):], ".") {
		return "", false
	}
	return e, true
}

func validPassword(pw string) string {
	switch {
	case utf8.RuneCountInString(pw) < 8:
		return "Use at least 8 characters for your password."
	case len(pw) > 72:
		return "Passwords can be at most 72 bytes long."
	}
	return ""
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !s.limits.signup.allow(s.clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many sign-ups from your network. Try again later.")
		return
	}
	name := strings.TrimSpace(req.Name)
	email, ok := normalizeEmail(req.Email)
	switch {
	case name == "" || utf8.RuneCountInString(name) > 80:
		writeError(w, http.StatusBadRequest, "invalid_name", "Please enter your name (up to 80 characters).")
		return
	case !ok:
		writeError(w, http.StatusBadRequest, "invalid_email", "Please enter a valid email address.")
		return
	}
	if msg := validPassword(req.Password); msg != "" {
		writeError(w, http.StatusBadRequest, "weak_password", msg)
		return
	}
	n, err := s.store.CountUsers(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	if n >= s.cfg.MaxUsers {
		writeError(w, http.StatusForbidden, "signup_closed", "This playground is full: all accounts are taken. Ask the admin for a spot.")
		return
	}
	if _, err := s.store.UserByEmail(r.Context(), email); err == nil {
		writeError(w, http.StatusConflict, "email_taken", "An account with this email already exists. Try signing in.")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		internalError(w, err)
		return
	}
	u, err := s.store.CreateUser(r.Context(), email, name, string(hash))
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			writeError(w, http.StatusConflict, "email_taken", "An account with this email already exists. Try signing in.")
			return
		}
		internalError(w, err)
		return
	}
	if err := s.prov.EnsureUserAccount(r.Context(), u.ID); err != nil {
		internalError(w, err)
		return
	}
	s.ProvisionAsync(u.ID, "provisioning")
	if err := s.startSession(w, r, u.ID); err != nil {
		internalError(w, err)
		return
	}
	u.DBStatus = "provisioning"
	writeJSON(w, http.StatusCreated, meResponse{User: u, Database: s.databaseStatus(r.Context(), u)})
}

// ProvisionAsync copies the core dataset into a user's schema in the background.
func (s *Server) ProvisionAsync(userID int, status string) {
	ctx := context.Background()
	if err := s.store.SetDBStatus(ctx, userID, status, ""); err != nil {
		log.Printf("set db status: %v", err)
	}
	s.sessions.closeUser(userID)
	go func() {
		start := time.Now()
		err := s.prov.CopyCore(ctx, userID)
		if err != nil {
			log.Printf("copy core for user %d: %v", userID, err)
			_ = s.store.SetDBStatus(ctx, userID, "error", err.Error())
			return
		}
		s.forgetUsage(userID)
		_ = s.store.SetDBStatus(ctx, userID, "ready", "")
		log.Printf("database for user %d ready in %s", userID, time.Since(start).Round(time.Millisecond))
	}()
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	ip := s.clientIP(r)
	if !s.limits.login.allow(ip) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many sign-in attempts. Wait a few minutes and try again.")
		return
	}
	email, _ := normalizeEmail(req.Email)
	u, err := s.store.UserByEmail(r.Context(), email)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		internalError(w, err)
		return
	}
	hash := dummyHash() // equalize timing for unknown emails
	if u != nil {
		hash = u.PasswordHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil || u == nil {
		writeError(w, http.StatusUnauthorized, "bad_credentials", "Email or password is incorrect.")
		return
	}
	if err := s.startSession(w, r, u.ID); err != nil {
		internalError(w, err)
		return
	}
	s.store.TouchUser(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, meResponse{User: u, Database: s.databaseStatus(r.Context(), u)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), c.Value)
	}
	if u := currentUser(r); u != nil {
		s.sessions.closeUser(u.ID)
	}
	s.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeJSON(w, http.StatusOK, meResponse{})
		return
	}
	writeJSON(w, http.StatusOK, meResponse{User: u, Database: s.databaseStatus(r.Context(), u)})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Current)) != nil {
		writeError(w, http.StatusUnauthorized, "bad_credentials", "Your current password is incorrect.")
		return
	}
	if msg := validPassword(req.New); msg != "" {
		writeError(w, http.StatusBadRequest, "weak_password", msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.New), 12)
	if err != nil {
		internalError(w, err)
		return
	}
	if err := s.store.SetPassword(r.Context(), u.ID, string(hash)); err != nil {
		internalError(w, err)
		return
	}
	c, _ := r.Cookie(sessionCookie)
	_ = s.store.DeleteUserSessions(r.Context(), u.ID, c.Value) // sign out other devices
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "bad_credentials", "Password is incorrect.")
		return
	}
	s.sessions.closeUser(u.ID)
	if err := s.prov.DropUser(r.Context(), u.ID); err != nil {
		internalError(w, err)
		return
	}
	if err := s.store.DeleteUser(r.Context(), u.ID); err != nil {
		internalError(w, err)
		return
	}
	s.setSessionCookie(w, "", -1)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
