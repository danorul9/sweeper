package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5E6AD2")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(lipgloss.Color("#A0A0A0"))

	activeTabStyle = tabStyle.Copy().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#5E6AD2")).
			Bold(true)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("#5E6AD2")).
				Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5E6AD2")).
			Bold(true)

	leftoverStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#46A758")).
			Bold(true)

	installedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5484D"))

	uncertainStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F5A623"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8E8E93")).
				Width(14).
				Align(lipgloss.Right)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	signalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0"))

	checkMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#46A758")).
			Render("✓")

	warningMark = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F5A623")).
			Render("⚠")

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			PaddingTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5E6AD2")).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0"))

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Italic(true).
			PaddingLeft(2)

	confirmTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#E5484D"))

	confirmItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	sizeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8E93")).
			Width(10).
			Align(lipgloss.Right)

	toastStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#30D158")).
			Padding(0, 1)

	groupHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#4A52B0")).
				Padding(0, 1)
)

// ageStyle returns a colored human-readable age string with one decimal for months/years.
func ageStyle(days int) string {
	style := lipgloss.NewStyle().Width(9).Align(lipgloss.Right)
	var s string
	switch {
	case days < 7:
		s = fmt.Sprintf("%dd", days)
		style = style.Foreground(lipgloss.Color("#46A758")) // green = recent
	case days < 60:
		s = fmt.Sprintf("%dd", days)
		style = style.Foreground(lipgloss.Color("#F5A623")) // yellow = recent-ish
	case days < 365:
		m := float64(days) / 30.0
		s = fmt.Sprintf("%.1fm", m)
		style = style.Foreground(lipgloss.Color("#E58D00")) // orange = months
	default:
		y := float64(days) / 365.0
		s = fmt.Sprintf("%.1fy", y)
		style = style.Foreground(lipgloss.Color("#E5484D")) // red = old
	}
	return style.Render(s)
}

var tabNames = []string{
	"Caches",
	"Saved State",
	"Logs",
	"Temp Items",
	"App Support",
	"Containers",
	"Hidden",
	"All",
}
