import { useState } from 'react'
import { suggestContact } from '@/api/client'
import type { Contact } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

// Manages the Contact for an email-method Application (story 7): entered
// manually, or Claude-suggested via web search — either way the user
// reviews the name/email in the form and must explicitly Save before
// anything is persisted (onSave is the only thing that writes).
export default function ContactSection({
  jobListingId,
  contact,
  inferredEmail,
  onSave,
}: {
  jobListingId: string
  contact: Contact | undefined
  inferredEmail: string | undefined
  onSave: (contact: Contact) => Promise<void>
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<Contact>({ name: contact?.name ?? '', email: contact?.email ?? inferredEmail ?? '' })
  const [saving, setSaving] = useState(false)
  const [suggesting, setSuggesting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function startManualEntry() {
    setDraft({ name: contact?.name ?? '', email: contact?.email ?? inferredEmail ?? '' })
    setError(null)
    setEditing(true)
  }

  async function handleSuggest() {
    setError(null)
    setSuggesting(true)
    try {
      const suggestion = await suggestContact(jobListingId)
      setDraft(suggestion)
      setEditing(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSuggesting(false)
    }
  }

  async function handleSave() {
    setSaving(true)
    setError(null)
    try {
      await onSave(draft)
      setEditing(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  if (!editing) {
    return (
      <div className="flex flex-wrap items-center gap-2 text-sm">
        {contact ? (
          <p className="mb-0">
            Contact: <strong>{contact.name || contact.email}</strong>
            {contact.name && <> ({contact.email})</>}
          </p>
        ) : (
          <p className="mb-0 text-muted-foreground">No Contact confirmed yet for this email application.</p>
        )}
        <Button size="sm" variant="outline" onClick={startManualEntry}>
          {contact ? 'Edit' : 'Enter manually'}
        </Button>
        {!contact && (
          <Button size="sm" variant="outline" onClick={handleSuggest} disabled={suggesting}>
            {suggesting ? 'Asking Claude…' : 'Suggest via Claude'}
          </Button>
        )}
        {error && (
          <p role="alert" className="mb-0 basis-full text-sm font-medium text-destructive">
            {error}
          </p>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Input
        className="h-7 w-48"
        value={draft.name}
        onChange={(e) => setDraft((d) => ({ ...d, name: e.target.value }))}
        placeholder="Contact name"
      />
      <Input
        className="h-7 w-56"
        type="email"
        value={draft.email}
        onChange={(e) => setDraft((d) => ({ ...d, email: e.target.value }))}
        placeholder="Contact email"
      />
      <Button size="sm" onClick={handleSave} disabled={saving}>
        {saving ? 'Saving…' : 'Confirm & Save'}
      </Button>
      <Button size="sm" variant="ghost" onClick={() => setEditing(false)} disabled={saving}>
        Cancel
      </Button>
      {error && (
        <p role="alert" className="mb-0 basis-full text-sm font-medium text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
