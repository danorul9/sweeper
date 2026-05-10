package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danorul9/sweeper/internal/core"
)

type model struct {
	results        *core.ScanResult
	tab            int
	cursor         int
	selected       map[int]bool
	width          int
	height         int
	err            error
	confirmDelete  bool
	deleting       bool
	deletedCount   int
	deleteError    string
	done           bool
	searching      bool
	searchQuery    string
}

func InitialModel(results *core.ScanResult) model {
	return model{
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
		if m.tab < len(tabNames)-1 && locationTab(item.Location) != tabType {
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
