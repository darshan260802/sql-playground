title: Self joins
module: Joins
level: Intermediate
minutes: 12
summary: Join a table to itself to compare rows, such as employees with their managers.
---
Some tables point at themselves: `employees.manager_id` holds another employee's `employee_id`, and `customers.referred_by_customer_id` holds another customer's id. To use them, join the table **to itself** with two different aliases:

```sql
SELECT e.employee_id,
       CONCAT(e.first_name, ' ', e.last_name) AS employee,
       CONCAT(m.first_name, ' ', m.last_name) AS manager
FROM employees e
LEFT JOIN employees m ON m.employee_id = e.manager_id
LIMIT 10;
```

Think of `e` and `m` as two copies of the same table: one playing the employee, one playing the manager. The `LEFT JOIN` keeps the CEO, who has no manager.

### Comparing rows to each other

Self joins also compare rows within one table, for example pairs of products in the same category where one costs twice the other. Add a condition like `a.product_id < b.product_id` so each pair appears once instead of twice.

### Joining the same table twice for different roles

A query can join one table several times, as long as each copy has its own alias. Here `salaries` is joined once for the employee and once for the manager, keeping only current salaries (`to_date IS NULL`):

```sql
SELECT e.employee_id, se.salary, sm.salary AS manager_salary
FROM employees e
JOIN salaries se ON se.employee_id = e.employee_id AND se.to_date IS NULL
JOIN salaries sm ON sm.employee_id = e.manager_id  AND sm.to_date IS NULL
LIMIT 10;
```

=== exercise
id: self-1
title: The legal team and their managers
--- prompt
For employees in department **10** (Legal), return `employee_id`, **employee** (first and last name separated by a space) and **manager** (the manager's full name in the same format).
--- solution
SELECT e.employee_id,
       CONCAT(e.first_name, ' ', e.last_name) AS employee,
       CONCAT(m.first_name, ' ', m.last_name) AS manager
FROM employees e
LEFT JOIN employees m ON m.employee_id = e.manager_id
WHERE e.department_id = 10;
--- hint
Join employees e to employees m ON m.employee_id = e.manager_id.
===

=== exercise
id: self-2
title: Top referrers
ordered: true
--- prompt
Return customers who **referred at least 3 other customers**: their `customer_id`, `first_name`, `last_name` and the number of customers they referred as **referrals**. Sort by referrals (descending), then customer_id.
--- solution
SELECT r.customer_id, r.first_name, r.last_name, COUNT(*) AS referrals
FROM customers c
JOIN customers r ON r.customer_id = c.referred_by_customer_id
GROUP BY r.customer_id, r.first_name, r.last_name
HAVING COUNT(*) >= 3
ORDER BY referrals DESC, r.customer_id;
--- hint
Join customers c (the referred) to customers r (the referrer) on r.customer_id = c.referred_by_customer_id, then group by the referrer.
===

=== exercise
id: self-3
title: Out-earning the boss
ordered: true
--- prompt
Find **current** employees in department **4** (Data) whose **current** salary is **higher than their manager's** current salary. A current salary has `to_date IS NULL`.

Return `employee_id`, **salary**, `manager_id` and **manager_salary**, sorted by employee_id.
--- solution
SELECT e.employee_id, se.salary, e.manager_id, sm.salary AS manager_salary
FROM employees e
JOIN salaries se ON se.employee_id = e.employee_id AND se.to_date IS NULL
JOIN salaries sm ON sm.employee_id = e.manager_id AND sm.to_date IS NULL
WHERE e.department_id = 4
  AND e.termination_date IS NULL
  AND se.salary > sm.salary
ORDER BY e.employee_id;
--- hint
Join salaries twice: once on e.employee_id and once on e.manager_id, both with to_date IS NULL.
===
