title: Views, functions, procedures and triggers
module: Designing databases
level: Advanced
minutes: 20
summary: Store queries and logic inside the database with views, stored functions, procedures and triggers.
---
### Views: saved queries

A view is a named `SELECT` that you can query like a table. It stores no data: the query runs every time you use the view.

```sql
CREATE OR REPLACE VIEW v_current_salaries AS
SELECT e.employee_id, e.department_id, s.salary
FROM employees e
JOIN salaries s ON s.employee_id = e.employee_id AND s.to_date IS NULL
WHERE e.termination_date IS NULL;

SELECT department_id, ROUND(AVG(salary)) FROM v_current_salaries GROUP BY department_id;
```

### Stored functions

A function returns a single value and can be used inside any query:

```sql
CREATE FUNCTION margin_pct(price DECIMAL(10,2), cost DECIMAL(10,2))
RETURNS DECIMAL(6,2) DETERMINISTIC
RETURN ROUND((price - cost) / price * 100, 2);

SELECT name, margin_pct(unit_price, unit_cost) FROM products LIMIT 5;
```

### Stored procedures and DELIMITER

Procedures can run several statements, take `IN`/`OUT` parameters and return result sets. They're called with `CALL`. A body with several statements uses `BEGIN ... END`, and the semicolons inside it would end the `CREATE` too early. So switch the client's statement delimiter temporarily (this editor supports `DELIMITER` just like the mysql command line):

```sql
DELIMITER //
CREATE PROCEDURE orders_in_month(IN p_year INT, IN p_month INT)
BEGIN
  SELECT COUNT(*) AS orders
  FROM orders
  WHERE YEAR(order_date) = p_year AND MONTH(order_date) = p_month;
END //
DELIMITER ;

CALL orders_in_month(2025, 11);
```

### Triggers

A trigger runs automatically `BEFORE` or `AFTER` an `INSERT`, `UPDATE` or `DELETE` on a table, once for each row. Inside it, `OLD.col` and `NEW.col` hold the row's values before and after the change.

```sql
CREATE TRIGGER trg_reviews_clamp BEFORE INSERT ON reviews
FOR EACH ROW
SET NEW.rating = LEAST(GREATEST(NEW.rating, 1), 5);
```

Typical uses are audit logs, derived columns and validation. Use them sparingly: logic hidden in triggers surprises people.

Everything you create shows up in the schema panel under your database. Remove objects with `DROP VIEW`, `DROP FUNCTION`, `DROP PROCEDURE` or `DROP TRIGGER`.

=== exercise
id: view-1
title: An order totals view
kind: ddl
--- prompt
Create a view **v_order_totals** with one row per order and these columns:

- `order_id`, `customer_id`, `order_date`
- **items**: the total `quantity` of the order's lines
- **total**: the order's line revenue (quantity × unit price × (1 − discount)), rounded to 2 decimals

Include every order that has lines, whatever its status.
--- setup
DROP VIEW IF EXISTS v_order_totals;
--- solution
CREATE VIEW v_order_totals AS
SELECT o.order_id, o.customer_id, o.order_date,
       SUM(oi.quantity) AS items,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS total
FROM orders o
JOIN order_items oi ON oi.order_id = o.order_id
GROUP BY o.order_id, o.customer_id, o.order_date;
--- check
SELECT COUNT(*) AS orders, SUM(items) AS items, ROUND(SUM(total), 2) AS total, MIN(order_id) AS first_order
FROM v_order_totals
WHERE order_date >= '2025-12-01';
--- hint
CREATE VIEW v_order_totals AS SELECT ... FROM orders JOIN order_items ... GROUP BY the order columns.
===

=== exercise
id: func-1
title: A net price function
kind: ddl
--- prompt
Create a function **net_price(price DECIMAL(10,2), discount DECIMAL(4,2))** that returns `DECIMAL(10,2)`: the price after the discount, i.e. `price × (1 − discount)`, rounded to 2 decimals.
--- setup
DROP FUNCTION IF EXISTS net_price;
--- solution
CREATE FUNCTION net_price(price DECIMAL(10,2), discount DECIMAL(4,2))
RETURNS DECIMAL(10,2) DETERMINISTIC
RETURN ROUND(price * (1 - discount), 2);
--- check
SELECT net_price(100, 0.15) AS a, net_price(19.99, 0) AS b, net_price(59.99, 0.25) AS c, net_price(0.99, 0.30) AS d;
--- hint
A one-line function needs no BEGIN/END: CREATE FUNCTION ... RETURNS DECIMAL(10,2) DETERMINISTIC RETURN ...;
===

=== exercise
id: proc-1
title: A customer summary procedure
kind: ddl
--- prompt
Create a procedure **customer_summary(IN p_customer_id INT)** that returns one row with:

- `customer_id`
- **orders**: the customer's number of orders (any status)
- **delivered_revenue**: line revenue of their `delivered` orders, rounded to 2 decimals (0 if none)

Use `DELIMITER` to wrap the body in BEGIN ... END. We'll check it with `CALL customer_summary(84)`.
--- setup
DROP PROCEDURE IF EXISTS customer_summary;
--- solution
DELIMITER //
CREATE PROCEDURE customer_summary(IN p_customer_id INT)
BEGIN
  SELECT p_customer_id AS customer_id,
         (SELECT COUNT(*) FROM orders WHERE customer_id = p_customer_id) AS orders,
         (SELECT COALESCE(ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2), 0)
            FROM orders o JOIN order_items oi ON oi.order_id = o.order_id
           WHERE o.customer_id = p_customer_id AND o.status = 'delivered') AS delivered_revenue;
END //
DELIMITER ;
--- check
CALL customer_summary(84);
--- hint
Two scalar subqueries inside one SELECT keep it simple. Remember COALESCE(..., 0) for customers without delivered orders.
===

=== exercise
id: trig-1
title: Audit price changes
kind: ddl
--- prompt
A table **price_audit** (`product_id`, `old_price`, `new_price`, `changed_at`) has been created for you.

Create a trigger named **trg_price_audit** that runs **after an update** on `products` and, **only when `unit_price` actually changed**, inserts a row with the product id, the old and new price, and the current time.

We'll check it by updating some prices (and one name) inside a rolled-back transaction.
--- setup
DROP TRIGGER IF EXISTS trg_price_audit;
DROP TABLE IF EXISTS price_audit;
CREATE TABLE price_audit (product_id INT UNSIGNED NOT NULL, old_price DECIMAL(10,2) NOT NULL, new_price DECIMAL(10,2) NOT NULL, changed_at DATETIME NOT NULL);
--- solution
CREATE TRIGGER trg_price_audit AFTER UPDATE ON products
FOR EACH ROW
INSERT INTO price_audit (product_id, old_price, new_price, changed_at)
SELECT NEW.product_id, OLD.unit_price, NEW.unit_price, NOW()
FROM DUAL
WHERE NEW.unit_price <> OLD.unit_price;
--- check
START TRANSACTION;
UPDATE products SET unit_price = unit_price + 1 WHERE product_id IN (1, 2);
UPDATE products SET name = CONCAT(name, '') WHERE product_id = 3;
SELECT product_id, new_price - old_price AS diff FROM price_audit ORDER BY product_id;
ROLLBACK;
--- hint
AFTER UPDATE ON products FOR EACH ROW, with an IF NEW.unit_price <> OLD.unit_price (inside BEGIN ... END), or an INSERT ... SELECT ... WHERE that compares NEW and OLD.
===
