import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import type { HistoryEntry, QueryResponse, Target } from '../lib/api'
import { storage } from '../lib/util'

// zustand's persist storage, guarded against browsers that block localStorage
const safeStorage = createJSONStorage(() => ({
  getItem: (k: string) => {
    try {
      return localStorage.getItem(k)
    } catch {
      return null
    }
  },
  setItem: (k: string, v: string) => {
    try {
      localStorage.setItem(k, v)
    } catch {
      /* ignore */
    }
  },
  removeItem: (k: string) => {
    try {
      localStorage.removeItem(k)
    } catch {
      /* ignore */
    }
  },
}))

// ------------------------------------------------------------------ theme

export type Theme = 'dark' | 'light'

interface ThemeState {
  theme: Theme
  toggle: () => void
}

export const useTheme = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'dark',
      toggle: () => set({ theme: get().theme === 'dark' ? 'light' : 'dark' }),
    }),
    { name: 'sqlpg.theme', storage: safeStorage },
  ),
)

// ------------------------------------------------------------------ UI

interface UIState {
  auth: null | 'signin' | 'signup'
  authReason?: string
  openAuth: (mode: 'signin' | 'signup', reason?: string) => void
  closeAuth: () => void
  schemaVersion: number
  bumpSchema: () => void
  historyVersion: number
  bumpHistory: () => void
  savedVersion: number
  bumpSaved: () => void
}

export const useUI = create<UIState>()((set) => ({
  auth: null,
  openAuth: (mode, reason) => set({ auth: mode, authReason: reason }),
  closeAuth: () => set({ auth: null, authReason: undefined }),
  schemaVersion: 0,
  bumpSchema: () => set((s) => ({ schemaVersion: s.schemaVersion + 1 })),
  historyVersion: 0,
  bumpHistory: () => set((s) => ({ historyVersion: s.historyVersion + 1 })),
  savedVersion: 0,
  bumpSaved: () => set((s) => ({ savedVersion: s.savedVersion + 1 })),
}))

// ------------------------------------------------------------------ editor tabs

export interface Tab {
  id: string
  title: string
  sql: string
  savedId?: number
}

export interface RunState {
  running: boolean
  runId?: string
  response?: QueryResponse
  error?: string
  errorCode?: string
  active: number // selected result index
  lineOffset: number // when running a selection: line of the selection start - 1
}

const WELCOME_SQL = `-- Welcome! Press Ctrl/⌘ + Enter to run.
-- ~1.7M rows of e-commerce, HR and finance data. Browse tables on the left.
SELECT c.country,
       COUNT(DISTINCT o.customer_id) AS customers,
       ROUND(SUM(oi.quantity * oi.unit_price * (1 - oi.discount))) AS revenue
FROM orders o
JOIN customers c ON c.customer_id = o.customer_id
JOIN order_items oi ON oi.order_id = o.order_id
WHERE o.status = 'delivered'
GROUP BY c.country
ORDER BY revenue DESC;
`

const newId = () => Math.random().toString(36).slice(2, 10)

interface TabsState {
  tabs: Tab[]
  activeId: string
  target: Target
  counter: number
  runs: Record<string, RunState>
  newTab: (sql?: string, title?: string, savedId?: number) => string
  openOrFocus: (sql: string, title: string, savedId?: number) => string
  closeTab: (id: string) => void
  setActive: (id: string) => void
  updateSql: (id: string, sql: string) => void
  updateTab: (id: string, patch: Partial<Tab>) => void
  setTarget: (t: Target) => void
  setRun: (id: string, patch: Partial<RunState>) => void
}

const firstTab: Tab = { id: 'welcome', title: 'Welcome', sql: WELCOME_SQL }

export const useTabs = create<TabsState>()(
  persist(
    (set, get) => ({
      tabs: [firstTab],
      activeId: firstTab.id,
      target: 'mine',
      counter: 1,
      runs: {},
      newTab: (sql = '', title, savedId) => {
        const id = newId()
        const n = get().counter + 1
        set((s) => ({
          tabs: [...s.tabs, { id, title: title ?? `Query ${n}`, sql, savedId }],
          activeId: id,
          counter: n,
        }))
        return id
      },
      openOrFocus: (sql, title, savedId) => {
        const existing = get().tabs.find((t) => (savedId !== undefined ? t.savedId === savedId : false))
        if (existing) {
          set({ activeId: existing.id })
          return existing.id
        }
        // reuse an empty active tab instead of piling up blank tabs
        const active = get().tabs.find((t) => t.id === get().activeId)
        if (active && active.sql.trim() === '' && !active.savedId) {
          get().updateTab(active.id, { sql, title, savedId })
          return active.id
        }
        return get().newTab(sql, title, savedId)
      },
      closeTab: (id) =>
        set((s) => {
          const idx = s.tabs.findIndex((t) => t.id === id)
          let tabs = s.tabs.filter((t) => t.id !== id)
          if (tabs.length === 0) tabs = [{ id: newId(), title: 'Query 1', sql: '' }]
          const activeId = s.activeId === id ? tabs[Math.max(0, idx - 1)].id : s.activeId
          const runs = { ...s.runs }
          delete runs[id]
          return { tabs, activeId, runs }
        }),
      setActive: (id) => set({ activeId: id }),
      updateSql: (id, sql) => set((s) => ({ tabs: s.tabs.map((t) => (t.id === id ? { ...t, sql } : t)) })),
      updateTab: (id, patch) => set((s) => ({ tabs: s.tabs.map((t) => (t.id === id ? { ...t, ...patch } : t)) })),
      setTarget: (target) => set({ target }),
      setRun: (id, patch) =>
        set((s) => {
          const prev: RunState = s.runs[id] ?? { running: false, active: 0, lineOffset: 0 }
          return { runs: { ...s.runs, [id]: { ...prev, ...patch } } }
        }),
    }),
    {
      name: 'sqlpg.tabs',
      storage: safeStorage,
      partialize: (s) => ({ tabs: s.tabs, activeId: s.activeId, target: s.target, counter: s.counter }),
    },
  ),
)

// ------------------------------------------------------------------ signed-out history and progress

const LOCAL_HISTORY = 'sqlpg.history'
const LOCAL_PROGRESS = 'sqlpg.progress'

export function localHistory(): HistoryEntry[] {
  return storage.get<HistoryEntry[]>(LOCAL_HISTORY, [])
}

export function addLocalHistory(e: Omit<HistoryEntry, 'id' | 'created_at'>) {
  const list = localHistory()
  list.unshift({ ...e, id: Date.now(), created_at: new Date().toISOString() })
  storage.set(LOCAL_HISTORY, list.slice(0, 60))
}

export function clearLocalHistory() {
  storage.remove(LOCAL_HISTORY)
}

export function localSolved(): string[] {
  return storage.get<string[]>(LOCAL_PROGRESS, [])
}

export function addLocalSolved(id: string) {
  const s = new Set(localSolved())
  s.add(id)
  storage.set(LOCAL_PROGRESS, [...s])
}

export function clearLocalSolved() {
  storage.remove(LOCAL_PROGRESS)
}
