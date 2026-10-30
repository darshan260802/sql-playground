import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AnimatePresence, motion, useMotionValue } from 'motion/react'
import { toast } from 'sonner'
import { Bookmark, Lock, MoreHorizontal, PanelLeft, Play, Plus, Share2, Square, X } from 'lucide-react'
import type { SQLNamespace } from '@codemirror/lang-sql'
import { api, ApiError, newRunId, type QueryResponse, type SchemaInfo, type Target } from '../lib/api'
import { useAuth, useEffectiveTarget } from '../state/auth'
import { addLocalHistory, useTabs, useUI } from '../state/stores'
import { copyText, formatBytes, formatInt, modKey, spring, springSnappy, storage } from '../lib/util'
import { useIsMobile } from '../lib/hooks'
import { SqlEditor, type EditorHandle } from '../components/SqlEditor'
import { ResultsPanel } from '../components/ResultsPanel'
import { Sidebar } from '../components/Sidebar'
import { Splitter } from '../components/Splitter'
import { Segmented } from '../components/Segmented'
import { Popover } from '../components/Overlay'
import { DatabaseBanner } from '../components/DatabaseBanner'

const WRITE_RE = /^\s*(INSERT|UPDATE|DELETE|REPLACE|CREATE|ALTER|DROP|TRUNCATE|RENAME|CALL|LOAD)\b/i
const DDL_RE = /^\s*(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b/i

function pickActive(resp: QueryResponse) {
  const err = resp.results.findIndex((r) => r.error)
  if (err >= 0) return err
  for (let i = resp.results.length - 1; i >= 0; i--) if (resp.results[i].columns) return i
  return Math.max(0, resp.results.length - 1)
}

export function Playground() {
  const { user, config, refresh } = useAuth()
  const { tabs, activeId, target: preferred, runs, setActive, newTab, closeTab, updateSql, updateTab, setTarget, setRun, openOrFocus } = useTabs()
  const target = useEffectiveTarget(preferred)
  const { schemaVersion, bumpSchema, bumpHistory, bumpSaved, openAuth } = useUI()
  const tab = tabs.find((t) => t.id === activeId) ?? tabs[0]
  const run = runs[tab.id]
  const editor = useRef<EditorHandle | null>(null)
  const mobile = useIsMobile()

  // ---------------------------------------------------------------- schema
  const [schema, setSchema] = useState<SchemaInfo | null>(null)
  const [schemaError, setSchemaError] = useState<string>()
  useEffect(() => {
    let cancelled = false
    setSchemaError(undefined)
    api
      .schema(target)
      .then((s) => !cancelled && setSchema(s))
      .catch((e) => !cancelled && setSchemaError((e as Error).message))
    return () => {
      cancelled = true
    }
  }, [target, schemaVersion, user?.id])

  const namespace = useMemo<SQLNamespace | undefined>(() => {
    if (!schema) return undefined
    const ns: Record<string, { label: string; type: string; detail: string }[]> = {}
    for (const t of schema.tables) ns[t.name] = t.columns.map((c) => ({ label: c.name, type: 'property', detail: c.type }))
    return ns
  }, [schema])

  // ---------------------------------------------------------------- running
  const [txOpen, setTxOpen] = useState<Record<Target, boolean>>({ mine: false, core: false })

  const execute = useCallback(
    async (tabId: string, sql: string, lineOffset = 0) => {
      if (!sql.trim()) {
        toast('Nothing to run', { description: 'Write a query first.' })
        return
      }
      if (useTabs.getState().runs[tabId]?.running) return
      const runId = newRunId()
      setRun(tabId, { running: true, runId, error: undefined, errorCode: undefined, lineOffset })
      const startedTarget = target
      try {
        const resp = await api.query(sql, startedTarget, runId)
        setRun(tabId, { running: false, response: resp, active: pickActive(resp) })
        if (resp.session) setTxOpen((s) => ({ ...s, [startedTarget]: resp.session!.in_transaction }))
        const ok = !resp.results.some((r) => r.error)
        if (!user) {
          const last = resp.results[resp.results.length - 1]
          addLocalHistory({ sql, target: startedTarget, ok, duration_ms: Math.round(resp.duration_ms), row_count: last?.columns ? last.row_count : undefined, error: resp.results.find((r) => r.error)?.error?.message })
        }
        bumpHistory()
        if (resp.results.some((r) => !r.error && DDL_RE.test(r.sql))) bumpSchema()
        if (startedTarget === 'mine' && resp.results.some((r) => !r.error && WRITE_RE.test(r.sql))) refresh()
      } catch (e) {
        const err = e as ApiError
        setRun(tabId, { running: false, response: undefined, error: err.message, errorCode: err.code })
      }
    },
    [target, user, setRun, bumpHistory, bumpSchema, refresh],
  )

  const runActive = useCallback(() => {
    const r = editor.current?.getRunnable() ?? { sql: tab.sql, lineOffset: 0 }
    execute(tab.id, r.sql, r.lineOffset)
  }, [execute, tab])

  const cancel = () => {
    if (run?.runId) api.cancel(run.runId).catch(() => {})
  }

  const quick = (sql: string) => execute(tab.id, sql)

  // ---------------------------------------------------------------- save / share
  const [saveOpen, setSaveOpen] = useState(false)
  const saveBtn = useRef<HTMLButtonElement>(null)
  const [saveTitle, setSaveTitle] = useState('')
  const save = useCallback(async () => {
    if (!user) {
      openAuth('signup', 'Create a free account to save queries and open them from anywhere.')
      return
    }
    if (tab.savedId) {
      try {
        await api.updateSaved(tab.savedId, tab.title, tab.sql)
        toast.success('Saved', { description: tab.title, duration: 1800 })
        bumpSaved()
      } catch (e) {
        if ((e as ApiError).status === 404) {
          updateTab(tab.id, { savedId: undefined })
          setSaveTitle(tab.title)
          setSaveOpen(true)
        } else toast.error((e as Error).message)
      }
      return
    }
    setSaveTitle(tab.title.startsWith('Query ') || tab.title === 'Welcome' ? '' : tab.title)
    setSaveOpen(true)
  }, [user, tab, openAuth, bumpSaved, updateTab])

  const confirmSave = async (e: React.FormEvent) => {
    e.preventDefault()
    const title = saveTitle.trim()
    if (!title) return
    try {
      const { id } = await api.createSaved(title, tab.sql)
      updateTab(tab.id, { savedId: id, title })
      setSaveOpen(false)
      toast.success('Query saved', { description: title })
      bumpSaved()
    } catch (err) {
      toast.error((err as Error).message)
    }
  }

  const share = async () => {
    if (!tab.sql.trim()) return
    try {
      const { id } = await api.share(tab.title, tab.sql)
      const url = `${window.location.origin}/q/${id}`
      const copied = await copyText(url)
      toast.success(copied ? 'Link copied to clipboard' : 'Share link created', { description: url })
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  // Shortcuts work anywhere on the page, not only while the editor has focus.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!(e.metaKey || e.ctrlKey) || e.altKey) return
      const inEditor = (e.target as HTMLElement | null)?.closest?.('.cm-editor')
      if (inEditor) return // the editor's own keymap handles these
      if (e.key === 's' || e.key === 'S') {
        e.preventDefault()
        save()
      } else if (e.key === 'Enter') {
        const el = e.target as HTMLElement | null
        if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) return
        e.preventDefault()
        runActive()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [save, runActive])

  // ---------------------------------------------------------------- layout
  const sidebarW = useMotionValue(storage.get('sqlpg.sidebarW', 280))
  const editorH = useMotionValue(storage.get('sqlpg.editorH', 300))
  const mainRef = useRef<HTMLDivElement>(null)
  // The tables panel remembers its state on wide screens; on narrower ones it starts closed.
  const wide = window.innerWidth >= 1100
  const [explorerOpen, setExplorerOpen] = useState(() => wide && storage.get('sqlpg.explorer', true))
  const [pane, setPane] = useState<'query' | 'results' | 'tables'>('query')
  const [renaming, setRenaming] = useState<string | null>(null)
  const [moreOpen, setMoreOpen] = useState(false)
  const moreBtn = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (window.innerWidth >= 1100) storage.set('sqlpg.explorer', explorerOpen)
  }, [explorerOpen])

  useEffect(() => {
    if (mobile) return
    const fit = () => {
      const h = mainRef.current?.clientHeight ?? 600
      const max = Math.max(160, h - 170)
      if (editorH.get() > max) editorH.set(max)
    }
    fit()
    window.addEventListener('resize', fit)
    return () => window.removeEventListener('resize', fit)
  }, [editorH, mobile])

  // On a phone one region shows at a time: a run shows its results.
  const runAndShow = (fn: () => void) => {
    fn()
    if (mobile) setPane('results')
  }

  const insert = (text: string) => {
    editor.current?.insert(text)
    if (mobile) setPane('query')
  }
  const preview = (table: string) => {
    const sql = `SELECT *\nFROM ${table}\nLIMIT 100;`
    const id = openOrFocus(sql, table)
    runAndShow(() => execute(id, sql))
  }
  const openSql = (sql: string, title: string, savedId?: number) => {
    openOrFocus(sql, title, savedId)
    if (mobile) setPane('query')
  }

  const current = run?.response?.results[run.active]
  const errorLine = current?.error?.line ? current.error.line + (run?.lineOffset ?? 0) : null
  const inTx = txOpen[target]
  const resultCount = run?.response ? (current?.columns ? formatInt(current.row_count) : run.response.results.some((r) => r.error) ? 'error' : 'OK') : undefined

  const sidebar = (
    <Sidebar target={target} schema={schema} schemaError={schemaError} onInsert={insert} onPreview={preview} onOpen={openSql} />
  )

  const runButton = run?.running ? (
    <button className="btn btn-run running" onClick={cancel}>
      <Square className="fill" /> Cancel
    </button>
  ) : (
    <button className="btn btn-primary btn-run" onClick={() => runAndShow(runActive)} title={`Run (${modKey} + Enter)`}>
      <Play className="fill" /> Run {!mobile && <kbd>{modKey}↵</kbd>}
    </button>
  )

  const txPill = (
    <AnimatePresence>
      {inTx && (
        <motion.div className="tx-pill" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} transition={spring}>
          <span className="tx-dot" /> Transaction open
          <button onClick={() => quick('COMMIT')}>Commit</button>
          <button onClick={() => quick('ROLLBACK')}>Rollback</button>
        </motion.div>
      )}
    </AnimatePresence>
  )

  const queryTabs = (
    <div className="qtabs">
      {!mobile && (
        <button className={`icon-btn sm ${explorerOpen ? 'on' : ''}`} onClick={() => setExplorerOpen((o) => !o)} aria-label={explorerOpen ? 'Hide tables' : 'Show tables'} aria-pressed={explorerOpen} title={explorerOpen ? 'Hide tables' : 'Show tables'}>
          <PanelLeft />
        </button>
      )}
      <div className="qtabs-scroll" role="tablist" aria-label="Query tabs">
        {tabs.map((t) => (
          <div key={t.id} className={`qtab ${t.id === tab.id ? 'active' : ''}`} role="tab" aria-selected={t.id === tab.id}>
            {renaming === t.id ? (
              <input
                className="qtab-input"
                defaultValue={t.title}
                autoFocus
                onFocus={(e) => e.currentTarget.select()}
                onBlur={(e) => {
                  if (e.target.value.trim()) updateTab(t.id, { title: e.target.value.trim().slice(0, 60) })
                  setRenaming(null)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') e.currentTarget.blur()
                  if (e.key === 'Escape') setRenaming(null)
                }}
                aria-label="Tab name"
              />
            ) : (
              <button className="qtab-label" onClick={() => setActive(t.id)} onDoubleClick={() => setRenaming(t.id)} title="Double-click to rename">
                {runs[t.id]?.running && <span className="spinner tiny" />}
                {t.savedId && <Bookmark className="qtab-saved" />}
                <span>{t.title}</span>
              </button>
            )}
            <button className="qtab-close" aria-label={`Close ${t.title}`} onClick={() => closeTab(t.id)}>
              <X />
            </button>
            {t.id === tab.id && <motion.span layoutId="qtab-line" className="tab-line" transition={springSnappy} />}
          </div>
        ))}
      </div>
      <button className="icon-btn sm" onClick={() => newTab()} aria-label="New query tab" title="New tab">
        <Plus />
      </button>
    </div>
  )

  const editorEl = (
    <SqlEditor
      docKey={tab.id}
      value={tab.sql}
      onChange={(v) => updateSql(tab.id, v)}
      onRun={() => runAndShow(runActive)}
      onSave={save}
      schema={namespace}
      errorLine={errorLine}
      handleRef={editor}
      placeholder="Write SQL here, then press Ctrl/⌘ + Enter"
    />
  )

  const results = (
    <ResultsPanel
      run={run}
      maxRows={config?.max_rows ?? 2000}
      onSelect={(i) => setRun(tab.id, { active: i })}
      onJumpToLine={(l) => {
        if (mobile) setPane('query')
        editor.current?.goToLine(l + (run?.lineOffset ?? 0))
      }}
      onSignup={() => openAuth('signup', 'Create a free account to get your own copy of the database. Every change stays private to you.')}
      onUseMine={user?.db_status === 'ready' ? () => setTarget('mine') : undefined}
      onStarter={(sql, title) => {
        const id = openOrFocus(sql, title)
        runAndShow(() => execute(id, sql))
      }}
    />
  )

  const scope = (
    <ScopeSwitch
      target={target}
      onChange={setTarget}
      compact={mobile}
      onSignup={() => openAuth('signup', 'Create a free account to INSERT, UPDATE, DELETE and CREATE on your own copy of the data.')}
    />
  )

  const saveForm = (
    <Popover open={saveOpen} onClose={() => setSaveOpen(false)} anchor={mobile ? moreBtn : saveBtn} width={300}>
      <form className="save-form" onSubmit={confirmSave}>
        <label htmlFor="save-title">Save query as</label>
        <input id="save-title" className="input input-sm" autoFocus value={saveTitle} onChange={(e) => setSaveTitle(e.target.value)} placeholder="e.g. Monthly revenue" maxLength={120} />
        <div className="save-actions">
          <button type="button" className="btn btn-text btn-sm" onClick={() => setSaveOpen(false)}>
            Cancel
          </button>
          <button className="btn btn-sm" disabled={!saveTitle.trim()}>
            Save
          </button>
        </div>
      </form>
    </Popover>
  )

  if (mobile) {
    return (
      <div className="playground phone">
        <header className="title-row">
          <h1>Playground</h1>
          {scope}
          <div className="spacer" />
          <button ref={moreBtn} className="icon-btn" onClick={() => setMoreOpen(true)} aria-label="Save or share this query">
            <MoreHorizontal />
          </button>
        </header>
        <DatabaseBanner />
        <Segmented
          ruled
          full
          className="pane-switch"
          label="Show"
          value={pane}
          onChange={setPane}
          options={[
            { value: 'query', label: 'Query' },
            { value: 'results', label: <>Results {resultCount && <span className="count">{resultCount}</span>}</> },
            { value: 'tables', label: 'Tables' },
          ]}
        />
        <div className="phone-pane">
          {pane === 'query' && (
            <div className="phone-query">
              {queryTabs}
              <div className="editor-pane">{editorEl}</div>
            </div>
          )}
          {pane === 'results' && <div className="results-pane">{results}</div>}
          {pane === 'tables' && sidebar}
        </div>
        <div className="run-bar">
          {txPill}
          {runButton}
        </div>
        <Popover open={moreOpen} onClose={() => setMoreOpen(false)} anchor={moreBtn} width={220}>
          <div className="menu-list">
            <button className="menu-item" onClick={() => { setMoreOpen(false); save() }}>
              <Bookmark /> Save query
            </button>
            <button className="menu-item" onClick={() => { setMoreOpen(false); share() }}>
              <Share2 /> Copy share link
            </button>
            <button className="menu-item" onClick={() => { setMoreOpen(false); newTab(); setPane('query') }}>
              <Plus /> New query tab
            </button>
          </div>
        </Popover>
        {saveForm}
      </div>
    )
  }

  return (
    <div className="playground">
      <header className="title-row">
        <h1>Playground</h1>
        {scope}
        <div className="spacer" />
        {txPill}
        <button ref={saveBtn} className="btn btn-text" onClick={save} title={`Save (${modKey} + S)`} aria-label="Save">
          <Bookmark /> <span className="label">Save</span>
        </button>
        <button className="btn btn-text" onClick={share} title="Copy a share link" aria-label="Share">
          <Share2 /> <span className="label">Share</span>
        </button>
        {runButton}
      </header>
      <DatabaseBanner />
      <div className="workbench">
        {explorerOpen && (
          <>
            <motion.div className="playground-side" style={{ width: sidebarW }}>
              {sidebar}
            </motion.div>
            <Splitter axis="x" size={sidebarW} min={220} max={() => Math.min(520, window.innerWidth * 0.4)} storageKey="sqlpg.sidebarW" label="Resize tables panel" />
          </>
        )}
        <div className="playground-main" ref={mainRef}>
          {queryTabs}
          <motion.div className="editor-pane" style={{ height: editorH }}>
            {editorEl}
          </motion.div>
          <Splitter axis="y" size={editorH} min={110} max={() => (mainRef.current?.clientHeight ?? 700) - 170} storageKey="sqlpg.editorH" label="Resize editor" />
          <div className="results-pane">{results}</div>
        </div>
      </div>
      {saveForm}
    </div>
  )
}

