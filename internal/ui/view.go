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
	if m.done || m.fDeleted {
		return m.doneView()
	}
	if m.featureLoading {
		return m.loadingView()
	}
	switch m.screen {
	case screenMenu:
		return m.menuView()
	case screenScan:
		return m.scanView()
	case screenApps, screenLarge, screenDupes, screenDoctor, screenReclaim, screenUndo, screenStats, screenLiveliness:
		if m.fConfirmDel {
			return m.featureConfirmView()
		}
		if m.fDeleting {
			return m.featureDeletingView()
		}
		return m.featureView()
	}
	return ""
}

func (m model) errorView() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5484D")).
		Bold(true).
		Render(fmt.Sprintf("Error: %v", m.err))
}

func (m model) loadingView() string {
	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			headerStyle.Render("Sweeper"),
			"",
			emptyStyle.Render("Working..."),
		),
	)
}

func (m model) doneView() string {
	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			headerStyle.Render("Sweeper \u2014 Complete"),
			"",
			emptyStyle.Render(fmt.Sprintf("Deleted %d items", m.deletedCount)),
			"",
			"Press q to quit.",
		),
	)
}

func (m model) menuView() string {
	title := headerStyle.Width(m.width - 4).Render("Sweeper \u2014 macOS App Leftover Detector")

	var sections []string

	titleSection := title
	if m.toast != "" {
		titleSection = lipgloss.JoinVertical(lipgloss.Left,
			title,
			toastStyle.Render(m.toast),
		)
	}
	sections = append(sections, titleSection)
	sections = append(sections, "")

	var items []string
	for i, mi := range menuItems {
		cursor := "  "
		if i == m.menuCursor {
			cursor = cursorStyle.Render("\u25b8 ")
		}
		line := fmt.Sprintf("%s%s", cursor, mi.title)
		if i == m.menuCursor {
			items = append(items, selectedItemStyle.Render(line))
		} else {
			items = append(items, itemStyle.Render(line))
		}
		items = append(items, itemStyle.Render(fmt.Sprintf("   %s", detailValueStyle.Render(mi.description))))
		items = append(items, "")
	}

	sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, items...))

	footer := footerStyle.Render("\u2191\u2193 nav | enter select | q quit")
	sections = append(sections, footer)

	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top, sections...),
	)
}

func (m model) scanView() string {
	if m.confirmDelete {
		return m.confirmView()
	}
	if m.deleting {
		return m.deletingView()
	}
	if m.results == nil {
		return m.loadingView()
	}
	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n")
	if m.searching {
		b.WriteString(m.searchBarView())
		b.WriteString("\n")
	} else {
		b.WriteString(m.tabView())
		b.WriteString("\n\n")
	}
	b.WriteString(m.listView())
	b.WriteString("\n")
	b.WriteString(m.detailView())
	b.WriteString(m.scanFooterView())
	return appStyle.Render(b.String())
}

