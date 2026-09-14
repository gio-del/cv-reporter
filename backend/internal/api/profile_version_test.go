package api_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func profilePayload(name string) map[string]any {
	return map[string]any{
		"name":     name,
		"location": "Milan, Italy",
		"email":    "test@example.com",
		"phone":    "+39 333 000 0000",
		"linkedin": "testuser",
		"github":   "testuser",
	}
}

const profileOutOfBand = `name: Edited By Hand
location: Milan, Italy
email: test@example.com
phone: "+39 333 000 0000"
linkedin: testuser
github: testuser
`

func TestGetProfile_ReturnsNonEmptyVersionToken(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	getVersion(t, server.URL+"/api/master-data/profile")
}

func TestUpdateProfile_WithCurrentVersion_Succeeds(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/profile"
	version := getVersion(t, url)

	resp := doJSON(t, http.MethodPut, url, profilePayload("Renamed User"), version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(readFile(t, filepath.Join(dataDir, "profile.yaml")), "Renamed User") {
		t.Error("expected the write to reach the file")
	}
}

func TestUpdateProfile_WithStaleVersion_Returns409AndLeavesFileUntouched(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/profile"
	path := filepath.Join(dataDir, "profile.yaml")
	version := getVersion(t, url)

	writeOutOfBand(t, path, profileOutOfBand)

	resp := doJSON(t, http.MethodPut, url, profilePayload("Renamed User"), version)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
	if got := readFile(t, path); got != profileOutOfBand {
		t.Errorf("expected the file to be byte-identical to its pre-request state, got:\n%s", got)
	}
}

func TestUpdateProfile_WithNoVersion_IsUnconditional(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	writeOutOfBand(t, filepath.Join(dataDir, "profile.yaml"), profileOutOfBand)

	resp := doJSON(t, http.MethodPut, server.URL+"/api/master-data/profile", profilePayload("Renamed User"), "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUpdateProfile_VersionIsNeverPersistedToTheFile(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(api.RouterConfig{DataDir: dataDir}))
	defer server.Close()

	url := server.URL + "/api/master-data/profile"
	version := getVersion(t, url)

	resp := doJSON(t, http.MethodPut, url, profilePayload("Renamed User"), version)
	defer resp.Body.Close()
	if next := decodeVersion(t, resp); next == "" || next == version {
		t.Errorf("expected a fresh version token in the write response, got %q", next)
	}
	if got := readFile(t, filepath.Join(dataDir, "profile.yaml")); strings.Contains(got, "version") {
		t.Errorf("expected the version token never to reach the file, got:\n%s", got)
	}
}
