import { useEffect, useRef } from 'react'
import { EditorState, Compartment, StateEffect, StateField, type Extension } from '@codemirror/state'
import {
  EditorView,
  Decoration,
  type DecorationSet,
  keymap,
  lineNumbers,
  highlightActiveLine,
  highlightActiveLineGutter,
  drawSelection,
  placeholder as cmPlaceholder,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { syntaxHighlighting, HighlightStyle, bracketMatching, indentOnInput } from '@codemirror/language'
import { autocompletion, closeBrackets, closeBracketsKeymap, completionKeymap } from '@codemirror/autocomplete'
import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
import { sql, MySQL, type SQLNamespace } from '@codemirror/lang-sql'
import { tags as t } from '@lezer/highlight'

export interface EditorHandle {
  focus: () => void
  insert: (text: string) => void
  getRunnable: () => { sql: string; lineOffset: number }
  goToLine: (line: number) => void
}

interface Props {
  docKey: string
  value: string
  onChange: (value: string) => void
  onRun?: () => void
  onSave?: () => void
  schema?: SQLNamespace
  errorLine?: number | null
  placeholder?: string
  handleRef?: React.MutableRefObject<EditorHandle | null>
  ariaLabel?: string
  compact?: boolean
}

const highlight = HighlightStyle.define([
  { tag: [t.keyword, t.operatorKeyword, t.modifier], color: 'var(--syn-keyword)', fontWeight: '500' },
  { tag: [t.string, t.special(t.string)], color: 'var(--syn-string)' },
  { tag: [t.number, t.bool, t.null], color: 'var(--syn-number)' },
  { tag: [t.comment, t.lineComment, t.blockComment], color: 'var(--syn-comment)', fontStyle: 'italic' },
  { tag: [t.operator, t.punctuation, t.bracket], color: 'var(--syn-operator)' },
  { tag: [t.typeName, t.standard(t.name)], color: 'var(--syn-type)' },
  { tag: [t.function(t.variableName), t.function(t.name)], color: 'var(--syn-function)' },
  { tag: [t.name, t.variableName, t.propertyName, t.special(t.name)], color: 'var(--syn-name)' },
])

const theme = EditorView.theme({
  '&': { height: '100%', backgroundColor: 'transparent', color: 'var(--syn-name)', fontSize: '13.5px' },
  '&.cm-focused': { outline: 'none' },
  '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.65', overflow: 'auto' },
  '.cm-content': { caretColor: 'var(--primary)', padding: '12px 0' },
  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--primary)', borderLeftWidth: '2px' },
  '.cm-gutters': { backgroundColor: 'transparent', color: 'var(--muted-soft)', border: 'none', paddingLeft: '6px' },
  '.cm-lineNumbers .cm-gutterElement': { minWidth: '28px', paddingRight: '10px' },
  '.cm-activeLine': { backgroundColor: 'var(--editor-active-line)' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--body)' },
  '&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, ::selection': {
    backgroundColor: 'var(--editor-selection) !important',
  },
  '.cm-selectionMatch': { backgroundColor: 'var(--accent-wash)' },
  '.cm-matchingBracket': { backgroundColor: 'var(--accent-wash)', outline: '1px solid var(--hairline-strong)' },
  '.cm-placeholder': { color: 'var(--muted-soft)' },
  '.cm-error-line': { backgroundColor: 'var(--error-wash)', boxShadow: 'inset 2px 0 0 var(--error)' },
  '.cm-tooltip': {
    backgroundColor: 'var(--surface-elevated)',
    border: '1px solid var(--hairline-strong)',
    borderRadius: '8px',
    boxShadow: 'var(--shadow-pop)',
    overflow: 'hidden',
  },
  '.cm-tooltip-autocomplete > ul': { fontFamily: 'var(--font-mono)', fontSize: '12.5px', maxHeight: '260px' },
  '.cm-tooltip-autocomplete > ul > li': { padding: '3px 10px !important', color: 'var(--body)' },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': { backgroundColor: 'var(--accent-wash)', color: 'var(--ink)' },
  '.cm-completionDetail': { color: 'var(--muted)', fontStyle: 'normal', marginLeft: '10px' },
  '.cm-completionIcon': { opacity: '0.6' },
  '.cm-panels': { backgroundColor: 'var(--surface-card)', color: 'var(--body)', borderColor: 'var(--hairline)' },
  '.cm-panel input, .cm-panel button': { fontSize: '12px' },
  '.cm-searchMatch': { backgroundColor: 'var(--accent-wash)', outline: '1px solid var(--accent-ring)' },
})

