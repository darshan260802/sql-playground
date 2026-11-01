title: GROUP BY and HAVING
module: Aggregation
level: Beginner
minutes: 14
summary: Aggregate per group, and filter the groups themselves with HAVING.
---
`GROUP BY` splits the rows into groups that share the same value(s), then runs the aggregates once **per group**:

```sql
SELECT country, COUNT(*) AS customers
FROM customers
GROUP BY country
ORDER BY customers DESC;
```

### The golden rule

Every column in the `SELECT` list must either be **in the GROUP BY** or be **inside an aggregate**. MySQL enforces this by default (`ONLY_FULL_GROUP_BY`):

```sql
-- error: city is neither grouped nor aggregated
SELECT country, city, COUNT(*) FROM customers GROUP BY country;
```

Which city should MySQL show for "India"? There are eight! Either group by it too, or aggregate it (`MAX(city)`, `COUNT(DISTINCT city)`).

### Grouping by expressions

You can group by computed values, like the year of a date:

```sql
SELECT YEAR(order_date) AS year, COUNT(*) AS orders
FROM orders
GROUP BY YEAR(order_date)
ORDER BY year;
```

MySQL also lets you `GROUP BY year` (the alias) or `GROUP BY 1` (the first select column).

### HAVING filters groups

`WHERE` runs **before** grouping and can't see aggregates. `HAVING` runs **after** grouping and filters whole groups:

```sql
SELECT brand, COUNT(*) AS products
FROM products
WHERE is_active = 1          -- row filter
GROUP BY brand
HAVING COUNT(*) >= 80;       -- group filter
```

### Execution order

MySQL logically processes a query in this order, which explains most "why can't I use that here?" errors:

`FROM` → `WHERE` → `GROUP BY` → `HAVING` → `SELECT` → `ORDER BY` → `LIMIT`

=== exercise
id: group-1
title: Customers per country
ordered: true
--- prompt
Return each `country` with its number of customers as **customers**, largest first (break ties by country name).
--- solution
SELECT country, COUNT(*) AS customers
FROM customers
GROUP BY country
ORDER BY customers DESC, country;
--- hint
GROUP BY country, COUNT(*), ORDER BY customers DESC.
===

=== exercise
id: group-2
title: Orders by year and status
--- prompt
Return the **year** of `order_date`, the `status`, and the number of orders as **orders** for every year/status combination.
--- solution
SELECT YEAR(order_date) AS year, status, COUNT(*) AS orders
FROM orders
GROUP BY YEAR(order_date), status;
--- hint
Group by two things: YEAR(order_date) and status.
===

=== exercise
id: group-3
title: Big brands
ordered: true
--- prompt
Return each `brand` that has **at least 80 products** together with its product count as **products**. Sort by products (descending), then brand.
--- solution
SELECT brand, COUNT(*) AS products
FROM products
GROUP BY brand
HAVING COUNT(*) >= 80
ORDER BY products DESC, brand;
--- hint
A condition on COUNT(*) belongs in HAVING, not WHERE.
===

=== exercise
id: group-4
title: Most loyal customers
ordered: true
--- prompt
Return the **5 customers with the most orders**: `customer_id` and their order count as **orders**. Break ties by `customer_id`.
--- solution
SELECT customer_id, COUNT(*) AS orders
FROM orders
GROUP BY customer_id
ORDER BY orders DESC, customer_id
LIMIT 5;
--- hint
Group orders by customer_id, sort by the count, LIMIT 5.
===

=== exercise
id: group-5
title: Large departments
ordered: true
--- prompt
Among **current** employees (no `termination_date`), return each `department_id` with more than **300** people and its headcount as **headcount**, largest first.
--- solution
SELECT department_id, COUNT(*) AS headcount
FROM employees
WHERE termination_date IS NULL
GROUP BY department_id
HAVING COUNT(*) > 300
ORDER BY headcount DESC;
--- hint
The termination filter goes in WHERE; the headcount filter goes in HAVING.
===
