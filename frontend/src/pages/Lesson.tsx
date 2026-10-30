import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { toast } from 'sonner'
import { ArrowLeft, ArrowRight, Check, ChevronDown, CircleAlert, Eye, Lightbulb, Lock, Play, Sparkles } from 'lucide-react'
import { api, ApiError, newRunId, type CheckOutcome, type Exercise, type Lesson as LessonT, type StatementResult } from '../lib/api'
import { useAuth } from '../state/auth'
import { addLocalSolved, localSolved, useTabs, useUI } from '../state/stores'
import { modKey, spring, springMomentum, springSnappy, storage } from '../lib/util'
import { useIsCompact } from '../lib/hooks'
import { Segmented } from '../components/Segmented'
import { Markdown, HighlightedSQL } from '../components/Markdown'
import { SqlEditor, type EditorHandle } from '../components/SqlEditor'
import { DataGrid } from '../components/DataGrid'

const KIND_LABEL: Record<Exercise['kind'], string> = { select: 'Query', dml: 'Changes data', ddl: 'Schema design' }

function scrollMainToTop() {
  document.querySelector('.app-main')?.scrollTo({ top: 0 })
}

export function Lesson() {
  const { slug = '' } = useParams()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [lesson, setLesson] = useState<LessonT | null>(null)
  const [error, setError] = useState('')
  const [idx, setIdx] = useState(0)
  const [solved, setSolved] = useState<Set<string>>(new Set())
  const mobile = useIsCompact()
  const [view, setView] = useState<'read' | 'practice'>('read')

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
    setView('read')
    scrollMainToTop()
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
      <div className="page-empty">
        <h1>{error}</h1>
        <Link to="/learn" className="btn">
          <ArrowLeft /> All lessons
        </Link>
      </div>
    )
  }
  if (!lesson) {
    return (
      <div className="lesson-page" aria-busy="true">
        <div className="lesson-head">
          <div className="skeleton" style={{ height: 13, width: 140, marginBottom: 14 }} />
          <div className="skeleton" style={{ height: 28, width: '50%', marginBottom: 12 }} />
          <div className="skeleton" style={{ height: 13, width: 220 }} />
        </div>
        <div className="lesson-article">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="skeleton" style={{ height: 15, width: `${70 + ((i * 13) % 30)}%`, marginBottom: 14 }} />
          ))}
        </div>
      </div>
    )
  }

  const ex = lesson.exercises[idx]
  const nSolved = lesson.exercises.filter((e) => solved.has(e.id)).length
  const allDone = nSolved === lesson.exercises.length
  const showRead = !mobile || view === 'read'
  const showPractice = !mobile || view === 'practice'
  const switchView = (v: 'read' | 'practice') => {
    setView(v)
    scrollMainToTop()
  }

  return (
    <div className="lesson-page">
      <header className="lesson-head">
        <nav className="crumbs" aria-label="Breadcrumb">
          <Link to="/learn">Learn</Link>
          <span aria-hidden="true">/</span>
          <span>{lesson.module}</span>
        </nav>
        <h1>{lesson.title}</h1>
        <p className="lesson-meta tnum">
          {lesson.level} · {lesson.minutes} min · {nSolved} of {lesson.exercises.length} exercises solved
        </p>
      </header>

      {mobile && (
        <Segmented
          ruled
          full
          className="lesson-switch"
          label="Show"
          value={view}
          onChange={switchView}
          options={[
            { value: 'read', label: 'Lesson' },
            { value: 'practice', label: <>Exercises <span className="count">{nSolved}/{lesson.exercises.length}</span></> },
          ]}
        />
      )}

      {showRead && (
        <article className="lesson-article">
          <p className="lesson-summary">{lesson.summary}</p>
          <Markdown text={lesson.body} onOpenSql={openInPlayground} className="lesson-body" />
          {mobile && (
            <button className="btn btn-primary btn-lg btn-block practice-cta" onClick={() => switchView('practice')}>
              {allDone ? 'Review the exercises' : `Practice: ${lesson.exercises.length - nSolved} exercises to go`} <ArrowRight />
            </button>
          )}
          <nav className="lesson-nav" aria-label="Other lessons">
            {lesson.prev ? (
              <Link to={`/learn/${lesson.prev.slug}`} className="lesson-nav-link">
                <span className="faint small">Previous lesson</span>
                <span>
                  <ArrowLeft size={15} /> {lesson.prev.title}
                </span>
              </Link>
            ) : (
              <span />
            )}
            {lesson.next && (
              <Link to={`/learn/${lesson.next.slug}`} className="lesson-nav-link next">
                <span className="faint small">Next lesson</span>
                <span>
                  {lesson.next.title} <ArrowRight size={15} />
                </span>
              </Link>
            )}
          </nav>
        </article>
      )}

      {showPractice && (
      <aside className="exercise-panel" aria-label="Exercises">
        <div className="exercise-sticky">
          <div className="exercise-tabs">
            <div className="tabs ruled" role="tablist" aria-label="Exercises in this lesson">
              {lesson.exercises.map((e, i) => (
                <button key={e.id} role="tab" aria-selected={i === idx} className={solved.has(e.id) ? 'done' : ''} onClick={() => setIdx(i)} title={e.title} aria-label={`Exercise ${i + 1}${solved.has(e.id) ? ', solved' : ''}`}>
                  {solved.has(e.id) ? <Check className="step-check" /> : <span className="num">{i + 1}</span>}
                  {i === idx && <motion.span layoutId="ex-line" className="tab-line" transition={springSnappy} />}
                </button>
              ))}
            </div>
            <span className="exercise-count faint small tnum">{allDone ? 'All solved' : `Exercise ${idx + 1} of ${lesson.exercises.length}`}</span>
          </div>
          <AnimatePresence mode="wait" initial={false}>
            <motion.div key={ex.id} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} transition={{ duration: 0.12 }}>
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
      )}
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
    <div className="exercise">
      <div className="exercise-head">
        <h2>{ex.title}</h2>
        {solved && (
          <span className="pill pill-positive">
            <Check /> Solved
          </span>
        )}
      </div>
      <p className="exercise-kind faint small">
        {KIND_LABEL[ex.kind]}
        {ex.ordered ? ' · row order matters' : ''}
        {ex.requires_login ? ' · runs on your copy' : ''}
      </p>
      <Markdown text={ex.prompt} className="exercise-prompt" />

      {ex.expected && (
        <div className="expected">
          <button className="expected-toggle" onClick={() => setShowExpected((s) => !s)} aria-expanded={showExpected}>
            <motion.span animate={{ rotate: showExpected ? 0 : -90 }} transition={spring} className="chev">
              <ChevronDown size={15} />
            </motion.span>
            Expected result
            <span className="faint tnum">
              {ex.expected.row_count} {ex.expected.row_count === 1 ? 'row' : 'rows'}
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
          <button className="btn btn-sm" onClick={() => openAuth('signup')}>
            Create account
          </button>
        </div>
      ) : (
        <>
          {ex.requires_login && !dbReady && <p className="faint small">Your database copy is still being prepared…</p>}
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
              {busy === 'check' ? <span className="spinner" /> : <Check />} Check <kbd className="hide-touch">{modKey}↵</kbd>
            </button>
            {ex.kind === 'select' && (
              <button className="btn" onClick={runOnly} disabled={busy !== null}>
                {busy === 'run' ? <span className="spinner" /> : <Play />} Run
              </button>
            )}
            <div className="spacer" />
            {ex.hint && (
              <button className="btn btn-text" onClick={() => setShowHint((s) => !s)} aria-expanded={showHint}>
                <Lightbulb /> Hint
              </button>
            )}
            <button className="btn btn-text" onClick={reveal} disabled={solution !== null}>
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
          <h3>
            Your result <span className="faint tnum">{outcome.result.row_count} rows</span>
          </h3>
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
              <h3>
                Output <span className="faint tnum">{runResult.row_count} rows</span>
              </h3>
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
              <h3>One possible solution</h3>
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
