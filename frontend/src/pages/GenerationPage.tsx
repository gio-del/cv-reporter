import { useEffect, useState } from 'react'
import BulletDiff from '../components/BulletDiff'
import RALBadge from '../components/RALBadge'
import { createGeneration, generationFileUrl, listEntries, renderGeneration } from '../api/client'
import type { Entry, RALRange, RenderResult, SelectedBullet, SelectedEntry } from '../api/types'

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
  const [entriesById, setEntriesById] = useState<Map<string, Entry>>(new Map())
  const [jobDescription, setJobDescription] = useState('')
  const [jobDescriptionUrl, setJobDescriptionUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mode, setMode] = useState<'default' | 'tailored' | null>(null)
  const [editable, setEditable] = useState<EditableEntry[] | null>(null)
  const [coverLetter, setCoverLetter] = useState<string | null>(null)
  const [ral, setRal] = useState<RALRange | null>(null)
  const [slug, setSlug] = useState('default')
  const [rendering, setRendering] = useState(false)
  const [renderError, setRenderError] = useState<string | null>(null)
  const [render, setRender] = useState<RenderResult | null>(null)

  useEffect(() => {
    listEntries()
      .then((entries) => setEntriesById(new Map(entries.map((e) => [e.id, e]))))
      .catch(() => {
        // Labels fall back to raw entryIds if Master Data can't be loaded.
      })
  }, [])

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
    setRendering(true)
    try {
      const result = await renderGeneration({
        slug,
        selection: { entries: selection },
        coverLetter: coverLetter !== null ? { body: coverLetter } : undefined,
      })
      setRender(result)
    } catch (err) {
      setRenderError(err instanceof Error ? err.message : String(err))
    } finally {
      setRendering(false)
    }
  }

  return (
    <>
      <h1>Generate a Tailored CV</h1>

      <section>
        <label>
          Job Description (paste text)
          <textarea
            rows={6}
            value={jobDescription}
            onChange={(e) => setJobDescription(e.target.value)}
            placeholder="Paste the job description here…"
          />
        </label>
        <label>
          …or a URL to fetch it from
          <input
            type="url"
            value={jobDescriptionUrl}
            onChange={(e) => setJobDescriptionUrl(e.target.value)}
            placeholder="https://…"
          />
        </label>
        <p>Leave both blank to run Default Mode (a general-purpose CV from your most representative Entries).</p>
        <div className="form-actions">
          <button onClick={handleStart} disabled={loading}>
            {loading ? 'Generating…' : 'Start Generation'}
          </button>
        </div>
      </section>

      {error && <p role="alert">{error}</p>}

      {editable && (
        <section>
          <h2>Text Review {mode === 'default' && '(Default Mode)'}</h2>
          <p>Review Selection and Rewrite before anything is rendered. Edit any bullet, or exclude one entirely.</p>

          {ral && <RALBadge ral={ral} />}

          {editable.map((entry) => {
            const label = entryLabel(entriesById.get(entry.entryId), entry.entryId)
            return (
              <div key={entry.entryId} className="review-entry">
                <h3>
                  <label>
                    <input type="checkbox" checked={entry.included} onChange={() => toggleEntry(entry.entryId)} />
                    {label}
                  </label>
                </h3>
                <p className="review-reason">{entry.reason}</p>
                {entry.included && (
                  <ul className="review-bullet-list">
                    {entry.bullets.map((bullet) => (
                      <li
                        key={bullet.sourceIndex}
                        className={`review-bullet${bullet.included ? '' : ' review-bullet-excluded'}`}
                      >
                        <label>
                          <input
                            type="checkbox"
                            checked={bullet.included}
                            onChange={() => toggleBullet(entry.entryId, bullet.sourceIndex)}
                          />
                          <BulletDiff source={bullet.source} rewritten={bullet.rewritten} />
                        </label>
                        {bullet.included && (
                          <textarea
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
            <div className="review-entry">
              <h3>Cover Letter</h3>
              <textarea
                rows={12}
                value={coverLetter}
                onChange={(e) => setCoverLetter(e.target.value)}
              />
            </div>
          )}

          <label htmlFor="generation-slug">
            Save as (used for the output filename)
            <input
              id="generation-slug"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="acme-corp"
              aria-describedby="generation-slug-hint"
            />
            <small id="generation-slug-hint" className="field-hint">
              Lowercase kebab-case, e.g. "acme-corp".
            </small>
          </label>

          {renderError && <p role="alert">{renderError}</p>}

          <div className="form-actions">
            <button onClick={handleRender} disabled={rendering}>
              {rendering ? 'Rendering…' : render ? 'Re-render' : 'Approve Text Review'}
            </button>
          </div>
        </section>
      )}

      {render && (
        <section>
          <h2>Visual Review</h2>
          {render.cvPageCount !== 1 && (
            <p role="alert">
              The CV rendered to {render.cvPageCount} pages — it should be one. Trim a bullet or Entry above and
              re-render.
            </p>
          )}
          <iframe
            className="cv-preview-frame"
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
