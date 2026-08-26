import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deleteSnippet, getSnippet } from '../api/client'
import type { Snippet } from '../api/types'
import SnippetEditForm from './SnippetEditForm'

export default function SnippetDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [snippet, setSnippet] = useState<Snippet | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setSnippet(null)
    setError(null)
    setEditing(false)
    setConfirmingDelete(false)
    getSnippet(id)
      .then(setSnippet)
      .catch((e) => setError(e.message))
  }, [id])

  async function handleDelete() {
    setDeleting(true)
    try {
      await deleteSnippet(id)
      navigate('/snippets')
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setDeleting(false)
      setConfirmingDelete(false)
    }
  }

  if (error) return <p role="alert">{error}</p>
  if (!snippet) return <p>Loading…</p>

  if (editing) {
    return (
      <>
        <p>
          <Link to="/snippets">← Back to Cover Letter Snippets</Link>
        </p>
        <h1>Edit {snippet.kind}</h1>
        <SnippetEditForm
          snippet={snippet}
          onSaved={(saved) => {
            setSnippet(saved)
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
        <Link to="/snippets">← Back to Cover Letter Snippets</Link>
      </p>
      <h1>{snippet.kind}</h1>
      <div>
        {snippet.tags.map((tag) => (
          <span className="tag" key={tag}>
            {tag}
          </span>
        ))}
      </div>
      <p style={{ whiteSpace: 'pre-wrap' }}>{snippet.body}</p>
      <button type="button" onClick={() => setEditing(true)}>
        Edit
      </button>{' '}
      {confirmingDelete ? (
        <>
          <span>Delete this snippet?</span>{' '}
          <button type="button" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Yes, delete'}
          </button>{' '}
          <button type="button" onClick={() => setConfirmingDelete(false)} disabled={deleting}>
            Cancel
          </button>
        </>
      ) : (
        <button type="button" onClick={() => setConfirmingDelete(true)}>
          Delete
        </button>
      )}
    </>
  )
}
