package tracking_test

import (
	"os"
	"path/filepath"
	"testing"
)

// legacyJobListingFixture and legacyApplicationFixture are, byte for byte,
// the oldest record shape known to exist on disk (issue #100): a Job
// Listing written before freshnessStatus existed, and its Application
// written before statusUpdatedAt, statusHistory and generations existed,
// still at Status Saved with its Application Method unresolved. The
// values are invented; the shape (key set, key order, 4-space indentation,
// quoting, the blank line after the closing delimiter) is what
// renderJobListing and renderApplication produced at the time.
const legacyJobListingFixture = "---\n" +
	"title: Backend Engineer\n" +
	"company: Example Co\n" +
	"url: https://jobs.example.com/example-co/1\n" +
	"source: manual\n" +
	"savedAt: \"2026-09-08T09:14:07.123456789Z\"\n" +
	"ral:\n" +
	"    min: 45000\n" +
	"    max: 55000\n" +
	"    currency: EUR\n" +
	"    source: estimated\n" +
	"---\n" +
	"\n" +
	"## About the role\n" +
	"\n" +
	"Build Go services.\n"

const legacyApplicationFixture = "jobListingId: example-co\n" +
	"status: saved\n" +
	"method:\n" +
	"    kind: unresolved\n" +
	"    value: \"\"\n"

const legacyFixtureID = "example-co"
const legacyFixtureSavedAt = "2026-09-08T09:14:07.123456789Z"

// writeRecord writes content to dataDir/dir/name, creating dir as needed.
func writeRecord(t *testing.T, dataDir, dir, name, content string) string {
	t.Helper()
	full := filepath.Join(dataDir, dir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(full, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// seedLegacyPair writes the legacy fixture pair into a fresh temp dataDir.
func seedLegacyPair(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	writeRecord(t, dataDir, "jobs", legacyFixtureID+".md", legacyJobListingFixture)
	writeRecord(t, dataDir, "applications", legacyFixtureID+".md", legacyApplicationFixture)
	return dataDir
}

func readRecord(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
