import { useEffect, useMemo, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { toast } from 'sonner'
import {
  Bookmark,
  ChevronRight,
  Eye,
  FunctionSquare,
  History,
  KeyRound,
  Link2,
  Pencil,
  Play,
  Plus,
  Search,
  Table2,
  Trash2,
  Zap,
} from 'lucide-react'
import { api, type HistoryEntry, type SavedQuery, type SchemaInfo, type TableInfo, type Target } from '../lib/api'
import { useAuth } from '../state/auth'
import { clearLocalHistory, localHistory, useUI } from '../state/stores'
import { firstLine, formatCompact, formatDuration, formatInt, spring, timeAgo } from '../lib/util'
import { Segmented } from './Segmented'

type Pane = 'schema' | 'saved' | 'history'

interface Props {
  target: Target
  schema: SchemaInfo | null
  schemaError?: string
  onInsert: (text: string) => void
  onPreview: (table: string) => void
  onOpen: (sql: string, title: string, savedId?: number) => void
}

export function Sidebar({ target, schema, schemaError, onInsert, onPreview, onOpen }: Props) {
  const [pane, setPane] = useState<Pane>('schema')
  return (
    <aside className="sidebar" aria-label="Database explorer">
      <div className="sidebar-top">
        <Segmented
          full
          label="Sidebar view"
          value={pane}
          onChange={setPane}
          options={[
            { value: 'schema', label: 'Schema' },
            { value: 'saved', label: 'Saved' },
            { value: 'history', label: 'History' },
          ]}
        />
      </div>
      <div className="sidebar-body">
        {pane === 'schema' && <SchemaBrowser target={target} schema={schema} error={schemaError} onInsert={onInsert} onPreview={onPreview} />}
        {pane === 'saved' && <SavedList onOpen={onOpen} />}
        {pane === 'history' && <HistoryList onOpen={onOpen} />}
      </div>
    </aside>
  )
}

// ------------------------------------------------------------------ schema

function SchemaBrowser({ target, schema, error, onInsert, onPreview }: { target: Target; schema: SchemaInfo | null; error?: string; onInsert: (t: string) => void; onPreview: (t: string) => void }) {
  const [q, setQ] = useState('')
  const [open, setOpen] = useState<Record<string, boolean>>({})
  const query = q.trim().toLowerCase()

  const groups = useMemo(() => {
    if (!schema) return []
    const out: { domain: string; tables: TableInfo[] }[] = []
    for (const t of schema.tables) {
      const matches = !query || t.name.includes(query) || t.columns.some((c) => c.name.includes(query))
      if (!matches) continue
      let g = out.find((x) => x.domain === t.domain)
      if (!g) out.push((g = { domain: t.domain, tables: [] }))
      g.tables.push(t)
    }
    return out
  }, [schema, query])

  return (
    <div className="schema">
      <div className="search-box">
        <Search />
        <input className="input input-sm" placeholder="Search tables and columns" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search schema" />
      </div>
      <div className="schema-db">
        <span className={`db-dot ${target === 'mine' ? 'mine' : ''}`} />
        <span className="mono">{schema?.schema ?? '…'}</span>
        <span className="muted">{target === 'mine' ? 'your copy' : 'read-only'}</span>
      </div>
      {error && <div className="sidebar-note error">{error}</div>}
      {!schema && !error && (
        <div className="schema-skeleton">
          {Array.from({ length: 10 }).map((_, i) => (
            <div key={i} className="skeleton" style={{ height: 18, width: `${55 + ((i * 37) % 40)}%` }} />
          ))}
        </div>
      )}
      {groups.map((g) => (
        <div key={g.domain} className="schema-group">
          <div className="eyebrow schema-domain">{g.domain}</div>
          {g.tables.map((t) => {
            const isOpen = open[t.name] || (query.length > 1 && !t.name.includes(query))
            return (
              <div key={t.name} className="schema-table">
                <div className="schema-row">
                  <button className="schema-toggle" onClick={() => setOpen((o) => ({ ...o, [t.name]: !o[t.name] }))} aria-expanded={isOpen} title={t.comment}>
                    <motion.span animate={{ rotate: isOpen ? 90 : 0 }} transition={spring} className="chev">
                      <ChevronRight />
                    </motion.span>
                    {t.type === 'view' ? <Eye className="tbl-icon" /> : <Table2 className="tbl-icon" />}
                    <span className="schema-name">{t.name}</span>
                    <span className="schema-count" title={t.exact ? `${formatInt(t.rows)} rows` : `about ${formatInt(t.rows)} rows (estimate)`}>
                      {formatCompact(t.rows)}
                    </span>
                  </button>
                  <div className="schema-actions">
                    <button className="icon-btn sm" title={`Preview ${t.name}`} aria-label={`Preview ${t.name}`} onClick={() => onPreview(t.name)}>
                      <Play />
                    </button>
                    <button className="icon-btn sm" title="Insert name" aria-label={`Insert ${t.name}`} onClick={() => onInsert(t.name)}>
                      <Plus />
                    </button>
                  </div>
                </div>
                <AnimatePresence initial={false}>
                  {isOpen && (
                    <motion.ul
                      className="schema-cols"
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: 'auto', opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={spring}
                    >
                      {t.comment && <li className="schema-comment">{t.comment}</li>}
                      {t.columns.map((c) => (
                        <li key={c.name}>
                          <button
                            className={`schema-col ${query && c.name.includes(query) ? 'match' : ''}`}
                            onClick={() => onInsert(c.name)}
                            title={[c.comment, c.ref ? `→ ${c.ref}` : '', c.nullable ? 'nullable' : 'not null'].filter(Boolean).join(' · ')}
                          >
                            <span className="col-key">
                              {c.key === 'PRI' ? <KeyRound className="pk" /> : c.ref ? <Link2 className="fk" /> : null}
                            </span>
                            <span className="col-name">{c.name}</span>
                            <span className="col-type">{c.type}</span>
                          </button>
                        </li>
                      ))}
                    </motion.ul>
                  )}
                </AnimatePresence>
              </div>
            )
          })}
        </div>
      ))}
      {schema && (schema.routines.length > 0 || schema.triggers.length > 0) && (
        <div className="schema-group">
          <div className="eyebrow schema-domain">Routines & triggers</div>
          {schema.routines.map((r) => (
            <button key={r.name} className="schema-misc" onClick={() => onInsert(r.type === 'PROCEDURE' ? `CALL ${r.name}()` : `${r.name}()`)}>
              <FunctionSquare /> <span>{r.name}</span> <span className="muted">{r.type.toLowerCase()}</span>
            </button>
          ))}
          {schema.triggers.map((t) => (
            <div key={t.name} className="schema-misc static" title={`${t.event} on ${t.table}`}>
              <Zap /> <span>{t.name}</span> <span className="muted">{t.table}</span>
            </div>
          ))}
        </div>
      )}
      {schema && groups.length === 0 && <div className="sidebar-note">No tables or columns match “{q}”.</div>}
    </div>
  )
}

// ------------------------------------------------------------------ saved queries

function SignInPrompt({ icon, text }: { icon: React.ReactNode; text: string }) {
  const openAuth = useUI((s) => s.openAuth)
  return (
    <div className="sidebar-cta">
      {icon}
      <p>{text}</p>
      <button className="btn btn-primary btn-sm" onClick={() => openAuth('signup')}>
        Create a free account
      </button>
      <button className="btn btn-ghost btn-sm" onClick={() => openAuth('signin')}>
        Sign in
      </button>
    </div>
  )
}

function SavedList({ onOpen }: { onOpen: (sql: string, title: string, savedId?: number) => void }) {
  const { user } = useAuth()
  const version = useUI((s) => s.savedVersion)
  const [items, setItems] = useState<SavedQuery[] | null>(null)
  const [editing, setEditing] = useState<number | null>(null)

  useEffect(() => {
    if (!user) return
    api.saved().then((r) => setItems(r.saved)).catch((e) => toast.error(e.message))
  }, [user, version])

  if (!user) return <SignInPrompt icon={<Bookmark />} text="Save your favourite queries and open them from any device." />
  if (!items) return <div className="sidebar-note">Loading…</div>
  if (items.length === 0)
    return (
      <div className="sidebar-note">
        No saved queries yet. Press <kbd>Ctrl</kbd>/<kbd>⌘</kbd> + <kbd>S</kbd> in the editor to save one.
      </div>
    )

  const rename = async (q: SavedQuery, title: string) => {
    setEditing(null)
    if (!title.trim() || title === q.title) return
    try {
      await api.updateSaved(q.id, title.trim(), q.sql)
      setItems((list) => list?.map((x) => (x.id === q.id ? { ...x, title: title.trim() } : x)) ?? null)
    } catch (e) {
      toast.error((e as Error).message)
    }
  }
  const remove = async (q: SavedQuery) => {
    const prev = items
    setItems(items.filter((x) => x.id !== q.id))
    try {
      await api.deleteSaved(q.id)
      toast('Deleted “' + q.title + '”', {
        action: {
          label: 'Undo',
          onClick: async () => {
            try {
              await api.createSaved(q.title, q.sql)
              useUI.getState().bumpSaved()
            } catch {
              setItems(prev)
            }
          },
        },
      })
    } catch (e) {
      setItems(prev)
      toast.error((e as Error).message)
    }
  }

  return (
    <ul className="list">
      <AnimatePresence initial={false}>
        {items.map((q) => (
          <motion.li key={q.id} layout initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0, height: 0 }} transition={spring}>
            <div className="list-item">
              {editing === q.id ? (
                <input
                  className="input input-sm"
                  defaultValue={q.title}
                  autoFocus
                  onBlur={(e) => rename(q, e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') rename(q, (e.target as HTMLInputElement).value)
                    if (e.key === 'Escape') setEditing(null)
                  }}
                  aria-label="Query name"
                />
              ) : (
                <button className="list-main" onClick={() => onOpen(q.sql, q.title, q.id)}>
                  <span className="list-title">{q.title}</span>
                  <span className="list-sql mono">{firstLine(q.sql)}</span>
                  <span className="list-meta">{timeAgo(q.updated_at)}</span>
                </button>
              )}
              <div className="list-actions">
                <button className="icon-btn sm" aria-label="Rename" onClick={() => setEditing(q.id)}>
                  <Pencil />
                </button>
                <button className="icon-btn sm" aria-label="Delete" onClick={() => remove(q)}>
                  <Trash2 />
                </button>
              </div>
            </div>
          </motion.li>
        ))}
      </AnimatePresence>
    </ul>
  )
}

