/**
 * A small, forgiving reader for the SQL statement being edited, used for autocompletion.
 * It finds what a statement defines for itself: CTEs (WITH name AS (...)), derived tables
 * (FROM (SELECT ...) AS t), table aliases (FROM orders o) and column aliases
 * (SELECT ... AS revenue), at any depth. It never rejects input: half-typed SQL gives
 * whatever could be read.
 */

export interface Column {
  name: string
  type?: string
}

/** Tables of the database, by lowercase name. */
export type TableColumns = Map<string, Column[]>

interface Tok {
  text: string // identifier text without backticks
  up: string // upper case, for keyword tests
  kind: 'word' | 'quoted' | 'string' | 'number' | 'punct'
  from: number
  depth: number // parenthesis depth the token sits at
}

export interface Source {
  name: string // what the statement calls it: an alias, or the table / CTE name
  table: string // the table or CTE it reads, or '' for a derived table
  kind: 'table' | 'cte' | 'derived'
  columns: Column[]
}

export interface Scope {
  ctes: Map<string, { name: string; columns: Column[] }> // lowercase name → the CTE as written
  sources: Source[] // every table, CTE and derived table the statement reads from
  columnAliases: string[] // names given in SELECT lists
}

const ENDS_FROM = new Set(['WHERE', 'GROUP', 'HAVING', 'ORDER', 'LIMIT', 'UNION', 'EXCEPT', 'INTERSECT', 'WINDOW', 'SET', 'VALUES', 'SELECT', 'FOR', 'LOCK', 'RETURNING'])
const NOT_ALIAS = new Set([
  ...ENDS_FROM, 'ON', 'USING', 'JOIN', 'INNER', 'LEFT', 'RIGHT', 'FULL', 'CROSS', 'NATURAL', 'OUTER', 'STRAIGHT_JOIN',
  'AS', 'USE', 'FORCE', 'IGNORE', 'PARTITION', 'LATERAL', 'WITH', 'AND', 'OR', 'NOT',
])
const SELECT_ENDS = new Set(['FROM', 'INTO', 'WHERE', 'GROUP', 'HAVING', 'ORDER', 'LIMIT', 'UNION', 'EXCEPT', 'INTERSECT', 'WINDOW', 'FOR'])
const SELECT_MODIFIERS = new Set(['DISTINCT', 'ALL', 'DISTINCTROW', 'HIGH_PRIORITY', 'STRAIGHT_JOIN', 'SQL_NO_CACHE', 'SQL_CALC_FOUND_ROWS', 'SQL_SMALL_RESULT', 'SQL_BIG_RESULT'])
const NOT_A_NAME = new Set(['END', 'NULL', 'TRUE', 'FALSE', 'DESC', 'ASC', 'THEN', 'ELSE', 'AND', 'OR', 'NOT', 'IS', 'IN', 'LIKE', 'BETWEEN', 'OVER', 'CASE', 'WHEN', 'AS', 'BY', 'ON'])

/** Splits SQL into tokens, skipping comments and whitespace. */
export function tokenize(sql: string): Tok[] {
  const out: Tok[] = []
  let depth = 0
  let i = 0
  const n = sql.length
  while (i < n) {
    const c = sql[i]
    if (c === ' ' || c === '\t' || c === '\n' || c === '\r') {
      i++
    } else if ((c === '-' && sql[i + 1] === '-' && (i + 2 >= n || /\s/.test(sql[i + 2]))) || c === '#') {
      while (i < n && sql[i] !== '\n') i++
    } else if (c === '/' && sql[i + 1] === '*') {
      const end = sql.indexOf('*/', i + 2)
      i = end < 0 ? n : end + 2
    } else if (c === "'" || c === '"') {
      const start = i++
      while (i < n && sql[i] !== c) i += sql[i] === '\\' ? 2 : 1
      i++
      out.push({ text: sql.slice(start, i), up: '', kind: 'string', from: start, depth })
    } else if (c === '`') {
      const start = i++
      while (i < n && sql[i] !== '`') i++
      const text = sql.slice(start + 1, i)
      i++
      out.push({ text, up: text.toUpperCase(), kind: 'quoted', from: start, depth })
    } else if (/[A-Za-z_$@]/.test(c)) {
      const start = i
      while (i < n && /[\w$@]/.test(sql[i])) i++
      const text = sql.slice(start, i)
      out.push({ text, up: text.toUpperCase(), kind: 'word', from: start, depth })
    } else if (/[0-9]/.test(c)) {
      const start = i
      while (i < n && /[\w.]/.test(sql[i])) i++
      out.push({ text: sql.slice(start, i), up: '', kind: 'number', from: start, depth })
    } else {
      if (c === ')') depth = Math.max(0, depth - 1)
      out.push({ text: c, up: c, kind: 'punct', from: i, depth })
      if (c === '(') depth++
      i++
    }
  }
  return out
}

