import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listEntries } from '../api/client'
import type { Entry } from '../api/types'

export default function EntriesListPage() {
  const [entries, setEntries] = useState<Entry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listEntries()
      .then((e) => setEntries(e ?? []))
      .catch((e) => setError(e.message))
  }, [])

  if (error) return <p role="alert">{error}</p>
  if (!entries) return <p>Loading…</p>

  const experience = entries.filter((e) => e.type === 'experience')
  const projects = entries.filter((e) => e.type === 'project')

  const byEmployer = new Map<string, Entry[]>()
  for (const entry of experience) {
    const key = entry.employer ?? 'Unknown employer'
    if (!byEmployer.has(key)) byEmployer.set(key, [])
    byEmployer.get(key)!.push(entry)
  }

  return (
    <>
      <div className="page-header">
        <h1>Master Data</h1>
        <Link to="/entries/new">+ New Entry</Link>
      </div>

      <section>
        <h2>Experience</h2>
        {[...byEmployer.entries()].map(([employer, employerEntries]) => (
          <div className="entry-group" key={employer}>
            <h3>{employer}</h3>
            <ul className="entry-list">
              {employerEntries.map((entry) => (
                <EntryListItem key={entry.id} entry={entry} label={entry.client ?? entry.role ?? entry.id} />
              ))}
            </ul>
          </div>
        ))}
      </section>

      <section>
        <h2>Projects</h2>
        <ul className="entry-list">
          {projects.map((entry) => (
            <EntryListItem key={entry.id} entry={entry} label={entry.name ?? entry.id} />
          ))}
        </ul>
      </section>
    </>
  )
}

function EntryListItem({ entry, label }: { entry: Entry; label: string }) {
  return (
    <li>
      <div className="entry-list-row">
        <Link to={`/entries/${entry.id}`}>{label}</Link>
        <span className="entry-list-dates">
          {entry.start} – {entry.end ?? 'present'}
        </span>
      </div>
      <div>
        {entry.tags.map((tag) => (
          <span className="tag" key={tag}>
            {tag}
          </span>
        ))}
      </div>
    </li>
  )
}
