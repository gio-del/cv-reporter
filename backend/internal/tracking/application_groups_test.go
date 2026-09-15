package tracking_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/gio-del/sumisura/backend/internal/tracking"
)

var groupsNow = time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

// daysAgo is a StatusUpdatedAt value d days before groupsNow.
func daysAgo(d int) string {
	return groupsNow.Add(-time.Duration(d) * 24 * time.Hour).Format(time.RFC3339Nano)
}

func pair(id string, status tracking.Status, statusUpdatedAt string) tracking.ListingWithApplication {
	return tracking.ListingWithApplication{
		JobListing:  tracking.JobListing{ID: id, Company: "Company " + id},
		Application: tracking.Application{ID: id, JobListingID: id, Status: status, StatusUpdatedAt: statusUpdatedAt},
	}
}

func groupFor(t *testing.T, groups tracking.ApplicationGroups, status tracking.Status) tracking.ApplicationGroup {
	t.Helper()
	for _, g := range groups.Groups {
		if g.Status == status {
			return g
		}
	}
	t.Fatalf("no group found for status %q", status)
	return tracking.ApplicationGroup{}
}

func itemIDs(g tracking.ApplicationGroup) []string {
	ids := make([]string, len(g.Items))
	for i, item := range g.Items {
		ids[i] = item.Application.ID
	}
	return ids
}

func TestGroupApplications_ZeroApplications_EveryStatusGroupEmptyInPipelineOrder(t *testing.T) {
	groups := tracking.GroupApplications(nil, groupsNow, tracking.DefaultStaleThreshold)

	want := []tracking.Status{
		tracking.StatusSaved,
		tracking.StatusTailoring,
		tracking.StatusSent,
		tracking.StatusInterviewing,
		tracking.StatusOffer,
		tracking.StatusRejected,
		tracking.StatusWithdrawn,
	}
	if len(groups.Groups) != len(want) {
		t.Fatalf("expected %d groups, got %d: %+v", len(want), len(groups.Groups), groups.Groups)
	}
	for i, status := range want {
		g := groups.Groups[i]
		if g.Status != status {
			t.Errorf("group %d: expected status %q, got %q", i, status, g.Status)
		}
		if g.Count != 0 {
			t.Errorf("group %q: expected count 0, got %d", status, g.Count)
		}
		if g.Items == nil {
			t.Errorf("group %q: expected an empty, non-nil item list", status)
		}
	}
	if groups.Total != 0 {
		t.Errorf("expected total 0, got %d", groups.Total)
	}
}

func TestGroupApplications_EachApplicationLandsOnceInItsStatusGroup_CountsMatchItems(t *testing.T) {
	items := []tracking.ListingWithApplication{
		pair("saved-1", tracking.StatusSaved, daysAgo(1)),
		pair("sent-1", tracking.StatusSent, daysAgo(2)),
		pair("saved-2", tracking.StatusSaved, daysAgo(3)),
		pair("offer-1", tracking.StatusOffer, daysAgo(4)),
		pair("withdrawn-1", tracking.StatusWithdrawn, daysAgo(5)),
	}

	groups := tracking.GroupApplications(items, groupsNow, tracking.DefaultStaleThreshold)

	if groups.Total != len(items) {
		t.Errorf("expected total %d, got %d", len(items), groups.Total)
	}
	seen := 0
	for _, g := range groups.Groups {
		if g.Count != len(g.Items) {
			t.Errorf("group %q: count %d does not match %d items", g.Status, g.Count, len(g.Items))
		}
		for _, item := range g.Items {
			if item.Application.Status != g.Status {
				t.Errorf("group %q holds %q in Status %q", g.Status, item.Application.ID, item.Application.Status)
			}
		}
		seen += len(g.Items)
	}
	if seen != len(items) {
		t.Errorf("expected every Application exactly once (%d), saw %d", len(items), seen)
	}
	if got := groupFor(t, groups, tracking.StatusSaved).Count; got != 2 {
		t.Errorf("expected 2 in Saved, got %d", got)
	}
	if got := groupFor(t, groups, tracking.StatusInterviewing).Count; got != 0 {
		t.Errorf("expected 0 in Interviewing, got %d", got)
	}
}

