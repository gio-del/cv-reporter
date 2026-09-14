// Type-level contract check between a golden API response fixture and the
// frontend type that declares it (issue #99). Nothing here executes: a
// fixture JSON file imported as a type is walked field by field against the
// declared type, and every disagreement becomes a string literal naming the
// JSON path. Contract<...> is `true` when there is none, so
//
//   export const x: Contract<typeof fixture, Declared, 'GET /api/x'> = true
//
// fails `tsc -b` with an error listing each offending path.
//
// Plain assignment (`const f: Declared = fixture`) is not enough: excess
// property checks never apply to imported values, so a field the backend
// sends and the frontend never declared would pass silently.
//
// Checked in every mode:
//   - a key sent but not declared,
//   - a required key not sent (in any element of an array),
//   - a value of the wrong kind (string/number/boolean/array/object),
//   - null sent where the declared type is not nullable.
// Additionally, per fixture:
//   - 'populated': every optional key is sent somewhere (so the frontend
//     declares everything that can arrive, not just the common case),
//   - 'sparse': no optional key is sent (so a field the backend always sends
//     is declared required, not optional).
// Exempt paths (X) switch the populated/sparse extra checks off for a key
// the scenario cannot reach; they never relax the base checks.
//
// String-literal unions (ApplicationStatus, ...) are compared as strings:
// JSON module imports widen every value to its primitive type, so only the
// shape is checked, never the literal values.

export type FixtureMode = 'shape' | 'populated' | 'sparse'

type Kind<V> = V extends string
  ? 'string'
  : V extends number
    ? 'number'
    : V extends boolean
      ? 'boolean'
      : V extends readonly unknown[]
        ? 'array'
        : V extends object
          ? 'object'
          : 'unknown'

// TypeScript normalizes the object literals of one JSON array to share their
// keys, giving `key?: undefined` to an element lacking one, so only keys with
// a non-undefined type are actually sent.
type Present<F> = {
  [K in keyof F]-?: [Exclude<F[K], undefined>] extends [never] ? never : K
}[keyof F]

type PresentInSome<F> = F extends unknown ? Present<F> : never
type MissingInSome<F, Keys> = F extends unknown ? Exclude<Keys, Present<F>> : never
type PresentInAll<F> = Exclude<PresentInSome<F>, MissingInSome<F, PresentInSome<F>>>

type RequiredKeys<T> = { [K in keyof T]-?: object extends Pick<T, K> ? never : K }[keyof T]
type OptionalKeys<T> = Exclude<keyof T, RequiredKeys<T>>

type Prop<F, K> = F extends unknown ? (K extends keyof F ? F[K] : never) : never
type Elem<F> = F extends readonly (infer E)[] ? E : never
type Join<P extends string, K> = `${P}.${K & string}`

type Check<F, T, P extends string, M extends FixtureMode, X> = [Exclude<F, undefined>] extends [never]
  ? never
  :
      | (null extends F ? (null extends T ? never : `${P} is sent as null but not declared nullable`) : never)
      | ValueCheck<Exclude<F, null | undefined>, Exclude<T, null | undefined>, P, M, X>

type KindErrors<F, T, P extends string> =
  Kind<F> extends infer K
    ? K extends string
      ? K extends Kind<T>
        ? never
        : `${P} is sent as ${K}, which its declared type does not allow`
      : never
    : never

type ObjectOf<V> = Exclude<Extract<V, object>, readonly unknown[]>
type ArrayOf<V> = Extract<V, readonly unknown[]>

type ValueCheck<F, T, P extends string, M extends FixtureMode, X> = [F] extends [never]
  ? never
  :
      | KindErrors<F, T, P>
      | ([ArrayOf<F>] extends [never]
          ? never
          : [ArrayOf<T>] extends [never]
            ? never
            : Check<Elem<ArrayOf<F>>, Elem<ArrayOf<T>>, `${P}[]`, M, X>)
      | ([ObjectOf<F>] extends [never]
          ? never
          : [ObjectOf<T>] extends [never]
            ? never
            : ObjectCheck<ObjectOf<F>, ObjectOf<T>, P, M, X>)

type ObjectCheck<F, T, P extends string, M extends FixtureMode, X> =
  | {
      [K in PresentInSome<F>]: K extends keyof T
        ? Check<Prop<F, K>, T[K], Join<P, K>, M, X>
        : `${Join<P, K>} is sent but not declared`
    }[PresentInSome<F>]
  | {
      [K in Exclude<RequiredKeys<T>, PresentInAll<F>>]: `${Join<P, K>} is declared required but not sent`
    }[Exclude<RequiredKeys<T>, PresentInAll<F>>]
  | (M extends 'populated'
      ? {
          [K in Exclude<OptionalKeys<T>, PresentInSome<F>>]: Join<P, K> extends X
            ? never
            : `${Join<P, K>} is declared optional but absent from the populated fixture`
        }[Exclude<OptionalKeys<T>, PresentInSome<F>>]
      : never)
  | (M extends 'sparse'
      ? {
          [K in Extract<OptionalKeys<T>, PresentInSome<F>>]: Join<P, K> extends X
            ? never
            : `${Join<P, K>} is declared optional but sent in the sparse fixture`
        }[Extract<OptionalKeys<T>, PresentInSome<F>>]
      : never)

/**
 * Contract is `true` when fixture F matches declared type T, and otherwise a
 * union of messages, each prefixed with label (the endpoint and variant),
 * naming a JSON path (`$` is the response body) and what is wrong with it.
 */
export type Contract<
  F,
  T,
  Label extends string,
  M extends FixtureMode = 'shape',
  X extends string = never,
> = [Check<F, T, '$', M, X>] extends [never] ? true : `${Label}: ${Check<F, T, '$', M, X>}`
