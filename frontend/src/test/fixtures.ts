import type {
  Application,
  Entry,
  GenerateResult,
  JobListing,
  JobListingWithApplication,
  RenderResult,
} from '@/api/types'

// Response builders mirroring what the Go handlers actually return, so a
// test states only the field it is about. The backend's `seedDataDir` /
// `saveListing` helpers are the model (see backend/internal/api/*_test.go).

/**
 * jobListing is a Job Listing as the backend returns it, with every optional
 * field (title, url, logo, freshnessCheckedAt) left absent unless asked for —
 * the "older record, written before those fields existed" case the backend
 * types document.
 */
export function jobListing(overrides: Partial<JobListing> = {}): JobListing {
  return {
    id: 'acme',
    company: 'Acme',
    source: 'manual',
    savedAt: '2026-01-05T10:00:00Z',
    jobDescription: 'We are hiring a Backend Engineer.',
    ral: { source: 'n/a' },
    freshnessStatus: 'not-yet-checked',
    ...overrides,
  }
}

export function application(overrides: Partial<Application> = {}): Application {
  return {
    id: 'acme',
    jobListingId: 'acme',
    status: 'saved',
    method: { kind: 'portal', value: 'https://acme.example/apply' },
    isStale: false,
    ...overrides,
  }
}

export function listingWithApplication(
  listingOverrides: Partial<JobListing> = {},
  applicationOverrides: Partial<Application> = {},
): JobListingWithApplication {
  const listing = jobListing(listingOverrides)
  return {
    jobListing: listing,
    application: application({ id: listing.id, jobListingId: listing.id, ...applicationOverrides }),
  }
}

export function entry(overrides: Partial<Entry> = {}): Entry {
  return {
    id: 'acme-backend',
    type: 'experience',
    employer: 'Globex',
    role: 'Backend Engineer',
    start: '2021-01',
    end: null,
    tags: ['go'],
    ...overrides,
  }
}

/**
 * generateResult is a Generate call's result: one Entry with two bullets, an
 * Italian-or-English language, usage and no Cover Letter unless asked for.
 */
export function generateResult(overrides: Partial<GenerateResult> = {}): GenerateResult {
  return {
    mode: 'tailored',
    selection: {
      entries: [
        {
          entryId: 'acme-backend',
          reason: 'Closest match to the Job Description.',
          bullets: [
            { sourceIndex: 0, source: 'Built an API.', rewritten: 'Built a Go HTTP API.' },
            { sourceIndex: 1, source: 'Ran deploys.', rewritten: 'Owned deploys end to end.' },
          ],
        },
      ],
    },
    usage: { inputTokens: 1200, outputTokens: 340, estimatedCostUsd: 0.0421 },
    language: 'en',
    ...overrides,
  }
}

export function renderResult(overrides: Partial<RenderResult> = {}): RenderResult {
  return {
    slug: 'acme',
    cvPath: 'output/acme/cv.pdf',
    cvPageCount: 1,
    cvParsability: { status: 'ok' },
    ...overrides,
  }
}
