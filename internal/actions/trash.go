package actions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Trash(paths ...string) error {
	for _, p := range paths {
		if strings.Contains(p, "LaunchAgents") || strings.Contains(p, "LaunchDaemons") {
			unloadLaunchdPlist(p)
		}
		if err := trashFile(p); err != nil {
			return fmt.Errorf("trash %s: %w", p, err)
		}
	}
	return nil
}

func unloadLaunchdPlist(path string) {
	cmd := exec.Command("launchctl", "unload", path)
	cmd.Run()
}

func trashFile(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fallbackOsaScript(path)
	}
	trashDir := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trashDir, 0700); err != nil {
		return fallbackOsaScript(path)
	}

	base := filepath.Base(path)
	dest := filepath.Join(trashDir, base)

	if _, err := os.Stat(dest); err == nil {
		dest = uniqueName(trashDir, base)
	}

	if err := os.Rename(path, dest); err != nil {
		if isCrossDevice(err) {
			return fallbackOsaScript(path)
		}
		return err
	}

	return nil
}

func uniqueName(dir, base string) string {
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	for i := 1; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func isCrossDevice(err error) bool {
	return strings.Contains(err.Error(), "cross-device") || strings.Contains(err.Error(), "invalid cross-device")
}

func fallbackOsaScript(path string) error {
	script := fmt.Sprintf(`tell app "Finder" to delete POSIX file "%s"`, strings.ReplaceAll(path, `"`, `\"`))
	cmd := exec.Command("osascript", "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "-43") || strings.Contains(string(output), "-1728") {
			return fallbackOsaScriptMove(path)
		}
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func fallbackOsaScriptMove(path string) error {
	mvCmd := exec.Command("osascript", "-e",
		fmt.Sprintf(`tell application "Finder" to move POSIX file "%s" to trash`, strings.ReplaceAll(path, `"`, `\"`)))
	if output, err := mvCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func TrashAvailable() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	trashDir := filepath.Join(home, ".Trash")
	if _, err := os.Stat(trashDir); os.IsNotExist(err) {
		if err := os.MkdirAll(trashDir, 0700); err != nil {
			return false
		}
	}
	return true
}
