import { useState } from 'react'
import { Link } from 'react-router-dom'
import RALBadge from '@/components/RALBadge'
import { saveJobListing } from '@/api/client'
import type { SaveJobListingResult } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'

const blankForm = { company: '', url: '', jobDescription: '', jobDescriptionUrl: '' }
type FormState = typeof blankForm

export default function JobListingCreatePage() {
  const [form, setForm] = useState<FormState>(blankForm)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState<SaveJobListingResult | null>(null)

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setSaving(true)
    try {
      const result = await saveJobListing({
        company: form.company,
        url: form.url.trim() || undefined,
        jobDescription: form.jobDescription.trim() || undefined,
        jobDescriptionUrl: form.jobDescriptionUrl.trim() || undefined,
      })
      setSaved(result)
      setForm(blankForm)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <p className="mb-4 inline-block text-sm">
        <Link to="/jobs" className="no-underline hover:underline">
          ← Back to Job Listings
        </Link>
      </p>
      <h1>Save a Job Listing</h1>
      <form onSubmit={handleSubmit}>
        {error && (
          <p role="alert" className="mb-4 font-medium text-destructive">
            {error}
          </p>
        )}

        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="company">Company</FieldLabel>
            <Input id="company" value={form.company} onChange={(e) => set('company', e.target.value)} required />
          </Field>
          <Field>
            <FieldLabel htmlFor="listing-url">Posting URL (optional)</FieldLabel>
            <Input id="listing-url" type="url" value={form.url} onChange={(e) => set('url', e.target.value)} placeholder="https://…" />
          </Field>
          <Field>
            <FieldLabel htmlFor="job-description">Job Description (paste text)</FieldLabel>
            <Textarea
              id="job-description"
              rows={8}
              value={form.jobDescription}
              onChange={(e) => set('jobDescription', e.target.value)}
              placeholder="Paste the job description here…"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="job-description-url">…or a URL to fetch it from</FieldLabel>
            <Input
              id="job-description-url"
              type="url"
              value={form.jobDescriptionUrl}
              onChange={(e) => set('jobDescriptionUrl', e.target.value)}
              placeholder="https://…"
            />
            <FieldDescription>One of the two Job Description fields is required.</FieldDescription>
          </Field>
        </FieldGroup>

        <div className="mt-6 flex gap-3">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Save Job Listing'}
          </Button>
        </div>
      </form>

      {saved && (
        <section className="mt-6 rounded-xl border border-border bg-card p-5">
          <h2 className="mt-0">Saved: {saved.jobListing.company}</h2>
          <p>
            Application status: <strong>{saved.application.status}</strong>
          </p>
          <RALBadge ral={saved.jobListing.ral} />
          <p>
            <Link to="/jobs" className="no-underline hover:underline">
              View all Job Listings →
            </Link>
          </p>
        </section>
      )}
    </>
  )
}
