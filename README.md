# SQL Playground

A self-hosted MySQL playground for learning SQL on realistic data, from your first `SELECT` to window functions, transactions and schema design.

- **Anyone** can open the playground and run read-only queries against ~1.7 million rows of e-commerce, HR and finance data.
- **Signed-in learners** get a **private copy of the whole database** where `INSERT`, `UPDATE`, `DELETE`, `CREATE`, triggers and stored procedures all work. Changes only affect their copy, and **Reset database** restores the original in about 10 seconds.
- **30 lessons with 116 auto-checked exercises** take you from basics to analytics case studies (cohorts, gaps-and-islands, RFM, drawdowns).

It's built for a small group (up to 20 accounts) on a single VPS. Behaviour matches real MySQL, because it *is* real MySQL 8.

![Playground](docs/screenshots/playground-dark.png)

| Lessons with instant feedback | Light theme |
|---|---|
| ![Lesson](docs/screenshots/lesson.png) | ![Light](docs/screenshots/playground-light.png) |

## Features

**Playground**
- SQL editor (CodeMirror 6) with MySQL syntax, autocompletion from the live schema, multiple tabs, and run-selection with Ctrl/⌘+Enter
- Run several statements at once, including `DELIMITER //` blocks for procedures and triggers, as in the mysql command line
- Virtualized results grid, per-statement result tabs, warnings, and friendly hints for common errors (`ONLY_FULL_GROUP_BY`, foreign keys, error 1093, …)
- Schema browser with row counts, column types and key/foreign-key markers, plus one-click table preview
- **Sessions persist between runs**: `START TRANSACTION` in one run and `ROLLBACK` in the next, with a "Transaction open" indicator and one-click Commit/Rollback
- Cancel running queries; time limits (30s signed in, 15s anonymous); a 2,000-row display cap with true row counts
- Saved queries, searchable history and share links (`/q/<id>`)

**Learning**
- 30 lessons in 9 modules: Basics → Aggregation → Joins → Subqueries & CTEs → Window functions → Dates, text & JSON → Changing data → Designing databases → Analytics case studies
- Exercises are checked against the real database:
  - **query** exercises compare your result with the solution's (order-aware when it matters, with numeric tolerance)
  - **DML** exercises run inside a transaction that is rolled back
  - **DDL** exercises inspect the objects you created
- Expected-output preview, hints, solutions, and progress tracking (in the browser when signed out, carried over when you sign up)

**Design**
- The ClickHouse design system (near-black canvas, electric-yellow accent, Inter + JetBrains Mono), with dark as the default and a light theme
- Apple-style motion: critically damped springs, press feedback on pointer-down, popovers that grow from their trigger, a draggable bottom sheet on phones, rubber-banded resizable panes. Respects reduced motion, reduced transparency and increased contrast.

## The dataset

Generated deterministically (same data on every install), stored in one InnoDB tablespace of **148 MB**:

| Domain | Tables (rows) |
|---|---|
| E-commerce | `customers` (40k), `orders` (207k), `order_items` (484k), `payments` (236k), `shipments` (175k), `reviews` (59k), `products` (2.5k), `categories` (3-level tree), `suppliers` |
| HR | `employees` (5k, 5-level manager hierarchy), `salaries` (22k, full history), `performance_reviews` (19k), `leave_requests` (26k), `departments`, `jobs`, `offices` |
| Finance | `stock_prices` (140k daily OHLCV, 2018–2025), `transactions` (190k), `trades` (16k), `exchange_rates` (19k), `accounts`, `loans`, `stocks` |
| Utility | `calendar` (2012–2030) |

The data has realistic texture: growth over time, Black Friday and holiday peaks, churn, cohorts, returns that skew to fashion, salary raises and promotions, a COVID-style market crash, IPOs and delistings, paired transfers between accounts, and some deliberate messiness for cleaning exercises (inconsistent phone formats, stray spaces in city names, mixed-case emails, NULLs).

## Architecture

```
browser ──▶ nginx (TLS) ──▶ sql-playground service, 127.0.0.1:8080 (systemd)
                              │  Go API + built React frontend
                              ▼
                            MySQL 8 from Ubuntu's packages (localhost only)
                              ├─ core            pristine dataset, read-only to everyone
                              ├─ u_<id>          each learner's private copy (own tablespace)
                              └─ playground_app  accounts, sessions, saved queries, history, progress
```

