package tracking

import "time"

// StaleEntries returns the subset of entryIDs that lookupFn reports as
// modified after generationCreatedAt — the Entries a Generation drew from
// (per its recorded GenerationRecord.EntryIDs) that Master Data has since
// changed (issue #52). An empty result means not stale.
//
// It has no dependency on Selection, Rewrite, or any Claude call, and no
// dependency on how lookupFn resolves an Entry's last-modified time (see
// masterdata.EntryLastModified, issue #40) — only that it reports ok=false
// when it can't find one, which this function treats as "not checkable",
// never as stale. entryIDs being empty (a GenerationRecord persisted before
// this field existed) degrades the same way: nothing to check, so nothing
// is reported stale.
func StaleEntries(entryIDs []string, generationCreatedAt time.Time, lookupFn func(entryID string) (time.Time, bool)) []string {
	var stale []string
	for _, id := range entryIDs {
		lastModified, ok := lookupFn(id)
		if !ok {
			continue
		}
		if lastModified.After(generationCreatedAt) {
			stale = append(stale, id)
		}
	}
	return stale
}
