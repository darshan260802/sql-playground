title: Working with NULL
module: Basics
level: Beginner
minutes: 10
summary: Understand missing values, test for them correctly and replace them with defaults.
---
`NULL` means **unknown or missing**: a customer who didn't give a phone number, an employee who hasn't left yet. It is not zero and not an empty string.

### Comparisons with NULL are never true

Any comparison involving `NULL` gives `NULL` (unknown), and `WHERE` only keeps rows where the condition is *true*:

```sql
SELECT COUNT(*) FROM customers WHERE phone = NULL;   -- always 0!
SELECT COUNT(*) FROM customers WHERE phone IS NULL;  -- correct
```

Use `IS NULL` and `IS NOT NULL`. MySQL also has the NULL-safe equality operator `<=>`, which treats two NULLs as equal.

### The NOT IN trap

`x NOT IN (1, 2, NULL)` is never true, because `x <> NULL` is unknown. Be careful with `NOT IN (subquery)` when the subquery can return NULLs. `NOT EXISTS` (covered later) avoids the problem.

### Replacing NULLs

`COALESCE(a, b, c, ...)` returns its first argument that isn't NULL. `IFNULL(a, b)` is MySQL's two-argument version.

```sql
SELECT order_id, COALESCE(coupon_code, 'no coupon') AS coupon
FROM orders
LIMIT 10;
```

`NULLIF(a, b)` does the reverse: it returns NULL when `a = b`. Its classic use is avoiding division by zero: `x / NULLIF(y, 0)`.

### NULL in arithmetic and text

`5 + NULL` is `NULL`, and so is `CONCAT('a', NULL)`. `CONCAT_WS(' ', a, b)` ("with separator") skips NULL arguments instead.

=== exercise
id: null-1
title: Unreachable customers
--- prompt
Return `customer_id`, `first_name` and `last_name` of customers in the city **Edinburgh** who have **no phone number**.
--- solution
SELECT customer_id, first_name, last_name
FROM customers
WHERE city = 'Edinburgh' AND phone IS NULL;
--- hint
`phone IS NULL`, not `phone = NULL`.
===

=== exercise
id: null-2
title: Coupon report
ordered: true
--- prompt
For the orders of customer **84**, return `order_id`, `order_date` and **coupon**: the `coupon_code`, or the text `none` when no coupon was used. Sort by `order_date`.
--- solution
SELECT order_id, order_date, COALESCE(coupon_code, 'none') AS coupon
FROM orders
WHERE customer_id = 84
ORDER BY order_date;
--- hint
COALESCE(coupon_code, 'none')
===

=== exercise
id: null-3
title: Who left in 2025?
ordered: true
--- prompt
Return `employee_id`, `first_name`, `last_name` and `termination_date` of employees who **left during 2025**, in order of `termination_date`, then `employee_id`.
--- solution
SELECT employee_id, first_name, last_name, termination_date
FROM employees
WHERE termination_date >= '2025-01-01' AND termination_date < '2026-01-01'
ORDER BY termination_date, employee_id;
--- hint
A date range on termination_date skips the NULLs automatically.
===

=== exercise
id: null-4
title: Missing product details
--- prompt
Some products have no `attributes` at all. Return their `product_id` and `name`.
--- solution
SELECT product_id, name
FROM products
WHERE attributes IS NULL;
--- hint
Test the column with IS NULL.
===
