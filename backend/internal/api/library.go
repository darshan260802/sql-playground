package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/darshan260802/sql-playground/backend/internal/store"
)

const maxSQLBytes = 200_000

type savedRequest struct {
	Title string `json:"title"`
	SQL   string `json:"sql"`
}

func (req *savedRequest) validate(w http.ResponseWriter) bool {
	req.Title = strings.TrimSpace(req.Title)
	switch {
	case req.Title == "" || utf8.RuneCountInString(req.Title) > 120:
		writeError(w, http.StatusBadRequest, "invalid_title", "Give the query a name (up to 120 characters).")
	case strings.TrimSpace(req.SQL) == "":
		writeError(w, http.StatusBadRequest, "empty", "There's no SQL to save.")
	case len(req.SQL) > maxSQLBytes:
		writeError(w, http.StatusBadRequest, "too_large", "That query is too long to save.")
	default:
		return true
	}
	return false
}

func (s *Server) handleListSaved(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListSaved(r.Context(), currentUser(r).ID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": list})
}

func (s *Server) handleCreateSaved(w http.ResponseWriter, r *http.Request) {
	var req savedRequest
	if !readJSON(w, r, &req) || !req.validate(w) {
		return
	}
	id, err := s.store.SaveQuery(r.Context(), currentUser(r).ID, req.Title, req.SQL)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"id": id})
}

func (s *Server) handleUpdateSaved(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Saved query not found.")
		return
	}
	var req savedRequest
	if !readJSON(w, r, &req) || !req.validate(w) {
		return
	}
	if err := s.store.UpdateSaved(r.Context(), currentUser(r).ID, id, req.Title, req.SQL); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Saved query not found.")
			return
		}
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleDeleteSaved(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))
	if err := s.store.DeleteSaved(r.Context(), currentUser(r).ID, id); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleListHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	list, err := s.store.ListHistory(r.Context(), currentUser(r).ID, r.URL.Query().Get("q"), limit)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": list})
}

func (s *Server) handleClearHistory(w http.ResponseWriter, r *http.Request) {
	if err := s.store.ClearHistory(r.Context(), currentUser(r).ID); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleCreateShare(w http.ResponseWriter, r *http.Request) {
	var req savedRequest
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.SQL) == "" || len(req.SQL) > maxSQLBytes {
		writeError(w, http.StatusBadRequest, "invalid_sql", "There's no SQL to share (or it's too long).")
		return
	}
	if !s.limits.share.allow(s.owner(r)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "You've created a lot of links recently. Try again later.")
		return
	}
	title := strings.TrimSpace(req.Title)
	if utf8.RuneCountInString(title) > 120 {
		title = string([]rune(title)[:120])
	}
	uid := 0
	if u := currentUser(r); u != nil {
		uid = u.ID
	}
	id, err := s.store.CreateShare(r.Context(), uid, title, req.SQL)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *Server) handleGetShare(w http.ResponseWriter, r *http.Request) {
	sh, err := s.store.GetShare(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "This shared query doesn't exist.")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sh)
}

// ---------------------------------------------------------------- the user's database

type dbStatus struct {
	Schema     string `json:"schema"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	Done       int    `json:"done"`
	Total      int    `json:"total"`
	UsageBytes int64  `json:"usage_bytes"`
	QuotaBytes int64  `json:"quota_bytes"`
	CoreBytes  int64  `json:"core_bytes"`
	ResetAt    any    `json:"reset_at"`
}

func (s *Server) databaseStatus(ctx context.Context, u *store.User) *dbStatus {
	st := &dbStatus{Schema: s.prov.Schema(u.ID), Status: u.DBStatus, Error: u.DBError, QuotaBytes: s.quotaBytes()}
	if u.DBResetAt != nil {
		st.ResetAt = u.DBResetAt
	}
	if p, ok := s.prov.Progress(u.ID); ok {
		st.Done, st.Total = p.Done, p.Total
	}
	if u.DBStatus == "ready" {
		st.UsageBytes = s.usage(u.ID, false)
	}
	st.CoreBytes, _ = s.prov.CoreSize(ctx)
	return st
}

func (s *Server) handleDatabase(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.databaseStatus(r.Context(), currentUser(r)))
}

func (s *Server) handleDatabaseReset(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u.DBStatus == "provisioning" || u.DBStatus == "resetting" {
		writeError(w, http.StatusConflict, "busy", "Your database is already being prepared.")
		return
	}
	s.ProvisionAsync(u.ID, "resetting")
	u.DBStatus = "resetting"
	writeJSON(w, http.StatusAccepted, s.databaseStatus(r.Context(), u))
}
