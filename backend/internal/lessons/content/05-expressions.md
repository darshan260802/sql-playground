title: Expressions, functions and CASE
module: Basics
level: Beginner
minutes: 12
summary: Compute new columns with arithmetic, built-in functions and conditional logic.
---
A `SELECT` list can contain **expressions**, not just column names. They are computed for every row:

```sql
SELECT product_id,
       unit_price - unit_cost              AS margin,
       ROUND(unit_price * 1.2, 2)          AS price_with_vat
FROM products;
```

### Arithmetic

`+ - * /` work as you'd expect. `/` always gives a decimal result, `DIV` gives an integer result and `%` (or `MOD`) gives the remainder. Dividing by zero returns `NULL` with a warning instead of failing.

### Operators at a glance

| Kind | Operators |
|---|---|
| Arithmetic | `+` `-` `*` `/` `DIV` (integer division) `%` or `MOD` (remainder) |
| Comparison | `=` `<>` `!=` `<` `<=` `>` `>=`, and `<=>` (equal, treating two NULLs as equal) |
| Logical | `AND` `OR` `NOT` `XOR`, plus `IN`, `BETWEEN`, `LIKE`, `EXISTS`, `ANY`, `ALL` |
| Bitwise | `&` (and) `\|` (or) `^` (xor) `~` (not) `<<` `>>` (shifts), on integers |

```sql
SELECT 17 DIV 5 AS whole,   -- 3
       17 % 5   AS rest,    -- 2
       6 & 3    AS bits;    -- 2 (binary 110 AND 011 = 010)
```

`%` is handy for sampling and alternating: `WHERE order_id % 2 = 0` keeps even ids. SQL Server's compound operators like `+=` don't exist in MySQL; write `SET x = x + 1`.

### Useful functions

| Function | Example | Result |
|---|---|---|
| `ROUND(x, d)` | `ROUND(12.345, 1)` | 12.3 |
| `CONCAT(a, b, ...)` | `CONCAT(first_name, ' ', last_name)` | Ada Lovelace |
| `UPPER(s)` / `LOWER(s)` | `LOWER('MiXeD')` | mixed |
| `LENGTH(s)` / `CHAR_LENGTH(s)` | `CHAR_LENGTH('São')` | 3 (`LENGTH` counts bytes: 4) |
| `ABS(x)`, `CEIL(x)`, `FLOOR(x)` | `FLOOR(9.9)` | 9 |

### Line totals

In this dataset an order line's revenue is quantity × price × (1 − discount), where `discount` is a fraction (0.15 = 15% off):

```sql
SELECT order_item_id,
       quantity * unit_price * (1 - discount) AS line_total
FROM order_items
WHERE order_id = 6;
```

### CASE: if/else for SQL

`CASE` checks its conditions in order and returns the value of the first one that's true:

```sql
SELECT name, unit_price,
       CASE
         WHEN unit_price < 50  THEN 'cheap'
         WHEN unit_price < 500 THEN 'mid'
         ELSE 'expensive'
       END AS price_band
FROM products;
```

If no `WHEN` matches and there's no `ELSE`, the result is `NULL`. MySQL also has `IF(condition, then, else)` as a compact shortcut for two-way choices.

=== exercise
id: expr-1
title: Headphone margins
ordered: true
--- prompt
For products in category **42** (Headphones), return `product_id`, `name`, `unit_price`, `unit_cost`, the **margin** (`unit_price - unit_cost`) and **margin_pct**: margin as a percentage of `unit_price`, rounded to 1 decimal.

Sort by `margin_pct` (highest first), then `product_id`.
--- solution
SELECT product_id, name, unit_price, unit_cost,
       unit_price - unit_cost AS margin,
       ROUND((unit_price - unit_cost) / unit_price * 100, 1) AS margin_pct
FROM products
WHERE category_id = 42
ORDER BY margin_pct DESC, product_id;
--- hint
You can sort by an alias defined in the SELECT list: `ORDER BY margin_pct DESC`.
===

=== exercise
id: expr-2
title: Customer cards
--- prompt
For customers in **Singapore**, return `customer_id`, their **full_name** (first name, a space, last name) and their email in lower case as **email**.
--- solution
SELECT customer_id,
       CONCAT(first_name, ' ', last_name) AS full_name,
       LOWER(email) AS email
FROM customers
WHERE country = 'Singapore';
--- hint
CONCAT(first_name, ' ', last_name) and LOWER(email).
===

=== exercise
id: expr-3
title: Laptop price bands
--- prompt
For laptops (category **39**), return `product_id`, `name`, `unit_price` and a **band** column:

- `budget` below 800
- `mid-range` from 800 up to (not including) 1500
- `premium` from 1500
--- solution
SELECT product_id, name, unit_price,
       CASE
         WHEN unit_price < 800 THEN 'budget'
         WHEN unit_price < 1500 THEN 'mid-range'
         ELSE 'premium'
       END AS band
FROM products
WHERE category_id = 39;
--- hint
Order the WHEN branches from cheapest to most expensive; the first true one wins.
===

=== exercise
id: expr-4
title: What did order 6 cost?
ordered: true
--- prompt
For the lines of order **6**, return `order_item_id`, `quantity`, `unit_price`, `discount` and **line_total** = quantity × unit price × (1 − discount), rounded to 2 decimals. Sort by `order_item_id`.
--- solution
SELECT order_item_id, quantity, unit_price, discount,
       ROUND(quantity * unit_price * (1 - discount), 2) AS line_total
FROM order_items
WHERE order_id = 6
ORDER BY order_item_id;
--- hint
ROUND(quantity * unit_price * (1 - discount), 2)
===

=== exercise
id: expr-5
title: A quality-check sample
--- prompt
Quality control checks **every 500th order**. Return the `order_id` and `order_date` of orders placed on or after **2025-01-01** whose `order_id` is a multiple of 500.
--- solution
SELECT order_id, order_date
FROM orders
WHERE order_id % 500 = 0
  AND order_date >= '2025-01-01';
--- hint
A number is a multiple of 500 when the remainder of dividing by 500 is 0: `order_id % 500 = 0` (or `MOD(order_id, 500) = 0`).
===
