import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createSnippet } from '../api/client'
import type { SnippetInput } from '../api/types'

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
      <p className="breadcrumb">
        <Link to="/snippets">← Back to Cover Letter Snippets</Link>
      </p>
      <h1>New Cover Letter Snippet</h1>
      <form onSubmit={handleSubmit}>
        {error && <p role="alert">{error}</p>}

        <label>
          Kind (e.g. opening, why-this-company, closing)
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
            {saving ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </>
  )
}
