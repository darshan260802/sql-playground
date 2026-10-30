import { useRef, useState, type FormEvent } from 'react'
import { motion } from 'motion/react'
import { toast } from 'sonner'
import { KeyRound, LogOut, RotateCcw, Trash2 } from 'lucide-react'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../state/auth'
import { useTabs, useUI } from '../state/stores'
import { formatBytes, spring } from '../lib/util'
import { Confirm, Dialog, Popover } from './Overlay'

/** The signed-in learner's account: their database copy, its storage, and account actions. */
export function AccountMenu({ placement }: { placement: 'rail' | 'bar' }) {
  const { user, database, signOut, refresh } = useAuth()
  const [open, setOpen] = useState(false)
  const [confirmReset, setConfirmReset] = useState(false)
  const [resetting, setResetting] = useState(false)
  const [pwOpen, setPwOpen] = useState(false)
  const [delOpen, setDelOpen] = useState(false)
  const anchor = useRef<HTMLButtonElement>(null)
  const bumpSchema = useUI((s) => s.bumpSchema)
  if (!user) return null

  const usage = database?.usage_bytes ?? 0
  const quota = database?.quota_bytes ?? 1
  const pct = Math.min(100, (usage / quota) * 100)
  const busy = database?.status === 'provisioning' || database?.status === 'resetting'
  const initial = user.name.trim().charAt(0).toUpperCase()

  const doReset = async () => {
    setResetting(true)
    try {
      await api.resetDatabase()
      await refresh()
      useTabs.setState({ runs: {} })
      bumpSchema()
      toast('Restoring the original data…', { description: 'Your tables, views and routines are being replaced. This takes about 10 seconds.' })
      setConfirmReset(false)
      setOpen(false)
    } catch (e) {
      toast.error((e as ApiError).message)
    } finally {
      setResetting(false)
    }
  }

  const panel = (
    <>
      <div className="menu-head">
        <div className="avatar lg">{initial}</div>
        <div style={{ minWidth: 0 }}>
          <div className="menu-name">{user.name}</div>
          <div className="menu-email">{user.email}</div>
        </div>
      </div>
      <div className="menu-section">
        <div className="usage-row">
          <span>Your database</span>
          <span className="mono">{database?.schema}</span>
        </div>
        {busy ? (
          <div className="usage-note">Preparing your copy…</div>
        ) : (
          <>
            <div className="meter" role="meter" aria-valuemin={0} aria-valuemax={quota} aria-valuenow={usage} aria-label="Storage used">
              <motion.span initial={false} animate={{ width: `${pct}%` }} transition={spring} className={pct > 90 ? 'over' : ''} />
            </div>
            <div className="usage-note num">
              {formatBytes(usage)} of {formatBytes(quota)}
            </div>
          </>
        )}
      </div>
      <div className="menu-list">
        <button className="menu-item" onClick={() => setConfirmReset(true)} disabled={busy}>
          <RotateCcw /> Reset to the original data
        </button>
        <button className="menu-item" onClick={() => { setPwOpen(true); setOpen(false) }}>
          <KeyRound /> Change password
        </button>
        <button className="menu-item" onClick={() => { signOut(); setOpen(false); toast('Signed out') }}>
          <LogOut /> Sign out
        </button>
        <div className="menu-sep" />
        <button className="menu-item danger" onClick={() => { setDelOpen(true); setOpen(false) }}>
          <Trash2 /> Delete account
        </button>
      </div>
    </>
  )

  return (
    <>
      {placement === 'rail' ? (
        <button ref={anchor} className="rail-item" onClick={() => setOpen((o) => !o)} aria-label="Account" aria-expanded={open} data-tip={open ? undefined : 'Account'}>
          <span className="avatar">{initial}</span>
        </button>
      ) : (
        <button ref={anchor} className="bottombar-item" onClick={() => setOpen(true)} aria-expanded={open}>
          <span className="avatar sm">{initial}</span>
          <span>Account</span>
        </button>
      )}
      {placement === 'rail' ? (
        <Popover open={open} onClose={() => setOpen(false)} anchor={anchor} placement="right-end" width={300}>
          {panel}
        </Popover>
      ) : (
        <Dialog open={open} onClose={() => setOpen(false)} title="Account">
          <div className="account-sheet">{panel}</div>
        </Dialog>
      )}
      <Confirm
        open={confirmReset}
        onClose={() => setConfirmReset(false)}
        title="Reset your database?"
        body={
          <>
            Every table, row, view, routine and trigger in <b className="mono">{database?.schema}</b> is replaced with a fresh copy of the original data.
            Your saved queries, history and lesson progress are kept.
          </>
        }
        confirmLabel="Reset database"
        danger
        busy={resetting}
        onConfirm={doReset}
      />
      <PasswordDialog open={pwOpen} onClose={() => setPwOpen(false)} />
      <DeleteDialog open={delOpen} onClose={() => setDelOpen(false)} />
    </>
  )
}

function PasswordDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.changePassword(current, next)
      toast.success('Password changed', { description: 'Other devices were signed out.' })
      setCurrent('')
      setNext('')
      onClose()
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onClose={onClose} title="Change password">
      <form className="auth-form" onSubmit={submit}>
        <div className="field">
          <label htmlFor="pw-current">Current password</label>
          <input id="pw-current" className="input" type="password" autoComplete="current-password" value={current} onChange={(e) => setCurrent(e.target.value)} autoFocus />
        </div>
        <div className="field">
          <label htmlFor="pw-new">New password</label>
          <input id="pw-new" className="input" type="password" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)} />
          <span className="help">At least 8 characters.</span>
        </div>
        {error && <div className="form-error" role="alert">{error}</div>}
        <button className="btn btn-primary btn-lg btn-block" disabled={busy || !current || next.length < 8}>
          {busy && <span className="spinner" />}Change password
        </button>
      </form>
    </Dialog>
  )
}

function DeleteDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { setSession } = useAuth()
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await api.deleteAccount(password)
      setSession(null)
      toast('Your account and database were deleted.')
      onClose()
    } catch (err) {
      setError((err as ApiError).message)
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog open={open} onClose={onClose} title="Delete your account?" description="This permanently removes your database copy, saved queries, history and progress. It can’t be undone.">
      <form className="auth-form" onSubmit={submit}>
        <div className="field">
          <label htmlFor="del-pw">Confirm with your password</label>
          <input id="del-pw" className="input" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} autoFocus />
        </div>
        {error && <div className="form-error" role="alert">{error}</div>}
        <button className="btn btn-danger btn-lg btn-block" disabled={busy || !password}>
          {busy && <span className="spinner" />}Delete account permanently
        </button>
      </form>
    </Dialog>
  )
}
