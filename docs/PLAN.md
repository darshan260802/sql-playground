# Plan and design decisions

## Goal

A learning-focused SQL playground in the spirit of sql.clickhouse.com, but **writable**: learners can practise the full language (reads, writes, DDL, transactions, routines) without ever damaging the shared data. Audience: a small group (≤ 20 accounts) pivoting into data careers, hosted on one VPS.

## Decisions (from the kickoff Q&A)

| Question | Decision |
|---|---|
| SQL engine | **MySQL 8** (8.0.31+, tested on 8.0 and 8.4): "support all SQL syntax and behave the same as MySQL", so it *is* MySQL |
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

### Deployment

No containers. The first version shipped with Docker Compose; it was replaced because it added moving parts (a second firewall layer that bypasses ufw, image builds, volumes) without adding anything a single-purpose VPS needs.

- **MySQL** comes from Ubuntu's own packages (8.0 on 24.04), with the settings in `deploy/mysql/playground.cnf` dropped into `/etc/mysql/mysql.conf.d/`. The app logs in as a dedicated `sqlpg_admin@127.0.0.1` account; root keeps Ubuntu's socket-only login.
- **The app** is one static Go binary plus the built website, run by systemd as an unprivileged `sqlpg` user with a read-only view of the system (`ProtectSystem=strict` and friends). It listens on `127.0.0.1` only, so ufw and nginx stay in charge of what's public.
- **`deploy/install.sh`** does the whole install and every later update: packages, MySQL config and account, a private Go and Node toolchain used only for building (so it never clashes with other projects), the build itself (as the `sqlpg` user), the service, the `sqlpg-admin` command and a nightly backup. It keeps settings and data, and checks that the app answers before it finishes.

## Frontend design

The interface was revamped with the `design-taste-frontend` skill (installed in `.claude/skills/` together with `minimalist-ui`, `high-end-visual-design` and `brandkit`). The playground is an app screen people work in, so it follows the skill's app-screen chapter rather than its landing-page rules.

**Style:** "data terminal" (dealt by the skill's `draw.mjs --app`, keeping the brand's palette and type): chrome on one ground (`#111`), the working surfaces (editor, results, code) one step darker (`#0a0a0a`), lines instead of shadows, tight radii (8/6/4px), and colour under 3% of the screen. Every selection (tabs, the database switch, the nav rail, exercises) is the same 1px accent line under its label; yellow otherwise appears only on the one primary action per screen (Run, Check, Continue). Figures that stand alone or sit in columns are set in JetBrains Mono so digits stack; SQL never uses ligatures, so `<=` reads as typed.

**Each screen answers one question first:**
- *Playground:* "What does my query return, and on which database?" The title row names the screen, shows the database switch (your copy, with its storage, or the read-only original) and the Run button. Before the first run, the results area offers example queries and the keyboard shortcuts instead of an empty panel.
- *Learn:* "Where do I continue, and how far along am I?" A Continue button and the progress at its true ratio come first; lessons are rows with a progress ring, level, minutes and solved count.
- *Lesson:* "Can I solve this exercise?" The article and the exercise panel sit side by side; below 1024px they take turns behind a Lesson / Exercises switch, so the exercise is one tap away instead of a long scroll.

**Phones** get their own composition: a bottom bar (Playground, Learn, theme, account) replaces the icon rail; the playground shows one region at a time (Query, Results, Tables) with Run in a thumb-height bar and switches to Results after a run; dialogs become draggable bottom sheets.

**Removed in the revamp** (generated-UI tells the skill flags): tracked uppercase labels, a glass top bar, zebra rows, shimmer skeletons, a 56px landing-page headline and stat block on Learn, numbered cards, the full-bleed yellow band, filled pill tabs.

**Checks:** the skill's linter (`lint.mjs --app`: 0 FAIL) and inspector (`inspect.mjs --app` at 1440, 768 and 390px: 0 FAIL on every screen), plus a Playwright run of the main flows on desktop and phone.

## Possible next steps

- Result charts (bar/line) and CSV export
- An example-query library alongside the lessons
- Optional SMTP for self-service password reset
- Multiplayer: share a link that opens a read-only copy of someone's result
