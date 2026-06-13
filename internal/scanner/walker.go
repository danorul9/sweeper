package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func DirSize(path string) int64 {
	// Use `du -sk` which is much faster than walking recursively in Go
	// and avoids OOM on large cache directories with millions of files.
	// macOS du doesn't support -b, so we use -sk (kilobytes) and multiply.
	cmd := exec.Command("du", "-sk", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "\t", 2)
	if len(parts) < 1 {
		return 0
	}
	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return size * 1024
}

func DirSizeNonAPFS(path string) int64 {
	return DirSize(path)
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

func ListHiddenFolders(basePath string) ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var folders []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, ".") {
			continue
		}
		if shouldSkipHiddenFolder(name) {
			continue
		}
		if e.IsDir() {
			folders = append(folders, filepath.Join(basePath, name))
		}
	}
	return folders, nil
}

func shouldSkipHiddenFolder(name string) bool {
	skip := map[string]bool{
		".":                    true,
		"..":                   true,
		".Trash":               true,
		".Spotlight-V100":      true,
		".DocumentRevisions-V100": true,
		".fseventsd":           true,
		".PKInstallSandboxManager": true,
		".localized":           true,
		".DS_Store":            true,
		".file":                true,
		".ssh":                 true,
		".gnupg":               true,
		".aws":                 true,
		".cache":               true,
	}
	return skip[name]
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
