package atomicfile

import (
	"strings"
	"testing"
)

// A temp file stranded by a crash between create and rename must never be
// mistaken for a record: both list paths enumerate by the ".md" suffix and
// fail the whole collection on one unparseable file, and the slug-uniqueness
// probe checks for "<slug>.md" (story 10). That makes the naming rule a
// correctness constraint, so it is asserted directly.
func TestTempPattern_IsInvisibleToReaders(t *testing.T) {
	for _, base := range []string{"record.md", "profile.yaml", "usage-log.json", "acme.png"} {
		pattern := tempPattern(base)
		name := strings.Replace(pattern, "*", "1234567890", 1)

		if !strings.HasPrefix(name, ".") {
			t.Errorf("tempPattern(%q) = %q: expected a dot-prefixed name", base, pattern)
		}
		for _, ext := range []string{".md", ".yaml", ".json"} {
			if strings.HasSuffix(name, ext) {
				t.Errorf("tempPattern(%q) = %q: must not end in %q, readers would pick it up", base, pattern, ext)
			}
		}
		if !strings.Contains(pattern, "*") {
			t.Errorf("tempPattern(%q) = %q: expected a random-substitution marker", base, pattern)
		}
	}
}
