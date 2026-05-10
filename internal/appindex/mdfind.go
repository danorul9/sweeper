package appindex

import (
	"os"
	"path/filepath"
	"strings"
)

func ScanApplicationsDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var apps []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			apps = append(apps, filepath.Join(dir, e.Name()))
		}
	}
	return apps, nil
}

func ScanAllApplications() []string {
	var all []string
	dirs := []string{
		"/Applications",
		"/System/Applications",
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		dirs = append(dirs, filepath.Join(home, "Applications"))
		setapp := filepath.Join(home, "Applications", "Setapp")
		if _, err := os.Stat(setapp); err == nil {
			dirs = append(dirs, setapp)
		}
	}

	for _, dir := range dirs {
		apps, err := ScanApplicationsDir(dir)
		if err == nil {
			all = append(all, apps...)
		}
	}
	return all
}

func AppNameFromPath(appPath string) string {
	return strings.TrimSuffix(filepath.Base(appPath), ".app")
}

func ExecutableName(appPath string) string {
	name := AppNameFromPath(appPath)
	execPath := filepath.Join(appPath, "Contents", "MacOS", name)
	if _, err := os.Stat(execPath); err == nil {
		return name
	}
	entries, err := os.ReadDir(filepath.Join(appPath, "Contents", "MacOS"))
	if err != nil || len(entries) == 0 {
		return name
	}
	return entries[0].Name()
}
