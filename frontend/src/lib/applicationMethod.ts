import type { ApplicationMethodKind } from '@/api/types'

// Application Method display labels, shared by the Method editor on the Job
// Listing page and the Applications view's rows (issue #95). Kept out of
// ApplicationMethodEditor.tsx so that file exports only a component.
export const methodKindLabel: Record<ApplicationMethodKind, string> = {
  portal: 'Portal',
  email: 'Email',
  easy_apply: 'LinkedIn Easy Apply',
  other: 'Other',
  unresolved: "Couldn't check — retry",
}
