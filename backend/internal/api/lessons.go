package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/darshan260802/sql-playground/backend/internal/lessons"
	"github.com/darshan260802/sql-playground/backend/internal/sqlexec"
	"github.com/darshan260802/sql-playground/backend/internal/store"
)

type exerciseView struct {
	*lessons.Exercise
	RequiresLogin bool             `json:"requires_login"`
	Expected      *lessons.Preview `json:"expected,omitempty"`
	Solved        bool             `json:"solved"`
	Attempts      int              `json:"attempts"`
	LastSQL       string           `json:"last_sql,omitempty"`
}

type lessonSummary struct {
	Slug      string   `json:"slug"`
	Title     string   `json:"title"`
	Level     string   `json:"level"`
	Minutes   int      `json:"minutes"`
	Summary   string   `json:"summary"`
	Exercises []string `json:"exercises"`
	Solved    []string `json:"solved"`
}

func (s *Server) progressMap(ctx context.Context, u *store.User) map[string]store.Progress {
	m := map[string]store.Progress{}
	if u == nil {
		return m
	}
	list, err := s.store.ListProgress(ctx, u.ID)
	if err != nil {
		log.Printf("progress: %v", err)
		return m
	}
	for _, p := range list {
		m[p.ExerciseID] = p
	}
	return m
}

func (s *Server) handleLessons(w http.ResponseWriter, r *http.Request) {
	prog := s.progressMap(r.Context(), currentUser(r))
	type moduleView struct {
		Name    string          `json:"name"`
		Lessons []lessonSummary `json:"lessons"`
	}
	mods := []moduleView{}
	for _, m := range s.lessons.Modules {
		mv := moduleView{Name: m.Name, Lessons: []lessonSummary{}}
		for _, l := range m.Lessons {
			ls := lessonSummary{Slug: l.Slug, Title: l.Title, Level: l.Level, Minutes: l.Minutes, Summary: l.Summary, Exercises: []string{}, Solved: []string{}}
			for _, ex := range l.Exercises {
				ls.Exercises = append(ls.Exercises, ex.ID)
				if p, ok := prog[ex.ID]; ok && p.SolvedAt != nil {
					ls.Solved = append(ls.Solved, ex.ID)
				}
			}
			mv.Lessons = append(mv.Lessons, ls)
		}
		mods = append(mods, mv)
	}
	writeJSON(w, http.StatusOK, map[string]any{"modules": mods})
}

func (s *Server) handleLesson(w http.ResponseWriter, r *http.Request) {
	l := s.lessons.Lesson(r.PathValue("slug"))
	if l == nil {
		writeError(w, http.StatusNotFound, "not_found", "Lesson not found.")
		return
	}
	prog := s.progressMap(r.Context(), currentUser(r))
	views := []exerciseView{}
	for _, ex := range l.Exercises {
		v := exerciseView{Exercise: ex, RequiresLogin: ex.RequiresLogin()}
		if ex.Kind == "select" {
			if exp, err := s.checker.Expected(r.Context(), ex, s.readerOpener()); err == nil {
				v.Expected = lessons.PreviewOf(exp, 6)
			} else {
				log.Printf("expected result: %v", err)
			}
		}
		if p, ok := prog[ex.ID]; ok {
			v.Solved = p.SolvedAt != nil
			v.Attempts = p.Attempts
			v.LastSQL = p.LastSQL
		}
		views = append(views, v)
	}
	var prev, next *lessonSummary
	for i, x := range s.lessons.Lessons {
		if x.Slug != l.Slug {
			continue
		}
		if i > 0 {
			p := s.lessons.Lessons[i-1]
			prev = &lessonSummary{Slug: p.Slug, Title: p.Title}
		}
		if i+1 < len(s.lessons.Lessons) {
			n := s.lessons.Lessons[i+1]
			next = &lessonSummary{Slug: n.Slug, Title: n.Title}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lesson": map[string]any{
			"slug": l.Slug, "title": l.Title, "module": l.Module, "level": l.Level, "minutes": l.Minutes,
			"summary": l.Summary, "body": l.Body, "exercises": views, "prev": prev, "next": next,
		},
	})
}

