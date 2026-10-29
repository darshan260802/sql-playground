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
      <div className="lesson-missing">
        <h2>{message || 'This shared query doesn’t exist.'}</h2>
        <Link to="/" className="btn">
          <ArrowLeft /> Back to the playground
        </Link>
      </div>
    )
  }
  return (
    <div className="lesson-missing">
      <span className="spinner" />
    </div>
  )
}

export function NotFound() {
  return (
    <div className="lesson-missing">
      <span className="eyebrow">404</span>
      <h2>There’s nothing at this address.</h2>
      <Link to="/" className="btn btn-primary">
        Go to the playground
      </Link>
    </div>
  )
}
