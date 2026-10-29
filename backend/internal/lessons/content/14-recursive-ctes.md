title: Recursive CTEs
module: Subqueries & CTEs
level: Advanced
minutes: 16
summary: Walk hierarchies like org charts and category trees, and generate sequences.
---
A **recursive CTE** refers to itself. It has two parts joined by `UNION ALL`:

1. an **anchor** query that produces the starting rows, and
2. a **recursive** query that builds new rows from the rows produced in the previous step.

It stops when the recursive part returns no new rows.

### Generating a sequence

```sql
WITH RECURSIVE seq (n) AS (
  SELECT 1                              -- anchor
  UNION ALL
  SELECT n + 1 FROM seq WHERE n < 10    -- recursive step
)
SELECT n FROM seq;
```

### Walking a hierarchy

The org chart is a tree: start from the CEO (no manager) and repeatedly add each person's direct reports:

```sql
WITH RECURSIVE chain AS (
  SELECT employee_id, manager_id, 0 AS depth
  FROM employees WHERE manager_id IS NULL
  UNION ALL
  SELECT e.employee_id, e.manager_id, c.depth + 1
  FROM employees e
  JOIN chain c ON e.manager_id = c.employee_id
)
SELECT depth, COUNT(*) AS people FROM chain GROUP BY depth;
```

### Building paths

Carry a growing string through the recursion. `CAST(... AS CHAR(500))` matters: MySQL sizes the column from the anchor row, and longer values would be cut off.

```sql
WITH RECURSIVE tree AS (
  SELECT category_id, name, CAST(name AS CHAR(500)) AS path
  FROM categories WHERE parent_category_id IS NULL
  UNION ALL
  SELECT c.category_id, c.name, CONCAT(t.path, ' > ', c.name)
  FROM categories c JOIN tree t ON c.parent_category_id = t.category_id
)
SELECT * FROM tree ORDER BY path;
```

### Safety net

MySQL stops a recursion after `@@cte_max_recursion_depth` steps (1000 by default) with error 3636. If you hit it, check your stopping condition, or raise the limit for your session: `SET SESSION cte_max_recursion_depth = 5000;`.

=== exercise
id: rec-1
title: Count to ten
ordered: true
--- prompt
Use a recursive CTE to return the numbers **1 to 10** in a column named **n**.
--- solution
WITH RECURSIVE seq (n) AS (
  SELECT 1
  UNION ALL
  SELECT n + 1 FROM seq WHERE n < 10
)
SELECT n FROM seq;
--- hint
Anchor SELECT 1, recursive step SELECT n + 1 ... WHERE n < 10.
===

=== exercise
id: rec-2
title: Electronics category paths
ordered: true
--- prompt
Return every category **under Electronics** (category 1), excluding Electronics itself, with its `category_id` and its full **path** from the top, like `Electronics > Audio > Headphones`. Sort by path.
--- solution
WITH RECURSIVE tree AS (
  SELECT category_id, CAST(name AS CHAR(500)) AS path
  FROM categories WHERE category_id = 1
  UNION ALL
  SELECT c.category_id, CONCAT(t.path, ' > ', c.name)
  FROM categories c JOIN tree t ON c.parent_category_id = t.category_id
)
SELECT category_id, path FROM tree
WHERE category_id <> 1
ORDER BY path;
--- hint
Start the anchor at category_id = 1, and remember the CAST so paths aren't cut off.
===

=== exercise
id: rec-3
title: Size of each VP's organization
ordered: true
--- prompt
VPs are the employees whose job **level is 7**. For each VP, count **everyone below them** in the reporting chain (direct and indirect reports, including people who have left).

Return the VP's `employee_id` as **vp_id**, their department name as **department**, and **team_size**. Largest team first, ties by vp_id.
--- solution
WITH RECURSIVE org AS (
  SELECT e.employee_id AS vp_id, e.employee_id
  FROM employees e JOIN jobs j ON j.job_id = e.job_id
  WHERE j.level = 7
  UNION ALL
  SELECT o.vp_id, e.employee_id
  FROM employees e JOIN org o ON e.manager_id = o.employee_id
)
SELECT o.vp_id, d.name AS department, COUNT(*) - 1 AS team_size
FROM org o
JOIN employees v ON v.employee_id = o.vp_id
JOIN departments d ON d.department_id = v.department_id
GROUP BY o.vp_id, d.name
ORDER BY team_size DESC, o.vp_id;
--- hint
Carry the VP's id (vp_id) unchanged through the recursion, then count rows per vp_id. Subtract 1 for the VP themselves.
===

=== exercise
id: rec-4
title: Daily orders, first week of December
ordered: true
--- prompt
Generate every date from **2025-12-01 to 2025-12-07** with a recursive CTE, and return each **day** with the number of orders placed that day as **orders**. Sort by day.
--- solution
WITH RECURSIVE days (day) AS (
  SELECT DATE('2025-12-01')
  UNION ALL
  SELECT day + INTERVAL 1 DAY FROM days WHERE day < '2025-12-07'
)
SELECT d.day, COUNT(o.order_id) AS orders
FROM days d
LEFT JOIN orders o ON o.order_date >= d.day AND o.order_date < d.day + INTERVAL 1 DAY
GROUP BY d.day
ORDER BY d.day;
--- hint
Generate the dates, then LEFT JOIN orders on a one-day range so empty days would still show 0.
===
