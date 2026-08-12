import { useState } from 'react'
import { updateEntry } from '../api/client'
import type { Entry, EntryInput } from '../api/types'

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
      {error && <p role="alert">{error}</p>}

      {entry.type === 'experience' ? (
        <>
          <label>
            Employer
            <input value={form.employer} onChange={(e) => set('employer', e.target.value)} />
          </label>
          <label>
            Client
            <input value={form.client} onChange={(e) => set('client', e.target.value)} />
          </label>
          <label>
            Role
            <input value={form.role} onChange={(e) => set('role', e.target.value)} />
          </label>
          <label>
            Location
            <input value={form.location} onChange={(e) => set('location', e.target.value)} />
          </label>
          <label>
            <input type="checkbox" checked={form.flagship} onChange={(e) => set('flagship', e.target.checked)} />
            Flagship
          </label>
        </>
      ) : (
        <>
          <label>
            Name
            <input value={form.name} onChange={(e) => set('name', e.target.value)} />
          </label>
          <label>
            Repo
            <input value={form.repo} onChange={(e) => set('repo', e.target.value)} />
          </label>
        </>
      )}

      <label>
        Start (YYYY or YYYY-MM)
        <input value={form.start} onChange={(e) => set('start', e.target.value)} />
      </label>
      <label>
        End (YYYY or YYYY-MM, blank for present)
        <input value={form.end} onChange={(e) => set('end', e.target.value)} />
      </label>
      <label>
        Tags (comma-separated)
        <input value={form.tags} onChange={(e) => set('tags', e.target.value)} />
      </label>
      <label>
        Bullets (one per line)
        <textarea rows={6} value={form.bullets} onChange={(e) => set('bullets', e.target.value)} />
      </label>

      <div>
        <button type="submit" disabled={saving}>
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button type="button" onClick={onCancel} disabled={saving}>
          Cancel
        </button>
      </div>
    </form>
  )
}
