title: Databases: create, switch, back up
module: Designing databases
level: Intermediate
minutes: 10
summary: What a database is on a MySQL server, how to create, switch and drop one, and how to back it up.
---
A MySQL **server** holds many **databases** (MySQL also calls them *schemas*). Each database is a folder of tables, views and routines. On a real server you create one per application: `shop`, `hr`, `analytics`…

### Create, list, switch and drop

```sql
CREATE DATABASE shop;            -- a new, empty database
CREATE DATABASE IF NOT EXISTS shop;
SHOW DATABASES;                  -- the ones you're allowed to see
USE shop;                        -- the default for the statements that follow
DROP DATABASE shop;              -- deletes it and EVERYTHING in it
```

`DROP DATABASE` can't be undone: every table and row in it is gone. On real servers only administrators get the privilege to create and drop databases.

> **In this playground** you work in exactly one database: the original data is `core`, and your private copy is `u_` plus your account number. That's why `CREATE DATABASE` and `DROP DATABASE` are switched off here. The **Your copy / Original** switch in the playground does what `USE` does, and **Reset database** in the account menu is your safe "drop and recreate".

### Which database am I in?

`DATABASE()` returns the current default database, and you can always name a table with its database in front, as `database.table`:

```sql
SELECT DATABASE();
SELECT COUNT(*) FROM core.orders;   -- works from your copy too
```

### Looking inside: information_schema

Every MySQL server has a read-only database called `information_schema` that describes all the others. It's how tools (including this playground's Tables panel) discover tables and columns:

```sql
-- the tables in the current database
SELECT table_name, table_rows
FROM information_schema.tables
WHERE table_schema = DATABASE();

-- the columns of one table, with their types
SELECT column_name, column_type, is_nullable
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = 'orders';
```

`SHOW TABLES;` and `DESCRIBE orders;` are shorter MySQL-only versions of the same questions.

### Backing up a database

SQL Server has a `BACKUP DATABASE` statement. MySQL doesn't; a backup is made outside SQL, with the **mysqldump** tool that ships with MySQL. It writes a file of `CREATE TABLE` and `INSERT` statements that rebuild the database:

```bash
mysqldump --single-transaction shop > shop-2026-01-31.sql   # back up
mysql shop < shop-2026-01-31.sql                            # restore
```

`--single-transaction` takes a consistent snapshot without locking the tables. Run backups on a schedule and copy the files to another machine: a backup on the same disk doesn't survive that disk failing. (This playground backs up its own accounts every night the same way.)

Inside SQL you can still copy **one table** quickly:

```sql
CREATE TABLE departments_copy LIKE departments;        -- same columns, keys and indexes
INSERT INTO departments_copy SELECT * FROM departments;  -- then the rows
```

`CREATE TABLE ... AS SELECT` (in the *INSERT, UPDATE and DELETE* lesson) is shorter, but it copies only columns and rows, not the primary key or indexes.

### Where databases run

A database server runs on a computer you can connect to over the network: your own laptop while learning, a server you manage (this playground runs MySQL on a small VPS), or a **managed database service** from a cloud provider, which handles backups, updates and failover for you. Applications connect with a host name, a port (3306 for MySQL), a user name and a password, and they should use an account that has only the privileges they need.

=== exercise
id: db-1
title: Which database am I in?
--- prompt
Return the name of the current default database in a column named **current_db**.
--- solution
SELECT DATABASE() AS current_db;
--- hint
MySQL has a function with no arguments for this: `DATABASE()`.
===

=== exercise
id: db-2
title: Every table in this database
ordered: true
--- prompt
Using `information_schema.tables`, list the name of every table in the **current database**, in alphabetical order.
--- solution
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
ORDER BY table_name;
--- hint
Filter with `table_schema = DATABASE()` and sort by `table_name`.
===

=== exercise
id: db-3
title: A faithful copy
kind: ddl
--- prompt
Make a backup of the `departments` table called **departments_backup** that has **all its rows and the same primary key**.

`CREATE TABLE ... AS SELECT` won't do here: it doesn't copy the primary key.
--- setup
DROP TABLE IF EXISTS departments_backup;
--- solution
CREATE TABLE departments_backup LIKE departments;
INSERT INTO departments_backup SELECT * FROM departments;
--- check
SELECT (SELECT COUNT(*) FROM departments_backup) AS rows_copied,
       (SELECT ROUND(SUM(budget), 2) FROM departments_backup) AS total_budget,
       (SELECT COUNT(*) FROM information_schema.statistics
         WHERE table_schema = DATABASE() AND table_name = 'departments_backup'
           AND index_name = 'PRIMARY') AS primary_key_columns;
--- hint
Two statements: `CREATE TABLE ... LIKE departments;` copies the structure, then `INSERT INTO ... SELECT * FROM departments;` copies the rows.
===
