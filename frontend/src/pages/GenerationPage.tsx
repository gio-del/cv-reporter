import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import BulletDiff from '@/components/BulletDiff'
import RALBadge from '@/components/RALBadge'
import {
  createGeneration,
  generationFileUrl,
  getJobListing,
  listEntries,
  recordApplicationGeneration,
  renderGeneration,
} from '@/api/client'
import type { Entry, JobListing, RALRange, RenderResult, SelectedBullet, SelectedEntry } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'

const SLUG_RE = /^[a-z0-9]+(-[a-z0-9]+)*$/

interface EditableBullet extends SelectedBullet {
  included: boolean
}

interface EditableEntry {
  entryId: string
  reason: string
  included: boolean
  bullets: EditableBullet[]
}

function toEditable(entries: SelectedEntry[]): EditableEntry[] {
  return entries.map((e) => ({
    entryId: e.entryId,
    reason: e.reason,
    included: true,
    bullets: e.bullets.map((b) => ({ ...b, included: true })),
  }))
}

function entryLabel(entry: Entry | undefined, entryId: string): string {
  if (!entry) return entryId
  if (entry.type === 'experience') {
    return entry.client ? `${entry.employer} — ${entry.client}` : (entry.employer ?? entryId)
  }
  return entry.name ?? entryId
}

