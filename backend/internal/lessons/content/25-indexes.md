title: Indexes and EXPLAIN
module: Designing databases
level: Advanced
minutes: 16
summary: Make queries fast with the right indexes, and read EXPLAIN to see what MySQL actually does.
---
Without an index, MySQL finds rows by reading **the whole table** (a *full table scan*). An index is a sorted copy of some columns plus a pointer to each row, like the index at the back of a book, so lookups can jump straight to the matching rows.

### See the plan with EXPLAIN

Put `EXPLAIN` in front of a query to see the plan without running it:

```sql
EXPLAIN SELECT * FROM orders WHERE customer_id = 84;
```

Columns to read first:

| Column | What it tells you |
|---|---|
| `type` | `const`/`eq_ref`/`ref`/`range` are good; `ALL` is a full table scan |
| `key` | the index MySQL chose (NULL = none) |
| `rows` | estimated rows it will examine |
| `Extra` | `Using index` = answered from the index alone; `Using filesort` / `Using temporary` = extra work |

`EXPLAIN ANALYZE` actually runs the query and reports real timings per step.

### Try it

`stock_prices` has a primary key on `(stock_id, trade_date)` but no index on `trade_date` alone. Compare these:

```sql
EXPLAIN SELECT * FROM stock_prices WHERE stock_id = 5 AND trade_date >= '2025-01-01';  -- uses PRIMARY
EXPLAIN SELECT * FROM stock_prices WHERE trade_date = '2025-12-31';                   -- type ALL
```

The second query can't use the primary key, because `trade_date` is its **second** column. This is the **leftmost-prefix rule**: an index on `(a, b, c)` helps with filters on `a`, `a, b` or `a, b, c`, but not on `b` alone.

In your own copy you can fix it, then run EXPLAIN again:

```sql
CREATE INDEX idx_stock_prices_date ON stock_prices (trade_date);
```

### Designing composite indexes

- Put **equality** columns first and the **range** column last: for `WHERE status = 'shipped' AND order_date >= ...` use `(status, order_date)`.
- An index that contains every column the query needs is a **covering index** (`Using index` in Extra).
- Wrapping a column in a function (`WHERE YEAR(order_date) = 2025`) usually prevents index use. Write a range instead.

### The cost

Every index takes disk space and slows down INSERT/UPDATE/DELETE, because the index must be updated too. Index what you actually filter, join and sort on.

=== exercise
id: idx-1
title: A composite index
kind: ddl
--- prompt
A table **practice_orders** (a copy of December 2025's orders, without indexes) has been created for you. The team often runs:

```sql
SELECT * FROM practice_orders WHERE status = 'shipped' AND order_date >= '2025-12-20';
```

Create an index named **idx_status_date** that serves this query well: equality column first, range column second.
--- setup
DROP TABLE IF EXISTS practice_orders;
CREATE TABLE practice_orders AS SELECT * FROM orders WHERE order_date >= '2025-12-01';
--- solution
CREATE INDEX idx_status_date ON practice_orders (status, order_date);
--- check
SELECT INDEX_NAME, SEQ_IN_INDEX, COLUMN_NAME, NON_UNIQUE
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'practice_orders'
ORDER BY INDEX_NAME, SEQ_IN_INDEX;
--- hint
CREATE INDEX idx_status_date ON practice_orders (status, order_date);
===

=== exercise
id: idx-2
title: Deduplicate, then enforce
kind: ddl
--- prompt
**practice_contacts** has `contact_id` (primary key) and `email`. A few emails appear twice (sometimes with different capitalization, which this collation treats as equal).

1. Delete the duplicates, keeping the row with the **lowest** `contact_id` for each email.
2. Add a **unique index** named **uq_contacts_email** on `email` so it can't happen again.
--- setup
DROP TABLE IF EXISTS practice_contacts;
CREATE TABLE practice_contacts (contact_id INT NOT NULL PRIMARY KEY, email VARCHAR(120) NOT NULL);
INSERT INTO practice_contacts SELECT customer_id, email FROM customers WHERE customer_id <= 200;
INSERT INTO practice_contacts SELECT customer_id + 1000, UPPER(email) FROM customers WHERE customer_id IN (5, 17, 42, 99);
--- solution
DELETE c2 FROM practice_contacts c1
JOIN practice_contacts c2 ON c1.email = c2.email AND c1.contact_id < c2.contact_id;
ALTER TABLE practice_contacts ADD UNIQUE INDEX uq_contacts_email (email);
--- check
SELECT 'rows' AS item, COUNT(*) AS value FROM practice_contacts
UNION ALL
SELECT 'highest id', MAX(contact_id) FROM practice_contacts
UNION ALL
SELECT CONCAT('index ', INDEX_NAME, ' unique'), 1 - NON_UNIQUE
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'practice_contacts' AND INDEX_NAME <> 'PRIMARY'
ORDER BY item;
--- hint
A self-join DELETE: DELETE c2 FROM t c1 JOIN t c2 ON c1.email = c2.email AND c1.contact_id < c2.contact_id. Then ALTER TABLE ... ADD UNIQUE INDEX.
===
