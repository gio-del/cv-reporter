import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createSnippet } from '@/api/client'
import type { SnippetInput } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

const blankForm = { kind: '', tags: '', body: '' }
type FormState = typeof blankForm

export default function SnippetCreatePage() {
  const navigate = useNavigate()
  const [form, setForm] = useState<FormState>(blankForm)
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
      const input: SnippetInput = {
        kind: form.kind,
        tags: form.tags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
        body: form.body,
      }
      const created = await createSnippet(input)
      navigate(`/snippets/${created.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <p className="mb-4 inline-block text-sm">
        <Link to="/snippets" className="no-underline hover:underline">
          ← Back to Cover Letter Snippets
        </Link>
      </p>
      <h1>New Cover Letter Snippet</h1>
      <form onSubmit={handleSubmit}>
        {error && (
          <p role="alert" className="mb-4 font-medium text-destructive">
            {error}
          </p>
        )}

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="kind">Kind (e.g. opening, why-this-company, closing)</FieldLabel>
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
            {saving ? 'Creating…' : 'Create'}
          </Button>
        </div>
      </form>
    </>
  )
}
