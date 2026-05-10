package scanner

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
)

func DirSize(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			size += stat.Blocks * 512
		} else {
			size += info.Size()
		}
		return nil
	})
	return size
}

func DirSizeNonAPFS(path string) int64 {
	var size int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func ListFolders(basePath string) ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var folders []string
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			if shouldSkipFolder(name) {
				continue
			}
			folders = append(folders, filepath.Join(basePath, name))
		}
	}
	return folders, nil
}

func shouldSkipFolder(name string) bool {
	skip := map[string]bool{
		".":                 true,
		"..":                true,
		".git":              true,
		"node_modules":      true,
		"Caches":            false,
		"com.apple.*":       false,
		".DS_Store":         false,
	}
	return skip[name]
}

func scanDirWithContext(ctx context.Context, basePath string, maxDepth int) ([]string, error) {
	var results []string
	err := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, err := filepath.Rel(basePath, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		depth := len(splitPath(rel))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			results = append(results, path)
		}
		return nil
	})
	return results, err
}

func splitPath(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
