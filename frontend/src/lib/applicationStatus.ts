import type { ApplicationStatus } from '@/api/types'

// The Status vocabulary and state machine shared by every page that shows or
// moves an Application's Status. Kept out of ApplicationStatusControl.tsx so
// that file exports only a component (Fast Refresh), while the list page's
// Status filter can still read the labels (issue #94).

export const statusLabel: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  tailoring: 'Tailoring',
  sent: 'Sent',
  interviewing: 'Interviewing',
  rejected: 'Rejected',
  offer: 'Offer',
  withdrawn: 'Withdrawn',
}

// Mirrors the backend's Status state machine (see tracking.allowedTransitions)
// so the FE only ever offers a valid next move — the backend remains the
// source of truth and re-validates on PATCH regardless (story 4). Exported
// so applicationStatus.test.ts can pin its exact content transition
// by transition: the duplication is deliberate, so drift from the backend
// has to fail loudly rather than silently (issue #90).
export const allowedNextStatuses: Record<ApplicationStatus, ApplicationStatus[]> = {
  saved: ['tailoring', 'withdrawn'],
  tailoring: ['sent', 'withdrawn'],
  sent: ['interviewing', 'rejected', 'withdrawn'],
  interviewing: ['rejected', 'offer', 'withdrawn'],
  rejected: ['interviewing'],
  offer: [],
  withdrawn: ['interviewing'],
}

// Moving into Rejected/Withdrawn, and Reopening out of either back to
// Interviewing, each reverse the other and need explicit confirmation
// before the PATCH fires (stories 2-4, PRD stories 3 and 5).
export function needsConfirmation(from: ApplicationStatus, to: ApplicationStatus): boolean {
  return (
    to === 'rejected' ||
    to === 'withdrawn' ||
    ((from === 'rejected' || from === 'withdrawn') && to === 'interviewing')
  )
}
