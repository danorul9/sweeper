package appindex

import "strings"

var macOSDefaultApps = map[string]bool{
	"Safari":              true,
	"Mail":                true,
	"Calendar":            true,
	"Notes":               true,
	"Reminders":           true,
	"Messages":            true,
	"FaceTime":            true,
	"Photos":              true,
	"Music":               true,
	"TV":                  true,
	"Podcasts":            true,
	"Books":               true,
	"App Store":           true,
	"System Settings":     true,
	"System Preferences":  true,
	"Finder":              true,
	"QuickTime Player":    true,
	"Stickies":            true,
	"Preview":             true,
	"TextEdit":            true,
	"Dictionary":          true,
	"Calculator":          true,
	"Chess":               true,
	"Activity Monitor":    true,
	"Console":             true,
	"Keychain Access":     true,
	"Disk Utility":        true,
	"Terminal":            true,
	"Script Editor":       true,
	"Screenshot":          true,
	"Voice Memos":         true,
	"Contacts":            true,
	"Maps":                true,
	"Home":                true,
	"Shortcuts":           true,
	"Freeform":            true,
	"Image Capture":       true,
	"Photo Booth":         true,
	"Grapher":             true,
	"ColorSync Utility":   true,
	"Digital Color Meter": true,
	"Migration Assistant": true,
	"System Information":  true,
	"Bluetooth File Exchange": true,
	"Archive Utility":     true,
	"Feedback Assistant":  true,
	"iMovie":              true,
	"GarageBand":          true,
	"Keynote":             true,
	"Numbers":             true,
	"Pages":               true,
}

func IsSystemApp(name, bundleID, path string) bool {
	if strings.HasPrefix(path, "/System/Applications") {
		return true
	}
	if bundleID != "" {
		if strings.HasPrefix(bundleID, "com.apple.") {
			return true
		}
	}
	if macOSDefaultApps[name] {
		return true
	}
	return false
}
