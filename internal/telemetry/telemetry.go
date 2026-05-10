package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/danorul9/sweeper/internal/config"
)

type Submission struct {
	Folder   string   `json:"folder"`
	BundleID string   `json:"bundle_id,omitempty"`
	Location string   `json:"location"`
	Size     int64    `json:"size"`
	Signals  []string `json:"signals,omitempty"`
	HostHash string   `json:"host_hash,omitempty"`
	Time     string   `json:"time"`
}

type submissionFile struct {
	Version     int          `json:"version"`
	Created     string       `json:"created"`
	Updated     string       `json:"updated"`
	Submissions []Submission `json:"submissions"`
}

var mu sync.Mutex

func Record(sub Submission) error {
	mu.Lock()
	defer mu.Unlock()

	supportDir, err := config.AppSupportDir()
	if err != nil {
		return err
	}

	path := filepath.Join(supportDir, "telemetry.json")

	var sf submissionFile
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &sf)
	}

	if sf.Version == 0 {
		sf.Version = 1
		sf.Created = time.Now().UTC().Format(time.RFC3339)
	}
	sf.Updated = time.Now().UTC().Format(time.RFC3339)
	sub.Time = time.Now().UTC().Format(time.RFC3339)
	sf.Submissions = append(sf.Submissions, sub)

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Count() (int, error) {
	mu.Lock()
	defer mu.Unlock()

	supportDir, err := config.AppSupportDir()
	if err != nil {
		return 0, err
	}

	path := filepath.Join(supportDir, "telemetry.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var sf submissionFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return 0, nil
	}

	return len(sf.Submissions), nil
}
