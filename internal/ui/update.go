package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danorul9/sweeper/internal/actions"
	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/core"
	"github.com/danorul9/sweeper/internal/doctor"
	"github.com/danorul9/sweeper/internal/dupes"
	"github.com/danorul9/sweeper/internal/large"
	"github.com/danorul9/sweeper/internal/liveliness"
	"github.com/danorul9/sweeper/internal/scanner"
)

type scanDoneMsg struct {
	result *core.ScanResult
	err    error
}

type featureDoneMsg struct {
	screen screen
	items  []HubItem
	title  string
	total  string
	err    error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case scanDoneMsg:
		m.featureLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.results = msg.result
		m.screen = screenScan
		m.sortColumn = ""
		m.sortAsc = true
		m.cursor = 0
		m.tab = 0
		return m, nil

	case featureDoneMsg:
		m.featureLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.items = msg.items
		m.fTitle = msg.title
		m.fTotal = msg.total
		m.screen = msg.screen
		m.sortColumn = ""
		m.sortAsc = true
		m.cursor = 0
		for m.cursor < len(m.items) && (m.items[m.cursor].IsHeader || m.items[m.cursor].IsColumnHeader) {
			m.cursor++
		}
		if m.cursor >= len(m.items) {
			m.cursor = 0
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}
func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		return m.cycleSortColumn(), nil
	case "S":
		return m.toggleSortDir(), nil
	}
	switch m.screen {
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenScan:
		return m.handleScanKey(msg)
	default:
		return m.handleFeatureKey(msg)
	}
}
func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	switch m.screen {
	case screenApps, screenLarge, screenLiveliness, screenDupes, screenDoctor, screenReclaim, screenUndo, screenStats:
		return m.handleFeatureMouse(msg)
	case screenScan:
		return m.handleScanMouse(msg)
	}
	return m, nil
}

func (m model) handleFeatureMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if len(m.items) == 0 || !m.items[0].IsColumnHeader {
		return m, nil
	}

	// Layout: appStyle.Padding(1,2) = 1 top, 2 left; then: title(1) + ""(1) + body...
	// Column header (body[0]) Y position = 3 (0-indexed from terminal top)
	if msg.Y != 3 {
		return m, nil
	}

	// Use shared header layout computation — single source of truth
	h := m.items[0]
	_, name, size, age, detail := m.headerLineAndBounds(h)

	// Determine which column was clicked
	col := ""
	if size.left > 0 && msg.X >= size.left && msg.X < size.right {
		col = "size"
	} else if age.left > 0 && msg.X >= age.left && msg.X < age.right {
		col = "age"
	} else if detail.left > 0 && msg.X >= detail.left {
		col = "detail"
	} else if msg.X >= name.left && msg.X < name.right {
		col = "name"
	}

	if col == "" {
		return m, nil
	}

	// Toggle direction if same column clicked
	asc := m.sortAsc
	if m.sortColumn == col {
		asc = !asc
	}

	// Sort items
	m.sortItems(col, asc)
	m.sortColumn = col
	m.sortAsc = asc
	return m, nil
}

func (m model) handleScanMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.results == nil || len(m.results.Items) == 0 {
		return m, nil
	}

	// Column header Y: Raw content split by \n: header(0) + tab/search(1) + blank(2) + NAME header(3)
	// (appStyle top pad may not add a visible line in terminal coords)
	// Y=3 normal, Y=2 searching
	headerY := 3
	if m.searching {
		headerY = 2
	}
	if msg.Y != headerY {
		return m, nil
	}

	// Replicate scan header rendering math for X positions
	nameLabel := "NAME" + m.sortArrow("name")
	left := fmt.Sprintf("     %-32s", nameLabel)
	leftWidth := lipgloss.Width(left)
	sizePart := fmt.Sprintf("%10s", "SIZE"+m.sortArrow("size"))
	verdictLabel := "VERDICT" + m.sortArrow("verdict")

	usable := m.width - 6
	rightWidth := lipgloss.Width(sizePart) + 1 + lipgloss.Width(verdictLabel) // %10s + " " + verdict
	gap := usable - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	padLeft := 3 // appStyle(2) + groupHeaderStyle(1)
	nameEnd := padLeft + leftWidth
	sizeStart := nameEnd + gap
	sizeEnd := sizeStart + 10

	// Determine which column was clicked (check NAME first, then SIZE, then VERDICT)
	col := ""
	if msg.X >= padLeft && msg.X < nameEnd {
		col = "name"
	} else if msg.X >= sizeStart && msg.X < sizeEnd {
		col = "size"
	} else if msg.X >= sizeEnd+1 {
		col = "verdict"
	}

	if col == "" {
		return m, nil
	}

	// Toggle direction if same column clicked
	asc := m.sortAsc
	if m.sortColumn == col {
		asc = !asc
	}

	m.sortScanItems(col, asc)
	m.sortColumn = col
	m.sortAsc = asc
	m.cursor = 0
	return m, nil
}

