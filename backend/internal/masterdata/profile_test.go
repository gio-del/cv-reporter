package masterdata

import (
	"errors"
	"strings"
	"testing"
)

// TestGetProfile_MissingFileIsActionable pins ADR-0037's user-facing half:
// profile.yaml is untracked on purpose, so a fresh clone legitimately has
// none, and the error the user then sees must name the fix rather than
// surfacing a raw "no such file or directory".
func TestGetProfile_MissingFileIsActionable(t *testing.T) {
	_, err := GetProfile(t.TempDir())
	if !errors.Is(err, ErrNoProfile) {
		t.Fatalf("expected ErrNoProfile, got %v", err)
	}
	for _, want := range []string{"profile.example.yaml", "data/profile.yaml"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error message should tell the user about %q, got %q", want, err.Error())
		}
	}
}
