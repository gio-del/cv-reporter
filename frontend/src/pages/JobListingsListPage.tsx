import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import ApplicationMethodEditor from '@/components/ApplicationMethodEditor'
import ApplyGuidance from '@/components/ApplyGuidance'
import RALBadge from '@/components/RALBadge'
import {
  generationFileUrl,
  listJobListings,
  updateApplicationContact,
  updateApplicationMethod,
  updateApplicationStatus,
} from '@/api/client'
import type { ApplicationMethod, ApplicationStatus, Contact, JobListingWithApplication } from '@/api/types'
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

  async function handleMethodChange(jobListingId: string, method: ApplicationMethod) {
    const application = await updateApplicationMethod(jobListingId, method)
    setListings((prev) =>
      prev ? prev.map((l) => (l.jobListing.id === jobListingId ? { ...l, application } : l)) : prev,
    )
  }

  async function handleContactChange(jobListingId: string, contact: Contact) {
    const application = await updateApplicationContact(jobListingId, contact)
    setListings((prev) =>
      prev ? prev.map((l) => (l.jobListing.id === jobListingId ? { ...l, application } : l)) : prev,
    )
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
                <ApplicationMethodEditor
                  method={application.method}
                  onSave={(method) => handleMethodChange(jobListing.id, method)}
                />
              </div>
              <div className="mt-2">
                <ApplyGuidance
                  jobListing={jobListing}
                  application={application}
                  onSaveContact={(contact) => handleContactChange(jobListing.id, contact)}
                />
              </div>
              <div className="mt-2">
                <RALBadge ral={jobListing.ral} />
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
                <Button asChild size="sm" variant="outline">
                  <Link to={`/jobs/${jobListing.id}/generate`}>
                    {application.generations?.length ? 'Regenerate CV' : 'Generate CV'}
                  </Link>
                </Button>
                {application.generations && application.generations.length > 0 && (
                  <span className="text-muted-foreground">
                    {application.generations.length} generation{application.generations.length > 1 ? 's' : ''} · latest:{' '}
                    <a
                      href={generationFileUrl(
                        application.generations[application.generations.length - 1].slug,
                        'cv.pdf',
                      )}
                      target="_blank"
                      rel="noreferrer"
                    >
                      CV
                    </a>
                    {application.generations[application.generations.length - 1].coverLetterPath && (
                      <>
                        {' · '}
                        <a
                          href={generationFileUrl(
                            application.generations[application.generations.length - 1].slug,
                            'cover-letter.pdf',
                          )}
                          target="_blank"
                          rel="noreferrer"
                        >
                          Cover Letter
                        </a>
                      </>
                    )}
                  </span>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </>
  )
}
