package tracking_test

import (
	"testing"
	"time"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func countFor(t *testing.T, stats tracking.Stats, status tracking.Status) int {
	t.Helper()
	for _, c := range stats.Counts {
		if c.Status == status {
			return c.Count
		}
	}
	t.Fatalf("no count entry found for status %q", status)
	return 0
}

func conversionFor(stats tracking.Stats, from, to tracking.Status) (tracking.ConversionRate, bool) {
	for _, c := range stats.Conversions {
		if c.From == from && c.To == to {
			return c, true
		}
	}
	return tracking.ConversionRate{}, false
}

func stageTimeFor(stats tracking.Stats, status tracking.Status) (tracking.StageTime, bool) {
	for _, s := range stats.TimeInStage {
		if s.Status == status {
			return s, true
		}
	}
	return tracking.StageTime{}, false
}

func TestComputeStats_ZeroApplications_NoCrashNoNaN(t *testing.T) {
	stats := tracking.ComputeStats(nil)

	if stats.Total != 0 {
		t.Errorf("expected total 0, got %d", stats.Total)
	}
	for _, status := range []tracking.Status{tracking.StatusSaved, tracking.StatusTailoring, tracking.StatusSent, tracking.StatusInterviewing, tracking.StatusRejected, tracking.StatusOffer} {
		if got := countFor(t, stats, status); got != 0 {
			t.Errorf("expected count 0 for %q, got %d", status, got)
		}
	}
	if len(stats.Conversions) != 0 {
		t.Errorf("expected no conversions with zero denominators, got %v", stats.Conversions)
	}
	if len(stats.TimeInStage) != 0 {
		t.Errorf("expected no time-in-stage entries with no history, got %v", stats.TimeInStage)
	}
}

func TestComputeStats_SingleApplicationNoHistory_CountsWorkTimeInStageOmitted(t *testing.T) {
	apps := []tracking.Application{
		{ID: "a1", Status: tracking.StatusSent},
	}
	stats := tracking.ComputeStats(apps)

	if stats.Total != 1 {
		t.Errorf("expected total 1, got %d", stats.Total)
	}
	if got := countFor(t, stats, tracking.StatusSent); got != 1 {
		t.Errorf("expected 1 in Sent, got %d", got)
	}
	if got := countFor(t, stats, tracking.StatusSaved); got != 0 {
		t.Errorf("expected 0 in Saved, got %d", got)
	}

	if conv, ok := conversionFor(stats, tracking.StatusSent, tracking.StatusInterviewing); !ok || conv.Rate != 0.0 {
		t.Errorf("expected Sent->Interviewing conversion rate 0, got %v (ok=%v)", conv, ok)
	}
	if _, ok := conversionFor(stats, tracking.StatusInterviewing, tracking.StatusOffer); ok {
		t.Errorf("expected Interviewing->Offer conversion to be omitted (zero denominator), but it was present")
	}

	if len(stats.TimeInStage) != 0 {
		t.Errorf("expected time-in-stage omitted for an application with no StatusHistory, got %v", stats.TimeInStage)
	}
}

func TestComputeStats_ReopenSequence_CountedOnceInLiveStatus(t *testing.T) {
	now := time.Now().UTC()
	apps := []tracking.Application{
		{
			ID:     "a1",
			Status: tracking.StatusInterviewing,
			StatusHistory: []tracking.StatusChange{
				{Status: tracking.StatusSaved, ChangedAt: now},
				{Status: tracking.StatusTailoring, ChangedAt: now.Add(24 * time.Hour)},
				{Status: tracking.StatusSent, ChangedAt: now.Add(48 * time.Hour)},
				{Status: tracking.StatusRejected, ChangedAt: now.Add(72 * time.Hour)},
				{Status: tracking.StatusInterviewing, ChangedAt: now.Add(96 * time.Hour)},
			},
		},
	}
	stats := tracking.ComputeStats(apps)

	if got := countFor(t, stats, tracking.StatusInterviewing); got != 1 {
		t.Errorf("expected 1 in Interviewing (its live status), got %d", got)
	}
	if got := countFor(t, stats, tracking.StatusRejected); got != 0 {
		t.Errorf("expected 0 in Rejected (reopened past it, no phantom double-count), got %d", got)
	}

	if st, ok := stageTimeFor(stats, tracking.StatusRejected); !ok || st.SampleSize != 1 || st.AverageDays != 1 {
		t.Errorf("expected 1-day Rejected->Interviewing sample from the reopen, got %v (ok=%v)", st, ok)
	}
}

func TestComputeStats_OfferAndRejected_CountedDistinctly(t *testing.T) {
	apps := []tracking.Application{
		{ID: "a1", Status: tracking.StatusOffer},
		{ID: "a2", Status: tracking.StatusRejected},
	}
	stats := tracking.ComputeStats(apps)

	if got := countFor(t, stats, tracking.StatusOffer); got != 1 {
		t.Errorf("expected 1 Offer, got %d", got)
	}
	if got := countFor(t, stats, tracking.StatusRejected); got != 1 {
		t.Errorf("expected 1 Rejected, got %d", got)
	}
}
