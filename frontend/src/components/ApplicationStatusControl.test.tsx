import { describe, expect, it } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import type { UserEvent } from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import type { ApplicationStatus } from '@/api/types'
import JobListingDetailPage from '@/pages/JobListingDetailPage'
import JobListingsListPage from '@/pages/JobListingsListPage'
import {
  APPLICATION_VERSION,
  application,
  listingSummaryWithApplication,
  listingWithApplication,
} from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { requestsTo, server, versionHeadersTo } from '@/test/server'

// ApplicationStatusControl is the one Status control both the Job Listings
// list (fast triage) and the Job Listing detail page render (issue #94). A
// move must offer the same next Statuses and ask the same confirmation on
// whichever page it is made from, so every case below runs against both
// real pages rather than the control in isolation.

const STATUS_PATH = '/api/applications/acme/status'

interface StatusPage {
  name: string
  /** show serves the one Job Listing, its Application at the given Status. */
  show: (status: ApplicationStatus, version?: string) => void
  /** readPath is the GET the page re-reads on "Reload the current version". */
  readPath: string
  render: () => ReturnType<typeof renderPage>
  loaded: () => Promise<unknown>
  /** shownStatus is the Status the page displays for that Application. */
  shownStatus: (label: string) => HTMLElement
}

const pages: StatusPage[] = [
  {
    name: 'the Job Listings list',
    show: (status, version = APPLICATION_VERSION) =>
      server.use(
        http.get('/api/job-listings', () =>
          HttpResponse.json([listingSummaryWithApplication({ id: 'acme', company: 'Acme' }, { status, version })]),
        ),
      ),
    readPath: '/api/job-listings',
    render: () => renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' }),
    loaded: () => screen.findByText('Acme'),
    shownStatus: (label) => within(screen.getByRole('listitem')).getByText(label),
  },
  {
    name: 'the Job Listing detail page',
    show: (status, version = APPLICATION_VERSION) =>
      server.use(
        http.get('/api/job-listings/acme', () =>
          HttpResponse.json(listingWithApplication({ id: 'acme', company: 'Acme' }, { status, version })),
        ),
      ),
    readPath: '/api/job-listings/acme',
    render: () => renderPage(<JobListingDetailPage />, { at: '/jobs/acme', pattern: '/jobs/:id' }),
    loaded: () => screen.findByRole('heading', { level: 1, name: 'Acme' }),
    shownStatus: (label) => screen.getByText(label),
  },
]

/** patchStatusReturning answers the Status PATCH with the updated Application. */
function patchStatusReturning(status: ApplicationStatus) {
  server.use(http.patch(STATUS_PATH, () => HttpResponse.json(application({ status }))))
}

/** moveTo drives the Status dropdown the way a user does. */
async function moveTo(user: UserEvent, status: string) {
  await user.click(await screen.findByRole('combobox', { name: 'Move Acme to a new status' }))
  await user.click(await screen.findByRole('option', { name: status }))
}

