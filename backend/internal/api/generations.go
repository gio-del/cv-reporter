package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gio-del/cv-reporter/backend/internal/generation"
)

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
