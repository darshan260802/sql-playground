import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { AlertTriangle, CheckCircle2, CircleAlert, Lightbulb, Play, TriangleAlert } from 'lucide-react'
import type { StatementResult } from '../lib/api'
import type { RunState } from '../state/stores'
import { formatDuration, formatInt, modKey, springSnappy } from '../lib/util'
import { DataGrid } from './DataGrid'

interface Props {
  run?: RunState
  onSelect: (i: number) => void
  onJumpToLine: (line: number) => void
  onSignup: () => void
  onUseMine?: () => void
  onStarter: (sql: string, title: string) => void
  maxRows: number
}

const STARTERS: { title: string; what: string; sql: string }[] = [
  {
    title: 'Revenue by month',
    what: 'JOIN two tables, GROUP BY a formatted date',
    sql: `SELECT DATE_FORMAT(o.order_date, '%Y-%m') AS month,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount)), 2) AS revenue
FROM orders o
JOIN order_items oi ON oi.order_id = o.order_id
WHERE o.status NOT IN ('cancelled', 'returned')
GROUP BY month
ORDER BY month;`,
  },
  {
    title: 'Best-selling products',
    what: 'Three-table JOIN, SUM, top 10 with LIMIT',
    sql: `SELECT p.name, c.name AS category, SUM(oi.quantity) AS units
FROM order_items oi
JOIN products p ON p.product_id = oi.product_id
JOIN categories c ON c.category_id = p.category_id
GROUP BY p.product_id, p.name, c.name
ORDER BY units DESC
LIMIT 10;`,
  },
  {
    title: 'Salary by department',
    what: 'AVG per group, ranked with a window function',
    sql: `SELECT d.name AS department,
       COUNT(*) AS people,
       ROUND(AVG(s.salary)) AS avg_salary,
       RANK() OVER (ORDER BY AVG(s.salary) DESC) AS pay_rank
FROM employees e
JOIN departments d ON d.department_id = e.department_id
JOIN salaries s ON s.employee_id = e.employee_id AND s.to_date IS NULL
WHERE e.termination_date IS NULL
GROUP BY d.name;`,
  },
  {
    title: 'Stock with 20-day average',
    what: 'A moving average with ROWS BETWEEN',
    sql: `SELECT s.ticker, p.trade_date, p.close,
       ROUND(AVG(p.close) OVER (ORDER BY p.trade_date ROWS BETWEEN 19 PRECEDING AND CURRENT ROW), 2) AS ma_20
FROM stock_prices p
JOIN stocks s ON s.stock_id = p.stock_id
WHERE s.stock_id = 1
ORDER BY p.trade_date DESC
LIMIT 60;`,
  },
]

