title: INSERT, UPDATE and DELETE
module: Changing data
level: Intermediate
minutes: 16
summary: Add, change and remove rows safely, including upserts and updates driven by joins.
---
These exercises **change data**, so they need your own copy of the database. Sign in, and every change only affects your copy. You can restore the original at any time with **Reset database**.

> When you press **Check**, your statements run inside a transaction that is **rolled back**, so checking never changes your data. Run them in the playground (target: *My database*) to apply them for real.

### INSERT

```sql
INSERT INTO suppliers (supplier_id, name, country, contact_email, rating, since)
VALUES (901, 'Example Traders', 'Spain', NULL, 4.0, '2026-02-01'),
       (902, 'Sample Goods',    'Italy', NULL, 3.5, '2026-02-01');
```

List the columns explicitly: the statement then keeps working if columns are added or reordered, and omitted columns get their default. `INSERT INTO ... SELECT ...` copies rows from a query.

### UPDATE

```sql
UPDATE products
SET unit_price = ROUND(unit_price * 1.05, 2)
WHERE category_id = 40;
```

**Always write the `WHERE` first.** An UPDATE without WHERE changes every row. A good habit is to run the same `WHERE` as a `SELECT` before turning it into an update.

MySQL can update using a join:

```sql
UPDATE customers c
JOIN (SELECT customer_id FROM orders GROUP BY customer_id HAVING COUNT(*) >= 20) big
  ON big.customer_id = c.customer_id
SET c.loyalty_tier = 'platinum';
```

### DELETE

```sql
DELETE FROM reviews WHERE rating = 1 AND helpful_votes = 0;
```

Foreign keys protect related data. Deleting a customer who still has orders fails with error 1451 until the orders are gone. `TRUNCATE TABLE t` empties a table instantly, but it can't be rolled back and doesn't work when other tables reference it.

### Upserts

`INSERT ... ON DUPLICATE KEY UPDATE` inserts a row, or updates the existing row when the primary key (or a unique key) already exists:

```sql
INSERT INTO departments (department_id, name, budget)
VALUES (4, 'Data', 3500000) AS new
ON DUPLICATE KEY UPDATE budget = new.budget;
```

The `AS new` alias (MySQL 8.0.19+) refers to the values you tried to insert.

=== exercise
id: dml-1
title: A new supplier
kind: dml
--- prompt
Add a supplier with `supplier_id` **900**, name **Practice Supplies Ltd**, country **India**, no contact email, rating **4.5** and `since` **2026-01-15**.
--- solution
INSERT INTO suppliers (supplier_id, name, country, contact_email, rating, since)
VALUES (900, 'Practice Supplies Ltd', 'India', NULL, 4.5, '2026-01-15');
--- check
SELECT supplier_id, name, country, contact_email, rating, since FROM suppliers WHERE supplier_id = 900;
--- hint
List the columns, then VALUES (...). "No contact email" means NULL.
===

=== exercise
id: dml-2
title: Price increase
kind: dml
--- prompt
Raise the `unit_price` of every product in category **41** (Keyboards & Mice) by **10%**, rounded to 2 decimals.
--- solution
UPDATE products
SET unit_price = ROUND(unit_price * 1.10, 2)
WHERE category_id = 41;
--- check
SELECT product_id, unit_price FROM products WHERE category_id IN (40, 41, 42) ORDER BY product_id;
--- hint
SET unit_price = ROUND(unit_price * 1.10, 2) with a WHERE on category_id.
===

=== exercise
id: dml-3
title: Prune old neutral reviews
kind: dml
--- prompt
Delete reviews that have a rating of **3**, **zero** helpful votes and were written **before 2022**.
--- solution
DELETE FROM reviews
WHERE rating = 3 AND helpful_votes = 0 AND review_date < '2022-01-01';
--- check
SELECT rating, COUNT(*) AS reviews FROM reviews GROUP BY rating ORDER BY rating;
--- hint
Three conditions joined with AND. Try it as a SELECT COUNT(*) first.
===

=== exercise
id: dml-4
title: Promote frequent buyers
kind: dml
--- prompt
Upgrade every **silver** customer who has placed **15 or more orders** (any status) to the **gold** tier.
--- solution
UPDATE customers c
JOIN (SELECT customer_id FROM orders GROUP BY customer_id HAVING COUNT(*) >= 15) f
  ON f.customer_id = c.customer_id
SET c.loyalty_tier = 'gold'
WHERE c.loyalty_tier = 'silver';
--- check
SELECT loyalty_tier, COUNT(*) AS customers FROM customers GROUP BY loyalty_tier ORDER BY loyalty_tier;
--- hint
UPDATE customers joined to a derived table of frequent customers, with WHERE c.loyalty_tier = 'silver'.
===

=== exercise
id: dml-5
title: Upsert departments
kind: dml
--- prompt
With a **single** `INSERT ... ON DUPLICATE KEY UPDATE` statement:

- set the budget of department **4** (Data) to **3,500,000**, and
- create department **15** named **Research** with a budget of **1,200,000**.
--- solution
INSERT INTO departments (department_id, name, budget)
VALUES (4, 'Data', 3500000), (15, 'Research', 1200000) AS new
ON DUPLICATE KEY UPDATE budget = new.budget;
--- check
SELECT department_id, name, budget FROM departments ORDER BY department_id;
--- hint
Insert both rows; for the existing one, ON DUPLICATE KEY UPDATE budget = new.budget.
===
