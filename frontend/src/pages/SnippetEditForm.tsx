import { useState } from 'react'
import { updateSnippet } from '@/api/client'
import type { Snippet, SnippetInput } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

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
      {error && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {error}
        </p>
      )}

      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="kind">Kind</FieldLabel>
          <Input id="kind" value={form.kind} onChange={(e) => set('kind', e.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="tags">Tags (comma-separated)</FieldLabel>
          <Input id="tags" value={form.tags} onChange={(e) => set('tags', e.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="body">Body</FieldLabel>
          <Textarea id="body" rows={8} value={form.body} onChange={(e) => set('body', e.target.value)} />
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
