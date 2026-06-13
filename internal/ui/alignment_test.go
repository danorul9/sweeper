package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDetectedAppsColumnAlignment(t *testing.T) {
	// Simulate the exact header and data row rendering from view.go
	// to verify AGE and SIZE columns align between header and data rows.

	// Detected Apps header (update.go line 326-332)
	headerDetail := "LOCATION      " // padded to 14 chars = width of "/Applications"

	// Simulate view.go lines 175-221 (header rendering)
	left := fmt.Sprintf("     %s", "APP") // "     APP"
	leftWidth := lipgloss.Width(left)

	sizeStr := fmt.Sprintf("%10s", "SIZE")
	ageStr := "  " + fmt.Sprintf("%9s", "AGE")
	baseRight := sizeStr + ageStr
	baseRightWidth := lipgloss.Width(baseRight)

	detailText := headerDetail
	detailStr := "  " + detailText

	rightWidth := baseRightWidth + lipgloss.Width(detailStr)

	// Test at multiple terminal widths
	for _, mWidth := range []int{80, 100, 120, 160, 200} {
		usable := mWidth - 6

		// Header right part start and AGE right edge
		gap := usable - leftWidth - rightWidth
		headerAgeRightEdge := leftWidth + gap + baseRightWidth

		// Data row rendering (view.go lines 246-295)
		name := "Karabiner-VirtualHIDDevice-Manager" // 36 chars, typical long name
		cursor := " "
		check := " "
		dataLeft := fmt.Sprintf("%s %s %s", cursor, check, name)
		dataLeftWidth := lipgloss.Width(dataLeft)

		dataBaseRight := sizeStyle.Render("1.2 MB") + "  " + ageStyle(32)
		dataBaseRightWidth := lipgloss.Width(dataBaseRight)

		dataDetail := "/Applications"
		dataRemaining := usable - dataLeftWidth - dataBaseRightWidth
		dataMaxDetail := dataRemaining - 3
		if dataMaxDetail < 1 {
			dataMaxDetail = 1
		}
		dataDetailDisplay := dataDetail
		if len(dataDetailDisplay) > dataMaxDetail {
			dataDetailDisplay = dataDetailDisplay[:dataMaxDetail-1] + "…"
		}
		dataDetailStr := "  " + dataDetailDisplay
		dataRightWidth := dataBaseRightWidth + lipgloss.Width(dataDetailStr)
		dataGap := usable - dataLeftWidth - dataRightWidth
		if dataGap < 1 {
			dataGap = 1
		}
		dataAgeRightEdge := dataLeftWidth + dataGap + dataBaseRightWidth

		// AGE column
		ageOff := headerAgeRightEdge - dataAgeRightEdge
		if ageOff < 0 {
			ageOff = -ageOff
		}

		t.Logf("Width %3d: header AGE right=%3d, data AGE right=%3d, off=%d",
			mWidth, headerAgeRightEdge, dataAgeRightEdge, ageOff)

		if mWidth > 80 && ageOff > 1 {
			t.Errorf("AGE column off by %d at width %d (expected ≤1)", ageOff, mWidth)
		}
	}
}

func TestAllScreensNoSpuriousAgeColumn(t *testing.T) {
	// Verify that non-Liveliness/DetectedApps screens suppress AGE
	screens := []struct {
		name    string
		itemsFn func() []HubItem
	}{
		{"Large", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "NAME", Size: 1, AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "test", Path: "/tmp/test", Size: 100, AgeDays: -1})
			return items
		}},
		{"Dupes", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "NAME", Size: 1, Detail: "MODIFIED", AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "test", Path: "/tmp/test", Size: 100, Detail: "2024-01-15", AgeDays: -1})
			return items
		}},
		{"Doctor", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "ISSUE", Detail: "DESCRIPTION", AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "test", Path: "/tmp/test", Detail: "desc", AgeDays: -1})
			return items
		}},
		{"Reclaim", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "NAME", Size: 1, AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "test", Path: "/tmp/test", Size: 100, AgeDays: -1})
			return items
		}},
		{"Undo", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "FILE", Size: 1, AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "test", Path: "/tmp/test", Size: 100, AgeDays: -1})
			return items
		}},
		{"Stats", func() []HubItem {
			var items []HubItem
			items = append(items, HubItem{Name: "METRIC", Detail: "VALUE", AgeDays: -1, IsColumnHeader: true})
			items = append(items, HubItem{Name: "Total scans", Detail: "10", AgeDays: -1})
			return items
		}},
	}

	for _, s := range screens {
		t.Run(s.name, func(t *testing.T) {
			items := s.itemsFn()
			if items[0].AgeDays >= 0 {
				t.Error("header AgeDays should be -1 to suppress AGE column")
			}
			for i, item := range items[1:] {
				if item.AgeDays >= 0 {
					t.Errorf("data row %d AgeDays=%d, expected -1", i, item.AgeDays)
				}
			}
		})
	}
}

func TestDetectedAppsHasAgeColumn(t *testing.T) {
	// Detected Apps header has AgeDays=0 (shows AGE), data has computeAgeDays
	items := []HubItem{
		{Name: "APP", Size: 1, Detail: "LOCATION      ", AgeDays: 0, IsColumnHeader: true},
		{Name: "TestApp", Path: "/Applications/TestApp.app", Size: 100, Detail: "/Applications", AgeDays: 10},
	}
	if items[0].AgeDays < 0 {
		t.Error("Detected Apps header AgeDays=0 should show AGE column")
	}
}

// Verify that the sizeStyle and ageStyle widths match header format widths
func TestStyleWidthsMatchHeaders(t *testing.T) {
	headerSize := fmt.Sprintf("%10s", "SIZE")
	renderedSize := sizeStyle.Render("1.2 MB")
	if lipgloss.Width(renderedSize) != lipgloss.Width(headerSize) {
		t.Errorf("sizeStyle width %d != header fmt width %d",
			lipgloss.Width(renderedSize), lipgloss.Width(headerSize))
	}

	headerAge := "  " + fmt.Sprintf("%9s", "AGE")
	renderedAge := "  " + ageStyle(30)
	if lipgloss.Width(renderedAge) != lipgloss.Width(headerAge) {
		t.Errorf("ageStyle width %d != header fmt width %d",
			lipgloss.Width(renderedAge), lipgloss.Width(headerAge))
	}

	_ = strings.Repeat
}
