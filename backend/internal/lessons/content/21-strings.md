title: Text functions and data cleaning
module: Dates, text & JSON
level: Intermediate
minutes: 14
summary: Slice, search and clean messy text with string functions and regular expressions.
---
Real data is messy, and so is this dataset, on purpose. A few customers have a city typed in lower case or with a trailing space, and phone numbers come in several formats.

### Core string functions

| Function | Example | Result |
|---|---|---|
| `TRIM(s)` / `LTRIM` / `RTRIM` | `TRIM('  Leeds ')` | `Leeds` |
| `LEFT(s, n)` / `RIGHT(s, n)` | `LEFT('Bengaluru', 3)` | `Ben` |
| `SUBSTRING(s, pos, len)` | `SUBSTRING('2025-12-31', 6, 2)` | `12` |
| `SUBSTRING_INDEX(s, delim, n)` | `SUBSTRING_INDEX('a@b.com', '@', -1)` | `b.com` |
| `LOCATE(sub, s)` | `LOCATE('@', 'a@b.com')` | 2 |
| `REPLACE(s, from, to)` | `REPLACE('a-b-c', '-', '')` | `abc` |
| `LPAD(s, n, pad)` | `LPAD('7', 3, '0')` | `007` |

### Regular expressions

MySQL 8 has full regex support:

```sql
-- keep only digits
SELECT phone, REGEXP_REPLACE(phone, '[^0-9]', '') AS digits
FROM customers WHERE country = 'United States' AND phone IS NOT NULL LIMIT 5;

-- names ending in a model code such as "X2" or "S5"
SELECT name FROM products WHERE name REGEXP ' [A-Z][0-9]$';
```

Other regex functions: `REGEXP_LIKE`, `REGEXP_SUBSTR` and `REGEXP_INSTR`.

### Why messy text breaks GROUP BY

`'London'`, `'london'` and `'London '` look alike to a person. The comparison collation used here ignores case, but **not trailing spaces**, so `'London '` becomes its own group. Clean the value before grouping:

```sql
SELECT TRIM(city) AS city, COUNT(*)
FROM customers
WHERE country = 'United Kingdom'
GROUP BY TRIM(city);
```

> Case-insensitive comparison comes from the column's collation (`utf8mb4_0900_ai_ci`, where *ci* = case-insensitive and *ai* = accent-insensitive). So `'Sao Paulo' = 'São Paulo'` is true here!

=== exercise
id: str-1
title: Email providers
ordered: true
--- prompt
Return each email **domain** (everything after the `@`, in lower case) with the number of **customers** using it. Most popular first, ties by domain.
--- solution
SELECT LOWER(SUBSTRING_INDEX(email, '@', -1)) AS domain, COUNT(*) AS customers
FROM customers
GROUP BY domain
ORDER BY customers DESC, domain;
--- hint
SUBSTRING_INDEX(email, '@', -1) returns the part after the last @.
===

=== exercise
id: str-2
title: Clean UK cities
ordered: true
--- prompt
For customers in the **United Kingdom**, group them by a cleaned city name: trimmed, with the first letter upper case and the rest lower case (so `london`, `London ` and `London` all become `London`). Return **city** and **customers**, most customers first.
--- solution
SELECT CONCAT(UPPER(LEFT(TRIM(city), 1)), LOWER(SUBSTRING(TRIM(city), 2))) AS city,
       COUNT(*) AS customers
FROM customers
WHERE country = 'United Kingdom'
GROUP BY 1
ORDER BY customers DESC, city;
--- hint
CONCAT(UPPER(LEFT(TRIM(city), 1)), LOWER(SUBSTRING(TRIM(city), 2)))
===

=== exercise
id: str-3
title: Normalize phone numbers
ordered: true
--- prompt
For the first 10 customers (by `customer_id`) from the **United States** who have a phone number, return `customer_id`, `phone` and **digits**: the **last 10 digits** of the phone with every non-digit character removed. Sort by customer_id.
--- solution
SELECT customer_id, phone, RIGHT(REGEXP_REPLACE(phone, '[^0-9]', ''), 10) AS digits
FROM customers
WHERE country = 'United States' AND phone IS NOT NULL
ORDER BY customer_id
LIMIT 10;
--- hint
REGEXP_REPLACE(phone, '[^0-9]', '') then RIGHT(..., 10) to drop a leading country code.
===

=== exercise
id: str-4
title: Model codes
--- prompt
How many products have a name that **ends** with a space followed by one capital letter and one digit (like `... X2` or `... S5`)? Return the count as **products**.
--- solution
SELECT COUNT(*) AS products
FROM products
WHERE REGEXP_LIKE(name, ' [A-Z][0-9]$', 'c');
--- hint
REGEXP_LIKE(name, ' [A-Z][0-9]$', 'c'): the 'c' flag makes the match case-sensitive.
===
