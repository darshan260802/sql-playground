import type { Completion, CompletionContext, CompletionResult, CompletionSource } from '@codemirror/autocomplete'
import { syntaxTree } from '@codemirror/language'
import type { SchemaInfo } from './api'
import { analyze, type Column, type TableColumns } from './sqlScope'

/** What the editor knows about the database: its tables and their columns, and its name. */
export interface CompletionSchema {
  tables: TableColumns
  databases: string[]
}

export function completionSchema(info: SchemaInfo): CompletionSchema {
  const tables: TableColumns = new Map()
  for (const t of info.tables) tables.set(t.name.toLowerCase(), t.columns.map((c) => ({ name: c.name, type: c.type })))
  return { tables, databases: [...new Set([info.schema, 'core'])] }
}

const WORD = /[\w$]*$/
const QUALIFIER = /(`[^`]+`|[\w$]+)\s*\.\s*$/

const columnOption = (c: Column, detail: string | undefined, boost: number): Completion => ({
  label: c.name,
  type: 'property',
  detail: detail ?? c.type,
  boost,
})

/**
 * Completes table and column names, including what the statement defines for itself:
 * CTE names and their columns, derived tables, table aliases (`o.` lists the columns of
 * `orders o`) and column aliases from the SELECT list. Keywords come from lang-sql.
 */
export function scopeCompletions(getSchema: () => CompletionSchema | undefined): CompletionSource {
  return (ctx: CompletionContext): CompletionResult | null => {
    const node = syntaxTree(ctx.state).resolveInner(ctx.pos, -1)
    if (/String|Comment/.test(node.name)) return null

    const schema = getSchema() ?? { tables: new Map(), databases: [] }
    const word = ctx.matchBefore(WORD)
    const from = word ? word.from : ctx.pos
    const before = ctx.state.doc.sliceString(Math.max(0, from - 200), from)
    const qualified = QUALIFIER.exec(before)
    const scope = analyze(ctx.state.doc.toString(), ctx.pos, schema.tables)

    // `something.`: columns of a table, CTE, derived table or alias, or tables of a database
    if (qualified) {
      const q = qualified[1].replace(/`/g, '').toLowerCase()
      const source = scope.sources.find((s) => s.name.toLowerCase() === q)
      const columns = source?.columns ?? scope.ctes.get(q)?.columns ?? schema.tables.get(q)
      if (columns) {
        return { from, options: columns.map((c: Column) => columnOption(c, source && source.kind !== 'table' ? source.table || 'derived' : undefined, 2)), validFor: /^[\w$]*$/ }
      }
      if (schema.databases.some((d) => d.toLowerCase() === q)) {
        return { from, options: [...schema.tables.keys()].map((t) => ({ label: t, type: 'type', detail: 'table' })), validFor: /^[\w$]*$/ }
      }
      return null
    }

    if (!word || (word.from === word.to && !ctx.explicit)) return null

    const options: Completion[] = []
    const seen = new Set<string>()
    const add = (o: Completion) => {
      const key = o.label.toLowerCase() + '|' + (o.detail ?? '')
      if (seen.has(key)) return
      seen.add(key)
      options.push(o)
    }
    for (const a of scope.columnAliases) add({ label: a, type: 'variable', detail: 'column alias', boost: 3 })
    // columns of the tables this statement reads, labelled with where they come from
    for (const s of scope.sources) for (const c of s.columns) add(columnOption(c, s.name, 2))
    for (const s of scope.sources) if (s.name !== s.table) add({ label: s.name, type: 'variable', detail: s.table ? `alias of ${s.table}` : 'derived table', boost: 1 })
    for (const cte of scope.ctes.values()) add({ label: cte.name, type: 'type', detail: `CTE · ${cte.columns.length} columns`, boost: 1 })
    for (const t of schema.tables.keys()) add({ label: t, type: 'type', detail: 'table' })
    return { from, options, validFor: /^[\w$]*$/ }
  }
}
