import { useState } from 'react'
import { updateEntry } from '@/api/client'
import type { Entry, EntryInput } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

function toFormState(entry: Entry) {
  return {
    employer: entry.employer ?? '',
    client: entry.client ?? '',
    role: entry.role ?? '',
    name: entry.name ?? '',
    location: entry.location ?? '',
    start: entry.start,
    end: entry.end ?? '',
    flagship: entry.flagship ?? false,
    tags: entry.tags.join(', '),
    repo: entry.repo ?? '',
    bullets: (entry.bullets ?? []).join('\n'),
  }
}

type FormState = ReturnType<typeof toFormState>

function toEntryInput(entry: Entry, form: FormState): EntryInput {
  return {
    type: entry.type,
    employer: form.employer,
    client: form.client,
    role: form.role,
    name: form.name,
    location: form.location,
    start: form.start,
    end: form.end === '' ? null : form.end,
    flagship: form.flagship,
    tags: form.tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean),
    repo: form.repo,
    bullets: form.bullets
      .split('\n')
      .map((b) => b.trim())
      .filter(Boolean),
  }
}

export default function EntryEditForm({
  entry,
  onSaved,
  onCancel,
}: {
  entry: Entry
  onSaved: (entry: Entry) => void
  onCancel: () => void
}) {
  const [form, setForm] = useState<FormState>(toFormState(entry))
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const saved = await updateEntry(entry.id, toEntryInput(entry, form))
      onSaved(saved)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      {error && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {error}
        </p>
      )}

      <FieldGroup>
        {entry.type === 'experience' ? (
          <>
            <Field>
              <FieldLabel htmlFor="employer">Employer</FieldLabel>
              <Input id="employer" value={form.employer} onChange={(e) => set('employer', e.target.value)} />
            </Field>
            <Field>
              <FieldLabel htmlFor="client">Client</FieldLabel>
              <Input id="client" value={form.client} onChange={(e) => set('client', e.target.value)} />
            </Field>
            <Field>
              <FieldLabel htmlFor="role">Role</FieldLabel>
              <Input id="role" value={form.role} onChange={(e) => set('role', e.target.value)} />
            </Field>
            <Field>
              <FieldLabel htmlFor="location">Location</FieldLabel>
              <Input id="location" value={form.location} onChange={(e) => set('location', e.target.value)} />
            </Field>
            <Field orientation="horizontal">
              <Checkbox
                id="flagship"
                checked={form.flagship}
                onCheckedChange={(checked) => set('flagship', checked === true)}
              />
              <FieldLabel htmlFor="flagship" className="font-normal">
                Flagship
              </FieldLabel>
            </Field>
          </>
        ) : (
          <>
            <Field>
              <FieldLabel htmlFor="name">Name</FieldLabel>
              <Input id="name" value={form.name} onChange={(e) => set('name', e.target.value)} />
            </Field>
            <Field>
              <FieldLabel htmlFor="repo">Repo</FieldLabel>
              <Input id="repo" value={form.repo} onChange={(e) => set('repo', e.target.value)} />
            </Field>
          </>
        )}

        <Field>
          <FieldLabel htmlFor="start">Start (YYYY or YYYY-MM)</FieldLabel>
          <Input id="start" value={form.start} onChange={(e) => set('start', e.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="end">End (YYYY or YYYY-MM, blank for present)</FieldLabel>
          <Input id="end" value={form.end} onChange={(e) => set('end', e.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="tags">Tags (comma-separated)</FieldLabel>
          <Input id="tags" value={form.tags} onChange={(e) => set('tags', e.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="bullets">Bullets (one per line)</FieldLabel>
          <Textarea id="bullets" rows={6} value={form.bullets} onChange={(e) => set('bullets', e.target.value)} />
        </Field>
      </FieldGroup>

      <div className="mt-6 flex gap-3">
        <Button type="submit" disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
          Cancel
        </Button>
      </div>
    </form>
  )
}
