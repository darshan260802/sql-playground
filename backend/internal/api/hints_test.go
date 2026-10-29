package api

import (
	"testing"

	"github.com/darshan260802/sql-playground/backend/internal/sqlexec"
)

func TestDeniedDatabase(t *testing.T) {
	denied := func(table string) *sqlexec.SQLError {
		return &sqlexec.SQLError{Code: 1142, Message: "SELECT command denied to user 'pgu_1'@'localhost' for table '" + table + "'"}
	}
	cases := []struct {
		err       *sqlexec.SQLError
		sql       string
		db, table string
	}{
		{denied("jobs"), "SELECT * from hr.jobs;", "hr", "jobs"},
		{denied("jobs"), "SELECT * FROM `hr`.`jobs` j WHERE 1", "hr", "jobs"},
		{denied("jobs"), "select * from HR . jobs", "HR", "jobs"},
		{denied("orders"), "DELETE FROM orders", "", "orders"},
		{denied("orders"), "UPDATE core.orders SET status = 'x'", "core", "orders"},
		{denied("hr.jobs"), "SELECT * FROM hr.jobs", "hr", "jobs"},
		{&sqlexec.SQLError{Code: 1044, Message: "Access denied for user 'pgu_1'@'%' to database 'hr'"}, "USE hr", "hr", ""},
		{&sqlexec.SQLError{Code: 1064, Message: "You have an error in your SQL syntax"}, "SELEC 1", "", ""},
	}
	for _, c := range cases {
		db, table := deniedDatabase(c.err, c.sql)
		if db != c.db || table != c.table {
			t.Errorf("%q: got (%q, %q), want (%q, %q)", c.sql, db, table, c.db, c.table)
		}
	}
}
