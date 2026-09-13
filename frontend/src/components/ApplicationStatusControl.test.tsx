import { describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'
import type { UserEvent } from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import type { ApplicationStatus } from '@/api/types'
import JobListingDetailPage from '@/pages/JobListingDetailPage'
import JobListingsListPage from '@/pages/JobListingsListPage'
import { application, listingWithApplication } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { requestsTo, server } from '@/test/server'

// ApplicationStatusControl is the one Status control both the Job Listings
// list (fast triage) and the Job Listing detail page render (issue #94). A
// move must offer the same next Statuses and ask the same confirmation on
// whichever page it is made from, so every case below runs against both
// real pages rather than the control in isolation.

const STATUS_PATH = '/api/applications/acme/status'

interface StatusPage {
  name: string
  /** show serves the one Job Listing, its Application at the given Status. */
  show: (status: ApplicationStatus) => void
  render: () => ReturnType<typeof renderPage>
  loaded: () => Promise<unknown>
  /** shownStatus is the Status the page displays for that Application. */
  shownStatus: (label: string) => HTMLElement
}

const pages: StatusPage[] = [
  {
    name: 'the Job Listings list',
    show: (status) =>
      server.use(
        http.get('/api/job-listings', () =>
          HttpResponse.json([listingWithApplication({ id: 'acme', company: 'Acme' }, { status })]),
        ),
      ),
    render: () => renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' }),
    loaded: () => screen.findByText('Acme'),
    shownStatus: (label) => within(screen.getByRole('listitem')).getByText(label),
  },
  {
    name: 'the Job Listing detail page',
    show: (status) =>
      server.use(
        http.get('/api/job-listings/acme', () =>
          HttpResponse.json(listingWithApplication({ id: 'acme', company: 'Acme' }, { status })),
        ),
      ),
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

  it('StatusChange_UpdateFails_SurfacesTheErrorAndLeavesTheShownStatusUnchanged', async () => {
    page.show('saved')
    server.use(http.patch(STATUS_PATH, () => new HttpResponse('invalid transition', { status: 409 })))
    const { user } = page.render()

    await moveTo(user, 'Tailoring')

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid transition')
    expect(page.shownStatus('Saved')).toBeInTheDocument()
  })
})
