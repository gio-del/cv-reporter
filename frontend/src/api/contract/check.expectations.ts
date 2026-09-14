import type { Contract } from './check'

// The contract check's own expectations, compiled by `tsc -b` like the real
// assertions in contract.ts: every `@ts-expect-error` below must still be an
// error, so a check that silently stopped catching a drift class fails the
// build (issue #99's "a contract test that has never been seen to fail").
// Fixture types are written the way a JSON module import infers them.

interface Declared {
  id: string
  status: 'saved' | 'sent'
  count: number
  end: string | null
  note?: string
  tags: string[]
  nested: { at: string; by?: string }
}

type Matching = {
  id: string
  status: string
  count: number
  end: null
  note: string
  tags: never[]
  nested: { at: string; by: string }
}

export const matches: Contract<Matching, Declared, 'matching'> = true
export const populatedMatches: Contract<Matching, Declared, 'populated', 'populated'> = true

// A field the backend sends that the frontend never declared.
// @ts-expect-error $.extra is sent but not declared
export const undeclared: Contract<Matching & { extra: string }, Declared, 'undeclared'> = true

// A field declared required that the backend stopped sending (or renamed).
// @ts-expect-error $.count is declared required but not sent
export const removed: Contract<Omit<Matching, 'count'>, Declared, 'removed'> = true

// A renamed field is reported on both sides at once.
// @ts-expect-error $.total is sent but not declared | $.count is declared required but not sent
export const renamed: Contract<Omit<Matching, 'count'> & { total: number }, Declared, 'renamed'> = true

// A field whose type changed: string to number, scalar to array, value to null.
// @ts-expect-error $.id is sent as number
export const retyped: Contract<Omit<Matching, 'id'> & { id: number }, Declared, 'retyped'> = true
// @ts-expect-error $.count is sent as array
export const toArray: Contract<Omit<Matching, 'count'> & { count: number[] }, Declared, 'to array'> = true
// @ts-expect-error $.id is sent as null but not declared nullable
export const toNull: Contract<Omit<Matching, 'id'> & { id: null }, Declared, 'to null'> = true

// Nested objects and array elements are walked too.
// @ts-expect-error $.nested.extra is sent but not declared
export const nestedExtra: Contract<Omit<Matching, 'nested'> & { nested: { at: string; extra: string } }, Declared, 'nested'> =
  true
// @ts-expect-error $.tags[] is sent as number
export const elementRetyped: Contract<Omit<Matching, 'tags'> & { tags: number[] }, Declared, 'element'> = true

// Array elements are normalized to share keys; a required key missing from
// only one element is still reported.
type Row = { id: string; label?: string }
export const rowsCoverOptional: Contract<({ id: string; label?: undefined } | { id: string; label: string })[], Row[], 'rows', 'populated'> =
  true
// @ts-expect-error $[].id is declared required but not sent
export const rowMissingRequired: Contract<({ id: string; label?: undefined } | { id?: undefined; label: string })[], Row[], 'rows'> = true

// Optionality flips. Populated: an optional field never seen means the
// frontend declares something the backend does not send.
// @ts-expect-error $.note is declared optional but absent from the populated fixture
export const neverSent: Contract<Omit<Matching, 'note'>, Declared, 'populated', 'populated'> = true
export const neverSentExempt: Contract<Omit<Matching, 'note'>, Declared, 'populated', 'populated', '$.note'> = true

// Sparse: an optional field that is sent anyway means the backend always
// sends it, so the frontend should declare it required.
type Sparse = Omit<Matching, 'note' | 'nested'> & { nested: { at: string } }
export const sparseMatches: Contract<Sparse, Declared, 'sparse', 'sparse'> = true
// @ts-expect-error $.note is declared optional but sent in the sparse fixture
export const alwaysSent: Contract<Sparse & { note: string }, Declared, 'sparse', 'sparse'> = true
