title: Sorting and limiting
module: Basics
level: Beginner
minutes: 8
summary: Put results in a guaranteed order with ORDER BY, then page through them with LIMIT and OFFSET.
---
Without `ORDER BY`, a database may return rows **in any order**, and that order can change between runs. Whenever order matters, say so explicitly.

```sql
SELECT name, unit_price
FROM products
ORDER BY unit_price DESC;
```

`ASC` (ascending, the default) sorts smallest first; `DESC` sorts largest first.

### Several sort keys and tie-breakers

Rows that tie on the first column are sorted by the next one:

```sql
SELECT department_id, hire_date, employee_id
FROM employees
ORDER BY department_id, hire_date DESC, employee_id;
```

Add a **unique column** as the last sort key (like an id) to make the order fully deterministic. Without it, ties can come back in any order.

### Top-N with LIMIT

`ORDER BY ... LIMIT n` is the classic "top N" query:

```sql
-- the 3 most recent orders
SELECT order_id, order_date
FROM orders
ORDER BY order_date DESC
LIMIT 3;
```

### Paging with OFFSET

`LIMIT 10 OFFSET 20` skips 20 rows and returns the next 10, i.e. page 3 with 10 rows per page. MySQL also accepts `LIMIT 20, 10` (offset first), which is easy to misread.

### Where NULLs go

When sorting, MySQL treats `NULL` as smaller than any value: NULLs come **first** with `ASC` and **last** with `DESC`.

> **Clause order:** `SELECT` → `FROM` → `WHERE` → `ORDER BY` → `LIMIT`. Writing them in another order is a syntax error.

=== exercise
id: order-1
title: The 10 most expensive products
ordered: true
--- prompt
Return `name` and `unit_price` of the **10 most expensive** products, most expensive first. Break ties by `product_id`.
--- solution
SELECT name, unit_price
FROM products
ORDER BY unit_price DESC, product_id
LIMIT 10;
--- hint
`ORDER BY unit_price DESC, product_id LIMIT 10`
===

=== exercise
id: order-2
title: Newest customers
ordered: true
--- prompt
Return `customer_id`, `first_name` and `signup_at` for the **5 most recent** customer sign-ups, newest first.
--- solution
SELECT customer_id, first_name, signup_at
FROM customers
ORDER BY signup_at DESC, customer_id DESC
LIMIT 5;
--- hint
Sort by signup_at descending and keep 5 rows.
===

=== exercise
id: order-3
title: Page 3 of the catalog
ordered: true
--- prompt
The catalog shows products sorted by `name` (ties broken by `product_id`), **10 per page**. Return `product_id` and `name` for **page 3**.
--- solution
SELECT product_id, name
FROM products
ORDER BY name, product_id
LIMIT 10 OFFSET 20;
--- hint
Page 3 skips the first 20 rows: `LIMIT 10 OFFSET 20`.
===

=== exercise
id: order-4
title: 2025 hires by department
ordered: true
--- prompt
Return `employee_id`, `department_id` and `hire_date` of employees hired in **2025**, sorted by `department_id` (ascending), then by `hire_date` (newest first), then by `employee_id`.
--- solution
SELECT employee_id, department_id, hire_date
FROM employees
WHERE hire_date >= '2025-01-01' AND hire_date < '2026-01-01'
ORDER BY department_id, hire_date DESC, employee_id;
--- hint
Filter with a date range, then use three sort keys with their own directions.
===
