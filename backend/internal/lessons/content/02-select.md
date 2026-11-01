title: SELECT and FROM
module: Basics
level: Beginner
minutes: 8
summary: Read rows from a table, choose columns, rename them and remove duplicates.
---
Every query starts with two questions: **which table** do I read from, and **which columns** do I want back?

```sql
SELECT name, brand, unit_price
FROM products;
```

`FROM` names the table; `SELECT` lists the columns, separated by commas. The result is a new, temporary table that MySQL sends back to you. Nothing in the database changes.

### Every column with `*`

`SELECT *` returns all columns in the order they were defined. It's handy for exploring, but in real queries list the columns you need: the result is easier to read and doesn't change if someone adds a column later.

```sql
SELECT * FROM departments;
```

### Renaming columns with `AS`

An alias gives a result column a new name. The `AS` keyword is optional, but writing it makes queries easier to read.

```sql
SELECT name AS product, unit_price AS price
FROM products;
```

### Removing duplicates with `DISTINCT`

`DISTINCT` keeps one copy of each distinct row in the result:

```sql
SELECT DISTINCT country FROM customers;
```

With several columns, `DISTINCT` applies to the combination: `SELECT DISTINCT country, city` returns each country/city pair once.

### Limiting the output

Tables here hold up to half a million rows. Add `LIMIT n` to peek at just a few:

```sql
SELECT * FROM orders LIMIT 5;
```

> **Tip:** SQL keywords are case-insensitive (`select` works too), but writing them in UPPER CASE makes queries easier to scan. Table and column names here are lower case.

=== exercise
id: select-1
title: Meet the departments
--- prompt
Return **every column** of the `departments` table.
--- solution
SELECT * FROM departments;
--- hint
`SELECT *` returns all columns.
===

=== exercise
id: select-2
title: Office locations
--- prompt
Return the `city` and `country` of every office in the `offices` table.
--- solution
SELECT city, country FROM offices;
--- hint
List the two columns after SELECT, separated by a comma.
===

=== exercise
id: select-3
title: Where do customers live?
--- prompt
Return each **distinct** `country` that appears in the `customers` table (one row per country).
--- solution
SELECT DISTINCT country FROM customers;
--- hint
Put DISTINCT right after SELECT.
===

=== exercise
id: select-4
title: Renamed columns
ordered: true
--- prompt
Return the first 5 products by `product_id` with three columns: `product_id`, the product `name` renamed to **product**, and `unit_price` renamed to **price**.

Order by `product_id`.
--- solution
SELECT product_id, name AS product, unit_price AS price
FROM products
ORDER BY product_id
LIMIT 5;
--- hint
Use `AS` for the aliases, then `ORDER BY product_id LIMIT 5`.
===
