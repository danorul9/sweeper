package liveliness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/core"
	"github.com/danorul9/sweeper/internal/matcher"
	"github.com/danorul9/sweeper/internal/scanner"
)

type Evidence struct {
	Name   string  `json:"name"`
	Detail string  `json:"detail"`
	Weight float64 `json:"weight"`
}

type Item struct {
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	Score     float64    `json:"score"`
	Verdict   string     `json:"verdict"`
	Dead      bool       `json:"dead"`
	Cold      bool       `json:"cold"`
	Evidences []Evidence `json:"evidences"`
}

// ScorePath evaluates a single directory using liveliness evidence.
// Returns nil if the item should be hidden (alive, Apple system, or 0-score noise).
func ScorePath(path string, index *appindex.AppIndex) *Item {
	info, err := os.Stat(path)
	if err != nil {
		return &Item{Path: path, Name: filepath.Base(path)}
	}
	size := scanner.DirSize(path)
	folderName := filepath.Base(path)

	var evs []Evidence

	// Check if this is an Apple system path FIRST — heavy penalty
	if isAppleProtected(folderName, path) {
		return &Item{
			Path:    path,
			Name:    folderName,
			Size:    size,
			Score:   1.5,
			Verdict: "alive",
			Dead:    false,
			Cold:    false,
			Evidences: []Evidence{
				{
					Name:   "apple_system",
					Detail: "Apple system directory — protected",
					Weight: 1.5,
				},
			},
		}
	}

	ev := checkRecentMod(info.ModTime())
	if ev != nil {
		evs = append(evs, *ev)
	}

	ev = checkOpenHandles(path, size)
	if ev != nil {
		evs = append(evs, *ev)
	}

	ev = checkNewestChild(path)
	if ev != nil {
		evs = append(evs, *ev)
	}

	ev = checkSizeEmpty(size)
	if ev != nil {
		evs = append(evs, *ev)
	}

	// Use the full matcher pipeline instead of simple name checks
	if index != nil {
		m := matcher.New(index)
		match := m.Match(folderName, path, info.ModTime())
		switch match.Verdict {
		case core.VerdictInstalled:
			evs = append(evs, Evidence{
				Name:   "app_installed",
				Detail: fmt.Sprintf("Matcher says INSTALLED (%.0f%% confidence)", match.Confidence*100),
				Weight: 0.5,
			})
		case core.VerdictLeftover:
			evs = append(evs, Evidence{
				Name:   "app_leftover",
				Detail: fmt.Sprintf("Matcher says LEFTOVER (%.0f%% confidence)", match.Confidence*100),
				Weight: -0.3,
			})
		}
		for _, s := range match.Signals {
			evs = append(evs, Evidence{
				Name:   s.Kind,
				Detail: s.Detail,
				Weight: s.Weight * 0.5,
			})
		}
	}

	// Sum all weights
	var total float64
	for _, e := range evs {
		total += e.Weight
	}

	// Determine verdict
	verdict := "cold"
	dead := false
	cold := true

	if total > 0.5 {
		verdict = "alive"
		cold = false
	} else if total > 0.3 {
		verdict = "cold"
	} else if total >= -0.1 {
		verdict = "cold"
		cold = true
	} else if total >= -0.5 {
		verdict = "stale"
		dead = true
		cold = false
	} else {
		verdict = "dead"
		dead = true
		cold = false
	}

	return &Item{
		Path:      path,
		Name:      folderName,
		Size:      size,
		Score:     total,
		Verdict:   verdict,
		Dead:      dead,
		Cold:      cold,
		Evidences: evs,
	}
}

func Run(index *appindex.AppIndex) ([]Item, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	var items []Item

	items = append(items, scanHiddenDirs(home, index)...)

	appSupport := filepath.Join(home, "Library", "Application Support")
	items = append(items, scanAppSupport(appSupport, index)...)

	// Filter out items to hide
	filtered := make([]Item, 0, len(items))
	for _, item := range items {
		// Hide alive items
		if item.Verdict == "alive" {
			continue
		}
		// Hide zero-score noise (evidence cancelled out to nothing)
		if item.Score >= -0.1 && item.Score <= 0.1 && item.Size < 10*1024*1024 {
			continue
		}
		// Hide truly empty DEAD items that are tiny
		if item.Verdict == "dead" && item.Size < 1024 && item.Score < -0.5 {
			continue
		}
		filtered = append(filtered, item)
	}

	return filtered, nil
}

func scanHiddenDirs(home string, index *appindex.AppIndex) []Item {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil
	}

	var items []Item
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, ".") {
			continue
		}
		if shouldSkipHiddenFolder(name) {
			continue
		}
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(home, name)

		item := ScorePath(path, index)
		if item.Verdict == "alive" {
			continue
		}
		items = append(items, *item)
	}
	return items
}

func scanAppSupport(appSupport string, index *appindex.AppIndex) []Item {
	entries, err := os.ReadDir(appSupport)
	if err != nil {
		return nil
	}

	var items []Item
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(appSupport, name)

		item := ScorePath(path, index)
		if item.Verdict == "alive" {
			continue
		}
		items = append(items, *item)
	}
	return items
}

// isAppleProtected returns true if the folder is an Apple system component
// that should never be touched. These are protected at the scoring level
// with a high ALIVE weight so they never appear in results.
func isAppleProtected(name, fullPath string) bool {
	lower := strings.ToLower(name)

	// com.apple.* namespace
	if strings.HasPrefix(lower, "com.apple.") {
		return true
	}

	// Known Apple system folders in ~/Library/Application Support/
	appleAppSupport := map[string]bool{
		"app store":               true,
		"appstore":                true,
		"automator":               true,
		"clouddocs":               true,
		"callhistorydb":           true,
		"callhistorytransactions": true,
		"syncservices":            true,
		"addressbook":             true,
		"cloudkit":                true,
		"dock":                    true,
		"spotlight":               true,
		"knowledge":               true,
		"mobile sync":             true,
		"mobilesync":              true,
		"controlcenter":           true,
		"crashreporter":           true,
		"diskimages":              true,
		"differentialprivacy":     true,
		"fileprovider":            true,
		"homeenergyd":             true,
		"icloud":                  true,
		"icloudmailagent":         true,
		"identityservicesd":       true,
		"locationaccessstored":    true,
		"networkserviceproxy":     true,
		"privatecloudcomputed":    true,
		"stickersd":               true,
		"summary-events":          true,
		"tipsd":                   true,
		"contactsd":               true,
		"icdd":                    true,
	}
	return appleAppSupport[lower]
}

func shouldSkipHiddenFolder(name string) bool {
	skip := map[string]bool{
		".":                        true,
		"..":                       true,
		".trash":                   true,
		".spotlight-v100":          true,
		".documentrevisions-v100":  true,
		".fseventsd":               true,
		".pkinstallsandboxmanager": true,
		".localized":               true,
		".ds_store":                true,
		".file":                    true,
		".ssh":                     true,
		".gnupg":                   true,
		".aws":                     true,
		".cups":                    true,
		".identityservice":         true,
		".servicehub":              true,
	}
	return skip[strings.ToLower(name)]
}
