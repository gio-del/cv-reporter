import { describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import JobListingsListPage from './JobListingsListPage'
import type { FreshnessStatus, JobListingSummaryWithApplication } from '@/api/types'
import { listingSummaryWithApplication } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { server } from '@/test/server'

function showList(listings: JobListingSummaryWithApplication[]) {
  server.use(http.get('/api/job-listings', () => HttpResponse.json(listings)))
  return renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })
}

function freshListing(freshnessStatus: FreshnessStatus): JobListingSummaryWithApplication {
  return listingSummaryWithApplication({ url: 'https://acme.example/jobs/1', freshnessStatus })
}

// A Job Listing's freshness statuses are four distinct domain states
// (ADR-0017). "Not yet checked" and "Unknown" are the pair most easily
// collapsed into each other, and collapsing them tells the user a posting
// was checked when it never was.
describe('Job Listing freshness', () => {
  it('FreshnessBadge_NeverChecked_ReadsAsNotYetCheckedRatherThanUnknown', async () => {
    showList([freshListing('not-yet-checked')])

    expect(await screen.findByText('Not yet checked')).toBeInTheDocument()
    expect(screen.queryByText('Unknown')).not.toBeInTheDocument()
  })

  it('FreshnessBadge_CheckedButInconclusive_ReadsAsUnknownRatherThanNotYetChecked', async () => {
    showList([freshListing('unknown')])

    expect(await screen.findByText('Unknown')).toBeInTheDocument()
    expect(screen.queryByText('Not yet checked')).not.toBeInTheDocument()
  })

  it('FreshnessBadge_UnreachableUrl_FlagsTheJobListingAsUnreachable', async () => {
    showList([freshListing('unreachable')])

    expect(await screen.findByText('Unreachable')).toBeInTheDocument()
  })

  // The row's badge is read-only: the last-checked time and the Check
  // freshness action live on the Job Listing's own page (issue #94).
  it('FreshnessBadge_LiveUrl_IsShownReadOnlyOnTheRow', async () => {
    showList([
      listingSummaryWithApplication({
        url: 'https://acme.example/jobs/1',
        freshnessStatus: 'live',
        freshnessCheckedAt: '2026-02-01T09:30:00Z',
      }),
    ])

    expect(await screen.findByText('Live')).toBeInTheDocument()
    expect(screen.queryByText(/^Checked /)).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Check freshness' })).not.toBeInTheDocument()
  })

  it('FreshnessBadge_NoSourceUrl_ShowsNoFreshnessControlAtAll', async () => {
    showList([listingSummaryWithApplication({ url: undefined })])

    await screen.findByText('Acme')
    expect(screen.queryByRole('button', { name: 'Check freshness' })).not.toBeInTheDocument()
    expect(screen.queryByText('Not yet checked')).not.toBeInTheDocument()
  })
})

describe('Application staleness', () => {
  it('StaleBadge_NoStatusChangeInOverFourteenDays_FlagsTheApplicationAsOverdue', async () => {
    showList([listingSummaryWithApplication({}, { isStale: true })])

    const row = await screen.findByRole('listitem')
    expect(within(row).getByText('Follow-up overdue')).toBeInTheDocument()
  })

  it('StaleBadge_RecentlyUpdatedApplication_ShowsNoOverdueFlag', async () => {
    showList([listingSummaryWithApplication({}, { isStale: false })])

    const row = await screen.findByRole('listitem')
    expect(within(row).queryByText('Follow-up overdue')).not.toBeInTheDocument()
  })
})

describe('Job Listing RAL Range in the list', () => {
  // The flag stays on the row for triage; the Resolve retry itself is on the
  // Job Listing's own page (issue #94).
  it('JobListingsListPage_UnresolvedRAL_FlagsTheRowAsNeedingAttention', async () => {
    showList([listingSummaryWithApplication({ ral: { source: 'unresolved' } })])

    const row = await screen.findByRole('listitem')
    expect(within(row).getByText('Needs attention')).toBeInTheDocument()
    expect(within(row).queryByRole('button', { name: 'Resolve' })).not.toBeInTheDocument()
  })

  it('JobListingsListPage_StatedRAL_ShowsTheRangeWithItsSourceLabelled', async () => {
    showList([
      listingSummaryWithApplication({ ral: { min: 45000, max: 55000, currency: 'EUR', source: 'stated' } }),
    ])

    expect(await screen.findByText('RAL Range: EUR 45,000 – 55,000')).toBeInTheDocument()
    expect(screen.getByText('Stated in the Job Description')).toBeInTheDocument()
  })
})
