import { useId } from 'react'
import { motion } from 'motion/react'
import { springSnappy } from '../lib/util'

interface Option<T extends string> {
  value: T
  label: React.ReactNode
  title?: string
  disabled?: boolean
}

interface Props<T extends string> {
  value: T
  options: Option<T>[]
  onChange: (v: T) => void
  full?: boolean
  label: string
}

/** A segmented control whose selection thumb slides between options. */
export function Segmented<T extends string>({ value, options, onChange, full, label }: Props<T>) {
  const id = useId()
  return (
    <div className={`segmented ${full ? 'full' : ''}`} role="group" aria-label={label}>
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          aria-pressed={o.value === value}
          title={o.title}
          disabled={o.disabled}
          onClick={() => onChange(o.value)}
        >
          {o.value === value && <motion.span layoutId={`seg-${id}`} className="thumb" transition={springSnappy} />}
          {o.label}
        </button>
      ))}
    </div>
  )
}
