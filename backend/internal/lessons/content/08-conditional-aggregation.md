title: Conditional aggregation and pivots
module: Aggregation
level: Intermediate
minutes: 12
summary: Count and sum only some rows inside an aggregate, and turn rows into columns.
---
Often you want several differently-filtered counts **in one row**. Instead of running several queries, put the condition *inside* the aggregate:

```sql
SELECT COUNT(*)                                        AS orders,
       SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled,
       SUM(CASE WHEN status = 'returned'  THEN 1 ELSE 0 END) AS returned
FROM orders;
```

### MySQL shortcuts

In MySQL a comparison evaluates to `1` (true) or `0` (false), so `SUM(condition)` counts matching rows:

```sql
SELECT SUM(status = 'cancelled') AS cancelled FROM orders;
```

`COUNT(CASE WHEN ... THEN 1 END)` also works: a CASE without ELSE gives NULL, and COUNT skips NULLs. Use `AVG(condition)` to get a rate directly (multiply by 100 for a percentage).

### Pivoting rows into columns

Grouping by one thing and spreading another across columns is called a **pivot**:

```sql
SELECT YEAR(order_date) AS year,
       SUM(channel = 'web')    AS web,
       SUM(channel = 'mobile') AS mobile,
       SUM(channel = 'store')  AS store
FROM orders
GROUP BY YEAR(order_date)
ORDER BY year;
```

MySQL has no `PIVOT` keyword, so this pattern is how you build pivot tables.

### Summing amounts conditionally

The same idea works for money: `SUM(CASE WHEN status = 'refunded' THEN amount ELSE 0 END)`.

=== exercise
id: cond-1
title: Channel scorecard
--- prompt
For each `channel`, return the total number of orders as **orders**, and the number of **delivered**, **cancelled** and **returned** orders in columns with those names.
--- solution
SELECT channel,
       COUNT(*) AS orders,
       SUM(status = 'delivered') AS delivered,
       SUM(status = 'cancelled') AS cancelled,
       SUM(status = 'returned') AS returned
FROM orders
GROUP BY channel;
--- hint
SUM(status = 'delivered') counts the delivered rows.
===

=== exercise
id: cond-2
title: Channel pivot by year
ordered: true
--- prompt
Return one row per **year** of `order_date` with the number of orders per channel in columns **web**, **mobile** and **store**. Sort by year.
--- solution
SELECT YEAR(order_date) AS year,
       SUM(channel = 'web') AS web,
       SUM(channel = 'mobile') AS mobile,
       SUM(channel = 'store') AS store
FROM orders
GROUP BY YEAR(order_date)
ORDER BY year;
--- hint
Group by year and use one SUM(channel = '...') per column.
===

=== exercise
id: cond-3
title: Return rate per year
ordered: true
--- prompt
For each **year**, return the number of orders as **orders**, the number with status `returned` as **returned**, and **return_rate_pct**: returned as a percentage of orders, rounded to 2 decimals. Sort by year.
--- solution
SELECT YEAR(order_date) AS year,
       COUNT(*) AS orders,
       SUM(status = 'returned') AS returned,
       ROUND(100 * SUM(status = 'returned') / COUNT(*), 2) AS return_rate_pct
FROM orders
GROUP BY YEAR(order_date)
ORDER BY year;
--- hint
100 * SUM(status = 'returned') / COUNT(*), or 100 * AVG(status = 'returned').
===

=== exercise
id: cond-4
title: Payment outcomes by method
ordered: true
--- prompt
For each `payment_method`, return **paid_amount** (total amount of payments with status `paid`) and **refunded_amount** (total of status `refunded`), both rounded to 2 decimals. Sort by paid_amount, highest first.
--- solution
SELECT payment_method,
       ROUND(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 2) AS paid_amount,
       ROUND(SUM(CASE WHEN status = 'refunded' THEN amount ELSE 0 END), 2) AS refunded_amount
FROM payments
GROUP BY payment_method
ORDER BY paid_amount DESC;
--- hint
SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END)
===
