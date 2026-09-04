import { useState } from 'react'
import type { ApplicationMethod, ApplicationMethodKind } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { cn } from '@/lib/utils'

const methodKindLabel: Record<ApplicationMethodKind, string> = {
  portal: 'Portal',
  email: 'Email',
  easy_apply: 'LinkedIn Easy Apply',
  other: 'Other',
  unresolved: "Couldn't check — retry",
}

// The kinds a user can correct to by hand — excludes `unresolved`, a
// system-set sentinel meaning inference couldn't even be attempted, not a
// real method to pick (story 15: the user can still correct straight to
// any of these at any time, unresolved or not).
const correctableMethodKinds: ApplicationMethodKind[] = ['portal', 'email', 'easy_apply', 'other']

// Displays the inferred Application Method and lets the user correct it if
// it's wrong (story 6) — Claude's inference at save time (story 5) is a
// guess, never trusted as final.
export default function ApplicationMethodEditor({
  method,
  onSave,
}: {
  method: ApplicationMethod
  onSave: (method: ApplicationMethod) => Promise<void>
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<ApplicationMethod>(method)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function startEditing() {
    setDraft(method)
    setError(null)
    setEditing(true)
  }

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      await onSave({ kind: draft.kind, value: draft.value?.trim() || undefined })
      setEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  if (!editing) {
    return (
      <p className="mb-0 text-sm">
        Apply via{' '}
        <strong className={cn('font-semibold', method.kind === 'unresolved' && 'text-unresolved')}>
          {methodKindLabel[method.kind]}
        </strong>
        {method.value && <>: {method.value}</>}
        {' · '}
        <button type="button" className="text-primary underline-offset-4 hover:underline" onClick={startEditing}>
          Correct
        </button>
      </p>
    )
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Select value={draft.kind} onValueChange={(value) => setDraft((d) => ({ ...d, kind: value as ApplicationMethodKind }))}>
        <SelectTrigger size="sm" aria-label="Application method">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {correctableMethodKinds.map((kind) => (
            <SelectItem key={kind} value={kind}>
              {methodKindLabel[kind]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        className="h-7 w-56"
        value={draft.value ?? ''}
        onChange={(e) => setDraft((d) => ({ ...d, value: e.target.value }))}
        placeholder="URL or email address"
      />
      <Button size="sm" onClick={handleSave} disabled={saving}>
        {saving ? 'Saving…' : 'Save'}
      </Button>
      <Button size="sm" variant="ghost" onClick={() => setEditing(false)} disabled={saving}>
        Cancel
      </Button>
      {error && (
        <p role="alert" className="mb-0 basis-full text-sm font-medium text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
