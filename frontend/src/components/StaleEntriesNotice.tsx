import type { GenerationRecord } from '@/api/types'

function formatDate(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString()
}

// Renders a non-blocking "this CV may be outdated" hint for every past
// Generation whose staleEntries is non-empty (issue #52) — purely
// informational, never blocking viewing, resending, or regenerating (story
// 4). The most recent Generation is called "Latest CV" since that's the
// version the user would actually send (story 5); older, superseded
// Generations still get the same treatment for accurate context if the
// user looks back at them (story 6). Names the affected Entries by
// employer/client + role, as the API already formats them, rather than raw
// ids (story 3).
export default function StaleEntriesNotice({ generations }: { generations: GenerationRecord[] | undefined }) {
  if (!generations || generations.length === 0) return null

  const flagged = generations
    .map((record, index) => ({ record, isLatest: index === generations.length - 1 }))
    .filter(({ record }) => record.staleEntries && record.staleEntries.length > 0)

  if (flagged.length === 0) return null

  return (
    <div className="mt-2 space-y-1 text-sm text-ral-estimated">
      {flagged.map(({ record, isLatest }) => (
        <p key={record.slug + record.createdAt} className="mb-0">
          {isLatest ? 'Latest CV' : `CV from ${formatDate(record.createdAt)}`} may be outdated — edited in Master
          Data since generation: {record.staleEntries!.join(', ')}.
        </p>
      ))}
    </div>
  )
}
