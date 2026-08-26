import { useEffect, useState } from 'react'
import BulletDiff from '../components/BulletDiff'
import { createGeneration, listEntries } from '../api/client'
import type { Entry, SelectedBullet, SelectedEntry } from '../api/types'

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
  const [approved, setApproved] = useState(false)

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
    setApproved(false)
    try {
      const result = await createGeneration({
        jobDescription: jobDescription.trim() || undefined,
        jobDescriptionUrl: jobDescriptionUrl.trim() || undefined,
      })
      setMode(result.mode)
      setEditable(toEditable(result.selection.entries))
      setCoverLetter(result.coverLetter?.body ?? null)
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

  function handleApprove() {
    setApproved(true)
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
        <button onClick={handleStart} disabled={loading}>
          {loading ? 'Generating…' : 'Start Generation'}
        </button>
      </section>

      {error && <p role="alert">{error}</p>}

      {editable && (
        <section>
          <h2>Text Review {mode === 'default' && '(Default Mode)'}</h2>
          <p>Review Selection and Rewrite before anything is rendered. Edit any bullet, or exclude one entirely.</p>

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
                  <ul>
                    {entry.bullets.map((bullet) => (
                      <li key={bullet.sourceIndex} className={bullet.included ? '' : 'review-bullet-excluded'}>
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

          <button onClick={handleApprove} disabled={approved}>
            {approved ? 'Approved' : 'Approve Text Review'}
          </button>
        </section>
      )}
    </>
  )
}
