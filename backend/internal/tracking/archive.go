package tracking

import (
	"path/filepath"

	"github.com/gio-del/sumisura/backend/internal/atomicfile"
	"github.com/gio-del/sumisura/backend/internal/recordversion"
)

// SetArchived archives (archived true) or unarchives (archived false) the
// Job Listing identified by id and returns the updated Job Listing (issue
// #98). It is idempotent: setting the flag to the value it already holds
// succeeds without rewriting the file. Only the Job Listing file is ever
// written; the Application, its Status history and its Generation history
// are left exactly as they were. A missing Job Listing surfaces as
// os.ErrNotExist, as GetJobListing does.
func SetArchived(dataDir, id string, archived bool) (JobListing, error) {
	return SetArchivedIfMatch(dataDir, id, archived, "")
}

// SetArchivedIfMatch is SetArchived, refusing with recordversion.ErrMismatch
// when version no longer matches the Job Listing file on disk. Archiving
// rewrites the whole Job Listing file just as the delete removes it, so a
// Job Listing changed elsewhere since the caller read it is not rewritten
// from a stale copy (issue #89, extended to the archive routes). The check
// runs before the idempotent no-op too, so a stale caller always hears the
// truthful reason rather than a success for a record it is not seeing. An
// empty version writes unconditionally.
func SetArchivedIfMatch(dataDir, id string, archived bool, version string) (JobListing, error) {
	path := filepath.Join(dataDir, jobsDir, id+".md")
	listing, err := getJobListing(dataDir, id)
	if err != nil {
		return JobListing{}, err
	}
	if err := recordversion.Check(path, version); err != nil {
		return JobListing{}, err
	}
	if listing.Archived == archived {
		return listing, nil
	}

	listing.Archived = archived
	if err := atomicfile.WriteFile(path, renderJobListing(listing), 0o644); err != nil {
		return JobListing{}, err
	}
	return listing, nil
}