export default function GenerationPage() {
  const { id: jobListingId } = useParams<{ id?: string }>()
  const [jobListing, setJobListing] = useState<JobListing | null>(null)
  const [entriesById, setEntriesById] = useState<Map<string, Entry>>(new Map())
  const [jobDescription, setJobDescription] = useState('')
  const [jobDescriptionUrl, setJobDescriptionUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mode, setMode] = useState<'default' | 'tailored' | null>(null)
  const [editable, setEditable] = useState<EditableEntry[] | null>(null)
  const [coverLetter, setCoverLetter] = useState<string | null>(null)
  const [ral, setRal] = useState<RALRange | null>(null)
  const [slug, setSlug] = useState(jobListingId ?? 'default')
  const [rendering, setRendering] = useState(false)
  const [renderError, setRenderError] = useState<string | null>(null)
  const [render, setRender] = useState<RenderResult | null>(null)
  const [linkError, setLinkError] = useState<string | null>(null)

  useEffect(() => {
    listEntries()
      .then((entries) => setEntriesById(new Map(entries.map((e) => [e.id, e]))))
      .catch(() => {
        // Labels fall back to raw entryIds if Master Data can't be loaded.
      })
  }, [])

  useEffect(() => {
    if (!jobListingId) return
    getJobListing(jobListingId)
      .then((listing) => {
        setJobListing(listing)
        setJobDescription(listing.jobDescription)
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [jobListingId])

  async function handleStart() {
    setError(null)
    setLoading(true)
    setRender(null)
    setRenderError(null)
    try {
      const result = await createGeneration({
        jobDescription: jobDescription.trim() || undefined,
        jobDescriptionUrl: jobDescriptionUrl.trim() || undefined,
      })
      setMode(result.mode)
      setEditable(toEditable(result.selection.entries))
      setCoverLetter(result.coverLetter?.body ?? null)
      setRal(result.ral ?? null)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  function updateBullet(entryId: string, sourceIndex: number, rewritten: string) {
    setEditable((prev) =>
      prev
        ? prev.map((e) =>
            e.entryId !== entryId
              ? e
              : { ...e, bullets: e.bullets.map((b) => (b.sourceIndex === sourceIndex ? { ...b, rewritten } : b)) },
          )
        : prev,
    )
  }

  function toggleBullet(entryId: string, sourceIndex: number) {
    setEditable((prev) =>
      prev
        ? prev.map((e) =>
            e.entryId !== entryId
              ? e
              : {
                  ...e,
                  bullets: e.bullets.map((b) =>
                    b.sourceIndex === sourceIndex ? { ...b, included: !b.included } : b,
                  ),
                }
          )
        : prev,
    )
  }

  function toggleEntry(entryId: string) {
    setEditable((prev) =>
      prev ? prev.map((e) => (e.entryId !== entryId ? e : { ...e, included: !e.included })) : prev,
    )
  }

  async function handleRender() {
    if (!editable) return
    if (!SLUG_RE.test(slug)) {
      setRenderError('Name must be lowercase kebab-case, e.g. "acme-corp".')
      return
    }

    const selection: SelectedEntry[] = editable
      .filter((e) => e.included)
      .map((e) => ({
        entryId: e.entryId,
        reason: e.reason,
        bullets: e.bullets
          .filter((b) => b.included)
          .map((b) => ({ sourceIndex: b.sourceIndex, source: b.source, rewritten: b.rewritten })),
      }))
      .filter((e) => e.bullets.length > 0)

    setRenderError(null)
    setLinkError(null)
    setRendering(true)
    try {
      const result = await renderGeneration({
        slug,
        selection: { entries: selection },
        coverLetter: coverLetter !== null ? { body: coverLetter } : undefined,
      })
      setRender(result)

      if (jobListingId) {
        try {
          await recordApplicationGeneration(jobListingId, {
            slug: result.slug,
            cvPath: result.cvPath,
            coverLetterPath: result.coverLetterPath,
          })
        } catch (err) {
          setLinkError(err instanceof Error ? err.message : String(err))
        }
      }
    } catch (err) {
      setRenderError(err instanceof Error ? err.message : String(err))
    } finally {
      setRendering(false)
    }
  }

  return (
    <>
      {jobListingId && (
        <p className="mb-4 inline-block text-sm">
          <Link to="/jobs" className="no-underline hover:underline">
            ← Back to Job Listings
          </Link>
        </p>
      )}
      <h1>{jobListing ? `Generate a Tailored CV for ${jobListing.company}` : 'Generate a Tailored CV'}</h1>

      <section>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="job-description">Job Description (paste text)</FieldLabel>
            <Textarea
              id="job-description"
              rows={6}
              value={jobDescription}
              onChange={(e) => setJobDescription(e.target.value)}
              placeholder="Paste the job description here…"
            />
          </Field>
          <Field>
            <FieldLabel htmlFor="job-description-url">…or a URL to fetch it from</FieldLabel>
            <Input
              id="job-description-url"
              type="url"
              value={jobDescriptionUrl}
              onChange={(e) => setJobDescriptionUrl(e.target.value)}
              placeholder="https://…"
            />
          </Field>
        </FieldGroup>
        <p>Leave both blank to run Default Mode (a general-purpose CV from your most representative Entries).</p>
        <div className="mt-6 flex gap-3">
          <Button onClick={handleStart} disabled={loading}>
            {loading ? 'Generating…' : 'Start Generation'}
          </Button>
        </div>
      </section>

      {error && (
        <p role="alert" className="font-medium text-destructive">
          {error}
        </p>
      )}

      {editable && (
        <section>
          <h2>Text Review {mode === 'default' && '(Default Mode)'}</h2>
          <p>Review Selection and Rewrite before anything is rendered. Edit any bullet, or exclude one entirely.</p>

          {ral && <RALBadge ral={ral} />}

          {editable.map((entry) => {
            const label = entryLabel(entriesById.get(entry.entryId), entry.entryId)
            return (
              <div key={entry.entryId} className="mb-4 rounded-xl border border-border bg-card p-5">
                <h3 className="mb-0">
                  <label className="flex items-center gap-2 font-semibold">
                    <Checkbox checked={entry.included} onCheckedChange={() => toggleEntry(entry.entryId)} />
                    {label}
                  </label>
                </h3>
                <p className="mt-2 text-muted-foreground italic">{entry.reason}</p>
                {entry.included && (
                  <ul className="mt-3 flex flex-col gap-3">
                    {entry.bullets.map((bullet) => (
                      <li
                        key={bullet.sourceIndex}
                        className={cn(
                          'border-t border-border pt-2 first:border-t-0 first:pt-0',
                          !bullet.included && 'opacity-50',
                        )}
                      >
                        <label className="flex items-start gap-2">
                          <Checkbox
                            className="mt-0.5"
                            checked={bullet.included}
                            onCheckedChange={() => toggleBullet(entry.entryId, bullet.sourceIndex)}
                          />
                          <span className={cn(bullet.included ? '' : 'line-through')}>
                            <BulletDiff source={bullet.source} rewritten={bullet.rewritten} />
                          </span>
                        </label>
                        {bullet.included && (
                          <Textarea
                            className="mt-2"
                            rows={2}
                            value={bullet.rewritten}
                            onChange={(e) => updateBullet(entry.entryId, bullet.sourceIndex, e.target.value)}
                          />
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )
          })}

          {coverLetter !== null && (
            <div className="mb-4 rounded-xl border border-border bg-card p-5">
              <h3>Cover Letter</h3>
              <Textarea rows={12} value={coverLetter} onChange={(e) => setCoverLetter(e.target.value)} />
            </div>
          )}

          <Field>
            <FieldLabel htmlFor="generation-slug">Save as (used for the output filename)</FieldLabel>
            <Input
              id="generation-slug"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="acme-corp"
              aria-describedby="generation-slug-hint"
            />
            <FieldDescription id="generation-slug-hint">
              Lowercase kebab-case, e.g. "acme-corp".
            </FieldDescription>
          </Field>

          {renderError && (
            <p role="alert" className="font-medium text-destructive">
              {renderError}
            </p>
          )}

          <div className="mt-6 flex gap-3">
            <Button onClick={handleRender} disabled={rendering}>
              {rendering ? 'Rendering…' : render ? 'Re-render' : 'Approve Text Review'}
            </Button>
          </div>
        </section>
      )}

      {render && (
        <section>
          <h2>Visual Review</h2>
          {jobListingId && !linkError && (
            <p>
              Linked to <strong>{jobListing?.company ?? jobListingId}</strong>'s Application.
            </p>
          )}
          {linkError && (
            <p role="alert" className="font-medium text-destructive">
              Rendered, but failed to link this Generation to the Application: {linkError}
            </p>
          )}
          {render.cvPageCount !== 1 && (
            <p role="alert" className="font-medium text-destructive">
              The CV rendered to {render.cvPageCount} pages — it should be one. Trim a bullet or Entry above and
              re-render.
            </p>
          )}
          <iframe
            className="block h-[min(800px,75vh)] w-full rounded-xl border border-border"
            title="Tailored CV preview"
            src={generationFileUrl(render.slug, 'cv.pdf')}
          />
          <p>
            <a href={generationFileUrl(render.slug, 'cv.pdf')} download={`${render.slug}-cv.pdf`}>
              Download CV (PDF)
            </a>
            {render.coverLetterPath && (
              <>
                {' · '}
                <a href={generationFileUrl(render.slug, 'cover-letter.pdf')} download={`${render.slug}-cover-letter.pdf`}>
                  Download Cover Letter (PDF)
                </a>
                {' · '}
                <a href={generationFileUrl(render.slug, 'cover-letter.txt')} download={`${render.slug}-cover-letter.txt`}>
                  Download Cover Letter (Text)
                </a>
              </>
            )}
          </p>
        </section>
      )}
    </>
  )
}
