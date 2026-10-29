package lessons

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darshan260802/sql-playground/backend/internal/sqlexec"
)

// Opener returns a dedicated connection (and its MySQL thread id).
type Opener func(ctx context.Context) (*sql.Conn, int64, error)

type Checker struct {
	Kill    func(connID int64)
	MaxRows int
	Timeout time.Duration

	mu    sync.Mutex
	cache map[string]*sqlexec.Result
}

func NewChecker(kill func(int64)) *Checker {
	return &Checker{Kill: kill, MaxRows: 10000, Timeout: 20 * time.Second, cache: map[string]*sqlexec.Result{}}
}

type Preview struct {
	Columns  []sqlexec.Column `json:"columns"`
	Rows     [][]*string      `json:"rows"`
	RowCount int              `json:"row_count"`
}

func PreviewOf(r *sqlexec.Result, n int) *Preview {
	if r == nil {
		return nil
	}
	rows := r.Rows
	if len(rows) > n {
		rows = rows[:n]
	}
	return &Preview{Columns: r.Columns, Rows: rows, RowCount: r.RowCount}
}

type Outcome struct {
	Correct bool              `json:"correct"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
	Error   *sqlexec.SQLError `json:"error,omitempty"`
	Result  *Preview          `json:"result,omitempty"` // what the learner's query returned
}

func (c *Checker) run(ctx context.Context, conn *sql.Conn, connID int64, script string, guard func(sqlexec.Statement) *sqlexec.SQLError) (*sqlexec.Result, *sqlexec.SQLError) {
	stmts := sqlexec.Split(script)
	if len(stmts) == 0 {
		return nil, &sqlexec.SQLError{Message: "There's no SQL to check."}
	}
	results := sqlexec.Run(ctx, conn, connID, stmts, sqlexec.Options{
		MaxRows: c.MaxRows, Timeout: c.Timeout, Kill: c.Kill, Guard: guard, StopOnError: true,
	})
	var last *sqlexec.Result
	for i := range results {
		if results[i].Error != nil {
			return nil, results[i].Error
		}
		if results[i].Columns != nil {
			last = &results[i]
		}
	}
	return last, nil
}

// Expected returns the (cached) result of a select exercise's solution.
func (c *Checker) Expected(ctx context.Context, ex *Exercise, open Opener) (*sqlexec.Result, error) {
	c.mu.Lock()
	if r, ok := c.cache[ex.ID]; ok {
		c.mu.Unlock()
		return r, nil
	}
	c.mu.Unlock()
	conn, id, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	r, e := c.run(ctx, conn, id, ex.Solution, nil)
	if e != nil {
		return nil, fmt.Errorf("solution for %s failed: %s", ex.ID, e.Message)
	}
	if r == nil {
		return nil, fmt.Errorf("solution for %s returns no result set", ex.ID)
	}
	c.mu.Lock()
	c.cache[ex.ID] = r
	c.mu.Unlock()
	return r, nil
}

// CheckSelect compares a learner's query with the solution on the read-only core data.
func (c *Checker) CheckSelect(ctx context.Context, ex *Exercise, userSQL string, open Opener) (*Outcome, error) {
	exp, err := c.Expected(ctx, ex, open)
	if err != nil {
		return nil, err
	}
	conn, id, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	got, e := c.run(ctx, conn, id, userSQL, nil)
	if e != nil {
		return &Outcome{Message: "Your query returned an error.", Error: e}, nil
	}
	if got == nil {
		return &Outcome{Message: "Your SQL ran, but it didn't return any rows to compare. End with a SELECT."}, nil
	}
	return compare(exp, got, ex.Ordered), nil
}

var dmlAllowed = map[string]bool{"INSERT": true, "UPDATE": true, "DELETE": true, "REPLACE": true, "SELECT": true, "WITH": true, "SET": true, "VALUES": true, "TABLE": true}

// CheckDML runs the learner's statements and the solution inside transactions
// that are rolled back, comparing what the check query sees afterwards.
func (c *Checker) CheckDML(ctx context.Context, ex *Exercise, userSQL string, open Opener) (*Outcome, error) {
	conn, id, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	guard := func(st sqlexec.Statement) *sqlexec.SQLError {
		if !dmlAllowed[sqlexec.FirstKeyword(st.SQL)] {
			return &sqlexec.SQLError{Message: fmt.Sprintf("%s statements aren't needed here.", sqlexec.FirstKeyword(st.SQL)),
				Hint: "This exercise is checked inside a transaction that is rolled back, so use only INSERT, UPDATE, DELETE and SELECT."}
		}
		return nil
	}
	attempt := func(script string, g func(sqlexec.Statement) *sqlexec.SQLError) (*sqlexec.Result, *sqlexec.SQLError, error) {
		if _, err := conn.ExecContext(ctx, "START TRANSACTION"); err != nil {
			return nil, nil, err
		}
		defer conn.ExecContext(context.Background(), "ROLLBACK")
		if _, e := c.run(ctx, conn, id, script, g); e != nil {
			return nil, e, nil
		}
		r, e := c.run(ctx, conn, id, ex.Check, nil)
		if e != nil {
			return nil, nil, fmt.Errorf("check for %s failed: %s", ex.ID, e.Message)
		}
		return r, nil, nil
	}
	exp, e, err := attempt(ex.Solution, nil)
	if err != nil {
		return nil, err
	}
	if e != nil {
		return nil, fmt.Errorf("solution for %s failed: %s", ex.ID, e.Message)
	}
	got, e, err := attempt(userSQL, guard)
	if err != nil {
		return nil, err
	}
	if e != nil {
		return &Outcome{Message: "Your statement returned an error.", Error: e}, nil
	}
	out := compare(exp, got, true)
	out.Result = nil // the check query's output isn't the learner's own result
	if out.Correct {
		out.Detail = "Your changes were checked inside a transaction and rolled back, so your data is unchanged. Run it in the playground to apply it for real."
	} else {
		out.Message = "Not quite: the data after your statements doesn't match what was expected."
	}
	return out, nil
}

// CheckDDL runs setup, the solution and the check query to learn the expected
// outcome, cleans up, then does the same with the learner's SQL (leaving the
// learner's objects in place).
func (c *Checker) CheckDDL(ctx context.Context, ex *Exercise, userSQL string, open Opener, guard func(sqlexec.Statement) *sqlexec.SQLError) (*Outcome, error) {
	conn, id, err := open(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	setup := func() error {
		if ex.Setup == "" {
			return nil
		}
		if _, e := c.run(ctx, conn, id, ex.Setup, nil); e != nil {
			return fmt.Errorf("setup for %s failed: %s", ex.ID, e.Message)
		}
		return nil
	}
	if err := setup(); err != nil {
		return nil, err
	}
	if _, e := c.run(ctx, conn, id, ex.Solution, nil); e != nil {
		return nil, fmt.Errorf("solution for %s failed: %s", ex.ID, e.Message)
	}
	exp, e := c.run(ctx, conn, id, ex.Check, nil)
	if e != nil {
		return nil, fmt.Errorf("check for %s failed: %s", ex.ID, e.Message)
	}
	if err := setup(); err != nil {
		return nil, err
	}
	if _, e := c.run(ctx, conn, id, userSQL, guard); e != nil {
		return &Outcome{Message: "Your statement returned an error.", Error: e}, nil
	}
	got, e := c.run(ctx, conn, id, ex.Check, nil)
	if e != nil {
		return &Outcome{Message: "We couldn't verify your objects.", Error: e}, nil
	}
	out := compare(exp, got, true)
	out.Result = nil
	if !out.Correct {
		out.Message = "Not quite: the objects you created don't match what was asked."
	}
	return out, nil
}

// ---------------------------------------------------------------- result comparison

var numRE = regexp.MustCompile(`^-?\d+(\.\d+)?([eE][+-]?\d+)?$`)

// norm makes values comparable: numbers are compared to 2 decimal places so
// 12.5, 12.50 and 12.4999 all match, and NULL is distinct from the text 'NULL'.
func norm(v *string) string {
	if v == nil {
		return "\x00NULL"
	}
	s := *v
	if numRE.MatchString(s) {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			r := math.Round(f*100) / 100
			if r == 0 {
				r = 0 // avoid -0.00
			}
			return strconv.FormatFloat(r, 'f', 2, 64)
		}
	}
	return s
}

func rowKey(row []*string) string {
	parts := make([]string, len(row))
	for i, v := range row {
		parts[i] = norm(v)
	}
	return strings.Join(parts, "\x1f")
}

func showRow(row []*string) string {
	parts := make([]string, len(row))
	for i, v := range row {
		if v == nil {
			parts[i] = "NULL"
		} else {
			parts[i] = *v
		}
	}
	s := "(" + strings.Join(parts, ", ") + ")"
	if len(s) > 160 {
		s = s[:157] + "...)"
	}
	return s
}

func colNames(cols []sqlexec.Column) string {
	n := make([]string, len(cols))
	for i, c := range cols {
		n[i] = c.Name
	}
	return strings.Join(n, ", ")
}

func compare(exp, got *sqlexec.Result, ordered bool) *Outcome {
	out := &Outcome{Result: PreviewOf(got, 50)}
	if got.Truncated {
		out.Message = fmt.Sprintf("Your query returns too many rows (%d). The expected answer has %d.", got.RowCount, exp.RowCount)
		return out
	}
	if len(got.Columns) != len(exp.Columns) {
		out.Message = fmt.Sprintf("Expected %d %s, but your query returns %d.", len(exp.Columns), plural(len(exp.Columns), "column", "columns"), len(got.Columns))
		out.Detail = "Expected columns: " + colNames(exp.Columns)
		return out
	}
	if got.RowCount != exp.RowCount {
		out.Message = fmt.Sprintf("Expected %d %s, but your query returns %d.", exp.RowCount, plural(exp.RowCount, "row", "rows"), got.RowCount)
		if got.RowCount > exp.RowCount {
			out.Detail = "Check your filters (WHERE / HAVING), joins that duplicate rows, or a missing LIMIT."
		} else {
			out.Detail = "Check your filters: they may be too strict, or an inner join may be dropping rows."
		}
		return out
	}
	expKeys := make([]string, len(exp.Rows))
	gotKeys := make([]string, len(got.Rows))
	for i := range exp.Rows {
		expKeys[i] = rowKey(exp.Rows[i])
	}
	for i := range got.Rows {
		gotKeys[i] = rowKey(got.Rows[i])
	}
	sameOrder := true
	for i := range expKeys {
		if expKeys[i] != gotKeys[i] {
			sameOrder = false
			break
		}
	}
	if sameOrder {
		out.Correct = true
		out.Message = "Correct!"
		return out
	}
	se := append([]string(nil), expKeys...)
	sg := append([]string(nil), gotKeys...)
	sort.Strings(se)
	sort.Strings(sg)
	sameSet := true
	for i := range se {
		if se[i] != sg[i] {
			sameSet = false
			break
		}
	}
	if sameSet {
		if !ordered {
			out.Correct = true
			out.Message = "Correct!"
			return out
		}
		out.Message = "Right rows, wrong order."
		out.Detail = "Check your ORDER BY: the column(s), the direction (ASC/DESC) and any tie-breakers asked for."
		return out
	}
	// find an example of a missing and an unexpected row
	count := map[string]int{}
	for _, k := range gotKeys {
		count[k]++
	}
	var missing []*string
	for i, k := range expKeys {
		if count[k] > 0 {
			count[k]--
		} else if missing == nil {
			missing = exp.Rows[i]
		}
	}
	out.Message = "Your result has the right shape, but different values."
	if missing != nil {
		out.Detail = "For example, this expected row is missing: " + showRow(missing)
	}
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
