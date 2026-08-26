import { useEffect, useState } from 'react'
import ContactSection from '@/components/ContactSection'
import GenerationFilesReminder from '@/components/GenerationFilesReminder'
import { getApplicationMailto } from '@/api/client'
import type { Application, Contact, JobListing } from '@/api/types'

// Branches the FE's apply guidance by Application Method (story 8): email
// gets Contact management (story 7) and, once a Contact is confirmed, a
// prefilled mailto: draft (story 9) plus the CV/Cover Letter reminder
// (story 10); portal/Easy Apply/other skip contact entirely and go
// straight to the application link with the same reminder.
export default function ApplyGuidance({
  jobListing,
  application,
  onSaveContact,
}: {
  jobListing: JobListing
  application: Application
  onSaveContact: (contact: Contact) => Promise<void>
}) {
  const { method, contact } = application
  const [mailtoUri, setMailtoUri] = useState<string | null>(null)
  const [mailtoError, setMailtoError] = useState<string | null>(null)

  useEffect(() => {
    if (method.kind !== 'email' || !contact) {
      setMailtoUri(null)
      return
    }
    setMailtoError(null)
    getApplicationMailto(jobListing.id)
      .then((r) => setMailtoUri(r.uri))
      .catch((err) => setMailtoError(err instanceof Error ? err.message : String(err)))
  }, [method.kind, contact, jobListing.id])

  if (method.kind === 'email') {
    return (
      <div className="flex flex-col gap-2">
        <ContactSection
          jobListingId={jobListing.id}
          contact={contact}
          inferredEmail={method.value}
          onSave={onSaveContact}
        />
        {contact && (
          <>
            {mailtoUri && (
              <p className="mb-0 text-sm">
                <a href={mailtoUri}>Open a prefilled email draft →</a>
              </p>
            )}
            {mailtoError && (
              <p role="alert" className="mb-0 text-sm font-medium text-destructive">
                {mailtoError}
              </p>
            )}
            <GenerationFilesReminder generations={application.generations} />
          </>
        )}
      </div>
    )
  }

  const link = method.value || jobListing.url

  return (
    <div className="flex flex-col gap-2">
      {link ? (
        <p className="mb-0 text-sm">
          Apply here:{' '}
          <a href={link} target="_blank" rel="noreferrer">
            {link}
          </a>
        </p>
      ) : (
        <p className="mb-0 text-sm text-muted-foreground">No application link detected — check the posting.</p>
      )}
      <GenerationFilesReminder generations={application.generations} />
    </div>
  )
}
