package tracking

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrInvalidTransition marks a requested Status change that CanTransition
// rejects — the tracker only moves forward along CONTEXT.md's Status chain
// (Saved → Tailoring → Sent → Interviewing → Rejected/Offer), never
// backward or by skipping a stage, so a manual correction is caught here
// rather than silently corrupting the pipeline view.
var ErrInvalidTransition = errors.New("invalid status transition")

// allowedTransitions is the Status state machine (story 4): each Status
// maps to the Statuses it may move to next. A missing/empty entry means
// terminal (Rejected, Offer). Rejected is reachable from Sent directly (a
// rejection without an interview) as well as from Interviewing.
var allowedTransitions = map[Status][]Status{
	StatusSaved:        {StatusTailoring},
	StatusTailoring:    {StatusSent},
	StatusSent:         {StatusInterviewing, StatusRejected},
	StatusInterviewing: {StatusRejected, StatusOffer},
	StatusRejected:     {},
	StatusOffer:        {},
}

// CanTransition reports whether an Application may move from `from` to
// `to`, per the Status state machine.
func CanTransition(from, to Status) bool {
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// Transition validates a Status change as a pure function, independent of
// storage (per the PRD's Testing Decisions), returning to if the move is
// allowed.
func Transition(current, to Status) (Status, error) {
	if _, ok := allowedTransitions[current]; !ok {
		return "", fmt.Errorf("%w: unknown status %q", ErrValidation, current)
	}
	if _, ok := allowedTransitions[to]; !ok {
		return "", fmt.Errorf("%w: unknown status %q", ErrValidation, to)
	}
	if !CanTransition(current, to) {
		return "", fmt.Errorf("%w: cannot move from %q to %q", ErrInvalidTransition, current, to)
	}
	return to, nil
}

// UpdateApplicationStatus validates and applies a Status change to the
// Application identified by id, writing it back to disk if allowed.
func UpdateApplicationStatus(dataDir, id string, to Status) (Application, error) {
	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}

	newStatus, err := Transition(application.Status, to)
	if err != nil {
		return Application{}, err
	}
	application.Status = newStatus

	if err := os.WriteFile(filepath.Join(dataDir, applicationsDir, id+".md"), renderApplication(application), 0o644); err != nil {
		return Application{}, err
	}
	return application, nil
}
