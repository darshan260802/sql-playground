title: Window functions: ranking
module: Window functions
level: Intermediate
minutes: 16
summary: Number and rank rows within groups without collapsing them, and solve top-N per group.
---
Aggregates with `GROUP BY` collapse each group into one row. **Window functions** compute across a group of rows but **keep every row**. The `OVER (...)` clause defines the window:

```sql
SELECT product_id, category_id, unit_price,
       ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY unit_price DESC) AS price_rank
FROM products;
```

- `PARTITION BY` splits the rows into groups (like GROUP BY, but nothing collapses).
- `ORDER BY` inside `OVER` sets the order used for numbering within each partition.

### The three ranking functions

For the values 100, 90, 90, 80:

| Function | Result | Ties |
|---|---|---|
| `ROW_NUMBER()` | 1, 2, 3, 4 | always unique, ties broken arbitrarily |
| `RANK()` | 1, 2, 2, 4 | ties share a rank, then a gap |
| `DENSE_RANK()` | 1, 2, 2, 3 | ties share a rank, no gap |

### Top-N per group

You can't use a window function in `WHERE`, because windows are computed after WHERE runs. Compute the rank in a CTE (or derived table), then filter:

```sql
WITH ranked AS (
  SELECT category_id, product_id, unit_price,
         ROW_NUMBER() OVER (PARTITION BY category_id
                            ORDER BY unit_price DESC, product_id) AS rn
  FROM products
)
SELECT * FROM ranked WHERE rn <= 3;
```

This is one of the most frequently asked SQL interview patterns.

### NTILE: buckets

`NTILE(4) OVER (ORDER BY x)` splits rows into 4 groups of (nearly) equal size, numbered 1–4. That's handy for quartiles and deciles.

### Named windows

When several functions share a window, name it once with `WINDOW w AS (...)` after `HAVING`, then write `OVER w`.

=== exercise
id: rank-1
title: Top 3 per fashion category
ordered: true
--- prompt
Consider the Fashion categories that hold products: `category_id IN (20, 21, 22, 46, 47)` (Shoes, Bags, Watches, Men's Clothing, Women's Clothing). Return the **3 most expensive** products of each category.

Return `category_id`, `product_id`, `unit_price` and **rn** (1–3), ranking by `unit_price` descending with `product_id` as the tie-breaker. Sort by category_id, then rn.
--- solution
WITH ranked AS (
  SELECT category_id, product_id, unit_price,
         ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY unit_price DESC, product_id) AS rn
  FROM products
  WHERE category_id IN (20, 21, 22, 46, 47)
)
SELECT category_id, product_id, unit_price, rn
FROM ranked
WHERE rn <= 3
ORDER BY category_id, rn;
--- hint
ROW_NUMBER() OVER (PARTITION BY category_id ORDER BY unit_price DESC, product_id) in a CTE, then WHERE rn <= 3.
===

=== exercise
id: rank-2
title: RANK vs DENSE_RANK
ordered: true
--- prompt
For **current** employees of department **10** (Legal), using their **current** salary (`to_date IS NULL`), return `employee_id`, `salary`, **rnk** = `RANK()` and **dense_rnk** = `DENSE_RANK()`, both ordered by salary (highest first). Sort by salary descending, then employee_id.
--- solution
SELECT e.employee_id, s.salary,
       RANK() OVER (ORDER BY s.salary DESC) AS rnk,
       DENSE_RANK() OVER (ORDER BY s.salary DESC) AS dense_rnk
FROM employees e
JOIN salaries s ON s.employee_id = e.employee_id AND s.to_date IS NULL
WHERE e.department_id = 10 AND e.termination_date IS NULL
ORDER BY s.salary DESC, e.employee_id;
--- hint
No PARTITION BY is needed: the whole result is one window.
===

=== exercise
id: rank-3
title: Top 3 customers per country (2025)
ordered: true
--- prompt
Rank customers by their **2025 revenue** (line revenue on 2025 orders with status `delivered`) **within their country**. Return the top 3 per country: `country`, `customer_id`, **revenue** (rounded to 2 decimals) and **rnk** (use `RANK()` ordered by revenue descending).

Sort by country, rnk, then customer_id.
--- solution
WITH rev AS (
  SELECT c.country, c.customer_id,
         SUM(oi.quantity * oi.unit_price * (1 - oi.discount)) AS revenue
  FROM orders o
  JOIN order_items oi ON oi.order_id = o.order_id
  JOIN customers c ON c.customer_id = o.customer_id
  WHERE o.status = 'delivered'
    AND o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
  GROUP BY c.country, c.customer_id
),
ranked AS (
  SELECT country, customer_id, revenue,
         RANK() OVER (PARTITION BY country ORDER BY revenue DESC) AS rnk
  FROM rev
)
SELECT country, customer_id, ROUND(revenue, 2) AS revenue, rnk
FROM ranked
WHERE rnk <= 3
ORDER BY country, rnk, customer_id;
--- hint
First aggregate revenue per customer, then rank with PARTITION BY country, then filter.
===

=== exercise
id: rank-4
title: Order-count quartiles
ordered: true
--- prompt
Take every customer who ordered in **2025** and their number of 2025 orders. Split them into **4 quartiles** with `NTILE(4)` ordered by order count (ascending, ties by customer_id).

Return **quartile**, the number of **customers** in it, and the **min_orders** and **max_orders** within it. Sort by quartile.
--- solution
WITH counts AS (
  SELECT customer_id, COUNT(*) AS n
  FROM orders
  WHERE order_date >= '2025-01-01' AND order_date < '2026-01-01'
  GROUP BY customer_id
),
tiles AS (
  SELECT customer_id, n, NTILE(4) OVER (ORDER BY n, customer_id) AS quartile
  FROM counts
)
SELECT quartile, COUNT(*) AS customers, MIN(n) AS min_orders, MAX(n) AS max_orders
FROM tiles
GROUP BY quartile
ORDER BY quartile;
--- hint
Count per customer, assign NTILE(4), then group by the quartile.
===
