package version

import "testing"

// TestUnstampedBuildReportsDev pins the default: a developer's `go build` must
// not claim to be a release. The release workflow overrides Version with
// -ldflags; nothing else should assign it.
func TestUnstampedBuildReportsDev(t *testing.T) {
	if Version != DevVersion {
		t.Fatalf("expected an unstamped test build to report %q, got %q", DevVersion, Version)
	}
	if IsRelease() {
		t.Fatal("an unstamped build must not report itself as a release")
	}
}

func TestIsRelease(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{value: "0.1.0", want: true},
		{value: DevVersion, want: false},
		{value: "", want: false},
	} {
		orig := Version
		Version = tc.value
		got := IsRelease()
		Version = orig
		if got != tc.want {
			t.Fatalf("IsRelease() with Version=%q = %v, want %v", tc.value, got, tc.want)
		}
	}
}
