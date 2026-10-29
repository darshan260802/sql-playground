# Plan and design decisions

## Goal

A learning-focused SQL playground in the spirit of sql.clickhouse.com, but **writable**: learners can practise the full language (reads, writes, DDL, transactions, routines) without ever damaging the shared data. Audience: a small group (≤ 20 accounts) pivoting into data careers, hosted on one VPS.

## Decisions (from the kickoff Q&A)

| Question | Decision |
|---|---|
| SQL engine | **MySQL 8.4 LTS**: "support all SQL syntax and behave the same as MySQL", so it *is* MySQL |
| Data domains | E-commerce, HR / org chart, Finance / markets |
| Backend | Go |
| Frontend | React + Vite + TypeScript |
| Signup | Open email signup, capped at 20 accounts |
| Email | None. The admin resets passwords via CLI |
| Learning | Lessons with auto-checked exercises |
| Extra features | Saved queries + history, share links |
| VPS | 8 GB+ RAM; must coexist with existing nginx + ufw |
| Per-user storage | 300 MB quota |
| Theme | Dark (default) + light |
| Look & motion | ClickHouse design system + Apple design / fluid-motion principles |

## How much data fits in ~150 MB?

Measured, not estimated. InnoDB reserves free extents **per file**, so with one file per table, mid-sized tables (1–3 MB of data) occupy 9–10 MB each. The same 1.0M rows took:

| Layout | On disk |
|---|---|
| file-per-table (one .ibd per table) | 198 MB |
| one general tablespace per schema | 100 MB |

So every schema (core and each learner copy) lives in **its own general tablespace**. That fits **~1.66M rows in 148 MB**, e.g. 484k order lines, 207k orders, 190k bank transactions and 140k daily stock prices. That's plenty for meaningful aggregation and window-function work, and small enough that a copy takes ~10s.

Capacity for 20 learners: 148 MB × 20 ≈ 3 GB baseline, at most 300 MB × 20 = 6 GB with every quota full, plus ~1.1 GB of MySQL base files.

## Architecture

### Isolation by MySQL privileges

Rather than parsing SQL to decide what's allowed (fragile), every request runs under a MySQL account that physically cannot do more than intended:

- `pg_reader`: `SELECT, SHOW VIEW` on `core.*`. Used for anonymous visitors and for checking query exercises.
- `pgu_<id>`: full DML/DDL/routine/trigger rights on `u_<id>.*`, plus `SELECT` on `core.*`. No global privileges, so no `FILE`, `SUPER`, `PROCESS` or `CREATE USER`. Schema names are escaped in `GRANT` (`u\_1`) so `_` can't act as a wildcard.
- Passwords are random, kept in memory, and rotated at every start.
- A small guard additionally blocks `CREATE/DROP DATABASE` and `TABLESPACE` / `DATA DIRECTORY` clauses with a friendly message (privileges would block most of it anyway).
- MySQL is configured with `local_infile=OFF`, `secure_file_priv=NULL`, and no binary log.

### Running SQL

- A statement splitter that follows the mysql CLI's rules (quotes, comments, `DELIMITER`) runs statements one at a time, so each gets its own result, timing, warnings and error line.
- **Sticky sessions**: a signed-in learner keeps one connection per target for 15 idle minutes, so variables, temp tables and open transactions survive between runs. The UI shows when a transaction is open (read from `performance_schema`).
- **Watchdog**: driver calls never see request cancellation (that would drop the connection mid-query). A separate goroutine issues `KILL QUERY` on timeout, on Cancel, or when the quota monitor sees the learner's tablespace files grow past the limit.
- Results are capped at 2,000 rows for display, but the full row count is reported.

### Reset

`DROP DATABASE u_<id>` → `DROP TABLESPACE` → recreate both → apply the schema DDL (with foreign keys, which `CREATE TABLE … LIKE` would lose) → `INSERT … SELECT` table by table. Progress is reported by rows copied and shown as a progress bar.

### Exercise checking

| Kind | How it's checked |
|---|---|
| Query (100) | Run the learner's SQL and the solution on `core` as `pg_reader`; compare the last result set: column count, row count, values normalized (numbers to 2 decimals, NULL distinct from 'NULL'), and order only when the exercise says so. Feedback names what's off (count, order, a missing row). |
| DML (7) | In the learner's copy: `START TRANSACTION` → their statements → check query → `ROLLBACK`, then the same with the solution. Compare the check results. The learner's data never changes. |
| DDL (9) | Setup (cleanup) → solution → check → cleanup → learner's SQL → check. Checks inspect `information_schema` or exercise behaviour (e.g. insert invalid rows with `INSERT IGNORE` to test CHECK constraints, fire a trigger inside a rolled-back transaction). |

A Go test runs every exercise against real MySQL (verified on 8.0.46 and 8.4.13). It asserts that the solution passes, that a wrong answer fails, and that each check takes under 8 seconds.

## Frontend design

**ClickHouse system:** near-black canvas `#0a0a0a`, surface cards `#1a1a1a` with hairlines, and electric yellow `#faff69` used sparingly: the primary button, stat numbers, the active step, the full-bleed CTA band. Typography is Inter (display at weight 700 with negative tracking) and JetBrains Mono. The light theme keeps yellow for fills only and uses an olive `#5f6300` for yellow-as-text (~6:1 contrast).

**Apple motion principles applied:**
- Critically damped springs (`bounce: 0`) everywhere by default; a little bounce only after a momentum gesture (sheet release) or a success moment.
- Feedback on pointer-down (`:active` scale 0.97) on every pressable element.
- Spatial consistency: popovers grow from their trigger (`transform-origin`), sheets enter and leave from the bottom, and dialogs scale in and out along the same path.
- Direct manipulation: resizable panes track the pointer 1:1 from where they were grabbed, rubber-band past their limits and spring back. The mobile sheet uses momentum projection to decide dismissal.
- Translucent top bar (material + blur) with content scrolling underneath. Shared-layout transitions for tab, segment and stepper indicators.
- Accessibility: `prefers-reduced-motion` (springs and slides become instant; the "wrong answer" shake is dropped), `prefers-reduced-transparency` (solid bars), `prefers-contrast: more`. Visible focus rings, keyboard-resizable splitters, ARIA roles on grid, tabs and dialogs.

## Possible next steps

- Result charts (bar/line) and CSV export
- An example-query library alongside the lessons
- Optional SMTP for self-service password reset
- Multiplayer: share a link that opens a read-only copy of someone's result
