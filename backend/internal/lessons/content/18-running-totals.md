title: Running totals and moving averages
module: Window functions
level: Advanced
minutes: 16
summary: Use aggregate window functions and frames for cumulative sums, moving averages and shares of total.
---
Any aggregate (`SUM`, `AVG`, `COUNT`, `MIN`, `MAX`) can be used as a window function with `OVER`.

### Share of total

An empty window `OVER ()` means "all rows of the result":

```sql
SELECT country, COUNT(*) AS customers,
       ROUND(100 * COUNT(*) / SUM(COUNT(*)) OVER (), 1) AS pct
FROM customers
GROUP BY country;
```

`SUM(COUNT(*)) OVER ()` looks odd, but it's simply "the sum of the per-group counts". Window functions are computed **after** `GROUP BY`, so they can aggregate the aggregates.

### Running totals

Adding `ORDER BY` inside `OVER` makes the window **cumulative**: each row sees all rows from the start up to itself.

```sql
SELECT txn_id, txn_at, amount,
       SUM(amount) OVER (ORDER BY txn_at, txn_id) AS balance
FROM transactions
WHERE account_id = 21;
```

### Frames: which rows the window sees

The frame clause says exactly which rows around the current one are included:

```sql
AVG(orders) OVER (ORDER BY day ROWS BETWEEN 6 PRECEDING AND CURRENT ROW)  -- 7-row moving average
```

| Frame | Meaning |
|---|---|
| `ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW` | running total (from the start) |
| `ROWS BETWEEN 6 PRECEDING AND CURRENT ROW` | last 7 rows |
| `ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING` | centered 3-row window |
| `RANGE BETWEEN INTERVAL 6 DAY PRECEDING AND CURRENT ROW` | last 7 *calendar days* (gaps respected) |

> **Gotcha:** with `ORDER BY` but no explicit frame, the default is `RANGE ... CURRENT ROW`, which includes **all rows tied** with the current one. When the ORDER BY values aren't unique, a "running total" can jump. Make the ordering unique or write `ROWS` explicitly.

=== exercise
id: run-1
title: Running revenue 2025
ordered: true
--- prompt
For each **month** (1–12) of 2025, return the month's **revenue** (orders not `cancelled`/`returned`) and **running_total**, the cumulative revenue from January up to and including that month. Round both to 2 decimals. Sort by month.
--- solution
WITH monthly AS (
  SELECT MONTH(o.order_date) AS month,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS revenue
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  WHERE o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
    AND o.status NOT IN ('cancelled', 'returned')
  GROUP BY MONTH(o.order_date)
)
SELECT month, ROUND(revenue, 2) AS revenue,
       ROUND(SUM(revenue) OVER (ORDER BY month), 2) AS running_total
FROM monthly
ORDER BY month;
--- hint
Aggregate per month in a CTE, then SUM(revenue) OVER (ORDER BY month).
===

=== exercise
id: run-2
title: 7-day moving average
ordered: true
--- prompt
For each day of **December 2025**, return **day** (a DATE), the number of **orders** that day, and **ma_7**: the average of orders over that day and the 6 previous days *within December*, rounded to 2 decimals. Sort by day.
--- solution
WITH daily AS (
  SELECT DATE(order_date) AS day, COUNT(*) AS orders
  FROM orders
  WHERE order_date >= '2025-12-01' AND order_date < '2026-01-01'
  GROUP BY DATE(order_date)
)
SELECT day, orders,
       ROUND(AVG(orders) OVER (ORDER BY day ROWS BETWEEN 6 PRECEDING AND CURRENT ROW), 2) AS ma_7
FROM daily
ORDER BY day;
--- hint
ROWS BETWEEN 6 PRECEDING AND CURRENT ROW gives a 7-row window.
===

=== exercise
id: run-3
title: Category share of 2025 revenue
ordered: true
--- prompt
For each product category (the category a product belongs to, via `products.category_id`), return **category** (its name), its 2025 **revenue** (orders not `cancelled`/`returned`, rounded to 2 decimals) and **pct_of_total**: its share of all 2025 revenue as a percentage, rounded to 2 decimals.

Sort by revenue, highest first.
--- solution
WITH rev AS (
  SELECT c.name AS category,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS revenue
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  JOIN products p ON p.product_id = oi.product_id
  JOIN categories c ON c.category_id = p.category_id
  WHERE o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
    AND o.status NOT IN ('cancelled', 'returned')
  GROUP BY c.category_id, c.name
)
SELECT category, ROUND(revenue, 2) AS revenue,
       ROUND(100 * revenue / SUM(revenue) OVER (), 2) AS pct_of_total
FROM rev
ORDER BY revenue DESC;
--- hint
Divide each revenue by SUM(revenue) OVER (), the total across all rows.
===

=== exercise
id: run-4
title: Account balance over time
ordered: true
--- prompt
For account **21**, return every transaction's `txn_id`, `txn_at`, `amount` and the running **balance** (sum of all amounts so far, in order of `txn_at` then `txn_id`). Sort by txn_at, txn_id.
--- solution
SELECT txn_id, txn_at, amount,
       SUM(amount) OVER (ORDER BY txn_at, txn_id) AS balance
FROM transactions
WHERE account_id = 21
ORDER BY txn_at, txn_id;
--- hint
SUM(amount) OVER (ORDER BY txn_at, txn_id). The txn_id tie-breaker makes the order unique.
===
