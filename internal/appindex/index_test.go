package appindex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAppIndex(t *testing.T) {
	idx := NewAppIndex()
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
	if idx.Names == nil {
		t.Error("expected Names map")
	}
	if idx.BundleIDs == nil {
		t.Error("expected BundleIDs map")
	}
}

func TestAppNameFromPath(t *testing.T) {
	name := AppNameFromPath("/Applications/Docker.app")
	if name != "Docker" {
		t.Errorf("expected Docker, got %s", name)
	}
}

func TestScanApplicationsDirNotFound(t *testing.T) {
	apps, err := ScanApplicationsDir("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Errorf("expected empty, got %d", len(apps))
	}
}

func TestScanApplicationsDirEmpty(t *testing.T) {
	dir := t.TempDir()
	apps, err := ScanApplicationsDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 0 {
		t.Errorf("expected empty, got %d", len(apps))
	}
}

func TestScanApplicationsDirWithApps(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "TestApp.app", "Contents", "MacOS"), 0755)
	os.MkdirAll(filepath.Join(dir, "Other.app", "Contents", "MacOS"), 0755)

	apps, err := ScanApplicationsDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Errorf("expected 2 apps, got %d", len(apps))
	}
}

func TestSplitReverseDomain(t *testing.T) {
	parts := splitReverseDomain("com.docker.docker")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}
	if parts[0] != "com" {
		t.Errorf("expected 'com', got '%s'", parts[0])
	}
}

func TestContainsGlob(t *testing.T) {
	if !hasGlob("com.adobe.*") {
		t.Error("expected hasGlob true for com.adobe.*")
	}
	if hasGlob("com.docker.docker") {
		t.Error("expected hasGlob false for simple string")
	}
}

func TestAppFamilyInstalled(t *testing.T) {
	idx := NewAppIndex()
	idx.BundleIDs["com.google.Chrome"] = struct{}{}

	if !idx.IsFamilyInstalled(AppFamilies[0]) {
		t.Error("expected Google family to be installed")
	}
}

func TestAppFamilyNotInstalled(t *testing.T) {
	idx := NewAppIndex()

	if idx.IsFamilyInstalled(AppFamilies[3]) {
		t.Error("expected Microsoft family NOT to be installed")
	}
}

func TestAppFamilyApple(t *testing.T) {
	idx := NewAppIndex()
	if !idx.IsFamilyInstalled(AppFamilies[4]) {
		t.Error("expected Apple family to always be installed")
	}
}

func TestFamilyForFolder(t *testing.T) {
	idx := NewAppIndex()

	family := idx.FamilyForFolder("Google")
	if family == nil {
		t.Fatal("expected Google folder to match a family")
	}
	if family.Vendor != "Google" {
		t.Errorf("expected Google vendor, got %s", family.Vendor)
	}
}

func TestFamilyForFolderUnknown(t *testing.T) {
	idx := NewAppIndex()

	family := idx.FamilyForFolder("SomeRandomFolder")
	if family != nil {
		t.Errorf("expected nil for unknown folder, got %v", family.Vendor)
	}
}

func TestSplitByDot(t *testing.T) {
	result := splitByDot("com.example.app")
	if len(result) != 3 {
		t.Errorf("expected 3 parts, got %d", len(result))
	}

	result = splitByDot("")
	if result != nil {
		t.Error("expected nil for empty string")
	}

	result = splitByDot("simple")
	if len(result) != 1 || result[0] != "simple" {
		t.Error("expected ['simple'] for string without dots")
	}
}

func TestNormalizeAppName(t *testing.T) {
	result := NormalizeAppName("Visual Studio Code.app")
	expected := "visual studio code"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestSaveLoadCache(t *testing.T) {
	original := NewAppIndex()
	original.Names["TestApp"] = struct{}{}
	original.BundleIDs["com.test.app"] = struct{}{}

	err := original.Save()
	if err != nil {
		t.Fatal(err)
	}

	cacheDir, err := configDir()
	if err != nil {
		t.Skip("no cache dir:", err)
	}

	loaded, _, err := LoadCached()
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil {
		t.Fatal("loaded nil index")
	}
	if _, ok := loaded.Names["TestApp"]; !ok {
		t.Error("expected TestApp in loaded index")
	}
	if _, ok := loaded.BundleIDs["com.test.app"]; !ok {
		t.Error("expected com.test.app in loaded index")
	}

	cacheFile := filepath.Join(cacheDir, "appindex.json")
	os.Remove(cacheFile)
}

func configDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "sweeper"), nil
}

func TestBuildEmpty(t *testing.T) {
	idx, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if idx == nil {
		t.Fatal("expected non-nil index")
	}
}

func TestSortStrings(t *testing.T) {
	input := []string{"z", "a", "b"}
	result := SortStrings(input)
	if result[0] != "a" || result[1] != "b" || result[2] != "z" {
		t.Error("expected sorted output")
	}
}
