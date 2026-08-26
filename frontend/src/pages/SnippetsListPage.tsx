import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listSnippets } from '../api/client'
import type { Snippet } from '../api/types'

export default function SnippetsListPage() {
  const [snippets, setSnippets] = useState<Snippet[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listSnippets()
      .then(setSnippets)
      .catch((e) => setError(e.message))
  }, [])

  if (error) return <p role="alert">{error}</p>
  if (!snippets) return <p>Loading…</p>

  return (
    <>
      <p>
        <Link to="/">← Back to Master Data</Link>
      </p>
      <h1>Cover Letter Snippets</h1>
      <p>
        <Link to="/snippets/new">+ New Snippet</Link>
      </p>

      {snippets.length === 0 && (
        <p>No snippets yet. Without any, Cover Letter drafting falls back to fresh prose.</p>
      )}

      <ul>
        {snippets.map((snippet) => (
          <li key={snippet.id}>
            <Link to={`/snippets/${snippet.id}`}>{snippet.kind}</Link>{' '}
            <span>{snippet.body.slice(0, 80)}{snippet.body.length > 80 ? '…' : ''}</span>
            <div>
              {snippet.tags.map((tag) => (
                <span className="tag" key={tag}>
                  {tag}
                </span>
              ))}
            </div>
          </li>
        ))}
      </ul>
    </>
  )
}