func (m model) featureView() string {
	title := headerStyle.Width(m.width - 4).Render(m.fTitle)

	if len(m.items) == 0 {
		body := emptyStyle.Render("No items found.")
		footer := footerStyle.Render("esc back | q quit")
		return appStyle.Render(
			lipgloss.JoinVertical(lipgloss.Top, title, "", body, footer),
		)
	}

	// Calculate info panel height for scroll accounting
	infoPanelHeight := m.featureInfoPanelHeight()

	availableHeight := m.height - 6 - infoPanelHeight
	if availableHeight < 3 {
		availableHeight = 3
	}
	start := 0
	if m.cursor > availableHeight-1 {
		start = m.cursor - availableHeight + 1
	}
	end := start + availableHeight
	if end > len(m.items) {
		end = len(m.items)
	}

	// Horizontal padding from outer containers
	// appStyle has Padding(1,2) = 2 left padding
	// itemStyle has PaddingLeft(2) = 2 left padding

	var lines []string
	for i, item := range m.items {
		if i < start {
			continue
		}
		if i >= end {
			break
		}

		if item.IsColumnHeader {
			// Plain text — matches data row alignment exactly
			// Data rows: itemStyle PaddingLeft(2) + cursor + " " + check + " " = 6 before name
			// groupHeaderStyle has Padding(0,1) = 1 left pad, so use 5 leading spaces
			left := fmt.Sprintf("     %s", item.Name)
			leftWidth := lipgloss.Width(left)

			// Show labels only for columns that the feature actually displays
			// Matches data row conditions: Size>0, AgeDays>=0, Detail!=""
			sizeStr := ""
			if item.Size > 0 {
				sizeStr = fmt.Sprintf("%10s", "SIZE")
			}
			ageStr := ""
			if item.AgeDays >= 0 {
				ageStr = "  " + fmt.Sprintf("%9s", "AGE")
			}
			baseRight := sizeStr + ageStr
			baseRightWidth := lipgloss.Width(baseRight)

			usable := m.width - 6
			remaining := usable - leftWidth - baseRightWidth
			if remaining < 1 {
				remaining = 1
			}

			detailStr := ""
			if item.Detail != "" && remaining > 3 {
				detailText := item.Detail
				maxDetail := remaining - 3
				if maxDetail < 1 {
					maxDetail = 1
				}
				if len(detailText) > maxDetail {
					detailText = detailText[:maxDetail-1] + "…"
				}
				detailStr = "  " + detailText
			}

			rightWidth := baseRightWidth + lipgloss.Width(detailStr)
			gap := usable - leftWidth - rightWidth
			if gap < 1 {
				gap = 1
			}

			line := left + strings.Repeat(" ", gap) + baseRight + detailStr
			lines = append(lines, groupHeaderStyle.Width(m.width-4).Render(line))
			continue
		}
		if item.IsHeader {
			hdr := groupHeaderStyle.Width(m.width - 4).Render(item.Name)
			lines = append(lines, hdr)
			continue
		}

		cursor := " "
		if i == m.cursor {
			cursor = cursorStyle.Render("\u25b8")
		}

		check := " "
		if m.fSelected[i] {
			check = cursorStyle.Render("\u25cf")
		}

		name := item.Name
		if name == "" {
			name = item.Path
		}
		displayName := name
		// Build left part (path, never truncated)
		left := fmt.Sprintf("%s %s %s", cursor, check, displayName)
		leftWidth := lipgloss.Width(left)

		// Build right part
		sizeStr := ""
		if item.Size > 0 {
			sizeStr = sizeStyle.Render(formatBytes(item.Size))
		}
		ageStr := ""
		if item.AgeDays >= 0 {
			ageStr = "  " + ageStyle(item.AgeDays)
		}
		baseRight := sizeStr + ageStr
		baseRightWidth := lipgloss.Width(baseRight)

		// Calculate available space for detail
		// Total usable: m.width - 4 (appStyle pad) - 2 (itemStyle pad)
		usable := m.width - 6
		remaining := usable - leftWidth - baseRightWidth
		if remaining < 1 {
			remaining = 1
		}

		detailStr := ""
		if item.Detail != "" && remaining > 3 {
			detailText := item.Detail
			// Leave 3 chars minimum for "..."
			maxDetail := remaining - 3
			if maxDetail < 1 {
				maxDetail = 1
			}
			if len(detailText) > maxDetail {
				detailText = detailText[:maxDetail-1] + "…"
			}
			detailStr = "  " + signalStyle.Render(detailText)
		}

		// Calculate gap to push right part to the right edge
		rightWidth := baseRightWidth + lipgloss.Width(detailStr)
		gap := usable - leftWidth - rightWidth
		if gap < 1 {
			gap = 1
		}

		line := left + strings.Repeat(" ", gap) + baseRight + detailStr
		if i == m.cursor {
			lines = append(lines, selectedItemStyle.Render(line))
		} else {
			lines = append(lines, itemStyle.Render(line))
		}
	}

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)

	totalLine := ""
	if m.fTotal != "" {
		totalLine = "  " + detailValueStyle.Render(m.fTotal)
	}

	selInfo := ""
	if n := m.fTotalSelected(); n > 0 {
		selInfo = fmt.Sprintf(" %d selected (%s) |", n, formatBytes(m.fSelectedSize()))
	}

	footer := footerStyle.Render(fmt.Sprintf("%s%s ↑↓/jk nav | space toggle | a all | n none | d delete | esc back | q quit", selInfo, totalLine))

	// Info panel at the bottom
	infoPanel := m.featureInfoPanel()

	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top, title, "", body, infoPanel, footer),
	)
}

func (m model) featureConfirmView() string {
	n := m.fTotalSelected()
	sz := formatBytes(m.fSelectedSize())

	var items []string
	for i := range m.items {
		if m.fSelected[i] {
			items = append(items, fmt.Sprintf("  \u2022 %s (%s)", m.items[i].Name, formatBytes(m.items[i].Size)))
		}
	}

	return appStyle.Render(
		lipgloss.JoinVertical(lipgloss.Top,
			confirmTitleStyle.Render(fmt.Sprintf("Delete %d items (%s)?", n, sz)),
			"",
			lipgloss.JoinVertical(lipgloss.Left, items...),
			"",
			helpKeyStyle.Render("y")+helpDescStyle.Render("es, delete")+"  "+helpKeyStyle.Render("n")+helpDescStyle.Render("o, cancel"),
			"",
		),
	)
}

func (m model) featureDeletingView() string {
	return appStyle.Render(
		confirmTitleStyle.Render(fmt.Sprintf("Deleting %d items...", m.fTotalSelected())),
	)
}

