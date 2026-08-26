import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listEntries } from '../api/client'
import type { Entry } from '../api/types'

export default function EntriesListPage() {
  const [entries, setEntries] = useState<Entry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listEntries()
      .then(setEntries)
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
      <h1>Master Data</h1>

      <p>
        <Link to="/entries/new">+ New Entry</Link> · <Link to="/profile">Profile</Link> ·{' '}
        <Link to="/snippets">Cover Letter Snippets</Link> · <Link to="/generate">Generate a Tailored CV</Link>
      </p>

      <section>
        <h2>Experience</h2>
        {[...byEmployer.entries()].map(([employer, employerEntries]) => (
          <div key={employer}>
            <h3>{employer}</h3>
            <ul>
              {employerEntries.map((entry) => (
                <EntryListItem key={entry.id} entry={entry} label={entry.client ?? entry.role ?? entry.id} />
              ))}
            </ul>
          </div>
        ))}
      </section>

      <section>
        <h2>Projects</h2>
        <ul>
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
      <Link to={`/entries/${entry.id}`}>{label}</Link>{' '}
      <span>
        {entry.start} – {entry.end ?? 'present'}
      </span>
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
