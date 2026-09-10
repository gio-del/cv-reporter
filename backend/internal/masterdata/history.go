package masterdata

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// LastModified is the most recent commit's timestamp (RFC3339) and subject
// line for an Entry's file, or the zero value if the file has no commit
// history a git-log lookup can find (freshly created and not yet committed,
// or outside the repo the lookup ran against).
type LastModified struct {
	At      string `json:"at,omitempty"`
	Subject string `json:"subject,omitempty"`
}

// EntryLastModified looks up the last commit that touched an Entry's file,
// by shelling out to `git log` against projectRoot — mirroring the
// exec.Command shell-out precedent already used for `typst compile` in the
// render path (ADR-0012) rather than reading filesystem mtime, since mtime
// doesn't survive clone/checkout/restore the way git history does.
//
// Any reason the lookup can't find history (uncommitted file, dataDir
// outside projectRoot's repository, no git repository at all) is reported
// as the zero LastModified with a nil error, not a failure — that's a
// valid state for a freshly added Entry, not an error condition.
func EntryLastModified(projectRoot, dataDir, id string) (LastModified, error) {
	dir, slug, ok := splitID(id)
	if !ok {
		return LastModified{}, fmt.Errorf("invalid entry id %q", id)
	}
	if _, ok := entryDirs[dir]; !ok {
		return LastModified{}, fmt.Errorf("invalid entry id %q", id)
	}

	path := filepath.Join(dataDir, dir, slug+".md")

	cmd := exec.Command("git", "-C", projectRoot, "log", "-1", "--format=%aI%x00%s", "--", path)
	out, err := cmd.Output()
	if err != nil {
		// Not a repo, path outside the repo, or any other git failure: no
		// history to report, not an error the caller needs to handle.
		return LastModified{}, nil
	}

	line := strings.TrimRight(string(out), "\n")
	if line == "" {
		// Valid path inside a repo, but never committed.
		return LastModified{}, nil
	}

	parts := strings.SplitN(line, "\x00", 2)
	lm := LastModified{At: parts[0]}
	if len(parts) == 2 {
		lm.Subject = parts[1]
	}
	return lm, nil
}
