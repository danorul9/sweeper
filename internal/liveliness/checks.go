package liveliness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/appindex"
)

func timeoutContext(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func checkRecentMod(modTime time.Time) *Evidence {
	age := time.Since(modTime)
	if age < 90*24*time.Hour {
		return &Evidence{
			Name:   "recent_mod",
			Detail: fmt.Sprintf("Modified %s ago (within 90 days)", roundAge(age)),
			Weight: 0.4,
		}
	}
	if age > 180*24*time.Hour {
		return &Evidence{
			Name:   "old_mod",
			Detail: fmt.Sprintf("Last modified %s ago (over 6 months)", roundAge(age)),
			Weight: -0.3,
		}
	}
	return nil
}

func roundAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	switch {
	case days >= 365:
		return fmt.Sprintf("%dy", days/365)
	case days >= 30:
		return fmt.Sprintf("%dm", days/30)
	default:
		return fmt.Sprintf("%dd", days)
	}
}

func checkOpenHandles(path string, size int64) *Evidence {
	// Skip very large dirs — lsof +D would take too long
	if size > 500*1024*1024 {
		return nil
	}

	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof", "+D", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	err := cmd.Run()

	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return &Evidence{
				Name:   "no_open_handles",
				Detail: "No process has open handles (lsof empty)",
				Weight: -0.1,
			}
		}
		return nil
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	count := len(lines)
	if count > 1 {
		return &Evidence{
			Name:   "open_handles",
			Detail: fmt.Sprintf("Running process has %d open handles", count-1),
			Weight: 0.5,
		}
	}
	return nil
}

func checkNewestChild(path string) *Evidence {
	var newest time.Time
	filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})

	if newest.IsZero() {
		return nil
	}

	age := time.Since(newest)
	if age > 180*24*time.Hour {
		return &Evidence{
			Name:   "all_children_old",
			Detail: fmt.Sprintf("Newest child is %s old (over 6 months)", roundAge(age)),
			Weight: -0.3,
		}
	}
	if age < 90*24*time.Hour {
		return &Evidence{
			Name:   "recent_child",
			Detail: fmt.Sprintf("Has child files modified %s ago (within 90 days)", roundAge(age)),
			Weight: 0.3,
		}
	}
	return nil
}

func checkSizeEmpty(size int64) *Evidence {
	if size == 0 {
		return &Evidence{
			Name:   "empty",
			Detail: "Directory is empty (0 bytes)",
			Weight: -0.4,
		}
	}
	if size < 4096 {
		return &Evidence{
			Name:   "nearly_empty",
			Detail: "Directory is nearly empty (< 4 KB)",
			Weight: -0.2,
		}
	}
	return nil
}

// checkBinaryOnPath looks for a binary on PATH that corresponds to this folder.
// Strips leading "." and tries multiple name variations.
// If a binary is found, it means the tool is currently installed.
func checkBinaryOnPath(name string) *Evidence {
	clean := strings.TrimPrefix(name, ".")
	if clean == "" {
		return nil
	}

	// Try multiple name variations
	candidates := []string{clean, strings.ToLower(clean), strings.ToUpper(clean)}
	// Add title case variants
	if len(clean) > 0 {
		title := strings.ToUpper(clean[:1]) + clean[1:]
		candidates = append(candidates, title)
	}

	seen := make(map[string]bool)
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		seen[c] = true
		_, err := exec.LookPath(c)
		if err == nil {
			return &Evidence{
				Name:   "binary_on_path",
				Detail: fmt.Sprintf("Binary %q found on PATH", c),
				Weight: 0.6,
			}
		}
	}

	// Also check via `which -a` for case-insensitive shells
	ctx, cancel := timeoutContext(2 * time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "which", "-a", clean)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		output := strings.TrimSpace(out.String())
		if output != "" {
			return &Evidence{
				Name:   "binary_on_path",
				Detail: fmt.Sprintf("Binary %q found on PATH", clean),
				Weight: 0.6,
			}
		}
	}

	return nil
}

func checkAppInstalled(name string, index *appindex.AppIndex) *Evidence {
	if index == nil {
		return nil
	}
	clean := strings.TrimPrefix(name, ".")
	if clean == "" {
		return nil
	}

	for bid := range index.BundleIDs {
		short := shorterBundleID(bid)
		if strings.EqualFold(bid, clean) || (short != "" && strings.EqualFold(short, clean)) {
			return &Evidence{
				Name:   "app_installed",
				Detail: "App found in index: " + bid,
				Weight: 0.4,
			}
		}
	}

	for appName := range index.Names {
		if strings.EqualFold(appName, clean) {
			return &Evidence{
				Name:   "app_installed",
				Detail: "App found in index: " + appName,
				Weight: 0.4,
			}
		}
	}

	return nil
}

func shorterBundleID(bid string) string {
	parts := strings.Split(bid, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}