- **Isolation is enforced by MySQL itself**, not by parsing SQL. Anonymous visitors use a `SELECT`-only account on `core`. Each learner gets their own MySQL account with full rights on their own schema and read-only access to `core`. They can't see other schemas, read files, change global settings or create users.
- **Copies** are made with `INSERT … SELECT` into a per-user general tablespace (about 10s). One file per user also makes storage easy to measure and to reclaim on reset.
- **Guards:** a watchdog runs `KILL QUERY` at the time limit or on Cancel; a quota monitor stops writes that push a database past its storage limit (300 MB by default); there are rate limits, a global concurrency limit, and `MAX_USER_CONNECTIONS` per account.
- MySQL passwords for these accounts are random, held in memory only, and rotated on every start.

More detail and the reasoning behind each choice: [docs/PLAN.md](docs/PLAN.md).

## Repository layout

```
backend/    Go API, dataset generator, lessons (Markdown) and exercise checker
frontend/   React + Vite + TypeScript app
deploy/     install.sh, systemd unit, settings template, MySQL config, nginx sites, scripts
docs/       plan, VPS setup guide (PDF) and screenshots
```

## Deploying on your VPS (nginx + ufw, no Docker)

Runs directly on Ubuntu (24.04 LTS recommended; 22.04 is tested too, and 26.04 with its MySQL 8.4 should work) next to whatever nginx and ufw already serve: nothing new is exposed to the internet.

> **Domain on Cloudflare?** Follow the step-by-step **[VPS setup guide (PDF)](docs/vps-setup-guide.pdf)**. It goes from a fresh Ubuntu server to HTTPS behind Cloudflare, using `deploy/nginx/sql-playground-cloudflare.conf` (origin certificate, Full (strict)) and `deploy/scripts/cloudflare-realip.sh` (real visitor IPs, optional Cloudflare-only firewall). The steps below are the short version for a server that already has nginx and certbot.

1. **Get the code**
   ```bash
   sudo mkdir -p /opt/sql-playground && sudo chown "$USER": /opt/sql-playground
   ```
   ```bash
   git clone https://github.com/darshan260802/sql-playground.git /opt/sql-playground
   ```

