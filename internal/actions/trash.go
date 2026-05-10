package actions

import (
	"fmt"
	"os/exec"
	"strings"
)

func Trash(paths ...string) error {
	for _, p := range paths {
		if err := trashFile(p); err != nil {
			return fmt.Errorf("trash %s: %w", p, err)
		}
	}
	return nil
}

func trashFile(path string) error {
	script := fmt.Sprintf(`tell app "Finder" to delete POSIX file "%s"`, strings.ReplaceAll(path, `"`, `\"`))
	cmd := exec.Command("osascript", "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "-43") || strings.Contains(string(output), "-1728") {
			return tryAltTrash(path)
		}
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func tryAltTrash(path string) error {
	mvCmd := exec.Command("osascript", "-e",
		fmt.Sprintf(`tell application "Finder" to move POSIX file "%s" to trash`, strings.ReplaceAll(path, `"`, `\"`)))
	if output, err := mvCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func TrashAvailable() bool {
	cmd := exec.Command("which", "osascript")
	return cmd.Run() == nil
}
