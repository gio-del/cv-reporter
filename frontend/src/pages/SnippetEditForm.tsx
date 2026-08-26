import { useState } from 'react'
import { updateSnippet } from '../api/client'
import type { Snippet, SnippetInput } from '../api/types'

function toFormState(snippet: Snippet) {
  return { kind: snippet.kind, tags: snippet.tags.join(', '), body: snippet.body }
}

type FormState = ReturnType<typeof toFormState>

function toSnippetInput(form: FormState): SnippetInput {
  return {
    kind: form.kind,
    tags: form.tags
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean),
    body: form.body,
  }
}

export default function SnippetEditForm({
  snippet,
  onSaved,
  onCancel,
}: {
  snippet: Snippet
  onSaved: (snippet: Snippet) => void
  onCancel: () => void
}) {
  const [form, setForm] = useState<FormState>(toFormState(snippet))
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
      const saved = await updateSnippet(snippet.id, toSnippetInput(form))
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

      <label>
        Kind
        <input value={form.kind} onChange={(e) => set('kind', e.target.value)} />
      </label>
      <label>
        Tags (comma-separated)
        <input value={form.tags} onChange={(e) => set('tags', e.target.value)} />
      </label>
      <label>
        Body
        <textarea rows={8} value={form.body} onChange={(e) => set('body', e.target.value)} />
      </label>

      <div className="form-actions">
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
