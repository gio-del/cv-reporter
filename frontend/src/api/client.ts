import type {
  Entry,
  EntryInput,
  GenerateRequest,
  GenerateResult,
  Profile,
  RenderRequest,
  RenderResult,
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