/**
 * Which database a run goes to: the learner's own copy (read and write) or the original data
 * (read-only). The question learners most often get wrong, so it sits next to the title.
 */
function ScopeSwitch({ target, onChange, onSignup, compact }: { target: Target; onChange: (t: Target) => void; onSignup: () => void; compact?: boolean }) {
  const { user, database } = useAuth()
  if (!user) {
    return (
      <div className="scope">
        <span className="scope-anon">
          <Lock /> {compact ? 'Read-only' : <><span className="wide-only">Original data </span><span className="mono faint">core</span> · read-only</>}
        </span>
        {!compact && (
          <button className="btn btn-sm" onClick={onSignup}>
            Get a writable copy
          </button>
        )}
      </div>
    )
  }
  const ready = user.db_status === 'ready'
  const usage = database && database.quota_bytes > 0 ? `${formatBytes(database.usage_bytes)} of ${formatBytes(database.quota_bytes)}` : ''
  return (
    <div className="scope">
      <Segmented
        label="Database to run on"
        value={target}
        onChange={onChange}
        options={[
          { value: 'mine', label: compact ? 'Mine' : <>Your copy <span className="mono faint">{user.db_status === 'ready' ? database?.schema ?? '' : '…'}</span></>, disabled: !ready, title: 'Your private copy: read and write' },
          { value: 'core', label: compact ? 'Original' : <>Original <span className="mono faint">core</span></>, title: 'The original data: read-only' },
        ]}
      />
      {!compact && <span className="scope-note num">{target === 'mine' ? (ready ? usage : 'preparing…') : 'read-only'}</span>}
    </div>
  )
}
