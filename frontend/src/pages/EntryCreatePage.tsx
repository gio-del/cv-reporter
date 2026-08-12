import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createEntry } from '../api/client'
import type { EntryInput, EntryType } from '../api/types'

const blankForm = {
  employer: '',
  client: '',
  role: '',
  name: '',
  location: '',
  start: '',
  end: '',
  flagship: false,
  tags: '',
  repo: '',
  bullets: '',
}

type FormState = typeof blankForm

export default function EntryCreatePage() {
  const navigate = useNavigate()
  const [type, setType] = useState<EntryType>('experience')
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
      const input: EntryInput = {
        type,
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
      const created = await createEntry(input)
      navigate(`/entries/${created.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <p>
        <Link to="/">← Back to Master Data</Link>
      </p>
      <h1>New Entry</h1>
      <form onSubmit={handleSubmit}>
        {error && <p role="alert">{error}</p>}

        <label>
          Type
          <select value={type} onChange={(e) => setType(e.target.value as EntryType)}>
            <option value="experience">Experience</option>
            <option value="project">Project</option>
          </select>
        </label>

        {type === 'experience' ? (
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
            {saving ? 'Creating…' : 'Create'}
          </button>
        </div>
      </form>
    </>
  )
}
