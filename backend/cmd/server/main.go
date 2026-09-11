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
	// LAN_AUTH_TOKEN being set is what makes LAN-reachable mode "active" from
	// the backend's point of view — the bind-address switch itself
	// (BIND_ADDR) lives entirely in docker-compose.yml's port mapping, which
	// the process inside the container can't observe (see issue #57).
	lanAuthToken := os.Getenv("LAN_AUTH_TOKEN")

	srv := api.NewServer(api.RouterConfig{
		Addr:             "0.0.0.0:" + port,
		DataDir:          dataDir,
		ProjectRoot:      projectRoot,
		GenerationClient: claude.New(),
		ATSHTTPDoer:      http.DefaultClient,
		LANAuthToken:     lanAuthToken,
	})

	if lanAuthToken != "" {
		log.Printf("LAN-reachable mode: binding %s, auth required", srv.Addr)
	}
	log.Printf("cv-reporter backend listening on %s (data dir: %s, project root: %s)", srv.Addr, dataDir, projectRoot)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
