// oxlint-disable react/only-export-components -- test-support module: never
// imported by the app, so Fast Refresh does not apply to it.
import type { ReactElement } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'

// Pages read route params and search params, so they are rendered inside a
// real MemoryRouter at their real route rather than having routing stubbed —
// the URL a filter writes is part of the behaviour under test (ADR-0018).

function LocationProbe() {
  const location = useLocation()
  return <span data-testid="location-search">{location.search}</span>
}

export interface RenderPageOptions {
  /** The route pattern the page is mounted at, e.g. '/jobs/:id/generate'. */
  pattern?: string
  /** The entry URL, e.g. '/jobs/acme/generate?status=sent'. */
  at?: string
}

export function renderPage(element: ReactElement, options: RenderPageOptions = {}) {
  const at = options.at ?? '/'
  const pattern = options.pattern ?? '/'
  // pointerEventsCheck is off because Radix's dialogs/selects put
  // `pointer-events: none` on the body while open, which jsdom reports
  // faithfully and user-event would otherwise refuse to click through.
  const user = userEvent.setup({ pointerEventsCheck: 0 })
  // TooltipProvider mirrors App.tsx: pages render Tooltips, and Radix needs
  // its provider above them exactly as the real app supplies it.
  const result = render(
    <TooltipProvider>
      <MemoryRouter initialEntries={[at]}>
        <LocationProbe />
        <Routes>
          <Route path={pattern} element={element} />
        </Routes>
      </MemoryRouter>
    </TooltipProvider>,
  )
  return { user, ...result }
}

/** currentSearch is the query string the page's own navigation has produced. */
export function currentSearch(): string {
  return screen.getByTestId('location-search').textContent ?? ''
}
