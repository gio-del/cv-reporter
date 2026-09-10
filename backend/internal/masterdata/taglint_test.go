package masterdata_test

import (
	"sort"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

func occ(tag, entryID string) masterdata.TagOccurrence {
	return masterdata.TagOccurrence{Tag: tag, EntryID: entryID}
}

// groupTags is a small test helper: which raw tag spellings ended up in the
// group with the given key, regardless of occurrence order.
func groupTags(t *testing.T, report masterdata.TagLintReport, key string) []string {
	t.Helper()
	for _, g := range report.Groups {
		if g.Key == key {
			var tags []string
			for _, o := range g.Occurrences {
				tags = append(tags, o.Tag)
			}
			sort.Strings(tags)
			return tags
		}
	}
	return nil
}

func groupConfidence(report masterdata.TagLintReport, key string) (masterdata.TagLintConfidence, bool) {
	for _, g := range report.Groups {
		if g.Key == key {
			return g.Confidence, true
		}
	}
	return "", false
}

func singletonTags(report masterdata.TagLintReport) []string {
	var tags []string
	for _, o := range report.Singletons {
		tags = append(tags, o.Tag)
	}
	sort.Strings(tags)
	return tags
}

func TestDetectTagGroups_CaseOnlyMatch_IsConfident(t *testing.T) {
	occurrences := []masterdata.TagOccurrence{
		occ("Go", "experience/a"),
		occ("go", "experience/b"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	tags := groupTags(t, report, "go")
	if len(tags) != 2 || tags[0] != "Go" || tags[1] != "go" {
		t.Fatalf("expected group 'go' with [Go, go], got %v", tags)
	}
	confidence, ok := groupConfidence(report, "go")
	if !ok || confidence != masterdata.TagLintConfident {
		t.Fatalf("expected confident group, got %v (ok=%v)", confidence, ok)
	}
}

func TestDetectTagGroups_AliasTableMatch_IsConfident(t *testing.T) {
	occurrences := []masterdata.TagOccurrence{
		occ("Go", "experience/a"),
		occ("Golang", "experience/b"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	tags := groupTags(t, report, "go")
	if len(tags) != 2 || tags[0] != "Go" || tags[1] != "Golang" {
		t.Fatalf("expected group 'go' with [Go, Golang], got %v", tags)
	}
	confidence, _ := groupConfidence(report, "go")
	if confidence != masterdata.TagLintConfident {
		t.Fatalf("expected confident group, got %v", confidence)
	}
}

func TestDetectTagGroups_EditDistanceMatch_IsSuggested(t *testing.T) {
	occurrences := []masterdata.TagOccurrence{
		occ("Kubernets", "experience/a"),
		occ("Kubernetes", "experience/b"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	tags := groupTags(t, report, "kubernetes")
	if len(tags) != 2 || tags[0] != "Kubernetes" || tags[1] != "Kubernets" {
		t.Fatalf("expected group 'kubernetes' with [Kubernetes, Kubernets], got %v", tags)
	}
	confidence, _ := groupConfidence(report, "kubernetes")
	if confidence != masterdata.TagLintSuggested {
		t.Fatalf("expected suggested group, got %v", confidence)
	}
}

func TestDetectTagGroups_EditDistanceNearMiss_DoesNotGroup(t *testing.T) {
	// "React" and "Redux" are 3 substitutions apart, one more than the
	// suggested-match threshold of 2 — they must NOT be grouped.
	occurrences := []masterdata.TagOccurrence{
		occ("React", "experience/a"),
		occ("Redux", "experience/b"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	if len(report.Groups) != 0 {
		t.Fatalf("expected no groups for React/Redux, got %v", report.Groups)
	}
	singles := singletonTags(report)
	if len(singles) != 2 || singles[0] != "React" || singles[1] != "Redux" {
		t.Fatalf("expected both React and Redux as singletons, got %v", singles)
	}
}

func TestDetectTagGroups_SingletonTags_AreInformationalNotHidden(t *testing.T) {
	occurrences := []masterdata.TagOccurrence{
		occ("Python", "experience/a"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	if len(report.Groups) != 0 {
		t.Fatalf("expected no groups, got %v", report.Groups)
	}
	singles := singletonTags(report)
	if len(singles) != 1 || singles[0] != "Python" {
		t.Fatalf("expected Python as a singleton, got %v", singles)
	}
}

func TestDetectTagGroups_FullFixture_GroupsAndTiersAsExpected(t *testing.T) {
	occurrences := []masterdata.TagOccurrence{
		occ("Go", "experience/a"),
		occ("go", "experience/b"),
		occ("Golang", "experience/c"),
		occ("Python", "experience/d"),
		occ("Kubernets", "experience/e"),
		occ("Kubernetes", "experience/f"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	if len(report.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %v", len(report.Groups), report.Groups)
	}

	goTags := groupTags(t, report, "go")
	if len(goTags) != 3 {
		t.Fatalf("expected 3 tags in 'go' group, got %v", goTags)
	}
	goConfidence, _ := groupConfidence(report, "go")
	if goConfidence != masterdata.TagLintConfident {
		t.Fatalf("expected 'go' group to be confident, got %v", goConfidence)
	}

	k8sTags := groupTags(t, report, "kubernetes")
	if len(k8sTags) != 2 {
		t.Fatalf("expected 2 tags in 'kubernetes' group, got %v", k8sTags)
	}
	k8sConfidence, _ := groupConfidence(report, "kubernetes")
	if k8sConfidence != masterdata.TagLintSuggested {
		t.Fatalf("expected 'kubernetes' group to be suggested, got %v", k8sConfidence)
	}

	singles := singletonTags(report)
	if len(singles) != 1 || singles[0] != "Python" {
		t.Fatalf("expected Python as the only singleton, got %v", singles)
	}
}

func TestDetectTagGroups_ShortAcronyms_DoNotChainByEditDistance(t *testing.T) {
	// AWS/GCP/MCP/A2A/Java are all short enough to be within the
	// edit-distance threshold of several unrelated acronyms — they must
	// not be chained into bogus "suggested" groups.
	occurrences := []masterdata.TagOccurrence{
		occ("AWS", "experience/a"),
		occ("GCP", "experience/b"),
		occ("MCP", "experience/c"),
		occ("A2A", "experience/d"),
		occ("Java", "experience/e"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	if len(report.Groups) != 0 {
		t.Fatalf("expected no groups for unrelated short acronyms, got %v", report.Groups)
	}
	if len(report.Singletons) != 5 {
		t.Fatalf("expected all 5 tags as singletons, got %d: %v", len(report.Singletons), report.Singletons)
	}
}

func TestDetectTagGroups_MultipleEntriesSameSpelling_NotAGroup(t *testing.T) {
	// The same exact spelling used by several Entries is not fragmentation
	// — it must not be reported as a duplicate group.
	occurrences := []masterdata.TagOccurrence{
		occ("React", "experience/a"),
		occ("React", "experience/b"),
		occ("React", "experience/c"),
	}

	report := masterdata.DetectTagGroups(occurrences)

	if len(report.Groups) != 0 {
		t.Fatalf("expected no groups, got %v", report.Groups)
	}
	if len(report.Singletons) != 3 {
		t.Fatalf("expected 3 singleton occurrences (one per Entry), got %d", len(report.Singletons))
	}
}
