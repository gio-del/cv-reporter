import type {
  Application,
  ApplicationMethod,
  ApplicationStatus,
  AtsListing,
  AtsProvider,
  Contact,
  Entry,
  EntryInput,
  GenerateRequest,
  GenerateResult,
  JobListing,
  JobListingWithApplication,
  Profile,
  RecordGenerationRequest,
  RenderRequest,
  RenderResult,
  SaveJobListingRequest,
  SaveJobListingResult,
  Snippet,
  SnippetInput,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(body || `Request to ${path} failed (${res.status})`)
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

export function listJobListings(): Promise<JobListingWithApplication[]> {
  return request('/api/job-listings')
}

export function getJobListing(id: string): Promise<JobListing> {
  return request(`/api/job-listings/${encodeURIComponent(id)}`)
}

export function saveJobListing(req: SaveJobListingRequest): Promise<SaveJobListingResult> {
  return request('/api/job-listings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
}

export function suggestContact(jobListingId: string): Promise<Contact> {
  return request(`/api/job-listings/${encodeURIComponent(jobListingId)}/suggest-contact`, { method: 'POST' })
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

export function listAtsListings(provider: AtsProvider, boardSlug: string): Promise<AtsListing[]> {
  return request(`/api/ats/${encodeURIComponent(provider)}/${encodeURIComponent(boardSlug)}/listings`)
}
