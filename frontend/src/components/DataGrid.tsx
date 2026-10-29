import { useMemo, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import type { Column } from '../lib/api'
import { copyText } from '../lib/util'
import { toast } from 'sonner'

const NUMERIC = /^(TINYINT|SMALLINT|MEDIUMINT|INT|INTEGER|BIGINT|DECIMAL|NUMERIC|FLOAT|DOUBLE|REAL|YEAR|UNSIGNED)/i
const ROW_H = 30
const CHAR_W = 7.7

interface Props {
  columns: Column[]
  rows: (string | null)[][]
  dense?: boolean
  maxHeight?: number
}

export function DataGrid({ columns, rows, dense, maxHeight }: Props) {
  const scroller = useRef<HTMLDivElement>(null)
  const [selected, setSelected] = useState<{ r: number; c: number } | null>(null)

  const { widths, numeric, template, totalWidth } = useMemo(() => {
    const numeric = columns.map((c) => NUMERIC.test(c.type))
    const sample = rows.slice(0, 300)
    const widths = columns.map((c, i) => {
      // the header shows the name and the (smaller) type label
      let max = Math.max(c.name.length + 2, Math.ceil(c.type.length * 0.85) + 1)
      for (const r of sample) {
        const v = r[i]
        const len = v === null ? 4 : v.length
        if (len > max) max = len
        if (max > 60) break
      }
      return Math.round(Math.max(72, Math.min(420, max * CHAR_W + 26)))
    })
    const indexW = Math.max(44, String(rows.length).length * 8 + 22)
    const template = [indexW, ...widths].map((w) => `${w}px`).join(' ')
    return { widths, numeric, template, totalWidth: indexW + widths.reduce((a, b) => a + b, 0) }
  }, [columns, rows])

  const rowH = dense ? 26 : ROW_H
  const virt = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => rowH,
    overscan: 14,
  })

  const copyCell = (v: string | null) => {
    copyText(v ?? 'NULL').then((ok) => ok && toast('Copied cell value', { duration: 1200 }))
  }

  return (
    <div
      ref={scroller}
      className={`grid-scroller ${dense ? 'dense' : ''}`}
      style={maxHeight ? { maxHeight } : undefined}
      role="grid"
      aria-rowcount={rows.length + 1}
      aria-colcount={columns.length}
    >
      <div style={{ width: totalWidth, minWidth: '100%' }}>
        <div className="grid-row grid-head" style={{ gridTemplateColumns: template }} role="row">
          <div className="grid-cell grid-index" role="columnheader" aria-label="Row number">
            #
          </div>
          {columns.map((c, i) => (
            <div key={i} className={`grid-cell ${numeric[i] ? 'num' : ''}`} role="columnheader" title={`${c.name} · ${c.type}`}>
              <span className="grid-colname">{c.name}</span>
              <span className="grid-coltype">{c.type.toLowerCase()}</span>
            </div>
          ))}
        </div>
        <div style={{ height: virt.getTotalSize(), position: 'relative' }}>
          {virt.getVirtualItems().map((vr) => {
            const row = rows[vr.index]
            return (
              <div
                key={vr.key}
                className={`grid-row ${vr.index % 2 ? 'odd' : ''}`}
                role="row"
                style={{ gridTemplateColumns: template, height: rowH, transform: `translateY(${vr.start}px)` }}
              >
                <div className="grid-cell grid-index" role="rowheader">
                  {vr.index + 1}
                </div>
                {row.map((v, ci) => {
                  const isSel = selected?.r === vr.index && selected?.c === ci
                  return (
                    <div
                      key={ci}
                      role="gridcell"
                      className={`grid-cell ${numeric[ci] ? 'num' : ''} ${isSel ? 'selected' : ''}`}
                      title={v !== null && v.length * CHAR_W > widths[ci] - 20 ? v : undefined}
                      onClick={() => setSelected({ r: vr.index, c: ci })}
                      onDoubleClick={() => copyCell(v)}
                    >
                      {v === null ? <span className="null">NULL</span> : v}
                    </div>
                  )
                })}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
