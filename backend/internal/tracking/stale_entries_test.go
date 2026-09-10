package tracking_test

import (
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestStaleEntries(t *testing.T) {
	createdAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	before := createdAt.Add(-24 * time.Hour)
	after := createdAt.Add(24 * time.Hour)

	lookup := func(times map[string]time.Time) func(string) (time.Time, bool) {
		return func(id string) (time.Time, bool) {
			t, ok := times[id]
			return t, ok
		}
	}

	cases := []struct {
		name      string
		entryIDs  []string
		lookupFn  func(string) (time.Time, bool)
		wantStale []string
	}{
		{
			name:      "no Entries changed since Generation: not stale",
			entryIDs:  []string{"experience/a", "experience/b"},
			lookupFn:  lookup(map[string]time.Time{"experience/a": before, "experience/b": before}),
			wantStale: nil,
		},
		{
			name:      "one Entry changed after Generation: stale, named",
			entryIDs:  []string{"experience/a", "experience/b"},
			lookupFn:  lookup(map[string]time.Time{"experience/a": before, "experience/b": after}),
			wantStale: []string{"experience/b"},
		},
		{
			name:      "an Entry changed before CreatedAt: not stale",
			entryIDs:  []string{"experience/a"},
			lookupFn:  lookup(map[string]time.Time{"experience/a": before}),
			wantStale: nil,
		},
		{
			name:      "Generation with no stored Entry ids: skipped, not stale-by-default",
			entryIDs:  nil,
			lookupFn:  lookup(map[string]time.Time{}),
			wantStale: nil,
		},
		{
			name:      "lookup can't find history for an Entry: not checkable, not stale",
			entryIDs:  []string{"experience/a"},
			lookupFn:  lookup(map[string]time.Time{}),
			wantStale: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tracking.StaleEntries(c.entryIDs, createdAt, c.lookupFn)
			if !equalStrings(got, c.wantStale) {
				t.Errorf("StaleEntries(%v, %v): expected %v, got %v", c.entryIDs, createdAt, c.wantStale, got)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
