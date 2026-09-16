package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/sumisura/backend/internal/api"
)

// The version token is opaque to these tests exactly as it is to the FE:
// it is read from a response and sent back, never constructed by hand
// except to produce a deliberately wrong value (issue #89's Testing
// Decisions).

const entryID = "experience/example-client-a"

func entryPayload(role string) map[string]any {
	return map[string]any{
		"employer": "Example Consulting S.p.A.",
		"role":     role,
		"client":   "Example Client A",
		"location": "Example City",
		"start":    "2024-10",
		"end":      nil,
		"flagship": true,
		"tags":     []string{"AI Platform", "React"},
		"bullets":  []string{"Designed and built an AI Platform."},
	}
}

// doJSON issues a request carrying payload as JSON, plus an If-Match
// header when version is non-empty — the FE's exact shape.
func doJSON(t *testing.T, method, url string, payload any, version string) *http.Response {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if version != "" {
		req.Header.Set("If-Match", version)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// decodeVersion reads the opaque version token off a record response.
func decodeVersion(t *testing.T, resp *http.Response) string {
	t.Helper()
	var record map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		t.Fatal(err)
	}
	version, _ := record["version"].(string)
	return version
}

// getVersion reads url and returns the version token of the record it
// serves.
func getVersion(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reading %s, got %d", url, resp.StatusCode)
	}
	version := decodeVersion(t, resp)
	if version == "" {
		t.Fatalf("expected a non-empty version token from %s", url)
	}
	return version
}

// writeOutOfBand simulates the tailor-cv skill (or a second tab, or a hand
// edit) rewriting a record file directly while the app holds a stale copy
// — a synchronous file write between two requests, never a sleep.
func writeOutOfBand(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func removeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestGetEntry_ReturnsNonEmptyVersionToken(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	getVersion(t, server.URL+"/api/master-data/entries/"+entryID)
}

func TestListEntries_ReturnsVersionTokenPerEntry(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/entries")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var entries []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected seeded entries")
	}
	for _, entry := range entries {
		if version, _ := entry["version"].(string); version == "" {
			t.Errorf("expected a version token on entry %v", entry["id"])
		}
	}
}

func TestUpdateEntry_WithCurrentVersion_Succeeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	version := getVersion(t, url)

	resp := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if !bytes.Contains([]byte(readFile(t, filepath.Join(dataDir, "experience", "example-client-a.md"))), []byte("Staff Engineer")) {
		t.Error("expected the write to reach the file")
	}
}

func TestUpdateEntry_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	path := filepath.Join(dataDir, "experience", "example-client-a.md")
	version := getVersion(t, url)

	outOfBand := `---
employer: Example Consulting S.p.A.
role: Data Engineer
client: Example Client A
location: Example City
start: "2024-10"
end: null
flagship: true
tags:
  - AI Platform
  - Kafka
---

- A bullet the skill added.
`
	writeOutOfBand(t, path, outOfBand)

	resp := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	if got := readFile(t, path); got != outOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

func TestUpdateEntry_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	path := filepath.Join(dataDir, "experience", "example-client-a.md")

	writeOutOfBand(t, path, `---
employer: Example Consulting S.p.A.
start: "2024-10"
---

- A bullet the skill added.
`)

	// No If-Match header: the tailor-cv skill and the browser extension
	// never send one, and must keep working (issue #89, story 32).
	resp := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !bytes.Contains([]byte(readFile(t, path)), []byte("Staff Engineer")) {
		t.Error("expected the unconditional write to reach the file")
	}
}

func TestUpdateEntry_WithGarbageVersion_Returns409(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	resp := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), "not-a-real-token")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestUpdateEntry_WithAnotherRecordsVersion_Returns409(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	other := getVersion(t, server.URL+"/api/master-data/entries/projects/emall")

	resp := doJSON(t, http.MethodPut, server.URL+"/api/master-data/entries/"+entryID, entryPayload("Staff Engineer"), other)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestUpdateEntry_RecordDeletedOutOfBand_Returns404NotConflict(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	version := getVersion(t, url)

	if err := os.Remove(filepath.Join(dataDir, "experience", "example-client-a.md")); err != nil {
		t.Fatal(err)
	}

	resp := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateEntry_ReusingAVersion_SecondWriteReturns409(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	version := getVersion(t, url)

	first := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), version)
	defer first.Body.Close()
	if first.StatusCode != http.StatusOK {
		t.Fatalf("expected the first write to succeed, got %d", first.StatusCode)
	}

	second := doJSON(t, http.MethodPut, url, entryPayload("Principal Engineer"), version)
	defer second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("expected the second write to be refused with 409, got %d", second.StatusCode)
	}
}

func TestUpdateEntry_ResponseVersionIsUsableForTheNextWrite(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/entries/" + entryID
	version := getVersion(t, url)

	first := doJSON(t, http.MethodPut, url, entryPayload("Staff Engineer"), version)
	defer first.Body.Close()
	next := decodeVersion(t, first)
	if next == "" {
		t.Fatal("expected the write response to carry a fresh version token")
	}
	if next == version {
		t.Error("expected the version token to change after a successful write")
	}

	second := doJSON(t, http.MethodPut, url, entryPayload("Principal Engineer"), next)
	defer second.Body.Close()
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected the response's token to work without a re-read, got %d", second.StatusCode)
	}
}
