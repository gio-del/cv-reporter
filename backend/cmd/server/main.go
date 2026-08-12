package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gio-del/cv-reporter/backend/internal/api"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := api.NewRouter(dataDir)

	addr := "0.0.0.0:" + port
	log.Printf("cv-reporter backend listening on %s (data dir: %s)", addr, dataDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
