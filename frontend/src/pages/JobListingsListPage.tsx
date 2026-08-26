import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import RALBadge from '@/components/RALBadge'
import { listJobListings, updateApplicationStatus } from '@/api/client'
import type { ApplicationStatus, JobListingWithApplication } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const statusLabel: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  tailoring: 'Tailoring',
  sent: 'Sent',
  interviewing: 'Interviewing',
  rejected: 'Rejected',
  offer: 'Offer',
}

// Mirrors the backend's Status state machine (see tracking.allowedTransitions)
// so the FE only ever offers a valid next move — the backend remains the
// source of truth and re-validates on PATCH regardless (story 4).
const allowedNextStatuses: Record<ApplicationStatus, ApplicationStatus[]> = {
  saved: ['tailoring'],
  tailoring: ['sent'],
  sent: ['interviewing', 'rejected'],
  interviewing: ['rejected', 'offer'],
  rejected: [],
  offer: [],
}

export default function JobListingsListPage() {
  const [listings, setListings] = useState<JobListingWithApplication[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)

  useEffect(() => {
    listJobListings()
      .then((l) => setListings(l ?? []))
      .catch((e) => setError(e.message))
  }, [])

  async function handleStatusChange(jobListingId: string, status: ApplicationStatus) {
    setStatusError(null)
    setUpdatingId(jobListingId)
    try {
      const application = await updateApplicationStatus(jobListingId, status)
      setListings((prev) =>
        prev ? prev.map((l) => (l.jobListing.id === jobListingId ? { ...l, application } : l)) : prev,
      )
    } catch (err) {
      setStatusError(err instanceof Error ? err.message : String(err))
    } finally {
      setUpdatingId(null)
    }
  }

  if (error)
    return (
      <p role="alert" className="font-medium text-destructive">
        {error}
      </p>
    )
  if (!listings) return <p>Loading…</p>

  return (
    <>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="mb-0">Job Listings</h1>
        <Button asChild>
          <Link to="/jobs/new">+ Save Job Listing</Link>
        </Button>
      </div>

      {statusError && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {statusError}
        </p>
      )}

      {listings.length === 0 && <p>No Job Listings saved yet.</p>}

      <ul className="flex flex-col gap-3">
        {listings.map(({ jobListing, application }) => {
          const nextStatuses = allowedNextStatuses[application.status]
          return (
            <li key={jobListing.id} className="rounded-xl border border-border bg-card px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <strong className="font-semibold">{jobListing.company}</strong>
                <div className="flex items-center gap-2">
                  <Badge variant="secondary">{statusLabel[application.status]}</Badge>
                  {nextStatuses.length > 0 && (
                    <Select
                      value=""
                      onValueChange={(value) => handleStatusChange(jobListing.id, value as ApplicationStatus)}
                      disabled={updatingId === jobListing.id}
                    >
                      <SelectTrigger size="sm" aria-label={`Move ${jobListing.company} to a new status`}>
                        <SelectValue placeholder="Move to…" />
                      </SelectTrigger>
                      <SelectContent>
                        {nextStatuses.map((status) => (
                          <SelectItem key={status} value={status}>
                            {statusLabel[status]}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                </div>
              </div>
              <p className="mb-0 text-sm text-muted-foreground">
                Saved {new Date(jobListing.savedAt).toLocaleDateString()}
                {jobListing.url && (
                  <>
                    {' · '}
                    <a href={jobListing.url} target="_blank" rel="noreferrer">
                      View posting
                    </a>
                  </>
                )}
              </p>
              <div className="mt-2">
                <RALBadge ral={jobListing.ral} />
              </div>
            </li>
          )
        })}
      </ul>
    </>
  )
}
