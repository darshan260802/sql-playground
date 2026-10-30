import { useEffect, useRef, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'
import { ArrowLeft } from 'lucide-react'
import { api, type ApiError } from '../lib/api'
import { useTabs } from '../state/stores'

/** /q/:id opens a shared query in a new playground tab. */
export function SharedQuery() {
  const { id = '' } = useParams()
  const [state, setState] = useState<'loading' | 'done' | 'missing'>('loading')
  const [message, setMessage] = useState('')
  const started = useRef(false)

  useEffect(() => {
    if (started.current) return
    started.current = true
    api
      .getShare(id)
      .then((sh) => {
        useTabs.getState().newTab(sh.sql, sh.title || 'Shared query')
        toast('Shared query opened in a new tab', { description: 'Press Ctrl/⌘ + Enter to run it.' })
        setState('done')
      })
      .catch((e) => {
        setMessage((e as ApiError).message)
        setState('missing')
      })
  }, [id])

  if (state === 'done') return <Navigate to="/" replace />
  if (state === 'missing') {
    return (
      <div className="page-empty">
        <h1>{message || 'This shared query doesn’t exist.'}</h1>
        <p className="muted">The link may be mistyped, or the query was shared from another playground.</p>
        <Link to="/" className="btn">
          <ArrowLeft /> Back to the playground
        </Link>
      </div>
    )
  }
  return (
    <div className="page-empty" aria-busy="true">
      <h1>Opening the shared query…</h1>
      <span className="spinner" />
    </div>
  )
}

export function NotFound() {
  return (
    <div className="page-empty">
      <h1>There’s nothing at this address</h1>
      <p className="muted">Error 404. The link may be mistyped or out of date. These are the two places to go:</p>
      <ul className="page-empty-links">
        <li>
          <Link to="/">
            <span className="link-title">Playground</span>
            <span className="link-what">Run any SQL on 24 tables and about 1.7 million rows</span>
          </Link>
        </li>
        <li>
          <Link to="/learn">
            <span className="link-title">Learn</span>
            <span className="link-what">30 lessons with exercises checked against the real database</span>
          </Link>
        </li>
      </ul>
    </div>
  )
}
