import '@testing-library/jest-dom/vitest'
import { afterAll, afterEach, beforeAll } from 'vitest'
import { cleanup } from '@testing-library/react'
import { resetRecordedRequests, server } from './server'

// jsdom has no fetch of its own, so the global here is Node's, which — unlike
// a browser's — refuses a relative URL. The app fetches '/api/…' (the Vite
// dev server proxies it, see vite.config.ts), so resolve relative inputs the
// way a browser would, against the document's origin. This is the only thing
// standing between the real `@/api/client` and MSW: the path and query string
// the client built are passed through untouched.
const nodeFetch = globalThis.fetch
globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => {
  if (typeof input === 'string' && input.startsWith('/')) {
    return nodeFetch(new URL(input, window.location.origin), init)
  }
  return nodeFetch(input, init)
}) as typeof fetch

// Radix (the shadcn/ui primitives) uses pointer capture, ResizeObserver and
// scrollIntoView, none of which jsdom implements. Stubbing them here is what
// lets a test click a real Select or AlertDialog instead of reaching past it.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
}
Element.prototype.hasPointerCapture ??= () => false
Element.prototype.setPointerCapture ??= () => {}
Element.prototype.releasePointerCapture ??= () => {}
Element.prototype.scrollIntoView ??= () => {}

// onUnhandledRequest: 'error' is deliberate — a page that starts calling a new
// endpoint must say so in the test, so an endpoint change can never pass as a
// green run against a silently empty response.
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))

afterEach(() => {
  cleanup()
  server.resetHandlers()
  resetRecordedRequests()
})

afterAll(() => server.close())
