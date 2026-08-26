package tracking_test

import (
	"errors"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

func TestTransition_AllowedMoves(t *testing.T) {
	cases := []struct {
		from tracking.Status
		to   tracking.Status
	}{
		{tracking.StatusSaved, tracking.StatusTailoring},
		{tracking.StatusTailoring, tracking.StatusSent},
		{tracking.StatusSent, tracking.StatusInterviewing},
		{tracking.StatusSent, tracking.StatusRejected},
		{tracking.StatusInterviewing, tracking.StatusRejected},
		{tracking.StatusInterviewing, tracking.StatusOffer},
	}
	for _, c := range cases {
		got, err := tracking.Transition(c.from, c.to)
		if err != nil {
			t.Errorf("Transition(%q, %q): expected no error, got %v", c.from, c.to, err)
		}
		if got != c.to {
			t.Errorf("Transition(%q, %q): expected %q, got %q", c.from, c.to, c.to, got)
		}
	}
}

func TestTransition_RejectedMoves(t *testing.T) {
	cases := []struct {
		from tracking.Status
		to   tracking.Status
	}{
		{tracking.StatusSaved, tracking.StatusSent},
		{tracking.StatusSaved, tracking.StatusInterviewing},
		{tracking.StatusSaved, tracking.StatusOffer},
		{tracking.StatusTailoring, tracking.StatusSaved},
		{tracking.StatusTailoring, tracking.StatusInterviewing},
		{tracking.StatusSent, tracking.StatusOffer},
		{tracking.StatusSent, tracking.StatusTailoring},
		{tracking.StatusInterviewing, tracking.StatusSent},
		{tracking.StatusRejected, tracking.StatusOffer},
		{tracking.StatusOffer, tracking.StatusRejected},
		{tracking.StatusSaved, tracking.StatusSaved},
	}
	for _, c := range cases {
		_, err := tracking.Transition(c.from, c.to)
		if !errors.Is(err, tracking.ErrInvalidTransition) {
			t.Errorf("Transition(%q, %q): expected ErrInvalidTransition, got %v", c.from, c.to, err)
		}
	}
}

func TestTransition_UnknownStatus_RejectsAsValidation(t *testing.T) {
	if _, err := tracking.Transition("bogus", tracking.StatusTailoring); !errors.Is(err, tracking.ErrValidation) {
		t.Errorf("expected ErrValidation for unknown current status, got %v", err)
	}
	if _, err := tracking.Transition(tracking.StatusSaved, "bogus"); !errors.Is(err, tracking.ErrValidation) {
		t.Errorf("expected ErrValidation for unknown target status, got %v", err)
	}
}
