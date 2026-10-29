title: UNION, INTERSECT and EXCEPT
module: Subqueries & CTEs
level: Intermediate
minutes: 10
summary: Stack, intersect and subtract the results of several queries.
---
Set operators combine the results of two queries **vertically** (rows), where joins combine them horizontally (columns). Both queries must return the **same number of columns**, with compatible types. Column names come from the first query.

| Operator | Result |
|---|---|
| `UNION` | rows from either query, duplicates removed |
| `UNION ALL` | rows from either query, duplicates kept (faster) |
| `INTERSECT` | rows that appear in both |
| `EXCEPT` | rows from the first query that are not in the second |

`INTERSECT` and `EXCEPT` are available from MySQL 8.0.31.

```sql
-- every city where we have an office or a German customer
SELECT city FROM offices
UNION
SELECT city FROM customers WHERE country = 'Germany';
```

### UNION ALL for activity feeds

Different event types often live in different tables. Give them a common shape and stack them:

```sql
SELECT 'order' AS event, order_id AS id, order_date AS at
FROM orders WHERE customer_id = 84
UNION ALL
SELECT 'review', review_id, review_date
FROM reviews WHERE customer_id = 84
ORDER BY at;
```

An `ORDER BY` at the end sorts the **combined** result.

### INTERSECT and EXCEPT

```sql
-- customers who ordered in both 2021 and 2025
SELECT customer_id FROM orders WHERE YEAR(order_date) = 2021
INTERSECT
SELECT customer_id FROM orders WHERE YEAR(order_date) = 2025;
```

`IN` / `EXISTS` (for intersect) and `NOT IN` / `NOT EXISTS` (for except) can express the same thing, and you'll see them more often in older code.

=== exercise
id: set-1
title: Office and customer cities
--- prompt
Return a single column **city** with every distinct city that hosts an **office** or has a **customer from Germany**.
--- solution
SELECT city FROM offices
UNION
SELECT city FROM customers WHERE country = 'Germany';
--- hint
UNION removes the duplicates for you.
===

=== exercise
id: set-2
title: Loyal since year one
--- prompt
How many customers placed orders in **both 2021 and 2025**? Return the count as **customers**.
--- solution
SELECT COUNT(*) AS customers
FROM (
  SELECT customer_id FROM orders WHERE order_date >= '2021-01-01' AND order_date < '2022-01-01'
  INTERSECT
  SELECT customer_id FROM orders WHERE order_date >= '2025-01-01' AND order_date < '2026-01-01'
) AS both_years;
--- hint
INTERSECT the two customer lists inside a derived table, then COUNT(*).
===

=== exercise
id: set-3
title: Dropped products
--- prompt
How many different products were ordered in **2024** but **not in 2025**? Return the count as **products**.
--- solution
SELECT COUNT(*) AS products
FROM (
  SELECT oi.product_id FROM order_items oi JOIN orders o ON o.order_id = oi.order_id
  WHERE o.order_date >= '2024-01-01' AND o.order_date < '2025-01-01'
  EXCEPT
  SELECT oi.product_id FROM order_items oi JOIN orders o ON o.order_id = oi.order_id
  WHERE o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
) AS dropped;
--- hint
EXCEPT keeps the 2024 products that don't appear in the 2025 list.
===

=== exercise
id: set-4
title: Customer 84's activity feed
ordered: true
--- prompt
Build an activity feed for customer **84** with columns **event** (`order` or `review`), **id** (order_id or review_id) and **at** (order_date, or the review_date). Sort by `at`, then event.
--- solution
SELECT 'order' AS event, order_id AS id, order_date AS at
FROM orders WHERE customer_id = 84
UNION ALL
SELECT 'review', review_id, review_date
FROM reviews WHERE customer_id = 84
ORDER BY at, event;
--- hint
UNION ALL two SELECTs with the same three columns, then ORDER BY at the very end.
===
