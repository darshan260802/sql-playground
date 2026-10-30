import { useEffect, useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, motion, useDragControls, type PanInfo } from 'motion/react'
import { X } from 'lucide-react'
import { useEscape, useIsMobile } from '../lib/hooks'
import { spring, springMomentum } from '../lib/util'

// ------------------------------------------------------------------ Dialog / bottom sheet

interface DialogProps {
  open: boolean
  onClose: () => void
  title: ReactNode
  description?: ReactNode
  children: ReactNode
  width?: number
}

/**
 * A modal task: dims the background to focus. Centered dialog on desktop, a
 * draggable bottom sheet on phones (enter from the bottom, dismiss downward).
 */
export function Dialog({ open, onClose, title, description, children, width = 420 }: DialogProps) {
  const mobile = useIsMobile()
  useEscape(open, onClose)
  useEffect(() => {
    if (!open) return
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [open])

  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="overlay-root">
          <motion.div
            className="scrim"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            onClick={onClose}
          />
          {mobile ? (
            <Sheet onClose={onClose} title={title} description={description}>
              {children}
            </Sheet>
          ) : (
            <motion.div
              role="dialog"
              aria-modal="true"
              aria-label={typeof title === 'string' ? title : undefined}
              className="dialog"
              style={{ width }}
              initial={{ opacity: 0, scale: 0.96, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.96, y: 10 }}
              transition={spring}
            >
              <DialogHeader title={title} description={description} onClose={onClose} />
              <div className="dialog-body">{children}</div>
            </motion.div>
          )}
        </div>
      )}
    </AnimatePresence>,
    document.body,
  )
}

function DialogHeader({ title, description, onClose }: { title: ReactNode; description?: ReactNode; onClose: () => void }) {
  return (
    <div className="dialog-header">
      <div>
        <h3>{title}</h3>
        {description && <p className="dialog-desc">{description}</p>}
      </div>
      <button className="icon-btn" onClick={onClose} aria-label="Close">
        <X />
      </button>
    </div>
  )
}

function Sheet({ onClose, title, description, children }: { onClose: () => void; title: ReactNode; description?: ReactNode; children: ReactNode }) {
  const controls = useDragControls()
  const onDragEnd = (_: unknown, info: PanInfo) => {
    // Project where the flick is heading instead of using the release point.
    const projected = info.offset.y + (info.velocity.y / 1000) * (0.998 / (1 - 0.998))
    if (projected > 160) onClose()
  }
  return (
    <motion.div
      role="dialog"
      aria-modal="true"
      aria-label={typeof title === 'string' ? title : undefined}
      className="sheet"
      initial={{ y: '100%' }}
      animate={{ y: 0 }}
      exit={{ y: '100%' }}
      transition={springMomentum}
      drag="y"
      dragControls={controls}
      dragListener={false}
      dragConstraints={{ top: 0, bottom: 0 }}
      dragElastic={{ top: 0.04, bottom: 0.9 }}
      onDragEnd={onDragEnd}
    >
      <div className="sheet-grab" onPointerDown={(e) => controls.start(e)}>
        <span />
      </div>
      <div onPointerDown={(e) => controls.start(e)}>
        <DialogHeader title={title} description={description} onClose={onClose} />
      </div>
      <div className="dialog-body">{children}</div>
    </motion.div>
  )
}

// ------------------------------------------------------------------ Confirm

interface ConfirmProps {
  open: boolean
  title: string
  body: ReactNode
  confirmLabel: string
  danger?: boolean
  busy?: boolean
  onConfirm: () => void
  onClose: () => void
}

export function Confirm({ open, title, body, confirmLabel, danger, busy, onConfirm, onClose }: ConfirmProps) {
  return (
    <Dialog open={open} onClose={onClose} title={title} width={400}>
      <div className="confirm-body">{body}</div>
      <div className="dialog-actions">
        <button className="btn btn-text" onClick={onClose}>
          Cancel
        </button>
        <button className={`btn ${danger ? 'btn-danger' : 'btn-primary'}`} onClick={onConfirm} disabled={busy} autoFocus>
          {busy && <span className="spinner" />}
          {confirmLabel}
        </button>
      </div>
    </Dialog>
  )
}

// ------------------------------------------------------------------ Popover

interface PopoverProps {
  open: boolean
  onClose: () => void
  anchor: RefObject<HTMLElement | null>
  align?: 'start' | 'end'
  /** below the anchor (default), or beside it with the bottoms aligned (for the nav rail) */
  placement?: 'below' | 'right-end'
  width?: number
  children: ReactNode
  className?: string
}

/** A non-modal panel that grows out of the element that opened it. */
export function Popover({ open, onClose, anchor, align = 'end', placement = 'below', width = 260, children, className }: PopoverProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState<{ top?: number; bottom?: number; left: number } | null>(null)
  useEscape(open, onClose)

  useLayoutEffect(() => {
    if (!open || !anchor.current) return
    const place = () => {
      const r = anchor.current!.getBoundingClientRect()
      const w = Math.min(width, window.innerWidth - 16)
      if (placement === 'right-end') {
        setPos({ bottom: Math.max(8, window.innerHeight - r.bottom), left: r.right + 10 })
        return
      }
      let left = align === 'end' ? r.right - w : r.left
      left = Math.max(8, Math.min(left, window.innerWidth - w - 8))
      setPos({ top: r.bottom + 6, left })
    }
    place()
    window.addEventListener('resize', place)
    return () => window.removeEventListener('resize', place)
  }, [open, anchor, align, placement, width])

  useEffect(() => {
    if (!open) return
    const onDown = (e: PointerEvent) => {
      const t = e.target as Node
      if (ref.current?.contains(t) || anchor.current?.contains(t)) return
      onClose()
    }
    window.addEventListener('pointerdown', onDown)
    return () => window.removeEventListener('pointerdown', onDown)
  }, [open, onClose, anchor])

  return createPortal(
    <AnimatePresence>
      {open && pos && (
        <motion.div
          ref={ref}
          className={`popover ${className ?? ''}`}
          style={{
            top: pos.top,
            bottom: pos.bottom,
            left: pos.left,
            width: Math.min(width, window.innerWidth - 16),
            transformOrigin: placement === 'right-end' ? 'bottom left' : align === 'end' ? 'top right' : 'top left',
          }}
          initial={{ opacity: 0, scale: 0.95, y: -4 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: -4 }}
          transition={{ ...spring, duration: 0.25 }}
        >
          {children}
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  )
}
