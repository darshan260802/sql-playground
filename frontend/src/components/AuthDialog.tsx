import { useEffect, useState, type FormEvent } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { toast } from 'sonner'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../state/auth'
import { useUI } from '../state/stores'
import { Dialog } from './Overlay'
import { Segmented } from './Segmented'
import { spring } from '../lib/util'

export function AuthDialog() {
  const { auth, authReason, closeAuth, openAuth } = useUI()
  const { setSession, config } = useAuth()
  const mode = auth ?? 'signin'
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<{ field?: string; message: string } | null>(null)
  const [touched, setTouched] = useState<Record<string, boolean>>({})

  useEffect(() => {
    setError(null)
    setTouched({})
  }, [auth])

  // validate inline as people type, not only on submit
  const emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())
  const pwOk = password.length >= 8
  const nameOk = name.trim().length > 0
  const canSubmit = emailOk && (mode === 'signin' ? password.length > 0 : pwOk && nameOk)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!canSubmit || busy) return
    setBusy(true)
    setError(null)
    try {
      if (mode === 'signup') {
        const me = await api.signup(name.trim(), email.trim(), password)
        setSession(me.user ?? null, me.database)
        toast.success(`Welcome, ${me.user?.name.split(' ')[0]}!`, {
          description: 'We’re copying the database for you. It takes about 10 seconds.',
        })
      } else {
        const me = await api.login(email.trim(), password)
        setSession(me.user ?? null, me.database)
        toast.success(`Signed in as ${me.user?.name}`)
      }
      setPassword('')
      closeAuth()
    } catch (err) {
      const e = err as ApiError
      const field = e.code === 'invalid_email' || e.code === 'email_taken' ? 'email' : e.code === 'weak_password' ? 'password' : e.code === 'invalid_name' ? 'name' : undefined
      setError({ field, message: e.message })
    } finally {
      setBusy(false)
    }
  }

  const signupClosed = config && !config.signup_open

  return (
    <Dialog
      open={auth !== null}
      onClose={closeAuth}
      title={mode === 'signin' ? 'Welcome back' : 'Create your account'}
      description={
        authReason ??
        (mode === 'signin'
          ? 'Sign in to work on your own copy of the database.'
          : 'Free. You get a private copy of all the data to INSERT, UPDATE, DELETE and CREATE as you like.')
      }
    >
      <Segmented
        full
        ruled
        label="Sign in or create account"
        value={mode}
        onChange={(m) => openAuth(m, authReason)}
        options={[
          { value: 'signin', label: 'Sign in' },
          { value: 'signup', label: 'Create account' },
        ]}
      />
      <form className="auth-form" onSubmit={submit} noValidate>
        <AnimatePresence initial={false}>
          {mode === 'signup' && (
            <motion.div
              key="name"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={spring}
              style={{ overflow: 'hidden' }}
            >
              <div className="field" style={{ paddingBottom: 14 }}>
                <label htmlFor="auth-name">Name</label>
                <input
                  id="auth-name"
                  className="input"
                  autoComplete="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onBlur={() => setTouched((t) => ({ ...t, name: true }))}
                  aria-invalid={(touched.name && !nameOk) || error?.field === 'name'}
                  placeholder="Ada Lovelace"
                  autoFocus
                />
              </div>
            </motion.div>
          )}
        </AnimatePresence>
        <div className="field">
          <label htmlFor="auth-email">Email</label>
          <input
            id="auth-email"
            className="input"
            type="email"
            inputMode="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onBlur={() => setTouched((t) => ({ ...t, email: true }))}
            aria-invalid={(touched.email && !emailOk) || error?.field === 'email'}
            autoFocus={mode === 'signin'}
          />
          {touched.email && !emailOk && <span className="help error">Enter a valid email address.</span>}
        </div>
        <div className="field">
          <label htmlFor="auth-password">Password</label>
          <input
            id="auth-password"
            className="input"
            type="password"
            autoComplete={mode === 'signin' ? 'current-password' : 'new-password'}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            onBlur={() => setTouched((t) => ({ ...t, password: true }))}
            aria-invalid={(mode === 'signup' && touched.password && !pwOk) || error?.field === 'password'}
          />
          {mode === 'signup' && (
            <span className={`help ${touched.password && !pwOk ? 'error' : ''}`}>At least 8 characters.</span>
          )}
        </div>
        <AnimatePresence>
          {error && (
            <motion.div
              className="form-error"
              role="alert"
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              transition={spring}
            >
              {error.message}
            </motion.div>
          )}
        </AnimatePresence>
        {mode === 'signup' && signupClosed ? (
          <div className="form-error">All accounts on this playground are taken. Ask the admin for a spot.</div>
        ) : (
          <button className="btn btn-primary btn-lg btn-block" type="submit" disabled={!canSubmit || busy}>
            {busy && <span className="spinner" />}
            {mode === 'signin' ? 'Sign in' : 'Create account'}
          </button>
        )}
        {mode === 'signin' && (
          <p className="auth-foot">Forgot your password? Ask the playground admin to reset it.</p>
        )}
      </form>
    </Dialog>
  )
}
