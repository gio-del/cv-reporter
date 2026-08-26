import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import RALBadge from '@/components/RALBadge'
import { listJobListings } from '@/api/client'
import type { ApplicationStatus, JobListingWithApplication } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

const statusLabel: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  tailoring: 'Tailoring',
  sent: 'Sent',
  interviewing: 'Interviewing',
  rejected: 'Rejected',
  offer: 'Offer',
}

export default function JobListingsListPage() {
  const [listings, setListings] = useState<JobListingWithApplication[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listJobListings()
      .then((l) => setListings(l ?? []))
      .catch((e) => setError(e.message))
  }, [])

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

      {listings.length === 0 && <p>No Job Listings saved yet.</p>}

      <ul className="flex flex-col gap-3">
        {listings.map(({ jobListing, application }) => (
          <li key={jobListing.id} className="rounded-xl border border-border bg-card px-4 py-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <strong className="font-semibold">{jobListing.company}</strong>
              <Badge variant="secondary">{statusLabel[application.status]}</Badge>
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
        ))}
      </ul>
    </>
  )
}
