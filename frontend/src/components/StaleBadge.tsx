import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

// StaleBadge is the one marker for an Application whose isStale flag is set
// (issue #37): shown on Job Listing rows, the detail page and the
// Applications view (issue #95) alike.
export default function StaleBadge() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant="outline" className="border-accent text-accent">
          Follow-up overdue
        </Badge>
      </TooltipTrigger>
      <TooltipContent>No Status change in over 14 days</TooltipContent>
    </Tooltip>
  )
}
