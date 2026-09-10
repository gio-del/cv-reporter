import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkBreaks from 'remark-breaks'
import ApplicationMethodEditor from '@/components/ApplicationMethodEditor'
import ApplyGuidance from '@/components/ApplyGuidance'
import RALBadge from '@/components/RALBadge'
import StaleEntriesNotice from '@/components/StaleEntriesNotice'
import {
  checkJobListingFreshness,
  deleteJobListing,
  exportDataUrl,
  generationFileUrl,
  jobListingLogoUrl,
  listJobListings,
  resolveJobListing,
  updateApplicationContact,
  updateApplicationMethod,
  updateApplicationStatus,
} from '@/api/client'
import type { ApplicationMethod, ApplicationStatus, Contact, FreshnessStatus, JobListingWithApplication } from '@/api/types'
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
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { jobListingHeading } from '@/lib/utils'

const ALL_STATUSES_VALUE = 'all'

const statusLabel: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  tailoring: 'Tailoring',
  sent: 'Sent',
  interviewing: 'Interviewing',
  rejected: 'Rejected',
  offer: 'Offer',
  withdrawn: 'Withdrawn',
}

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

// Mirrors the backend's Status state machine (see tracking.allowedTransitions)
// so the FE only ever offers a valid next move — the backend remains the
// source of truth and re-validates on PATCH regardless (story 4).
const allowedNextStatuses: Record<ApplicationStatus, ApplicationStatus[]> = {
  saved: ['tailoring', 'withdrawn'],
  tailoring: ['sent', 'withdrawn'],
  sent: ['interviewing', 'rejected', 'withdrawn'],
  interviewing: ['rejected', 'offer', 'withdrawn'],
  rejected: ['interviewing'],
  offer: [],
  withdrawn: ['interviewing'],
}

