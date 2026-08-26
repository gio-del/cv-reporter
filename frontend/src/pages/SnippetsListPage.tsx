import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listSnippets } from '../api/client'
import type { Snippet } from '../api/types'

export default function SnippetsListPage() {
  const [snippets, setSnippets] = useState<Snippet[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listSnippets()
      .then((s) => setSnippets(s ?? []))
      .catch((e) => setError(e.message))
  }, [])

  if (error) return <p role="alert">{error}</p>
  if (!snippets) return <p>Loading…</p>

  return (
    <>
      <p className="breadcrumb">
        <Link to="/">← Back to Master Data</Link>
      </p>
      <div className="page-header">
        <h1>Cover Letter Snippets</h1>
        <Link to="/snippets/new">+ New Snippet</Link>
      </div>

      {snippets.length === 0 && (
        <p>No snippets yet. Without any, Cover Letter drafting falls back to fresh prose.</p>
      )}

      <ul className="entry-list">
        {snippets.map((snippet) => (
          <li key={snippet.id}>
            <div className="entry-list-row">
              <Link to={`/snippets/${snippet.id}`}>{snippet.kind}</Link>
            </div>
            <p>
              {snippet.body.slice(0, 80)}
              {snippet.body.length > 80 ? '…' : ''}
            </p>
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
