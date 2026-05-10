package scanner

import (
	"os"
	"path/filepath"
)

func LibraryLocations() []Location {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	lib := filepath.Join(home, "Library")

	return []Location{
		{Type: LocAppSupport, Path: filepath.Join(lib, "Application Support")},
		{Type: LocCaches, Path: filepath.Join(lib, "Caches")},
		{Type: LocState, Path: filepath.Join(lib, "Saved Application State")},
		{Type: LocPreferences, Path: filepath.Join(lib, "Preferences")},
		{Type: LocLogs, Path: filepath.Join(lib, "Logs")},
		{Type: LocLaunchAgents, Path: filepath.Join(lib, "LaunchAgents")},
		{Type: LocContainers, Path: filepath.Join(lib, "Containers")},
		{Type: LocCrashes, Path: filepath.Join(lib, "Application Support/Crashes")},
		{Type: LocCookies, Path: filepath.Join(lib, "Cookies")},
		{Type: LocWebKit, Path: filepath.Join(lib, "WebKit")},
		{Type: LocFonts, Path: filepath.Join(lib, "Fonts")},
		{Type: LocSyncedPrefs, Path: filepath.Join(lib, "SyncedPreferences")},
		{Type: LocGroupContainers, Path: filepath.Join(lib, "Group Containers")},
		{Type: LocTemp, Path: filepath.Join(lib, "TemporaryItems")},
	}
}

func SafeLocations() []Location {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	lib := filepath.Join(home, "Library")

	return []Location{
		{Type: LocCaches, Path: filepath.Join(lib, "Caches")},
		{Type: LocLogs, Path: filepath.Join(lib, "Logs")},
		{Type: LocState, Path: filepath.Join(lib, "Saved Application State")},
		{Type: LocTemp, Path: filepath.Join(lib, "TemporaryItems")},
		{Type: LocCrashes, Path: filepath.Join(lib, "Application Support/Crashes")},
		{Type: LocCookies, Path: filepath.Join(lib, "Cookies")},
	}
}

func AggressiveLocations() []Location {
	return LibraryLocations()
}

func ReclaimLocations() []Location {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	lib := filepath.Join(home, "Library")

	return []Location{
		{Type: LocCaches, Path: filepath.Join(lib, "Caches")},
		{Type: LocLogs, Path: filepath.Join(lib, "Logs")},
		{Type: LocState, Path: filepath.Join(lib, "Saved Application State")},
		{Type: LocTemp, Path: filepath.Join(lib, "TemporaryItems")},
	}
}

func FolderName(path string) string {
	return filepath.Base(path)
}