// Moving into Rejected/Withdrawn, and Reopening out of either back to
// Interviewing, each reverse the other and need explicit confirmation
// before the PATCH fires (stories 2-4, PRD stories 3 and 5).
function needsConfirmation(from: ApplicationStatus, to: ApplicationStatus): boolean {
  return (
    to === 'rejected' ||
    to === 'withdrawn' ||
    ((from === 'rejected' || from === 'withdrawn') && to === 'interviewing')
  )
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

type RALSortChoice = 'none' | 'asc' | 'desc'

interface AppliedRALFilter {
  min: number
  max: number
  currency: string
}

export default function JobListingsListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [listings, setListings] = useState<JobListingWithApplication[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)
  const [resolveError, setResolveError] = useState<string | null>(null)
  const [resolvingId, setResolvingId] = useState<string | null>(null)
  const [freshnessError, setFreshnessError] = useState<string | null>(null)
  const [checkingFreshnessId, setCheckingFreshnessId] = useState<string | null>(null)
  const [pendingChange, setPendingChange] = useState<PendingStatusChange | null>(null)
  const [expandedDescriptions, setExpandedDescriptions] = useState<Set<string>>(new Set())
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null)
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Filter state lives in the URL (story 7: bookmarkable/reload-safe views).
  const statusFilter = (searchParams.get('status') as ApplicationStatus | null) ?? ''
  const companyFilter = searchParams.get('company') ?? ''
  const savedFromFilter = searchParams.get('savedFrom') ?? ''
  const savedToFilter = searchParams.get('savedTo') ?? ''
  const hasActiveFilter = Boolean(statusFilter || companyFilter || savedFromFilter || savedToFilter)

  function setFilter(key: 'status' | 'company' | 'savedFrom' | 'savedTo', value: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) {
        next.set(key, value)
      } else {
        next.delete(key)
      }
      return next
    })
  }

  function clearFilters() {
    setSearchParams(new URLSearchParams())
  }

  // RAL Range sort/filter (issue #51): appliedRALFilter/ralSort together
  // drive the fetch below, so applying a filter or sort doesn't get
  // silently reset by anything else on the page (story 9). knownCurrencies
  // is (re)derived only from an *unfiltered* fetch, so it keeps reflecting
  // every currency actually in use even while a currency-scoped filter is
  // active (story 8, currency-handling decision).
  const [ralSort, setRalSort] = useState<RALSortChoice>('none')
  const [ralMinInput, setRalMinInput] = useState('')
  const [ralMaxInput, setRalMaxInput] = useState('')
  const [ralCurrency, setRalCurrency] = useState('EUR')
  const [knownCurrencies, setKnownCurrencies] = useState<string[]>([])
  const [appliedRALFilter, setAppliedRALFilter] = useState<AppliedRALFilter | null>(null)
  const [ralFilterError, setRalFilterError] = useState<string | null>(null)

  useEffect(() => {
    listJobListings({
      status: statusFilter || undefined,
      company: companyFilter || undefined,
      savedFrom: savedFromFilter || undefined,
      savedTo: savedToFilter || undefined,
      sortByRAL: ralSort === 'none' ? undefined : ralSort,
      ralMin: appliedRALFilter?.min,
      ralMax: appliedRALFilter?.max,
      ralCurrency: appliedRALFilter?.currency,
    })
      .then((l) => {
        setListings(l ?? [])
        if (!appliedRALFilter) {
          const currencies = Array.from(
            new Set(
              (l ?? [])
                .map((x) => x.jobListing.ral)
                .filter((ral) => ral.source === 'stated' || ral.source === 'estimated')
                .map((ral) => ral.currency)
                .filter((currency): currency is string => Boolean(currency)),
            ),
          ).sort()
          setKnownCurrencies(currencies)
          if (currencies.length === 1) setRalCurrency(currencies[0])
        }
      })
      .catch((e) => setError(e.message))
  }, [statusFilter, companyFilter, savedFromFilter, savedToFilter, ralSort, appliedRALFilter])

  function handleApplyRALFilter() {
    setRalFilterError(null)
    const min = ralMinInput.trim() === '' ? 0 : Number(ralMinInput)
    const max = ralMaxInput.trim() === '' ? Number.MAX_SAFE_INTEGER : Number(ralMaxInput)
    if (Number.isNaN(min) || Number.isNaN(max)) {
      setRalFilterError('Min/Max RAL must be numbers.')
      return
    }
    if (min > max) {
      setRalFilterError('Min RAL must not exceed Max RAL.')
      return
    }
    setAppliedRALFilter({ min, max, currency: ralCurrency || knownCurrencies[0] || 'EUR' })
  }

  function handleClearRALFilter() {
    setRalMinInput('')
    setRalMaxInput('')
    setRalFilterError(null)
    setAppliedRALFilter(null)
  }

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

  async function handleCheckFreshness(jobListingId: string) {
    setFreshnessError(null)
    setCheckingFreshnessId(jobListingId)
    try {
      const updated = await checkJobListingFreshness(jobListingId)
      setListings((prev) =>
        prev ? prev.map((l) => (l.jobListing.id === jobListingId ? { ...l, jobListing: updated } : l)) : prev,
      )
    } catch (err) {
      setFreshnessError(err instanceof Error ? err.message : String(err))
    } finally {
      setCheckingFreshnessId(null)
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
        <div className="flex items-center gap-2">
          <Button asChild variant="outline">
            <a href={exportDataUrl()} download>
              Export data
            </a>
          </Button>
          <Button asChild>
            <Link to="/jobs/new">+ Save Job Listing</Link>
          </Button>
        </div>
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

      {freshnessError && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {freshnessError}
        </p>
      )}

      {deleteError && (
        <p role="alert" className="mb-4 font-medium text-destructive">
          {deleteError}
        </p>
      )}

      <div className="sticky top-0 z-10 mb-4 flex flex-wrap items-end gap-3 rounded-xl border border-border bg-card/95 p-3 backdrop-blur">
        <div className="flex flex-col gap-1">
          <label htmlFor="filter-status" className="text-xs font-medium text-muted-foreground">
            Status
          </label>
          <Select
            value={statusFilter || ALL_STATUSES_VALUE}
            onValueChange={(value) => setFilter('status', value === ALL_STATUSES_VALUE ? '' : value)}
          >
            <SelectTrigger id="filter-status" size="sm" className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_STATUSES_VALUE}>All statuses</SelectItem>
              {(Object.keys(statusLabel) as ApplicationStatus[]).map((status) => (
                <SelectItem key={status} value={status}>
                  {statusLabel[status]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="filter-company" className="text-xs font-medium text-muted-foreground">
            Company
          </label>
          <Input
            id="filter-company"
            className="h-9 w-48"
            placeholder="e.g. acme"
            value={companyFilter}
            onChange={(e) => setFilter('company', e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="filter-saved-from" className="text-xs font-medium text-muted-foreground">
            Saved from
          </label>
          <Input
            id="filter-saved-from"
            type="date"
            className="h-9"
            value={savedFromFilter}
            onChange={(e) => setFilter('savedFrom', e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="filter-saved-to" className="text-xs font-medium text-muted-foreground">
            Saved to
          </label>
          <Input
            id="filter-saved-to"
            type="date"
            className="h-9"
            value={savedToFilter}
            onChange={(e) => setFilter('savedTo', e.target.value)}
          />
        </div>
        {hasActiveFilter && (
          <Button size="sm" variant="ghost" onClick={clearFilters}>
            Clear filters
          </Button>
        )}
      </div>

      <div className="mb-4 flex flex-wrap items-end gap-3 rounded-xl border border-border bg-card p-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="ral-sort" className="text-xs font-medium text-muted-foreground">
            Sort by RAL Range
          </label>
          <Select value={ralSort} onValueChange={(value) => setRalSort(value as RALSortChoice)}>
            <SelectTrigger id="ral-sort" size="sm" className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">Default (newest first)</SelectItem>
              <SelectItem value="desc">RAL: high to low</SelectItem>
              <SelectItem value="asc">RAL: low to high</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="ral-min" className="text-xs font-medium text-muted-foreground">
            Min RAL
          </label>
          <Input
            id="ral-min"
            type="number"
            inputMode="numeric"
            className="w-28"
            placeholder="e.g. 40000"
            value={ralMinInput}
            onChange={(e) => setRalMinInput(e.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label htmlFor="ral-max" className="text-xs font-medium text-muted-foreground">
            Max RAL
          </label>
          <Input
            id="ral-max"
            type="number"
            inputMode="numeric"
            className="w-28"
            placeholder="e.g. 60000"
            value={ralMaxInput}
            onChange={(e) => setRalMaxInput(e.target.value)}
          />
        </div>

        {knownCurrencies.length > 1 && (
          <div className="flex flex-col gap-1">
            <label htmlFor="ral-currency" className="text-xs font-medium text-muted-foreground">
              Currency
            </label>
            <Select value={ralCurrency} onValueChange={setRalCurrency}>
              <SelectTrigger id="ral-currency" size="sm" className="w-24">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {knownCurrencies.map((currency) => (
                  <SelectItem key={currency} value={currency}>
                    {currency}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <Button
          size="sm"
          variant="outline"
          onClick={handleApplyRALFilter}
          disabled={ralMinInput.trim() === '' && ralMaxInput.trim() === ''}
        >
          Apply RAL filter
        </Button>

        {appliedRALFilter && (
          <Button size="sm" variant="ghost" onClick={handleClearRALFilter}>
            Clear RAL filter
          </Button>
        )}

        {ralFilterError && (
          <p role="alert" className="text-sm font-medium text-destructive">
            {ralFilterError}
          </p>
        )}
      </div>

      {listings.length === 0 && (hasActiveFilter || appliedRALFilter) && (
        <p>No Job Listings match the current filters.</p>
      )}
      {listings.length === 0 && !hasActiveFilter && !appliedRALFilter && <p>No Job Listings saved yet.</p>}

      <ul className="flex flex-col gap-3">
        {listings.map(({ jobListing, application }) => {
          const nextStatuses = allowedNextStatuses[application.status]
          const needsResolve = jobListing.ral.source === 'unresolved' || application.method.kind === 'unresolved'
          const isDescriptionExpanded = expandedDescriptions.has(jobListing.id)
          return (
            <li key={jobListing.id} className="rounded-xl border border-border bg-card px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex min-w-0 items-center gap-2">
                  {jobListing.logo && (
                    <img
                      src={jobListingLogoUrl(jobListing.id)}
                      alt=""
                      className="h-8 w-8 shrink-0 rounded object-contain"
                    />
                  )}
                  <strong className="break-words font-semibold">{jobListingHeading(jobListing)}</strong>
                </span>
                <div className="flex flex-wrap items-center gap-2">
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
                  {application.isStale && (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Badge variant="outline" className="border-accent text-accent">
                          Follow-up overdue
                        </Badge>
                      </TooltipTrigger>
                      <TooltipContent>No Status change in over 14 days</TooltipContent>
                    </Tooltip>
                  )}
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
              {jobListing.url && (
                <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
                  <Badge variant={freshnessBadgeVariant[jobListing.freshnessStatus]}>
                    {freshnessLabel[jobListing.freshnessStatus]}
                  </Badge>
                  {jobListing.freshnessCheckedAt && (
                    <span className="text-muted-foreground">
                      Checked {new Date(jobListing.freshnessCheckedAt).toLocaleString()}
                    </span>
                  )}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleCheckFreshness(jobListing.id)}
                        disabled={checkingFreshnessId === jobListing.id}
                      >
                        {checkingFreshnessId === jobListing.id ? 'Checking…' : 'Check freshness'}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Re-fetches the source URL to see if the posting is still live</TooltipContent>
                  </Tooltip>
                </div>
              )}
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
                      className="mt-2 overflow-x-auto rounded-lg border border-border bg-muted/40 p-3 text-sm
                        break-words [&_a]:underline [&_ol]:list-decimal [&_ol]:pl-5 [&_p+p]:mt-2 [&_p+ul]:mt-2
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
              <StaleEntriesNotice generations={application.generations} />
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
                : pendingChange?.to === 'withdrawn'
                  ? 'Mark this Application Withdrawn?'
                  : 'Reopen this Application to Interviewing?'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {pendingChange?.to === 'rejected'
                ? `${pendingChange.company} will be marked Rejected. You can Reopen it back to Interviewing later if this turns out to be premature.`
                : pendingChange?.to === 'withdrawn'
                  ? `${pendingChange.company} will be marked Withdrawn. You can Reopen it back to Interviewing later if you change your mind.`
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
              {pendingChange?.to === 'rejected'
                ? 'Yes, mark Rejected'
                : pendingChange?.to === 'withdrawn'
                  ? 'Yes, mark Withdrawn'
                  : 'Yes, reopen'}
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