const isName = (t?: Tok) => !!t && (t.kind === 'quoted' || (t.kind === 'word' && !t.text.startsWith('@')))

function matching(ts: Tok[], open: number): number {
  const d = ts[open].depth
  for (let i = open + 1; i < ts.length; i++) if (ts[i].text === ')' && ts[i].depth === d && ts[i].kind === 'punct') return i
  return ts.length
}

/** The tokens of the statement around `pos` (statements end at a top-level `;`). */
function statementAt(ts: Tok[], pos: number): Tok[] {
  let start = 0
  for (let i = 0; i < ts.length; i++) {
    if (ts[i].text === ';' && ts[i].kind === 'punct' && ts[i].depth === 0) {
      if (ts[i].from >= pos) return ts.slice(start, i)
      start = i + 1
    }
  }
  return ts.slice(start)
}

type Item = { name?: string; alias?: boolean; star?: string | true }

/** The output columns of the first SELECT in ts[from..to): names, plus `*` / `t.*` markers. */
function selectList(ts: Tok[], from: number, to: number): Item[] {
  const base = ts[from]?.depth ?? 0
  let i = from
  while (i < to && !(ts[i].up === 'SELECT' && ts[i].depth === base)) i++
  if (i >= to) return []
  i++
  while (i < to && SELECT_MODIFIERS.has(ts[i].up) && ts[i].depth === base) i++
  const items: Item[] = []
  let cur: Tok[] = []
  const flush = () => {
    if (cur.length) items.push(itemName(cur))
    cur = []
  }
  for (; i < to; i++) {
    const t = ts[i]
    if (t.depth === base && t.kind === 'word' && SELECT_ENDS.has(t.up)) break
    if (t.depth === base && t.text === ',' && t.kind === 'punct') flush()
    else cur.push(t)
  }
  flush()
  return items
}

function itemName(item: Tok[]): Item {
  const last = item[item.length - 1]
  const prev = item[item.length - 2]
  if (last.text === '*' && last.kind === 'punct') {
    return prev?.text === '.' && isName(item[item.length - 3]) ? { star: item[item.length - 3].text } : { star: true }
  }
  if (!isName(last) || (last.kind === 'word' && NOT_A_NAME.has(last.up))) return {}
  if (item.length === 1 || prev.text === '.') return { name: last.text } // col, or t.col
  if (prev.kind === 'word' && prev.up === 'AS') return { name: last.text, alias: true } // expr AS name
  if (prev.text === ')' || prev.kind === 'number' || prev.kind === 'string' || isName(prev)) return { name: last.text, alias: true } // expr name
  return {}
}

