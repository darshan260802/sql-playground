title: Dates and times
module: Dates, text & JSON
level: Intermediate
minutes: 14
summary: Extract parts of dates, do date arithmetic and bucket events by day, week and month.
---
MySQL has `DATE` (2025-12-31), `DATETIME` (2025-12-31 23:59:59) and `TIMESTAMP` types. Date literals are written as strings in `'YYYY-MM-DD'` format.

> The data in this playground runs until **2025-12-31**. Functions like `CURDATE()` and `NOW()` return *today's* real date, so for "the last 30 days" anchor on the data's end date instead.

### Extracting parts

| Function | Example result for `2025-12-24 18:05:00` |
|---|---|
| `DATE(d)` | 2025-12-24 |
| `YEAR(d)`, `MONTH(d)`, `DAY(d)` | 2025, 12, 24 |
| `HOUR(d)` | 18 |
| `DAYNAME(d)`, `MONTHNAME(d)` | Wednesday, December |
| `DAYOFWEEK(d)` | 4 (1 = Sunday) |
| `WEEKDAY(d)` | 2 (0 = Monday) |
| `QUARTER(d)` | 4 |
| `DATE_FORMAT(d, '%Y-%m')` | 2025-12 |

### Arithmetic

```sql
SELECT DATE_ADD('2025-12-31', INTERVAL 30 DAY),   -- 2026-01-30
       '2025-12-31' - INTERVAL 1 MONTH,           -- 2025-11-30
       DATEDIFF('2025-12-31', '2025-12-01'),      -- 30 (days)
       TIMESTAMPDIFF(HOUR, '2025-12-01 08:00', '2025-12-02 10:30'),  -- 26
       LAST_DAY('2025-02-10');                    -- 2025-02-28
```

`TIMESTAMPDIFF(unit, start, end)` accepts SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, QUARTER and YEAR. It counts **completed** units, which makes it exactly right for ages.

### Bucketing

```sql
SELECT DATE_FORMAT(order_date, '%Y-%m') AS month, COUNT(*) AS orders
FROM orders
GROUP BY month
ORDER BY month;
```

Remember the half-open range from the WHERE lesson: filtering on `order_date >= '2025-01-01' AND order_date < '2026-01-01'` lets MySQL use an index, while `WHERE YEAR(order_date) = 2025` has to compute `YEAR()` for every row.

=== exercise
id: date-1
title: Busiest weekday
ordered: true
--- prompt
For orders placed in **2025**, return each **day_name** (`DAYNAME`) and its number of **orders**, busiest day first.
--- solution
SELECT DAYNAME(order_date) AS day_name, COUNT(*) AS orders
FROM orders
WHERE order_date >= '2025-01-01' AND order_date < '2026-01-01'
GROUP BY DAYNAME(order_date)
ORDER BY orders DESC;
--- hint
GROUP BY DAYNAME(order_date).
===

=== exercise
id: date-2
title: Delivery speed by carrier
ordered: true
--- prompt
For **delivered** shipments (`delivered_at` not NULL), return each `carrier` with **avg_hours**: the average time from `shipped_at` to `delivered_at` in hours (`TIMESTAMPDIFF`), rounded to 1 decimal. Fastest first.
--- solution
SELECT carrier,
       ROUND(AVG(TIMESTAMPDIFF(HOUR, shipped_at, delivered_at)), 1) AS avg_hours
FROM shipments
WHERE delivered_at IS NOT NULL
GROUP BY carrier
ORDER BY avg_hours;
--- hint
AVG(TIMESTAMPDIFF(HOUR, shipped_at, delivered_at))
===

=== exercise
id: date-3
title: Age at signup
ordered: true
--- prompt
For customers with a known `birth_date`, compute their age in whole years **at signup** (`TIMESTAMPDIFF(YEAR, birth_date, signup_at)`). Return each `country` with **avg_age** rounded to 1 decimal, oldest average first.
--- solution
SELECT country, ROUND(AVG(TIMESTAMPDIFF(YEAR, birth_date, signup_at)), 1) AS avg_age
FROM customers
WHERE birth_date IS NOT NULL
GROUP BY country
ORDER BY avg_age DESC;
--- hint
TIMESTAMPDIFF(YEAR, birth_date, signup_at) is the age at signup.
===

=== exercise
id: date-4
title: The last 30 days
--- prompt
How many orders were placed in the **30 days up to the end of the data**, i.e. from `2025-12-31` minus 30 days (2025-12-01 00:00) up to the end of 2025-12-31? Return the count as **orders**.
--- solution
SELECT COUNT(*) AS orders
FROM orders
WHERE order_date >= DATE('2025-12-31') - INTERVAL 30 DAY
  AND order_date < DATE('2025-12-31') + INTERVAL 1 DAY;
--- hint
Anchor on '2025-12-31' (not CURDATE()) and use INTERVAL arithmetic.
===

=== exercise
id: date-5
title: Monthly signups
ordered: true
--- prompt
Return each **month** of 2025 formatted as `YYYY-MM` (e.g. `2025-01`) with the number of customer **signups**. Sort by month.
--- solution
SELECT DATE_FORMAT(signup_at, '%Y-%m') AS month, COUNT(*) AS signups
FROM customers
WHERE signup_at >= '2025-01-01' AND signup_at < '2026-01-01'
GROUP BY month
ORDER BY month;
--- hint
DATE_FORMAT(signup_at, '%Y-%m')
===
