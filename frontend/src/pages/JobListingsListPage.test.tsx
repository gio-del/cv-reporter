import { describe, expect, it } from 'vitest'
import { screen, waitFor, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import JobListingsListPage from './JobListingsListPage'
import type { JobListingWithApplication } from '@/api/types'
import { listingWithApplication } from '@/test/fixtures'
import { browserBack, currentPath, currentSearch, renderApp, renderPage } from '@/test/render'
import { recordedRequests, requestsTo, server } from '@/test/server'

function showList(listings: JobListingWithApplication[]) {
  server.use(http.get('/api/job-listings', () => HttpResponse.json(listings)))
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

// The list is for triage across many Job Listings: each row identifies one,
// shows its state at a glance, offers the Status move and the Generate
// shortcut, and links to the Job Listing's own page for everything else
// (issue #94). Deletion, the Job Description, Method/Contact editing, the
// Resolve retry, the freshness check and Generation history are covered on
// that page, in JobListingDetailPage.test.tsx.
describe('Job Listing rows', () => {
  it('Row_Shown_KeepsOnlyTheTriageAffordances', async () => {
    showList([
      listingWithApplication(
        {
          id: 'acme',
          company: 'Acme',
          title: 'Backend Engineer',
          url: 'https://acme.example/jobs/1',
          jobDescription: 'We are hiring a Backend Engineer.',
          ral: { min: 45000, max: 55000, currency: 'EUR', source: 'stated' },
        },
        {
          status: 'sent',
          generations: [{ slug: 'acme-1', createdAt: '2026-01-10T10:00:00Z', cvPath: 'output/acme-1/cv.pdf' }],
        },
      ),
    ])
    renderPage(<JobListingsListPage />, { at: '/jobs', pattern: '/jobs' })

    const row = await screen.findByRole('listitem')
    expect(within(row).getByRole('link', { name: 'Backend Engineer — Acme' })).toHaveAttribute('href', '/jobs/acme')
    expect(within(row).getByText('Sent')).toBeInTheDocument()
    expect(within(row).getByText('Not yet checked')).toBeInTheDocument()
    expect(within(row).getByText('RAL Range: EUR 45,000 – 55,000')).toBeInTheDocument()
    expect(within(row).getByText(/^Saved /)).toBeInTheDocument()
    expect(within(row).getByRole('combobox', { name: 'Move Acme to a new status' })).toBeInTheDocument()
    expect(within(row).getByRole('link', { name: 'Regenerate CV' })).toHaveAttribute('href', '/jobs/acme/generate')

    for (const detailOnly of ['Delete', 'View description', 'Check freshness', 'Correct', 'Resolve']) {
      expect(within(row).queryByRole('button', { name: detailOnly })).not.toBeInTheDocument()
    }
    expect(within(row).queryByText('We are hiring a Backend Engineer.')).not.toBeInTheDocument()
    expect(within(row).queryByRole('link', { name: 'View posting' })).not.toBeInTheDocument()
    expect(within(row).queryByRole('link', { name: 'CV' })).not.toBeInTheDocument()
  })
})

describe('Opening a Job Listing from the list', () => {
  function serveListAndDetail() {
    const record = listingWithApplication({ id: 'acme', company: 'Acme', title: 'Backend Engineer' })
    server.use(
      http.get('/api/job-listings', () => HttpResponse.json([record])),
      http.get('/api/job-listings/acme', () => HttpResponse.json(record)),
    )
  }

  const FILTERED = '?status=saved&company=acme&ralSort=desc'

  it('RowLink_Clicked_OpensTheJobListingsOwnPage', async () => {
    serveListAndDetail()
    const { user } = renderApp({ at: '/jobs' })

    await user.click(await screen.findByRole('link', { name: 'Backend Engineer — Acme' }))

    expect(await screen.findByRole('heading', { level: 1, name: 'Backend Engineer — Acme' })).toBeInTheDocument()
    expect(currentPath()).toBe('/jobs/acme')
    expect(await requestsTo('/api/job-listings/acme')).toHaveLength(1)
  })

  it('BackLink_AfterOpeningFromAFilteredList_ReturnsToTheListAsItWasFiltered', async () => {
    serveListAndDetail()
    const { user } = renderApp({ at: `/jobs${FILTERED}` })

    await user.click(await screen.findByRole('link', { name: 'Backend Engineer — Acme' }))
    const back = await screen.findByRole('link', { name: '← Back to Job Listings' })
    expect(back).toHaveAttribute('href', `/jobs${FILTERED}`)

    await user.click(back)

    expect(await screen.findByRole('link', { name: 'Backend Engineer — Acme' })).toBeInTheDocument()
    expect(currentPath()).toBe('/jobs')
    expect(currentSearch()).toBe(FILTERED)
    const listRequests = await requestsTo('/api/job-listings')
    expect(listRequests[listRequests.length - 1].search).toBe('?status=saved&company=acme&sort=ral&order=desc')
  })

  it('BrowserBack_AfterOpeningFromAFilteredList_ReturnsToTheListAsItWasFiltered', async () => {
    serveListAndDetail()
    const { user } = renderApp({ at: `/jobs${FILTERED}` })

    await user.click(await screen.findByRole('link', { name: 'Backend Engineer — Acme' }))
    await screen.findByRole('heading', { level: 1, name: 'Backend Engineer — Acme' })

    browserBack()

    await waitFor(() => expect(currentPath()).toBe('/jobs'))
    expect(currentSearch()).toBe(FILTERED)
    expect(await screen.findByRole('link', { name: 'Backend Engineer — Acme' })).toBeInTheDocument()
  })
})
