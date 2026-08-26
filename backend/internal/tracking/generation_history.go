package tracking

import (
	"os"
	"path/filepath"
)

// RecordGeneration appends record to the Application identified by id's
// Generation history (story 11), after the FE has rendered it via POST
// /api/generations/render. Always appends — regenerating (story 12) never
// overwrites a prior record (story 13).
func RecordGeneration(dataDir, id string, record GenerationRecord) (Application, error) {
	application, err := getApplication(dataDir, id)
	if err != nil {
		return Application{}, err
	}
	application.Generations = append(application.Generations, record)

	if err := os.WriteFile(filepath.Join(dataDir, applicationsDir, id+".md"), renderApplication(application), 0o644); err != nil {
		return Application{}, err
	}
	return application, nil
}
