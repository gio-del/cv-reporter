import { useEffect, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import ApplicationStatusBadges from '@/components/ApplicationStatusBadges'
import ApplicationStatusControl from '@/components/ApplicationStatusControl'
import FreshnessBadge from '@/components/FreshnessBadge'
import RALBadge from '@/components/RALBadge'
import { exportDataUrl, jobListingLogoUrl, listJobListings, updateApplicationStatus } from '@/api/client'
import type { ApplicationStatus, JobListingSummaryWithApplication } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { statusLabel } from '@/lib/applicationStatus'
import { jobListingHeading } from '@/lib/utils'
import type { JobListingDetailLocationState } from '@/pages/JobListingDetailPage'

const ALL_STATUSES_VALUE = 'all'

type RALSortChoice = 'none' | 'asc' | 'desc'

const RAL_SORT_CHOICES: RALSortChoice[] = ['none', 'asc', 'desc']

export default function JobListingsListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const location = useLocation()
  const [listings, setListings] = useState<JobListingSummaryWithApplication[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)

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
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      for (const key of ['status', 'company', 'savedFrom', 'savedTo']) next.delete(key)
      return next
    })
  }

  // RAL Range sort/filter (issue #51): the applied sort and bounds live in
  // the URL alongside the filters above, so a detour into one Job Listing's
  // page and back — by its back link or by browser Back — restores the view
  // as it was (issue #94, stories 5-6). Only *applied* bounds are written
  // there: the Min/Max inputs stay local until "Apply RAL filter", so typing
  // never fires a request per keystroke. A bound left blank is absent from
  // the URL and falls back to 0 / no ceiling when the request is built.
  // knownCurrencies is (re)derived only from an *unfiltered* fetch, so it
  // keeps reflecting every currency actually in use even while a
  // currency-scoped filter is active (story 8, currency-handling decision).
  const ralSortParam = searchParams.get('ralSort') as RALSortChoice | null
  const ralSort: RALSortChoice = ralSortParam && RAL_SORT_CHOICES.includes(ralSortParam) ? ralSortParam : 'none'
  const appliedRALMin = searchParams.get('ralMin') ?? ''
  const appliedRALMax = searchParams.get('ralMax') ?? ''
  const appliedRALCurrency = searchParams.get('ralCurrency') ?? ''
  const hasAppliedRALFilter = Boolean(appliedRALMin || appliedRALMax)
  const [ralMinInput, setRalMinInput] = useState(appliedRALMin)
  const [ralMaxInput, setRalMaxInput] = useState(appliedRALMax)
  const [ralCurrency, setRalCurrency] = useState(appliedRALCurrency || 'EUR')
  const [knownCurrencies, setKnownCurrencies] = useState<string[]>([])
  const [ralFilterError, setRalFilterError] = useState<string | null>(null)

  function setRalSort(choice: RALSortChoice) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (choice === 'none') {
        next.delete('ralSort')
      } else {
        next.set('ralSort', choice)
      }
      return next
    })
  }

  useEffect(() => {
    listJobListings({
      status: statusFilter || undefined,
      company: companyFilter || undefined,
      savedFrom: savedFromFilter || undefined,
      savedTo: savedToFilter || undefined,
      sortByRAL: ralSort === 'none' ? undefined : ralSort,
      ralMin: hasAppliedRALFilter ? Number(appliedRALMin || 0) : undefined,
      ralMax: hasAppliedRALFilter ? (appliedRALMax === '' ? Number.MAX_SAFE_INTEGER : Number(appliedRALMax)) : undefined,
      ralCurrency: hasAppliedRALFilter ? appliedRALCurrency || 'EUR' : undefined,
    })
      .then((l) => {
        setListings(l ?? [])
        if (!hasAppliedRALFilter) {
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
  }, [
    statusFilter,
    companyFilter,
    savedFromFilter,
    savedToFilter,
    ralSort,
    hasAppliedRALFilter,
    appliedRALMin,
    appliedRALMax,
    appliedRALCurrency,
  ])

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
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('ralMin', String(min))
      if (ralMaxInput.trim() === '') {
        next.delete('ralMax')
      } else {
        next.set('ralMax', String(max))
      }
      next.set('ralCurrency', ralCurrency || knownCurrencies[0] || 'EUR')
      return next
    })
  }

  function handleClearRALFilter() {
    setRalMinInput('')
    setRalMaxInput('')
    setRalFilterError(null)
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      for (const key of ['ralMin', 'ralMax', 'ralCurrency']) next.delete(key)
      return next
    })
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

        {hasAppliedRALFilter && (
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

      {listings.length === 0 && (hasActiveFilter || hasAppliedRALFilter) && (
        <p>No Job Listings match the current filters.</p>
      )}
      {listings.length === 0 && !hasActiveFilter && !hasAppliedRALFilter && <p>No Job Listings saved yet.</p>}

      <ul className="flex flex-col gap-3">
        {listings.map(({ jobListing, application }) => {
          const needsResolve = jobListing.ral.source === 'unresolved' || application.method.kind === 'unresolved'
          const detailState: JobListingDetailLocationState = { from: location.pathname + location.search }
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
                  <Link
                    to={`/jobs/${encodeURIComponent(jobListing.id)}`}
                    state={detailState}
                    className="break-words font-semibold no-underline hover:underline"
                  >
                    {jobListingHeading(jobListing)}
                  </Link>
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <ApplicationStatusBadges application={application} needsResolve={needsResolve} />
                  <ApplicationStatusControl
                    company={jobListing.company}
                    status={application.status}
                    disabled={updatingId === jobListing.id}
                    onMove={(next) => handleStatusChange(jobListing.id, next)}
                  />
                </div>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
                <span className="text-muted-foreground">Saved {new Date(jobListing.savedAt).toLocaleDateString()}</span>
                {jobListing.url && <FreshnessBadge status={jobListing.freshnessStatus} />}
                <Button asChild size="sm" variant="outline" className="ml-auto">
                  <Link to={`/jobs/${jobListing.id}/generate`}>
                    {application.generations?.length ? 'Regenerate CV' : 'Generate CV'}
                  </Link>
                </Button>
              </div>
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
