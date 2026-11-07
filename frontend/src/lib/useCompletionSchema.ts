import { useEffect, useState } from 'react'
import { api, type Target } from './api'
import { completionSchema, type CompletionSchema } from './sqlComplete'

// One request per database and visitor for the whole visit: every exercise editor shares it.
const cache = new Map<string, Promise<CompletionSchema>>()

function load(target: Target, visitor: string): Promise<CompletionSchema> {
  const key = `${target}:${visitor}`
  let p = cache.get(key)
  if (!p) {
    p = api.schema(target).then(completionSchema)
    p.catch(() => cache.delete(key)) // try again next time
    cache.set(key, p)
  }
  return p
}

/** Tables and columns for an editor's autocompletion; undefined until they've loaded. */
export function useCompletionSchema(target: Target, visitor: string): CompletionSchema | undefined {
  const [schema, setSchema] = useState<CompletionSchema>()
  useEffect(() => {
    let live = true
    load(target, visitor)
      .then((s) => live && setSchema(s))
      .catch(() => {})
    return () => {
      live = false
    }
  }, [target, visitor])
  return schema
}