func (m model) sortItems(col string, asc bool) {
	if len(m.items) < 2 {
		return
	}

	// Check if this screen has section headers (Duplicates, Doctor, Stats)
	hasSections := false
	for i := 1; i < len(m.items); i++ {
		if m.items[i].IsHeader {
			hasSections = true
			break
		}
	}

	if hasSections {
		// Sort within each section, preserving section header positions.
		// Pattern: [column header, section header, data rows, section header, data rows, ...]
		start := 0
		for i := 1; i < len(m.items); i++ {
			if m.items[i].IsHeader {
				if start > 0 {
					m.sortSlice(m.items[start:i], col, asc)
				}
				start = i + 1
			}
		}
		// Last block
		if start > 0 && start < len(m.items) {
			m.sortSlice(m.items[start:], col, asc)
		}
	} else {
		// Flat screen: sort all data rows (skip column header at index 0)
		m.sortSlice(m.items[1:], col, asc)
	}
}

func (m model) sortSlice(data []HubItem, col string, asc bool) {
	sort.SliceStable(data, func(i, j int) bool {
		var less bool
		switch col {
		case "name":
			less = data[i].Name < data[j].Name
		case "size":
			less = data[i].Size < data[j].Size
		case "age":
			less = data[i].AgeDays < data[j].AgeDays
		case "detail":
			less = data[i].Detail < data[j].Detail
		default:
			return false
		}
		if !asc {
			less = !less
		}
		return less
	})
}

// sortableColumns returns the ordered list of sortable columns for the current screen.
// For feature screens: derived from the column header HubItem conditions.
// For scan screen: fixed set of columns.
func (m model) sortableColumns() []string {
	cols := []string{"name"}
	if len(m.items) > 0 && m.items[0].IsColumnHeader {
		h := m.items[0]
		if h.Size > 0 {
			cols = append(cols, "size")
		}
		if h.AgeDays >= 0 {
			cols = append(cols, "age")
		}
		if h.Detail != "" {
			cols = append(cols, "detail")
		}
		return cols
	}
	if m.screen == screenScan {
		return []string{"name", "size", "verdict"}
	}
	return cols
}

// sortScanItems sorts m.results.Items by the given column and direction.
func (m model) sortScanItems(col string, asc bool) {
	sort.SliceStable(m.results.Items, func(i, j int) bool {
		var less bool
		switch col {
		case "name":
			less = m.results.Items[i].Name < m.results.Items[j].Name
		case "size":
			less = m.results.Items[i].Size < m.results.Items[j].Size
		case "verdict":
			// Sort by confidence descending by default (higher = more certain of verdict)
			ci, cj := 0.0, 0.0
			if m.results.Items[i].Match != nil {
				ci = m.results.Items[i].Match.Confidence
			}
			if m.results.Items[j].Match != nil {
				cj = m.results.Items[j].Match.Confidence
			}
			less = ci < cj
		default:
			return false
		}
		if !asc {
			less = !less
		}
		return less
	})
}

// cycleSortColumn advances m.sortColumn to the next available column and sorts.
func (m model) cycleSortColumn() model {
	cols := m.sortableColumns()
	if len(cols) == 0 {
		return m
	}
	nextCol := cols[0]
	for i, c := range cols {
		if c == m.sortColumn {
			nextCol = cols[(i+1)%len(cols)]
			break
		}
	}
	m.sortColumn = nextCol
	m.sortAsc = true
	m.applySort()
	if m.screen == screenScan {
		m.cursor = 0
	} else {
		m.cursor = 0
		for m.cursor < len(m.items) && (m.items[m.cursor].IsHeader || m.items[m.cursor].IsColumnHeader) {
			m.cursor++
		}
	}
	return m
}

