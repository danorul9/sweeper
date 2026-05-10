package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/danorul9/sweeper/internal/core"
)

func (m model) View() string {
	if m.err != nil {
		return m.errorView()
	}
	if m.done {
		return m.doneView()
	}
	if m.confirmDelete {
		return m.confirmView()
	}
	if m.deleting {
		return m.deletingView()
	}
	return m.mainView()
}

func (m model) mainView() string {
	var b strings.Builder

	b.WriteString(m.headerView())
	b.WriteString("\n")
	if m.searching {
		b.WriteString(m.searchBarView())
		b.WriteString("\n")
	} else {
		b.WriteString(m.tabView())
		b.WriteString("\n")
	}
	b.WriteString(m.listView())
	b.WriteString("\n")
	b.WriteString(m.detailView())
	b.WriteString(m.footerView())

	return appStyle.Render(b.String())
}

func (m model) errorView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5484D")).
		Bold(true).
		Render(fmt.Sprintf("Error: %v", m.err))
}

func (m model) doneView() string {
	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			headerStyle.Render("Sweeper — Complete"),
			"",
			emptyStyle.Render(fmt.Sprintf("Deleted %d items", m.deletedCount)),
			"",
			"Press q to quit.",
		),
	)
}

func (m model) headerView() string {
	totalSize := formatBytes(m.results.TotalSize)
	title := fmt.Sprintf("Sweeper — %d items  %s", len(m.results.Items), totalSize)
	return headerStyle.Width(m.width - 4).Render(title)
}

func (m model) tabView() string {
	var tabs []string
	for i, name := range tabNames {
		if i == m.tab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, tabStyle.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m model) listView() string {
	items := m.filteredItems()
	if len(items) == 0 {
		return emptyStyle.Render("No items in this category.")
	}

	availableHeight := m.height - 12
	start := 0
	if m.cursor > availableHeight-1 {
		start = m.cursor - availableHeight + 1
	}
	end := start + availableHeight
	if end > len(items) {
		end = len(items)
	}

	var itemLines []string
	for i, item := range items {
		if i < start {
			continue
		}
		if i >= end {
			break
		}

		globalIdx := m.globalIndex(i)
		sel := m.selected[globalIdx]

		cursor := " "
		if i == m.cursor {
			cursor = cursorStyle.Render("▸")
		}

		check := " "
		if sel {
			check = cursorStyle.Render("●")
		}

		name := item.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		sizeStr := sizeStyle.Render(formatBytes(item.Size))

		badge := m.verdictBadge(item.Match)

		line := fmt.Sprintf("%s %s %-32s %s %s", cursor, check, name, sizeStr, badge)
		if i == m.cursor {
			itemLines = append(itemLines, selectedItemStyle.Render(line))
		} else {
			itemLines = append(itemLines, itemStyle.Render(line))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, itemLines...)
}

func (m model) globalIndex(localIdx int) int {
	if m.tab >= len(tabNames)-1 {
		return localIdx
	}
	count := 0
	for i, item := range m.results.Items {
		if locationTab(item.Location) == tabNames[m.tab] {
			if count == localIdx {
				return i
			}
			count++
		}
	}
	return localIdx
}

func (m model) verdictBadge(match *core.MatchResult) string {
	if match == nil {
		return uncertainStyle.Render("?")
	}
	switch match.Verdict {
	case core.VerdictLeftover:
		return leftoverStyle.Render(fmt.Sprintf("SAFE %.0f%%", match.Confidence*100))
	case core.VerdictInstalled:
		return installedStyle.Render(fmt.Sprintf("KEEP %.0f%%", match.Confidence*100))
	default:
		return uncertainStyle.Render(fmt.Sprintf("UNSURE %.0f%%", match.Confidence*100))
	}
}

func (m model) detailView() string {
	items := m.filteredItems()
	if len(items) == 0 {
		return ""
	}
	if m.cursor >= len(items) {
		m.cursor = len(items) - 1
	}

	item := items[m.cursor]
	match := item.Match
	if match == nil {
		return ""
	}

	var lines []string
	lines = append(lines, "")
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C")).Render(strings.Repeat("─", m.width-4))
	lines = append(lines, sep)
	lines = append(lines, fmt.Sprintf(" %s", itemStyle.Render(item.Path)))
	lines = append(lines, "")

	for _, s := range match.Signals {
		prefix := checkMark
		switch s.Kind {
		case "no_match", "partial_name":
			prefix = warningMark
		}
		lines = append(lines, fmt.Sprintf("   %s %s", prefix, signalStyle.Render(s.Detail)))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("   Location: %s", item.Location))
	lines = append(lines, fmt.Sprintf("   Modified: %s", item.ModTime.Format("Jan 2, 2006")))
	lines = append(lines, fmt.Sprintf("   Size:     %s", formatBytes(item.Size)))
	lines = append(lines, sep)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) searchBarView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#3A3A3C")).
		Padding(0, 1).
		Render(fmt.Sprintf("Search: %s█", m.searchQuery))
}

func (m model) footerView() string {
	selected := m.totalSelected()
	selectedSize := formatBytes(m.selectedSize())
	selInfo := ""
	if selected > 0 {
		selInfo = fmt.Sprintf(" %d selected (%s) |", selected, selectedSize)
	}

	return footerStyle.Render(
		fmt.Sprintf("%s ↑↓/jk nav | space toggle | d delete | tab switch | / search | q quit",
			selInfo,
		),
	)
}

func (m model) confirmView() string {
	selected := m.totalSelected()
	selectedSize := formatBytes(m.selectedSize())

	var items []string
	for i, item := range m.results.Items {
		if m.selected[i] {
			items = append(items, fmt.Sprintf("  • %s (%s)", item.Name, formatBytes(item.Size)))
		}
	}

	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			confirmTitleStyle.Render(fmt.Sprintf("Delete %d items (%s)?", selected, selectedSize)),
			"",
			lipgloss.JoinVertical(lipgloss.Left, items...),
			"",
			helpKeyStyle.Render("y")+helpDescStyle.Render("es, delete")+"  "+helpKeyStyle.Render("n")+helpDescStyle.Render("o, cancel"),
			"",
		),
	)
}

func (m model) deletingView() string {
	return appStyle.Render(
		confirmTitleStyle.Render(fmt.Sprintf("Deleting %d items...", m.totalSelected())),
	)
}
