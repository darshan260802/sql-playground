title: Joining tables
module: Joins
level: Beginner
minutes: 15
summary: Combine rows from related tables with INNER JOIN and table aliases.
---
Data is split across tables to avoid repetition: an order line stores a `product_id`, not the product's name. A **join** puts related rows back together.

```sql
SELECT oi.order_id, p.name, oi.quantity
FROM order_items AS oi
JOIN products    AS p ON p.product_id = oi.product_id
WHERE oi.order_id = 6;
```

- `JOIN` (short for `INNER JOIN`) pairs each row on the left with every row on the right where the `ON` condition is true.
- Rows **without a match are dropped** on either side.
- `oi` and `p` are **table aliases**. They keep queries short, and they're required when two tables share a column name.

### How tables relate here

Columns ending in `_id` usually point at another table's primary key (the Tables panel marks them with a link icon):

```
orders.customer_id      → customers.customer_id
order_items.order_id    → orders.order_id
order_items.product_id  → products.product_id
products.category_id    → categories.category_id
employees.department_id → departments.department_id
```

### Joining several tables

Chain joins, one `ON` per join:

```sql
SELECT c.name AS category,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS revenue
FROM order_items oi
JOIN orders     o ON o.order_id    = oi.order_id
JOIN products   p ON p.product_id  = oi.product_id
JOIN categories c ON c.category_id = p.category_id
WHERE o.status NOT IN ('cancelled', 'returned')
GROUP BY c.name
ORDER BY revenue DESC;
```

### `USING` shorthand

When the column has the same name on both sides, `JOIN products USING (product_id)` is equivalent to the ON version.

> **Watch the row count.** Joining one order to its three items gives three rows. If a total looks too large, a join is probably multiplying rows.

=== exercise
id: join-1
title: Products with their category
ordered: true
--- prompt
For products **1 to 10**, return `product_id`, the product name as **product** and the category name as **category**. Sort by `product_id`.
--- solution
SELECT p.product_id, p.name AS product, c.name AS category
FROM products p
JOIN categories c ON c.category_id = p.category_id
WHERE p.product_id BETWEEN 1 AND 10
ORDER BY p.product_id;
--- hint
Both tables have a `name` column, so qualify them: p.name and c.name.
===

=== exercise
id: join-2
title: Category revenue in 2024
ordered: true
--- prompt
Compute 2024 revenue per (leaf) category. Revenue = quantity × unit price × (1 − discount) over order lines whose order was placed in **2024** and whose status is **not** `cancelled` or `returned`.

Return the category name as **category** and **revenue** rounded to 2 decimals, highest revenue first.
--- solution
SELECT c.name AS category,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS revenue
FROM order_items oi
JOIN orders o ON o.order_id = oi.order_id
JOIN products p ON p.product_id = oi.product_id
JOIN categories c ON c.category_id = p.category_id
WHERE o.order_date >= '2024-01-01' AND o.order_date < '2025-01-01'
  AND o.status NOT IN ('cancelled', 'returned')
GROUP BY c.category_id, c.name
ORDER BY revenue DESC;
--- hint
Join order_items → orders (for the date and status) → products → categories, then GROUP BY the category.
===

=== exercise
id: join-3
title: The Tokyo office
--- prompt
For **current** employees working in the **Tokyo** office, return `employee_id`, `first_name`, `last_name`, their department name as **department** and their job title as **title**.
--- solution
SELECT e.employee_id, e.first_name, e.last_name, d.name AS department, j.title
FROM employees e
JOIN offices o ON o.office_id = e.office_id
JOIN departments d ON d.department_id = e.department_id
JOIN jobs j ON j.job_id = e.job_id
WHERE o.city = 'Tokyo' AND e.termination_date IS NULL;
--- hint
Join employees to offices, departments and jobs; filter on o.city.
===

=== exercise
id: join-4
title: 2025 orders by customer country
ordered: true
--- prompt
Return each customer `country` with the number of orders placed in **2025** by customers from that country, as **orders**. Most orders first.
--- solution
SELECT c.country, COUNT(*) AS orders
FROM orders o
JOIN customers c ON c.customer_id = o.customer_id
WHERE o.order_date >= '2025-01-01' AND o.order_date < '2026-01-01'
GROUP BY c.country
ORDER BY orders DESC;
--- hint
The country lives on customers, the date on orders.
===
