import type {
  AddTrackedBoardRequest,
  Application,
  ApplicationGroups,
  ApplicationMethod,
  ApplicationStats,
  ApplicationStatus,
  ArchivedView,
  AtsListing,
  AtsProvider,
  Contact,
  Entry,
  EntryInput,
  GenerateRequest,
  GenerateResult,
  GenerationUsage,
  JobListing,
  JobListingSummaryWithApplication,
  JobListingWithApplication,
  Profile,
  RALListQuery,
  RecordGenerationRequest,
  RenderRequest,
  RenderResult,
  SaveJobListingRequest,
  SaveJobListingResult,
  Snippet,
  SnippetInput,
  TagLintReport,
  TrackedBoard,
} from './types'

// ApiError is what request() throws for a non-2xx response: the message is
// still the backend's body (what every page shows today), and status lets a
// page tell "this record does not exist" apart from any other failure — the
// Job Listing detail page's not-found state (issue #94).
export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new ApiError(body || `Request to ${path} failed (${res.status})`, res.status)
  }
  return res.json()
}

export function listEntries(): Promise<Entry[]> {
  return request('/api/master-data/entries')
}

export function getEntry(id: string): Promise<Entry> {
  return request(`/api/master-data/entries/${id}`)
}

export function updateEntry(id: string, input: EntryInput): Promise<Entry> {
  return request(`/api/master-data/entries/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export function createEntry(input: EntryInput): Promise<Entry> {
  return request('/api/master-data/entries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export async function deleteEntry(id: string): Promise<void> {
  const res = await fetch(`/api/master-data/entries/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body || `Delete failed (${res.status})`)
  }
}

export function getTagLint(): Promise<TagLintReport> {
  return request('/api/master-data/tag-lint')
}

export function getProfile(): Promise<Profile> {
  return request('/api/master-data/profile')
}

export function updateProfile(profile: Profile): Promise<Profile> {
  return request('/api/master-data/profile', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(profile),
  })
}

export function listSnippets(): Promise<Snippet[]> {
  return request('/api/master-data/cover-letter-snippets')
}

export function getSnippet(id: string): Promise<Snippet> {
  return request(`/api/master-data/cover-letter-snippets/${id}`)
}

export function createSnippet(input: SnippetInput): Promise<Snippet> {
  return request('/api/master-data/cover-letter-snippets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export function updateSnippet(id: string, input: SnippetInput): Promise<Snippet> {
  return request(`/api/master-data/cover-letter-snippets/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export async function deleteSnippet(id: string): Promise<void> {
  const res = await fetch(`/api/master-data/cover-letter-snippets/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body || `Delete failed (${res.status})`)
  }
}

