import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import { addApplicationNote, deleteApplicationNote, editApplicationNote, isConflict } from '@/api/client'
import type { Application, Note } from '@/api/types'
import ConflictAlert from '@/components/ConflictAlert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
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
 *
 * Every write presents the Application's version token (issue #89). A 409
 * shows the shared conflict alert next to the control that was refused, and
 * onReload — the page re-reading its record — is the explicit way back.
 */
export default function ApplicationNotes({
  applicationId,
  version,
  notes,
  onApplicationChange,
  onReload,
}: {
  applicationId: string
  // version is the Application's token as the page last read it.
  version: string | undefined
  notes: Note[] | undefined
  onApplicationChange: (application: Application) => void
  onReload: () => Promise<void>
}) {
  const [draft, setDraft] = useState('')
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [conflict, setConflict] = useState(false)
  const [reloading, setReloading] = useState(false)
  const ordered = newestFirst(notes ?? [])

  async function handleAdd() {
    setError(null)
    setConflict(false)
    setAdding(true)
    try {
      onApplicationChange(await addApplicationNote(applicationId, draft, version))
      setDraft('')
    } catch (err) {
      // The draft stays in the box either way; reloading refreshes the log
      // and the token around it without touching what was typed.
      if (isConflict(err)) {
        setConflict(true)
      } else {
        setError(errorMessage(err))
      }
    } finally {
      setAdding(false)
    }
  }

  async function handleReload() {
    setReloading(true)
    try {
      await onReload()
      setConflict(false)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setReloading(false)
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
        {conflict && (
          <ConflictAlert
            record="Application"
            keepsEdits={false}
            className="mb-0"
            onReload={handleReload}
            reloading={reloading}
          />
        )}
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
              version={version}
              note={note}
              onApplicationChange={onApplicationChange}
              onReload={onReload}
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
  version,
  note,
  onApplicationChange,
  onReload,
}: {
  applicationId: string
  version: string | undefined
  note: Note
  onApplicationChange: (application: Application) => void
  onReload: () => Promise<void>
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(note.body)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  // conflict is which write the backend refused with 409, if any.
  const [conflict, setConflict] = useState<'save' | 'delete' | null>(null)
  const [reloading, setReloading] = useState(false)

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)

  // On success the Note leaves the list and this item unmounts, so only the
  // failure path resets local state.
  async function handleConfirmDelete() {
    setError(null)
    setConflict(null)
    setDeleting(true)
    try {
      onApplicationChange(await deleteApplicationNote(applicationId, note.id, version))
    } catch (err) {
      if (isConflict(err)) {
        setConflict('delete')
      } else {
        setError(errorMessage(err))
      }
      setDeleting(false)
      setDeleteOpen(false)
    }
  }

  function startEditing() {
    setDraft(note.body)
    setError(null)
    setConflict(null)
    setEditing(true)
  }

  async function handleSave() {
    setError(null)
    setConflict(null)
    setSaving(true)
    try {
      onApplicationChange(await editApplicationNote(applicationId, note.id, draft, version))
      setEditing(false)
    } catch (err) {
      // A conflict keeps the correction on screen until the user reloads,
      // which closes it onto the Note as it now is on disk.
      if (isConflict(err)) {
        setConflict('save')
      } else {
        setError(errorMessage(err))
      }
    } finally {
      setSaving(false)
    }
  }

  async function handleReload() {
    setReloading(true)
    try {
      await onReload()
      setConflict(null)
      setEditing(false)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setReloading(false)
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
          <span className="flex gap-1">
            <Button size="sm" variant="ghost" aria-label="Edit Note" onClick={startEditing} disabled={deleting}>
              Edit
            </Button>
            <Button
              size="sm"
              variant="ghost"
              aria-label="Delete Note"
              onClick={() => setDeleteOpen(true)}
              disabled={deleting}
            >
              Delete
            </Button>
          </span>
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

      {conflict && (
        <ConflictAlert
          record="Application"
          action={conflict}
          keepsEdits={conflict === 'save'}
          className="mt-2 mb-0"
          onReload={handleReload}
          reloading={reloading}
        />
      )}
      {error && (
        <p role="alert" className="mt-2 mb-0 text-sm font-medium text-destructive">
          {error}
        </p>
      )}

      {/* Deleting is permanent — no tombstone — so it asks first (story 12). */}
      <AlertDialog open={deleteOpen} onOpenChange={(open) => !deleting && setDeleteOpen(open)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete this Note?</AlertDialogTitle>
            <AlertDialogDescription>
              The Note written {formatDate(note.createdAt)} will be removed from this Application. This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={(e) => {
                e.preventDefault()
                handleConfirmDelete()
              }}
              disabled={deleting}
            >
              {deleting ? 'Deleting…' : 'Yes, delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </li>
  )
}