func (m model) featureInfoPanel() string {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]
	if item.IsHeader || item.IsColumnHeader || len(item.InfoRows) == 0 {
		return ""
	}

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8E8E93")).
		PaddingLeft(1)

	// Limit rows to fit on screen
	maxRows := 6
	rows := item.InfoRows
	if len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	var lines []string
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C")).Render(strings.Repeat("─", m.width-4))
	lines = append(lines, sep)
	for _, row := range rows {
		lines = append(lines, infoStyle.Render(row))
	}
	lines = append(lines, sep)

	s := strings.Builder{}
	for i, line := range lines {
		if i > 0 {
			s.WriteString("\n")
		}
		s.WriteString(line)
	}
	return s.String()
}

func (m model) featureInfoPanelHeight() int {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return 0
	}
	item := m.items[m.cursor]
	if item.IsHeader || item.IsColumnHeader || len(item.InfoRows) == 0 {
		return 0
	}
	// 2 separator lines + max 5 content rows + 1 padding = 8
	h := len(item.InfoRows) + 2
	if h > 7 {
		h = 7
	}
	return h
}

func (m model) headerView() string {
	totalSize := formatBytes(m.results.TotalSize)
	knownInfo := ""
	if m.results.KnownAppsCount > 0 {
		knownInfo = fmt.Sprintf(" | %d apps (%d known)", m.results.AppCount, m.results.KnownAppsCount)
	}
	title := fmt.Sprintf("Sweeper \u2014 %d items  %s%s", len(m.results.Items), totalSize, knownInfo)
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

	detailH := m.detailViewHeight()
	availableHeight := m.height - 8 - detailH
	if availableHeight < 3 {
		availableHeight = 3
	}
	start := 0
	if m.cursor > availableHeight-1 {
		start = m.cursor - availableHeight + 1
	}
	end := start + availableHeight
	if end > len(items) {
		end = len(items)
	}

	var itemLines []string
	// Column header — dynamic gap to match data row right-edge alignment
	// Data rows: itemStyle PaddingLeft(2) + cursor+" "+check+" " = 6 before name
	// groupHeaderStyle Padding(0,1) = 1 left pad, so use 5 leading spaces
	left := fmt.Sprintf("     %-32s", "NAME")
	rightPart := fmt.Sprintf("%10s %s", "SIZE", "VERDICT")
	usable := m.width - 6
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(rightPart)
	gap := usable - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}
	headerLine := left + strings.Repeat(" ", gap) + rightPart
	itemLines = append(itemLines, groupHeaderStyle.Width(m.width-4).Render(headerLine), "")
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
			cursor = cursorStyle.Render("\u25b8")
		}

		check := " "
		if sel {
			check = cursorStyle.Render("\u25cf")
		}

		name := item.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}

		sizeStr := sizeStyle.Render(formatBytes(item.Size))

		badge := m.verdictBadge(item.Match)

		// Build left and right parts with dynamic gap — matches featureView pattern
		left := fmt.Sprintf("%s %s %s", cursor, check, name)
		leftWidth := lipgloss.Width(left)
		baseRight := sizeStr + " " + badge
		baseRightWidth := lipgloss.Width(baseRight)
		usable := m.width - 6
		gap := usable - leftWidth - baseRightWidth
		if gap < 1 {
			gap = 1
		}
		line := left + strings.Repeat(" ", gap) + baseRight
		if i == m.cursor {
			itemLines = append(itemLines, selectedItemStyle.Render(line))
		} else {
			itemLines = append(itemLines, itemStyle.Render(line))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, itemLines...)
}

func (m model) globalIndex(localIdx int) int {
	if tabNames[m.tab] == "All" {
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

func (m model) detailViewHeight() int {
	items := m.filteredItems()
	if len(items) == 0 {
		return 0
	}
	if m.cursor >= len(items) {
		return 0
	}
	item := items[m.cursor]
	match := item.Match
	if match == nil {
		return 0
	}
	return 8 + len(match.Signals)
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
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3C")).Render(strings.Repeat("\u2500", m.width-4))
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
		Render(fmt.Sprintf("Search: %s\u2588", m.searchQuery))
}

func (m model) scanFooterView() string {
	selected := m.totalSelected()
	selectedSize := formatBytes(m.selectedSize())
	selInfo := ""
	if selected > 0 {
		selInfo = fmt.Sprintf(" %d selected (%s) |", selected, selectedSize)
	}

	return footerStyle.Render(
		fmt.Sprintf("%s \u2191\u2193/jk nav | space toggle | d delete | tab switch | / search | esc menu | q quit",
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
			items = append(items, fmt.Sprintf("  \u2022 %s (%s)", item.Name, formatBytes(item.Size)))
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
