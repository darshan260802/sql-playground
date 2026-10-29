title: CREATE TABLE and constraints
module: Designing databases
level: Intermediate
minutes: 18
summary: Design tables with the right types, keys and constraints, and change them with ALTER TABLE.
---
`CREATE TABLE` defines the columns, their types, and the **rules** the data must follow.

```sql
CREATE TABLE wishlist_demo (
  wishlist_id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  customer_id INT UNSIGNED NOT NULL,
  product_id  INT UNSIGNED NOT NULL,
  added_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_wishlist_demo (customer_id, product_id),
  FOREIGN KEY (customer_id) REFERENCES customers (customer_id),
  FOREIGN KEY (product_id)  REFERENCES products (product_id)
);
```

### Choosing types

| Use | Type |
|---|---|
| ids, counts | `INT` / `BIGINT` (`UNSIGNED` when never negative) |
| money | `DECIMAL(10,2)`, never `FLOAT` (binary floats can't represent 0.10 exactly) |
| short text | `VARCHAR(n)` |
| long text | `TEXT` |
| dates / timestamps | `DATE`, `DATETIME` |
| yes/no | `TINYINT(1)` (MySQL's `BOOLEAN` is an alias for it) |
| fixed set of values | `ENUM('a','b')` |

### Constraints

- `PRIMARY KEY`: uniquely identifies a row; implies NOT NULL. One per table.
- `UNIQUE`: no two rows share the value(s). Multi-column: the *combination* must be unique.
- `NOT NULL`: a value is required.
- `DEFAULT`: the value used when an INSERT omits the column.
- `FOREIGN KEY`: the value must exist in the referenced table. The column types must match exactly, so `INT UNSIGNED` can only reference `INT UNSIGNED`.
- `CHECK (condition)`: an arbitrary rule, enforced since MySQL 8.0.16.

### Changing tables

```sql
ALTER TABLE wishlist_demo ADD COLUMN note VARCHAR(200) NULL;
ALTER TABLE wishlist_demo RENAME COLUMN note TO comment;
ALTER TABLE wishlist_demo MODIFY comment VARCHAR(500) NULL;    -- change type
ALTER TABLE wishlist_demo DROP COLUMN comment;
DROP TABLE IF EXISTS wishlist_demo;
```

`SHOW CREATE TABLE products;` prints the exact DDL of any table, and `DESCRIBE products;` lists its columns.

> DDL commits implicitly and can't be rolled back. If you break something in your copy, **Reset database** brings back the original.

=== exercise
id: ddl-1
title: A wishlist table
kind: ddl
--- prompt
Create a table named **wishlist** with these columns, in this order:

- `wishlist_id`: INT UNSIGNED, primary key, auto-increment
- `customer_id`: INT UNSIGNED, required, foreign key to `customers`
- `product_id`: INT UNSIGNED, required, foreign key to `products`
- `added_at`: DATETIME, required, defaulting to the current time

A customer may add the same product only **once** (unique on the customer/product pair).
--- setup
DROP TABLE IF EXISTS wishlist;
--- solution
CREATE TABLE wishlist (
  wishlist_id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  customer_id INT UNSIGNED NOT NULL,
  product_id INT UNSIGNED NOT NULL,
  added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_wishlist_customer_product (customer_id, product_id),
  FOREIGN KEY (customer_id) REFERENCES customers (customer_id),
  FOREIGN KEY (product_id) REFERENCES products (product_id)
);
--- check
SELECT COLUMN_NAME AS item, COLUMN_TYPE AS detail, IS_NULLABLE AS nullable, EXTRA AS extra
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wishlist'
UNION ALL
SELECT CONCAT('constraint: ', CONSTRAINT_TYPE), COUNT(*), '', ''
FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'wishlist'
GROUP BY CONSTRAINT_TYPE
ORDER BY item;
--- hint
Look at the example at the top of the lesson. Foreign key columns must be INT UNSIGNED to match the referenced keys.
===

=== exercise
id: ddl-2
title: Reshape a table
kind: ddl
--- prompt
A table **practice_products** (columns `product_id`, `name`, `unit_price`) has been created for you. Use `ALTER TABLE` to:

1. make `product_id` the **primary key**,
2. rename `unit_price` to **price**, and
3. add a column **on_sale** of type `TINYINT(1)`, NOT NULL, default **0**, as the last column.
--- setup
DROP TABLE IF EXISTS practice_products;
CREATE TABLE practice_products AS SELECT product_id, name, unit_price FROM products WHERE category_id = 42;
--- solution
ALTER TABLE practice_products ADD PRIMARY KEY (product_id);
ALTER TABLE practice_products RENAME COLUMN unit_price TO price;
ALTER TABLE practice_products ADD COLUMN on_sale TINYINT(1) NOT NULL DEFAULT 0;
--- check
SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'practice_products'
ORDER BY ORDINAL_POSITION;
--- hint
Three ALTER TABLE statements (or one, with the changes separated by commas): ADD PRIMARY KEY (...), RENAME COLUMN ... TO ..., ADD COLUMN ...
===

=== exercise
id: ddl-3
title: Rules with CHECK
kind: ddl
--- prompt
Create a table **promotions** with:

- `code`: VARCHAR(20), primary key
- `pct`: DECIMAL(4,2), required, and it must be **greater than 0 and at most 0.50**
- `starts_on` and `ends_on`: DATE, both required, and `ends_on` must **not be before** `starts_on`

We'll check it by trying to insert valid and invalid promotions.
--- setup
DROP TABLE IF EXISTS promotions;
--- solution
CREATE TABLE promotions (
  code VARCHAR(20) NOT NULL PRIMARY KEY,
  pct DECIMAL(4,2) NOT NULL CHECK (pct > 0 AND pct <= 0.50),
  starts_on DATE NOT NULL,
  ends_on DATE NOT NULL,
  CHECK (ends_on >= starts_on)
);
--- check
START TRANSACTION;
INSERT IGNORE INTO promotions (code, pct, starts_on, ends_on) VALUES
  ('TOO_BIG', 0.60, '2026-01-01', '2026-01-31'),
  ('ZERO', 0.00, '2026-01-01', '2026-01-31'),
  ('BACKWARDS', 0.20, '2026-03-01', '2026-02-01'),
  ('OK_SPRING', 0.15, '2026-03-01', '2026-05-31'),
  ('OK_MAX', 0.50, '2026-06-01', '2026-06-01');
SELECT code, pct FROM promotions ORDER BY code;
ROLLBACK;
--- hint
Column-level: pct ... CHECK (pct > 0 AND pct <= 0.50). Table-level: CHECK (ends_on >= starts_on).
===
