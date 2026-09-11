// Package atomicfile holds the one helper every record under the data
// directory is written with.
//
// The rule: every write under the data directory goes through
// [WriteFile]; writes under the output directory stay on os.WriteFile.
// That line matches the existing source-versus-derived split (ADR-0012,
// and CLAUDE.md's "output/ is derived, not Master Data") — losing a
// rendered PDF means re-running a Render, losing a record means losing
// history that cannot be rebuilt (ADR-0003, ADR-0008: these files are the
// database).
//
// A plain whole-file write truncates the target before refilling it, so a
// process killed mid-write — or a full disk — destroys the record rather
// than leaving the previous version. [WriteFile] instead writes a temp
// file in the same directory and renames it over the target, so a reader
// sees either the complete previous version or the complete new one.
//
// This addresses partial writes only. Two writers racing to update the
// same record (lost updates) is a different failure with a different
// mechanism, and is deliberately not handled here.
package atomicfile

import (
	"os"
	"path/filepath"
)

// WriteFile writes data to the file named by name, creating it if it does
// not exist and replacing its contents if it does, with permissions perm.
// It mirrors os.WriteFile's signature so converting a call site is a
// one-identifier change.
//
// The write is atomic with respect to readers: data lands in a temp file in
// name's directory, which is flushed and then renamed over name. Any
// failure before the rename removes the temp file and leaves the directory
// exactly as it was, with any pre-existing name byte-identical. The temp
// file is created in the same directory because rename is only atomic
// within a filesystem, and the data directory is a bind-mounted volume in
// the container.
//
// Not covered: durability against sudden power loss. The containing
// directory is deliberately not synced — the failure modes this guards
// against (process killed, container stopped, disk full) are all covered by
// flush-then-rename.
func WriteFile(name string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(name)

	f, err := os.CreateTemp(dir, tempPattern(filepath.Base(name)))
	if err != nil {
		return err
	}
	tmp := f.Name()

	// Any exit before the successful rename leaves nothing behind.
	cleanup := func(err error) error {
		f.Close()
		os.Remove(tmp)
		return err
	}

	if _, err := f.Write(data); err != nil {
		return cleanup(err)
	}
	// CreateTemp makes the file owner-only; today's writes produce files
	// the skill and the user's editor read, so the requested mode is set
	// explicitly rather than left to the temp file's default.
	if err := f.Chmod(perm); err != nil {
		return cleanup(err)
	}
	// Flush before the rename so a write error (notably a full disk)
	// surfaces as a failed write, instead of being discovered after the
	// rename has already published a short file.
	if err := f.Sync(); err != nil {
		return cleanup(err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, name); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// tempPattern builds the os.CreateTemp pattern for a target file named
// base. The name is dot-prefixed and carries a ".tmp" marker after the
// real extension, so a temp file stranded by a crash is never picked up by
// the list paths (which enumerate records by the ".md" suffix and fail the
// whole collection on one unparseable file) nor by the slug-uniqueness
// probe (which checks for "<slug>.md"). That naming — not an opportunistic
// sweep — is what makes stranded temp files harmless.
func tempPattern(base string) string {
	return "." + base + ".*.tmp"
}
