package tracking

import "time"

// SnippetLastUsed scans every Application's recorded Generation history and
// returns, per Cover Letter Snippet id, the CreatedAt of the most recent
// Generation that cited it (RFC3339Nano string, matching GenerationRecord).
// A Generation record with no SourceSnippetIDs contributes nothing — that's
// either "no Snippet was used" or "recorded before this field existed", and
// both must stay indistinguishable from "never used" here; the caller (the
// Snippets list) is what decides how to label an id absent from this map.
func SnippetLastUsed(applications []Application) map[string]string {
	lastUsed := make(map[string]string)
	for _, app := range applications {
		for _, gen := range app.Generations {
			for _, id := range gen.SourceSnippetIDs {
				if isMoreRecent(gen.CreatedAt, lastUsed[id]) {
					lastUsed[id] = gen.CreatedAt
				}
			}
		}
	}
	return lastUsed
}

// isMoreRecent reports whether candidate is a later timestamp than current
// ("" counts as never). Malformed timestamps lose to any parseable one and
// tie amongst themselves, rather than panicking or silently misordering.
func isMoreRecent(candidate, current string) bool {
	if current == "" {
		return true
	}
	c, cErr := time.Parse(time.RFC3339Nano, candidate)
	cur, curErr := time.Parse(time.RFC3339Nano, current)
	if cErr != nil {
		return false
	}
	if curErr != nil {
		return true
	}
	return c.After(cur)
}
