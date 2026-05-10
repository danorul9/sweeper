package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danorul9/sweeper/internal/actions"
	"github.com/danorul9/sweeper/internal/core"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmDelete {
		return m.handleConfirmKey(msg)
	}

	if m.deleting {
		return m, tea.Quit
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		items := m.filteredItems()
		if m.cursor < len(items)-1 {
			m.cursor++
		}

	case "left", "h":
		if m.tab > 0 {
			m.tab--
			m.cursor = 0
		}

	case "right", "l":
		if m.tab < len(tabNames)-1 {
			m.tab++
			m.cursor = 0
		}

	case "tab":
		m.tab = (m.tab + 1) % len(tabNames)
		m.cursor = 0

	case " ":
		items := m.filteredItems()
		if len(items) > 0 && m.cursor < len(items) {
			globalIdx := m.globalIndex(m.cursor)
			m.selected[globalIdx] = !m.selected[globalIdx]
		}

	case "d":
		if m.totalSelected() > 0 {
			m.confirmDelete = true
		}

	case "/":
		m.searching = true
		m.searchQuery = ""

	case "a":
		for i := range m.results.Items {
			m.selected[i] = true
		}
	}

	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeDelete()
	case "n", "N", "esc", "q":
		m.confirmDelete = false
		return m, nil
	}
	return m, nil
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.searching = false
		m.cursor = 0
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		m.cursor = 0
	case "tab":
		m.searching = false
		return m.handleKey(msg)
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) executeDelete() (tea.Model, tea.Cmd) {
	m.deleting = true

	var paths []string
	var items []core.Leftover
	for i, item := range m.results.Items {
		if m.selected[i] {
			paths = append(paths, item.Path)
			items = append(items, item)
		}
	}

	snapshot, err := actions.SaveSnapshot(items)
	if err != nil {
		m.deleteError = err.Error()
		return m, tea.Quit
	}

	if err := actions.Trash(paths...); err != nil {
		m.deleteError = err.Error()
		return m, tea.Quit
	}

	actions.LogOperation(actions.OperationLog{
		Timestamp: snapshot.Timestamp,
		Action:    "delete",
		Paths:     paths,
		TotalSize: m.selectedSize(),
		Success:   true,
	})

	m.deletedCount = len(paths)
	m.done = true
	return m, tea.Quit
}

func RunTUI(result *core.ScanResult) {
	if len(result.Items) == 0 {
		return
	}

	p := tea.NewProgram(
		InitialModel(result),
		tea.WithAltScreen(),
		tea.WithANSICompressor(),
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
