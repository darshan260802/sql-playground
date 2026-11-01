title: What is SQL?
module: Basics
level: Beginner
minutes: 8
summary: What a relational database is, how SQL statements are written, and how to add comments.
---
**SQL** (Structured Query Language) is the language for working with **relational databases**: MySQL, PostgreSQL, SQL Server, Oracle, SQLite and many more. With it you read data, add, change and delete rows, and create the tables that hold them. This playground runs **MySQL 8**, and everything you learn here works the same way on a real MySQL server.

### Tables, rows and columns

A relational database stores data in **tables**. A table looks like a spreadsheet with strict rules:

- every **column** has a name and a data type (a number, text, a date…);
- every **row** is one record: one customer, one order, one employee;
- a **primary key** column (like `customer_id`) identifies each row uniquely;
- tables are linked through **keys**: `orders.customer_id` says which customer placed an order.

The database behind this playground has 24 tables in three areas: an online store (`customers`, `orders`, `products`…), an HR system (`employees`, `departments`, `salaries`…) and a bank (`accounts`, `transactions`, `stock_prices`…). The **Tables** panel in the playground lists them all.

### Your first statement

A SQL **statement** tells the database what you want, not how to get it:

```sql
SELECT first_name, country
FROM customers
WHERE country = 'Japan';
```

It reads almost like English: *select these columns from this table where this is true*. The database works out the fastest way to find the rows.

### Syntax rules

- **Keywords are case-insensitive.** `SELECT`, `select` and `Select` are the same. Writing keywords in capitals is a convention that makes queries easier to read.
- **End each statement with a semicolon** `;`. It's required when you run several statements at once, and a good habit always.
- **Text goes in single quotes**: `'Japan'`. Numbers don't: `WHERE customer_id = 42`.
- **Spacing and line breaks don't matter.** Put each clause on its own line so long queries stay readable.
- **Names with spaces or reserved words** go in backticks in MySQL: `` `order` ``. It's easier to avoid them.

### The main kinds of statements

| Kind | Statements | What they do |
|---|---|---|
| Querying | `SELECT` | read data |
| Changing data | `INSERT`, `UPDATE`, `DELETE` | add, change and remove rows |
| Defining structure | `CREATE`, `ALTER`, `DROP` | create and change tables, views, indexes |
| Transactions | `START TRANSACTION`, `COMMIT`, `ROLLBACK` | group changes so they succeed or fail together |

Signed-out visitors can run `SELECT` on the original data. A free account gives you a private copy of the database where every kind of statement works.

### Comments

Comments are notes for people; the database ignores them. Use them to explain *why* a query does something:

```sql
-- a single-line comment runs to the end of the line
SELECT name, budget   # MySQL also accepts # for single-line comments
FROM departments
/* a multi-line comment
   can span several lines */
WHERE budget > 10000000;
```

`--` needs a space after it in MySQL. Commenting out a line is also a quick way to switch part of a query off while you experiment.

=== exercise
id: intro-1
title: Hello, SQL
--- prompt
Your first query doesn't even need a table. Return the text **Hello, SQL!** in a column named **greeting**.
--- solution
SELECT 'Hello, SQL!' AS greeting;
--- hint
`SELECT 'some text' AS column_name;` Text goes in single quotes.
===

=== exercise
id: intro-2
title: How many customers?
--- prompt
Count the rows in the `customers` table. Type this query, with or without the comment:

```sql
-- every customer in the store
SELECT COUNT(*) AS customers FROM customers;
```

You'll learn `COUNT` properly in the aggregation lessons.
--- solution
-- every customer in the store
SELECT COUNT(*) AS customers FROM customers;
--- hint
Copy the query exactly, then press Check. The comment line is optional.
===
