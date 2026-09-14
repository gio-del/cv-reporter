package api_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

// The frontend declares these list fields as arrays (frontend/src/api/types.ts)
// and maps over them unguarded, so an empty one must reach it as [] rather
// than null — which is what encoding/json sends for a nil slice. Each case
// is a record that legitimately has none: a Snippet or Entry written without
// tags, a profile.yaml without Static Sections, no Applications to compute
// stats from (issue #99).

func decodeObject(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("decoding %s: %v", body, err)
	}
	return object
}

func assertEmptyArray(t *testing.T, object map[string]json.RawMessage, field string) {
	t.Helper()
	if got := string(object[field]); got != "[]" {
		t.Errorf("expected %s to be sent as [], got %s", field, got)
	}
}

func TestGetSnippet_WithoutTags_SendsEmptyTagsArray(t *testing.T) {
	server := newSimpleServer(t, seedSnippetsDataDir(t))

	body := call(t, http.MethodGet, server.URL+"/api/master-data/cover-letter-snippets/closing-standard", nil, http.StatusOK)

	assertEmptyArray(t, decodeObject(t, body), "tags")
}

func TestListSnippets_WithoutTags_SendsEmptyTagsArray(t *testing.T) {
	server := newSimpleServer(t, seedSnippetsDataDir(t))

	body := call(t, http.MethodGet, server.URL+"/api/master-data/cover-letter-snippets", nil, http.StatusOK)

	var snippets []map[string]json.RawMessage
	if err := json.Unmarshal(body, &snippets); err != nil {
		t.Fatal(err)
	}
	for _, snippet := range snippets {
		if string(snippet["id"]) == `"closing-standard"` {
			assertEmptyArray(t, snippet, "tags")
			return
		}
	}
	t.Fatalf("closing-standard not listed in %s", body)
}

func TestGetEntry_WithoutTags_SendsEmptyTagsArray(t *testing.T) {
	dataDir := seedDataDir(t)
	writeFile(t, filepath.Join(dataDir, "experience", "untagged.md"), "---\nemployer: Untagged Co\nrole: Engineer\nstart: \"2020-01\"\nend: null\n---\n")
	server := newSimpleServer(t, dataDir)

	body := call(t, http.MethodGet, server.URL+"/api/master-data/entries/experience/untagged", nil, http.StatusOK)

	assertEmptyArray(t, decodeObject(t, body), "tags")
}

func TestGetProfile_WithoutStaticSections_SendsEmptyArrays(t *testing.T) {
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "profile.yaml"), "name: Test User\nlocation: Milan\nemail: test@example.com\nphone: \"1\"\nlinkedin: t\ngithub: t\n")
	server := newSimpleServer(t, dataDir)

	body := call(t, http.MethodGet, server.URL+"/api/master-data/profile", nil, http.StatusOK)

	profile := decodeObject(t, body)
	for _, section := range []string{"education", "publications", "awards", "activities", "languages"} {
		assertEmptyArray(t, profile, section)
	}
}

func TestGetApplicationsStats_NoApplications_SendsEmptyArrays(t *testing.T) {
	server := newSimpleServer(t, seedDataDir(t))

	body := call(t, http.MethodGet, server.URL+"/api/applications/stats", nil, http.StatusOK)

	stats := decodeObject(t, body)
	assertEmptyArray(t, stats, "conversions")
	assertEmptyArray(t, stats, "timeInStage")
}
