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
  label: string
  full?: boolean
  ruled?: boolean
  className?: string
}

/** A choice between a few views or targets: the selected one carries a 1px accent line that slides between labels. */
export function Segmented<T extends string>({ value, options, onChange, label, full, ruled, className }: Props<T>) {
  const id = useId()
  return (
    <div className={`tabs ${full ? 'full' : ''} ${ruled ? 'ruled' : ''} ${className ?? ''}`} role="group" aria-label={label}>
      {options.map((o) => (
        <button key={o.value} type="button" aria-pressed={o.value === value} title={o.title} disabled={o.disabled} onClick={() => onChange(o.value)}>
          {o.label}
          {o.value === value && <motion.span layoutId={`line-${id}`} className="tab-line" transition={springSnappy} />}
        </button>
      ))}
    </div>
  )
}
