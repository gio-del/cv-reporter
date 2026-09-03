import { useState } from 'react'
import { Link } from 'react-router-dom'
import { listAtsListings } from '@/api/client'
import type { AtsListing, AtsProvider } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const providerLabel: Record<AtsProvider, string> = {
  greenhouse: 'Greenhouse',
  lever: 'Lever',
  ashby: 'Ashby',
}

export default function AtsBrowsePage() {
  const [provider, setProvider] = useState<AtsProvider>('greenhouse')
  const [boardSlug, setBoardSlug] = useState('')
  const [listings, setListings] = useState<AtsListing[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

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

      {listings && listings.length === 0 && <p className="mt-4">No open roles found on this board.</p>}

      {listings && listings.length > 0 && (
        <ul className="mt-6 flex flex-col gap-3">
          {listings.map((listing) => (
            <li key={listing.url} className="rounded-xl border border-border bg-card px-4 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <strong className="font-semibold">{listing.title}</strong>
              </div>
              <p className="mb-0 text-sm text-muted-foreground">
                {listing.location}
                {' · '}
                <a href={listing.url} target="_blank" rel="noreferrer">
                  View posting
                </a>
              </p>
            </li>
          ))}
        </ul>
      )}
    </>
  )
}
