title: SQL injection and prepared statements
module: Designing databases
level: Intermediate
minutes: 12
summary: How gluing user input into SQL lets attackers rewrite your query, and how placeholders prevent it.
---
**SQL injection** is one of the most common and most damaging security bugs on the web. It happens when an application builds a SQL statement by **pasting user input into the SQL text**.

### How it happens

Imagine a page that looks up a department by the name a visitor types. The code builds the query by joining strings:

```text
query = "SELECT name, budget FROM departments WHERE name = '" + input + "'"
```

With the input `Data` the database receives a harmless query:

```sql
SELECT name, budget FROM departments WHERE name = 'Data';
```

Now an attacker types `x' OR '1'='1` instead. The quote in their input **closes the string early**, and the rest becomes SQL:

```sql
SELECT name, budget FROM departments WHERE name = 'x' OR '1'='1';
```

`'1'='1'` is true for every row, so the page now shows **every department's budget**. With other inputs an attacker can read other tables (`' UNION SELECT email, phone FROM customers -- `), skip a password check (`' OR 1=1 -- ` in a login form), or, where the account is allowed to, change and delete data.

### The fix: placeholders, not string gluing

Send the SQL and the values **separately**. The SQL contains a placeholder `?`, and the value is passed alongside; the database never reads the value as SQL, whatever characters it contains. This is a **prepared statement** (or *parameterized query*). MySQL lets you try it directly:

```sql
SET @input = 'x'' OR ''1''=''1';                 -- the attacker's text, as data
PREPARE find_dept FROM 'SELECT name, budget FROM departments WHERE name = ?';
EXECUTE find_dept USING @input;                 -- 0 rows: no department has that name
DEALLOCATE PREPARE find_dept;
```

(Inside a SQL string, a quote is written as two quotes `''`, so `@input` holds exactly `x' OR '1'='1`.)

In application code it looks like this in every language; the driver does the `PREPARE` and `EXECUTE` for you:

```text
Python:  cursor.execute("SELECT name, budget FROM departments WHERE name = %s", (name,))
Go:      db.Query("SELECT name, budget FROM departments WHERE name = ?", name)
Node:    conn.execute("SELECT name, budget FROM departments WHERE name = ?", [name])
```

### Defence in depth

- **Always use placeholders** for values. Escaping quotes by hand is easy to get wrong.
- Placeholders work for **values only**, not table or column names. If users choose a column to sort by, check it against a fixed list of allowed names.
- Give the application's database account **only the privileges it needs**. This playground does exactly that: visitors run queries as an account that can only `SELECT` from the original data, so even a successful injection couldn't change it.
- Don't show raw database errors to visitors; they help attackers map your tables.

=== exercise
id: inj-1
title: What the database receives
--- prompt
A page builds this query by pasting in what the visitor typed:

```text
"SELECT name, budget FROM departments WHERE name = '" + input + "'"
```

An attacker types `x' OR '1'='1`. Write out the **full query the database receives** and run it. (It returns every department, which is the problem.)
--- solution
SELECT name, budget FROM departments WHERE name = 'x' OR '1'='1';
--- hint
Replace `input` with the attacker's text, character for character: `... WHERE name = 'x' OR '1'='1'`.
===

=== exercise
id: inj-2
title: A safe lookup
--- prompt
Look up the department named **Data** (its `name` and `budget`) with a **prepared statement**: put the name in a user variable, `PREPARE` the query with a `?` placeholder, and `EXECUTE` it `USING` the variable.
--- solution
SET @dept = 'Data';
PREPARE find_dept FROM 'SELECT name, budget FROM departments WHERE name = ?';
EXECUTE find_dept USING @dept;
--- hint
Three statements: `SET @dept = 'Data';`, then `PREPARE find_dept FROM 'SELECT ... WHERE name = ?';`, then `EXECUTE find_dept USING @dept;`
===
