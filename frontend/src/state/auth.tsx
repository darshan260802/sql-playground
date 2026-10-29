import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api, type AppConfig, type DatabaseStatus, type User } from '../lib/api'
import { clearLocalSolved, localSolved } from './stores'

interface AuthContextValue {
  ready: boolean
  user: User | null
  database: DatabaseStatus | null
  config: AppConfig | null
  refresh: () => Promise<void>
  setSession: (user: User | null, database?: DatabaseStatus | null) => void
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false)
  const [user, setUser] = useState<User | null>(null)
  const [database, setDatabase] = useState<DatabaseStatus | null>(null)
  const [config, setConfig] = useState<AppConfig | null>(null)

  const refresh = useCallback(async () => {
    try {
      const me = await api.me()
      setUser(me.user ?? null)
      setDatabase(me.database ?? null)
    } catch {
      /* keep the last known state */
    } finally {
      setReady(true)
    }
  }, [])

  useEffect(() => {
    refresh()
    api.config().then(setConfig).catch(() => {})
  }, [refresh])

  // While the personal database copy is being prepared, poll its progress.
  const busy = database?.status === 'provisioning' || database?.status === 'resetting'
  const wasBusy = useRef(false)
  useEffect(() => {
    if (!user || !busy) {
      if (wasBusy.current && user) refresh() // pick up the final user status
      wasBusy.current = false
      return
    }
    wasBusy.current = true
    const id = setInterval(async () => {
      try {
        const db = await api.database()
        setDatabase(db)
        if (db.status !== 'provisioning' && db.status !== 'resetting') {
          setUser((u) => (u ? { ...u, db_status: db.status } : u))
        }
      } catch {
        /* try again */
      }
    }, 900)
    return () => clearInterval(id)
  }, [user, busy, refresh])

  const setSession = useCallback((u: User | null, db?: DatabaseStatus | null) => {
    setUser(u)
    setDatabase(db ?? null)
    // progress made while signed out moves to the new account
    const solved = localSolved()
    if (u && solved.length) {
      api.importProgress(solved).then(clearLocalSolved).catch(() => {})
    }
  }, [])

  const signOut = useCallback(async () => {
    try {
      await api.logout()
    } finally {
      setUser(null)
      setDatabase(null)
    }
  }, [])

  const value = useMemo(
    () => ({ ready, user, database, config, refresh, setSession, signOut }),
    [ready, user, database, config, refresh, setSession, signOut],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth outside AuthProvider')
  return ctx
}

/** The database the playground should run against right now. */
export function useEffectiveTarget(preferred: 'mine' | 'core') {
  const { user } = useAuth()
  if (!user || user.db_status !== 'ready') return 'core' as const
  return preferred
}
