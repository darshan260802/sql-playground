title: Aggregate functions
module: Aggregation
level: Beginner
minutes: 10
summary: Summarise many rows into one with COUNT, SUM, AVG, MIN and MAX.
---
Aggregate functions collapse many rows into a single value:

```sql
SELECT COUNT(*)            AS orders,
       MIN(order_date)     AS first_order,
       MAX(order_date)     AS last_order
FROM orders;
```

| Function | Returns |
|---|---|
| `COUNT(*)` | number of rows |
| `COUNT(col)` | number of rows where `col` is **not NULL** |
| `COUNT(DISTINCT col)` | number of different non-NULL values |
| `SUM(col)`, `AVG(col)` | total and mean (NULLs are ignored) |
| `MIN(col)`, `MAX(col)` | smallest and largest value (works on text and dates too) |

### COUNT(*) vs COUNT(column)

The difference is a classic interview question:

```sql
SELECT COUNT(*)     AS customers,
       COUNT(phone) AS with_phone
FROM customers;
```

### Aggregating an expression

You can aggregate any expression. For example, total revenue across all order lines:

```sql
SELECT ROUND(SUM(quantity * unit_price * (1 - discount)), 2) AS revenue
FROM order_items;
```

### WHERE filters before aggregating

`WHERE` removes rows *before* they are aggregated, so this counts only 2024 orders:

```sql
SELECT COUNT(*) FROM orders
WHERE order_date >= '2024-01-01' AND order_date < '2025-01-01';
```

> Without `GROUP BY` (next lesson), an aggregate query returns exactly **one row**, even if no rows match. In that case `COUNT` gives 0 and `SUM` gives NULL.

=== exercise
id: agg-1
title: Phone coverage
--- prompt
Return two numbers in one row: the total number of customers as **customers**, and how many of them have a phone number as **with_phone**.
--- solution
SELECT COUNT(*) AS customers, COUNT(phone) AS with_phone
FROM customers;
--- hint
COUNT(phone) skips NULL phones.
===

=== exercise
id: agg-2
title: Laptop price range
--- prompt
For laptops (category **39**), return the cheapest price as **min_price**, the most expensive as **max_price**, and the average rounded to 2 decimals as **avg_price**.
--- solution
SELECT MIN(unit_price) AS min_price,
       MAX(unit_price) AS max_price,
       ROUND(AVG(unit_price), 2) AS avg_price
FROM products
WHERE category_id = 39;
--- hint
Filter with WHERE, then use MIN, MAX and ROUND(AVG(...), 2).
===

=== exercise
id: agg-3
title: Active customers in 2024
--- prompt
How many **different customers** placed at least one order in **2024**? Return a single number.
--- solution
SELECT COUNT(DISTINCT customer_id) AS customers
FROM orders
WHERE order_date >= '2024-01-01' AND order_date < '2025-01-01';
--- hint
COUNT(DISTINCT customer_id) with a date range.
===

=== exercise
id: agg-4
title: Units sold of product 1
--- prompt
Return the total number of **units** sold of product **1** as **units**, and the total revenue it generated (quantity × price × (1 − discount), rounded to 2 decimals) as **revenue**.
--- solution
SELECT SUM(quantity) AS units,
       ROUND(SUM(quantity * unit_price * (1 - discount)), 2) AS revenue
FROM order_items
WHERE product_id = 1;
--- hint
Both values come from order_items filtered on product_id = 1.
===
