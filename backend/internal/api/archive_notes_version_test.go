package api_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

// Issue #89's guard, extended to the record-writing routes that landed after
// it was written: archive/unarchive rewrite the Job Listing file, and the
// Note routes rewrite the Application file, exactly as the routes #89 named
// do. The seam and the rules are the same as there: a real server over a
// temp data dir, the concurrent writer simulated by writing the file
// directly between requests, tokens only ever read from a response and sent
// back, and the header optional.

func newVersionServer(t *testing.T) (dataDir string, serverURL string) {
	t.Helper()
	dataDir = seedDataDir(t)
	server := httptest.NewServer(api.NewRouterWithGenerationClient(dataDir, &fakeGenerationClient{}))
	t.Cleanup(server.Close)
	return dataDir, server.URL
}

// archive POSTs to /api/job-listings/{id}/{action} with an optional If-Match.
func archive(t *testing.T, serverURL, id, action, version string) *http.Response {
	t.Helper()
	return doJSON(t, http.MethodPost, serverURL+"/api/job-listings/"+id+"/"+action, nil, version)
}

// jobListingTokenIn decodes an archive response, {jobListing}, and returns
// the Job Listing's token.
func jobListingTokenIn(t *testing.T, resp *http.Response) string {
	t.Helper()
	jobListing, _ := pairVersions(t, resp)
	return jobListing
}

func TestArchiveJobListing_WithCurrentVersion_SucceedsAndReturnsATokenForUnarchive(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	version, _ := listingVersions(t, url, id)

	resp := archive(t, url, id, "archive", version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	next := jobListingTokenIn(t, resp)
	if next == "" || next == version {
		t.Fatalf("expected a fresh Job Listing token in the archive response, got %q", next)
	}
	if !strings.Contains(readFile(t, filepath.Join(dataDir, "jobs", id+".md")), "archived: true") {
		t.Error("expected the archive to be written")
	}

	unarchive := archive(t, url, id, "unarchive", next)
	defer unarchive.Body.Close()
	if unarchive.StatusCode != http.StatusOK {
		t.Fatalf("expected the response token to unarchive with 200, got %d", unarchive.StatusCode)
	}
}

func TestArchiveJobListing_WithStaleVersion_Returns409AndLeavesTheFileByteIdentical(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	version, _ := listingVersions(t, url, id)

	path := filepath.Join(dataDir, "jobs", id+".md")
	changed := readFile(t, path) + "\nEdited by hand.\n"
	writeOutOfBand(t, path, changed)

	for _, action := range []string{"archive", "unarchive"} {
		resp := archive(t, url, id, action, version)
		resp.Body.Close()
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("%s: expected 409, got %d", action, resp.StatusCode)
		}
		if got := readFile(t, path); got != changed {
			t.Fatalf("%s: expected the Job Listing file byte-identical to the out-of-band edit, got:\n%s", action, got)
		}
	}
}

func TestArchiveJobListing_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")

	path := filepath.Join(dataDir, "jobs", id+".md")
	writeOutOfBand(t, path, readFile(t, path)+"\nEdited by hand.\n")

	resp := archive(t, url, id, "archive", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(readFile(t, path), "archived: true") {
		t.Error("expected the unconditional archive to be written")
	}
}

func TestArchiveJobListing_DeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	version, _ := listingVersions(t, url, id)
	removeFile(t, filepath.Join(dataDir, "jobs", id+".md"))

	resp := archive(t, url, id, "archive", version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// The Application token is a different file's: presenting it on an archive
// is a mismatch, not a pass.
func TestArchiveJobListing_WithTheApplicationsToken_Returns409(t *testing.T) {
	_, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	_, application := listingVersions(t, url, id)

	resp := archive(t, url, id, "archive", application)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

// noteRequest issues a Note route with an optional If-Match.
func noteRequest(t *testing.T, method, url string, body map[string]any, version string) *http.Response {
	t.Helper()
	return doJSON(t, method, url, body, version)
}

func notesURL(serverURL, id string) string {
	return serverURL + "/api/applications/" + id + "/notes"
}

func TestAddNote_WithCurrentVersion_Returns201AndAFreshToken(t *testing.T) {
	_, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	_, version := listingVersions(t, url, id)

	resp := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "Recruiter call."}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	next := decodeVersion(t, resp)
	if next == "" || next == version {
		t.Fatalf("expected a fresh Application token in the add response, got %q", next)
	}

	second := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "Second round booked."}, next)
	defer second.Body.Close()
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("expected the response token to add the next Note with 201, got %d", second.StatusCode)
	}
}

