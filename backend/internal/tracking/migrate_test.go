package tracking_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/tracking"
)

// snapshotDir returns every file under dataDir keyed by its slash-separated
// relative path, so a test can assert nothing on disk changed.
func snapshotDir(t *testing.T, dataDir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dataDir, path)
		files[filepath.ToSlash(rel)] = readRecord(t, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func migrate(t *testing.T, dataDir string, dryRun bool) tracking.MigrationReport {
	t.Helper()
	report, err := tracking.MigrateRecords(dataDir, tracking.MigrationOptions{DryRun: dryRun})
	if err != nil {
		t.Fatalf("MigrateRecords(dryRun=%v): %v", dryRun, err)
	}
	return report
}

func findMigration(report tracking.MigrationReport, path string) (tracking.RecordMigration, bool) {
	for _, m := range report.Migrated {
		if m.Path == path {
			return m, true
		}
	}
	return tracking.RecordMigration{}, false
}

func backfilled(m tracking.RecordMigration, field string) (string, bool) {
	for _, f := range m.Backfilled {
		if f.Field == field {
			return f.Value, true
		}
	}
	return "", false
}

func TestMigrateRecords_LegacyJobListing_StampsVersionAndFreshness(t *testing.T) {
	dataDir := seedLegacyPair(t)
	before, err := tracking.GetJobListing(dataDir, legacyFixtureID)
	if err != nil {
		t.Fatal(err)
	}

	report := migrate(t, dataDir, false)

	m, ok := findMigration(report, "jobs/example-co.md")
	if !ok {
		t.Fatalf("expected the report to name jobs/example-co.md, got %+v", report.Migrated)
	}
	if m.Kind != tracking.RecordJobListing || m.FromVersion != tracking.LegacySchemaVersion || m.ToVersion != tracking.CurrentSchemaVersion {
		t.Errorf("expected a job listing migrated from v%d to v%d, got %+v", tracking.LegacySchemaVersion, tracking.CurrentSchemaVersion, m)
	}
	if v, ok := backfilled(m, "freshnessStatus"); !ok || v != string(tracking.FreshnessNotYetChecked) {
		t.Errorf("expected freshnessStatus backfilled as not-yet-checked, got %+v", m.Backfilled)
	}

	after, err := tracking.GetJobListing(dataDir, legacyFixtureID)
	if err != nil {
		t.Fatal(err)
	}
	if after.SchemaVersion != tracking.CurrentSchemaVersion {
		t.Errorf("expected the Job Listing at v%d on disk, got v%d", tracking.CurrentSchemaVersion, after.SchemaVersion)
	}
	if !strings.Contains(readRecord(t, filepath.Join(dataDir, "jobs", "example-co.md")), "freshnessStatus: not-yet-checked") {
		t.Error("expected freshnessStatus written explicitly to the frontmatter")
	}
	after.SchemaVersion = before.SchemaVersion
	if !reflect.DeepEqual(before, after) {
		t.Errorf("migration changed more than the version:\nbefore: %+v\nafter:  %+v", before, after)
	}
}

func TestMigrateRecords_JobListingWithFreshness_OnlyStampsVersion(t *testing.T) {
	dataDir := seedLegacyPair(t)
	checked := strings.Replace(legacyJobListingFixture, "---\n\n", "freshnessStatus: unreachable\nfreshnessCheckedAt: \"2026-09-10T08:00:00Z\"\n---\n\n", 1)
	writeRecord(t, dataDir, "jobs", legacyFixtureID+".md", checked)

	m, ok := findMigration(migrate(t, dataDir, false), "jobs/example-co.md")
	if !ok {
		t.Fatal("expected the unversioned Job Listing to be migrated")
	}
	if len(m.Backfilled) != 0 {
		t.Errorf("expected nothing backfilled on a Job Listing already carrying freshnessStatus, got %+v", m.Backfilled)
	}
	after, err := tracking.GetJobListing(dataDir, legacyFixtureID)
	if err != nil {
		t.Fatal(err)
	}
	if after.FreshnessStatus != tracking.FreshnessUnreachable || after.FreshnessCheckedAt != "2026-09-10T08:00:00Z" {
		t.Errorf("expected the prior freshness check kept, got %q at %q", after.FreshnessStatus, after.FreshnessCheckedAt)
	}
}

// The Job Description body is split from the frontmatter on a delimiter,
// so this is the one place asserting on raw bytes is right: everything
// from the closing delimiter onwards must survive exactly.
func TestMigrateRecords_JobDescriptionBodySurvivesByteForByte(t *testing.T) {
	dataDir := seedLegacyPair(t)
	tail := "---\n\n## Role\n\nIntro line with trailing spaces   \n\n---\n\nA horizontal rule above, then:\n\n---\ntitle: not frontmatter\n---\n\ttabbed\r\nwindows line\r\n\n\n\n"
	fixture := strings.SplitAfter(legacyJobListingFixture, "    source: estimated\n")[0] + tail
	path := writeRecord(t, dataDir, "jobs", legacyFixtureID+".md", fixture)

	migrate(t, dataDir, false)

	got := readRecord(t, path)
	if !strings.HasSuffix(got, "\n"+tail) {
		t.Errorf("body not preserved byte-for-byte.\nwant suffix: %q\ngot file:    %q", tail, got)
	}
}

func TestMigrateRecords_LeavesCompanyLogoAndUnknownKeysAlone(t *testing.T) {
	dataDir := seedLegacyPair(t)
	logo := string([]byte{0x89, 'P', 'N', 'G', 0, 1, 2, 3})
	logoPath := writeRecord(t, dataDir, "jobs", "example-co-logo.png", logo)
	withExtras := strings.Replace(legacyJobListingFixture, "---\n\n", "logo: example-co-logo.png\narchived: true\n# a hand-written comment\nsomeFutureKey: keep me\n---\n\n", 1)
	jobPath := writeRecord(t, dataDir, "jobs", legacyFixtureID+".md", withExtras)
	logoInfo, _ := os.Stat(logoPath)

	migrate(t, dataDir, false)

	if readRecord(t, logoPath) != logo {
		t.Error("expected the Company Logo file untouched")
	}
	if info, _ := os.Stat(logoPath); !info.ModTime().Equal(logoInfo.ModTime()) {
		t.Error("expected the Company Logo file never rewritten")
	}
	got := readRecord(t, jobPath)
	for _, want := range []string{"someFutureKey: keep me", "# a hand-written comment", "logo: example-co-logo.png", "archived: true"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected migration to keep %q, got:\n%s", want, got)
		}
	}
	listing, err := tracking.GetJobListing(dataDir, legacyFixtureID)
	if err != nil {
		t.Fatal(err)
	}
	if !listing.Archived || listing.Logo != "example-co-logo.png" {
		t.Errorf("expected archived and logo to read back unchanged, got %+v", listing)
	}
}