describe.each(pages)('Application Status transitions from $name', (page) => {
  it('StatusDropdown_Interviewing_OffersExactlyTheBackendsAllowedMoves', async () => {
    page.show('interviewing')
    const { user } = page.render()

    await user.click(await screen.findByRole('combobox', { name: 'Move Acme to a new status' }))

    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
      'Rejected',
      'Offer',
      'Withdrawn',
    ])
  })

  it('StatusDropdown_Offer_OffersNoStatusControlAtAll', async () => {
    page.show('offer')
    page.render()

    await page.loaded()
    expect(screen.queryByRole('combobox', { name: 'Move Acme to a new status' })).not.toBeInTheDocument()
  })

  it('StatusChange_ToRejected_AsksForConfirmationBeforeSendingAnything', async () => {
    page.show('interviewing')
    patchStatusReturning('rejected')
    const { user } = page.render()

    await moveTo(user, 'Rejected')

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Mark this Application Rejected?')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)

    await user.click(within(dialog).getByRole('button', { name: 'Yes, mark Rejected' }))

    expect(await screen.findByText('Rejected')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'rejected' } },
    ])
  })

  it('StatusChange_ToWithdrawn_AsksForConfirmationBeforeSendingAnything', async () => {
    page.show('sent')
    patchStatusReturning('withdrawn')
    const { user } = page.render()

    await moveTo(user, 'Withdrawn')

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Mark this Application Withdrawn?')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)

    await user.click(within(dialog).getByRole('button', { name: 'Yes, mark Withdrawn' }))

    expect(await screen.findByText('Withdrawn')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'withdrawn' } },
    ])
  })

  // Both directions of each reversible pair are guarded, per CONTEXT.md's
  // Status entry: Reopening is as deliberate an act as ending the process.
  it('StatusChange_ReopenFromRejected_AsksForConfirmationBeforeSendingAnything', async () => {
    page.show('rejected')
    patchStatusReturning('interviewing')
    const { user } = page.render()

    await moveTo(user, 'Interviewing')

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Reopen this Application to Interviewing?')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)

    await user.click(within(dialog).getByRole('button', { name: 'Yes, reopen' }))

    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'interviewing' } },
    ])
  })

  it('StatusChange_ReopenFromWithdrawn_AsksForConfirmationBeforeSendingAnything', async () => {
    page.show('withdrawn')
    patchStatusReturning('interviewing')
    const { user } = page.render()

    await moveTo(user, 'Interviewing')

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Reopen this Application to Interviewing?')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)

    await user.click(within(dialog).getByRole('button', { name: 'Yes, reopen' }))

    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'interviewing' } },
    ])
  })

  it('StatusChange_ConfirmationCancelled_LeavesTheStatusUntouchedAndSendsNothing', async () => {
    page.show('interviewing')
    const { user } = page.render()

    await moveTo(user, 'Rejected')
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(page.shownStatus('Interviewing')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)
  })

  it('StatusChange_NoConfirmationNeeded_AppliesDirectly', async () => {
    page.show('saved')
    patchStatusReturning('tailoring')
    const { user } = page.render()

    await moveTo(user, 'Tailoring')

    expect(await screen.findByText('Tailoring')).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'tailoring' } },
    ])
  })

  // The backend answers an illegal move with 400 (409 is reserved for a
  // version conflict, issue #89, covered below).
  it('StatusChange_UpdateFails_SurfacesTheErrorAndLeavesTheShownStatusUnchanged', async () => {
    page.show('saved')
    server.use(http.patch(STATUS_PATH, () => new HttpResponse('invalid transition', { status: 400 })))
    const { user } = page.render()

    await moveTo(user, 'Tailoring')

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid transition')
    expect(page.shownStatus('Saved')).toBeInTheDocument()
  })

  it('StatusChange_Sent_PresentsTheApplicationsVersionAsIfMatch', async () => {
    page.show('saved')
    patchStatusReturning('tailoring')
    const { user } = page.render()

    await moveTo(user, 'Tailoring')

    await screen.findByText('Tailoring')
    expect(versionHeadersTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, ifMatch: APPLICATION_VERSION, applicationIfMatch: undefined },
    ])
  })

  // A 409 is not a validation error: the page says the Application changed
  // on disk and offers an explicit reload, which re-reads the record so the
  // next move presents the fresh token (issue #89, stories 15 and 20).
  it('StatusChange_ChangedOnDisk_OffersAReloadWhoseTokenTheNextMovePresents', async () => {
    page.show('saved')
    server.use(
      http.patch(STATUS_PATH, () => new HttpResponse('this Application changed on disk', { status: 409 }), {
        once: true,
      }),
    )
    const { user } = page.render()

    await moveTo(user, 'Tailoring')

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('This Application changed on disk since you opened it.')
    expect(page.shownStatus('Saved')).toBeInTheDocument()

    page.show('saved', 'application-v2')
    patchStatusReturning('tailoring')
    await user.click(within(alert).getByRole('button', { name: 'Reload the current version' }))
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument())
    expect(await requestsTo(page.readPath)).toHaveLength(2)

    await moveTo(user, 'Tailoring')

    await screen.findByText('Tailoring')
    expect(versionHeadersTo(STATUS_PATH).map((r) => r.ifMatch)).toEqual([APPLICATION_VERSION, 'application-v2'])
  })
})