func TestAddNote_WithStaleVersion_Returns409AndLeavesTheFileByteIdentical(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	_, version := listingVersions(t, url, id)

	path := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, path, applicationOutOfBand)

	resp := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "Recruiter call."}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != applicationOutOfBand {
		t.Errorf("expected the Application file byte-identical to the out-of-band edit, got:\n%s", got)
	}
}

func TestAddNote_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	writeOutOfBand(t, filepath.Join(dataDir, "applications", id+".md"), applicationOutOfBand)

	resp := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "Recruiter call."}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

// Validation needs nothing from disk, so an empty body is still the 400 it
// always was, whatever the token.
func TestAddNote_EmptyBodyWithStaleVersion_StillReturns400(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	_, version := listingVersions(t, url, id)
	writeOutOfBand(t, filepath.Join(dataDir, "applications", id+".md"), applicationOutOfBand)

	resp := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "  "}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAddNote_ApplicationDeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	_, version := listingVersions(t, url, id)
	removeFile(t, filepath.Join(dataDir, "applications", id+".md"))

	resp := noteRequest(t, http.MethodPost, notesURL(url, id), map[string]any{"body": "Recruiter call."}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// noteWithToken adds one Note unconditionally and returns its id and the
// Application token the add answered with.
func noteWithToken(t *testing.T, serverURL, id string) (noteID, version string) {
	t.Helper()
	application := addNoteOK(t, serverURL, id, "Recruiter call.")
	notes := notesOf(t, application)
	if len(notes) != 1 {
		t.Fatalf("expected one Note, got %d", len(notes))
	}
	noteID, _ = notes[0]["id"].(string)
	version, _ = application["version"].(string)
	if noteID == "" || version == "" {
		t.Fatalf("expected a Note id and a version token, got (%q, %q)", noteID, version)
	}
	return noteID, version
}

func TestEditNote_WithCurrentVersion_SucceedsAndStaleVersionReturns409(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	noteID, version := noteWithToken(t, url, id)

	ok := noteRequest(t, http.MethodPatch, notesURL(url, id)+"/"+noteID, map[string]any{"body": "Recruiter call, went well."}, version)
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", ok.StatusCode)
	}
	if next := decodeVersion(t, ok); next == "" || next == version {
		t.Errorf("expected a fresh token in the edit response, got %q", next)
	}

	// The token the edit consumed is spent: the file moved on with that edit.
	path := filepath.Join(dataDir, "applications", id+".md")
	before := readFile(t, path)
	stale := noteRequest(t, http.MethodPatch, notesURL(url, id)+"/"+noteID, map[string]any{"body": "Overwritten."}, version)
	defer stale.Body.Close()
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for a spent token, got %d", stale.StatusCode)
	}
	if got := readFile(t, path); got != before {
		t.Errorf("expected the Application file byte-identical after a refused edit, got:\n%s", got)
	}
}