function verb(sql: string) {
  const m = sql.replace(/^\s*(\/\*[\s\S]*?\*\/|--[^\n]*\n|#[^\n]*\n|\s)*/, '').match(/^[a-zA-Z]+/)
  return m ? m[0].toUpperCase() : 'SQL'
}

function Elapsed() {
  const [start] = useState(() => performance.now())
  const [now, setNow] = useState(start)
  useEffect(() => {
    const id = setInterval(() => setNow(performance.now()), 100)
    return () => clearInterval(id)
  }, [])
  return <span className="mono">{((now - start) / 1000).toFixed(1)} s</span>
}

export function ResultsPanel({ run, onSelect, onJumpToLine, onSignup, onUseMine, onStarter, maxRows }: Props) {
  const resp = run?.response
  const results = resp?.results ?? []
  const active = Math.min(run?.active ?? 0, Math.max(0, results.length - 1))
  const current: StatementResult | undefined = results[active]

  if (!run || (!resp && !run.running && !run.error)) {
    return (
      <div className="results-empty">
        <div className="results-empty-main">
          <h2>Results appear here</h2>
          <p className="muted">
            Run the query above with <kbd>{modKey}</kbd> + <kbd>Enter</kbd>, select part of it to run only that, or start from an example:
          </p>
          <ul className="starters">
            {STARTERS.map((s) => (
              <li key={s.title}>
                <button className="starter" onClick={() => onStarter(s.sql, s.title)}>
                  <span className="starter-title">{s.title}</span>
                  <span className="starter-what">{s.what}</span>
                  <span className="starter-run">
                    <Play /> Run
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
        <aside className="results-empty-side">
          <h2>Good to know</h2>
          <dl className="keys">
            <div>
              <dt>Run the query</dt>
              <dd>
                <kbd>{modKey}</kbd> <kbd>Enter</kbd>
              </dd>
            </div>
            <div>
              <dt>Run only the selection</dt>
              <dd>Select, then <kbd>{modKey}</kbd> <kbd>Enter</kbd></dd>
            </div>
            <div>
              <dt>Save the query</dt>
              <dd>
                <kbd>{modKey}</kbd> <kbd>S</kbd>
              </dd>
            </div>
            <div>
              <dt>Several statements</dt>
              <dd>End each with ;</dd>
            </div>
          </dl>
          <p className="muted small">
            Results show up to {formatInt(maxRows)} rows with the true row count. Double-click a cell to copy its value.
          </p>
        </aside>
      </div>
    )
  }

  return (
    <div className="results">
      <div className="results-bar">
        <div className="tabs results-tabs" role="tablist" aria-label="Statement results">
          {results.length > 1 &&
            results.map((r, i) => (
              <button key={i} role="tab" aria-selected={i === active} onClick={() => onSelect(i)} title={r.sql.slice(0, 200)}>
                <span className={`dot ${r.error ? 'err' : 'ok'}`} />
                <span className="mono">{verb(r.sql)}</span>
                <span className="count">{i + 1}</span>
                {i === active && <motion.span layoutId="res-line" className="tab-line" transition={springSnappy} />}
              </button>
            ))}
          {results.length <= 1 && current && <span className="results-title mono">{verb(current.sql)}</span>}
        </div>
        <div className="results-meta num">
          {run.running ? (
            <span className="running">
              <span className="spinner" /> Running <Elapsed />
            </span>
          ) : (
            current &&
            !current.error && (
              <>
                {current.columns && (
                  <span>
                    {current.truncated
                      ? `First ${formatInt(current.rows?.length ?? 0)} of ${formatInt(current.row_count)} rows`
                      : `${formatInt(current.row_count)} ${current.row_count === 1 ? 'row' : 'rows'}`}
                  </span>
                )}
                <span className="mono">{formatDuration(current.duration_ms)}</span>
              </>
            )
          )}
        </div>
      </div>
      {run.running && <div className="progress-indeterminate" />}
      <div className={`results-body ${run.running ? 'stale' : ''}`}>
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={`${resp?.duration_ms ?? 'x'}-${active}-${run.error ?? ''}`}
            className="results-content"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.12 }}
          >
            {run.error && !resp ? (
              <div className="result-card error">
                <CircleAlert />
                <div>
                  <div className="result-card-title">{run.error}</div>
                  {run.errorCode === 'db_not_ready' && <p>Switch to “Original” while your copy is prepared.</p>}
                </div>
              </div>
            ) : current ? (
              <ResultView r={current} maxRows={maxRows} onJumpToLine={onJumpToLine} onSignup={onSignup} onUseMine={onUseMine} />
            ) : null}
          </motion.div>
        </AnimatePresence>
      </div>
    </div>
  )
}

function ResultView({ r, maxRows, onJumpToLine, onSignup, onUseMine }: { r: StatementResult; maxRows: number; onJumpToLine: (l: number) => void; onSignup: () => void; onUseMine?: () => void }) {
  if (r.error) {
    const e = r.error
    return (
      <div className="result-scroll">
        <div className="result-card error" role="alert">
          <CircleAlert />
          <div className="result-card-main">
            <div className="result-card-title">
              {e.code ? <span className="pill pill-negative num">Error {e.code}</span> : null}
              <span>{e.message}</span>
            </div>
            {e.hint && (
              <p className="hint">
                <Lightbulb /> {e.hint}
              </p>
            )}
            <div className="result-actions">
              {e.line ? (
                <button className="btn btn-sm" onClick={() => onJumpToLine(e.line!)}>
                  Go to line {e.line}
                </button>
              ) : null}
              {e.reason === 'read_only_anon' && (
                <button className="btn btn-sm" onClick={onSignup}>
                  Create a free account
                </button>
              )}
              {e.reason === 'read_only_core' && onUseMine && (
                <button className="btn btn-sm" onClick={onUseMine}>
                  Switch to your copy
                </button>
              )}
            </div>
          </div>
        </div>
        <Statement sql={r.sql} />
      </div>
    )
  }
  if (r.columns && r.columns.length > 0) {
    return (
      <>
        {r.truncated && (
          <div className="result-note">
            <AlertTriangle /> Showing the first {formatInt(maxRows)} rows. Add a <code>LIMIT</code> or filters to narrow it down.
          </div>
        )}
        <DataGrid columns={r.columns} rows={r.rows ?? []} />
        <Warnings w={r.warnings} />
      </>
    )
  }
  return (
    <div className="result-scroll">
      <div className="result-card ok">
        <CheckCircle2 />
        <div className="result-card-main">
          <div className="result-card-title">{r.message ?? 'Query OK'}</div>
          {r.last_insert_id ? <p className="muted">Last insert id <span className="num">{r.last_insert_id}</span></p> : null}
          <p className="muted num">{formatDuration(r.duration_ms)}</p>
        </div>
      </div>
      <Warnings w={r.warnings} />
      <Statement sql={r.sql} />
    </div>
  )
}

function Warnings({ w }: { w?: string[] }) {
  if (!w || w.length === 0) return null
  return (
    <div className="warnings">
      {w.map((x, i) => (
        <div key={i}>
          <TriangleAlert /> {x}
        </div>
      ))}
    </div>
  )
}

function Statement({ sql }: { sql: string }) {
  return <pre className="statement-echo">{sql.length > 600 ? sql.slice(0, 600) + '…' : sql}</pre>
}
