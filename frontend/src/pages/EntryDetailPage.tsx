import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { deleteEntry, getEntry } from '../api/client'
import type { Entry } from '../api/types'
import EntryEditForm from './EntryEditForm'

export default function EntryDetailPage() {
  const params = useParams()
  const navigate = useNavigate()
  const id = params['*'] ?? ''
  const [entry, setEntry] = useState<Entry | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setEntry(null)
    setError(null)
    setEditing(false)
    setConfirmingDelete(false)
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
      setConfirmingDelete(false)
    }
  }

  if (error) return <p role="alert">{error}</p>
  if (!entry) return <p>Loading…</p>

  const title = entry.type === 'experience' ? `${entry.client ?? entry.employer} — ${entry.role}` : entry.name

  if (editing) {
    return (
      <>
        <p>
          <Link to="/">← Back to Master Data</Link>
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
        <Link to="/">← Back to Master Data</Link>
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
      <div>
        {entry.tags.map((tag) => (
          <span className="tag" key={tag}>
            {tag}
          </span>
        ))}
      </div>
      <ul>
        {entry.bullets?.map((bullet, i) => (
          <li key={i}>{bullet}</li>
        ))}
      </ul>
      <button type="button" onClick={() => setEditing(true)}>
        Edit
      </button>{' '}
      {confirmingDelete ? (
        <>
          <span>Delete this entry?</span>{' '}
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
