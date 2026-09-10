package masterdata

import (
	"sort"
	"strings"
)

// TagOccurrence is one Entry's use of one exact Tag spelling — the input to
// DetectTagGroups. It carries no file I/O or HTTP concerns; callers build
// these from ListEntries's result (see the Tag consistency lint PRD's
// Testing Decisions).
type TagOccurrence struct {
	Tag     string
	EntryID string
}

// TagLintConfidence is the confidence tier of a TagGroup: how sure
// DetectTagGroups is that a group is really the same tag spelled
// differently, versus merely a suggestion worth a human's judgment.
type TagLintConfidence string

const (
	// TagLintConfident groups tags that differ only by case, or are linked
	// by the small built-in alias table (Go/Golang, JS/JavaScript, ...).
	TagLintConfident TagLintConfidence = "confident"
	// TagLintSuggested groups tags that are merely close by edit distance
	// — a possible typo, not a certain match.
	TagLintSuggested TagLintConfidence = "suggested"
)

// TagGroup is one cluster of near-duplicate tag spellings.
type TagGroup struct {
	// Key is the normalized form the group was clustered under (lowercased
	// tag text, or its alias-table canonical form).
	Key         string
	Confidence  TagLintConfidence
	Occurrences []TagOccurrence
}

// TagLintReport is DetectTagGroups's result: near-duplicate groups, plus
// every tag occurrence that didn't end up in one (shown as informational,
// never hidden — story 6 of the Tag consistency lint PRD).
type TagLintReport struct {
	Groups     []TagGroup
	Singletons []TagOccurrence
}

// editDistanceThreshold is the maximum Levenshtein distance (on normalized,
// lowercased tag text) for an edit-distance match to be considered a
// "suggested" (not "confident") group.
const editDistanceThreshold = 2

// minLengthForEditDistanceMatch is the shortest a normalized tag may be to
// take part in edit-distance clustering at all. Short acronyms (AWS, GCP,
// MCP, ...) are only 3-4 characters apart from many unrelated acronyms
// within editDistanceThreshold, so without this gate they chain together
// into noisy, unrelated "suggested" groups — observed against this repo's
// own real Master Data during development. Case-only and alias-table
// (confident) matching is unaffected by this gate.
const minLengthForEditDistanceMatch = 5

// eligibleForEditDistanceMatch reports whether a and b are both long enough
// for a Levenshtein-distance match between them to be meaningful rather
// than incidental.
func eligibleForEditDistanceMatch(a, b string) bool {
	return len([]rune(a)) >= minLengthForEditDistanceMatch && len([]rune(b)) >= minLengthForEditDistanceMatch
}

// tagAliases is a small, hardcoded seed of common tech-naming variants that
// should be treated as the same tag even though they aren't a case-only
// match. It is not user-editable Master Data (see the PRD's Implementation
// Decisions) — Tags remain free-text and per-Entry.
var tagAliases = map[string]string{
	"go":         "go",
	"golang":     "go",
	"js":         "javascript",
	"javascript": "javascript",
	"ts":         "typescript",
	"typescript": "typescript",
	"node":       "node",
	"node.js":    "node",
	"nodejs":     "node",
	"k8s":        "kubernetes",
	"kubernetes": "kubernetes",
}

// confidentKey returns the key DetectTagGroups uses to cluster confident
// (case-only or alias-table) matches: the alias table's canonical form when
// the normalized tag is a known variant, otherwise the normalized tag
// itself.
func confidentKey(tag string) string {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	if canonical, ok := tagAliases[normalized]; ok {
		return canonical
	}
	return normalized
}

