import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import ApplicationMethodEditor from '@/components/ApplicationMethodEditor'
import ApplyGuidance from '@/components/ApplyGuidance'
import RALBadge from '@/components/RALBadge'
import {
  deleteJobListing,
  generationFileUrl,
  jobListingLogoUrl,
  listJobListings,
  resolveJobListing,
  updateApplicationContact,
  updateApplicationMethod,
  updateApplicationStatus,
} from '@/api/client'
import type { ApplicationMethod, ApplicationStatus, Contact, JobListingWithApplication } from '@/api/types'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { jobListingHeading } from '@/lib/utils'

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
  rejected: ['interviewing'],
  offer: [],
}

// Moving into Rejected, and Reopening out of it, each reverse the other and
// need explicit confirmation before the PATCH fires (stories 2-4).
function needsConfirmation(from: ApplicationStatus, to: ApplicationStatus): boolean {
  return to === 'rejected' || (from === 'rejected' && to === 'interviewing')
}

interface PendingStatusChange {
  jobListingId: string
  company: string
  from: ApplicationStatus
  to: ApplicationStatus
}

interface PendingDelete {
  jobListingId: string
  heading: string
}

export default function JobListingsListPage() {
  const [listings, setListings] = useState<JobListingWithApplication[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)
  const [resolveError, setResolveError] = useState<string | null>(null)
  const [resolvingId, setResolvingId] = useState<string | null>(null)
  const [pendingChange, setPendingChange] = useState<PendingStatusChange | null>(null)
  const [expandedDescriptions, setExpandedDescriptions] = useState<Set<string>>(new Set())
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

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

  function handleStatusSelect(jobListingId: string, company: string, from: ApplicationStatus, to: ApplicationStatus) {
    if (needsConfirmation(from, to)) {
      setPendingChange({ jobListingId, company, from, to })
      return
    }
    handleStatusChange(jobListingId, to)
  }

  function handleConfirmPendingChange() {
    if (!pendingChange) return
    handleStatusChange(pendingChange.jobListingId, pendingChange.to)
    setPendingChange(null)
  }

  function toggleDescription(jobListingId: string) {
    setExpandedDescriptions((prev) => {
      const next = new Set(prev)
      if (next.has(jobListingId)) {
        next.delete(jobListingId)
      } else {
        next.add(jobListingId)
      }
      return next
    })
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

  async function handleConfirmDelete() {
    if (!pendingDelete) return
    const { jobListingId } = pendingDelete
    setDeleteError(null)
    setDeletingId(jobListingId)
    try {
      await deleteJobListing(jobListingId)
      setListings((prev) => (prev ? prev.filter((l) => l.jobListing.id !== jobListingId) : prev))
      setPendingDelete(null)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : String(err))
    } finally {
      setDeletingId(null)
    }
  }

  async function handleResolve(jobListingId: string) {
    setResolveError(null)
    setResolvingId(jobListingId)
    try {
      const resolved = await resolveJobListing(jobListingId)
      setListings((prev) => (prev ? prev.map((l) => (l.jobListing.id === jobListingId ? resolved : l)) : prev))
    } catch (err) {
      setResolveError(err instanceof Error ? err.message : String(err))
    } finally {
      setResolvingId(null)
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

      {resolveError && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {resolveError}
        </p>
      )}

      {deleteError && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {deleteError}
        </p>
      )}

      {listings.length === 0 && <p>No Job Listings saved yet.</p>}

      <ul className="flex flex-col gap-3">
        {listings.map(({ jobListing, application }) => {
          const nextStatuses = allowedNextStatuses[application.status]
          const needsResolve = jobListing.ral.source === 'unresolved' || application.method.kind === 'unresolved'
          const isDescriptionExpanded = expandedDescriptions.has(jobListing.id)
          return (
            <li key={jobListing.id} className="rounded-xl border border-border bg-card px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex items-center gap-2">
                  {jobListing.logo && (
                    <img
                      src={jobListingLogoUrl(jobListing.id)}
                      alt=""
                      className="h-8 w-8 rounded object-contain"
                    />
                  )}
                  <strong className="font-semibold">{jobListingHeading(jobListing)}</strong>
                </span>
                <div className="flex items-center gap-2">
                  {needsResolve && (
                    <Badge variant="outline" className="border-unresolved text-unresolved">
                      Needs attention
                    </Badge>
                  )}
                  <Badge variant="secondary">{statusLabel[application.status]}</Badge>
                  {needsResolve && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleResolve(jobListing.id)}
                          disabled={resolvingId === jobListing.id}
                        >
                          {resolvingId === jobListing.id ? 'Resolving…' : 'Resolve'}
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>Retries RAL Range and Application Method resolution</TooltipContent>
                    </Tooltip>
                  )}
                  {nextStatuses.length > 0 && (
                    <Select
                      value=""
                      onValueChange={(value) =>
                        handleStatusSelect(
                          jobListing.id,
                          jobListing.company,
                          application.status,
                          value as ApplicationStatus,
                        )
                      }
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
              {jobListing.jobDescription && (
                <div className="mt-2">
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-auto p-0 text-sm font-normal underline-offset-2 hover:underline"
                    onClick={() => toggleDescription(jobListing.id)}
                  >
                    {isDescriptionExpanded ? 'Hide description' : 'View description'}
                  </Button>
                  {isDescriptionExpanded && (
                    <div
                      className="mt-2 rounded-lg border border-border bg-muted/40 p-3 text-sm
                        [&_a]:underline [&_ol]:list-decimal [&_ol]:pl-5 [&_p+p]:mt-2 [&_p+ul]:mt-2
                        [&_strong]:font-semibold [&_ul]:list-disc [&_ul]:pl-5"
                    >
                      <ReactMarkdown remarkPlugins={[remarkBreaks]}>{jobListing.jobDescription}</ReactMarkdown>
                    </div>
                  )}
                </div>
              )}
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
                <Button
                  size="sm"
                  variant="outline"
                  className="ml-auto"
                  onClick={() =>
                    setPendingDelete({ jobListingId: jobListing.id, heading: jobListingHeading(jobListing) })
                  }
                  disabled={deletingId === jobListing.id}
                >
                  Delete
                </Button>
              </div>
            </li>
          )
        })}
      </ul>

      <AlertDialog open={pendingChange !== null} onOpenChange={(open) => !open && setPendingChange(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pendingChange?.to === 'rejected'
                ? 'Mark this Application Rejected?'
                : 'Reopen this Application to Interviewing?'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingChange?.to === 'rejected'
                ? `${pendingChange.company} will be marked Rejected. You can Reopen it back to Interviewing later if this turns out to be premature.`
                : `${pendingChange?.company} will move back to Interviewing.`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault()
                handleConfirmPendingChange()
              }}
            >
              {pendingChange?.to === 'rejected' ? 'Yes, mark Rejected' : 'Yes, reopen'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={pendingDelete !== null} onOpenChange={(open) => !open && setPendingDelete(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {pendingDelete?.heading}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will also remove its Application (Status, Method, Contact, and Generation history). This action
              cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deletingId !== null}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={(e) => {
                e.preventDefault()
                handleConfirmDelete()
              }}
              disabled={deletingId !== null}
            >
              {deletingId !== null ? 'Deleting…' : 'Yes, delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
