title: Transactions
module: Changing data
level: Intermediate
minutes: 14
summary: Group statements so they succeed or fail together, and undo work with ROLLBACK.
---
A **transaction** is a group of statements that either all take effect or none do. The classic example is a money transfer: debiting one account without crediting the other would be a disaster.

```sql
START TRANSACTION;
INSERT INTO transactions (txn_id, account_id, txn_at, amount, txn_type)
VALUES (950001, 20, '2026-01-02 10:00:00', -100.00, 'transfer_out');
INSERT INTO transactions (txn_id, account_id, txn_at, amount, txn_type)
VALUES (950002, 21, '2026-01-02 10:00:00',  100.00, 'transfer_in');
COMMIT;
```

- `START TRANSACTION` (or `BEGIN`) opens a transaction.
- `COMMIT` makes every change permanent and visible to others.
- `ROLLBACK` throws all of them away.

### Autocommit

By default MySQL runs in **autocommit** mode: every statement outside an explicit transaction is committed immediately. `START TRANSACTION` suspends autocommit until you COMMIT or ROLLBACK.

### Try it in the playground

Your playground session stays open between runs (for 15 minutes of inactivity), so you can work step by step:

1. Run `START TRANSACTION; DELETE FROM reviews WHERE rating = 1;`. The editor shows a **Transaction open** badge.
2. Run `SELECT COUNT(*) FROM reviews WHERE rating = 1;`: you see 0.
3. Run `ROLLBACK;` and count again: the reviews are back.

### Savepoints

`SAVEPOINT s1;` marks a point you can return to with `ROLLBACK TO SAVEPOINT s1;` without abandoning the whole transaction.

### Implicit commits

DDL statements (`CREATE`, `ALTER`, `DROP`, `TRUNCATE`) **commit automatically**, including any work in progress. You can't roll back a dropped table.

### ACID in one line each

- **Atomicity:** all or nothing.
- **Consistency:** constraints hold before and after.
- **Isolation:** concurrent transactions don't see each other's half-done work. InnoDB's default level is REPEATABLE READ.
- **Durability:** once committed, changes survive a crash.

> The exercises below are checked inside a transaction that we roll back for you, so write only the data-changing statements (no START TRANSACTION or COMMIT).

=== exercise
id: txn-1
title: Transfer between accounts
kind: dml
--- prompt
Customer 607 moves **250.00** from their checking account **20** to their savings account **21**. Record it as two rows in `transactions`, both at `2026-01-02 10:00:00`:

- txn **900001**: account 20, amount **-250.00**, type `transfer_out`
- txn **900002**: account 21, amount **250.00**, type `transfer_in`

Leave `merchant` and `merchant_category` empty.
--- solution
INSERT INTO transactions (txn_id, account_id, txn_at, amount, txn_type, merchant, merchant_category)
VALUES (900001, 20, '2026-01-02 10:00:00', -250.00, 'transfer_out', NULL, NULL),
       (900002, 21, '2026-01-02 10:00:00', 250.00, 'transfer_in', NULL, NULL);
--- check
SELECT account_id, ROUND(SUM(amount), 2) AS balance, COUNT(*) AS txns,
       SUM(txn_type = 'transfer_in' AND txn_at = '2026-01-02 10:00:00') AS new_in,
       SUM(txn_type = 'transfer_out' AND txn_at = '2026-01-02 10:00:00') AS new_out
FROM transactions WHERE account_id IN (20, 21)
GROUP BY account_id ORDER BY account_id;
--- hint
One INSERT with two rows in VALUES, or two INSERTs. A negative amount means money leaves the account.
===

=== exercise
id: txn-2
title: Cancel and refund an order
kind: dml
--- prompt
Order **205194** has been cancelled. Do both steps:

1. set the order's `status` to `cancelled`, and
2. record the refund as a new `payments` row: `payment_id` **900001**, same order, method **card**, amount **82.07**, `paid_at` **2026-01-02 12:00:00**, status `refunded`.
--- solution
UPDATE orders SET status = 'cancelled' WHERE order_id = 205194;
INSERT INTO payments (payment_id, order_id, payment_method, amount, paid_at, status)
VALUES (900001, 205194, 'card', 82.07, '2026-01-02 12:00:00', 'refunded');
--- check
SELECT o.status,
       (SELECT COUNT(*) FROM payments p WHERE p.order_id = o.order_id AND p.status = 'refunded') AS refunds,
       (SELECT SUM(amount) FROM payments p WHERE p.order_id = o.order_id AND p.status = 'refunded') AS refunded
FROM orders o WHERE o.order_id = 205194;
--- hint
Two statements: an UPDATE on orders and an INSERT into payments.
===
