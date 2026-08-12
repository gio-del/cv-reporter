package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func TestGetProfile_ReturnsContactInfoAndStaticSections(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/master-data/profile")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var profile map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatal(err)
	}

	if profile["name"] != "Test User" {
		t.Errorf("expected name Test User, got %v", profile["name"])
	}
	if profile["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", profile["email"])
	}

	education, ok := profile["education"].([]any)
	if !ok || len(education) != 1 {
		t.Fatalf("expected 1 education entry, got %v", profile["education"])
	}
	edu := education[0].(map[string]any)
	if edu["institution"] != "Test University" {
		t.Errorf("expected institution Test University, got %v", edu["institution"])
	}

	languages, ok := profile["languages"].([]any)
	if !ok || len(languages) != 2 {
		t.Fatalf("expected 2 languages, got %v", profile["languages"])
	}
}

func TestUpdateProfile_ValidPayload_WritesFileAndReturns200(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	payload := map[string]any{
		"name":     "Updated User",
		"location": "Rome, Italy",
		"email":    "updated@example.com",
		"phone":    "+39 333 111 1111",
		"linkedin": "updateduser",
		"github":   "updateduser",
		"education": []map[string]any{
			{
				"degree":      "MSc",
				"institution": "Test University",
				"program":     "Computer Science",
				"start":       "2022",
				"end":         "2024",
				"grade":       "110/110",
				"courses":     []string{"Distributed Systems", "Machine Learning"},
			},
		},
		"publications": []map[string]any{},
		"awards":       []map[string]any{},
		"activities":   []map[string]any{},
		"languages": []map[string]any{
			{"name": "Italian", "level": "Native"},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/master-data/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
		t.Fatalf("expected 200, got %d; file on disk:\n%s", resp.StatusCode, respBody)
	}

	updated, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte("Updated User")) {
		t.Errorf("expected file to contain updated name, got:\n%s", updated)
	}
	if !bytes.Contains(updated, []byte("Machine Learning")) {
		t.Errorf("expected file to contain updated course, got:\n%s", updated)
	}
	if bytes.Contains(updated, []byte("English")) {
		t.Errorf("expected removed language to be gone, got:\n%s", updated)
	}
}

func TestUpdateProfile_MissingRequiredField_Returns400AndFileUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	original, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"name":  "",
		"email": "test@example.com",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/master-data/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	after, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Errorf("expected file to be untouched after invalid update, got:\n%s", after)
	}
}

func TestUpdateProfile_InvalidEmail_Returns400AndFileUnchanged(t *testing.T) {
	dataDir := seedDataDir(t)
	server := httptest.NewServer(api.NewRouter(dataDir))
	defer server.Close()

	original, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"name":  "Test User",
		"email": "not-an-email",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/master-data/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	after, err := os.ReadFile(filepath.Join(dataDir, "profile.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Errorf("expected file to be untouched after invalid update, got:\n%s", after)
	}
}
