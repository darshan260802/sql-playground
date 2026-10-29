title: Gaps and islands
module: Analytics case studies
level: Advanced
minutes: 20
summary: Find streaks, missing values, overlapping ranges and sessions in ordered data.
---
"Gaps and islands" problems ask about **runs** in ordered data: the longest winning streak, consecutive login days, periods without activity, overlapping bookings. They look hard, but a few patterns solve almost all of them.

### Islands: the row-number difference trick

Number the rows twice: once over everything, and once only within the rows that have the property. Inside an unbroken run, both numbers grow by 1 per row, so their **difference stays constant**, and that difference identifies the island:

```
day  up  rn_all  rn_up  rn_all - rn_up
 1   1     1       1          0   ┐ island A
 2   1     2       2          0   ┘
 3   0     3       -          -
 4   1     4       3          1   ┐ island B
 5   1     5       4          1   │
 6   1     6       5          1   ┘
```

Group by that difference and count to get the length of each island.

### Gaps: compare with a complete list

To find what's *missing*, generate or take the complete list (the `calendar` table is perfect for this) and anti-join it with the data.

### Overlapping ranges

Two ranges `[a_start, a_end]` and `[b_start, b_end]` overlap exactly when

```sql
a_start <= b_end AND b_start <= a_end
```

Remember this one: it covers every overlap case, including one range containing the other.

### Sessions

To split events into sessions, start a new session whenever the gap to the previous event is too large. Then a running `SUM` of those "new session" flags numbers the sessions:

```sql
SUM(is_new_session) OVER (ORDER BY event_time) AS session_id
```

=== exercise
id: gap-1
title: Longest winning streak in 2025
ordered: true
--- prompt
A stock has an **up day** when its close is higher than its previous trading day's close. Using only 2025 prices (the first 2025 day of each stock counts as not up), find each stock's **longest run of consecutive up days**.

Return `ticker` and **longest_streak**, longest first, ties by ticker.
--- solution
WITH moves AS (
  SELECT stock_id, trade_date,
         COALESCE(close > LAG(close) OVER (PARTITION BY stock_id ORDER BY trade_date), 0) AS up
  FROM stock_prices
  WHERE trade_date >= '2025-01-01'
),
numbered AS (
  SELECT stock_id, up,
         ROW_NUMBER() OVER (PARTITION BY stock_id ORDER BY trade_date)
       - ROW_NUMBER() OVER (PARTITION BY stock_id, up ORDER BY trade_date) AS grp
  FROM moves
),
streaks AS (
  SELECT stock_id, grp, COUNT(*) AS len
  FROM numbered
  WHERE up = 1
  GROUP BY stock_id, grp
)
SELECT s.ticker, MAX(st.len) AS longest_streak
FROM streaks st
JOIN stocks s ON s.stock_id = st.stock_id
GROUP BY s.ticker
ORDER BY longest_streak DESC, s.ticker;
--- hint
Flag up days with LAG, apply the row-number difference trick partitioned by stock, then take the MAX island length per stock.
===

=== exercise
id: gap-2
title: Weekdays without trading
ordered: true
--- prompt
Return every **weekday** of **2025** (from the `calendar` table) on which **no stock has a price**, i.e. the market holidays: `date_key` and `day_name`. Sort by date.
--- solution
SELECT c.date_key, c.day_name
FROM calendar c
LEFT JOIN (SELECT DISTINCT trade_date FROM stock_prices WHERE trade_date >= '2025-01-01') p
  ON p.trade_date = c.date_key
WHERE c.date_key >= '2025-01-01' AND c.date_key < '2026-01-01'
  AND c.is_weekend = 0
  AND p.trade_date IS NULL
ORDER BY c.date_key;
--- hint
Anti-join the calendar's weekdays against the DISTINCT trade dates. (stock_prices has no index on trade_date alone, so a correlated NOT EXISTS would be slow.)
===

=== exercise
id: gap-3
title: Overlapping leave
ordered: true
--- prompt
Find pairs of **approved** leave requests of the **same employee** whose date ranges **overlap** (the end dates are inclusive). Return `employee_id`, **leave_a** and **leave_b** (the two `leave_id`s, smaller first). Sort by employee_id, leave_a, leave_b.
--- solution
SELECT a.employee_id, a.leave_id AS leave_a, b.leave_id AS leave_b
FROM leave_requests a
JOIN leave_requests b
  ON b.employee_id = a.employee_id
 AND a.leave_id < b.leave_id
 AND a.start_date <= b.end_date
 AND b.start_date <= a.end_date
WHERE a.status = 'approved' AND b.status = 'approved'
ORDER BY a.employee_id, leave_a, leave_b;
--- hint
Self-join on the same employee with a.leave_id < b.leave_id and the overlap test a.start <= b.end AND b.start <= a.end.
===

=== exercise
id: gap-4
title: Shopping sprees
ordered: true
--- prompt
For customer **22210**, group orders into **sprees**: a new spree starts when an order comes **more than 14 days** after the previous one (the first order starts spree 1).

Return **spree** (1, 2, 3, ...), **first_order** and **last_order** (dates and times of the first and last order in it) and **orders**. Sort by spree.
--- solution
WITH flagged AS (
  SELECT order_id, order_date,
         CASE WHEN LAG(order_date) OVER (ORDER BY order_date) IS NULL
                OR order_date > LAG(order_date) OVER (ORDER BY order_date) + INTERVAL 14 DAY
              THEN 1 ELSE 0 END AS new_spree
  FROM orders
  WHERE customer_id = 22210
),
numbered AS (
  SELECT order_id, order_date, SUM(new_spree) OVER (ORDER BY order_date, order_id) AS spree
  FROM flagged
)
SELECT spree, MIN(order_date) AS first_order, MAX(order_date) AS last_order, COUNT(*) AS orders
FROM numbered
GROUP BY spree
ORDER BY spree;
--- hint
Flag rows where the gap to LAG(order_date) exceeds 14 days, then a running SUM of the flags numbers the sprees.
===
