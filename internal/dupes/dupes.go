package dupes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cespare/xxhash/v2"
)

type File struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

type Group struct {
	SHA256    string `json:"sha256"`
	Size      int64  `json:"size"`
	Count     int    `json:"count"`
	TotalSize int64  `json:"total_size"`
	Files     []File `json:"files"`
}

var dupeDirs = []string{"Downloads", "Desktop", "Documents"}
var dupeDirsAggressive = []string{"Downloads", "Desktop", "Documents", "Pictures", "Movies", "Music"}

func Find(aggressive bool) ([]Group, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	dirs := dupeDirs
	if aggressive {
		dirs = dupeDirsAggressive
	}

	// Phase 1: xxhash fast pass
	fmt.Fprintf(os.Stderr, "Phase 1: Fast checksum (xxhash)...\n")
	hashGroups := make(map[uint64][]string)

	for _, d := range dirs {
		dir := filepath.Join(home, d)
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", dir, err)
			continue
		}
		if !info.IsDir() {
			continue
		}

		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			fi, err := d.Info()
			if err != nil {
				return nil
			}
			if fi.Size() == 0 {
				return nil
			}
			if fi.Size() < 1 {
				return nil
			}

			h := xxhash.New()
			f, err := os.Open(path)
			if err != nil {
				return nil
			}
			_, err = io.Copy(h, f)
			f.Close()
			if err != nil {
				return nil
			}
			sum := h.Sum64()
			hashGroups[sum] = append(hashGroups[sum], path)
			return nil
		})
	}

	// Phase 2: SHA-256 confirmation
	fmt.Fprintf(os.Stderr, "Phase 2: Confirming duplicates (SHA-256)...\n")
	var groups []Group

	for _, paths := range hashGroups {
		if len(paths) < 2 {
			continue
		}

		shaGroups := make(map[string][]string)
		for _, p := range paths {
			f, err := os.Open(p)
			if err != nil {
				continue
			}
			h := sha256.New()
			_, err = io.Copy(h, f)
			f.Close()
			if err != nil {
				continue
			}
			sum := hex.EncodeToString(h.Sum(nil))
			shaGroups[sum] = append(shaGroups[sum], p)
		}

		for sha, dupePaths := range shaGroups {
			if len(dupePaths) < 2 {
				continue
			}

			var files []File
			var totalSize int64
			for _, p := range dupePaths {
				fi, err := os.Stat(p)
				if err != nil {
					continue
				}
				files = append(files, File{
					Path:    p,
					Size:    fi.Size(),
					ModTime: fi.ModTime().Format("2006-01-02 15:04"),
				})
				totalSize += fi.Size()
			}

			if len(files) >= 2 {
				groups = append(groups, Group{
					SHA256:    sha,
					Size:      files[0].Size,
					Count:     len(files),
					TotalSize: totalSize,
					Files:     files,
				})
			}
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].TotalSize > groups[j].TotalSize
	})

	return groups, nil
}
