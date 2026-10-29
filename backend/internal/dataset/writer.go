package dataset

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Value wrappers that control how a Go value is rendered into SQL.
type (
	// Dec renders a float with a fixed number of decimal places.
	Dec struct {
		V float64
		P int
	}
	// Date renders as 'YYYY-MM-DD'.
	Date time.Time
	// DateTime renders as 'YYYY-MM-DD HH:MM:SS'.
	DateTime time.Time
)

// D2 is shorthand for a 2-decimal money value.
func D2(v float64) Dec { return Dec{V: v, P: 2} }

// genError is used to abort generation from deep inside loops; Generate recovers it.
type genError struct{ err error }

// inserter batches rows into multi-row INSERT statements of roughly 1 MB.
type inserter struct {
	ctx   context.Context
	conn  *sql.Conn
	table string
	head  string
	buf   []byte
	rows  int
	Total int
}

func newInserter(ctx context.Context, conn *sql.Conn, table string, cols ...string) *inserter {
	return &inserter{
		ctx:   ctx,
		conn:  conn,
		table: table,
		head:  "INSERT INTO " + table + " (" + strings.Join(cols, ",") + ") VALUES ",
		buf:   make([]byte, 0, 1<<20+4096),
	}
}

// Row appends one row. It panics with genError on database errors.
func (w *inserter) Row(vals ...any) {
	if w.rows == 0 {
		w.buf = append(w.buf, w.head...)
	} else {
		w.buf = append(w.buf, ',')
	}
	w.buf = append(w.buf, '(')
	for i, v := range vals {
		if i > 0 {
			w.buf = append(w.buf, ',')
		}
		w.buf = appendValue(w.buf, v)
	}
	w.buf = append(w.buf, ')')
	w.rows++
	w.Total++
	if len(w.buf) >= 1<<20 {
		w.Flush()
	}
}

// Flush sends any buffered rows.
func (w *inserter) Flush() {
	if w.rows == 0 {
		return
	}
	if _, err := w.conn.ExecContext(w.ctx, string(w.buf)); err != nil {
		panic(genError{fmt.Errorf("insert into %s: %w", w.table, err)})
	}
	w.buf = w.buf[:0]
	w.rows = 0
}

func appendValue(b []byte, v any) []byte {
	switch x := v.(type) {
	case nil:
		return append(b, "NULL"...)
	case int:
		return strconv.AppendInt(b, int64(x), 10)
	case int32:
		return strconv.AppendInt(b, int64(x), 10)
	case int64:
		return strconv.AppendInt(b, x, 10)
	case uint32:
		return strconv.AppendUint(b, uint64(x), 10)
	case uint64:
		return strconv.AppendUint(b, x, 10)
	case float64:
		return strconv.AppendFloat(b, x, 'f', -1, 64)
	case Dec:
		p := math.Pow(10, float64(x.P))
		return strconv.AppendFloat(b, math.Round(x.V*p)/p, 'f', x.P, 64)
	case bool:
		if x {
			return append(b, '1')
		}
		return append(b, '0')
	case string:
		return appendQuoted(b, x)
	case Date:
		b = append(b, '\'')
		b = time.Time(x).AppendFormat(b, "2006-01-02")
		return append(b, '\'')
	case DateTime:
		b = append(b, '\'')
		b = time.Time(x).AppendFormat(b, "2006-01-02 15:04:05")
		return append(b, '\'')
	case *time.Time:
		if x == nil {
			return append(b, "NULL"...)
		}
		return appendValue(b, Date(*x))
	}
	panic(fmt.Sprintf("dataset: unsupported value type %T", v))
}

func appendQuoted(b []byte, s string) []byte {
	b = append(b, '\'')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\'':
			b = append(b, '\\', '\'')
		case '\\':
			b = append(b, '\\', '\\')
		case 0:
			b = append(b, '\\', '0')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case 0x1a:
			b = append(b, '\\', 'Z')
		default:
			b = append(b, c)
		}
	}
	return append(b, '\'')
}
