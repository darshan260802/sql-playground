import type { Transition } from 'motion/react'

// ------------------------------------------------------------------ motion
// Apple-style springs: critically damped by default (no overshoot); a little
// bounce only where a gesture carried momentum.
export const spring: Transition = { type: 'spring', bounce: 0, duration: 0.35 }
export const springSnappy: Transition = { type: 'spring', bounce: 0, duration: 0.25 }
export const springMomentum: Transition = { type: 'spring', bounce: 0.18, duration: 0.4 }

// ------------------------------------------------------------------ formatting
const nf = new Intl.NumberFormat('en-US')
export const formatInt = (n: number) => nf.format(n)

export function formatCompact(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1).replace(/\.0$/, '') + 'M'
  if (n >= 10_000) return Math.round(n / 1000) + 'k'
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
  return String(n)
}

export function formatBytes(b: number): string {
  if (b >= 1 << 30) return (b / (1 << 30)).toFixed(1) + ' GB'
  if (b >= 1 << 20) return Math.round(b / (1 << 20)) + ' MB'
  if (b >= 1 << 10) return Math.round(b / (1 << 10)) + ' KB'
  return b + ' B'
}

export function formatDuration(ms: number): string {
  if (ms < 1) return '<1 ms'
  if (ms < 1000) return Math.round(ms) + ' ms'
  return (ms / 1000).toFixed(ms < 10_000 ? 2 : 1) + ' s'
}

export function timeAgo(iso: string): string {
  const t = new Date(iso).getTime()
  const s = Math.round((Date.now() - t) / 1000)
  if (s < 45) return 'just now'
  const m = Math.round(s / 60)
  if (m < 60) return `${m} min ago`
  const h = Math.round(m / 60)
  if (h < 24) return `${h} h ago`
  const d = Math.round(h / 24)
  if (d < 30) return `${d} d ago`
  return new Date(iso).toLocaleDateString()
}

export const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent)
export const modKey = isMac ? '⌘' : 'Ctrl'

// ------------------------------------------------------------------ storage
// localStorage can throw (private mode, blocked storage): never let it break the app.
export const storage = {
  get<T>(key: string, fallback: T): T {
    try {
      const raw = localStorage.getItem(key)
      return raw === null ? fallback : (JSON.parse(raw) as T)
    } catch {
      return fallback
    }
  },
  set(key: string, value: unknown) {
    try {
      localStorage.setItem(key, JSON.stringify(value))
    } catch {
      /* ignore */
    }
  },
  remove(key: string) {
    try {
      localStorage.removeItem(key)
    } catch {
      /* ignore */
    }
  },
}

export function firstLine(sql: string, max = 90): string {
  const line = sql.replace(/\s+/g, ' ').trim()
  return line.length > max ? line.slice(0, max - 1) + '…' : line
}

export function clamp(v: number, lo: number, hi: number) {
  return Math.max(lo, Math.min(hi, v))
}

// Rubber-banding past a boundary: the further you pull, the less it follows.
export function rubberband(overshoot: number, dimension: number, constant = 0.55) {
  const sign = Math.sign(overshoot)
  const o = Math.abs(overshoot)
  return (sign * (o * dimension * constant)) / (dimension + constant * o)
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
