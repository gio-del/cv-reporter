import { describe, expect, it } from 'vitest'
import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import EntryDetailPage from './EntryDetailPage'
import type { Entry } from '@/api/types'
import { entry } from '@/test/fixtures'
import { renderPage } from '@/test/render'
import { requestsTo, server, versionHeadersTo } from '@/test/server'

const ENTRY_PATH = '/api/master-data/entries/experience/acme-backend'

function serveEntry(record: Entry) {
  server.use(http.get(ENTRY_PATH, () => HttpResponse.json(record)))
}

function open(record: Entry) {
  serveEntry(record)
  return renderPage(<EntryDetailPage />, { at: '/entries/experience/acme-backend', pattern: '/entries/*' })
}

// The Entry edit form is where a lost update hurts most: Selection runs
// against Master Data, so a Tag silently dropped by a stale save surfaces
// only in a later Tailored CV (issue #89).
describe('Entry edit conflicts', () => {
  it('EntrySave_Sent_PresentsTheVersionItRead', async () => {
    const { user } = open(entry({ id: 'experience/acme-backend', version: 'entry-v1' }))
    server.use(http.put(ENTRY_PATH, () => HttpResponse.json(entry({ id: 'experience/acme-backend', version: 'entry-v2' }))))

    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(versionHeadersTo(ENTRY_PATH).filter((r) => r.method === 'PUT').map((r) => r.ifMatch)).toEqual(['entry-v1'])
  })

  it('EntrySave_ChangedOnDisk_KeepsTheTypedInputUntilReloadThenSavesWithTheFreshToken', async () => {
    const { user } = open(entry({ id: 'experience/acme-backend', role: 'Backend Engineer', version: 'entry-v1' }))
    server.use(http.put(ENTRY_PATH, () => new HttpResponse('this Entry changed on disk', { status: 409 }), { once: true }))

    await user.click(await screen.findByRole('button', { name: 'Edit' }))
    const role = screen.getByLabelText('Role')
    await user.clear(role)
    await user.type(role, 'Staff Engineer')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('This Entry changed on disk since you opened it.')
    expect(alert).toHaveTextContent('Your edits are still here')
    expect(screen.getByLabelText('Role')).toHaveValue('Staff Engineer')

    serveEntry(entry({ id: 'experience/acme-backend', role: 'Platform Engineer', version: 'entry-v2' }))
    server.use(http.put(ENTRY_PATH, () => HttpResponse.json(entry({ id: 'experience/acme-backend', version: 'entry-v3' }))))
    await user.click(within(alert).getByRole('button', { name: 'Reload the current version' }))

    expect(await screen.findByDisplayValue('Platform Engineer')).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('button', { name: 'Edit' })).toBeInTheDocument()
    expect(versionHeadersTo(ENTRY_PATH).filter((r) => r.method === 'PUT').map((r) => r.ifMatch)).toEqual([
      'entry-v1',
      'entry-v2',
    ])
    expect((await requestsTo(ENTRY_PATH)).filter((r) => r.method === 'GET')).toHaveLength(2)
  })
})