// ------------------------------------------------------------------ history

function HistoryList({ onOpen }: { onOpen: (sql: string, title: string) => void }) {
  const { user } = useAuth()
  const version = useUI((s) => s.historyVersion)
  const [items, setItems] = useState<HistoryEntry[] | null>(null)
  const [q, setQ] = useState('')
  const timer = useRef<number | undefined>(undefined)

  useEffect(() => {
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(
      () => {
        if (user) {
          api.history(q).then((r) => setItems(r.history)).catch(() => setItems([]))
        } else {
          const needle = q.toLowerCase()
          setItems(localHistory().filter((h) => !needle || h.sql.toLowerCase().includes(needle)))
        }
      },
      q ? 200 : 0,
    )
    return () => window.clearTimeout(timer.current)
  }, [user, version, q])

  const clear = async () => {
    if (user) await api.clearHistory().catch(() => {})
    else clearLocalHistory()
    setItems([])
  }

  return (
    <div className="history">
      <div className="search-box">
        <Search />
        <input className="input input-sm" placeholder="Search history" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search history" />
      </div>
      {!user && <div className="sidebar-note small">History is stored in this browser. Sign in to keep it across devices.</div>}
      {items && items.length === 0 && (
        <div className="sidebar-note">
          <History size={16} /> {q ? 'No matching queries.' : 'Queries you run will appear here.'}
        </div>
      )}
      <ul className="list">
        {items?.map((h) => (
          <li key={h.id}>
            <button className="list-item list-main history-item" onClick={() => onOpen(h.sql, 'From history')}>
              <span className="list-sql mono">{firstLine(h.sql, 120)}</span>
              <span className="list-meta">
                <span className={`dot ${h.ok ? 'ok' : 'err'}`} />
                {h.ok ? (h.row_count !== undefined ? `${formatInt(h.row_count)} rows` : 'OK') : 'Error'} · {formatDuration(h.duration_ms)} · {timeAgo(h.created_at)}
                {h.target === 'core' && user ? ' · original' : ''}
              </span>
            </button>
          </li>
        ))}
      </ul>
      {items && items.length > 0 && (
        <button className="btn btn-ghost btn-sm clear-btn" onClick={clear}>
          <Trash2 /> Clear history
        </button>
      )}
    </div>
  )
}