const setErrorLine = StateEffect.define<number | null>()
const errorLineField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(deco, tr) {
    deco = deco.map(tr.changes)
    for (const e of tr.effects) {
      if (e.is(setErrorLine)) {
        if (e.value === null || e.value < 1 || e.value > tr.state.doc.lines) return Decoration.none
        const line = tr.state.doc.line(e.value)
        return Decoration.set([Decoration.line({ class: 'cm-error-line' }).range(line.from)])
      }
    }
    if (tr.docChanged) return Decoration.none // edits clear the marker
    return deco
  },
  provide: (f) => EditorView.decorations.from(f),
})

export function runnableText(state: EditorState) {
  const sel = state.selection.main
  if (!sel.empty) {
    return { sql: state.sliceDoc(sel.from, sel.to), lineOffset: state.doc.lineAt(sel.from).number - 1 }
  }
  return { sql: state.doc.toString(), lineOffset: 0 }
}

export function SqlEditor({ docKey, value, onChange, onRun, onSave, schema, errorLine, placeholder, handleRef, ariaLabel, compact }: Props) {
  const host = useRef<HTMLDivElement>(null)
  const view = useRef<EditorView | null>(null)
  const langConf = useRef(new Compartment())
  const callbacks = useRef({ onChange, onRun, onSave })
  callbacks.current = { onChange, onRun, onSave }

  const makeState = (doc: string, ns?: SQLNamespace) => {
    const extensions: Extension[] = [
      lineNumbers(),
      highlightActiveLineGutter(),
      history(),
      drawSelection(),
      indentOnInput(),
      bracketMatching(),
      closeBrackets(),
      autocompletion({ activateOnTyping: true, icons: false }),
      highlightActiveLine(),
      highlightSelectionMatches(),
      syntaxHighlighting(highlight),
      langConf.current.of(sql({ dialect: MySQL, schema: ns, upperCaseKeywords: true })),
      errorLineField,
      theme,
      EditorView.lineWrapping,
      EditorView.contentAttributes.of({ 'aria-label': ariaLabel ?? 'SQL editor' }),
      keymap.of([
        {
          key: 'Mod-Enter',
          preventDefault: true,
          run: () => {
            callbacks.current.onRun?.()
            return true
          },
        },
        {
          key: 'Mod-s',
          preventDefault: true,
          run: () => {
            callbacks.current.onSave?.()
            return true
          },
        },
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...searchKeymap,
        ...historyKeymap,
        ...completionKeymap,
        indentWithTab,
      ]),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) callbacks.current.onChange(u.state.doc.toString())
      }),
    ]
    if (placeholder) extensions.push(cmPlaceholder(placeholder))
    return EditorState.create({ doc, extensions })
  }

  // create once
  useEffect(() => {
    if (!host.current) return
    view.current = new EditorView({ state: makeState(value, schema), parent: host.current })
    return () => {
      view.current?.destroy()
      view.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // a different tab gets its own state (and undo history)
  const lastKey = useRef(docKey)
  useEffect(() => {
    const v = view.current
    if (!v) return
    if (lastKey.current !== docKey) {
      lastKey.current = docKey
      v.setState(makeState(value, schema))
      return
    }
    if (v.state.doc.toString() !== value) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [docKey, value])

  useEffect(() => {
    view.current?.dispatch({ effects: langConf.current.reconfigure(sql({ dialect: MySQL, schema, upperCaseKeywords: true })) })
  }, [schema])

  useEffect(() => {
    view.current?.dispatch({ effects: setErrorLine.of(errorLine ?? null) })
  }, [errorLine])

  useEffect(() => {
    if (!handleRef) return
    handleRef.current = {
      focus: () => view.current?.focus(),
      insert: (text: string) => {
        const v = view.current
        if (!v) return
        const sel = v.state.selection.main
        v.dispatch({ changes: { from: sel.from, to: sel.to, insert: text }, selection: { anchor: sel.from + text.length }, scrollIntoView: true })
        v.focus()
      },
      getRunnable: () => (view.current ? runnableText(view.current.state) : { sql: value, lineOffset: 0 }),
      goToLine: (n: number) => {
        const v = view.current
        if (!v || n < 1 || n > v.state.doc.lines) return
        const line = v.state.doc.line(n)
        v.dispatch({ selection: { anchor: line.from, head: line.to }, scrollIntoView: true })
        v.focus()
      },
    }
  })

  return <div ref={host} className={`sql-editor ${compact ? 'compact' : ''}`} />
}
