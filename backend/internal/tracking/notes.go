package tracking

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrNoteNotFound marks an edit or delete addressing a Note id the
// Application does not carry.
var ErrNoteNotFound = errors.New("note not found")

// Note is one timestamped, Markdown observation the user records against an
// Application as its process unfolds (issue #96, CONTEXT.md's Note entry).
// Only ever authored by the user: no Status change, Generation or inference
// writes one. The body may be corrected and the Note deleted outright, but
// CreatedAt is fixed the moment it is written.
type Note struct {
	// ID is server-assigned at creation, derived from the creation instant
	// at nanosecond precision. Opaque to clients, which echo it back — never
	// a list index, which a delete would silently renumber.
	ID string `json:"id" yaml:"id"`
	// CreatedAt (RFC3339Nano, UTC) is when the Note was written. Never
	// changed, including by an edit.
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
	// EditedAt (RFC3339Nano, UTC) is when the body was last corrected,
	// absent until the first edit.
	EditedAt string `json:"editedAt,omitempty" yaml:"editedAt,omitempty"`
	// Body is Markdown, rendered on display.
	Body string `json:"body" yaml:"body"`
}

// AddNote writes a new Note to the Application identified by id and returns
// the updated Application, Notes newest-first (story 1-7). An empty or
// whitespace-only body is rejected with ErrValidation before anything is
// read or written (story 13). Status, StatusUpdatedAt and StatusHistory are
// never touched, so a Note neither advances the pipeline nor resets
// staleness (stories 19-20).
func AddNote(dataDir, id, body string) (Application, error) {
	body, err := validNoteBody(body)
	if err != nil {
		return Application{}, err
	}

	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}

	now := time.Now().UTC()
	note := Note{ID: uniqueNoteID(application.Notes, now), CreatedAt: now.Format(time.RFC3339Nano), Body: body}
	application.Notes = append([]Note{note}, application.Notes...)

	if err := writeApplication(dataDir, application); err != nil {
		return Application{}, err
	}
	return application, nil
}

// EditNote replaces the body of the Note noteID on the Application id and
// returns the updated Application (story 8). CreatedAt never changes, so an
// edit neither reorders the log nor misdates the event (story 9); EditedAt
// records the correction (story 10). Submitting the body the Note already
// has is a no-op that writes nothing and does not mark it edited. An unknown
// Note id is ErrNoteNotFound; an empty body is ErrValidation.
func EditNote(dataDir, id, noteID, body string) (Application, error) {
	body, err := validNoteBody(body)
	if err != nil {
		return Application{}, err
	}

	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}
	i := noteIndex(application.Notes, noteID)
	if i < 0 {
		return Application{}, ErrNoteNotFound
	}
	if application.Notes[i].Body == body {
		return application, nil
	}

	application.Notes[i].Body = body
	application.Notes[i].EditedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeApplication(dataDir, application); err != nil {
		return Application{}, err
	}
	return application, nil
}

func noteIndex(notes []Note, noteID string) int {
	for i, n := range notes {
		if n.ID == noteID {
			return i
		}
	}
	return -1
}

func validNoteBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", fmt.Errorf("%w: note body is required", ErrValidation)
	}
	return trimmed, nil
}

// uniqueNoteID derives a Note id from its creation instant, stepping forward
// a nanosecond in the (practically impossible) case that instant is already
// taken on this Application, so ids stay unique within it.
func uniqueNoteID(existing []Note, at time.Time) string {
	taken := make(map[string]bool, len(existing))
	for _, n := range existing {
		taken[n.ID] = true
	}
	nanos := at.UnixNano()
	for {
		id := "n" + strconv.FormatInt(nanos, 36)
		if !taken[id] {
			return id
		}
		nanos++
	}
}

// sortNotesNewestFirst orders Notes by CreatedAt, newest first, so a
// hand-edited file still reads as a log (story 3). RFC3339Nano timestamps
// in UTC compare correctly once parsed; an unparseable one sorts last.
func sortNotesNewestFirst(notes []Note) {
	parsed := make(map[string]time.Time, len(notes))
	for _, n := range notes {
		if t, err := time.Parse(time.RFC3339Nano, n.CreatedAt); err == nil {
			parsed[n.CreatedAt] = t
		}
	}
	sort.SliceStable(notes, func(i, j int) bool {
		return parsed[notes[i].CreatedAt].After(parsed[notes[j].CreatedAt])
	})
}

func writeApplication(dataDir string, application Application) error {
	return os.WriteFile(filepath.Join(dataDir, applicationsDir, application.ID+".md"), renderApplication(application), 0o644)
}
