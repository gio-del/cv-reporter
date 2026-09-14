package masterdata_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/masterdata"
)

const sampleEntry = "---\nemployer: Acme\nrole: Engineer\nstart: \"2020\"\n---\n\n- shipped a thing\n"

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// makeReadOnly makes dir unwritable, which is what a write that cannot
// complete looks like from the store's side: the previous record must
// survive it.
func makeReadOnly(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

// Story 2: an interrupted save of an Entry leaves the previous version
// intact, rather than truncating years of accumulated bullets.
func TestUpdateEntry_FailedWriteLeavesPreviousEntryIntact(t *testing.T) {
	dataDir := t.TempDir()
	path := filepath.Join(dataDir, "experience", "acme.md")
	writeFile(t, path, sampleEntry)
	makeReadOnly(t, filepath.Join(dataDir, "experience"))

	entry := masterdata.Entry{
		Type:     masterdata.TypeExperience,
		Employer: "Acme",
		Role:     "Engineer",
		Start:    "2020",
		Bullets:  []string{"a replacement bullet"},
	}
	if _, err := masterdata.UpdateEntry(dataDir, "experience/acme", entry); err == nil {
		t.Fatal("expected an error when the write cannot complete, got nil")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading entry after failed write: %v", err)
	}
	if !bytes.Equal(got, []byte(sampleEntry)) {
		t.Errorf("entry was damaged by a failed write:\nwant %q\ngot  %q", sampleEntry, string(got))
	}
}

// Story 4: the same guarantee for a Cover Letter Snippet.
func TestUpdateSnippet_FailedWriteLeavesPreviousSnippetIntact(t *testing.T) {
	dataDir := t.TempDir()
	const original = "---\nkind: opening\n---\n\nOriginal body.\n"
	path := filepath.Join(dataDir, "cover-letter-snippets", "opening.md")
	writeFile(t, path, original)
	makeReadOnly(t, filepath.Join(dataDir, "cover-letter-snippets"))

	snippet := masterdata.Snippet{Kind: "opening", Body: "Replacement body."}
	if _, err := masterdata.UpdateSnippet(dataDir, "opening", snippet); err == nil {
		t.Fatal("expected an error when the write cannot complete, got nil")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading snippet after failed write: %v", err)
	}
	if string(got) != original {
		t.Errorf("snippet was damaged by a failed write:\nwant %q\ngot  %q", original, string(got))
	}
}

// Story 3: the same guarantee for the profile and its Static Sections.
func TestUpdateProfile_FailedWriteLeavesPreviousProfileIntact(t *testing.T) {
	dataDir := t.TempDir()
	const original = "name: Original Name\nemail: original@example.com\n"
	path := filepath.Join(dataDir, "profile.yaml")
	writeFile(t, path, original)
	makeReadOnly(t, dataDir)

	profile := masterdata.Profile{Name: "New Name", Email: "new@example.com"}
	if _, err := masterdata.UpdateProfile(dataDir, profile); err == nil {
		t.Fatal("expected an error when the write cannot complete, got nil")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading profile after failed write: %v", err)
	}
	if string(got) != original {
		t.Errorf("profile was damaged by a failed write:\nwant %q\ngot  %q", original, string(got))
	}
}

// Story 9/10: debris left by a crashed write is not an Entry, and does not
// take the whole list down with it.
func TestListEntries_IgnoresLeftoverTempFile(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "experience", "acme.md"), sampleEntry)
	writeFile(t, filepath.Join(dataDir, "experience", ".acme.md.123456.tmp"), "half a fi")
	writeFile(t, filepath.Join(dataDir, "projects", ".widget.md.987654.tmp"), "---\nnot: parseable")

	entries, err := masterdata.ListEntries(dataDir)
	if err != nil {
		t.Fatalf("unexpected error listing entries alongside leftover temp files: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].ID != "experience/acme" {
		t.Errorf("expected entry experience/acme, got %q", entries[0].ID)
	}
}

func TestListSnippets_IgnoresLeftoverTempFile(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "cover-letter-snippets", "opening.md"), "---\nkind: opening\n---\n\nBody.\n")
	writeFile(t, filepath.Join(dataDir, "cover-letter-snippets", ".opening.md.123456.tmp"), "---\nkind: op")

	snippets, err := masterdata.ListSnippets(dataDir)
	if err != nil {
		t.Fatalf("unexpected error listing snippets alongside a leftover temp file: %v", err)
	}
	if len(snippets) != 1 || snippets[0].ID != "opening" {
		t.Fatalf("expected only the opening snippet, got %+v", snippets)
	}
}

// Story 10: a leftover temp file for the same slug must not push a newly
// created Entry onto a suffixed slug.
func TestCreateEntry_SlugUnaffectedByLeftoverTempFile(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "experience", ".acme.md.123456.tmp"), "half a fi")

	created, err := masterdata.CreateEntry(dataDir, masterdata.Entry{
		Type:     masterdata.TypeExperience,
		Employer: "Acme",
		Role:     "Engineer",
		Start:    "2020",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "experience/acme" {
		t.Errorf("expected slug unaffected by debris (experience/acme), got %q", created.ID)
	}
}

// Story 11: records stay readable by the tailor-cv skill and the user's own
// editor — the permission mode is what it was before atomic writes.
func TestWrittenRecordsKeepTheirPermissionMode(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "experience", "acme.md"), sampleEntry)
	writeFile(t, filepath.Join(dataDir, "profile.yaml"), "name: N\nemail: n@example.com\n")

	if _, err := masterdata.UpdateEntry(dataDir, "experience/acme", masterdata.Entry{
		Type:     masterdata.TypeExperience,
		Employer: "Acme",
		Role:     "Engineer",
		Start:    "2020",
		Bullets:  []string{"one"},
	}); err != nil {
		t.Fatalf("unexpected error updating entry: %v", err)
	}
	created, err := masterdata.CreateSnippet(dataDir, masterdata.Snippet{Kind: "opening", Body: "Body."})
	if err != nil {
		t.Fatalf("unexpected error creating snippet: %v", err)
	}
	if _, err := masterdata.UpdateProfile(dataDir, masterdata.Profile{Name: "N", Email: "n@example.com"}); err != nil {
		t.Fatalf("unexpected error updating profile: %v", err)
	}

	for _, rel := range []string{
		filepath.Join("experience", "acme.md"),
		filepath.Join("cover-letter-snippets", created.ID+".md"),
		"profile.yaml",
	} {
		info, err := os.Stat(filepath.Join(dataDir, rel))
		if err != nil {
			t.Fatalf("stat %s: %v", rel, err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Errorf("%s: expected mode %v, got %v", rel, os.FileMode(0o644), info.Mode().Perm())
		}
	}
}

// Story 12: file contents are byte-for-byte what they were, so git diff on
// data/ stays as reviewable as ADR-0003 intended.
func TestUpdateEntry_ContentIsUnchangedByAtomicWrites(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "experience", "acme.md"), sampleEntry)

	entry := masterdata.Entry{
		Type:     masterdata.TypeExperience,
		Employer: "Acme",
		Role:     "Engineer",
		Start:    "2020",
		Tags:     []string{"go", "typst"},
		Bullets:  []string{"one", "two"},
	}
	if _, err := masterdata.UpdateEntry(dataDir, "experience/acme", entry); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const want = "---\nemployer: Acme\nrole: Engineer\nstart: \"2020\"\ntags:\n    - go\n    - typst\n---\n\n- one\n- two\n"
	got, err := os.ReadFile(filepath.Join(dataDir, "experience", "acme.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("rendered content changed:\nwant %q\ngot  %q", want, string(got))
	}
}