/** Reads a statement's CTEs, sources and column aliases. */
export function analyze(sql: string, pos: number, tables: TableColumns): Scope {
  const ts = statementAt(tokenize(sql), pos)
  const ctes = new Map<string, { name: string; columns: Column[] }>()
  const sources: Source[] = []
  const columnAliases = new Set<string>()

  const columnsOf = (name: string): Column[] | undefined => ctes.get(name.toLowerCase())?.columns ?? tables.get(name.toLowerCase())

  // Resolves a select list to columns; `*` and `t.*` expand to the columns of the
  // tables that query reads (one level deep), when they are known.
  const resolve = (from: number, to: number): Column[] => {
    const items = selectList(ts, from, to)
    const out: Column[] = []
    let local: Source[] | null = null
    for (const it of items) {
      if (it.name) out.push({ name: it.name })
      else if (it.star) {
        local ??= readSources(from, to, false)
        for (const s of local) if (it.star === true || s.name.toLowerCase() === it.star.toLowerCase()) out.push(...s.columns)
      }
    }
    return out
  }

  // Tables after FROM / JOIN / UPDATE / INTO (and after commas in a FROM list).
  const readSources = (from: number, to: number, record: boolean): Source[] => {
    const found: Source[] = []
    const inFrom = new Map<number, boolean>()
    for (let i = from; i < to; i++) {
      const t = ts[i]
      const opens = t.kind === 'word' && (t.up === 'FROM' || t.up === 'JOIN' || t.up === 'UPDATE' || t.up === 'INTO')
      if (t.kind === 'word' && t.up === 'FROM') inFrom.set(t.depth, true)
      else if (t.kind === 'word' && ENDS_FROM.has(t.up)) inFrom.set(t.depth, false)
      const comma = t.text === ',' && t.kind === 'punct' && inFrom.get(t.depth)
      if (!opens && !comma) continue
      let j = i + 1
      if (ts[j]?.up === 'LATERAL') j++
      if (ts[j]?.text === '(') {
        const close = matching(ts, j)
        const inner = ts[j + 1]?.up === 'SELECT' || ts[j + 1]?.up === 'WITH'
        let k = close + 1
        if (ts[k]?.up === 'AS') k++
        if (inner && isName(ts[k]) && !NOT_ALIAS.has(ts[k].up)) {
          found.push({ name: ts[k].text, table: '', kind: 'derived', columns: resolve(j + 1, close) })
        }
        continue
      }
      if (!isName(ts[j]) || NOT_ALIAS.has(ts[j].up)) continue
      let table = ts[j].text
      let k = j + 1
      if (ts[k]?.text === '.' && isName(ts[k + 1])) {
        table = ts[k + 1].text // db.table
        k += 2
      }
      const isCte = ctes.has(table.toLowerCase())
      let alias = table
      if (ts[k]?.up === 'AS' && isName(ts[k + 1])) alias = ts[k + 1].text
      else if (isName(ts[k]) && !NOT_ALIAS.has(ts[k].up)) alias = ts[k].text
      found.push({ name: alias, table, kind: isCte ? 'cte' : 'table', columns: columnsOf(table) ?? [] })
    }
    if (record) sources.push(...found)
    return found
  }

  // CTEs, in order: a later CTE may read an earlier one.
  for (let i = 0; i < ts.length; i++) {
    if (ts[i].up !== 'WITH' || ts[i].kind !== 'word') continue
    let j = i + 1
    if (ts[j]?.up === 'RECURSIVE') j++
    while (isName(ts[j])) {
      const name = ts[j].text
      let k = j + 1
      let explicit: Column[] | null = null
      if (ts[k]?.text === '(') {
        const close = matching(ts, k)
        explicit = ts.slice(k + 1, close).filter((t) => isName(t)).map((t) => ({ name: t.text }))
        k = close + 1
      }
      if (ts[k]?.up !== 'AS' || ts[k + 1]?.text !== '(') break
      const open = k + 1
      const close = matching(ts, open)
      ctes.set(name.toLowerCase(), { name, columns: [] }) // a recursive CTE can refer to itself
      ctes.set(name.toLowerCase(), { name, columns: explicit ?? resolve(open + 1, close) })
      if (ts[close + 1]?.text !== ',') break
      j = close + 2
    }
  }

  readSources(0, ts.length, true)

  // Column aliases from every SELECT list in the statement.
  for (let i = 0; i < ts.length; i++) {
    if (ts[i].up !== 'SELECT' || ts[i].kind !== 'word') continue
    let end = i + 1
    while (end < ts.length && !(ts[end].depth < ts[i].depth)) end++
    for (const it of selectList(ts, i, end)) if (it.alias && it.name) columnAliases.add(it.name)
  }

  return { ctes, sources, columnAliases: [...columnAliases] }
}
