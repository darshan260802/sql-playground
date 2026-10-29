// Typed client for the Go API. Every request sends the X-Requested-With header
// the server requires on state-changing requests (CSRF protection).

export type Target = 'mine' | 'core'

export interface User {
  id: number
  email: string
  name: string
  db_status: 'provisioning' | 'ready' | 'resetting' | 'error'
  db_error?: string
  created_at: string
}

export interface DatabaseStatus {
  schema: string
  status: User['db_status']
  error?: string
  done: number
  total: number
  usage_bytes: number
  quota_bytes: number
  core_bytes: number
  reset_at: string | null
}

export interface Me {
  user?: User
  database?: DatabaseStatus
}

export interface AppConfig {
  max_rows: number
  query_timeout_s: number
  anon_timeout_s: number
  user_quota_mb: number
  signup_open: boolean
  core_schema: string
  lesson_count: number
  exercise_count: number
}

export interface Column {
  name: string
  type: string
}

export interface SQLError {
  code?: number
  message: string
  hint?: string
  line?: number
  reason?: string
}

export interface StatementResult {
  index: number
  sql: string
  line: number
  columns?: Column[]
  rows?: (string | null)[][]
  row_count: number
  truncated?: boolean
  rows_affected?: number
  last_insert_id?: number
  message?: string
  warnings?: string[]
  duration_ms: number
  error?: SQLError
}

export interface QueryResponse {
  results: StatementResult[]
  duration_ms: number
  target: Target
  database: string
  session?: { in_transaction: boolean }
}

export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  key?: string
  default?: string
  comment?: string
  ref?: string
}

export interface TableInfo {
  name: string
  type: 'table' | 'view'
  domain: string
  comment: string
  rows: number
  exact: boolean
  columns: ColumnInfo[]
}

export interface SchemaInfo {
  schema: string
  target: Target
  tables: TableInfo[]
  routines: { name: string; type: string }[]
  triggers: { name: string; table: string; event: string }[]
}

export interface SavedQuery {
  id: number
  title: string
  sql: string
  created_at: string
  updated_at: string
}

export interface HistoryEntry {
  id: number
  sql: string
  target: Target
  ok: boolean
  error?: string
  duration_ms: number
  row_count?: number
  created_at: string
}

export interface LessonSummary {
  slug: string
  title: string
  level: string
  minutes: number
  summary: string
  exercises: string[]
  solved: string[]
}

export interface ModuleSummary {
  name: string
  lessons: LessonSummary[]
}

export interface Preview {
  columns: Column[]
  rows: (string | null)[][]
  row_count: number
}

export interface Exercise {
  id: string
  lesson: string
  title: string
  kind: 'select' | 'dml' | 'ddl'
  ordered: boolean
  prompt: string
  hint?: string
  requires_login: boolean
  expected?: Preview
  solved: boolean
  attempts: number
  last_sql?: string
}

export interface Lesson {
  slug: string
  title: string
  module: string
  level: string
  minutes: number
  summary: string
  body: string
  exercises: Exercise[]
  prev: { slug: string; title: string } | null
  next: { slug: string; title: string } | null
}

export interface CheckOutcome {
  correct: boolean
  message: string
  detail?: string
  error?: SQLError
  result?: Preview
}

export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function request<T>(method: string, path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method,
      credentials: 'same-origin',
      signal,
      headers: {
        'X-Requested-With': 'sqlpg',
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e
    throw new ApiError(0, 'network', 'Can’t reach the server. Check your connection and try again.')
  }
  const text = await res.text()
  let data: unknown = undefined
  try {
    data = text ? JSON.parse(text) : undefined
  } catch {
    /* non-JSON error page */
  }
  if (!res.ok) {
    const err = (data as { error?: { code: string; message: string } } | undefined)?.error
    throw new ApiError(res.status, err?.code ?? 'http_' + res.status, err?.message ?? `Request failed (${res.status}).`)
  }
  return data as T
}

export const api = {
  config: () => request<AppConfig>('GET', '/api/config'),
  me: () => request<Me>('GET', '/api/auth/me'),
  signup: (name: string, email: string, password: string) =>
    request<Me>('POST', '/api/auth/signup', { name, email, password }),
  login: (email: string, password: string) => request<Me>('POST', '/api/auth/login', { email, password }),
  logout: () => request<{ ok: boolean }>('POST', '/api/auth/logout', {}),
  changePassword: (current_password: string, new_password: string) =>
    request<{ ok: boolean }>('POST', '/api/account/password', { current_password, new_password }),
  deleteAccount: (password: string) => request<{ ok: boolean }>('POST', '/api/account/delete', { password }),

  query: (sql: string, target: Target, run_id: string, signal?: AbortSignal) =>
    request<QueryResponse>('POST', '/api/query', { sql, target, run_id }, signal),
  cancel: (run_id: string) => request<{ cancelled: boolean }>('POST', '/api/query/cancel', { run_id }),
  resetSession: () => request<{ ok: boolean }>('POST', '/api/session/reset', {}),
  schema: (target: Target) => request<SchemaInfo>('GET', `/api/schema?target=${target}`),

  database: () => request<DatabaseStatus>('GET', '/api/database'),
  resetDatabase: () => request<DatabaseStatus>('POST', '/api/database/reset', {}),

  saved: () => request<{ saved: SavedQuery[] }>('GET', '/api/saved'),
  createSaved: (title: string, sql: string) => request<{ id: number }>('POST', '/api/saved', { title, sql }),
  updateSaved: (id: number, title: string, sql: string) =>
    request<{ ok: boolean }>('PUT', `/api/saved/${id}`, { title, sql }),
  deleteSaved: (id: number) => request<{ ok: boolean }>('DELETE', `/api/saved/${id}`),
  history: (q = '') => request<{ history: HistoryEntry[] }>('GET', `/api/history?q=${encodeURIComponent(q)}`),
  clearHistory: () => request<{ ok: boolean }>('DELETE', '/api/history'),
  share: (title: string, sql: string) => request<{ id: string }>('POST', '/api/share', { title, sql }),
  getShare: (id: string) => request<{ id: string; title: string; sql: string }>('GET', `/api/share/${id}`),

  lessons: () => request<{ modules: ModuleSummary[] }>('GET', '/api/lessons'),
  lesson: (slug: string) => request<{ lesson: Lesson }>('GET', `/api/lessons/${slug}`),
  check: (id: string, sql: string) => request<CheckOutcome>('POST', `/api/exercises/${id}/check`, { sql }),
  solution: (id: string) => request<{ solution: string }>('GET', `/api/exercises/${id}/solution`),
  importProgress: (solved: string[]) => request<{ imported: number }>('POST', '/api/progress/import', { solved }),
}

export function newRunId(): string {
  return crypto.randomUUID ? crypto.randomUUID() : Math.random().toString(36).slice(2) + Date.now().toString(36)
}
