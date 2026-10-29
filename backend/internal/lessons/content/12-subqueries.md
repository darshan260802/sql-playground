title: Subqueries
module: Subqueries & CTEs
level: Intermediate
minutes: 15
summary: Nest queries inside queries: scalar values, IN lists, EXISTS checks and derived tables.
---
A **subquery** is a query in parentheses used inside another query. Where you put it decides what it must return.

### Scalar subquery: one value

```sql
-- products priced above the overall average
SELECT product_id, name, unit_price
FROM products
WHERE unit_price > (SELECT AVG(unit_price) FROM products);
```

If a subquery used with `=` or `>` returns more than one row, MySQL raises error 1242.

### IN: a list of values

```sql
-- customers who used the CYBER30 coupon
SELECT customer_id, first_name
FROM customers
WHERE customer_id IN (SELECT customer_id FROM orders WHERE coupon_code = 'CYBER30');
```

### Correlated subqueries and EXISTS

A **correlated** subquery refers to the outer row (`e.employee_id` below), so it's logically re-evaluated for every outer row. `EXISTS` is true when the subquery returns at least one row; what it selects doesn't matter, so people write `SELECT 1`:

```sql
SELECT e.employee_id, e.first_name
FROM employees e
WHERE EXISTS (
  SELECT 1 FROM leave_requests l
  WHERE l.employee_id = e.employee_id AND l.leave_type = 'parental'
);
```

`NOT EXISTS` is the safest way to say "has no matching rows": unlike `NOT IN` it isn't broken by NULLs.

### Derived tables: a subquery in FROM

A subquery in `FROM` acts as a temporary table, and it **must have an alias**:

```sql
SELECT ROUND(AVG(n), 2) AS avg_orders_per_customer
FROM (SELECT customer_id, COUNT(*) AS n
      FROM orders
      GROUP BY customer_id) AS per_customer;
```

This "aggregate, then aggregate again" pattern comes up constantly in analytics.

### Correlated scalar subquery in SELECT

```sql
SELECT p.product_id, p.name,
       (SELECT COUNT(*) FROM reviews r WHERE r.product_id = p.product_id) AS reviews
FROM products p
WHERE p.category_id = 33;
```

=== exercise
id: sub-1
title: Pricier than the average laptop
--- prompt
Return `product_id`, `name` and `unit_price` of laptops (category **39**) priced **above the average laptop price**.
--- solution
SELECT product_id, name, unit_price
FROM products
WHERE category_id = 39
  AND unit_price > (SELECT AVG(unit_price) FROM products WHERE category_id = 39);
--- hint
The scalar subquery needs its own WHERE category_id = 39.
===

=== exercise
id: sub-2
title: Cyber Monday shoppers in Germany
--- prompt
Return `customer_id`, `first_name` and `last_name` of customers from **Germany** who placed at least one order with coupon **CYBER30**. Each customer should appear only once.
--- solution
SELECT customer_id, first_name, last_name
FROM customers
WHERE country = 'Germany'
  AND customer_id IN (SELECT customer_id FROM orders WHERE coupon_code = 'CYBER30');
--- hint
Using IN with a subquery returns each customer once. A JOIN would repeat customers who used the coupon twice.
===

=== exercise
id: sub-3
title: Parents in the data team
--- prompt
Return `employee_id`, `first_name` and `last_name` of employees in department **4** (Data) who have at least one **parental** leave request.
--- solution
SELECT e.employee_id, e.first_name, e.last_name
FROM employees e
WHERE e.department_id = 4
  AND EXISTS (SELECT 1 FROM leave_requests l
              WHERE l.employee_id = e.employee_id AND l.leave_type = 'parental');
--- hint
A correlated EXISTS on leave_requests.
===

=== exercise
id: sub-4
title: Orders per customer
--- prompt
Among customers who placed **at least one** order, what is the **average number of orders** per customer? Return it rounded to 2 decimals as **avg_orders**.
--- solution
SELECT ROUND(AVG(n), 2) AS avg_orders
FROM (SELECT customer_id, COUNT(*) AS n FROM orders GROUP BY customer_id) AS t;
--- hint
Count orders per customer in a derived table, then AVG over it.
===

=== exercise
id: sub-5
title: Products that were never reviewed
--- prompt
Return `product_id` and `name` of **active** products in category **44** (Cookware) that have **no reviews**. Use NOT EXISTS.
--- solution
SELECT p.product_id, p.name
FROM products p
WHERE p.category_id = 44
  AND p.is_active = 1
  AND NOT EXISTS (SELECT 1 FROM reviews r WHERE r.product_id = p.product_id);
--- hint
NOT EXISTS (SELECT 1 FROM reviews r WHERE r.product_id = p.product_id)
===
