import { useEffect, useState } from 'react'
import { getUsageSummary } from '@/api/client'
import type { UsageSummary } from '@/api/types'

// UsageIndicator shows the running-total Claude API cost across every
// Generation and standalone (RAL/Method/Contact) call ever recorded
// (issue #39: cost/usage visibility, story 11) — a small, always-visible
// reminder rather than a page someone has to remember to check. When the
// backend reports some usage is known to be missing (issue #102), the
// figure is shown as a lower bound, with the reason on hover.
export default function UsageIndicator() {
  const [usage, setUsage] = useState<UsageSummary | null>(null)

  useEffect(() => {
    getUsageSummary()
      .then(setUsage)
      .catch(() => {
        // Best-effort: a stale/missing total shouldn't disrupt navigation.
      })
  }, [])

  if (usage === null || typeof usage.estimatedCostUsd !== 'number') return null

  const cost = usage.estimatedCostUsd.toFixed(2)

  if (usage.incomplete) {
    const reason = usage.incompleteReason || 'Some Claude API usage could not be recorded.'
    return (
      <span
        className="ml-auto text-xs text-unresolved"
        title={`Estimated lifetime Claude API cost — incomplete, so this is a lower bound.\n${reason}`}
      >
        ≥ ${cost} Claude API spend (incomplete)
      </span>
    )
  }

  return (
    <span className="ml-auto text-xs text-muted-foreground" title="Estimated lifetime Claude API cost">
      ~${cost} Claude API spend
    </span>
  )
}
