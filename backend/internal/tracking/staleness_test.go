package tracking_test

import (
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestIsStale(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	old := now.Add(-15 * 24 * time.Hour).Format(time.RFC3339Nano)
	recent := now.Add(-1 * 24 * time.Hour).Format(time.RFC3339Nano)

	cases := []struct {
		name            string
		status          tracking.Status
		statusUpdatedAt string
		want            bool
	}{
		{"sent past threshold", tracking.StatusSent, old, true},
		{"interviewing past threshold", tracking.StatusInterviewing, old, true},
		{"sent under threshold", tracking.StatusSent, recent, false},
		{"interviewing under threshold", tracking.StatusInterviewing, recent, false},
		{"saved past threshold ignored", tracking.StatusSaved, old, false},
		{"tailoring past threshold ignored", tracking.StatusTailoring, old, false},
		{"rejected past threshold ignored", tracking.StatusRejected, old, false},
		{"offer past threshold ignored", tracking.StatusOffer, old, false},
		{"empty timestamp", tracking.StatusSent, "", false},
		{"unparseable timestamp", tracking.StatusSent, "not-a-timestamp", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tracking.IsStale(c.status, c.statusUpdatedAt, now, tracking.DefaultStaleThreshold)
			if got != c.want {
				t.Errorf("IsStale(%q, %q, now, threshold) = %v, want %v", c.status, c.statusUpdatedAt, got, c.want)
			}
		})
	}
}

func TestDefaultStaleThreshold_Is14Days(t *testing.T) {
	if tracking.DefaultStaleThreshold != 14*24*time.Hour {
		t.Errorf("expected DefaultStaleThreshold to be 14 days, got %v", tracking.DefaultStaleThreshold)
	}
}
