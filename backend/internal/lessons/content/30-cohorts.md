title: Cohort retention
module: Analytics case studies
level: Advanced
minutes: 20
summary: Group customers by when they started and measure how many come back.
---
A **cohort** is a group of customers who started in the same period, for example everyone whose *first order* was in March 2025. Comparing cohorts over time answers questions like "are customers acquired this year more loyal than last year's?"

The recipe has three steps:

1. **Find each customer's start**: their first order date.
2. **Label every later event** with how long after the start it happened.
3. **Count active customers** per cohort and period.

### Step 1: first orders

```sql
WITH firsts AS (
  SELECT customer_id, MIN(order_date) AS first_order
  FROM orders
  GROUP BY customer_id
)
SELECT DATE_FORMAT(first_order, '%Y-%m') AS cohort, COUNT(*) AS new_customers
FROM firsts
GROUP BY cohort
ORDER BY cohort;
```

### Step 2: months since the start

Compare the *first day of the month* of both dates, so an order on 31 January and one on 1 February count as one month apart:

```sql
TIMESTAMPDIFF(MONTH,
              DATE_FORMAT(first_order, '%Y-%m-01'),
              DATE_FORMAT(order_date,  '%Y-%m-01')) AS month_number
```

### Step 3: the retention matrix

Count **distinct** customers per (cohort, month_number). Month 0 is always the full cohort. Divide the later months by it to get retention rates, a natural fit for `FIRST_VALUE(...) OVER (PARTITION BY cohort ORDER BY month_number)`.

> Retention questions always need a clear definition: "came back within 90 days", "active in month N", "made a second purchase ever". Write it down before you write the SQL.

=== exercise
id: cohort-1
title: New customers per month (2025)
ordered: true
--- prompt
A customer's cohort is the month of their **first order**. For first orders placed in **2025**, return **cohort** (`YYYY-MM`) and **new_customers**. Sort by cohort.
--- solution
WITH firsts AS (
  SELECT customer_id, MIN(order_date) AS first_order
  FROM orders
  GROUP BY customer_id
)
SELECT DATE_FORMAT(first_order, '%Y-%m') AS cohort, COUNT(*) AS new_customers
FROM firsts
WHERE first_order >= '2025-01-01' AND first_order < '2026-01-01'
GROUP BY cohort
ORDER BY cohort;
--- hint
MIN(order_date) per customer first, then filter and group by month.
===

=== exercise
id: cohort-2
title: Came back within 90 days
ordered: true
--- prompt
For customers whose first order was in **2024**, grouped by first-order month (**cohort**, `YYYY-MM`), return:

- **customers**: cohort size
- **repeaters**: customers who placed **another order within 90 days** after their first one (later than the first order, at most 90 days after it)
- **repeat_rate_pct**: repeaters / customers × 100, rounded to 1 decimal

Sort by cohort.
--- solution
WITH firsts AS (
  SELECT customer_id, MIN(order_date) AS first_order
  FROM orders
  GROUP BY customer_id
),
flags AS (
  SELECT f.customer_id, DATE_FORMAT(f.first_order, '%Y-%m') AS cohort,
         EXISTS (SELECT 1 FROM orders o
                 WHERE o.customer_id = f.customer_id
                   AND o.order_date > f.first_order
                   AND o.order_date <= f.first_order + INTERVAL 90 DAY) AS repeated
  FROM firsts f
  WHERE f.first_order >= '2024-01-01' AND f.first_order < '2025-01-01'
)
SELECT cohort, COUNT(*) AS customers, SUM(repeated) AS repeaters,
       ROUND(100 * AVG(repeated), 1) AS repeat_rate_pct
FROM flags
GROUP BY cohort
ORDER BY cohort;
--- hint
EXISTS (...) returns 1 or 0, so you can SUM and AVG it.
===

=== exercise
id: cohort-3
title: Retention matrix, Q1 2025 cohorts
ordered: true
--- prompt
For customers whose first order was in **January, February or March 2025**, return the retention matrix: **cohort** (`YYYY-MM`), **month_number** (months between the first order's month and the order's month, 0 = first month) and **active_customers** (distinct customers who ordered in that month).

Sort by cohort, then month_number.
--- solution
WITH firsts AS (
  SELECT customer_id, MIN(order_date) AS first_order
  FROM orders
  GROUP BY customer_id
)
SELECT DATE_FORMAT(f.first_order, '%Y-%m') AS cohort,
       TIMESTAMPDIFF(MONTH, DATE_FORMAT(f.first_order, '%Y-%m-01'), DATE_FORMAT(o.order_date, '%Y-%m-01')) AS month_number,
       COUNT(DISTINCT o.customer_id) AS active_customers
FROM firsts f
JOIN orders o ON o.customer_id = f.customer_id
WHERE f.first_order >= '2025-01-01' AND f.first_order < '2025-04-01'
GROUP BY cohort, month_number
ORDER BY cohort, month_number;
--- hint
Join every order back to the customer's first order, compute month_number, then COUNT(DISTINCT customer_id).
===
