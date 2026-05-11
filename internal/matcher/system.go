package matcher

import "strings"

var AppleSystemPrefixes = []string{
	"com.apple.",
	"com.apple.launchd",
	"com.apple.security",
	"com.apple.assistant",
	"com.apple.siri",
	"com.apple.Safari",
	"com.apple.itunes",
	"com.apple.iTunes",
	"com.apple.Terminal",
	"com.apple.finder",
	"com.apple.Finder",
	"com.apple.system",
	"com.apple.print",
	"com.apple.preference",
	"com.apple.studentd",
}

var AppleSystemFolders = []string{
	"./",
	".",
	"..",
	".DS_Store",
	"local",
	"Preferences",
	"Application Support",
	"Caches",
	"Containers",
	"Cookies",
	"Logs",
	"WebKit",
	"Safari",
	"com.apple.*",
	"CloudKit",
	"HomeKit",
	"iCloud",
	"Speech",
	"Voicememos",
	"Stocks",
	"Weather",
	"Maps",
	"Calendar",
	"Reminders",
	"Notes",
	"Messages",
}

func IsSystemPath(name string) bool {
	lower := strings.ToLower(name)

	// Direct prefix match
	for _, prefix := range AppleSystemPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}

	// Check known Apple system folders
	for _, folder := range AppleSystemFolders {
		if filepathMatch(folder, name) {
			return true
		}
		if strings.EqualFold(folder, name) {
			return true
		}
	}

	// Handle group.com.apple.* (e.g. group.com.apple.Safari.SandboxBroker)
	if strings.HasPrefix(lower, "group.com.apple.") {
		return true
	}

	// Handle team-ID-prefixed Apple paths (e.g. EQHXZ8M8AV.com.apple.*)
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 && isTeamID(parts[0]) {
		cleaned := strings.ToLower(parts[1])
		for _, prefix := range AppleSystemPrefixes {
			if strings.HasPrefix(cleaned, strings.ToLower(prefix)) {
				return true
			}
		}
	}

	return false
}



func filepathMatch(pattern, name string) bool {
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 && parts[0] == "" {
			return strings.HasSuffix(name, parts[1])
		}
		if len(parts) == 2 && parts[1] == "" {
			return strings.HasPrefix(name, parts[0])
		}
	}
	return pattern == name
}
