import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import JobListingsListPage from './JobListingsListPage'
import { listingWithApplication } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { recordedRequests, server } from '@/test/server'

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