2. **Install** (about 2–5 minutes the first time)
   ```bash
   cd /opt/sql-playground && sudo ./deploy/install.sh
   ```
   The script installs MySQL from Ubuntu's packages, applies `deploy/mysql/playground.cnf`, creates the app's MySQL account with a random password, builds the app with its own copy of Go and Node.js (kept in `/usr/local/lib/sql-playground/toolchain`, so it doesn't touch any you already have), and starts the `sql-playground` systemd service. It finishes once the app answers on `127.0.0.1:8080`. If port 8080 is taken, use `sudo ./deploy/install.sh --port 8081`.

3. **nginx**: copy `deploy/nginx/sql-playground.conf` to `/etc/nginx/sites-available/`, replace `sql.example.com` with your domain (and the port if you changed it), then:
   ```bash
   sudo ln -s /etc/nginx/sites-available/sql-playground.conf /etc/nginx/sites-enabled/
   ```
   ```bash
   sudo nginx -t && sudo systemctl reload nginx
   ```
   ```bash
   sudo certbot --nginx -d sql.example.com
   ```

4. **ufw**: nothing to open. The app listens on `127.0.0.1` only and MySQL on localhost only, so your existing `Nginx Full` / `OpenSSH` rules are enough.

> Behind HTTPS keep `SQLPG_COOKIE_SECURE=true`. For a quick test over plain HTTP set it to `false`, otherwise the browser won't send the login cookie.

### What gets installed where

| Path | What it is |
|---|---|
| `/etc/sql-playground/config.env` | settings and the app's MySQL password (root-only) |
| `/etc/systemd/system/sql-playground.service` | the service, run as the unprivileged `sqlpg` user |
| `/usr/local/lib/sql-playground/` | the app (`bin/sqlpg`, `web/`), its build toolchain and build cache |
| `/usr/local/bin/sqlpg-admin` | admin commands (below) |
| `/etc/mysql/mysql.conf.d/zz-sql-playground.cnf` | MySQL settings (server-wide: read it first if other projects share this MySQL) |
| `/etc/cron.d/sql-playground` | nightly backup at 03:17 |
| `/var/backups/sql-playground/` | backups (the last 14) |

### Resources (measured)

- **Disk:** ~1.1 GB for MySQL's base files and the core data, plus **148 MB per learner**, up to the 300 MB quota. 20 learners need 3–6 GB in total. The build toolchain and caches take about 650 MB.
- **RAM:** MySQL about 1.2 GB (2 GB buffer pool cap); the app about 30 MB.
- To use less RAM, lower `innodb_buffer_pool_size` in `deploy/mysql/playground.cnf` (e.g. `1G`) and run `install.sh` again.

### Everyday operations

| Task | Command |
|---|---|
| Status / logs | `systemctl status sql-playground` · `journalctl -u sql-playground -f` |
| List accounts and storage | `sudo sqlpg-admin list-users` |
| Reset a forgotten password (prints a new one; there's no email) | `sudo sqlpg-admin reset-password EMAIL` |
| Restore a learner's data to the original | `sudo sqlpg-admin reset-db EMAIL` |
| Delete an account and its database | `sudo sqlpg-admin delete-user EMAIL` |
| Update to the latest code | `cd /opt/sql-playground && git pull && sudo ./deploy/install.sh` |
| Change a setting | `sudo nano /etc/sql-playground/config.env && sudo systemctl restart sql-playground` |
| Back up now | `sudo /usr/local/lib/sql-playground/backup.sh` |

**Backups:** only `playground_app` holds data worth keeping (accounts, saved queries, progress); learner copies are recreated automatically after a restore. To restore, on the same server or a new one after `install.sh`:
```bash
sudo gunzip -c /var/backups/sql-playground/app-2026-01-31.sql.gz | sudo mysql
```
```bash
sudo systemctl restart sql-playground
```

### Configuration

All settings live in `/etc/sql-playground/config.env` (template: `deploy/config.env.example`):

| Variable | Default | Meaning |
|---|---|---|
| `SQLPG_LISTEN` | 127.0.0.1:8080 | where nginx reaches the app |
| `SQLPG_MAX_USERS` | 20 | signups close when this many accounts exist |
| `SQLPG_USER_QUOTA_MB` | 300 | storage limit per learner database |
| `SQLPG_QUERY_TIMEOUT` / `SQLPG_ANON_QUERY_TIMEOUT` | 30s / 15s | per-run time limit |
| `SQLPG_MAX_ROWS` | 2000 | rows sent to the browser per result |
| `SQLPG_MAX_CONCURRENT` | 8 | queries executing at once, server-wide |
| `SQLPG_SESSION_IDLE` | 15m | idle time before a learner's SQL session (and open transaction) is closed |
| `SQLPG_SESSION_DAYS` | 30 | login cookie lifetime |
| `SQLPG_COOKIE_SECURE` | true | send cookies over HTTPS only |
| `SQLPG_TRUST_PROXY` | true | take client IPs from nginx headers (for rate limits) |

### Uninstall

```bash
sudo systemctl disable --now sql-playground
```
```bash
sudo rm -rf /etc/systemd/system/sql-playground.service /usr/local/lib/sql-playground /usr/local/bin/sqlpg-admin /etc/cron.d/sql-playground /etc/sql-playground
```
The databases stay in MySQL until you drop them (`core`, `playground_app`, `u_*`) or remove MySQL itself.

## Local development

Requirements: Go 1.24+, Node 22+, and MySQL 8.0.31+. On Ubuntu, `sudo apt install mysql-server`, then create a development account:

```bash
sudo mysql -e "CREATE USER 'dev'@'127.0.0.1' IDENTIFIED BY 'dev'; GRANT ALL ON *.* TO 'dev'@'127.0.0.1' WITH GRANT OPTION;"
```

API on :8080 (generates the dataset on first start):

```bash
cd backend && SQLPG_MYSQL_ADMIN_USER=dev SQLPG_MYSQL_ADMIN_PASSWORD=dev go run ./cmd/sqlpg serve
```

Frontend on :5173 with hot reload (proxies `/api` to :8080):

```bash
cd frontend && npm install && npm run dev
```

### Tests

```bash
cd backend && go test ./...
```

The lesson suite runs **every exercise against a real MySQL**. It checks that the solution passes its own check and that a wrong answer fails:

```bash
cd backend && SQLPG_TEST_DSN='dev:dev@tcp(127.0.0.1:3306)/' go test ./internal/lessons/ -run TestSolutions -v
```

### Writing lessons

Lessons are Markdown files in `backend/internal/lessons/content/` (`NN-slug.md`) with a small header and `=== exercise` blocks (prompt, solution, hint, and for DML/DDL a `check` query and a `setup`). See the format notes at the top of `backend/internal/lessons/lessons.go`, then run the lesson test suite.
