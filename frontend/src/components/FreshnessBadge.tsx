import type { FreshnessStatus } from '@/api/types'
import { Badge } from '@/components/ui/badge'

const freshnessLabel: Record<FreshnessStatus, string> = {
  'not-yet-checked': 'Not yet checked',
  live: 'Live',
  unreachable: 'Unreachable',
  unknown: 'Unknown',
}

// 'destructive'/'outline'/'secondary' are the existing Badge variants
// (see components/ui/badge.tsx) — reused as-is rather than inventing a new
// color for "live" (issue #59).
const freshnessBadgeVariant: Record<FreshnessStatus, 'secondary' | 'destructive' | 'outline'> = {
  'not-yet-checked': 'outline',
  live: 'secondary',
  unreachable: 'destructive',
  unknown: 'outline',
}

// FreshnessBadge is a Job Listing's freshness state (ADR-0017), shown
// read-only on its list row and alongside the Check-freshness action on its
// detail page (issue #94).
export default function FreshnessBadge({ status }: { status: FreshnessStatus }) {
  return <Badge variant={freshnessBadgeVariant[status]}>{freshnessLabel[status]}</Badge>
}
