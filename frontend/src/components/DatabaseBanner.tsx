import { AnimatePresence, motion } from 'motion/react'
import { AlertTriangle } from 'lucide-react'
import { toast } from 'sonner'
import { api } from '../lib/api'
import { useAuth } from '../state/auth'
import { spring } from '../lib/util'

/** Shows progress while a learner's private copy of the data is being built. */
export function DatabaseBanner() {
  const { user, database, refresh } = useAuth()
  const status = database?.status ?? user?.db_status
  const busy = status === 'provisioning' || status === 'resetting'
  const failed = status === 'error'
  const pct = database && database.total > 0 ? Math.round((database.done / database.total) * 100) : 0

  const retry = async () => {
    try {
      await api.resetDatabase()
      await refresh()
    } catch (e) {
      toast.error((e as Error).message)
    }
  }

  return (
    <AnimatePresence initial={false}>
      {user && (busy || failed) && (
        <motion.div
          className={`db-banner ${failed ? 'failed' : ''}`}
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={spring}
          role="status"
        >
          <div className="db-banner-inner">
            {failed ? (
              <>
                <AlertTriangle />
                <span>We couldn’t prepare your database. {database?.error}</span>
                <button className="btn btn-sm" onClick={retry}>
                  Try again
                </button>
              </>
            ) : (
              <>
                <span className="db-banner-text">
                  {status === 'resetting' ? 'Restoring the original data' : 'Preparing your private copy of the database'}
                  <span className="muted"> · you can query the original data meanwhile</span>
                </span>
                <div className="db-progress" aria-label="Progress" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} role="progressbar">
                  <motion.span animate={{ width: `${Math.max(4, pct)}%` }} transition={spring} />
                </div>
                <span className="mono db-pct">{pct}%</span>
              </>
            )}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
