import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { ArrowRight, Check } from 'lucide-react'
import { api, type LessonSummary, type ModuleSummary } from '../lib/api'
import { useAuth } from '../state/auth'
import { localSolved, useUI } from '../state/stores'
import { spring } from '../lib/util'

export function Learn() {
  const { user, config } = useAuth()
  const openAuth = useUI((s) => s.openAuth)
  const [modules, setModules] = useState<ModuleSummary[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api
      .lessons()
      .then((r) => setModules(r.modules))
      .catch((e) => setError((e as Error).message))
  }, [user?.id])

  const solved = useMemo(() => {
    const s = new Set<string>(user ? [] : localSolved())
    modules?.forEach((m) => m.lessons.forEach((l) => l.solved.forEach((x) => s.add(x))))
    return s
  }, [modules, user])

  const lessons = modules?.flatMap((m) => m.lessons) ?? []
  const total = lessons.reduce((k, l) => k + l.exercises.length, 0) || config?.exercise_count || 0
  const lessonCount = lessons.length || config?.lesson_count || 0
  const solvedIn = (l: LessonSummary) => l.exercises.filter((x) => solved.has(x)).length
  const done = lessons.reduce((k, l) => k + solvedIn(l), 0)
  const lessonsDone = lessons.filter((l) => l.exercises.length > 0 && solvedIn(l) === l.exercises.length).length
  // the first lesson with an unsolved exercise is where "continue" goes
  const next = lessons.find((l) => l.exercises.some((x) => !solved.has(x)))

  return (
    <div className="page learn">
      <header className="page-head">
        <div className="page-head-text">
          <h1>Learn</h1>
          <p className="muted">
            {lessonCount} lessons from your first SELECT to window functions. Every exercise is checked against the real database.
          </p>
        </div>
        {next && (
          <Link to={`/learn/${next.slug}`} className="btn btn-primary btn-lg">
            {done > 0 ? 'Continue' : 'Start'}: {next.title} <ArrowRight />
          </Link>
        )}
      </header>

      <div className="learn-layout">
        <aside className="learn-side">
          <section className="side-block">
            <h2>Your progress</h2>
            <p className="progress-figure num" aria-label={`${done} of ${total} exercises solved`}>
              {done}
              <span className="faint"> / {total}</span>
            </p>
            <div className="meter" role="meter" aria-valuemin={0} aria-valuemax={total} aria-valuenow={done} aria-label="Exercises solved">
              <motion.span initial={false} animate={{ width: `${total ? (done / total) * 100 : 0}%` }} transition={spring} />
            </div>
            <p className="muted small tnum">
              Exercises solved, across {lessonsDone} of {lessonCount} completed lessons. {user ? 'Saved to your account.' : 'Saved in this browser.'}
            </p>
          </section>
          {!user && (
            <section className="side-block">
              <h2>Practice on your own copy</h2>
              <p className="muted small">
                Anyone can run read queries. A free account gives you a private copy of every table to INSERT, UPDATE, DELETE and CREATE in,
                which you can reset to the original at any time. It also keeps your progress on every device.
              </p>
              <button className="btn btn-block" onClick={() => openAuth('signup')}>
                Create a free account
              </button>
            </section>
          )}
          <section className="side-block">
            <h2>The data</h2>
            <p className="muted small">
              24 tables and about 1.7 million rows: an online store (orders, customers, products), an HR system (employees, salaries,
              departments) and a bank (accounts, transactions, stock prices).
            </p>
            <Link to="/" className="btn btn-text btn-sm side-link">
              Explore it in the playground <ArrowRight />
            </Link>
          </section>
        </aside>

        <div className="learn-main">
          {error && <p className="block-error">{error}</p>}
          {!modules && !error && <LessonListSkeleton />}
          {modules?.map((m) => {
            const mTotal = m.lessons.reduce((k, l) => k + l.exercises.length, 0)
            const mDone = m.lessons.reduce((k, l) => k + solvedIn(l), 0)
            return (
              <section key={m.name} className="module">
                <div className="module-head">
                  <h2>{m.name}</h2>
                  <span className="num faint">
                    {mDone}/{mTotal}
                  </span>
                </div>
                <ol className="lesson-list">
                  {m.lessons.map((l) => {
                    const n = solvedIn(l)
                    return (
                      <li key={l.slug}>
                        <Link to={`/learn/${l.slug}`} className={`lesson-row ${next?.slug === l.slug ? 'is-next' : ''}`}>
                          <ProgressMark n={n} total={l.exercises.length} />
                          <span className="lesson-row-main">
                            <span className="lesson-row-title">{l.title}</span>
                            <span className="lesson-row-summary">{l.summary}</span>
                          </span>
                          <span className="lesson-row-meta">
                            <span>{l.level}</span>
                            <span className="num">{l.minutes} min</span>
                          </span>
                          <span className="lesson-row-count num" aria-label={`${n} of ${l.exercises.length} exercises solved`}>
                            {n}/{l.exercises.length}
                          </span>
                        </Link>
                      </li>
                    )
                  })}
                </ol>
              </section>
            )
          })}
        </div>
      </div>
    </div>
  )
}

/** A lesson's progress at its true ratio: an empty ring, a partial arc, or a check when every exercise is solved. */
export function ProgressMark({ n, total }: { n: number; total: number }) {
  if (total > 0 && n >= total) {
    return (
      <span className="progress-mark done" aria-hidden="true">
        <Check />
      </span>
    )
  }
  const r = 7
  const c = 2 * Math.PI * r
  return (
    <svg className="progress-mark" width="18" height="18" viewBox="0 0 18 18" aria-hidden="true">
      <circle cx="9" cy="9" r={r} fill="none" stroke="var(--control)" strokeWidth="1.5" />
      {n > 0 && (
        <circle
          cx="9"
          cy="9"
          r={r}
          fill="none"
          stroke="var(--text-2)"
          strokeWidth="1.5"
          strokeDasharray={`${(n / total) * c} ${c}`}
          transform="rotate(-90 9 9)"
          strokeLinecap="butt"
        />
      )}
    </svg>
  )
}

function LessonListSkeleton() {
  return (
    <section className="module" aria-busy="true">
      <div className="module-head">
        <div className="skeleton" style={{ height: 16, width: 120 }} />
      </div>
      <ol className="lesson-list">
        {Array.from({ length: 5 }).map((_, i) => (
          <li key={i} className="lesson-row">
            <span className="skeleton" style={{ width: 18, height: 18, borderRadius: 9 }} />
            <span className="lesson-row-main">
              <span className="skeleton" style={{ height: 14, width: `${40 + ((i * 17) % 30)}%` }} />
              <span className="skeleton" style={{ height: 12, width: `${60 + ((i * 11) % 30)}%` }} />
            </span>
          </li>
        ))}
      </ol>
    </section>
  )
}
