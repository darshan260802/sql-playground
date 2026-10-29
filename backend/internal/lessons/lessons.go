// Package lessons loads the curriculum and checks exercise answers.
//
// Each lesson is a Markdown file in content/ named NN-slug.md:
//
//	title: Filtering rows
//	module: Basics
//	level: Beginner
//	minutes: 12
//	summary: Keep only the rows you need with WHERE.
//	---
//	Markdown body...
//
//	=== exercise
//	id: where-1
//	title: Expensive products
//	kind: select            (select | dml | ddl; default select)
//	ordered: true           (row order matters; default false)
//	--- prompt
//	...
//	--- solution
//	...
//	--- hint
//	...
//	--- check               (dml/ddl: query whose result is compared)
//	--- setup               (ddl: cleanup run before and after)
//	===
package lessons

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed content/*.md
var contentFS embed.FS

type Lesson struct {
	Slug      string      `json:"slug"`
	Order     int         `json:"order"`
	Title     string      `json:"title"`
	Module    string      `json:"module"`
	Level     string      `json:"level"`
	Minutes   int         `json:"minutes"`
	Summary   string      `json:"summary"`
	Body      string      `json:"body"`
	Exercises []*Exercise `json:"exercises"`
}

type Exercise struct {
	ID       string `json:"id"`
	Lesson   string `json:"lesson"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Ordered  bool   `json:"ordered"`
	Prompt   string `json:"prompt"`
	Hint     string `json:"hint,omitempty"`
	Solution string `json:"-"`
	Check    string `json:"-"`
	Setup    string `json:"-"`
}

// RequiresLogin reports whether the exercise modifies data (needs the learner's own copy).
func (e *Exercise) RequiresLogin() bool { return e.Kind == "dml" || e.Kind == "ddl" }

type Module struct {
	Name    string    `json:"name"`
	Lessons []*Lesson `json:"lessons"`
}

type Catalog struct {
	Lessons   []*Lesson
	Modules   []*Module
	bySlug    map[string]*Lesson
	exercises map[string]*Exercise
}

func (c *Catalog) Lesson(slug string) *Lesson         { return c.bySlug[slug] }
func (c *Catalog) Exercise(id string) *Exercise       { return c.exercises[id] }
func (c *Catalog) ExerciseCount() int                 { return len(c.exercises) }
func (c *Catalog) AllExercises() map[string]*Exercise { return c.exercises }

// Load parses the embedded curriculum.
func Load() (*Catalog, error) {
	return loadFS(contentFS, "content")
}

func loadFS(fsys fs.FS, dir string) (*Catalog, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	c := &Catalog{bySlug: map[string]*Lesson{}, exercises: map[string]*Exercise{}}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		l, err := parseLesson(e.Name(), string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if c.bySlug[l.Slug] != nil {
			return nil, fmt.Errorf("duplicate lesson slug %q", l.Slug)
		}
		c.bySlug[l.Slug] = l
		c.Lessons = append(c.Lessons, l)
		for _, ex := range l.Exercises {
			if c.exercises[ex.ID] != nil {
				return nil, fmt.Errorf("duplicate exercise id %q", ex.ID)
			}
			c.exercises[ex.ID] = ex
		}
	}
	sort.Slice(c.Lessons, func(a, b int) bool { return c.Lessons[a].Order < c.Lessons[b].Order })
	for _, l := range c.Lessons {
		if n := len(c.Modules); n == 0 || c.Modules[n-1].Name != l.Module {
			c.Modules = append(c.Modules, &Module{Name: l.Module})
		}
		m := c.Modules[len(c.Modules)-1]
		m.Lessons = append(m.Lessons, l)
	}
	return c, nil
}

func parseLesson(filename, raw string) (*Lesson, error) {
	base := strings.TrimSuffix(filename, ".md")
	num, slug, ok := strings.Cut(base, "-")
	order, err := strconv.Atoi(num)
	if !ok || err != nil {
		return nil, fmt.Errorf("file name must look like 01-slug.md")
	}
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	head, rest, ok := strings.Cut(raw, "\n---\n")
	if !ok {
		return nil, fmt.Errorf("missing --- after the header")
	}
	l := &Lesson{Slug: slug, Order: order, Exercises: []*Exercise{}}
	for _, line := range strings.Split(head, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "title":
			l.Title = v
		case "module":
			l.Module = v
		case "level":
			l.Level = v
		case "minutes":
			l.Minutes, _ = strconv.Atoi(v)
		case "summary":
			l.Summary = v
		}
	}
	if l.Title == "" || l.Module == "" {
		return nil, fmt.Errorf("title and module are required")
	}
	blocks := strings.Split(rest, "\n=== exercise\n")
	l.Body = strings.TrimSpace(blocks[0])
	for i, b := range blocks[1:] {
		end := strings.LastIndex(b, "\n===")
		if end < 0 {
			return nil, fmt.Errorf("exercise %d is not closed with ===", i+1)
		}
		ex, err := parseExercise(b[:end])
		if err != nil {
			return nil, fmt.Errorf("exercise %d: %w", i+1, err)
		}
		ex.Lesson = slug
		l.Exercises = append(l.Exercises, ex)
	}
	return l, nil
}

func parseExercise(b string) (*Exercise, error) {
	ex := &Exercise{Kind: "select"}
	lines := strings.Split(b, "\n")
	section := ""
	var buf []string
	flush := func() {
		text := strings.TrimSpace(strings.Join(buf, "\n"))
		switch section {
		case "prompt":
			ex.Prompt = text
		case "solution":
			ex.Solution = text
		case "hint":
			ex.Hint = text
		case "check":
			ex.Check = text
		case "setup":
			ex.Setup = text
		}
		buf = nil
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "--- ") {
			flush()
			section = strings.TrimSpace(strings.TrimPrefix(line, "--- "))
			continue
		}
		if section == "" {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			v = strings.TrimSpace(v)
			switch strings.TrimSpace(k) {
			case "id":
				ex.ID = v
			case "title":
				ex.Title = v
			case "kind":
				ex.Kind = v
			case "ordered":
				ex.Ordered = v == "true"
			}
			continue
		}
		buf = append(buf, line)
	}
	flush()
	switch {
	case ex.ID == "" || ex.Title == "":
		return nil, fmt.Errorf("id and title are required")
	case ex.Prompt == "" || ex.Solution == "":
		return nil, fmt.Errorf("%s: prompt and solution are required", ex.ID)
	case ex.Kind != "select" && ex.Kind != "dml" && ex.Kind != "ddl":
		return nil, fmt.Errorf("%s: unknown kind %q", ex.ID, ex.Kind)
	case ex.Kind != "select" && ex.Check == "":
		return nil, fmt.Errorf("%s: %s exercises need a check query", ex.ID, ex.Kind)
	}
	return ex, nil
}
