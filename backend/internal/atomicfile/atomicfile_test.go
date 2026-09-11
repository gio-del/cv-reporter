package atomicfile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/atomicfile"
)

// makeReadOnly makes dir unwritable so that creating the helper's temp file
// fails deterministically — the only portable way to force a write failure
// with nothing but the standard library (no filesystem abstraction, see the
// PRD's Testing Decisions). Permissions are restored at test end so
// t.TempDir's own cleanup can remove the directory.
func makeReadOnly(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}

func TestWriteFile_CreatesFileWithContentAndMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.md")

	if err := atomicfile.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("content: expected %q, got %q", "hello\n", string(got))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Errorf("mode: expected %v, got %v", os.FileMode(0o644), info.Mode().Perm())
	}
}

func TestWriteFile_ReplacesExistingFileAndLeavesNoOtherFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.md")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := atomicfile.WriteFile(path, []byte("new\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != "new\n" {
		t.Errorf("content: expected %q, got %q", "new\n", string(got))
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != "record.md" {
		var names []string
		for _, f := range files {
			names = append(names, f.Name())
		}
		t.Errorf("expected only record.md in the directory, got %v", names)
	}
}

func TestWriteFile_FailedWriteLeavesExistingFileByteIdentical(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.md")
	original := []byte("---\nemployer: Foo\n---\n\n- a bullet\n")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, dir)

	if err := atomicfile.WriteFile(path, []byte("replacement\n"), 0o644); err == nil {
		t.Fatal("expected an error writing into a read-only directory, got nil")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading pre-existing file: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("pre-existing file changed: expected %q, got %q", string(original), string(got))
	}
}

func TestWriteFile_FailedWriteLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.md")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	makeReadOnly(t, dir)

	if err := atomicfile.WriteFile(path, []byte("replacement\n"), 0o644); err == nil {
		t.Fatal("expected an error writing into a read-only directory, got nil")
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name() != "record.md" {
		var names []string
		for _, f := range files {
			names = append(names, f.Name())
		}
		t.Errorf("expected the directory unchanged (only record.md), got %v", names)
	}
}

func TestWriteFile_ResultIsByteIdenticalToPlainWrite(t *testing.T) {
	dir := t.TempDir()
	content := []byte("---\nkind: opening\ntags:\n    - go\n---\n\nSome body text.\n")

	atomicPath := filepath.Join(dir, "atomic.md")
	plainPath := filepath.Join(dir, "plain.md")

	if err := atomicfile.WriteFile(atomicPath, content, 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(plainPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	atomicBytes, err := os.ReadFile(atomicPath)
	if err != nil {
		t.Fatal(err)
	}
	plainBytes, err := os.ReadFile(plainPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(atomicBytes, plainBytes) {
		t.Errorf("content differs from a plain write: %q vs %q", string(atomicBytes), string(plainBytes))
	}
}
