import { useState } from 'react'
import { Link } from 'react-router-dom'
import { listAtsListings, saveJobListing } from '@/api/client'
import type { AtsListing, AtsProvider, SaveJobListingResult } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const providerLabel: Record<AtsProvider, string> = {
  greenhouse: 'Greenhouse',
  lever: 'Lever',
  ashby: 'Ashby',
}

function titleCase(slug: string): string {
  return slug
    .split(/[-_]/)
    .filter(Boolean)
    .map((word) => word[0].toUpperCase() + word.slice(1))
    .join(' ')
}

export default function AtsBrowsePage() {
  const [provider, setProvider] = useState<AtsProvider>('greenhouse')
  const [boardSlug, setBoardSlug] = useState('')
  const [listings, setListings] = useState<AtsListing[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [savingUrl, setSavingUrl] = useState<string | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [savedByUrl, setSavedByUrl] = useState<Record<string, SaveJobListingResult>>({})

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setListings(null)
    setLoading(true)
    try {
      const result = await listAtsListings(provider, boardSlug.trim())
      setListings(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  async function handleSave(listing: AtsListing) {
    setSaveError(null)
    setSavingUrl(listing.url)
    try {
      const result = await saveJobListing({
        company: titleCase(boardSlug.trim()),
        url: listing.url,
        jobDescription: listing.description,
      })
      setSavedByUrl((prev) => ({ ...prev, [listing.url]: result }))
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : String(err))
    } finally {
      setSavingUrl(null)
    }
  }

  return (
    <>
      <p className="mb-4 inline-block text-sm">
        <Link to="/jobs" className="no-underline hover:underline">
          ← Back to Job Listings
        </Link>
      </p>
      <h1>Browse ATS Job Boards</h1>
      <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-3">
        <FieldGroup className="flex-row flex-wrap gap-3">
          <Field>
            <FieldLabel htmlFor="ats-provider">Provider</FieldLabel>
            <Select value={provider} onValueChange={(value) => setProvider(value as AtsProvider)}>
              <SelectTrigger id="ats-provider" className="w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(providerLabel).map(([value, label]) => (
                  <SelectItem key={value} value={value}>
                    {label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field>
            <FieldLabel htmlFor="board-slug">Board slug</FieldLabel>
            <Input
              id="board-slug"
              value={boardSlug}
              onChange={(e) => setBoardSlug(e.target.value)}
              placeholder="e.g. acme"
              required
            />
          </Field>
        </FieldGroup>
        <Button type="submit" disabled={loading || !boardSlug.trim()}>
          {loading ? 'Fetching…' : 'Fetch open roles'}
        </Button>
      </form>

      {error && (
        <p role="alert" className="mt-4 font-medium text-destructive">
          {error}
        </p>
      )}
      {saveError && (
        <p role="alert" className="mt-4 font-medium text-destructive">
          {saveError}
        </p>
      )}

      {listings && listings.length === 0 && <p className="mt-4">No open roles found on this board.</p>}

      {listings && listings.length > 0 && (
        <ul className="mt-6 flex flex-col gap-3">
          {listings.map((listing) => {
            const saved = savedByUrl[listing.url]
            return (
              <li key={listing.url} className="rounded-xl border border-border bg-card px-4 py-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <strong className="font-semibold">{listing.title}</strong>
                  {saved ? (
                    <Link to="/jobs" className="text-sm no-underline hover:underline">
                      Saved ✓ — view in Job Listings
                    </Link>
                  ) : (
                    <Button size="sm" onClick={() => handleSave(listing)} disabled={savingUrl === listing.url}>
                      {savingUrl === listing.url ? 'Saving…' : 'Save as Job Listing'}
                    </Button>
                  )}
                </div>
                <p className="mb-0 text-sm text-muted-foreground">
                  {listing.location}
                  {' · '}
                  <a href={listing.url} target="_blank" rel="noreferrer">
                    View posting
                  </a>
                </p>
              </li>
            )
          })}
        </ul>
      )}
    </>
  )
}
