import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deleteSnippet, getSnippet, isConflict } from '@/api/client'
import type { Snippet } from '@/api/types'
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
import ConflictAlert from '@/components/ConflictAlert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import SnippetEditForm from './SnippetEditForm'

export default function SnippetDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [snippet, setSnippet] = useState<Snippet | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [deleteConflict, setDeleteConflict] = useState(false)
  const [reloading, setReloading] = useState(false)

  useEffect(() => {
    setSnippet(null)
    setError(null)
    setEditing(false)
    setDeleteOpen(false)
    setDeleteConflict(false)
    getSnippet(id)
      .then(setSnippet)
      .catch((e) => setError(e.message))
  }, [id])

  async function handleDelete() {
    setDeleting(true)
    try {
      await deleteSnippet(id, snippet?.version)
      navigate('/snippets')
    } catch (e) {
      // A refused delete leaves the record on disk and the page usable, so
      // it shows the conflict alert rather than replacing the page with an
      // error (issue #89, story 18).
      if (isConflict(e)) {
        setDeleteConflict(true)
      } else {
        setError(e instanceof Error ? e.message : String(e))
      }
      setDeleting(false)
      setDeleteOpen(false)
    }
  }

  async function handleReload() {
    setReloading(true)
    try {
      setSnippet(await getSnippet(id))
      setDeleteConflict(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setReloading(false)
    }
  }

  if (error)
    return (
      <p role="alert" className="font-medium text-destructive">
        {error}
      </p>
    )
  if (!snippet) return <p>Loading…</p>

  if (editing) {
    return (
      <>
        <p className="mb-4 inline-block text-sm">
          <Link to="/snippets" className="no-underline hover:underline">
            ← Back to Cover Letter Snippets
          </Link>
        </p>
        <h1>Edit {snippet.kind}</h1>
        <SnippetEditForm
          snippet={snippet}
          onSaved={(saved) => {
            setSnippet(saved)
            setEditing(false)
          }}
          onReloaded={setSnippet}
          onCancel={() => setEditing(false)}
        />
      </>
    )
  }

  return (
    <>
      <p className="mb-4 inline-block text-sm">
        <Link to="/snippets" className="no-underline hover:underline">
          ← Back to Cover Letter Snippets
        </Link>
      </p>
      {deleteConflict && (
        <ConflictAlert record="Cover Letter Snippet" action="delete" keepsEdits={false} onReload={handleReload} reloading={reloading} />
      )}
      <h1>{snippet.kind}</h1>
      <div className="flex flex-wrap gap-1">
        {snippet.tags.map((tag) => (
          <Badge variant="secondary" key={tag}>
            {tag}
          </Badge>
        ))}
      </div>
      <p className="whitespace-pre-wrap">{snippet.body}</p>
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
              <AlertDialogTitle>Delete this snippet?</AlertDialogTitle>
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
