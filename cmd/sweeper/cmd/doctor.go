package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"howett.net/plist"
)

type plistAgent struct {
	Program         string   `plist:"Program"`
	ProgramArgs     []string `plist:"ProgramArguments"`
	Label           string   `plist:"Label"`
	KeepAlive       bool     `plist:"KeepAlive"`
	RunAtLoad       bool     `plist:"RunAtLoad"`
}

type doctorIssue struct {
	Category    string
	Path        string
	Description string
	Severity    string // "error", "warning", "info"
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Detect zombie services, dead symlinks, and system cruft",
	Long: `Check for orphaned LaunchAgents, LaunchDaemons, dead symlinks,
corrupted plists, old Xcode derived data, iOS device backups, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, _ := cmd.Flags().GetBool("json")

		var issues []doctorIssue

		issues = append(issues, checkLaunchAgents()...)
		issues = append(issues, checkDeadSymlinks()...)
		issues = append(issues, checkXcodeDerivedData()...)
		issues = append(issues, checkIOSBackups()...)

		if jsonOutput {
			return printDoctorJSON(issues)
		}

		printDoctorReport(issues)
		return nil
	},
}

func init() {
	doctorCmd.Flags().Bool("json", false, "Output as JSON")
	doctorCmd.Flags().Bool("dry-run", false, "Preview only (no-op for doctor)")
}

func checkLaunchAgents() []doctorIssue {
	var issues []doctorIssue
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
			issue := inspectPlist(path)
			if issue != nil {
				issues = append(issues, *issue)
			}
		}
	}
	return issues
}

func inspectPlist(path string) *doctorIssue {
	f, err := os.Open(path)
	if err != nil {
		return &doctorIssue{
			Category:    "unreadable",
			Path:        path,
			Description: "Cannot read plist",
			Severity:    "warning",
		}
	}
	defer f.Close()

	var agent plistAgent
	if err := plist.NewDecoder(f).Decode(&agent); err != nil {
		return &doctorIssue{
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

	// Resolve ~/ and relative paths
	if strings.HasPrefix(programPath, "~/") {
		programPath = filepath.Join(homeDir(), programPath[2:])
	}

	if _, err := os.Stat(programPath); os.IsNotExist(err) {
		cat := "launchagent_zombie"
		if strings.Contains(path, "LaunchDaemons") {
			cat = "launchdaemon_zombie"
		}
		return &doctorIssue{
			Category:    cat,
			Path:        path,
			Description: fmt.Sprintf("References missing binary: %s", programPath),
			Severity:    "error",
		}
	}

	return nil
}

func checkDeadSymlinks() []doctorIssue {
	var issues []doctorIssue

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
				issues = append(issues, doctorIssue{
					Category:    "dead_symlink",
					Path:        path,
					Description: fmt.Sprintf("Broken symlink → %s", readLink(path)),
					Severity:    "info",
				})
			}
		}
	}
	return issues
}

func checkXcodeDerivedData() []doctorIssue {
	dd := filepath.Join(homeDir(), "Library", "Developer", "Xcode", "DerivedData")
	entries, err := os.ReadDir(dd)
	if err != nil {
		return nil
	}

	var issues []doctorIssue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dd, e.Name())
		size := dirSizeQuick(path)
		if size > 100_000_000 {
			issues = append(issues, doctorIssue{
				Category:    "xcode_derived",
				Path:        path,
				Description: fmt.Sprintf("Xcode DerivedData: %s is %s", e.Name(), formatSize(size)),
				Severity:    "info",
			})
		}
	}
	return issues
}

func checkIOSBackups() []doctorIssue {
	backupDir := filepath.Join(homeDir(), "Library", "Application Support", "MobileSync", "Backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}

	var issues []doctorIssue
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(backupDir, e.Name())
		size := dirSizeQuick(path)
		if size > 50_000_000 {
			issues = append(issues, doctorIssue{
				Category:    "ios_backup",
				Path:        path,
				Description: fmt.Sprintf("iOS backup: %s is %s", e.Name(), formatSize(size)),
				Severity:    "info",
			})
		}
	}
	return issues
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
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

func printDoctorReport(issues []doctorIssue) {
	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return
	}

	fmt.Printf("Found %d issues:\n\n", len(issues))
	for _, iss := range issues {
		severity := "INFO"
		switch iss.Severity {
		case "error":
			severity = "ERROR"
		case "warning":
			severity = "WARN"
		}
		fmt.Printf("  [%s] %s\n", severity, iss.Description)
		fmt.Printf("        %s\n\n", iss.Path)
	}
}

func printDoctorJSON(issues []doctorIssue) error {
	enc := map[string]interface{}{
		"total":  len(issues),
		"issues": issues,
	}
	data, err := json.MarshalIndent(enc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
