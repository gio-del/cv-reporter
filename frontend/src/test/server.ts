import { setupServer } from 'msw/node'

// The one faked boundary for the whole suite (ADR-0018): HTTP and nothing
// below it. `@/api/client` runs for real against this, so the paths and
// query strings it builds — and its `!res.ok` error path — stay under test.
// No default handlers: every test declares the endpoints its page touches,
// which is what makes `onUnhandledRequest: 'error'` meaningful.
export const server = setupServer()

// RecordedRequest is what a test asserts a page actually sent: the method,
// path and query string that left the app, plus the parsed JSON body.
export interface RecordedRequest {
  method: string
  path: string
  search: string
  body: unknown
}

const pending: Promise<RecordedRequest>[] = []

server.events.on('request:start', ({ request }) => {
  const clone = request.clone()
  pending.push(
    (async () => {
      const url = new URL(request.url)
      const text = await clone.text().catch(() => '')
      let body: unknown
      if (text) {
        try {
          body = JSON.parse(text)
        } catch {
          body = text
        }
      }
      return { method: request.method, path: url.pathname, search: url.search, body }
    })(),
  )
})

/** recordedRequests returns every request the app has made, in order. */
export function recordedRequests(): Promise<RecordedRequest[]> {
  return Promise.all([...pending])
}

/** requestsTo narrows the log to one path, for sequencing assertions. */
export async function requestsTo(path: string): Promise<RecordedRequest[]> {
  return (await recordedRequests()).filter((r) => r.path === path)
}

export function resetRecordedRequests(): void {
  pending.length = 0
}
