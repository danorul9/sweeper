package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/core"
)

type Snapshot struct {
	Timestamp time.Time      `json:"timestamp"`
	Items     []SnapshotItem `json:"items"`
}

type SnapshotItem struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func SaveSnapshot(items []core.Leftover) (*Snapshot, error) {
	supportDir, err := config.AppSupportDir()
	if err != nil {
		return nil, err
	}

	snapshotsDir := filepath.Join(supportDir, "snapshots")
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return nil, fmt.Errorf("create snapshots dir: %w", err)
	}

	snapshot := Snapshot{
		Timestamp: time.Now(),
	}

	for _, item := range items {
		snapshot.Items = append(snapshot.Items, SnapshotItem{
			Path: item.Path,
			Size: item.Size,
		})
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal snapshot: %w", err)
	}

	filename := snapshot.Timestamp.Format("20060102T150405") + ".json"
	path := filepath.Join(snapshotsDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("write snapshot: %w", err)
	}

	return &snapshot, nil
}

func LoadLatestSnapshot() (*Snapshot, error) {
	supportDir, err := config.AppSupportDir()
	if err != nil {
		return nil, err
	}

	snapshotsDir := filepath.Join(supportDir, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read snapshots dir: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	latest := entries[0].Name()
	for _, e := range entries {
		if e.Name() > latest {
			latest = e.Name()
		}
	}

	data, err := os.ReadFile(filepath.Join(snapshotsDir, latest))
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

func (s *Snapshot) Restore() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	trashDir := filepath.Join(home, ".Trash")

	for _, item := range s.Items {
		base := filepath.Base(item.Path)
		trashPath := filepath.Join(trashDir, base)

		_, err := os.Stat(trashPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("stat trashed %s: %w", base, err)
			}
			// try alternate paths: macOS appends (1), (2), etc. for conflicts
			found := false
			entries, _ := os.ReadDir(trashDir)
			for _, e := range entries {
				if matchesTrashName(e.Name(), base) {
					trashPath = filepath.Join(trashDir, e.Name())
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%s not found in Trash", base)
			}
		}

		parent := filepath.Dir(item.Path)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("create parent dir %s: %w", parent, err)
		}

		if err := os.Rename(trashPath, item.Path); err != nil {
			return fmt.Errorf("restore %s: %w", base, err)
		}
	}
	return nil
}

func matchesTrashName(entry, original string) bool {
	if entry == original {
		return true
	}
	ext := filepath.Ext(original)
	base := original[:len(original)-len(ext)]
	// match patterns like "foo (1).ext", "foo 2.ext"
	for i := 1; i <= 99; i++ {
		if entry == fmt.Sprintf("%s (%d)%s", base, i, ext) {
			return true
		}
		if entry == fmt.Sprintf("%s %d%s", base, i, ext) {
			return true
		}
	}
	return false
}
