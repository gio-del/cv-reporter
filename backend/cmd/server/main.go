package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/api"
	"github.com/gio-del/cv-reporter/backend/internal/claude"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = "."
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := api.NewRouterFull(dataDir, projectRoot, claude.New())

	addr := "0.0.0.0:" + port
	log.Printf("cv-reporter backend listening on %s (data dir: %s, project root: %s)", addr, dataDir, projectRoot)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
