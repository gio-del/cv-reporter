import { describe, expect, it } from 'vitest'
import { allowedNextStatuses } from './applicationStatus'

// ApplicationStatusControl is the one Status control both the Job Listings
// list and the Job Listing detail page render (issue #94). Its behaviour —
// which moves it offers, which it confirms, what it sends — is covered
// through the pages that render it; this file pins the state machine it
// reads.
describe('Application Status state machine', () => {
  // The frontend's table duplicates the backend's tracking.allowedTransitions
  // by design (the UI needs it to offer a next move at all). Pinning its exact
  // content is what stops the two drifting apart silently: a backend change
  // that is not mirrored here fails with a readable diff.
  it('AllowedNextStatuses_EveryStatus_MatchesTheBackendStateMachine', () => {
    expect(allowedNextStatuses).toEqual({
      saved: ['tailoring', 'withdrawn'],
      tailoring: ['sent', 'withdrawn'],
      sent: ['interviewing', 'rejected', 'withdrawn'],
      interviewing: ['rejected', 'offer', 'withdrawn'],
      rejected: ['interviewing'],
      offer: [],
      withdrawn: ['interviewing'],
    })
  })
})
