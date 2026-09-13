import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import { addApplicationNote, editApplicationNote } from '@/api/client'
import type { Application, Note } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

// newestFirst orders Notes by when they were written. The API already
// returns them that way; sorting here keeps the log readable regardless.
function newestFirst(notes: Note[]): Note[] {
  return [...notes].sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
}

/**
 * ApplicationNotes is an Application's Notes (issue #96): the user's own
 * timestamped Markdown log, newest first, with an always-available add box.
 * It talks to the API itself and hands the page the updated Application each
 * write answers with, so it can move to any surface that shows an
 * Application without changing inside.
 */
export default function ApplicationNotes({
  applicationId,
  notes,
  onApplicationChange,
}: {
  applicationId: string
  notes: Note[] | undefined
  onApplicationChange: (application: Application) => void
}) {
  const [draft, setDraft] = useState('')
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const ordered = newestFirst(notes ?? [])

  async function handleAdd() {
    setError(null)
    setAdding(true)
    try {
      onApplicationChange(await addApplicationNote(applicationId, draft))
      setDraft('')
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setAdding(false)
    }
  }

  return (
    <section className="mt-6">
      <h2 className="mt-0 mb-2">Notes</h2>

      <div className="flex flex-col gap-2">
        <Textarea
          aria-label="New Note"
          placeholder="What happened? Markdown is supported."
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          disabled={adding}
        />
        <div className="flex justify-end">
          <Button size="sm" onClick={handleAdd} disabled={adding || draft.trim() === ''}>
            {adding ? 'Adding…' : 'Add Note'}
          </Button>
        </div>
        {error && (
          <p role="alert" className="mb-0 text-sm font-medium text-destructive">
            {error}
          </p>
        )}
      </div>

      {ordered.length === 0 ? (
        <p className="mt-3 text-sm text-muted-foreground">No Notes yet.</p>
      ) : (
        <ol aria-label="Notes" className="mt-3 flex flex-col gap-3 pl-0">
          {ordered.map((note) => (
            <NoteItem
              key={note.id}
              applicationId={applicationId}
              note={note}
              onApplicationChange={onApplicationChange}
            />
          ))}
        </ol>
      )}
    </section>
  )
}

// NoteItem is one Note in the log: its fixed creation date, an edited marker
// once corrected, its body rendered as Markdown, and the correction form.
function NoteItem({
  applicationId,
  note,
  onApplicationChange,
}: {
  applicationId: string
  note: Note
  onApplicationChange: (application: Application) => void
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(note.body)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function startEditing() {
    setDraft(note.body)
    setError(null)
    setEditing(true)
  }

  async function handleSave() {
    setError(null)
    setSaving(true)
    try {
      onApplicationChange(await editApplicationNote(applicationId, note.id, draft))
      setEditing(false)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <li className="list-none rounded-lg border border-border bg-card px-3 py-2">
      <div className="mb-1 flex flex-wrap items-center justify-between gap-2">
        <p className="mb-0 text-xs text-muted-foreground">
          <time dateTime={note.createdAt}>{formatDate(note.createdAt)}</time>
          {note.editedAt && (
            <>
              {' '}
              <span title={`Edited ${formatDate(note.editedAt)}`}>(edited)</span>
            </>
          )}
        </p>
        {!editing && (
          <Button size="sm" variant="ghost" aria-label="Edit Note" onClick={startEditing}>
            Edit
          </Button>
        )}
      </div>

      {editing ? (
        <div className="flex flex-col gap-2">
          <Textarea aria-label="Edit Note" value={draft} onChange={(e) => setDraft(e.target.value)} disabled={saving} />
          <div className="flex justify-end gap-2">
            <Button size="sm" variant="ghost" onClick={() => setEditing(false)} disabled={saving}>
              Cancel
            </Button>
            <Button size="sm" onClick={handleSave} disabled={saving || draft.trim() === ''}>
              {saving ? 'Saving…' : 'Save Note'}
            </Button>
          </div>
        </div>
      ) : (
        <div
          className="text-sm break-words [&_a]:underline [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:mb-0 [&_p+p]:mt-2
            [&_p+ul]:mt-2 [&_strong]:font-semibold [&_ul]:list-disc [&_ul]:pl-5"
        >
          <ReactMarkdown remarkPlugins={[remarkBreaks]}>{note.body}</ReactMarkdown>
        </div>
      )}

      {error && (
        <p role="alert" className="mt-2 mb-0 text-sm font-medium text-destructive">
          {error}
        </p>
      )}
    </li>
  )
}
