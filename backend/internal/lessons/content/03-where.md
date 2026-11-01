title: Filtering rows with WHERE
module: Basics
level: Beginner
minutes: 12
summary: Keep only the rows you need using comparisons, AND, OR, NOT, IN, BETWEEN, LIKE and wildcards.
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

### Combining conditions: AND, OR, NOT

- `AND`: both conditions must be true.
- `OR`: at least one must be true.
- `NOT`: reverses a condition.

`AND` binds tighter than `OR`, so use parentheses when you mix them:

```sql
SELECT customer_id, country, loyalty_tier
FROM customers
WHERE (country = 'Canada' OR country = 'Australia')
  AND loyalty_tier = 'platinum';
```

Without the parentheses, MySQL would read it as "Canadians, plus platinum Australians".

`NOT` goes in front of a condition, or inside `NOT IN`, `NOT BETWEEN`, `NOT LIKE` and `IS NOT NULL`:

```sql
-- everyone except the US and Canada
WHERE country NOT IN ('United States', 'Canada')

-- same as: WHERE NOT (unit_price >= 10 AND unit_price <= 20)
WHERE unit_price NOT BETWEEN 10 AND 20

WHERE NOT (status = 'delivered' OR status = 'shipped')
```

### IN and BETWEEN

```sql
-- shorthand for several ORs
WHERE country IN ('Canada', 'Australia')

-- inclusive range (both ends included)
WHERE unit_price BETWEEN 10 AND 20
```

`BETWEEN` also works on text (alphabetical) and dates.

### LIKE and wildcards

`LIKE` matches text against a pattern with two **wildcards**:

| Wildcard | Matches | Example | Matches |
|---|---|---|---|
| `%` | any number of characters, including none | `'%Wireless%'` | *Pro Wireless Mouse*, *Wireless* |
| `_` | exactly one character | `'ELE-____5'` | *ELE-10045*, but not *ELE-1045* |

```sql
SELECT name FROM products WHERE name LIKE 'Zentek%';    -- starts with Zentek
SELECT email FROM customers WHERE email LIKE '%.org';   -- ends with .org
SELECT first_name FROM customers WHERE first_name LIKE '___';  -- exactly three letters
```

- `NOT LIKE` keeps the rows that *don't* match.
- To search for a literal `%` or `_`, escape it with a backslash: `LIKE '%50\%%'` finds text containing *50%*.
- Like `=`, `LIKE` is case-insensitive on this database's text columns.
- Other databases add `[abc]` character lists to `LIKE` (SQL Server). In MySQL, use `REGEXP` for patterns like that: `WHERE sku REGEXP '^(ELE|SPO)-'`.

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

=== exercise
id: where-6
title: Platinum outside the Americas
--- prompt
Return the `customer_id` and `country` of **platinum** customers who signed up on or after **2025-06-01** and do **not** live in the United States, Canada, Mexico or Brazil.
--- solution
SELECT customer_id, country
FROM customers
WHERE loyalty_tier = 'platinum'
  AND signup_at >= '2025-06-01'
  AND country NOT IN ('United States', 'Canada', 'Mexico', 'Brazil');
--- hint
`NOT IN ('United States', 'Canada', 'Mexico', 'Brazil')`, combined with the other two conditions using AND.
===

=== exercise
id: where-7
title: Electronics SKUs ending in 5
--- prompt
Electronics SKUs look like `ELE-` followed by five digits. Return the `sku` and `name` of every product whose SKU is `ELE-`, then **any four characters**, then the digit **5**.
--- solution
SELECT sku, name FROM products WHERE sku LIKE 'ELE-____5';
--- hint
`_` matches exactly one character, so four of them in a row match any four: `LIKE 'ELE-____5'`.
===
