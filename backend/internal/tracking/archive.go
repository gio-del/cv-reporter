package tracking

import (
	"path/filepath"

	"github.com/gio-del/cv-reporter/backend/internal/atomicfile"
)

// SetArchived archives (archived true) or unarchives (archived false) the
// Job Listing identified by id and returns the updated Job Listing (issue
// #98). It is idempotent: setting the flag to the value it already holds
// succeeds without rewriting the file. Only the Job Listing file is ever
// written; the Application, its Status history and its Generation history
// are left exactly as they were. A missing Job Listing surfaces as
// os.ErrNotExist, as GetJobListing does.
func SetArchived(dataDir, id string, archived bool) (JobListing, error) {
	listing, err := getJobListing(dataDir, id)
	if err != nil {
		return JobListing{}, err
	}
	if listing.Archived == archived {
		return listing, nil
	}

	listing.Archived = archived
	if err := atomicfile.WriteFile(filepath.Join(dataDir, jobsDir, id+".md"), renderJobListing(listing), 0o644); err != nil {
		return JobListing{}, err
	}
	return listing, nil
}
