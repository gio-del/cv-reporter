package api

import (
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

// tagLintOccurrence is the wire shape of one Tag spelling used by one
// Entry, enriched with the Entry's type so the FE can label it without a
// second lookup.
type tagLintOccurrence struct {
	Tag       string `json:"tag"`
	EntryID   string `json:"entryId"`
	EntryType string `json:"entryType"`
}

// tagLintGroup is the wire shape of one near-duplicate group.
type tagLintGroup struct {
	Key         string              `json:"key"`
	Confidence  string              `json:"confidence"`
	Occurrences []tagLintOccurrence `json:"occurrences"`
}

// tagLintReport is the wire shape of GET /api/master-data/tag-lint's body.
type tagLintReport struct {
	Groups     []tagLintGroup      `json:"groups"`
	Singletons []tagLintOccurrence `json:"singletons"`
}

// tagLintHandler recomputes the tag consistency lint report from
// ListEntries's current result on every call — there is no persistence or
// caching, since Master Data changes infrequently and the scan is cheap
// (see the PRD's Implementation Decisions). It is read-only: it never
// modifies any Entry file.
func tagLintHandler(dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := masterdata.ListEntries(dataDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		entryTypeByID := make(map[string]string, len(entries))
		var occurrences []masterdata.TagOccurrence
		for _, e := range entries {
			entryTypeByID[e.ID] = e.Type
			for _, tag := range e.Tags {
				occurrences = append(occurrences, masterdata.TagOccurrence{Tag: tag, EntryID: e.ID})
			}
		}

		report := masterdata.DetectTagGroups(occurrences)
		writeJSON(w, http.StatusOK, tagLintReport{
			Groups:     toTagLintGroups(report.Groups, entryTypeByID),
			Singletons: toTagLintOccurrences(report.Singletons, entryTypeByID),
		})
	}
}

func toTagLintGroups(groups []masterdata.TagGroup, entryTypeByID map[string]string) []tagLintGroup {
	out := make([]tagLintGroup, 0, len(groups))
	for _, g := range groups {
		out = append(out, tagLintGroup{
			Key:         g.Key,
			Confidence:  string(g.Confidence),
			Occurrences: toTagLintOccurrences(g.Occurrences, entryTypeByID),
		})
	}
	return out
}

func toTagLintOccurrences(occs []masterdata.TagOccurrence, entryTypeByID map[string]string) []tagLintOccurrence {
	out := make([]tagLintOccurrence, 0, len(occs))
	for _, o := range occs {
		out = append(out, tagLintOccurrence{
			Tag:       o.Tag,
			EntryID:   o.EntryID,
			EntryType: entryTypeByID[o.EntryID],
		})
	}
	return out
}
