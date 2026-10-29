title: Customer segmentation (RFM)
module: Analytics case studies
level: Advanced
minutes: 18
summary: Score customers on recency, frequency and monetary value, then turn the scores into segments.
---
**RFM** is a classic, simple segmentation used by marketing teams everywhere:

- **Recency:** how long ago did they last buy? (lower is better)
- **Frequency:** how many orders have they placed?
- **Monetary:** how much have they spent?

### Step 1: the base table

Measure all three per customer. Our data ends on 2025-12-31, so recency is measured from there:

```sql
SELECT o.customer_id,
       DATEDIFF('2025-12-31', MAX(o.order_date))                AS recency_days,
       COUNT(DISTINCT o.order_id)                                AS frequency,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS monetary
FROM orders o
JOIN order_items oi ON oi.order_id = o.order_id
WHERE o.status = 'delivered'
GROUP BY o.customer_id;
```

`COUNT(DISTINCT o.order_id)` matters: the join produces one row per order **line**, so `COUNT(*)` would count lines, not orders.

### Step 2: scores with NTILE

`NTILE(5)` turns each measure into a 1–5 score. For recency, a *small* number of days is good, so sort descending to give the most recent customers score 5:

```sql
NTILE(5) OVER (ORDER BY recency_days DESC) AS r_score,
NTILE(5) OVER (ORDER BY frequency)         AS f_score,
NTILE(5) OVER (ORDER BY monetary)          AS m_score
```

### Step 3: segments

Combine the scores with `CASE`:

| Segment | Rule |
|---|---|
| champions | r ≥ 4 and f ≥ 4 |
| at_risk | r ≤ 2 and f ≥ 4 (used to buy a lot, gone quiet) |
| new | r ≥ 4 and f ≤ 2 |
| others | everyone else |

=== exercise
id: rfm-1
title: RFM base for Tokyo
ordered: true
--- prompt
For customers in **Tokyo**, using only their **delivered** orders, return `customer_id`, **recency_days** (days from their last delivered order to 2025-12-31), **frequency** (number of delivered orders) and **monetary** (line revenue, rounded to 2 decimals). Sort by monetary (highest first), then customer_id.
--- solution
SELECT o.customer_id,
       DATEDIFF('2025-12-31', MAX(o.order_date)) AS recency_days,
       COUNT(DISTINCT o.order_id) AS frequency,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS monetary
FROM orders o
JOIN order_items oi ON oi.order_id = o.order_id
JOIN customers c ON c.customer_id = o.customer_id
WHERE o.status = 'delivered' AND c.city = 'Tokyo'
GROUP BY o.customer_id
ORDER BY monetary DESC, o.customer_id;
--- hint
Watch out: COUNT(*) after joining order_items counts lines. Use COUNT(DISTINCT o.order_id).
===

=== exercise
id: rfm-2
title: Segment sizes
ordered: true
--- prompt
For **all** customers with delivered orders, compute recency_days, frequency and monetary as above, then scores with `NTILE(5)`:

- **r**: `NTILE(5) OVER (ORDER BY recency_days DESC, customer_id)`
- **f**: `NTILE(5) OVER (ORDER BY frequency, customer_id)`

Then assign a **segment**: `champions` (r ≥ 4 and f ≥ 4), `at_risk` (r ≤ 2 and f ≥ 4), `new` (r ≥ 4 and f ≤ 2), otherwise `others`.

Return **segment** and **customers**, sorted by segment name.
--- solution
WITH base AS (
  SELECT o.customer_id,
         DATEDIFF('2025-12-31', MAX(o.order_date)) AS recency_days,
         COUNT(DISTINCT o.order_id) AS frequency
  FROM orders o
  WHERE o.status = 'delivered'
  GROUP BY o.customer_id
),
scored AS (
  SELECT customer_id,
         NTILE(5) OVER (ORDER BY recency_days DESC, customer_id) AS r,
         NTILE(5) OVER (ORDER BY frequency, customer_id) AS f
  FROM base
)
SELECT CASE
         WHEN r >= 4 AND f >= 4 THEN 'champions'
         WHEN r <= 2 AND f >= 4 THEN 'at_risk'
         WHEN r >= 4 AND f <= 2 THEN 'new'
         ELSE 'others'
       END AS segment,
       COUNT(*) AS customers
FROM scored
GROUP BY segment
ORDER BY segment;
--- hint
Monetary isn't needed for these two scores. Build the base CTE, score it with NTILE, then CASE into segments and count.
===

=== exercise
id: rfm-3
title: Most valuable customers
ordered: true
--- prompt
Return the **10 customers with the highest lifetime revenue** from **delivered** orders: `customer_id`, **orders** (delivered orders), **revenue** (rounded to 2 decimals), **avg_order_value** (revenue / orders, rounded to 2 decimals), **first_order** and **last_order** (dates of their first and last delivered order, as DATE). Ties by customer_id.
--- solution
SELECT o.customer_id,
       COUNT(DISTINCT o.order_id) AS orders,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS revenue,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) / COUNT(DISTINCT o.order_id), 2) AS avg_order_value,
       DATE(MIN(o.order_date)) AS first_order,
       DATE(MAX(o.order_date)) AS last_order
FROM orders o
JOIN order_items oi ON oi.order_id = o.order_id
WHERE o.status = 'delivered'
GROUP BY o.customer_id
ORDER BY revenue DESC, o.customer_id
LIMIT 10;
--- hint
Aggregate per customer over delivered orders joined to their lines; DATE() strips the time.
===
