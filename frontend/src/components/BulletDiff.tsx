import { diffWords } from 'diff'

// Renders source -> rewritten as an inline word diff, so the user can spot
// drift from Master Data facts without re-reading both bullets in full
// (story 4).
export default function BulletDiff({ source, rewritten }: { source: string; rewritten: string }) {
  if (source === rewritten) {
    return <span>{rewritten}</span>
  }

  const parts = diffWords(source, rewritten)
  return (
    <span>
      {parts.map((part, i) => {
        if (part.added) {
          return (
            <ins
              key={i}
              className="bg-diff-added underline decoration-diff-added-underline"
            >
              {part.value}
            </ins>
          )
        }
        if (part.removed) {
          return (
            <del key={i} className="bg-diff-removed">
              {part.value}
            </del>
          )
        }
        return <span key={i}>{part.value}</span>
      })}
    </span>
  )
}
