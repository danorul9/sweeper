package large

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type File struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

var scanDirs = []string{"Downloads", "Desktop", "Documents", "Movies"}

func Scan(threshold int64) ([]File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	var files []File

	for _, d := range scanDirs {
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

		filepath.WalkDir(dir, func(path string, de os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if de.IsDir() {
				return nil
			}
			fi, err := de.Info()
			if err != nil {
				return nil
			}
			if fi.Size() >= threshold {
				files = append(files, File{
					Path:    path,
					Size:    fi.Size(),
					ModTime: fi.ModTime().Format("2006-01-02 15:04"),
				})
			}
			return nil
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})

	return files, nil
}
