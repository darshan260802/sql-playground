import { useRef } from 'react'
import { animate, type MotionValue } from 'motion/react'
import { clamp, rubberband, springMomentum, storage } from '../lib/util'

interface Props {
  axis: 'x' | 'y'
  size: MotionValue<number>
  min: number
  max: () => number
  storageKey?: string
  label: string
  /** The pane sits after the handle (to its right or below), so dragging towards it shrinks it. */
  invert?: boolean
  /** Double-click returns to this size. */
  defaultSize?: number
  className?: string
}

/**
 * A drag handle that resizes a pane 1:1 with the pointer (respecting where it
 * was grabbed), resists softly past its limits, and springs back on release.
 */
export function Splitter({ axis, size, min, max, storageKey, label, invert, defaultSize, className }: Props) {
  const drag = useRef<{ start: number; startSize: number } | null>(null)

  const onPointerDown = (e: React.PointerEvent) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { start: axis === 'x' ? e.clientX : e.clientY, startSize: size.get() }
    document.body.classList.add(axis === 'x' ? 'resizing-x' : 'resizing-y')
  }
  const onPointerMove = (e: React.PointerEvent) => {
    if (!drag.current) return
    const delta = (axis === 'x' ? e.clientX : e.clientY) - drag.current.start
    const raw = drag.current.startSize + (invert ? -delta : delta)
    const hi = max()
    let v = raw
    if (raw < min) v = min + rubberband(raw - min, 200)
    else if (raw > hi) v = hi + rubberband(raw - hi, 200)
    size.set(v)
  }
  const end = (e: React.PointerEvent) => {
    if (!drag.current) return
    drag.current = null
    e.currentTarget.releasePointerCapture?.(e.pointerId)
    document.body.classList.remove('resizing-x', 'resizing-y')
    const target = clamp(size.get(), min, max())
    animate(size, target, springMomentum)
    if (storageKey) storage.set(storageKey, target)
  }
  const onKeyDown = (e: React.KeyboardEvent) => {
    const step = e.shiftKey ? 48 : 16
    const dec = axis === 'x' ? 'ArrowLeft' : 'ArrowUp'
    const inc = axis === 'x' ? 'ArrowRight' : 'ArrowDown'
    if (e.key !== dec && e.key !== inc) return
    e.preventDefault()
    const grow = (e.key === inc) !== !!invert
    const target = clamp(size.get() + (grow ? step : -step), min, max())
    animate(size, target, springMomentum)
    if (storageKey) storage.set(storageKey, target)
  }
  const onDoubleClick = () => {
    if (defaultSize === undefined) return
    const target = clamp(defaultSize, min, max())
    animate(size, target, springMomentum)
    if (storageKey) storage.remove(storageKey)
  }

  return (
    <div
      className={`splitter splitter-${axis} ${className ?? ''}`}
      role="separator"
      aria-orientation={axis === 'x' ? 'vertical' : 'horizontal'}
      aria-label={label}
      tabIndex={0}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={end}
      onPointerCancel={end}
      onKeyDown={onKeyDown}
      onDoubleClick={onDoubleClick}
      title={defaultSize !== undefined ? 'Drag to resize, double-click to reset' : undefined}
    >
      <span />
    </div>
  )
}
