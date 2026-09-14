import type { Application } from '@/api/types'
import StaleBadge from '@/components/StaleBadge'
import { Badge } from '@/components/ui/badge'
import { statusLabel } from '@/lib/applicationStatus'

// ApplicationStatusBadges is the at-a-glance state of a Job Listing's
// Application — needs-attention, Status and follow-up-overdue — rendered
// identically on its list row and its detail page (issue #94).
export default function ApplicationStatusBadges({
  application,
  needsResolve,
}: {
  application: Application
  needsResolve: boolean
}) {
  return (
    <>
      {needsResolve && (
        <Badge variant="outline" className="border-unresolved text-unresolved">
          Needs attention
        </Badge>
      )}
      {application.status === 'withdrawn' ? (
        <Badge variant="outline" className="border-withdrawn text-withdrawn">
          {statusLabel[application.status]}
        </Badge>
      ) : (
        <Badge variant="secondary">{statusLabel[application.status]}</Badge>
      )}
      {application.isStale && <StaleBadge />}
    </>
  )
}
