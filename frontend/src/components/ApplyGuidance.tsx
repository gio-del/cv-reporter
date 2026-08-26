import ContactSection from '@/components/ContactSection'
import GenerationFilesReminder from '@/components/GenerationFilesReminder'
import type { Application, Contact, JobListing } from '@/api/types'

// Branches the FE's apply guidance by Application Method (story 8): email
// gets Contact management (story 7); portal/Easy Apply skip contact
// entirely and go straight to the application link with the CV/Cover
// Letter ready to attach.
export default function ApplyGuidance({
  jobListing,
  application,
  onSaveContact,
}: {
  jobListing: JobListing
  application: Application
  onSaveContact: (contact: Contact) => Promise<void>
}) {
  const { method } = application

  if (method.kind === 'email') {
    return (
      <ContactSection
        jobListingId={jobListing.id}
        contact={application.contact}
        inferredEmail={method.value}
        onSave={onSaveContact}
      />
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
