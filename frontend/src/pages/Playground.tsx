import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AnimatePresence, motion, useMotionValue } from 'motion/react'
import { toast } from 'sonner'
import { Bookmark, Database, Lock, PanelLeft, Play, Plus, Share2, Square, X } from 'lucide-react'
import type { SQLNamespace } from '@codemirror/lang-sql'
import { api, ApiError, newRunId, type QueryResponse, type SchemaInfo, type Target } from '../lib/api'
import { useAuth, useEffectiveTarget } from '../state/auth'
import { addLocalHistory, useTabs, useUI } from '../state/stores'
import { copyText, modKey, spring, storage } from '../lib/util'
import { useIsMobile } from '../lib/hooks'
import { SqlEditor, type EditorHandle } from '../components/SqlEditor'
import { ResultsPanel } from '../components/ResultsPanel'
import { Sidebar } from '../components/Sidebar'
import { Splitter } from '../components/Splitter'
import { Segmented } from '../components/Segmented'
import { Dialog, Popover } from '../components/Overlay'
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
  const sidebarW = useMotionValue(storage.get('sqlpg.sidebarW', 288))
  const editorH = useMotionValue(storage.get('sqlpg.editorH', 300))
  const mainRef = useRef<HTMLDivElement>(null)
  const [sidebarSheet, setSidebarSheet] = useState(false)
  const [renaming, setRenaming] = useState<string | null>(null)

  useEffect(() => {
    const fit = () => {
      const h = mainRef.current?.clientHeight ?? 600
      const max = Math.max(160, h - 170)
      if (editorH.get() > max) editorH.set(max)
    }
    fit()
    window.addEventListener('resize', fit)
    return () => window.removeEventListener('resize', fit)
  }, [editorH])

  const insert = (text: string) => {
    editor.current?.insert(text)
    if (mobile) setSidebarSheet(false)
  }
  const preview = (table: string) => {
    const sql = `SELECT *\nFROM ${table}\nLIMIT 100;`
    const id = openOrFocus(sql, table)
    execute(id, sql)
    if (mobile) setSidebarSheet(false)
  }
  const openSql = (sql: string, title: string, savedId?: number) => {
    openOrFocus(sql, title, savedId)
    if (mobile) setSidebarSheet(false)
  }

  const current = run?.response?.results[run.active]
  const errorLine = current?.error?.line ? current.error.line + (run?.lineOffset ?? 0) : null
  const inTx = txOpen[target]

  const sidebar = (
    <Sidebar target={target} schema={schema} schemaError={schemaError} onInsert={insert} onPreview={preview} onOpen={openSql} />
  )

  return (
    <div className="playground">
      {!mobile && (
        <>
          <motion.div className="playground-side" style={{ width: sidebarW }}>
            {sidebar}
          </motion.div>
          <Splitter axis="x" size={sidebarW} min={220} max={() => Math.min(520, window.innerWidth * 0.45)} storageKey="sqlpg.sidebarW" label="Resize sidebar" />
        </>
      )}
      <div className="playground-main" ref={mainRef}>
        <DatabaseBanner />
        <div className="tabs-bar">
          {mobile && (
            <button className="icon-btn" onClick={() => setSidebarSheet(true)} aria-label="Show tables">
              <PanelLeft />
            </button>
          )}
          <div className="tabs-scroll" role="tablist" aria-label="Query tabs">
            {tabs.map((t) => (
              <div key={t.id} className={`qtab ${t.id === tab.id ? 'active' : ''}`} role="tab" aria-selected={t.id === tab.id}>
                {t.id === tab.id && <motion.span layoutId="qtab-bg" className="qtab-bg" transition={{ ...spring, duration: 0.25 }} />}
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
              </div>
            ))}
          </div>
          <button className="icon-btn" onClick={() => newTab()} aria-label="New query tab" title="New tab">
            <Plus />
          </button>
        </div>

        <div className="toolbar">
          {run?.running ? (
            <button className="btn btn-run running" onClick={cancel}>
              <Square className="fill" /> Cancel
            </button>
          ) : (
            <button className="btn btn-primary btn-run" onClick={runActive} title={`Run (${modKey} + Enter)`}>
              <Play className="fill" /> Run <kbd className="hide-sm">{modKey}↵</kbd>
            </button>
          )}
          {user ? (
            <Segmented
              label="Database to query"
              value={target}
              onChange={(v) => setTarget(v)}
              options={[
                { value: 'mine', label: 'My database', disabled: user.db_status !== 'ready', title: 'Your private copy: read and write' },
                { value: 'core', label: 'Original', title: 'The pristine data: read-only' },
              ]}
            />
          ) : (
            <button className="readonly-chip" onClick={() => openAuth('signup', 'Create a free account to INSERT, UPDATE, DELETE and CREATE on your own copy of the data.')}>
              <Lock /> <span className="hide-sm">Original data ·&nbsp;</span>read-only
            </button>
          )}
          <div className="toolbar-spacer" />
          <AnimatePresence>
            {inTx && (
              <motion.div className="tx-pill" initial={{ opacity: 0, scale: 0.9 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.9 }} transition={spring}>
                <span className="tx-dot" /> Transaction open
                <button onClick={() => quick('COMMIT')}>Commit</button>
                <button onClick={() => quick('ROLLBACK')}>Rollback</button>
              </motion.div>
            )}
          </AnimatePresence>
          <button ref={saveBtn} className="btn btn-sm btn-ghost" onClick={save} title={`Save (${modKey} + S)`}>
            <Bookmark /> <span className="hide-sm">Save</span>
          </button>
          <button className="btn btn-sm btn-ghost" onClick={share} title="Copy a share link">
            <Share2 /> <span className="hide-sm">Share</span>
          </button>
        </div>

        <motion.div className="editor-pane" style={{ height: editorH }}>
          <SqlEditor
            docKey={tab.id}
            value={tab.sql}
            onChange={(v) => updateSql(tab.id, v)}
            onRun={runActive}
            onSave={save}
            schema={namespace}
            errorLine={errorLine}
            handleRef={editor}
            placeholder="Write SQL here, then press Ctrl/⌘ + Enter"
          />
        </motion.div>
        <Splitter axis="y" size={editorH} min={110} max={() => (mainRef.current?.clientHeight ?? 700) - 170} storageKey="sqlpg.editorH" label="Resize editor" />
        <div className="results-pane">
          <ResultsPanel
            run={run}
            maxRows={config?.max_rows ?? 2000}
            onSelect={(i) => setRun(tab.id, { active: i })}
            onJumpToLine={(l) => editor.current?.goToLine(l + (run?.lineOffset ?? 0))}
            onSignup={() => openAuth('signup', 'Create a free account to get your own copy of the database. Every change stays private to you.')}
            onUseMine={user?.db_status === 'ready' ? () => setTarget('mine') : undefined}
            onStarter={(sql, title) => {
              const id = openOrFocus(sql, title)
              execute(id, sql)
            }}
          />
        </div>
      </div>

      <Popover open={saveOpen} onClose={() => setSaveOpen(false)} anchor={saveBtn} width={300}>
        <form className="save-form" onSubmit={confirmSave}>
          <label className="eyebrow" htmlFor="save-title">
            Save query as
          </label>
          <input id="save-title" className="input input-sm" autoFocus value={saveTitle} onChange={(e) => setSaveTitle(e.target.value)} placeholder="e.g. Monthly revenue" maxLength={120} />
          <div className="save-actions">
            <button type="button" className="btn btn-ghost btn-sm" onClick={() => setSaveOpen(false)}>
              Cancel
            </button>
            <button className="btn btn-primary btn-sm" disabled={!saveTitle.trim()}>
              Save
            </button>
          </div>
        </form>
      </Popover>

      {mobile && (
        <Dialog open={sidebarSheet} onClose={() => setSidebarSheet(false)} title={<span className="sheet-title"><Database size={16} /> Explorer</span>}>
          <div className="sheet-sidebar">{sidebar}</div>
        </Dialog>
      )}
    </div>
  )
}
