package lessons

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/darshan260802/sql-playground/backend/internal/dataset"
	"github.com/darshan260802/sql-playground/backend/internal/sqlexec"
)

func TestCatalogLoads(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Lessons) == 0 {
		t.Fatal("no lessons")
	}
	for _, l := range c.Lessons {
		if l.Level == "" || l.Summary == "" || l.Minutes == 0 {
			t.Errorf("%s: level, summary and minutes are required", l.Slug)
		}
		if len(l.Exercises) == 0 {
			t.Errorf("%s: has no exercises", l.Slug)
		}
		if strings.Count(l.Body, "```")%2 != 0 {
			t.Errorf("%s: unbalanced code fences", l.Slug)
		}
	}
}

func TestCompare(t *testing.T) {
	s := func(v string) *string { return &v }
	exp := &sqlexec.Result{Columns: make([]sqlexec.Column, 2), Rows: [][]*string{{s("a"), s("12.50")}, {s("b"), nil}}, RowCount: 2}
	same := &sqlexec.Result{Columns: make([]sqlexec.Column, 2), Rows: [][]*string{{s("a"), s("12.5")}, {s("b"), nil}}, RowCount: 2}
	swapped := &sqlexec.Result{Columns: make([]sqlexec.Column, 2), Rows: [][]*string{{s("b"), nil}, {s("a"), s("12.4999")}}, RowCount: 2}
	nullText := &sqlexec.Result{Columns: make([]sqlexec.Column, 2), Rows: [][]*string{{s("a"), s("12.50")}, {s("b"), s("NULL")}}, RowCount: 2}
	if !compare(exp, same, true).Correct {
		t.Error("12.5 should equal 12.50")
	}
	if !compare(exp, swapped, false).Correct {
		t.Error("unordered comparison should ignore order")
	}
	if o := compare(exp, swapped, true); o.Correct || !strings.Contains(o.Message, "order") {
		t.Errorf("ordered comparison should flag order, got %q", o.Message)
	}
	if compare(exp, nullText, false).Correct {
		t.Error("NULL must differ from the text 'NULL'")
	}
}

// TestSolutions runs every exercise against a real MySQL with the core dataset.
// Set SQLPG_TEST_DSN, e.g. "user:pass@tcp(127.0.0.1:3306)/" (admin rights needed).
func TestSolutions(t *testing.T) {
	dsn := os.Getenv("SQLPG_TEST_DSN")
	if dsn == "" {
		t.Skip("SQLPG_TEST_DSN not set")
	}
	core := os.Getenv("SQLPG_TEST_CORE")
	if core == "" {
		core = "core"
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MultiStatements = false
	ctx := context.Background()
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// a scratch copy of core for DML/DDL exercises
	const scratch = "lessons_test"
	if os.Getenv("SQLPG_TEST_REUSE") == "" {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		ts := dataset.TablespaceName(scratch)
		for _, q := range []string{"DROP DATABASE IF EXISTS " + scratch} {
			if _, err := conn.ExecContext(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
		if err := dataset.DropTablespace(ctx, conn, ts); err != nil {
			t.Fatal(err)
		}
		for _, q := range []string{
			"CREATE TABLESPACE " + ts + " ADD DATAFILE '" + ts + ".ibd'",
			"CREATE DATABASE " + scratch, "USE " + scratch, "SET foreign_key_checks = 0",
		} {
			if _, err := conn.ExecContext(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
		if err := dataset.ApplySchema(ctx, conn, ts); err != nil {
			t.Fatal(err)
		}
		for _, tb := range dataset.Tables {
			if _, err := conn.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s.%s SELECT * FROM %s.%s", scratch, tb, core, tb)); err != nil {
				t.Fatal(err)
			}
		}
		conn.Close()
	}

	opener := func(schema string) Opener {
		return func(ctx context.Context) (*sql.Conn, int64, error) {
			conn, err := db.Conn(ctx)
			if err != nil {
				return nil, 0, err
			}
			for _, q := range []string{"USE " + schema, "SET SESSION innodb_lock_wait_timeout = 10", "SET SESSION lock_wait_timeout = 10"} {
				if _, err := conn.ExecContext(ctx, q); err != nil {
					conn.Close()
					return nil, 0, err
				}
			}
			id, err := sqlexec.ConnectionID(ctx, conn)
			return conn, id, err
		}
	}
	kill := func(id int64) { db.Exec(fmt.Sprintf("KILL QUERY %d", id)) }

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ch := NewChecker(kill)
	only := os.Getenv("SQLPG_TEST_ONLY")
	for _, l := range c.Lessons {
		for _, ex := range l.Exercises {
			if only != "" && !strings.HasPrefix(ex.ID, only) && l.Slug != only {
				continue
			}
			t.Run(ex.ID, func(t *testing.T) {
				start := time.Now()
				var good, bad *Outcome
				var err error
				switch ex.Kind {
				case "select":
					exp, err := ch.Expected(ctx, ex, opener(core))
					if err != nil {
						t.Fatal(err)
					}
					if exp.RowCount == 0 {
						t.Fatal("solution returns no rows")
					}
					if exp.RowCount > 2000 {
						t.Errorf("solution returns %d rows; keep exercises under 2000", exp.RowCount)
					}
					good, err = ch.CheckSelect(ctx, ex, ex.Solution, opener(core))
					if err == nil {
						bad, err = ch.CheckSelect(ctx, ex, "SELECT 1 AS wrong", opener(core))
					}
				case "dml":
					good, err = ch.CheckDML(ctx, ex, ex.Solution, opener(scratch))
					if err == nil {
						bad, err = ch.CheckDML(ctx, ex, "SELECT 1", opener(scratch))
					}
				case "ddl":
					bad, err = ch.CheckDDL(ctx, ex, "SELECT 1", opener(scratch), nil)
					if err == nil {
						good, err = ch.CheckDDL(ctx, ex, ex.Solution, opener(scratch), nil)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if !good.Correct {
					msg := good.Message + " " + good.Detail
					if good.Error != nil {
						msg += " error: " + good.Error.Message
					}
					t.Errorf("solution is not accepted: %s", msg)
				}
				if bad.Correct {
					t.Errorf("a wrong answer is accepted")
				}
				if d := time.Since(start); d > 8*time.Second {
					t.Errorf("slow exercise: %s", d)
				}
			})
		}
	}
}