func TestMigrateRecords_DryRunReportsWithoutWriting(t *testing.T) {
	dataDir := seedLegacyPair(t)
	before := snapshotDir(t, dataDir)

	report := migrate(t, dataDir, true)

	if !report.DryRun || !report.Pending() {
		t.Errorf("expected a dry run with work pending, got DryRun=%v Pending=%v", report.DryRun, report.Pending())
	}
	if _, ok := findMigration(report, "jobs/example-co.md"); !ok {
		t.Errorf("expected the dry run to name the Job Listing it would migrate, got %+v", report.Migrated)
	}
	if after := snapshotDir(t, dataDir); !reflect.DeepEqual(before, after) {
		t.Errorf("dry run changed files on disk:\nbefore: %v\nafter:  %v", before, after)
	}
}

func TestMigrateRecords_IsIdempotent(t *testing.T) {
	dataDir := seedLegacyPair(t)
	migrate(t, dataDir, false)
	afterFirst := snapshotDir(t, dataDir)

	dry := migrate(t, dataDir, true)
	second := migrate(t, dataDir, false)

	if dry.Pending() || second.Pending() || len(second.Migrated) != 0 {
		t.Errorf("expected a migrated corpus to report nothing to do, got dry=%+v second=%+v", dry.Migrated, second.Migrated)
	}
	if second.Scanned == 0 {
		t.Error("expected the second run still to count the records it scanned")
	}
	if afterSecond := snapshotDir(t, dataDir); !reflect.DeepEqual(afterFirst, afterSecond) {
		t.Errorf("second run changed files:\nfirst:  %v\nsecond: %v", afterFirst, afterSecond)
	}
}

