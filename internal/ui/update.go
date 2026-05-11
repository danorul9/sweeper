package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		m.cursor = 0
		for m.cursor < len(m.items) && m.items[m.cursor].IsHeader {
			m.cursor++
		}
		if m.cursor >= len(m.items) {
			m.cursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMenu:
		return m.handleMenuKey(msg)
	case screenScan:
		return m.handleScanKey(msg)
	default:
		return m.handleFeatureKey(msg)
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
			for m.cursor > 0 && m.items[m.cursor].IsHeader {
				m.cursor--
			}
		}

	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
			for m.cursor < len(m.items)-1 && m.items[m.cursor].IsHeader {
				m.cursor++
			}
		}

	case " ", "a", "n", "d":
		switch msg.String() {
		case " ":
			if len(m.items) > 0 && m.cursor < len(m.items) && !m.items[m.cursor].IsHeader {
				m.fSelected[m.cursor] = !m.fSelected[m.cursor]
			}
		case "a":
			for i := range m.items {
				if !m.items[i].IsHeader {
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

			detail := bid
			if detail == "" {
				detail = loc
			} else {
				detail = bid + "  ·  " + loc
			}

			items = append(items, HubItem{
				Name: name,
				Path: ap,
				Size: size,
				Detail: detail,
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
		var total int64
		for _, f := range files {
			items = append(items, HubItem{
				Name: filepath.Base(f.Path),
				Path: f.Path,
				Size: f.Size,
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
		var totalWaste int64
		for _, g := range groups {
			waste := g.TotalSize - g.Size
			totalWaste += waste
			// first file as the main item
			items = append(items, HubItem{
				Name:    g.SHA256[:12],
				Path:    g.Files[0].Path,
				Size:    g.Size,
				Detail:  fmt.Sprintf("%d copies, %s wasted", g.Count, formatBytes(waste)),
				Signals: mapFilesToStrings(g.Files),
			})
		}

		return featureDoneMsg{
			screen: screenDupes,
			items:  items,
			title:  fmt.Sprintf("Duplicates \u2014 %d groups  %s reclaimable", len(items), formatBytes(totalWaste)),
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
		sevCounts := map[string]int{}
		for _, iss := range issues {
			sevCounts[iss.Severity]++
			items = append(items, HubItem{
				Name:   iss.Category,
				Path:   iss.Path,
				Detail: fmt.Sprintf("[%s] %s", iss.Severity, iss.Description),
			})
		}

		title := fmt.Sprintf("Doctor \u2014 %d issues", len(items))
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
		var total int64
		for _, item := range result.Items {
			items = append(items, HubItem{
				Name: item.Name,
				Path: item.Path,
				Size: item.Size,
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
		var total int64
		for _, si := range snapshot.Items {
			items = append(items, HubItem{
				Name: filepath.Base(si.Path),
				Path: si.Path,
				Size: si.Size,
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
		if stats.TotalScans > 0 || stats.TotalDeletes > 0 {
			items = append(items, HubItem{Name: "Total scans", Detail: fmt.Sprintf("%d", stats.TotalScans)})
			items = append(items, HubItem{Name: "Total deletes", Detail: fmt.Sprintf("%d", stats.TotalDeletes)})
			items = append(items, HubItem{Name: "Successful", Detail: fmt.Sprintf("%d", stats.TotalSuccess)})
			items = append(items, HubItem{Name: "Failed", Detail: fmt.Sprintf("%d", stats.TotalFail)})
			items = append(items, HubItem{Name: "Space freed", Detail: formatBytes(stats.TotalSize)})
			if !stats.FirstScan.IsZero() {
				items = append(items, HubItem{Name: "First scan", Detail: stats.FirstScan.Format("Jan 2, 2006 15:04")})
			}
			if !stats.LastScan.IsZero() {
				items = append(items, HubItem{Name: "Last scan", Detail: stats.LastScan.Format("Jan 2, 2006 15:04")})
			}
			if len(stats.RecentEntries) > 0 {
				items = append(items, HubItem{Name: "", Detail: ""})
				items = append(items, HubItem{Name: "Recent Activity", Detail: ""})
				for _, e := range stats.RecentEntries {
					status := "OK"
					if !e.Success {
						status = "FAIL"
					}
					items = append(items, HubItem{
						Name: e.Timestamp.Format("Jan 2 15:04"),
						Detail: fmt.Sprintf("[%s] %s  items=%d  size=%s",
							status, e.Action, e.Items, formatBytes(e.Size)),
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
		if !m.fSelected[i] || m.items[i].IsHeader {
			continue
		}
		item := m.items[i]
		if m.screen == screenDupes {
			for _, p := range item.Signals {
				paths = append(paths, p)
				items = append(items, core.Leftover{Path: p, Name: filepath.Base(p), Size: item.Size})
			}
		} else {
			paths = append(paths, item.Path)
			items = append(items, core.Leftover{Path: item.Path, Name: item.Name, Size: item.Size})
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
		TotalSize: m.fSelectedSize(),
		Success:   true,
	})

	m.toast = fmt.Sprintf("Deleted %d items (%s)", len(paths), formatBytes(m.fSelectedSize()))
	return m.backToMenu(), nil
}

func mapFilesToStrings(files []dupes.File) []string {
	var s []string
	for _, f := range files {
		s = append(s, f.Path)
	}
	return s
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
	)

	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}
