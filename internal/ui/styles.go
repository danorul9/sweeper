package ui

import (
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
)

var tabNames = []string{
	"Caches",
	"Saved State",
	"Logs",
	"Temp Items",
	"App Support",
	"Containers",
	"All",
}
