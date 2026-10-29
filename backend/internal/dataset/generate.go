// Package dataset builds the read-only "core" schema that every learner starts from:
// an e-commerce store, an HR system and a small bank/brokerage, generated
// deterministically so every install has identical data.
package dataset

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"
)

//go:embed schema.sql
var SchemaSQL string

// Tables lists every core table, parents before children.
var Tables = []string{
	"categories", "suppliers", "products", "customers", "orders", "order_items", "payments", "shipments", "reviews",
	"offices", "departments", "jobs", "employees", "salaries", "performance_reviews", "leave_requests",
	"stocks", "stock_prices", "accounts", "transactions", "trades", "exchange_rates", "loans",
	"calendar",
}

// Counts controls the size of the generated data.
type Counts struct {
	Customers     int
	Products      int
	Suppliers     int
	Orders        int // approximate
	Employees     int
	Stocks        int
	BankCustomers int // customers that also hold bank accounts
	Trades        int // approximate
	Loans         int
}

// DefaultCounts is tuned so the core schema is about 150 MB on disk (in one general tablespace).
var DefaultCounts = Counts{
	Customers:     40000,
	Products:      2500,
	Suppliers:     80,
	Orders:        205000,
	Employees:     5000,
	Stocks:        80,
	BankCustomers: 750,
	Trades:        15000,
	Loans:         5000,
}

// Options for Generate.
type Options struct {
	Schema string // target schema, normally "core"
	Seed   uint64
	Counts Counts
	Log    func(format string, args ...any)
}

type gen struct {
	ctx       context.Context
	conn      *sql.Conn
	opt       Options
	counts    Counts
	start     time.Time
	end       time.Time
	products  []productRec
	customers []customerRec
}

func (g *gen) logf(format string, args ...any) {
	if g.opt.Log != nil {
		g.opt.Log(format, args...)
	}
}

// Generate drops and rebuilds the target schema with freshly generated data.
func Generate(ctx context.Context, db *sql.DB, opt Options) (err error) {
	if opt.Schema == "" {
		opt.Schema = "core"
	}
	if opt.Seed == 0 {
		opt.Seed = 20260101
	}
	if opt.Counts == (Counts{}) {
		opt.Counts = DefaultCounts
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer func() {
		if r := recover(); r != nil {
			if ge, ok := r.(genError); ok {
				err = ge.err
				return
			}
			panic(r)
		}
	}()

	q := "`" + opt.Schema + "`"
	ts := TablespaceName(opt.Schema)
	if _, err := conn.ExecContext(ctx, "DROP DATABASE IF EXISTS "+q); err != nil {
		return err
	}
	if err := DropTablespace(ctx, conn, ts); err != nil {
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
	if err := ApplySchema(ctx, conn, ts); err != nil {
		return err
	}

	g := &gen{
		ctx:    ctx,
		conn:   conn,
		opt:    opt,
		counts: opt.Counts,
		start:  day(2021, 1, 1),
		end:    time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
	}
	began := time.Now()
	g.calendar()
	g.ecommerce()
	g.hr()
	g.finance()
	if _, err := conn.ExecContext(ctx, "SET SESSION foreign_key_checks = 1"); err != nil {
		return err
	}
	for _, t := range Tables {
		if _, err := conn.ExecContext(ctx, "ANALYZE TABLE "+q+".`"+t+"`"); err != nil {
			return err
		}
	}
	g.logf("generated %s in %s", opt.Schema, time.Since(began).Round(time.Millisecond))
	return nil
}

// ApplySchema creates all core tables in the connection's current schema,
// placing them in the given general tablespace. Keeping a schema's tables in a
// single tablespace file roughly halves disk use compared to file-per-table,
// because InnoDB reserves free extents per file.
func ApplySchema(ctx context.Context, conn *sql.Conn, tablespace string) error {
	for _, stmt := range strings.Split(SchemaSQL, ";\n") {
		if strings.TrimSpace(stripLineComments(stmt)) == "" {
			continue
		}
		stmt = strings.Replace(stmt, ") ENGINE=InnoDB", ") TABLESPACE `"+tablespace+"` ENGINE=InnoDB", 1)
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	}
	return nil
}

// TablespaceName is the general tablespace that holds a schema's dataset tables.
func TablespaceName(schema string) string { return schema + "_ts" }

// DropTablespace removes a general tablespace if it exists.
func DropTablespace(ctx context.Context, conn *sql.Conn, ts string) error {
	var n int
	err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.INNODB_TABLESPACES WHERE NAME = ?", ts).Scan(&n)
	if err != nil || n == 0 {
		return err
	}
	_, err = conn.ExecContext(ctx, "DROP TABLESPACE `"+ts+"`")
	return err
}

func stripLineComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
