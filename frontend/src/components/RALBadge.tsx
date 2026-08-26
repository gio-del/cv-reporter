import type { RALRange } from '../api/types'

const sourceLabel: Record<RALRange['source'], string> = {
  stated: 'Stated in the Job Description',
  estimated: "Claude's estimate — not a fact",
  'n/a': 'Not found',
}

// Always labels the RAL Range's source, so the user is never misled into
// treating a guess as a fact (story 8, CONTEXT.md's RAL Range entry).
export default function RALBadge({ ral }: { ral: RALRange }) {
  const figure =
    ral.source === 'n/a' || ral.min == null || ral.max == null
      ? null
      : ral.min === ral.max
        ? `${ral.currency ?? ''} ${ral.min.toLocaleString()}`.trim()
        : `${ral.currency ?? ''} ${ral.min.toLocaleString()} – ${ral.max.toLocaleString()}`.trim()

  return (
    <div className={`ral-badge ral-badge-${ral.source === 'n/a' ? 'na' : ral.source}`}>
      <strong>RAL Range: {figure ?? 'Not available'}</strong>
      <div className="ral-badge-source">{sourceLabel[ral.source]}</div>
    </div>
  )
}
