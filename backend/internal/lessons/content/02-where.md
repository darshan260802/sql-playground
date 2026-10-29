title: Filtering rows with WHERE
module: Basics
level: Beginner
minutes: 12
summary: Keep only the rows you need using comparisons, AND/OR, IN, BETWEEN and LIKE.
---
`WHERE` keeps the rows for which a condition is true. It comes right after `FROM`:

```sql
SELECT name, unit_price
FROM products
WHERE unit_price > 1000;
```

### Comparison operators

| Operator | Meaning |
|---|---|
| `=` | equal |
| `<>` or `!=` | not equal |
| `<`, `<=`, `>`, `>=` | less / greater than |

Text values go in single quotes: `WHERE country = 'India'`. Text comparisons here are **case-insensitive**, so `'india'` matches too.

### Combining conditions

Use `AND` (both must be true), `OR` (either is true) and `NOT`. `AND` binds tighter than `OR`, so use parentheses when you mix them:

```sql
SELECT customer_id, country, loyalty_tier
FROM customers
WHERE (country = 'Canada' OR country = 'Australia')
  AND loyalty_tier = 'platinum';
```

### IN, BETWEEN and LIKE

```sql
-- shorthand for several ORs
WHERE country IN ('Canada', 'Australia')

-- inclusive range (both ends included)
WHERE unit_price BETWEEN 10 AND 20

-- pattern matching: % = any characters, _ = exactly one character
WHERE name LIKE '%Wireless%'
```

### Filtering dates and times

`order_date` is a `DATETIME`, so it includes the time of day. To get everything on one day, use a **half-open range**: greater than or equal to the start, less than the next day.

```sql
SELECT order_id, order_date
FROM orders
WHERE order_date >= '2024-11-29'
  AND order_date <  '2024-11-30';
```

> `BETWEEN '2024-11-29' AND '2024-11-30'` would also include orders placed exactly at midnight on the 30th, and `order_date = '2024-11-29'` only matches `00:00:00`. The half-open range is always right.

=== exercise
id: where-1
title: Premium products
--- prompt
Return `product_id`, `name` and `unit_price` of **active** products (`is_active = 1`) priced **above 1500**.
--- solution
SELECT product_id, name, unit_price
FROM products
WHERE is_active = 1 AND unit_price > 1500;
--- hint
Combine the two conditions with AND.
===

=== exercise
id: where-2
title: VIP customers in Toronto
--- prompt
Return `customer_id`, `first_name`, `last_name` and `loyalty_tier` of customers whose `city` is **Toronto** and whose loyalty tier is **gold or platinum**.
--- solution
SELECT customer_id, first_name, last_name, loyalty_tier
FROM customers
WHERE city = 'Toronto'
  AND loyalty_tier IN ('gold', 'platinum');
--- hint
`loyalty_tier IN ('gold', 'platinum')`. If you use OR instead, wrap it in parentheses.
===

=== exercise
id: where-3
title: Find the wireless gear
--- prompt
Return `product_id` and `name` of every product whose name contains the word **Wireless**.
--- solution
SELECT product_id, name
FROM products
WHERE name LIKE '%Wireless%';
--- hint
LIKE with % on both sides matches the text anywhere in the name.
===

=== exercise
id: where-4
title: Black Friday 2024
--- prompt
Return `order_id`, `customer_id` and `order_date` of orders placed on **Black Friday, 29 November 2024** (any time that day) through the **mobile** channel.
--- solution
SELECT order_id, customer_id, order_date
FROM orders
WHERE order_date >= '2024-11-29'
  AND order_date < '2024-11-30'
  AND channel = 'mobile';
--- hint
Use a half-open range on order_date: `>= '2024-11-29' AND < '2024-11-30'`.
===

=== exercise
id: where-5
title: Current legal and people teams
--- prompt
Return `employee_id`, `first_name`, `last_name` and `department_id` of employees in departments **9 or 10** (People and Legal) who still work at the company (`termination_date` is empty, i.e. NULL).
--- solution
SELECT employee_id, first_name, last_name, department_id
FROM employees
WHERE department_id IN (9, 10)
  AND termination_date IS NULL;
--- hint
Use `IS NULL` for missing values: `= NULL` never matches anything.
===