func (s *Server) readerOpener() lessons.Opener {
	return func(ctx context.Context) (*sql.Conn, int64, error) {
		pool, err := s.prov.ReaderPool()
		if err != nil {
			return nil, 0, err
		}
		return s.openConn(ctx, pool, s.checker.Timeout)
	}
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	ex := s.lessons.Exercise(r.PathValue("id"))
	if ex == nil {
		writeError(w, http.StatusNotFound, "not_found", "Exercise not found.")
		return
	}
	var req struct {
		SQL string `json:"sql"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeError(w, http.StatusBadRequest, "empty", "Write a query first.")
		return
	}
	u := currentUser(r)
	if !s.limits.check.allow(s.owner(r)) {
		writeError(w, http.StatusTooManyRequests, "rate_limited", "You're checking answers very quickly. Try again in a minute.")
		return
	}
	if ex.RequiresLogin() {
		if u == nil {
			writeError(w, http.StatusUnauthorized, "auth_required", "This exercise changes data, so it needs your own database copy. Create a free account to try it.")
			return
		}
		if u.DBStatus != "ready" {
			writeError(w, http.StatusConflict, "db_not_ready", "Your database copy is still being prepared. Try again in a few seconds.")
			return
		}
	}

	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	case <-r.Context().Done():
		return
	}

	var out *lessons.Outcome
	var err error
	switch ex.Kind {
	case "select":
		out, err = s.checker.CheckSelect(r.Context(), ex, req.SQL, s.readerOpener())
	case "dml", "ddl":
		opener := func(ctx context.Context) (*sql.Conn, int64, error) {
			pool, err := s.prov.UserPool(u.ID)
			if err != nil {
				return nil, 0, err
			}
			return s.openConn(ctx, pool, s.checker.Timeout)
		}
		if ex.Kind == "dml" {
			out, err = s.checker.CheckDML(r.Context(), ex, req.SQL, opener)
		} else {
			out, err = s.checker.CheckDDL(r.Context(), ex, req.SQL, opener, s.guard(u, "mine"))
		}
	}
	if err != nil {
		internalError(w, fmt.Errorf("check %s: %w", ex.ID, err))
		return
	}
	if out.Error != nil {
		s.hinter(u, map[bool]string{true: "mine", false: "core"}[ex.RequiresLogin()])(out.Error, sqlexec.Statement{SQL: req.SQL})
	}
	if u != nil {
		if err := s.store.RecordAttempt(r.Context(), u.ID, ex.ID, req.SQL, out.Correct); err != nil {
			log.Printf("record attempt: %v", err)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSolution(w http.ResponseWriter, r *http.Request) {
	ex := s.lessons.Exercise(r.PathValue("id"))
	if ex == nil {
		writeError(w, http.StatusNotFound, "not_found", "Exercise not found.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"solution": ex.Solution})
}

func (s *Server) handleImportProgress(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Solved []string `json:"solved"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	var valid []string
	for _, id := range req.Solved {
		if s.lessons.Exercise(id) != nil {
			valid = append(valid, id)
		}
	}
	if err := s.store.ImportSolved(r.Context(), currentUser(r).ID, valid); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": len(valid)})
}

// WarmLessons computes every query exercise's expected result in the
// background, so the first visitor to a lesson doesn't wait for it.
func (s *Server) WarmLessons(ctx context.Context) {
	start := time.Now()
	n := 0
	for _, l := range s.lessons.Lessons {
		for _, ex := range l.Exercises {
			if ex.Kind != "select" {
				continue
			}
			if _, err := s.checker.Expected(ctx, ex, s.readerOpener()); err != nil {
				log.Printf("warm %s: %v", ex.ID, err)
				continue
			}
			n++
		}
	}
	log.Printf("prepared %d expected results in %s", n, time.Since(start).Round(time.Millisecond))
}
