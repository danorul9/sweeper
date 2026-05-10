package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/config"
	"github.com/danorul9/sweeper/internal/core"
)

func TestListFolders(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub1"), 0755)
	os.MkdirAll(filepath.Join(dir, "sub2"), 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644)

	folders, err := ListFolders(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(folders) != 2 {
		t.Errorf("expected 2 folders, got %d", len(folders))
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file1.bin"), make([]byte, 100), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "file2.bin"), make([]byte, 200), 0644)

	size := DirSize(dir)
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

func TestDirSizeEmpty(t *testing.T) {
	dir := t.TempDir()
	size := DirSize(dir)
	if size != 0 {
		t.Errorf("expected 0 for empty dir, got %d", size)
	}
}

func TestSafeLocations(t *testing.T) {
	locs := SafeLocations()
	if len(locs) == 0 {
		t.Fatal("expected locations")
	}

	types := make(map[core.LocationType]bool)
	for _, l := range locs {
		if types[l.Type] {
			t.Errorf("duplicate location type: %s", l.Type)
		}
		types[l.Type] = true
	}

	if !types[core.LocCaches] {
		t.Error("expected Caches in safe locations")
	}
	if types[core.LocContainers] {
		t.Error("did not expect Containers in safe locations")
	}
}

func TestAggressiveLocations(t *testing.T) {
	locs := AggressiveLocations()
	if len(locs) <= len(SafeLocations()) {
		t.Error("expected aggressive to have more locations than safe")
	}
}

func TestReclaimLocations(t *testing.T) {
	locs := ReclaimLocations()
	for _, l := range locs {
		if l.Type == core.LocAppSupport {
			t.Error("expected no Application Support in reclaim mode")
		}
	}
}

func TestShouldSkipFolder(t *testing.T) {
	if !shouldSkipFolder(".git") {
		t.Error("expected .git to be skipped")
	}
	if shouldSkipFolder("Docker") {
		t.Error("expected Docker not to be skipped")
	}
}

func TestScannerNew(t *testing.T) {
	cfg := &config.Config{SafeMode: true}
	s := New(cfg, config.ModeSafe)
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestScannerSetIndex(t *testing.T) {
	cfg := &config.Config{}
	s := New(cfg, config.ModeSafe)
	idx := appindex.NewAppIndex()
	s.SetIndex(idx)
}

func TestVerdictString(t *testing.T) {
	if core.VerdictInstalled.String() != "INSTALLED" {
		t.Errorf("expected INSTALLED, got %s", core.VerdictInstalled.String())
	}
	if core.VerdictLeftover.String() != "LEFTOVER" {
		t.Errorf("expected LEFTOVER, got %s", core.VerdictLeftover.String())
	}
	if core.VerdictUncertain.String() != "UNCERTAIN" {
		t.Errorf("expected UNCERTAIN, got %s", core.VerdictUncertain.String())
	}
}

func TestScannerShouldIgnore(t *testing.T) {
	cfg := &config.Config{
		Ignore: []string{"Google", "Adobe*"},
	}
	s := New(cfg, config.ModeSafe)

	if !s.shouldIgnore("Google") {
		t.Error("expected Google to be ignored")
	}
	if s.shouldIgnore("Docker") {
		t.Error("expected Docker not to be ignored")
	}
}
