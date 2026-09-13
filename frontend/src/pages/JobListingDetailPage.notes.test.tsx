import { describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import JobListingDetailPage from './JobListingDetailPage'
import type { JobListingWithApplication, Note } from '@/api/types'
import { application, listingWithApplication } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { requestsTo, server } from '@/test/server'

// Notes (issue #96) are the user's own timestamped log on an Application,
// shown on the Job Listing page next to its Application Method and Contact.
// Every write answers with the whole Application, which replaces the page's.

const NOTES_PATH = '/api/applications/acme/notes'

function note(overrides: Partial<Note> = {}): Note {
  return { id: 'n1', createdAt: '2026-03-02T09:30:00Z', body: 'Screening call booked.', ...overrides }
}

function open(record: JobListingWithApplication) {
  server.use(http.get('/api/job-listings/acme', () => HttpResponse.json(record)))
  return renderPage(<JobListingDetailPage />, { at: '/jobs/acme', pattern: '/jobs/:id' })
}

async function notesSection(): Promise<HTMLElement> {
  const heading = await screen.findByRole('heading', { name: 'Notes' })
  return heading.closest('section') as HTMLElement
}

function shownNotes(section: HTMLElement): HTMLElement[] {
  return within(within(section).getByRole('list', { name: 'Notes' })).queryAllByRole('listitem')
}

describe('Notes on the Job Listing page', () => {
  it('Notes_NoneYet_ShowsAPlainEmptyStateAndTheAddAction', async () => {
    open(listingWithApplication())

    const section = await notesSection()
    expect(within(section).getByText('No Notes yet.')).toBeInTheDocument()
    expect(within(section).getByRole('textbox', { name: 'New Note' })).toBeInTheDocument()
    expect(within(section).getByRole('button', { name: 'Add Note' })).toBeDisabled()
  })

  it('Notes_Several_ShowNewestFirstWithTheirDates', async () => {
    const older = note({ id: 'n1', createdAt: '2026-03-02T09:30:00Z', body: 'Screening call booked.' })
    const newer = note({ id: 'n2', createdAt: '2026-03-09T16:00:00Z', body: 'Take-home due Friday.' })
    open(listingWithApplication({}, { notes: [older, newer] }))

    const items = shownNotes(await notesSection())
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('Take-home due Friday.')
    expect(items[1]).toHaveTextContent('Screening call booked.')
    expect(within(items[0]).getByText(new Date(newer.createdAt).toLocaleString())).toBeInTheDocument()
    expect(within(items[1]).getByText(new Date(older.createdAt).toLocaleString())).toBeInTheDocument()
  })

  it('Note_WrittenInMarkdown_RendersAsFormattedText', async () => {
    const body = 'Budget tops out at **€48k**.\n\n- Platform lead\n- Staff engineer'
    open(listingWithApplication({}, { notes: [note({ body })] }))

    const [item] = shownNotes(await notesSection())
    expect(within(item).getByText('€48k').tagName).toBe('STRONG')
    expect(within(item).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Platform lead',
      'Staff engineer',
    ])
  })

  it('AddNote_Typed_SendsTheBodyAndShowsTheNoteFromTheResponse', async () => {
    const { user } = open(listingWithApplication())
    server.use(
      http.post(NOTES_PATH, () =>
        HttpResponse.json(
          application({ notes: [note({ id: 'n9', createdAt: '2026-03-10T10:00:00Z', body: 'Hybrid, **three** days.' })] }),
          { status: 201 },
        ),
      ),
    )

    const section = await notesSection()
    await user.type(within(section).getByRole('textbox', { name: 'New Note' }), 'Hybrid, **three** days.')
    await user.click(within(section).getByRole('button', { name: 'Add Note' }))

    expect(await within(section).findByText('three')).toBeInTheDocument()
    expect(within(section).getByRole('textbox', { name: 'New Note' })).toHaveValue('')
    expect(within(section).queryByText('No Notes yet.')).not.toBeInTheDocument()
    expect(await requestsTo(NOTES_PATH)).toEqual([
      { method: 'POST', path: NOTES_PATH, search: '', body: { body: 'Hybrid, **three** days.' } },
    ])
  })

  it('AddNote_WhitespaceOnly_CannotBeSubmitted', async () => {
    const { user } = open(listingWithApplication())

    const section = await notesSection()
    await user.type(within(section).getByRole('textbox', { name: 'New Note' }), '   ')

    expect(within(section).getByRole('button', { name: 'Add Note' })).toBeDisabled()
    expect(await requestsTo(NOTES_PATH)).toHaveLength(0)
  })

  it('AddNote_SaveFails_SaysSoAndKeepsWhatWasTyped', async () => {
    const { user } = open(listingWithApplication())
    server.use(http.post(NOTES_PATH, () => new HttpResponse('application not found', { status: 404 })))

    const section = await notesSection()
    await user.type(within(section).getByRole('textbox', { name: 'New Note' }), 'Call went well.')
    await user.click(within(section).getByRole('button', { name: 'Add Note' }))

    expect(await within(section).findByRole('alert')).toHaveTextContent('application not found')
    expect(within(section).getByRole('textbox', { name: 'New Note' })).toHaveValue('Call went well.')
    expect(within(section).getByText('No Notes yet.')).toBeInTheDocument()
  })
})
