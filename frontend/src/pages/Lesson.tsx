import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { toast } from 'sonner'
import { ArrowLeft, ArrowRight, Check, ChevronDown, CircleAlert, Clock, Eye, Lightbulb, Lock, Play, Sparkles } from 'lucide-react'
import { api, ApiError, newRunId, type CheckOutcome, type Exercise, type Lesson as LessonT, type StatementResult } from '../lib/api'
import { useAuth } from '../state/auth'
import { addLocalSolved, localSolved, useTabs, useUI } from '../state/stores'
import { modKey, spring, springMomentum, storage } from '../lib/util'
import { Markdown, HighlightedSQL } from '../components/Markdown'
import { SqlEditor, type EditorHandle } from '../components/SqlEditor'
import { DataGrid } from '../components/DataGrid'

const KIND_LABEL: Record<Exercise['kind'], string> = { select: 'Query', dml: 'Changes data', ddl: 'Schema design' }

export function Lesson() {
  const { slug = '' } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [lesson, setLesson] = useState<LessonT | null>(null)
  const [error, setError] = useState('')
  const [idx, setIdx] = useState(0)
  const [solved, setSolved] = useState<Set<string>>(new Set())

  useEffect(() => {
    setLesson(null)
    setError('')
    api
      .lesson(slug)
      .then(({ lesson }) => {
        setLesson(lesson)
        const s = new Set<string>(lesson.exercises.filter((e) => e.solved).map((e) => e.id))
        if (!user) localSolved().forEach((id) => s.add(id))
        setSolved(s)
        const firstOpen = lesson.exercises.findIndex((e) => !s.has(e.id))
        setIdx(firstOpen === -1 ? 0 : firstOpen)
      })
      .catch((e) => setError((e as ApiError).message))
    window.scrollTo({ top: 0 })
  }, [slug, user])

  const openInPlayground = useCallback(
    (sql: string) => {
      useTabs.getState().newTab(sql, lesson?.title ?? 'Lesson example')
      navigate('/')
    },
    [lesson, navigate],
  )

  if (error) {
    return (
      <div className="lesson-missing">
        <h2>{error}</h2>
        <Link to="/learn" className="btn">
          <ArrowLeft /> All lessons
        </Link>
      </div>
    )
  }
  if (!lesson) {
    return (
      <div className="lesson-layout">
        <div className="lesson-article">
          <div className="skeleton" style={{ height: 22, width: 160, marginBottom: 18 }} />
          <div className="skeleton" style={{ height: 48, width: '70%', marginBottom: 28 }} />
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="skeleton" style={{ height: 16, width: `${70 + ((i * 13) % 30)}%`, marginBottom: 12 }} />
          ))}
        </div>
      </div>
    )
  }

  const ex = lesson.exercises[idx]
  const allDone = lesson.exercises.every((e) => solved.has(e.id))

  return (
    <div className="lesson-layout">
      <article className="lesson-article">
        <nav className="crumbs" aria-label="Breadcrumb">
          <Link to="/learn">Learn</Link>
          <span>/</span>
          <span>{lesson.module}</span>
        </nav>
        <h1 className="lesson-title">{lesson.title}</h1>
        <div className="lesson-meta">
          <span className={`level level-${lesson.level.toLowerCase()}`}>{lesson.level}</span>
          <span>
            <Clock size={14} /> {lesson.minutes} min
          </span>
          <span>{lesson.exercises.length} exercises</span>
        </div>
        <p className="lesson-summary">{lesson.summary}</p>
        <Markdown text={lesson.body} onOpenSql={openInPlayground} className="lesson-body" />
        <div className="lesson-nav">
          {lesson.prev ? (
            <Link to={`/learn/${lesson.prev.slug}`} className="lesson-nav-link">
              <span className="eyebrow">Previous</span>
              <span>
                <ArrowLeft size={15} /> {lesson.prev.title}
              </span>
            </Link>
          ) : (
            <span />
          )}
          {lesson.next && (
            <Link to={`/learn/${lesson.next.slug}`} className="lesson-nav-link next">
              <span className="eyebrow">Next</span>
              <span>
                {lesson.next.title} <ArrowRight size={15} />
              </span>
            </Link>
          )}
        </div>
      </article>

      <aside className="exercise-panel" aria-label="Exercises">
        <div className="exercise-sticky">
          <div className="stepper" role="tablist" aria-label="Exercises in this lesson">
            {lesson.exercises.map((e, i) => (
              <button key={e.id} role="tab" aria-selected={i === idx} className={`step ${solved.has(e.id) ? 'done' : ''}`} onClick={() => setIdx(i)} title={e.title}>
                {i === idx && <motion.span layoutId="step-ring" className="step-ring" transition={spring} />}
                {solved.has(e.id) ? <Check /> : i + 1}
              </button>
            ))}
            <span className="stepper-label">
              {allDone ? 'Lesson complete' : `Exercise ${idx + 1} of ${lesson.exercises.length}`}
            </span>
          </div>
          <AnimatePresence mode="wait" initial={false}>
            <motion.div key={ex.id} initial={{ opacity: 0, x: 12 }} animate={{ opacity: 1, x: 0 }} exit={{ opacity: 0, x: -12 }} transition={{ ...spring, duration: 0.28 }}>
              <ExerciseCard
                ex={ex}
                signedIn={!!user}
                dbReady={user?.db_status === 'ready'}
                solved={solved.has(ex.id)}
                onSolved={() => {
                  setSolved((s) => new Set(s).add(ex.id))
                  if (!user) addLocalSolved(ex.id)
                }}
                onNext={idx < lesson.exercises.length - 1 ? () => setIdx(idx + 1) : undefined}
                nextLesson={lesson.next}
              />
            </motion.div>
          </AnimatePresence>
        </div>
      </aside>
    </div>
  )
}

