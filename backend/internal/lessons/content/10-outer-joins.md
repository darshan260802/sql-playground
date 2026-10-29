title: LEFT JOIN and finding what's missing
module: Joins
level: Intermediate
minutes: 14
summary: Keep unmatched rows with outer joins, and use them to find things that don't exist.
---
`INNER JOIN` drops rows without a match. `LEFT JOIN` keeps **every row from the left table**. When there's no match, the right table's columns are filled with `NULL`:

```sql
SELECT c.customer_id, c.first_name, o.order_id
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.customer_id
WHERE c.customer_id IN (1, 2, 3);
```

A customer with no orders still appears, with `order_id` = NULL.

### The anti-join: "rows with no match"

Combine `LEFT JOIN` with `IS NULL` on the right side's key to find rows that have **no** related row:

```sql
-- products that have never been ordered
SELECT p.product_id, p.name
FROM products p
LEFT JOIN order_items oi ON oi.product_id = p.product_id
WHERE oi.order_item_id IS NULL;
```

`NOT EXISTS (SELECT 1 FROM ... WHERE ...)` expresses the same idea and is often clearer. You'll see it in the subqueries lesson.

### Counting with zeros

`COUNT(*)` counts the NULL-filled row too, so use `COUNT(right_table.column)` to get **0** for rows without matches:

```sql
SELECT p.product_id, COUNT(r.review_id) AS reviews
FROM products p
LEFT JOIN reviews r ON r.product_id = p.product_id
GROUP BY p.product_id;
```

### ON vs WHERE in outer joins

A condition on the right table in `WHERE` throws away the NULL rows, silently turning your LEFT JOIN into an INNER JOIN. Put it in `ON` instead:

```sql
-- keeps every customer; counts only their 2025 orders
SELECT c.customer_id, COUNT(o.order_id) AS orders_2025
FROM customers c
LEFT JOIN orders o
  ON o.customer_id = c.customer_id
 AND o.order_date >= '2025-01-01'
GROUP BY c.customer_id;
```

`RIGHT JOIN` is the mirror image (keep every row of the right table). MySQL has no `FULL OUTER JOIN`; emulate it with a `UNION` of a left and a right join.

=== exercise
id: left-1
title: Singapore customers who never ordered
--- prompt
Return `customer_id`, `first_name` and `last_name` of customers from **Singapore** who have **never placed an order**.
--- solution
SELECT c.customer_id, c.first_name, c.last_name
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.customer_id
WHERE c.country = 'Singapore' AND o.order_id IS NULL;
--- hint
LEFT JOIN orders, then keep rows where o.order_id IS NULL.
===

=== exercise
id: left-2
title: Never-ordered products
--- prompt
Return `product_id`, `name` and `launched_on` of products that appear in **no order line** at all.
--- solution
SELECT p.product_id, p.name, p.launched_on
FROM products p
LEFT JOIN order_items oi ON oi.product_id = p.product_id
WHERE oi.order_item_id IS NULL;
--- hint
An anti-join between products and order_items.
===

=== exercise
id: left-3
title: Delivered without a shipment?
--- prompt
Some orders have status `delivered` but no row in `shipments`. Count them **per channel**: return `channel` and the count as **orders**.

(Can you guess why from the result?)
--- solution
SELECT o.channel, COUNT(*) AS orders
FROM orders o
LEFT JOIN shipments s ON s.order_id = o.order_id
WHERE o.status = 'delivered' AND s.shipment_id IS NULL
GROUP BY o.channel;
--- hint
LEFT JOIN shipments and keep rows where s.shipment_id IS NULL.
===

=== exercise
id: left-4
title: Board games and their reviews
--- prompt
For every product in category **33** (Board Games), return `product_id`, `name` and its number of reviews as **reviews**, showing **0** for products with no reviews.
--- solution
SELECT p.product_id, p.name, COUNT(r.review_id) AS reviews
FROM products p
LEFT JOIN reviews r ON r.product_id = p.product_id
WHERE p.category_id = 33
GROUP BY p.product_id, p.name;
--- hint
COUNT(r.review_id), not COUNT(*): the NULL row of an unmatched product must count as 0.
===