func TestMigrateRecords_EmptyOrMissingDataDirIsNotAnError(t *testing.T) {
	for name, dataDir := range map[string]string{
		"empty":   t.TempDir(),
		"missing": filepath.Join(t.TempDir(), "does-not-exist"),
		"empty subdirs": func() string {
			d := t.TempDir()
			os.MkdirAll(filepath.Join(d, "jobs"), 0o755)
			os.MkdirAll(filepath.Join(d, "applications"), 0o755)
			return d
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			report := migrate(t, dataDir, false)
			if report.Pending() || report.Scanned != 0 || len(report.Inconsistencies) != 0 {
				t.Errorf("expected an empty report, got %+v", report)
			}
		})
	}
}

func TestMigrateRecords_UnparseableRecordHaltsNamingTheFileAndWritesNothing(t *testing.T) {
	for name, broken := range map[string]string{
		"invalid yaml":   "---\ncompany: [unclosed\n---\n\nBody.\n",
		"no frontmatter": "Just a body, no delimiters.\n",
		"unclosed fence": "---\ncompany: Broken Co\n",
		"not a mapping":  "---\n- a\n- list\n---\n\nBody.\n",
	} {
		t.Run(name, func(t *testing.T) {
			dataDir := seedLegacyPair(t)
			writeRecord(t, dataDir, "jobs", "broken-co.md", broken)
			writeRecord(t, dataDir, "applications", "broken-co.md", legacyApplicationFixture)
			before := snapshotDir(t, dataDir)

			_, err := tracking.MigrateRecords(dataDir, tracking.MigrationOptions{})
			if err == nil || !strings.Contains(err.Error(), "jobs/broken-co.md") {
				t.Fatalf("expected an error naming jobs/broken-co.md, got %v", err)
			}
			if after := snapshotDir(t, dataDir); !reflect.DeepEqual(before, after) {
				t.Error("expected no file written when a record is unparseable, including the valid ones")
			}
		})
	}
}

func TestMigrateRecords_NewerSchemaVersionHalts(t *testing.T) {
	dataDir := seedLegacyPair(t)
	writeRecord(t, dataDir, "jobs", "newer-co.md", "---\nschemaVersion: 99\ncompany: Newer Co\n---\n\nBody.\n")
	writeRecord(t, dataDir, "applications", "newer-co.md", legacyApplicationFixture)

	_, err := tracking.MigrateRecords(dataDir, tracking.MigrationOptions{})
	if !errors.Is(err, tracking.ErrUnsupportedSchemaVersion) || !strings.Contains(err.Error(), "jobs/newer-co.md") {
		t.Fatalf("expected ErrUnsupportedSchemaVersion naming jobs/newer-co.md, got %v", err)
	}
}

func TestMigrateRecords_JobListingWithoutApplicationIsReportedAndStillMigrated(t *testing.T) {
	dataDir := seedLegacyPair(t)
	os.Remove(filepath.Join(dataDir, "applications", legacyFixtureID+".md"))

	report := migrate(t, dataDir, false)

	found := false
	for _, inc := range report.Inconsistencies {
		if inc.Path == "jobs/example-co.md" && strings.Contains(inc.Problem, "applications/example-co.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected the missing Application reported against jobs/example-co.md, got %+v", report.Inconsistencies)
	}
	listing, err := tracking.GetJobListing(dataDir, legacyFixtureID)
	if err != nil {
		t.Fatal(err)
	}
	if listing.SchemaVersion != tracking.CurrentSchemaVersion {
		t.Errorf("expected the Job Listing still migrated, got v%d", listing.SchemaVersion)
	}
}

func TestMigrateRecords_LeavesNoTemporaryFilesBehind(t *testing.T) {
	dataDir := seedLegacyPair(t)
	migrate(t, dataDir, false)
	for path := range snapshotDir(t, dataDir) {
		if path != "jobs/example-co.md" && path != "applications/example-co.md" {
			t.Errorf("unexpected file left behind: %s", path)
		}
	}
}
