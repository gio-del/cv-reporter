import { setupServer } from 'msw/node'

// The one faked boundary for the whole suite (ADR-0019): HTTP and nothing
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

// VersionHeaders are the conditional-write headers a request carried (issue
// #89), recorded apart from RecordedRequest so the many assertions on
// method/path/body keep their exact shape. Absent headers are undefined.
export interface VersionHeaders {
  method: string
  path: string
  ifMatch?: string
  applicationIfMatch?: string
}

const headersLog: VersionHeaders[] = []

server.events.on('request:start', ({ request }) => {
  const clone = request.clone()
  headersLog.push({
    method: request.method,
    path: new URL(request.url).pathname,
    ifMatch: request.headers.get('If-Match') ?? undefined,
    applicationIfMatch: request.headers.get('Application-If-Match') ?? undefined,
  })
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

/**
 * versionHeadersTo is the If-Match / Application-If-Match each request to
 * path carried, in order — what a page presented as the version it read.
 */
export function versionHeadersTo(path: string): VersionHeaders[] {
  return headersLog.filter((r) => r.path === path)
}

export function resetRecordedRequests(): void {
  pending.length = 0
  headersLog.length = 0
}
