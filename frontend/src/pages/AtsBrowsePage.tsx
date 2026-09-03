import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { addTrackedBoard, listAtsListings, listTrackedBoards, removeTrackedBoard, saveJobListing } from '@/api/client'
import type { AtsListing, AtsProvider, SaveJobListingResult, TrackedBoard } from '@/api/types'
import { Badge } from '@/components/ui/badge'
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
  const [trackedBoards, setTrackedBoards] = useState<TrackedBoard[]>([])
  const [trackingBusy, setTrackingBusy] = useState(false)

  useEffect(() => {
    listTrackedBoards()
      .then(setTrackedBoards)
      .catch(() => {})
  }, [])

  const currentTracked = trackedBoards.find((b) => b.provider === provider && b.slug === boardSlug.trim())

  async function fetchListings(p: AtsProvider, slug: string) {
    setError(null)
    setListings(null)
    setLoading(true)
    try {
      const result = await listAtsListings(p, slug)
      setListings(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    await fetchListings(provider, boardSlug.trim())
  }

  function handleTrackedBoardClick(board: TrackedBoard) {
    setProvider(board.provider)
    setBoardSlug(board.slug)
    fetchListings(board.provider, board.slug)
  }

  async function handleToggleTracked() {
    const slug = boardSlug.trim()
    if (!slug) return
    setTrackingBusy(true)
    try {
      if (currentTracked) {
        await removeTrackedBoard(currentTracked.id)
        setTrackedBoards((prev) => prev.filter((b) => b.id !== currentTracked.id))
      } else {
        const board = await addTrackedBoard({ provider, slug, label: titleCase(slug) })
        setTrackedBoards((prev) => [...prev.filter((b) => b.id !== board.id), board])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setTrackingBusy(false)
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

      {trackedBoards.length > 0 && (
        <div className="mb-4 flex flex-wrap items-center gap-2">
          <span className="text-sm text-muted-foreground">Tracked:</span>
          {trackedBoards.map((board) => (
            <Button key={board.id} type="button" size="sm" variant="outline" onClick={() => handleTrackedBoardClick(board)}>
              {board.label || board.slug} <span className="text-muted-foreground">({providerLabel[board.provider]})</span>
            </Button>
          ))}
        </div>
      )}

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
        <Button
          type="button"
          variant="outline"
          disabled={trackingBusy || !boardSlug.trim()}
          onClick={handleToggleTracked}
        >
          {currentTracked ? 'Untrack this board' : 'Track this board'}
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
                    <Badge variant="secondary">
                      <Link to="/jobs" className="no-underline hover:underline">
                        Saved ✓ — view in Job Listings
                      </Link>
                    </Badge>
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
