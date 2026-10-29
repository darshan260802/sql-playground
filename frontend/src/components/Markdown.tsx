import { useMemo, type ReactNode } from 'react'
import { marked, type Token, type Tokens } from 'marked'
import { highlightCode, classHighlighter } from '@lezer/highlight'
import { MySQL } from '@codemirror/lang-sql'
import { Copy, SquareArrowOutUpRight } from 'lucide-react'
import { toast } from 'sonner'
import { copyText } from '../lib/util'

marked.setOptions({ gfm: true, breaks: false })

/** Static SQL highlighting with the same parser the editor uses. */
export function HighlightedSQL({ code }: { code: string }) {
  const nodes = useMemo(() => {
    const out: ReactNode[] = []
    const tree = MySQL.language.parser.parse(code)
    let i = 0
    highlightCode(
      code,
      tree,
      classHighlighter,
      (text, classes) => {
        out.push(classes ? <span key={i++} className={classes}>{text}</span> : text)
      },
      () => out.push('\n'),
    )
    return out
  }, [code])
  return <code>{nodes}</code>
}

function CodeBlock({ code, lang, onOpen }: { code: string; lang?: string; onOpen?: (sql: string) => void }) {
  const isSql = !lang || lang === 'sql'
  return (
    <div className="md-code">
      <pre>{isSql ? <HighlightedSQL code={code} /> : <code>{code}</code>}</pre>
      {isSql && (
        <div className="md-code-actions">
          <button
            className="icon-btn sm"
            title="Copy"
            aria-label="Copy SQL"
            onClick={() => copyText(code).then((ok) => ok && toast('Copied', { duration: 1200 }))}
          >
            <Copy />
          </button>
          {onOpen && (
            <button className="btn btn-sm md-open" onClick={() => onOpen(code)}>
              <SquareArrowOutUpRight /> Open in playground
            </button>
          )}
        </div>
      )}
    </div>
  )
}

export function Markdown({ text, onOpenSql, className }: { text: string; onOpenSql?: (sql: string) => void; className?: string }) {
  const blocks = useMemo(() => {
    const tokens = marked.lexer(text)
    const out: ({ kind: 'html'; html: string } | { kind: 'code'; code: string; lang?: string })[] = []
    let pending: Token[] = []
    const flush = () => {
      if (pending.length) {
        const list = Object.assign(pending, { links: (tokens as unknown as { links: Tokens.Generic }).links ?? {} })
        out.push({ kind: 'html', html: marked.parser(list as unknown as Token[]) })
        pending = []
      }
    }
    for (const t of tokens) {
      if (t.type === 'code') {
        flush()
        out.push({ kind: 'code', code: (t as Tokens.Code).text, lang: (t as Tokens.Code).lang || undefined })
      } else {
        pending.push(t)
      }
    }
    flush()
    return out
  }, [text])

  return (
    <div className={`markdown ${className ?? ''}`}>
      {blocks.map((b, i) =>
        b.kind === 'html' ? (
          <div key={i} dangerouslySetInnerHTML={{ __html: b.html }} />
        ) : (
          <CodeBlock key={i} code={b.code} lang={b.lang} onOpen={onOpenSql} />
        ),
      )}
    </div>
  )
}