// A Note deleted elsewhere changed the file, so a stale edit of it hears
// "changed on disk" — the truthful reason — rather than a bare 404.
func TestEditNote_NoteRemovedOutOfBandWithStaleVersion_Returns409(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	noteID, version := noteWithToken(t, url, id)
	writeOutOfBand(t, filepath.Join(dataDir, "applications", id+".md"), applicationOutOfBand)

	resp := noteRequest(t, http.MethodPatch, notesURL(url, id)+"/"+noteID, map[string]any{"body": "Edited."}, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestEditNote_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	noteID, _ := noteWithToken(t, url, id)

	path := filepath.Join(dataDir, "applications", id+".md")
	writeOutOfBand(t, path, strings.Replace(readFile(t, path), "status: saved", "status: tailoring", 1))

	resp := noteRequest(t, http.MethodPatch, notesURL(url, id)+"/"+noteID, map[string]any{"body": "Edited."}, "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDeleteNote_WithStaleVersion_Returns409AndTheNoteSurvives(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	noteID, version := noteWithToken(t, url, id)

	path := filepath.Join(dataDir, "applications", id+".md")
	changed := strings.Replace(readFile(t, path), "status: saved", "status: tailoring", 1)
	writeOutOfBand(t, path, changed)

	resp := noteRequest(t, http.MethodDelete, notesURL(url, id)+"/"+noteID, nil, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != changed {
		t.Errorf("expected the Application file byte-identical after a refused delete, got:\n%s", got)
	}
}

func TestDeleteNote_WithCurrentVersion_RemovesTheNote(t *testing.T) {
	_, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")
	noteID, version := noteWithToken(t, url, id)

	resp := noteRequest(t, http.MethodDelete, notesURL(url, id)+"/"+noteID, nil, version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	application := decodeApplication(t, resp)
	if len(notesOf(t, application)) != 0 {
		t.Error("expected the Note to be removed")
	}
	if v, _ := application["version"].(string); v == "" || v == version {
		t.Errorf("expected a fresh token in the delete response, got %q", v)
	}
}

// applicationsViewVersions finds id in GET /api/applications and returns
// its row's two tokens.
func applicationsViewVersions(t *testing.T, serverURL, id string) (jobListing, application string) {
	t.Helper()
	status, body := getApplicationsView(t, serverURL, "")
	if status != http.StatusOK {
		t.Fatalf("expected 200 from the Applications view, got %d", status)
	}
	for _, g := range body.Groups {
		for _, item := range g.Items {
			if item["jobListing"].(map[string]any)["id"] != id {
				continue
			}
			jobListing, _ = item["jobListing"].(map[string]any)["version"].(string)
			application, _ = item["application"].(map[string]any)["version"].(string)
			return jobListing, application
		}
	}
	t.Fatalf("application %q not found in the Applications view", id)
	return "", ""
}

// The Applications page moves a row's Status from the grouped view, so its
// rows carry the same tokens as the Job Listings list and the row's
// Application token guards the move.
func TestListApplications_RowsCarryTokensThatGuardTheRowStatusMove(t *testing.T) {
	dataDir, url := newVersionServer(t)
	id := saveJobListing(t, url, "Acme Corp")

	jobListing, application := applicationsViewVersions(t, url, id)
	wantJobListing, wantApplication := listingVersions(t, url, id)
	if jobListing == "" || application == "" {
		t.Fatalf("expected both tokens on the Applications view row, got (%q, %q)", jobListing, application)
	}
	if jobListing != wantJobListing || application != wantApplication {
		t.Errorf("expected the row tokens to match the Job Listings list, got (%q, %q) want (%q, %q)", jobListing, application, wantJobListing, wantApplication)
	}

	path := filepath.Join(dataDir, "applications", id+".md")
	changed := strings.Replace(readFile(t, path), "status: saved", "status: tailoring", 1)
	writeOutOfBand(t, path, changed)

	stale := doJSON(t, http.MethodPatch, url+"/api/applications/"+id+"/status", map[string]any{"status": "tailoring"}, application)
	defer stale.Body.Close()
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for the row's stale token, got %d", stale.StatusCode)
	}
	if got := readFile(t, path); got != changed {
		t.Errorf("expected the Application file byte-identical after a refused move, got:\n%s", got)
	}

	_, fresh := applicationsViewVersions(t, url, id)
	ok := doJSON(t, http.MethodPatch, url+"/api/applications/"+id+"/status", map[string]any{"status": "sent"}, fresh)
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after re-reading the view, got %d", ok.StatusCode)
	}
}
