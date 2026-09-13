import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// ConflictAlert is what a write refused with 409 renders (issue #89): the
// record changed on disk since it was read, so the save or delete was not
// applied. It is deliberately distinct from a validation error — the
// user's input is fine — and names the likely causes, the tailor-cv skill
// first since in this app it is the most common one.
//
// Reload is always an explicit click. The form behind this alert keeps
// every character the user typed until they choose to reload; nothing is
// merged and nothing is refreshed on their behalf.
export default function ConflictAlert({
  record,
  action = 'save',
  keepsEdits = true,
  onReload,
  reloading = false,
  className,
}: {
  // record names what changed, e.g. "Entry", "Profile", "Application".
  record: string
  action?: 'save' | 'delete' | 'change'
  // keepsEdits is false where there is no form holding typed input (a
  // delete, a one-click Status change), so the alert doesn't promise
  // edits that don't exist.
  keepsEdits?: boolean
  onReload: () => void
  reloading?: boolean
  className?: string
}) {
  return (
    <div
      role="alert"
      className={cn('rounded-xl border border-destructive/40 bg-destructive/5 p-4 text-sm', className ?? 'mb-4')}
    >
      <p className="mb-1 font-semibold text-destructive">This {record} changed on disk since you opened it.</p>
      <p className="mb-3">
        Your {action} was not applied. The <code>tailor-cv</code> skill, another browser tab, or a hand edit may have
        changed it.{' '}
        {keepsEdits
          ? 'Your edits are still here — reloading replaces them with the current version, so copy anything you want to keep first.'
          : 'Reload to see the current version before trying again.'}
      </p>
      <Button type="button" size="sm" variant="outline" onClick={onReload} disabled={reloading}>
        {reloading ? 'Reloading…' : 'Reload the current version'}
      </Button>
    </div>
  )
}