// toggleSortDir reverses the current sort direction. Sets to "name" ascending if no column is active.
func (m model) toggleSortDir() model {
	if m.sortColumn == "" {
		m.sortColumn = "name"
	}
	m.sortAsc = !m.sortAsc
	m.applySort()
	return m
}

// applySort delegates to the correct sort implementation based on screen.
func (m model) applySort() {
	if m.screen == screenScan {
		m.sortScanItems(m.sortColumn, m.sortAsc)
	} else {
		m.sortItems(m.sortColumn, m.sortAsc)
	}
}

func (m model) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.toast = ""

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}

	case "down", "j":
		if m.menuCursor < len(menuItems)-1 {
			m.menuCursor++
		}

	case "enter", " ":
		mi := menuItems[m.menuCursor]
		return m.runFeature(mi.screen)
	}

	return m, nil
}

func (m model) handleScanKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case "esc":
		m.screen = screenMenu
		m.menuCursor = 0
		return m, nil

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
	case "n":
		for i := range m.results.Items {
			m.selected[i] = false
		}
	}

	return m, nil
}

func (m model) handleFeatureKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.fConfirmDel {
		return m.handleFeatureConfirmKey(msg)
	}
	if m.fDeleting {
		return m, tea.Quit
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.screen = screenMenu
		m.menuCursor = 0
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			for m.cursor > 0 && (m.items[m.cursor].IsHeader || m.items[m.cursor].IsColumnHeader) {
				m.cursor--
			}
		}

	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			for m.cursor < len(m.items)-1 && (m.items[m.cursor].IsHeader || m.items[m.cursor].IsColumnHeader) {
				m.cursor++
			}
		}

	case " ", "a", "n", "d":
		switch msg.String() {
		case " ":
			if len(m.items) > 0 && m.cursor < len(m.items) && !m.items[m.cursor].IsHeader && !m.items[m.cursor].IsColumnHeader {
				m.fSelected[m.cursor] = !m.fSelected[m.cursor]
			}
		case "a":
			for i := range m.items {
				if !m.items[i].IsHeader && !m.items[i].IsColumnHeader {
					m.fSelected[i] = true
				}
			}
		case "n":
			for i := range m.items {
				m.fSelected[i] = false
			}
		case "d":
			if m.fTotalSelected() > 0 {
				m.fConfirmDel = true
			}
		}
	}

	return m, nil
}

func (m model) handleFeatureConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.executeFeatureDelete()
	case "n", "N", "esc", "q":
		m.fConfirmDel = false
		return m, nil
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
		return m.handleScanKey(msg)
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) runFeature(s screen) (tea.Model, tea.Cmd) {
	m.featureLoading = true

	switch s {
	case screenApps:
		return m, m.runDetectedApps()
	case screenScan:
		return m, m.runScan()
	case screenLarge:
		return m, m.runLarge()
	case screenDupes:
		return m, m.runDupes()
	case screenDoctor:
		return m, m.runDoctor()
	case screenReclaim:
		return m, m.runReclaim()
	case screenUndo:
		return m, m.runUndo()
	case screenStats:
		return m, m.runStats()
	case screenLiveliness:
		return m, m.runLiveliness()
	}
	return m, nil
}

func (m model) runDetectedApps() tea.Cmd {
	return func() tea.Msg {
		apps := appindex.ScanAllApplications()
		var items []HubItem
		items = append(items, HubItem{
			Name:           "APP",
			Size:           1,
			Detail:         "LOCATION      ",
			AgeDays:        0,
			IsColumnHeader: true,
		})
		var totalSize int64

		for _, ap := range apps {
			info, err := appindex.ReadAppInfo(ap)
			name := appindex.AppNameFromPath(ap)
			bid := ""
			if err == nil {
				if info.Name != "" {
					name = info.Name
				}
				bid = info.BundleID
			}

			if appindex.IsSystemApp(name, bid, ap) {
				continue
			}

			loc := "/Applications"
			if homeLoc, ok := findHome(ap); ok {
				loc = homeLoc
			}

			size := appSize(ap)
			totalSize += size

			detail := loc

			items = append(items, HubItem{
				Name: name,
				Path: ap,
				Size: size,
				Detail: detail,
				AgeDays: computeAgeDays(ap),
			})
		}

		title := fmt.Sprintf("Detected Apps \u2014 %d user-installed  %s",
			len(items), formatBytes(totalSize))
		if n := len(appindex.KnownAppNames); n > 0 {
			title += fmt.Sprintf("  (%d known apps)", n)
		}

		return featureDoneMsg{
			screen: screenApps,
			items:  items,
			title:  title,
		}
	}

}

