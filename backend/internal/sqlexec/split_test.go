package sqlexec

import (
	"reflect"
	"testing"
)

func sqls(stmts []Statement) []string {
	out := make([]string, len(stmts))
	for i, s := range stmts {
		out[i] = s.SQL
	}
	return out
}

func TestSplit(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"single no semicolon", "SELECT 1", []string{"SELECT 1"}},
		{"two statements", "SELECT 1; SELECT 2;", []string{"SELECT 1", "SELECT 2"}},
		{"semicolon in string", "SELECT 'a;b'; SELECT \"c;d\"", []string{"SELECT 'a;b'", "SELECT \"c;d\""}},
		{"escaped quotes", `SELECT 'it\'s;' , 'x''y;z'; SELECT 2`, []string{`SELECT 'it\'s;' , 'x''y;z'`, "SELECT 2"}},
		{"backtick", "SELECT `a;b` FROM t; SELECT 2", []string{"SELECT `a;b` FROM t", "SELECT 2"}},
		{"line comments", "-- hello; world\nSELECT 1; # x;y\nSELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"block comment", "/* a; b */ SELECT 1; SELECT /* ; */ 2", []string{"SELECT 1", "SELECT /* ; */ 2"}},
		{"only comments", "-- nothing here\n/* nope */", nil},
		{"empty statements", ";;SELECT 1;;", []string{"SELECT 1"}},
		{"minus is not comment", "SELECT 5--3; SELECT 1", []string{"SELECT 5--3", "SELECT 1"}},
		{
			"delimiter",
			"DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\n  SELECT 2;\nEND //\nDELIMITER ;\nCALL p();",
			[]string{"CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\n  SELECT 2;\nEND", "CALL p()"},
		},
		{
			"delimiter $$ lowercase",
			"delimiter $$\nCREATE TRIGGER t BEFORE INSERT ON x FOR EACH ROW BEGIN SET NEW.a = 1; END$$\ndelimiter ;\nSELECT 1",
			[]string{"CREATE TRIGGER t BEFORE INSERT ON x FOR EACH ROW BEGIN SET NEW.a = 1; END", "SELECT 1"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sqls(Split(c.in))
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Split(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	got := Split("SELECT 1;\n\n-- c\nSELECT\n2;\nSELECT 3")
	lines := []int{got[0].Line, got[1].Line, got[2].Line}
	if !reflect.DeepEqual(lines, []int{1, 4, 6}) {
		t.Fatalf("lines = %v", lines)
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]Kind{
		"SELECT * FROM t":                         KindRead,
		"  (SELECT 1) UNION (SELECT 2)":           KindRead,
		"-- c\nshow tables":                       KindRead,
		"WITH x AS (SELECT 1) SELECT * FROM x":    KindRead,
		"WITH x AS (SELECT 1) UPDATE t SET a = 1": KindWrite,
		"INSERT INTO t VALUES (1)":                KindWrite,
		"create table t (a int)":                  KindWrite,
		"DELETE FROM t":                           KindFree,
		"DROP TABLE t":                            KindFree,
		"START TRANSACTION":                       KindTxn,
		"rollback":                                KindTxn,
		"UPDATE t SET a = 1":                      KindWrite,
	}
	for in, want := range cases {
		if got := Classify(in); got != want {
			t.Errorf("Classify(%q) = %v, want %v", in, got, want)
		}
	}
}