func TestGroupApplications_WithinAGroup_LeastRecentlyChangedFirst(t *testing.T) {
	items := []tracking.ListingWithApplication{
		pair("yesterday", tracking.StatusTailoring, daysAgo(1)),
		pair("last-month", tracking.StatusTailoring, daysAgo(30)),
		pair("last-week", tracking.StatusTailoring, daysAgo(7)),
	}

	groups := tracking.GroupApplications(items, groupsNow, tracking.DefaultStaleThreshold)

	got := itemIDs(groupFor(t, groups, tracking.StatusTailoring))
	want := []string{"last-month", "last-week", "yesterday"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestGroupApplications_StaleItemsPrecedeNonStale_EachRunLeastRecentlyChangedFirst(t *testing.T) {
	// Under a 14-day threshold: 20 and 15 days are stale, 10 and 2 are not.
	items := []tracking.ListingWithApplication{
		pair("fresh-2", tracking.StatusSent, daysAgo(2)),
		pair("stale-15", tracking.StatusSent, daysAgo(15)),
		pair("fresh-10", tracking.StatusSent, daysAgo(10)),
		pair("stale-20", tracking.StatusSent, daysAgo(20)),
	}

	groups := tracking.GroupApplications(items, groupsNow, tracking.DefaultStaleThreshold)

	sent := groupFor(t, groups, tracking.StatusSent)
	want := []string{"stale-20", "stale-15", "fresh-10", "fresh-2"}
	if got := itemIDs(sent); !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
	wantStale := []bool{true, true, false, false}
	for i, item := range sent.Items {
		if item.Application.IsStale != wantStale[i] {
			t.Errorf("item %q: expected isStale %v, got %v", item.Application.ID, wantStale[i], item.Application.IsStale)
		}
	}
}

func TestGroupApplications_StalenessUsesInjectedNowAndThreshold_NotTheLoadedFlag(t *testing.T) {
	// The loaded flag was computed against the wall clock at read time; the
	// grouping recomputes it so ordering and flag always agree.
	loadedAsStale := pair("recent", tracking.StatusInterviewing, daysAgo(3))
	loadedAsStale.Application.IsStale = true
	loadedAsFresh := pair("old", tracking.StatusInterviewing, daysAgo(5))

	groups := tracking.GroupApplications(
		[]tracking.ListingWithApplication{loadedAsStale, loadedAsFresh},
		groupsNow,
		4*24*time.Hour,
	)

	interviewing := groupFor(t, groups, tracking.StatusInterviewing)
	if got, want := itemIDs(interviewing), []string{"old", "recent"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if !interviewing.Items[0].Application.IsStale || interviewing.Items[1].Application.IsStale {
		t.Errorf("expected only %q stale under a 4-day threshold, got %+v", "old", interviewing.Items)
	}
}

func TestGroupApplications_OnlySentAndInterviewingEverProduceStaleItems(t *testing.T) {
	var items []tracking.ListingWithApplication
	for _, status := range []tracking.Status{
		tracking.StatusSaved, tracking.StatusTailoring, tracking.StatusSent, tracking.StatusInterviewing,
		tracking.StatusOffer, tracking.StatusRejected, tracking.StatusWithdrawn,
	} {
		items = append(items, pair(string(status), status, daysAgo(90)))
	}

	groups := tracking.GroupApplications(items, groupsNow, tracking.DefaultStaleThreshold)

	for _, g := range groups.Groups {
		wantStale := g.Status == tracking.StatusSent || g.Status == tracking.StatusInterviewing
		for _, item := range g.Items {
			if item.Application.IsStale != wantStale {
				t.Errorf("group %q: expected isStale %v, got %v", g.Status, wantStale, item.Application.IsStale)
			}
		}
	}
}

func TestGroupApplications_AbsentOrUnparseableLastChanged_SortsLastNeverFirst(t *testing.T) {
	items := []tracking.ListingWithApplication{
		pair("missing", tracking.StatusSent, ""),
		pair("recent", tracking.StatusSent, daysAgo(1)),
		pair("garbage", tracking.StatusSent, "not-a-timestamp"),
		pair("stale", tracking.StatusSent, daysAgo(30)),
		pair("older", tracking.StatusSent, daysAgo(10)),
	}

	groups := tracking.GroupApplications(items, groupsNow, tracking.DefaultStaleThreshold)

	got := itemIDs(groupFor(t, groups, tracking.StatusSent))
	want := []string{"stale", "older", "recent", "missing", "garbage"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestGroupApplications_RecordsMissingOptionalFields_StillAppearInTheirGroup(t *testing.T) {
	// A Job Listing saved before Status history, statusUpdatedAt,
	// Generations or a Company Logo existed.
	old := tracking.ListingWithApplication{
		JobListing:  tracking.JobListing{ID: "legacy", Company: "Legacy Corp"},
		Application: tracking.Application{ID: "legacy", JobListingID: "legacy", Status: tracking.StatusInterviewing},
	}

	groups := tracking.GroupApplications([]tracking.ListingWithApplication{old}, groupsNow, tracking.DefaultStaleThreshold)

	interviewing := groupFor(t, groups, tracking.StatusInterviewing)
	if got, want := itemIDs(interviewing), []string{"legacy"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	if interviewing.Items[0].JobListing.Company != "Legacy Corp" {
		t.Errorf("expected the Job Listing to travel with its Application, got %+v", interviewing.Items[0].JobListing)
	}
	if interviewing.Items[0].Application.IsStale {
		t.Errorf("expected a record with no statusUpdatedAt not to be stale")
	}
}
