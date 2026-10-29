title: Common Table Expressions (WITH)
module: Subqueries & CTEs
level: Intermediate
minutes: 14
summary: Name intermediate results with WITH to build complex queries step by step.
---
A **CTE** (Common Table Expression) gives a name to a subquery, placed *before* the main query:

```sql
WITH monthly AS (
  SELECT DATE_FORMAT(o.order_date, '%Y-%m') AS month,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS revenue
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  WHERE o.status NOT IN ('cancelled', 'returned')
  GROUP BY month
)
SELECT month, ROUND(revenue, 2) AS revenue
FROM monthly
ORDER BY month;
```

### Why use CTEs?

- **Readability:** you read a CTE query top to bottom, like a recipe, instead of inside-out.
- **Reuse:** you can reference a CTE several times in the main query.
- **Step-by-step building:** each CTE can use the ones defined before it.

### Chaining several CTEs

Separate them with commas. Only the first one gets the `WITH` keyword:

```sql
WITH spend AS (
  SELECT o.customer_id,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS total
  FROM orders o JOIN order_items oi ON oi.order_id = o.order_id
  WHERE o.status = 'delivered'
  GROUP BY o.customer_id
),
avg_spend AS (
  SELECT AVG(total) AS avg_total FROM spend
)
SELECT s.customer_id, ROUND(s.total, 2) AS total
FROM spend s CROSS JOIN avg_spend a
WHERE s.total > 5 * a.avg_total
ORDER BY total DESC;
```

`CROSS JOIN` with a one-row CTE is a neat way to make a single value (the average) available on every row.

> CTEs exist only while their statement runs. To keep a result around, use a view or a table (later lessons).

=== exercise
id: cte-1
title: Monthly revenue 2025
ordered: true
--- prompt
Using a CTE, return the **month number** (1–12) of 2025 as **month** and that month's **revenue** rounded to 2 decimals. Count only orders whose status is not `cancelled` or `returned`. Sort by month.
--- solution
WITH order_lines AS (
  SELECT MONTH(o.order_date) AS month,
         oi.quantity * oi.unit_price * (1 - oi.discount) AS amount
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  WHERE o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
    AND o.status NOT IN ('cancelled', 'returned')
)
SELECT month, ROUND(SUM(amount), 2) AS revenue
FROM order_lines
GROUP BY month
ORDER BY month;
--- hint
Compute MONTH(order_date) and the line amount in the CTE, then GROUP BY month outside it.
===

=== exercise
id: cte-2
title: Whales of 2025
ordered: true
--- prompt
A "whale" spent more than **10 times** the average customer spend in **2025**. Customer spend = sum of line revenue on their 2025 orders with status `delivered`. Average over the customers who have any such spend.

Return `customer_id` and **spend** rounded to 2 decimals, biggest first.
--- solution
WITH spend AS (
  SELECT o.customer_id,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS spend
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  WHERE o.status = 'delivered'
    AND o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
  GROUP BY o.customer_id
),
stats AS (SELECT AVG(spend) AS avg_spend FROM spend)
SELECT s.customer_id, ROUND(s.spend, 2) AS spend
FROM spend s CROSS JOIN stats
WHERE s.spend > 10 * stats.avg_spend
ORDER BY spend DESC;
--- hint
One CTE for per-customer spend, one for the average, then compare.
===

=== exercise
id: cte-3
title: Departments above average
ordered: true
--- prompt
For each department, compare its number of **current** employees with the average headcount across departments. Return the department `name`, **headcount**, and **above_avg** (`yes` or `no`). Sort by headcount (descending).
--- solution
WITH hc AS (
  SELECT d.name, COUNT(*) AS headcount
  FROM employees e
  JOIN departments d ON d.department_id = e.department_id
  WHERE e.termination_date IS NULL
  GROUP BY d.department_id, d.name
),
avg_hc AS (SELECT AVG(headcount) AS a FROM hc)
SELECT hc.name, hc.headcount,
       CASE WHEN hc.headcount > avg_hc.a THEN 'yes' ELSE 'no' END AS above_avg
FROM hc CROSS JOIN avg_hc
ORDER BY hc.headcount DESC;
--- hint
Reuse the headcount CTE: once for the rows, once for the average.
===
