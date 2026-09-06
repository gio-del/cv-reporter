import type { RALRange } from '@/api/types'
import { cn } from '@/lib/utils'

const sourceLabel: Record<RALRange['source'], string> = {
  stated: 'Stated in the Job Description',
  estimated: "Claude's estimate — not a fact",
  'n/a': 'Not found',
  unresolved: "Couldn't check — retry",
}

const sourceBorderClass: Record<RALRange['source'], string> = {
  stated: 'border-ral-stated',
  estimated: 'border-ral-estimated',
  'n/a': 'border-border',
  unresolved: 'border-unresolved',
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
    <div className={cn('mb-4 inline-block rounded-xl border bg-card px-4 py-2', sourceBorderClass[ral.source])}>
      <strong className="font-semibold">RAL Range: {figure ?? 'Not available'}</strong>
      <div className="text-sm text-muted-foreground">{sourceLabel[ral.source]}</div>
    </div>
  )
}
