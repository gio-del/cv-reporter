import { useEffect, useState } from 'react'
import { getUsageSummary } from '@/api/client'

// UsageIndicator shows the running-total Claude API cost across every
// Generation and standalone (RAL/Method/Contact) call ever recorded
// (issue #39: cost/usage visibility, story 11) — a small, always-visible
// reminder rather than a page someone has to remember to check.
export default function UsageIndicator() {
  const [totalCostUsd, setTotalCostUsd] = useState<number | null>(null)

  useEffect(() => {
    getUsageSummary()
      .then((usage) => setTotalCostUsd(usage.estimatedCostUsd))
      .catch(() => {
        // Best-effort: a stale/missing total shouldn't disrupt navigation.
      })
  }, [])

  if (totalCostUsd === null) return null

  return (
    <span className="ml-auto text-xs text-muted-foreground" title="Estimated lifetime Claude API cost">
      ~${totalCostUsd.toFixed(2)} Claude API spend
    </span>
  )
}