// DetectTagGroups groups occurrences whose tags look like near-duplicate
// spellings of each other. It is a pure function: the same input always
// produces the same output, with no file I/O or HTTP involved.
//
// Grouping runs in two passes:
//  1. Occurrences are bucketed by confidentKey. A bucket that contains more
//     than one distinct raw spelling is a "confident" group — its members
//     differ only by case, or are linked by the alias table.
//  2. Buckets left with a single distinct spelling (no confident match)
//     are clustered pairwise by Levenshtein edit distance on their
//     confidentKey text. Clusters of two or more become a "suggested"
//     group. This never reconsiders tags already placed in a confident
//     group.
//
// Any occurrence not placed in a group (by either pass) is returned as a
// Singleton — informational, never hidden (story 6).
func DetectTagGroups(occurrences []TagOccurrence) TagLintReport {
	byConfidentKey := make(map[string][]TagOccurrence)
	var keyOrder []string
	for _, o := range occurrences {
		key := confidentKey(o.Tag)
		if _, seen := byConfidentKey[key]; !seen {
			keyOrder = append(keyOrder, key)
		}
		byConfidentKey[key] = append(byConfidentKey[key], o)
	}

	var groups []TagGroup
	var soloKeys []string
	for _, key := range keyOrder {
		occs := byConfidentKey[key]
		if distinctSpellingCount(occs) >= 2 {
			groups = append(groups, TagGroup{
				Key:         key,
				Confidence:  TagLintConfident,
				Occurrences: occs,
			})
			continue
		}
		soloKeys = append(soloKeys, key)
	}

	suggestedGroups, ungroupedKeys := clusterByEditDistance(soloKeys)
	for _, cluster := range suggestedGroups {
		var occs []TagOccurrence
		for _, key := range cluster {
			occs = append(occs, byConfidentKey[key]...)
		}
		groups = append(groups, TagGroup{
			Key:         cluster[0],
			Confidence:  TagLintSuggested,
			Occurrences: occs,
		})
	}

	var singletons []TagOccurrence
	for _, key := range ungroupedKeys {
		singletons = append(singletons, byConfidentKey[key]...)
	}

	sortGroups(groups)
	sortOccurrences(singletons)

	return TagLintReport{Groups: groups, Singletons: singletons}
}

// distinctSpellingCount returns how many distinct raw (case-sensitive) tag
// spellings appear among occs.
func distinctSpellingCount(occs []TagOccurrence) int {
	seen := make(map[string]struct{})
	for _, o := range occs {
		seen[o.Tag] = struct{}{}
	}
	return len(seen)
}

// clusterByEditDistance unions keys whose Levenshtein distance is within
// editDistanceThreshold, using each key's first appearance order as a
// stable tie-breaker. It returns clusters with 2+ members (suggested
// groups, keys sorted for deterministic output) and the keys left alone.
func clusterByEditDistance(keys []string) (clusters [][]string, alone []string) {
	parent := make(map[string]string, len(keys))
	for _, k := range keys {
		parent[k] = k
	}
	var find func(string) string
	find = func(k string) string {
		if parent[k] != k {
			parent[k] = find(parent[k])
		}
		return parent[k]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if !eligibleForEditDistanceMatch(keys[i], keys[j]) {
				continue
			}
			if levenshtein(keys[i], keys[j]) <= editDistanceThreshold {
				union(keys[i], keys[j])
			}
		}
	}

	byRoot := make(map[string][]string)
	for _, k := range keys {
		root := find(k)
		byRoot[root] = append(byRoot[root], k)
	}

	for _, members := range byRoot {
		sort.Strings(members)
		if len(members) >= 2 {
			clusters = append(clusters, members)
		} else {
			alone = append(alone, members[0])
		}
	}

	sort.Slice(clusters, func(i, j int) bool { return clusters[i][0] < clusters[j][0] })
	sort.Strings(alone)
	return clusters, alone
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			deletion := prev[j] + 1
			insertion := curr[j-1] + 1
			substitution := prev[j-1] + cost
			curr[j] = min3(deletion, insertion, substitution)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func sortGroups(groups []TagGroup) {
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })
	for i := range groups {
		sortOccurrences(groups[i].Occurrences)
	}
}

func sortOccurrences(occs []TagOccurrence) {
	sort.Slice(occs, func(i, j int) bool {
		if occs[i].EntryID != occs[j].EntryID {
			return occs[i].EntryID < occs[j].EntryID
		}
		return occs[i].Tag < occs[j].Tag
	})
}
