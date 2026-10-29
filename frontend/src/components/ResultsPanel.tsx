import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { AlertTriangle, CheckCircle2, CircleAlert, Lightbulb, TableProperties, TriangleAlert } from 'lucide-react'
import type { StatementResult } from '../lib/api'
import type { RunState } from '../state/stores'
import { formatDuration, formatInt, spring, springSnappy } from '../lib/util'
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

const STARTERS: { title: string; sql: string }[] = [
  {
    title: 'Revenue by month',
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
        <div className="results-empty-inner">
          <TableProperties className="results-empty-icon" />
          <h3>Run a query to see results</h3>
          <p>
            Press <kbd>Ctrl</kbd>/<kbd>⌘</kbd> + <kbd>Enter</kbd>, or try one of these:
          </p>
          <div className="starter-grid">
            {STARTERS.map((s) => (
              <button key={s.title} className="starter" onClick={() => onStarter(s.sql, s.title)}>
                {s.title}
              </button>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="results">
      <div className="results-bar">
        <div className="results-tabs" role="tablist" aria-label="Statement results">
          {results.length > 1 &&
            results.map((r, i) => (
              <button key={i} role="tab" aria-selected={i === active} className="results-tab" onClick={() => onSelect(i)}>
                {i === active && <motion.span layoutId="res-tab" className="results-tab-bg" transition={springSnappy} />}
                <span className={`dot ${r.error ? 'err' : 'ok'}`} />
                <span>
                  {i + 1} · {verb(r.sql)}
                </span>
              </button>
            ))}
          {results.length <= 1 && current && <span className="results-title">{verb(current.sql)}</span>}
        </div>
        <div className="results-meta">
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
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ ...spring, duration: 0.25 }}
          >
            {run.error && !resp ? (
              <div className="result-card error">
                <CircleAlert />
                <div>
                  <div className="result-card-title">{run.error}</div>
                  {run.errorCode === 'db_not_ready' && <p>Switch the target to “Original data” while your copy is prepared.</p>}
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
              {e.code ? <span className="badge badge-error">Error {e.code}</span> : null}
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
                <button className="btn btn-sm btn-primary" onClick={onSignup}>
                  Create a free account
                </button>
              )}
              {e.reason === 'read_only_core' && onUseMine && (
                <button className="btn btn-sm btn-primary" onClick={onUseMine}>
                  Switch to My database
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
          {r.last_insert_id ? <p className="muted">Last insert id: {r.last_insert_id}</p> : null}
          <p className="muted mono">{formatDuration(r.duration_ms)}</p>
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