func appSize(ap string) int64 {
	var total int64
	filepath.WalkDir(ap, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, e := d.Info()
			if e == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

func findHome(ap string) (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	prefix := home + "/Applications"
	if len(ap) >= len(prefix) && ap[:len(prefix)] == prefix {
		return "~" + ap[len(home):], true
	}
	return "", false
}

func (m model) runScan() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return scanDoneMsg{err: fmt.Errorf("load config: %w", err)}
		}

		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return scanDoneMsg{err: fmt.Errorf("build app index: %w", err)}
		}

		s := scanner.New(cfg, config.ModeAggressive)
		s.SetIndex(idx)

		result, err := s.Scan()
		if err != nil {
			return scanDoneMsg{err: err}
		}
		result.AppCount = len(idx.Names)
		result.KnownAppsCount = len(appindex.KnownAppNames)

		return scanDoneMsg{result: result}
	}
}

func (m model) runLarge() tea.Cmd {
	return func() tea.Msg {
		files, err := large.Scan(100 * 1024 * 1024)
		if err != nil {
			return featureDoneMsg{screen: screenLarge, err: err}
		}

		var items []HubItem
		items = append(items, HubItem{
			Name:           "NAME",
			Size:           1,
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		var total int64
		for _, f := range files {
			items = append(items, HubItem{
				Name: filepath.Base(f.Path),
				Path: f.Path,
				Size: f.Size,
				AgeDays: -1,
			})
			total += f.Size
		}

		return featureDoneMsg{
			screen: screenLarge,
			items:  items,
			title:  fmt.Sprintf("Large Files \u2014 %d items  %s", len(items), formatBytes(total)),
			total:  fmt.Sprintf("Total: %s", formatBytes(total)),
		}
	}
}

func (m model) runDupes() tea.Cmd {
	return func() tea.Msg {
		groups, err := dupes.Find(false)
		if err != nil {
			return featureDoneMsg{screen: screenDupes, err: err}
		}
		var items []HubItem
		items = append(items, HubItem{
			Name:           "NAME",
			Size:           1,
			Detail:         "MODIFIED",
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		var totalWaste int64
		for _, g := range groups {
			waste := g.TotalSize - g.Size
			totalWaste += waste
			// Group header
			items = append(items, HubItem{
				Name:     fmt.Sprintf("SHA:%s — %d copies, %s wasted", g.SHA256[:12], g.Count, formatBytes(waste)),
				IsHeader: true,
			})
			// Sub-item per file copy
			for _, f := range g.Files {
				items = append(items, HubItem{
					Name:   filepath.Base(f.Path),
					Path:   f.Path,
					Size:   f.Size,
					Detail: f.ModTime,
				AgeDays: -1,
				})
			}
		}
		return featureDoneMsg{
			screen: screenDupes,
			items:  items,
			title:  fmt.Sprintf("Duplicates \u2014 %d groups  %s reclaimable", len(groups), formatBytes(totalWaste)),
			total:  fmt.Sprintf("Total: %s reclaimable", formatBytes(totalWaste)),
		}
	}
}

func (m model) runDoctor() tea.Cmd {
	return func() tea.Msg {
		issues, err := doctor.Run()
		if err != nil {
			return featureDoneMsg{screen: screenDoctor, err: err}
		}

		var items []HubItem
		items = append(items, HubItem{
			Name:           "ISSUE",
			Detail:         "DESCRIPTION",
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		sevCounts := map[string]int{}
		for _, iss := range issues {
			sevCounts[iss.Severity]++
		}
		sevOrder := map[string]int{"error": 0, "warning": 1, "info": 2}
		sort.SliceStable(issues, func(i, j int) bool {
			return sevOrder[issues[i].Severity] < sevOrder[issues[j].Severity]
		})
		var lastSev string
		for _, iss := range issues {
			if iss.Severity != lastSev {
				label := strings.ToUpper(iss.Severity) + "S"
				items = append(items, HubItem{
					Name:     fmt.Sprintf("%s — %d issues", label, sevCounts[iss.Severity]),
					IsHeader: true,
				})
				lastSev = iss.Severity
			}
			items = append(items, HubItem{
				Name:   iss.Category,
				Path:   iss.Path,
				Detail: iss.Description,
				AgeDays: -1,
			})
		}

		title := fmt.Sprintf("Doctor \u2014 %d issues", len(issues))
		if n := sevCounts["error"]; n > 0 {
			title += fmt.Sprintf("  (%d errors)", n)
		}

		return featureDoneMsg{
			screen: screenDoctor,
			items:  items,
			title:  title,
		}
	}
}

func (m model) runReclaim() tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return featureDoneMsg{screen: screenReclaim, err: fmt.Errorf("load config: %w", err)}
		}

		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return featureDoneMsg{screen: screenReclaim, err: fmt.Errorf("build app index: %w", err)}
		}

		s := scanner.New(cfg, config.ModeReclaim)
		s.SetIndex(idx)

		result, err := s.Scan()
		if err != nil {
			return featureDoneMsg{screen: screenReclaim, err: err}
		}

		var items []HubItem
		items = append(items, HubItem{
			Name:           "NAME",
			Size:           1,
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		var total int64
		for _, item := range result.Items {
			items = append(items, HubItem{
				Name: item.Name,
				Path: item.Path,
				Size: item.Size,
				AgeDays: -1,
			})
			total += item.Size
		}

		return featureDoneMsg{
			screen: screenReclaim,
			items:  items,
			title:  fmt.Sprintf("Reclaim \u2014 %d items  %s", len(items), formatBytes(total)),
			total:  fmt.Sprintf("Total: %s", formatBytes(total)),
		}
	}
}

func (m model) runUndo() tea.Cmd {
	return func() tea.Msg {
		snapshot, err := actions.LoadLatestSnapshot()
		if err != nil {
			return featureDoneMsg{screen: screenUndo, err: err}
		}
		if snapshot == nil {
			return featureDoneMsg{
				screen: screenUndo,
				items:  []HubItem{},
				title:  "Undo \u2014 No snapshots found",
			}
		}

		var items []HubItem
		items = append(items, HubItem{
			Name:           "FILE",
			Size:           1,
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		var total int64
		for _, si := range snapshot.Items {
			items = append(items, HubItem{
				Name: filepath.Base(si.Path),
				Path: si.Path,
				Size: si.Size,
				AgeDays: -1,
			})
			total += si.Size
		}

		return featureDoneMsg{
			screen: screenUndo,
			items:  items,
			title:  fmt.Sprintf("Undo \u2014 Snapshot from %s  (%d items, %s)", snapshot.Timestamp.Format("Jan 2, 2006 15:04"), len(items), formatBytes(total)),
			total:  fmt.Sprintf("Total: %s", formatBytes(total)),
		}
	}
}

func (m model) runStats() tea.Cmd {
	return func() tea.Msg {
		stats, err := actions.LoadStats()
		if err != nil {
			return featureDoneMsg{screen: screenStats, err: err}
		}

		var items []HubItem
		items = append(items, HubItem{
			Name:           "METRIC",
			Detail:         "VALUE",
			AgeDays:        -1,
			IsColumnHeader: true,
		})
		if stats.TotalScans > 0 || stats.TotalDeletes > 0 {
			items = append(items, HubItem{
				Name:     "Total scans",
				Detail:   fmt.Sprintf("%d", stats.TotalScans),
				AgeDays:  -1,
			})
			items = append(items, HubItem{
				Name:     "Total deletes",
				Detail:   fmt.Sprintf("%d", stats.TotalDeletes),
				AgeDays:  -1,
			})
			items = append(items, HubItem{
				Name:     "Successful",
				Detail:   fmt.Sprintf("%d", stats.TotalSuccess),
				AgeDays:  -1,
			})
			items = append(items, HubItem{
				Name:     "Failed",
				Detail:   fmt.Sprintf("%d", stats.TotalFail),
				AgeDays:  -1,
			})
			items = append(items, HubItem{
				Name:     "Space freed",
				Detail:   formatBytes(stats.TotalSize),
				AgeDays:  -1,
			})
			if !stats.FirstScan.IsZero() {
			items = append(items, HubItem{
				Name:     "First scan",
				Detail:   stats.FirstScan.Format("Jan 2, 2006 15:04"),
				AgeDays:  -1,
			})
			}
			if !stats.LastScan.IsZero() {
			items = append(items, HubItem{
				Name:     "Last scan",
				Detail:   stats.LastScan.Format("Jan 2, 2006 15:04"),
				AgeDays:  -1,
			})
			}
			if len(stats.RecentEntries) > 0 {
				items = append(items, HubItem{Name: "Recent Activity", IsHeader: true})
				for _, e := range stats.RecentEntries {
					status := "OK"
					if !e.Success {
						status = "FAIL"
					}
					items = append(items, HubItem{
						Name: e.Timestamp.Format("Jan 2 15:04"),
						Detail: fmt.Sprintf("[%s] %s  items=%d  size=%s",
							status, e.Action, e.Items, formatBytes(e.Size)),
						AgeDays: -1,
					})
				}
			}
		}

		return featureDoneMsg{
			screen: screenStats,
			items:  items,
			title:  fmt.Sprintf("Stats \u2014 %d entries", len(items)),
		}
	}
}

func (m model) runLiveliness() tea.Cmd {
	return func() tea.Msg {
		idx, err := appindex.BuildOrLoadCached()
		if err != nil {
			return featureDoneMsg{screen: screenLiveliness, err: fmt.Errorf("build app index: %w", err)}
		}

		items, err := liveliness.Run(idx)
		if err != nil {
			return featureDoneMsg{screen: screenLiveliness, err: err}
		}

		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Dead != items[j].Dead {
				return items[i].Dead
			}
			return items[i].Score > items[j].Score
		})

		var nDead, nCold int
		var totalSize int64
		for _, item := range items {
			if item.Dead {
				nDead++
			} else {
				nCold++
			}
			totalSize += item.Size
		}

		var hubItems []HubItem

		// Column header — rendered as a styled header bar at the top
		hubItems = append(hubItems, HubItem{
			Name:           "NAME",
			Size:           1,
			Detail:         "SCORE",
			AgeDays:        0,
			IsColumnHeader: true,
		})

		for i, item := range items {
			if i == 0 || item.Dead != items[i-1].Dead {
				label := "DEAD"
				count := nDead
				if !item.Dead {
					label = "COLD"
					count = nCold
				}
				hubItems = append(hubItems, HubItem{
					Name:     fmt.Sprintf("%s — %d items", label, count),
					IsHeader: true,
				})
			}

			scorePct := fmt.Sprintf("%.0f%%", item.Score*100)
			label := "COLD"
			if item.Dead {
				label = "DEAD"
			}

			// Build info rows for the bottom panel
			var infoRows []string
			infoRows = append(infoRows, fmt.Sprintf("Name: %s", item.Name))
			infoRows = append(infoRows, fmt.Sprintf("Path: %s", item.Path))
			infoRows = append(infoRows, fmt.Sprintf("Size: %s  |  Score: %.0f%%  |  %s", formatBytes(item.Size), item.Score*100, label))

			// 3 biggest files inside
			topFiles := biggestFiles(item.Path, 3)
			if len(topFiles) > 0 {
				infoRows = append(infoRows, "Biggest files:")
				for _, f := range topFiles {
					infoRows = append(infoRows, fmt.Sprintf("  %s  (%s)", f.Name, formatBytes(f.Size)))
				}
			}

			// Quick directory stats
			fileCount, dirCount, newest := quickDirStats(item.Path)
			if fileCount >= 0 {
				infoRows = append(infoRows, fmt.Sprintf("Contents: %d files, %d subdirs", fileCount, dirCount))
				if !newest.IsZero() {
					age := time.Since(newest)
					days := int(age.Hours() / 24)
					if days >= 365 {
						infoRows = append(infoRows, fmt.Sprintf("Newest file: %dy old", days/365))
					} else if days >= 30 {
						infoRows = append(infoRows, fmt.Sprintf("Newest file: %dm old", days/30))
					} else {
						infoRows = append(infoRows, fmt.Sprintf("Newest file: %dd old", days))
					}
				}
			}

			hubItems = append(hubItems, HubItem{
				Name:     item.Name,
				Path:     item.Path,
				Size:     item.Size,
				Detail:   scorePct,
				InfoRows: infoRows,
				AgeDays:  computeAgeDays(item.Path),
			})
		}

		title := fmt.Sprintf("Liveliness \u2014 %d DEAD + %d COLD  %s", nDead, nCold, formatBytes(totalSize))
		if len(hubItems) == 0 {
			title = "Liveliness \u2014 no dead/cold items found"
		}

		return featureDoneMsg{
			screen: screenLiveliness,
			items:  hubItems,
			title:  title,
			total:  fmt.Sprintf("Total: %s", formatBytes(totalSize)),
		}
	}
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
		m.toast = "Error saving snapshot: " + err.Error()
		return m.backToMenu(), nil
	}

	if err := actions.Trash(paths...); err != nil {
		m.toast = "Error deleting: " + err.Error()
		return m.backToMenu(), nil
	}

	actions.LogOperation(actions.OperationLog{
		Timestamp: snapshot.Timestamp,
		Action:    "delete",
		Paths:     paths,
		TotalSize: m.selectedSize(),
		Success:   true,
	})

	m.toast = fmt.Sprintf("Deleted %d items (%s)", len(paths), formatBytes(m.selectedSize()))
	return m.backToMenu(), nil
}

func (m model) backToMenu() model {
	m.screen = screenMenu
	m.menuCursor = 0
	m.confirmDelete = false
	m.deleting = false
	m.done = false
	m.fConfirmDel = false
	m.fDeleting = false
	m.fDeleted = false
	m.selected = make(map[int]bool)
	m.fSelected = make(map[int]bool)
	return m
}

func (m model) executeFeatureDelete() (tea.Model, tea.Cmd) {
	m.fDeleting = true

	var paths []string
	var items []core.Leftover
	for i := range m.items {
		if !m.fSelected[i] || m.items[i].IsHeader || m.items[i].IsColumnHeader {
			continue
		}
		item := m.items[i]
		paths = append(paths, item.Path)
		items = append(items, core.Leftover{Path: item.Path, Name: item.Name, Size: item.Size})
	}

	snapshot, err := actions.SaveSnapshot(items)
	if err != nil {
		m.toast = "Error saving snapshot: " + err.Error()
		return m.backToMenu(), nil
	}

	if err := actions.Trash(paths...); err != nil {
		m.toast = "Error deleting: " + err.Error()
		return m.backToMenu(), nil
	}

	actions.LogOperation(actions.OperationLog{
		Timestamp: snapshot.Timestamp,
		Action:    "delete",
		Paths:     paths,
		TotalSize: m.fSelectedSize(),
		Success:   true,
	})

	m.toast = fmt.Sprintf("Deleted %d items (%s)", len(paths), formatBytes(m.fSelectedSize()))
	return m.backToMenu(), nil
}

// quickDirStats walks a directory to count files and subdirs and find newest file mod time.
// Returns -1 for fileCount on error.
func quickDirStats(path string) (fileCount, dirCount int, newest time.Time) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return -1, 0, time.Time{}
	}
	for _, e := range entries {
		if e.IsDir() {
			// Recurse one level for subdirs
			subs, _ := os.ReadDir(filepath.Join(path, e.Name()))
			for _, sub := range subs {
				info, err := sub.Info()
				if err != nil {
					continue
				}
				if info.ModTime().After(newest) {
					newest = info.ModTime()
				}
				if sub.IsDir() {
					dirCount++
				} else {
					fileCount++
				}
			}
			dirCount++
		} else {
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
			fileCount++
		}
	}
	return
}

type fileInfo struct {
	Name string
	Size int64
}

// biggestFiles returns the N largest files in a directory (non-recursive).
func biggestFiles(path string, n int) []fileInfo {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{Name: e.Name(), Size: info.Size()})
	}
	// Sort by size descending
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
	if len(files) > n {
		files = files[:n]
	}
	return files
}

// computeAgeDays returns the age of a folder's newest content in days.
// Returns -1 if unknown.
func computeAgeDays(path string) int {
	info, err := os.Stat(path)
	if err != nil {
		return -1
	}
	age := time.Since(info.ModTime())
	days := int(age.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func RunTUI(result *core.ScanResult) {
	m := InitialScanModel(result)
	if len(result.Items) == 0 {
		return
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithANSICompressor(),
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}

func RunHub() {
	m := InitialModel()

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithANSICompressor(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
