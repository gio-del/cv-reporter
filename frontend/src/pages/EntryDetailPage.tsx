import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deleteEntry, getEntry } from '@/api/client'
import type { Entry } from '@/api/types'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import EntryEditForm from './EntryEditForm'

export default function EntryDetailPage() {
  const params = useParams()
  const navigate = useNavigate()
  const id = params['*'] ?? ''
  const [entry, setEntry] = useState<Entry | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setEntry(null)
    setError(null)
    setEditing(false)
    setDeleteOpen(false)
    getEntry(id)
      .then(setEntry)
      .catch((e) => setError(e.message))
  }, [id])

  async function handleDelete() {
    setDeleting(true)
    try {
      await deleteEntry(id)
      navigate('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setDeleting(false)
      setDeleteOpen(false)
    }
  }

  if (error)
    return (
      <p role="alert" className="font-medium text-destructive">
        {error}
      </p>
    )
  if (!entry) return <p>Loading…</p>

  const title = entry.type === 'experience' ? `${entry.client ?? entry.employer} — ${entry.role}` : entry.name

  if (editing) {
    return (
      <>
        <p className="mb-4 inline-block text-sm">
          <Link to="/" className="no-underline hover:underline">
            ← Back to Master Data
          </Link>
        </p>
        <h1>Edit {title}</h1>
        <EntryEditForm
          entry={entry}
          onSaved={(saved) => {
            setEntry(saved)
            setEditing(false)
          }}
          onCancel={() => setEditing(false)}
        />
      </>
    )
  }

  return (
    <>
      <p>
        <Link to="/" className="no-underline hover:underline">
          ← Back to Master Data
        </Link>
      </p>
      <h1>{title}</h1>
      {entry.type === 'experience' && (
        <dl>
          <dt>Employer</dt>
          <dd>{entry.employer}</dd>
          {entry.client && (
            <>
              <dt>Client</dt>
              <dd>{entry.client}</dd>
            </>
          )}
          <dt>Location</dt>
          <dd>{entry.location}</dd>
        </dl>
      )}
      <p>
        {entry.start} – {entry.end ?? 'present'}
      </p>
      <div className="flex flex-wrap gap-1">
        {entry.tags.map((tag) => (
          <Badge variant="secondary" key={tag}>
            {tag}
          </Badge>
        ))}
      </div>
      <ul className="list-disc pl-5">
        {entry.bullets?.map((bullet, i) => (
          <li key={i}>{bullet}</li>
        ))}
      </ul>
      <div className="mt-6 flex items-center gap-3">
        <Button type="button" onClick={() => setEditing(true)}>
          Edit
        </Button>
        <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
          <AlertDialogTrigger asChild>
            <Button type="button" variant="outline">
              Delete
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Delete this entry?</AlertDialogTitle>
              <AlertDialogDescription>This action cannot be undone.</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={deleting}>Cancel</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={(e) => {
                  e.preventDefault()
                  handleDelete()
                }}
                disabled={deleting}
              >
                {deleting ? 'Deleting…' : 'Yes, delete'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>
    </>
  )
}
