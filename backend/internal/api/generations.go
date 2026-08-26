package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

var generationSlugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// generationFileContentTypes allowlists the files Render produces under
// output/<slug>/ that may be served/downloaded (story 10's PDF preview,
// story 11's downloads) — never an arbitrary path under projectRoot.
var generationFileContentTypes = map[string]string{
	"cv.pdf":           "application/pdf",
	"cover-letter.pdf": "application/pdf",
	"cover-letter.txt": "text/plain; charset=utf-8",
}

func getGenerationFileHandler(projectRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		file := r.PathValue("file")

		contentType, allowed := generationFileContentTypes[file]
		if !allowed || !generationSlugRe.MatchString(slug) {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(projectRoot, "output", slug, file)
		if _, err := os.Stat(path); err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, r, path)
	}
}

func renderGenerationHandler(dataDir, projectRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generation.RenderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
			return
		}

		result, err := generation.Render(projectRoot, dataDir, req)
		if errors.Is(err, generation.ErrInvalidRenderRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func createGenerationHandler(dataDir string, client generation.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generation.GenerateRequest
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
				return
			}
		}

		result, err := generation.Generate(r.Context(), dataDir, client, req)
		if errors.Is(err, generation.ErrInvalidSelection) {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
