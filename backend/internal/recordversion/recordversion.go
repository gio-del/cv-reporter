// Package recordversion holds the one helper every conditional write
// under the data directory checks itself against.
//
// The problem it solves is the lost update, not the partial one: the web
// app and the tailor-cv skill are two front doors onto the same files
// (ADR-0004), so a record the app read minutes ago may have been rewritten
// underneath it. An unconditional read-modify-write silently discards that
// other writer's change. A conditional one compares the bytes on disk
// against the version the caller saw and refuses with [ErrMismatch] when
// they differ. (Making the write itself indivisible is a different problem
// with a different mechanism — see internal/atomicfile and ADR-0018.)
//
// The token is a hash of the record file's raw bytes. Not the filesystem
// mtime: the Entry last-modified lookup already shells out to git log
// precisely because mtime does not survive a clone, checkout or restore,
// and a token with that weakness would raise phantom conflicts on
// untouched content. Not the git-log value either: it only moves when a
// commit happens, so the single most important case — the skill rewriting
// a record while the app is open — would be invisible. A content hash
// changes if and only if the bytes change, with no clock and no VCS
// involved.
//
// Hashing raw bytes rather than the parsed record means a
// semantically-neutral reformat (a writer reordering YAML keys, say)
// counts as a conflict. That is deliberate: a spurious conflict costs one
// reload, a missed one costs data.
//
// The token travels asymmetrically, and the asymmetry is intentional: it
// is returned as a read-time field in the JSON body (like the Entry's
// last-modified or the Application's staleness flag — computed on read,
// never persisted), but sent back in an If-Match request header. Keeping
// it out of write payloads means request bodies keep exactly the shape
// they have now, which matters because the Entry write route decodes its
// body straight into the Entry type.
package recordversion

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
)

// ErrMismatch marks a conditional write whose token no longer matches the
// file on disk: something else wrote the record since the caller read it.
// The API layer maps it to 409 Conflict.
var ErrMismatch = errors.New("record changed on disk since it was read")

// Of returns the version token for the file at path, or the os.ReadFile
// error (notably an os.ErrNotExist a caller can map to 404).
func Of(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return OfBytes(content), nil
}

// OfBytes returns the version token for content already in hand.
func OfBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// Check reports whether a write carrying token may proceed against the
// file at path. An empty token means the caller sent no version at all —
// the skill, the browser extension, any non-FE API client — and the write
// is unconditional, so Check returns nil without reading the file. A
// non-empty token that no longer matches yields ErrMismatch; a missing
// file yields os.ErrNotExist, so "this record is gone" always outranks
// "this record changed".
func Check(path, token string) error {
	if token == "" {
		return nil
	}
	current, err := Of(path)
	if err != nil {
		return err
	}
	if current != token {
		return ErrMismatch
	}
	return nil
}
