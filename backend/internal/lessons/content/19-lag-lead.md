title: LAG, LEAD and FIRST_VALUE
module: Window functions
level: Advanced
minutes: 14
summary: Compare each row with the previous or next one for growth rates, gaps and changes.
---
Navigation functions read a value **from another row** of the window:

| Function | Returns |
|---|---|
| `LAG(x, n, default)` | `x` from *n* rows **before** (n = 1 if omitted) |
| `LEAD(x, n, default)` | `x` from *n* rows **after** |
| `FIRST_VALUE(x)` | `x` from the first row of the frame |
| `LAST_VALUE(x)` | `x` from the last row of the frame |
| `NTH_VALUE(x, n)` | `x` from the n-th row of the frame |

### Period-over-period growth

```sql
WITH monthly AS (
  SELECT DATE_FORMAT(order_date, '%Y-%m') AS month, COUNT(*) AS orders
  FROM orders GROUP BY month
)
SELECT month, orders,
       LAG(orders) OVER (ORDER BY month) AS prev_orders,
       ROUND(100 * (orders / LAG(orders) OVER (ORDER BY month) - 1), 1) AS growth_pct
FROM monthly;
```

The first row has no previous row, so `LAG` returns NULL (or the default you pass as the third argument).

### Time between events

`DATEDIFF(order_date, LAG(order_date) OVER (PARTITION BY customer_id ORDER BY order_date))` gives the days since each customer's previous order. That's the basis of churn and repeat-purchase analysis.

### The LAST_VALUE trap

With an `ORDER BY` window, the default frame ends at the **current row**, so `LAST_VALUE` just returns the current row's value. Widen the frame to see the true last row:

```sql
LAST_VALUE(close) OVER (PARTITION BY stock_id ORDER BY trade_date
                        ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
```

=== exercise
id: lag-1
title: Month-over-month growth 2025
ordered: true
--- prompt
For each **month** (1–12) of 2025, return the number of **orders**, **prev_orders** (the previous month's count, NULL for January) and **growth_pct** = (orders / prev_orders − 1) × 100, rounded to 1 decimal. Sort by month.
--- solution
WITH monthly AS (
  SELECT MONTH(order_date) AS month, COUNT(*) AS orders
  FROM orders
  WHERE order_date >= '2025-01-01' AND order_date < '2026-01-01'
  GROUP BY MONTH(order_date)
)
SELECT month, orders,
       LAG(orders) OVER (ORDER BY month) AS prev_orders,
       ROUND((orders / LAG(orders) OVER (ORDER BY month) - 1) * 100, 1) AS growth_pct
FROM monthly
ORDER BY month;
--- hint
LAG(orders) OVER (ORDER BY month)
===

=== exercise
id: lag-2
title: Days between orders
ordered: true
--- prompt
For customer **84**, return `order_id`, `order_date` and **days_since_prev**: whole days since that customer's previous order (`DATEDIFF`), NULL for the first one. Sort by order_date.
--- solution
SELECT order_id, order_date,
       DATEDIFF(order_date, LAG(order_date) OVER (ORDER BY order_date)) AS days_since_prev
FROM orders
WHERE customer_id = 84
ORDER BY order_date;
--- hint
DATEDIFF(order_date, LAG(order_date) OVER (ORDER BY order_date))
===

=== exercise
id: lag-3
title: Salary history with raises
ordered: true
--- prompt
For employee **4**, return each salary period's `from_date`, `salary`, **prev_salary** and **raise_pct** (percentage increase over the previous salary, rounded to 2 decimals). Sort by from_date.
--- solution
SELECT from_date, salary,
       LAG(salary) OVER (ORDER BY from_date) AS prev_salary,
       ROUND((salary / LAG(salary) OVER (ORDER BY from_date) - 1) * 100, 2) AS raise_pct
FROM salaries
WHERE employee_id = 4
ORDER BY from_date;
--- hint
Same LAG pattern, ordered by from_date.
===

=== exercise
id: lag-4
title: Total return per stock
ordered: true
--- prompt
For every stock, return `ticker`, **first_close** (close on its first trading day in the data), **last_close** (close on its last trading day) and **return_pct** = (last / first − 1) × 100, rounded to 2 decimals. One row per stock, best performer first.
--- solution
WITH bounds AS (
  SELECT DISTINCT p.stock_id,
         FIRST_VALUE(p.close) OVER w AS first_close,
         LAST_VALUE(p.close) OVER w AS last_close
  FROM stock_prices p
  WINDOW w AS (PARTITION BY p.stock_id ORDER BY p.trade_date
               ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING)
)
SELECT s.ticker, b.first_close, b.last_close,
       ROUND((b.last_close / b.first_close - 1) * 100, 2) AS return_pct
FROM bounds b
JOIN stocks s ON s.stock_id = b.stock_id
ORDER BY return_pct DESC;
--- hint
FIRST_VALUE and LAST_VALUE over a frame covering the whole partition, then DISTINCT to keep one row per stock.
===