interface CardProps {
  ex: Exercise
  signedIn: boolean
  dbReady: boolean
  solved: boolean
  onSolved: () => void
  onNext?: () => void
  nextLesson: LessonT['next']
}

function ExerciseCard({ ex, signedIn, dbReady, solved, onSolved, onNext, nextLesson }: CardProps) {
  const openAuth = useUI((s) => s.openAuth)
  const draftKey = `sqlpg.draft.${ex.id}`
  const [sql, setSql] = useState(() => storage.get<string>(draftKey, '') || ex.last_sql || '')
  const [busy, setBusy] = useState<'check' | 'run' | null>(null)
  const [outcome, setOutcome] = useState<CheckOutcome | null>(null)
  const [runResult, setRunResult] = useState<StatementResult | null>(null)
  const [showHint, setShowHint] = useState(false)
  const [solution, setSolution] = useState<string | null>(null)
  const [showExpected, setShowExpected] = useState(true)
  const [attempt, setAttempt] = useState(0)
  const editor = useRef<EditorHandle | null>(null)
  const reduce = useReducedMotion()
  const locked = ex.requires_login && !signedIn

  useEffect(() => {
    storage.set(draftKey, sql)
  }, [draftKey, sql])

  const check = useCallback(async () => {
    if (locked) {
      openAuth('signup', 'This exercise changes data, so it runs on your own copy of the database. Create a free account to try it.')
      return
    }
    const text = editor.current?.getRunnable().sql ?? sql
    if (!text.trim()) {
      toast('Write a query first')
      return
    }
    setBusy('check')
    setRunResult(null)
    try {
      const o = await api.check(ex.id, text)
      setOutcome(o)
      setAttempt((a) => a + 1)
      if (o.correct) onSolved()
    } catch (e) {
      const err = e as ApiError
      if (err.code === 'auth_required') openAuth('signup', err.message)
      else toast.error(err.message)
    } finally {
      setBusy(null)
    }
  }, [ex.id, sql, locked, openAuth, onSolved])

  const runOnly = async () => {
    const text = editor.current?.getRunnable().sql ?? sql
    if (!text.trim()) return
    setBusy('run')
    setOutcome(null)
    try {
      const r = await api.query(text, 'core', newRunId())
      const last = [...r.results].reverse().find((x) => x.error || x.columns) ?? r.results[r.results.length - 1]
      setRunResult(last)
    } catch (e) {
      toast.error((e as Error).message)
    } finally {
      setBusy(null)
    }
  }

  const reveal = async () => {
    try {
      const { solution } = await api.solution(ex.id)
      setSolution(solution)
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  return (
    <div className="exercise card">
      <div className="exercise-head">
        <h3>{ex.title}</h3>
        <div className="exercise-badges">
          <span className="badge">{KIND_LABEL[ex.kind]}</span>
          {ex.ordered && <span className="badge" title="Row order matters">Ordered</span>}
          {solved && (
            <span className="badge badge-success">
              <Check size={12} /> Solved
            </span>
          )}
        </div>
      </div>
      <Markdown text={ex.prompt} className="exercise-prompt" />

      {ex.expected && (
        <div className="expected">
          <button className="expected-toggle" onClick={() => setShowExpected((s) => !s)} aria-expanded={showExpected}>
            <motion.span animate={{ rotate: showExpected ? 0 : -90 }} transition={spring} className="chev">
              <ChevronDown size={15} />
            </motion.span>
            Expected result
            <span className="muted">
              · {ex.expected.row_count} {ex.expected.row_count === 1 ? 'row' : 'rows'}
              {ex.expected.rows.length < ex.expected.row_count ? `, first ${ex.expected.rows.length} shown` : ''}
            </span>
          </button>
          <AnimatePresence initial={false}>
            {showExpected && (
              <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={spring} style={{ overflow: 'hidden' }}>
                <DataGrid columns={ex.expected.columns} rows={ex.expected.rows} dense maxHeight={190} />
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      )}

      {locked ? (
        <div className="locked">
          <Lock />
          <div>
            <b>This one changes data.</b> It runs on your own private copy of the database, so you need a free account.
          </div>
          <button className="btn btn-primary btn-sm" onClick={() => openAuth('signup')}>
            Create account
          </button>
        </div>
      ) : (
        <>
          {ex.requires_login && !dbReady && <div className="sidebar-note small">Your database copy is still being prepared…</div>}
          <div className="exercise-editor">
            <SqlEditor
              docKey={ex.id}
              value={sql}
              onChange={setSql}
              onRun={check}
              handleRef={editor}
              placeholder={ex.kind === 'select' ? '-- Write your query here' : '-- Write your statements here'}
              ariaLabel={`Answer for ${ex.title}`}
              compact
            />
          </div>
          <div className="exercise-actions">
            <button className="btn btn-primary" onClick={check} disabled={busy !== null}>
              {busy === 'check' ? <span className="spinner" /> : <Check />} Check <kbd className="hide-sm">{modKey}↵</kbd>
            </button>
            {ex.kind === 'select' && (
              <button className="btn" onClick={runOnly} disabled={busy !== null}>
                {busy === 'run' ? <span className="spinner" /> : <Play />} Run
              </button>
            )}
            <div className="toolbar-spacer" />
            {ex.hint && (
              <button className="btn btn-ghost btn-sm" onClick={() => setShowHint((s) => !s)} aria-expanded={showHint}>
                <Lightbulb /> Hint
              </button>
            )}
            <button className="btn btn-ghost btn-sm" onClick={reveal} disabled={solution !== null}>
              <Eye /> Solution
            </button>
          </div>
        </>
      )}

      <AnimatePresence initial={false}>
        {showHint && ex.hint && (
          <motion.div className="hint-box" initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={spring}>
            <div>
              <Lightbulb /> <Markdown text={ex.hint} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      <AnimatePresence initial={false}>
        {outcome && (
          <motion.div
            key={attempt}
            className={`verdict ${outcome.correct ? 'ok' : 'bad'}`}
            role="status"
            initial={{ opacity: 0, y: 8 }}
            animate={outcome.correct || reduce ? { opacity: 1, y: 0 } : { opacity: 1, y: 0, x: [0, -7, 6, -4, 3, 0] }}
            exit={{ opacity: 0 }}
            transition={outcome.correct || reduce ? spring : { ...spring, x: { duration: 0.42, ease: 'easeOut' } }}
          >
            <motion.span
              className="verdict-icon"
              initial={outcome.correct && !reduce ? { scale: 0.4 } : false}
              animate={{ scale: 1 }}
              transition={springMomentum}
            >
              {outcome.correct ? <Sparkles /> : <CircleAlert />}
            </motion.span>
            <div className="verdict-main">
              <div className="verdict-title">{outcome.error ? outcome.error.message : outcome.message}</div>
              {outcome.error?.hint && <p>{outcome.error.hint}</p>}
              {outcome.detail && <p>{outcome.detail}</p>}
              {outcome.correct && (
                <div className="verdict-actions">
                  {onNext ? (
                    <button className="btn btn-sm btn-primary" onClick={onNext}>
                      Next exercise <ArrowRight />
                    </button>
                  ) : nextLesson ? (
                    <Link className="btn btn-sm btn-primary" to={`/learn/${nextLesson.slug}`}>
                      Next lesson: {nextLesson.title} <ArrowRight />
                    </Link>
                  ) : (
                    <span className="muted">You finished the whole course. Impressive work!</span>
                  )}
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {outcome?.result && !outcome.correct && outcome.result.columns.length > 0 && (
        <div className="your-result">
          <div className="eyebrow">Your result · {outcome.result.row_count} rows</div>
          <DataGrid columns={outcome.result.columns} rows={outcome.result.rows} dense maxHeight={180} />
        </div>
      )}

      {runResult && (
        <div className="your-result">
          {runResult.error ? (
            <div className="verdict bad">
              <span className="verdict-icon">
                <CircleAlert />
              </span>
              <div className="verdict-main">
                <div className="verdict-title">{runResult.error.message}</div>
                {runResult.error.hint && <p>{runResult.error.hint}</p>}
              </div>
            </div>
          ) : runResult.columns ? (
            <>
              <div className="eyebrow">Output · {runResult.row_count} rows</div>
              <DataGrid columns={runResult.columns} rows={runResult.rows ?? []} dense maxHeight={200} />
            </>
          ) : (
            <div className="muted small">{runResult.message}</div>
          )}
        </div>
      )}

      <AnimatePresence initial={false}>
        {solution !== null && (
          <motion.div className="solution" initial={{ height: 0, opacity: 0 }} animate={{ height: 'auto', opacity: 1 }} exit={{ height: 0, opacity: 0 }} transition={spring}>
            <div className="solution-head">
              <span className="eyebrow">One possible solution</span>
              <button className="btn btn-sm" onClick={() => setSql(solution)}>
                Use it
              </button>
            </div>
            <pre>
              <HighlightedSQL code={solution} />
            </pre>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
