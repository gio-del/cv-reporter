import { describe, expect, it } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import type { UserEvent } from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import JobListingsListPage, { allowedNextStatuses } from './JobListingsListPage'
import type { ApplicationStatus, JobListingWithApplication } from '@/api/types'
import { application, listingWithApplication } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { recordedRequests, requestsTo, server } from '@/test/server'

const STATUS_PATH = '/api/applications/acme/status'

function listing(status: ApplicationStatus): JobListingWithApplication {
  return listingWithApplication({ id: 'acme', company: 'Acme' }, { status })
}

function showList(listings: JobListingWithApplication[]) {
  server.use(http.get('/api/job-listings', () => HttpResponse.json(listings)))
}

/** patchStatusReturning answers the Status PATCH with the updated Application. */
function patchStatusReturning(status: ApplicationStatus) {
  server.use(http.patch(STATUS_PATH, () => HttpResponse.json(application({ status }))))
}

/** moveTo drives the Status dropdown the way a user does. */
async function moveTo(user: UserEvent, status: string) {
  await user.click(await screen.findByRole('combobox', { name: 'Move Acme to a new status' }))
  await user.click(await screen.findByRole('option', { name: status }))
}

/** shownStatus asserts on the Status the Job Listing's own row displays. */
function shownStatus(label: string): HTMLElement {
  return within(screen.getByRole('listitem')).getByText(label)
}

describe('JobListingsListPage', () => {
  it('JobListingsListPage_NoFilters_ListsEveryJobListingFromAnUnfilteredRequest', async () => {
    server.use(
      http.get('/api/job-listings', () =>
        HttpResponse.json([
          listingWithApplication({ id: 'acme', company: 'Acme', title: 'Backend Engineer' }),
          listingWithApplication({ id: 'globex', company: 'Globex' }),
        ]),
      ),
    )

    renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    expect(await screen.findByText('Backend Engineer — Acme')).toBeInTheDocument()
    expect(screen.getByText('Globex')).toBeInTheDocument()

    const requests = await recordedRequests()
    expect(requests).toHaveLength(1)
    expect(requests[0]).toMatchObject({ method: 'GET', path: '/api/job-listings', search: '' })
  })

  it('JobListingsListPage_ListRequestFails_ShowsTheErrorAndNoListings', async () => {
    server.use(
      http.get('/api/job-listings', () => new HttpResponse('data directory unreadable', { status: 500 })),
    )

    renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    expect(await screen.findByRole('alert')).toHaveTextContent('data directory unreadable')
  })
})

describe('Application Status transitions', () => {
  // The frontend's table duplicates the backend's tracking.allowedTransitions
  // by design (the UI needs it to offer a next move at all). Pinning its exact
  // content is what stops the two drifting apart silently: a backend change
  // that is not mirrored here fails with a readable diff.
  it('AllowedNextStatuses_EveryStatus_MatchesTheBackendStateMachine', () => {
    expect(allowedNextStatuses).toEqual({
      saved: ['tailoring', 'withdrawn'],
      tailoring: ['sent', 'withdrawn'],
      sent: ['interviewing', 'rejected', 'withdrawn'],
      interviewing: ['rejected', 'offer', 'withdrawn'],
      rejected: ['interviewing'],
      offer: [],
      withdrawn: ['interviewing'],
    })
  })

  it('StatusDropdown_Interviewing_OffersExactlyTheBackendsAllowedMoves', async () => {
    showList([listing('interviewing')])
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await user.click(await screen.findByRole('combobox', { name: 'Move Acme to a new status' }))

    expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([
      'Rejected',
      'Offer',
      'Withdrawn',
    ])
  })

  it('StatusDropdown_Offer_OffersNoStatusControlAtAll', async () => {
    showList([listing('offer')])
    renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await screen.findByText('Acme')
    expect(screen.queryByRole('combobox', { name: 'Move Acme to a new status' })).not.toBeInTheDocument()
  })

  it('StatusChange_ToRejected_AsksForConfirmationBeforeSendingAnything', async () => {
    showList([listing('interviewing')])
    patchStatusReturning('rejected')
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

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
    showList([listing('sent')])
    patchStatusReturning('withdrawn')
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

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
    showList([listing('rejected')])
    patchStatusReturning('interviewing')
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

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
    showList([listing('withdrawn')])
    patchStatusReturning('interviewing')
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

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
    showList([listing('interviewing')])
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await moveTo(user, 'Rejected')
    const dialog = await screen.findByRole('alertdialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(shownStatus('Interviewing')).toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toHaveLength(0)
  })

  it('StatusChange_NoConfirmationNeeded_AppliesDirectly', async () => {
    showList([listing('saved')])
    patchStatusReturning('tailoring')
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await moveTo(user, 'Tailoring')

    expect(await screen.findByText('Tailoring')).toBeInTheDocument()
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(await requestsTo(STATUS_PATH)).toEqual([
      { method: 'PATCH', path: STATUS_PATH, search: '', body: { status: 'tailoring' } },
    ])
  })

  it('StatusChange_UpdateFails_SurfacesTheErrorAndLeavesTheShownStatusUnchanged', async () => {
    showList([listing('saved')])
    server.use(http.patch(STATUS_PATH, () => new HttpResponse('invalid transition', { status: 409 })))
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await moveTo(user, 'Tailoring')

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid transition')
    expect(shownStatus('Saved')).toBeInTheDocument()
  })
})

// Deleting a Job Listing takes its Application — Status, Method, Contact and
// the whole Generation history — with it, so it is confirmation-gated and the
// dialog names what is about to go.
describe('Job Listing deletion', () => {
  const DELETE_PATH = '/api/job-listings/acme'

  it('Delete_Requested_AsksForConfirmationNamingTheJobListing', async () => {
    showList([listingWithApplication({ id: 'acme', company: 'Acme', title: 'Backend Engineer' })])
    server.use(http.delete(DELETE_PATH, () => new HttpResponse(null, { status: 204 })))
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await user.click(await screen.findByRole('button', { name: 'Delete' }))

    const dialog = await screen.findByRole('alertdialog')
    expect(within(dialog).getByText('Delete Backend Engineer — Acme?')).toBeInTheDocument()
    expect(await requestsTo(DELETE_PATH)).toHaveLength(0)

    await user.click(within(dialog).getByRole('button', { name: 'Yes, delete' }))

    await waitFor(() => expect(screen.queryByText('Backend Engineer — Acme')).not.toBeInTheDocument())
    expect(await requestsTo(DELETE_PATH)).toHaveLength(1)
  })

  it('Delete_ConfirmationCancelled_DeletesNothingAndSendsNothing', async () => {
    showList([listing('saved')])
    const { user } = renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    await user.click(await screen.findByRole('button', { name: 'Delete' }))
    await user.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(screen.getByText('Acme')).toBeInTheDocument()
    expect(await requestsTo(DELETE_PATH)).toHaveLength(0)
  })
})
