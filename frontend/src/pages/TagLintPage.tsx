import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getTagLint } from '@/api/client'
import type { TagLintGroup, TagLintOccurrence, TagLintReport } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export default function TagLintPage() {
  const [report, setReport] = useState<TagLintReport | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getTagLint()
      .then(setReport)
      .catch((e) => setError(e.message))
  }, [])

  if (error)
    return (
      <p role="alert" className="font-medium text-destructive">
        {error}
      </p>
    )
  if (!report) return <p>Loading…</p>

  const confidentGroups = report.groups.filter((g) => g.confidence === 'confident')
  const suggestedGroups = report.groups.filter((g) => g.confidence === 'suggested')

  return (
    <>
      <div className="mb-6">
        <h1 className="mb-1">Tag Lint</h1>
        <p className="mb-0 text-muted-foreground">
          Near-duplicate Tag spellings found across every Entry under <code>data/experience/</code> and{' '}
          <code>data/projects/</code>. This is a read-only report — reconcile a group by editing the affected
          Entries yourself; nothing here rewrites Master Data.
        </p>
      </div>

      {report.groups.length === 0 && report.singletons.length === 0 && <p>No Tags found.</p>}

      {confidentGroups.length > 0 && (
        <section className="mb-8">
          <h2>Confident groups</h2>
          <p className="text-sm text-muted-foreground">
            Tags that differ only by case, or are known aliases of each other (e.g. Go/Golang).
          </p>
          <div className="flex flex-col gap-3">
            {confidentGroups.map((group) => (
              <TagGroupCard key={group.key} group={group} />
            ))}
          </div>
        </section>
      )}

      {suggestedGroups.length > 0 && (
        <section className="mb-8">
          <h2>Suggested groups</h2>
          <p className="text-sm text-muted-foreground">
            Tags that are merely close by spelling — a possible typo, not a certain match. Give these more
            scrutiny before editing anything.
          </p>
          <div className="flex flex-col gap-3">
            {suggestedGroups.map((group) => (
              <TagGroupCard key={group.key} group={group} />
            ))}
          </div>
        </section>
      )}

      {report.singletons.length > 0 && (
        <section>
          <h2>Other tags</h2>
          <p className="text-sm text-muted-foreground">
            Tags with no near-duplicate found — shown for completeness, not flagged as an issue.
          </p>
          <div className="flex flex-wrap gap-1">
            {report.singletons.map((occurrence, i) => (
              <OccurrenceBadge key={`${occurrence.tag}-${occurrence.entryId}-${i}`} occurrence={occurrence} />
            ))}
          </div>
        </section>
      )}
    </>
  )
}

function TagGroupCard({ group }: { group: TagLintGroup }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{group.key}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-1">
        {group.occurrences.map((occurrence, i) => (
          <OccurrenceBadge key={`${occurrence.tag}-${occurrence.entryId}-${i}`} occurrence={occurrence} />
        ))}
      </CardContent>
    </Card>
  )
}

function OccurrenceBadge({ occurrence }: { occurrence: TagLintOccurrence }) {
  return (
    <Link to={`/entries/${occurrence.entryId}`} className="no-underline">
      <Badge variant="secondary" title={`${occurrence.entryType} · ${occurrence.entryId}`}>
        {occurrence.tag}
      </Badge>
    </Link>
  )
}
