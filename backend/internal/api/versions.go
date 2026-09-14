package api

import (
	"net/http"
	"strings"
)

// versionHeader carries the version token a conditional write is checked
// against. It is deliberately optional: the tailor-cv skill writes files
// directly and will never present one, and non-FE API callers (the browser
// extension's capture route, the skill-parity work in issue #54) must keep
// working unchanged, so a request without it is an unconditional write.
// Requiring it (428 Precondition Required) would break those callers on
// day one; the guarantee is enforced in the FE, which sends it on every
// write path.
const versionHeader = "If-Match"

// applicationVersionHeader carries the Application's token on the Job
// Listing delete, alongside the Job Listing's own token in If-Match. The
// delete removes two files sharing one id, each with its own token, and
// If-Match's list form means "any of these matches" — the wrong semantics
// for "both must still match" — so the second token gets its own header.
// Like If-Match it is optional (issue #89, story 19).
const applicationVersionHeader = "Application-If-Match"

// requestVersion reads the version token off r, empty when the caller sent
// none.
func requestVersion(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(versionHeader))
}

// writeConflict refuses a write whose version token no longer matches the
// file on disk. 409 (rather than 412, the more literal reading of a failed
// If-Match) keeps this consistent with how the rest of this API signals a
// refused-because-of-state write, and gives the FE one branch to check.
// The body names the record and says what happened in the words the user
// needs — "precondition failed" would tell them nothing.
func writeConflict(w http.ResponseWriter, record string) {
	http.Error(w, "this "+record+" changed on disk since it was read; reload to see the current version", http.StatusConflict)
}
