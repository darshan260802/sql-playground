import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion } from 'motion/react'
import { ArrowRight, CheckCircle2, Clock } from 'lucide-react'
import { api, type ModuleSummary } from '../lib/api'
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

  const total = modules?.reduce((n, m) => n + m.lessons.reduce((k, l) => k + l.exercises.length, 0), 0) ?? 0
  const lessonCount = modules?.reduce((n, m) => n + m.lessons.length, 0) ?? 0
  const done = solved.size
  let lessonNo = 0

  // first lesson with unsolved exercises: where "continue" takes you
  const next = modules?.flatMap((m) => m.lessons).find((l) => l.exercises.some((x) => !solved.has(x)))

  return (
    <div className="learn">
      <section className="learn-hero">
        <div className="learn-hero-text">
          <span className="eyebrow">Learn SQL by doing</span>
          <h1>From your first SELECT to window functions.</h1>
          <p className="lead">
            Hands-on lessons on realistic data from an online store, an HR system and a bank. Every exercise is checked
            automatically against the real database, so you know the moment you get it right.
          </p>
          <div className="hero-actions">
            {next && (
              <Link to={`/learn/${next.slug}`} className="btn btn-primary btn-lg">
                {done > 0 ? 'Continue learning' : 'Start the first lesson'} <ArrowRight />
              </Link>
            )}
            <Link to="/" className="btn btn-lg">
              Open the playground
            </Link>
          </div>
        </div>
        <div className="stats">
          <Stat value={String(lessonCount || config?.lesson_count || 30)} label="lessons" />
          <Stat value={String(total || config?.exercise_count || 116)} label="checked exercises" />
          <Stat value="1.7M" label="rows of data" />
          <Stat value="24" label="tables, 3 domains" />
        </div>
      </section>

      {done > 0 && total > 0 && (
        <div className="progress-card card">
          <div>
            <div className="progress-title">
              {done} of {total} exercises solved
            </div>
            <div className="muted small">{user ? 'Saved to your account.' : 'Saved in this browser. Create an account to keep it everywhere.'}</div>
          </div>
          <div className="meter big">
            <motion.span initial={{ width: 0 }} animate={{ width: `${(done / total) * 100}%` }} transition={spring} />
          </div>
        </div>
      )}

      {error && <div className="sidebar-note error">{error}</div>}
      {!modules && !error && (
        <div className="lesson-grid">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="skeleton" style={{ height: 172, borderRadius: 12 }} />
          ))}
        </div>
      )}

      {modules?.map((m) => (
        <section key={m.name} className="module">
          <div className="module-head">
            <h2>{m.name}</h2>
            <span className="muted">{m.lessons.length} lessons</span>
          </div>
          <div className="lesson-grid">
            {m.lessons.map((l) => {
              lessonNo++
              const n = l.exercises.filter((x) => solved.has(x)).length
              const complete = n === l.exercises.length && n > 0
              return (
                <Link key={l.slug} to={`/learn/${l.slug}`} className={`lesson-card ${complete ? 'complete' : ''}`}>
                  <div className="lesson-card-top">
                    <span className="lesson-no mono">{String(lessonNo).padStart(2, '0')}</span>
                    {complete ? <CheckCircle2 className="lesson-done" /> : <span className={`level level-${l.level.toLowerCase()}`}>{l.level}</span>}
                  </div>
                  <h3>{l.title}</h3>
                  <p>{l.summary}</p>
                  <div className="lesson-card-foot">
                    <span>
                      <Clock size={13} /> {l.minutes} min
                    </span>
                    <span>
                      {n}/{l.exercises.length} exercises
                    </span>
                  </div>
                  <div className="meter thin">
                    <span style={{ width: `${(n / Math.max(1, l.exercises.length)) * 100}%` }} />
                  </div>
                </Link>
              )
            })}
          </div>
        </section>
      ))}

      {!user && modules && (
        <section className="cta-band">
          <div>
            <h2>Practice on your own copy of the data.</h2>
            <p>
              Signed-out visitors can run any read query. With a free account you get a private copy of every table to
              INSERT, UPDATE, DELETE and CREATE in, and you can reset it to the original whenever you like.
            </p>
          </div>
          <button className="btn btn-lg btn-ink" onClick={() => openAuth('signup')}>
            Create a free account <ArrowRight />
          </button>
        </section>
      )}
    </div>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="stat">
      <div className="stat-value">{value}</div>
      <div className="stat-label">{label}</div>
    </div>
  )
}
