package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"howett.net/plist"
)

type Issue struct {
	Category    string `json:"category"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

func Run() ([]Issue, error) {
	var issues []Issue
	issues = append(issues, checkLaunchAgents()...)
	issues = append(issues, checkDeadSymlinks()...)
	issues = append(issues, checkXcodeDerivedData()...)
	issues = append(issues, checkIOSBackups()...)
	return issues, nil
}

type plistAgent struct {
	Program         string   `plist:"Program"`
	ProgramArgs     []string `plist:"ProgramArguments"`
	Label           string   `plist:"Label"`
	KeepAlive       bool     `plist:"KeepAlive"`
	RunAtLoad       bool     `plist:"RunAtLoad"`
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func checkLaunchAgents() []Issue {
	var issues []Issue
	dirs := []string{
		filepath.Join(homeDir(), "Library", "LaunchAgents"),
		"/Library/LaunchAgents",
		"/Library/LaunchDaemons",
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".plist") {
				continue
			}
			path := filepath.Join(dir, e.Name())
			if iss := inspectPlist(path); iss != nil {
				issues = append(issues, *iss)
			}
		}
	}
	return issues
}

func inspectPlist(path string) *Issue {
	f, err := os.Open(path)
	if err != nil {
		return &Issue{
			Category:    "unreadable",
			Path:        path,
			Description: "Cannot read plist",
			Severity:    "warning",
		}
	}
	defer f.Close()

	var agent plistAgent
	if err := plist.NewDecoder(f).Decode(&agent); err != nil {
		return &Issue{
			Category:    "corrupt_plist",
			Path:        path,
			Description: "Corrupted or unparseable plist",
			Severity:    "warning",
		}
	}

	if agent.Label == "" {
		return nil
	}

	programPath := agent.Program
	if programPath == "" && len(agent.ProgramArgs) > 0 {
		programPath = agent.ProgramArgs[0]
	}
	if programPath == "" {
		return nil
	}

	if strings.HasPrefix(programPath, "~/") {
		programPath = filepath.Join(homeDir(), programPath[2:])
	}

	if _, err := os.Stat(programPath); os.IsNotExist(err) {
		cat := "launchagent_zombie"
		if strings.Contains(path, "LaunchDaemons") {
			cat = "launchdaemon_zombie"
		}
		return &Issue{
			Category:    cat,
			Path:        path,
			Description: fmt.Sprintf("References missing binary: %s", programPath),
			Severity:    "error",
		}
	}

	return nil
}

func checkDeadSymlinks() []Issue {
	var issues []Issue
	links := filepath.Join(homeDir(), "Library", "Preferences")
	entries, err := os.ReadDir(links)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		path := filepath.Join(links, e.Name())
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				issues = append(issues, Issue{
					Category:    "dead_symlink",
					Path:        path,
					Description: fmt.Sprintf("Broken symlink \u2192 %s", readLink(path)),
					Severity:    "info",
				})
			}
		}
	}
	return issues
}

func checkXcodeDerivedData() []Issue {
	dd := filepath.Join(homeDir(), "Library", "Developer", "Xcode", "DerivedData")
	entries, err := os.ReadDir(dd)
	if err != nil {
		return nil
	}

	var issues []Issue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dd, e.Name())
		if size := dirSizeQuick(path); size > 100_000_000 {
			issues = append(issues, Issue{
				Category:    "xcode_derived",
				Path:        path,
				Description: fmt.Sprintf("Xcode DerivedData: %s is %s", e.Name(), formatBytes(size)),
				Severity:    "info",
			})
		}
	}
	return issues
}

func checkIOSBackups() []Issue {
	backupDir := filepath.Join(homeDir(), "Library", "Application Support", "MobileSync", "Backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}

	var issues []Issue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(backupDir, e.Name())
		if size := dirSizeQuick(path); size > 50_000_000 {
			issues = append(issues, Issue{
				Category:    "ios_backup",
				Path:        path,
				Description: fmt.Sprintf("iOS backup: %s is %s", e.Name(), formatBytes(size)),
				Severity:    "info",
			})
		}
	}
	return issues
}

func readLink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return "?"
	}
	return target
}

func dirSizeQuick(path string) int64 {
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

func formatBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
