package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danorul9/sweeper/internal/core"
)

type screen int

const (
	screenMenu screen = iota
	screenScan
	screenApps
	screenLarge
	screenDupes
	screenDoctor
	screenReclaim
	screenUndo
	screenStats
	screenLiveliness
)

type HubItem struct {
	Name     string
	Path     string
	Size     int64
	Detail   string
	Signals  []string
	IsHeader       bool
	IsColumnHeader bool
	InfoRows []string // bottom info panel rows for the focused item
	AgeDays  int       // age of newest content in days, -1 if unknown
}

type model struct {
	screen         screen
	menuCursor     int
	width, height  int
	err            error
	featureLoading bool
	toast          string

	results        *core.ScanResult
	tab            int
	cursor         int
	selected       map[int]bool
	confirmDelete  bool
	deleting       bool
	deletedCount   int
	deleteError    string
	done           bool
	searching      bool
	searchQuery    string

	items        []HubItem
	fTitle       string
	fTotal       string
	fSelected    map[int]bool
	fConfirmDel  bool
	fDeleting    bool
	fDeleted     bool
}

var menuItems = []struct {
	title       string
	description string
	screen      screen
}{
	{"Detected Apps", "Scan and list all installed applications", screenApps},
	{"Orphan Scanner", "Find leftover files from uninstalled apps", screenScan},
	{"Liveliness", "Evidence-based orphan detection for ~/.* directories", screenLiveliness},
	{"Large Files", "Find files over 100MB", screenLarge},
	{"Duplicates", "Find duplicate files by checksum", screenDupes},
	{"Doctor", "Zombie services, dead symlinks, system cruft", screenDoctor},
	{"Reclaim", "Safe caches & logs only", screenReclaim},
	{"Undo Last Cleanup", "Restore files from Trash", screenUndo},
	{"Stats", "Historical cleanup analytics", screenStats},
}

func InitialModel() model {
	return model{
		screen:     screenMenu,
		menuCursor: 0,
		selected:   make(map[int]bool),
		fSelected:  make(map[int]bool),
		width:      80,
		height:     24,
	}
}

func InitialScanModel(results *core.ScanResult) model {
	return model{
		screen:   screenScan,
		results:  results,
		tab:      0,
		cursor:   0,
		selected: make(map[int]bool),
		width:    80,
		height:   24,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) filteredItems() []core.Leftover {
	var filtered []core.Leftover
	tabType := tabNames[m.tab]
	for _, item := range m.results.Items {
		if tabType != "All" && locationTab(item.Location) != tabType {
			continue
		}
		if m.searchQuery != "" && !matchesSearch(item, m.searchQuery) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func matchesSearch(item core.Leftover, q string) bool {
	q = strings.ToLower(q)
	return strings.Contains(strings.ToLower(item.Name), q) ||
		strings.Contains(strings.ToLower(item.Path), q)
}

func (m model) totalSelected() int {
	count := 0
	for _, sel := range m.selected {
		if sel {
			count++
		}
	}
	return count
}

func (m model) selectedSize() int64 {
	var total int64
	for i, item := range m.results.Items {
		if m.selected[i] {
			total += item.Size
		}
	}
	return total
}

func (m model) fTotalSelected() int {
	count := 0
	for _, sel := range m.fSelected {
		if sel {
			count++
		}
	}
	return count
}

func (m model) fSelectedSize() int64 {
	var total int64
	for i := range m.items {
		if m.fSelected[i] {
			total += m.items[i].Size
		}
	}
	return total
}

func locationTab(loc string) string {
	switch loc {
	case "Caches":
		return "Caches"
	case "Saved Application State":
		return "Saved State"
	case "Logs":
		return "Logs"
	case "TemporaryItems":
		return "Temp Items"
	case "Application Support":
		return "App Support"
	case "Containers":
		return "Containers"
	case "Hidden Home":
		return "Hidden"
	case "Dot Cache":
		return "Dot Cache"
	default:
		return "Other"
	}
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

func colorForVerdict(v core.Verdict) string {
	switch v {
	case core.VerdictLeftover:
		return "leftover"
	case core.VerdictInstalled:
		return "installed"
	default:
		return "uncertain"
	}
}
