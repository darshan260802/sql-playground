package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type SchemaInfo struct {
	Schema   string      `json:"schema"`
	Target   string      `json:"target"`
	Tables   []TableInfo `json:"tables"`
	Routines []Routine   `json:"routines"`
	Triggers []Trigger   `json:"triggers"`
}

type TableInfo struct {
	Name    string       `json:"name"`
	Type    string       `json:"type"` // table | view
	Domain  string       `json:"domain"`
	Comment string       `json:"comment"`
	Rows    int64        `json:"rows"`
	Exact   bool         `json:"exact"`
	Columns []ColumnInfo `json:"columns"`
}

type ColumnInfo struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Nullable bool    `json:"nullable"`
	Key      string  `json:"key,omitempty"` // PRI, UNI, MUL
	Default  *string `json:"default,omitempty"`
	Comment  string  `json:"comment,omitempty"`
	Ref      string  `json:"ref,omitempty"` // "table.column" for foreign keys
}

type Routine struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Trigger struct {
	Name  string `json:"name"`
	Table string `json:"table"`
	Event string `json:"event"`
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	target := r.URL.Query().Get("target")
	if u == nil || target != "mine" || u.DBStatus != "ready" {
		info, err := s.coreSchemaInfo(r.Context())
		if err != nil {
			internalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
		return
	}
	info, err := loadSchema(r.Context(), s.prov.Admin(), s.prov.Schema(u.ID))
	if err != nil {
		internalError(w, err)
		return
	}
	info.Target = "mine"
	writeJSON(w, http.StatusOK, info)
}

// coreSchemaInfo is cached: the core dataset never changes while running.
func (s *Server) coreSchemaInfo(ctx context.Context) (*SchemaInfo, error) {
	s.schemaMu.Lock()
	defer s.schemaMu.Unlock()
	if s.coreSchema != nil {
		return s.coreSchema, nil
	}
	info, err := loadSchema(ctx, s.prov.Admin(), s.cfg.CoreSchema)
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for i := range info.Tables {
		t := &info.Tables[i]
		if err := s.prov.Admin().QueryRowContext(cctx, fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`", s.cfg.CoreSchema, t.Name)).Scan(&t.Rows); err == nil {
			t.Exact = true
		}
	}
	info.Target = "core"
	s.coreSchema = info
	return info, nil
}

var domainOrder = map[string]int{"E-commerce": 0, "HR": 1, "Finance": 2, "Utility": 3, "Your tables": 4}

func loadSchema(ctx context.Context, db *sql.DB, schema string) (*SchemaInfo, error) {
	info := &SchemaInfo{Schema: schema, Tables: []TableInfo{}, Routines: []Routine{}, Triggers: []Trigger{}}
	rows, err := db.QueryContext(ctx, `SELECT TABLE_NAME, TABLE_TYPE, COALESCE(TABLE_ROWS,0), COALESCE(TABLE_COMMENT,'')
		FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`, schema)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for rows.Next() {
		var t TableInfo
		var typ string
		if err := rows.Scan(&t.Name, &typ, &t.Rows, &t.Comment); err != nil {
			rows.Close()
			return nil, err
		}
		t.Type = "table"
		if typ == "VIEW" {
			t.Type = "view"
			t.Comment = ""
		}
		t.Domain = "Your tables"
		if d, rest, ok := strings.Cut(t.Comment, ":"); ok {
			if _, known := domainOrder[d]; known && d != "Your tables" {
				t.Domain = d
				t.Comment = strings.TrimSpace(rest)
			}
		}
		t.Columns = []ColumnInfo{}
		idx[t.Name] = len(info.Tables)
		info.Tables = append(info.Tables, t)
	}
	rows.Close()

	refs := map[string]string{}
	rows, err = db.QueryContext(ctx, `SELECT TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = ? AND REFERENCED_TABLE_NAME IS NOT NULL`, schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var t, c, rt, rc string
		if err := rows.Scan(&t, &c, &rt, &rc); err == nil {
			refs[t+"."+c] = rt + "." + rc
		}
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT, COALESCE(COLUMN_COMMENT,'')
		FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? ORDER BY TABLE_NAME, ORDINAL_POSITION`, schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tname, nullable string
		var c ColumnInfo
		var def sql.NullString
		if err := rows.Scan(&tname, &c.Name, &c.Type, &nullable, &c.Key, &def, &c.Comment); err != nil {
			rows.Close()
			return nil, err
		}
		c.Nullable = nullable == "YES"
		if def.Valid {
			c.Default = &def.String
		}
		c.Ref = refs[tname+"."+c.Name]
		if i, ok := idx[tname]; ok {
			info.Tables[i].Columns = append(info.Tables[i].Columns, c)
		}
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT ROUTINE_NAME, ROUTINE_TYPE FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = ? ORDER BY ROUTINE_NAME`, schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var rt Routine
		if rows.Scan(&rt.Name, &rt.Type) == nil {
			info.Routines = append(info.Routines, rt)
		}
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT TRIGGER_NAME, EVENT_OBJECT_TABLE, CONCAT(ACTION_TIMING, ' ', EVENT_MANIPULATION)
		FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA = ? ORDER BY TRIGGER_NAME`, schema)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var tr Trigger
		if rows.Scan(&tr.Name, &tr.Table, &tr.Event) == nil {
			info.Triggers = append(info.Triggers, tr)
		}
	}
	rows.Close()

	sort.SliceStable(info.Tables, func(a, b int) bool {
		ta, tb := info.Tables[a], info.Tables[b]
		if domainOrder[ta.Domain] != domainOrder[tb.Domain] {
			return domainOrder[ta.Domain] < domainOrder[tb.Domain]
		}
		return ta.Name < tb.Name
	})
	return info, nil
}