export function createGeneration(req: GenerateRequest): Promise<GenerateResult> {
  return request('/api/generations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export function previewGeneration(req: GenerateRequest): Promise<GenerateResult> {
  return request('/api/generations/preview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export function renderGeneration(req: RenderRequest): Promise<RenderResult> {
  return request('/api/generations/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export function generationFileUrl(slug: string, file: string): string {
  return `/api/generations/${encodeURIComponent(slug)}/${encodeURIComponent(file)}`
}

export interface JobListingsFilter {
  status?: ApplicationStatus
  company?: string
  savedFrom?: string
  savedTo?: string
  archived?: ArchivedView
}

// listJobListings passes filter's status/company/savedFrom/savedTo (issue
// #45), archived (issue #98) and sortByRAL/ralMin/ralMax/ralCurrency (issue #51) through to GET
// /api/job-listings's matching optional query params — omitted entirely
// when not given, matching the endpoint's own unfiltered/unsorted default.
// Each row's Job Listing is a summary without the Job Description text
// (issue #97); getJobListing is where that text comes from.
export function listJobListings(
  filter?: JobListingsFilter & RALListQuery,
): Promise<JobListingSummaryWithApplication[]> {
  const params = new URLSearchParams()
  if (filter?.status) params.set('status', filter.status)
  if (filter?.company) params.set('company', filter.company)
  if (filter?.savedFrom) params.set('savedFrom', filter.savedFrom)
  if (filter?.savedTo) params.set('savedTo', filter.savedTo)
  // exclude is the backend's own default, so it is left off the request.
  if (filter?.archived && filter.archived !== 'exclude') params.set('archived', filter.archived)
  if (filter?.sortByRAL) {
    params.set('sort', 'ral')
    params.set('order', filter.sortByRAL)
  }
  if (filter?.ralMin != null) params.set('ral_min', String(filter.ralMin))
  if (filter?.ralMax != null) params.set('ral_max', String(filter.ralMax))
  if (filter?.ralCurrency) params.set('ral_currency', filter.ralCurrency)
  const qs = params.toString()
  return request(`/api/job-listings${qs ? `?${qs}` : ''}`)
}

export function exportDataUrl(): string {
  return '/api/export'
}

// listApplications reads the Applications view (issue #95): every tracked
// Application grouped under its Status, server-ordered. archived mirrors
// listJobListings's, and exclude, the backend's default, is left off.
export function listApplications(archived?: ArchivedView): Promise<ApplicationGroups> {
  const qs = archived && archived !== 'exclude' ? `?archived=${archived}` : ''
  return request(`/api/applications${qs}`)
}

export function getApplicationsStats(): Promise<ApplicationStats> {
  return request('/api/applications/stats')
}

// getJobListing returns a Job Listing paired with its Application — the
// shape listJobListings returns per row, stale-Entry information included
// (issue #94), but with the whole Job Listing, Job Description text
// included (issue #97), so the Job Listing detail page needs one request.
export function getJobListing(id: string): Promise<JobListingWithApplication> {
  return request(`/api/job-listings/${encodeURIComponent(id)}`)
}

export function jobListingLogoUrl(id: string): string {
  return `/api/job-listings/${encodeURIComponent(id)}/logo`
}

export function saveJobListing(req: SaveJobListingRequest): Promise<SaveJobListingResult> {
  return request('/api/job-listings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export async function deleteJobListing(id: string): Promise<void> {
  const res = await fetch(`/api/job-listings/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body || `Delete failed (${res.status})`)
  }
}

export function suggestContact(jobListingId: string): Promise<Contact> {
  return request(`/api/job-listings/${encodeURIComponent(jobListingId)}/suggest-contact`, { method: 'POST' })
}

export function resolveJobListing(jobListingId: string): Promise<JobListingWithApplication> {
  return request(`/api/job-listings/${encodeURIComponent(jobListingId)}/resolve`, { method: 'POST' })
}

export async function checkJobListingFreshness(jobListingId: string): Promise<JobListing> {
  const result = await request<{ jobListing: JobListing }>(
    `/api/job-listings/${encodeURIComponent(jobListingId)}/check-freshness`,
    { method: 'POST' },
  )
  return result.jobListing
}

// setJobListingArchived archives (true) or unarchives (false) a Job Listing
// (issue #98). Both calls are idempotent and never touch the Application.
export async function setJobListingArchived(jobListingId: string, archived: boolean): Promise<JobListing> {
  const result = await request<{ jobListing: JobListing }>(
    `/api/job-listings/${encodeURIComponent(jobListingId)}/${archived ? 'archive' : 'unarchive'}`,
    { method: 'POST' },
  )
  return result.jobListing
}

export function updateApplicationStatus(id: string, status: ApplicationStatus): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  })
}

export function updateApplicationMethod(id: string, method: ApplicationMethod): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/method`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(method),
  })
}

export function getApplicationMailto(id: string): Promise<{ uri: string }> {
  return request(`/api/applications/${encodeURIComponent(id)}/mailto`)
}

export function updateApplicationContact(id: string, contact: Contact): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/contact`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(contact),
  })
}

export function recordApplicationGeneration(id: string, req: RecordGenerationRequest): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/generations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

// addApplicationNote writes a Note (issue #96) and answers with the whole
// updated Application, as every other Application action does.
export function addApplicationNote(id: string, body: string): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/notes`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body }),
  })
}

// editApplicationNote corrects a Note's body; its createdAt never changes.
export function editApplicationNote(id: string, noteId: string, body: string): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/notes/${encodeURIComponent(noteId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ body }),
  })
}

// deleteApplicationNote hard-deletes one Note, answering with the updated
// Application.
export function deleteApplicationNote(id: string, noteId: string): Promise<Application> {
  return request(`/api/applications/${encodeURIComponent(id)}/notes/${encodeURIComponent(noteId)}`, {
    method: 'DELETE',
  })
}

export function listAtsListings(provider: AtsProvider, boardSlug: string): Promise<AtsListing[]> {
  return request(`/api/ats/${encodeURIComponent(provider)}/${encodeURIComponent(boardSlug)}/listings`)
}

export function listTrackedBoards(): Promise<TrackedBoard[]> {
  return request('/api/ats/tracked-boards')
}

export function addTrackedBoard(req: AddTrackedBoardRequest): Promise<TrackedBoard> {
  return request('/api/ats/tracked-boards', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export async function removeTrackedBoard(id: string): Promise<void> {
  const res = await fetch(`/api/ats/tracked-boards/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body || `Delete failed (${res.status})`)
  }
}

export function getUsageSummary(): Promise<GenerationUsage> {
  return request('/api/usage')
}
