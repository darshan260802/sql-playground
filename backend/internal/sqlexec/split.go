// Package sqlexec splits, classifies and executes user SQL against MySQL.
package sqlexec

import (
	"strings"
	"unicode"
)

// Statement is one SQL statement extracted from a script.
type Statement struct {
	SQL  string `json:"sql"`
	Line int    `json:"line"` // 1-based line where the statement starts
}

// Split breaks a script into statements the way the mysql command-line client
// does: it respects quotes, backticks and comments, and understands the
// client-side DELIMITER command used to write stored routines and triggers.
func Split(script string) []Statement {
	var out []Statement
	delim := ";"
	n := len(script)
	start := 0
	line := 1
	startLine := 1
	atStmtStart := true // only whitespace/comments seen since the last statement

	emit := func(end int) {
		text := strings.TrimSpace(script[start:end])
		if text != "" && !onlyComments(text) {
			out = append(out, Statement{SQL: text, Line: startLine})
		}
	}

	i := 0
	for i < n {
		c := script[i]

		if atStmtStart {
			if c == '\n' {
				line++
				i++
				start = i
				startLine = line
				continue
			}
			if c == ' ' || c == '\t' || c == '\r' {
				i++
				continue
			}
			// DELIMITER is a client command and must start a statement.
			if hasPrefixFold(script[i:], "delimiter") && i+9 < n && (script[i+9] == ' ' || script[i+9] == '\t') {
				j := i + 9
				for j < n && (script[j] == ' ' || script[j] == '\t') {
					j++
				}
				k := j
				for k < n && script[k] != '\n' && script[k] != '\r' && script[k] != ' ' && script[k] != '\t' {
					k++
				}
				if k > j {
					delim = script[j:k]
				}
				for k < n && script[k] != '\n' {
					k++
				}
				i = k
				start = i
				continue
			}
			if !isCommentStart(script, i) {
				atStmtStart = false
				start = i
				startLine = line
			}
		}

		switch {
		case c == '\'' || c == '"' || c == '`':
			i, line = skipQuoted(script, i, line)
			continue
		case c == '#' || (c == '-' && isCommentStart(script, i)):
			for i < n && script[i] != '\n' {
				i++
			}
			continue
		case c == '/' && i+1 < n && script[i+1] == '*':
			j := strings.Index(script[i+2:], "*/")
			end := n
			if j >= 0 {
				end = i + 2 + j + 2
			}
			line += strings.Count(script[i:end], "\n")
			i = end
			continue
		case c == '\n':
			line++
			i++
			continue
		}

		if strings.HasPrefix(script[i:], delim) {
			emit(i)
			i += len(delim)
			start = i
			startLine = line
			atStmtStart = true
			continue
		}
		i++
	}
	if !atStmtStart || start < n {
		emit(n)
	}
	return out
}

func isCommentStart(s string, i int) bool {
	n := len(s)
	switch s[i] {
	case '#':
		return true
	case '-':
		return i+1 < n && s[i+1] == '-' && (i+2 >= n || unicode.IsSpace(rune(s[i+2])))
	case '/':
		return i+1 < n && s[i+1] == '*'
	}
	return false
}

func skipQuoted(s string, i, line int) (int, int) {
	q := s[i]
	i++
	for i < len(s) {
		c := s[i]
		if c == '\n' {
			line++
		}
		if c == '\\' && q != '`' && i+1 < len(s) {
			if s[i+1] == '\n' {
				line++
			}
			i += 2
			continue
		}
		if c == q {
			if i+1 < len(s) && s[i+1] == q { // doubled quote escape
				i += 2
				continue
			}
			return i + 1, line
		}
		i++
	}
	return i, line
}

func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// onlyComments reports whether text has no SQL besides comments.
func onlyComments(text string) bool {
	return strings.TrimSpace(StripComments(text)) == ""
}

// StripComments removes -- , # and /* */ comments (but keeps /*! */ executable
// comments) outside of quoted strings.
func StripComments(s string) string {
	var b strings.Builder
	n := len(s)
	for i := 0; i < n; {
		c := s[i]
		switch {
		case c == '\'' || c == '"' || c == '`':
			j, _ := skipQuoted(s, i, 0)
			b.WriteString(s[i:j])
			i = j
		case c == '#' || (c == '-' && i+1 < n && s[i+1] == '-' && (i+2 >= n || unicode.IsSpace(rune(s[i+2])))):
			for i < n && s[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < n && s[i+1] == '*' && !(i+2 < n && s[i+2] == '!'):
			j := strings.Index(s[i+2:], "*/")
			if j < 0 {
				i = n
			} else {
				i = i + 2 + j + 2
			}
			b.WriteByte(' ')
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// Kind is a coarse statement category used for permissions and quota checks.
type Kind int

const (
	KindRead  Kind = iota // SELECT, SHOW, DESCRIBE, EXPLAIN, WITH ... SELECT
	KindWrite             // anything that may add data or objects
	KindFree              // statements that only remove data or are session-only
	KindTxn               // START TRANSACTION, BEGIN, COMMIT, ROLLBACK, SAVEPOINT
)

// FirstKeyword returns the upper-cased first word of a statement, ignoring
// comments and leading parentheses.
func FirstKeyword(sql string) string {
	words := Keywords(sql, 1)
	if len(words) == 0 {
		return ""
	}
	return words[0]
}

// Keywords returns up to max upper-cased leading words of the statement.
func Keywords(sql string, max int) []string {
	s := StripComments(sql)
	var words []string
	i := 0
	for i < len(s) && len(words) < max {
		for i < len(s) && (unicode.IsSpace(rune(s[i])) || s[i] == '(') {
			i++
		}
		j := i
		for j < len(s) && (s[j] == '_' || unicode.IsLetter(rune(s[j])) || unicode.IsDigit(rune(s[j]))) {
			j++
		}
		if j == i {
			break
		}
		words = append(words, strings.ToUpper(s[i:j]))
		i = j
	}
	return words
}

// Classify returns the statement kind.
func Classify(sql string) Kind {
	w := Keywords(sql, 3)
	if len(w) == 0 {
		return KindRead
	}
	switch w[0] {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "TABLE", "VALUES", "HELP", "DO":
		return KindRead
	case "WITH":
		u := strings.ToUpper(StripComments(sql))
		for _, kw := range []string{"INSERT ", "UPDATE ", "DELETE ", "REPLACE "} {
			if strings.Contains(u, kw) {
				return KindWrite
			}
		}
		return KindRead
	case "DELETE", "TRUNCATE", "DROP", "OPTIMIZE", "ANALYZE", "CHECK", "SET", "USE", "CALL", "DEALLOCATE":
		// CALL may write, but the watchdog still enforces the quota while it runs.
		return KindFree
	case "START", "BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE", "XA":
		return KindTxn
	}
	return KindWrite
}
