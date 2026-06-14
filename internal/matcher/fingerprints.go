package matcher

import (
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/core"
)

type AppFingerprint struct {
	Name      string
	BundleIDs []string
	Vendors   []string
	Paths     []string
}

var Fingerprints = []AppFingerprint{
	{
		Name:      "Docker Desktop",
		BundleIDs: []string{"com.docker.docker"},
		Paths:     []string{"Docker", "Docker Desktop", "com.docker.docker", "com.electron.dockerdesktop"},
	},
	{
		Name:      "Zoom",
		BundleIDs: []string{"us.zoom.xos", "zoom.us"},
		Paths:     []string{"us.zoom.xos", "zoom.us", "Zoom"},
	},
	{
		Name:      "Visual Studio Code",
		BundleIDs: []string{"com.microsoft.VSCode"},
		Vendors:   []string{"microsoft"},
		Paths:     []string{"Code", "Visual Studio Code", "com.microsoft.VSCode"},
	},
	{
		Name:      "OBS",
		BundleIDs: []string{"com.obsproject.obs-studio"},
		Vendors:   []string{"obsproject"},
		Paths:     []string{"obs-studio", "com.obsproject.obs-studio"},
	},
	{
		Name:      "Raycast",
		BundleIDs: []string{"com.raycast.macos"},
		Vendors:   []string{"raycast"},
		Paths:     []string{"com.raycast.macos", "Raycast"},
	},
	{
		Name:      "Slack",
		BundleIDs: []string{"com.tinyspeck.slackmacgap"},
		Vendors:   []string{"slack", "tinyspeck"},
		Paths:     []string{"Slack", "com.tinyspeck.slackmacgap", "com.electron.slack"},
	},
	{
		Name:      "Discord",
		BundleIDs: []string{"com.hnc.Discord"},
		Vendors:   []string{"discord", "hnc"},
		Paths:     []string{"Discord", "com.hnc.Discord", "com.electron.discord"},
	},
	{
		Name:      "Google Drive",
		BundleIDs: []string{"com.google.drive", "com.google.BackupAndSync"},
		Paths:     []string{"Google Drive", "com.google.drive"},
	},
	{
		Name:      "Google Chrome",
		BundleIDs: []string{"com.google.Chrome"},
		Paths:     []string{"Google/Chrome", "com.google.Chrome", "Google Chrome"},
	},
	{
		Name:      "Epic Games Launcher",
		BundleIDs: []string{"com.epicgames.EpicGamesLauncher"},
		Vendors:   []string{"epic", "epicgames"},
		Paths:     []string{"Epic", "Epic Games Launcher", "com.epicgames.*"},
	},
	{
		Name:      "Steam",
		BundleIDs: []string{"com.valve.steam"},
		Vendors:   []string{"valve"},
		Paths:     []string{"Steam", "com.valve.steam"},
	},
	{
		Name:      "Spotify",
		BundleIDs: []string{"com.spotify.client"},
		Vendors:   []string{"spotify"},
		Paths:     []string{"Spotify", "com.spotify.client"},
	},
	{
		Name:      "iTerm2",
		BundleIDs: []string{"com.googlecode.iterm2"},
		Vendors:   []string{"iterm2", "googlecode"},
		Paths:     []string{"iTerm2", "com.googlecode.iterm2"},
	},
	{
		Name:      "Transmission",
		BundleIDs: []string{"org.m0k.transmission"},
		Vendors:   []string{"transmission", "m0k"},
		Paths:     []string{"Transmission", "org.m0k.transmission"},
	},
	{
		Name:      "1Password",
		BundleIDs: []string{"com.agilebits.onepassword7", "com.agilebits.onepassword"},
		Vendors:   []string{"agilebits"},
		Paths:     []string{"1Password", "com.agilebits.onepassword", "com.agilebits.onepassword7"},
	},
	{
		Name:      "Alfred",
		BundleIDs: []string{"com.runningwithcrayons.Alfred"},
		Vendors:   []string{"runningwithcrayons"},
		Paths:     []string{"Alfred", "com.runningwithcrayons.Alfred"},
	},
	{
		Name:      "Postman",
		BundleIDs: []string{"com.postmanlabs.mac"},
		Vendors:   []string{"postman", "postmanlabs"},
		Paths:     []string{"Postman", "com.postmanlabs.mac"},
	},
	{
		Name:      "Figma",
		BundleIDs: []string{"com.figma.Desktop"},
		Vendors:   []string{"figma"},
		Paths:     []string{"Figma", "com.figma.Desktop"},
	},
	{
		Name:      "Notion",
		BundleIDs: []string{"notion.id"},
		Vendors:   []string{"notion"},
		Paths:     []string{"Notion", "notion.id"},
	},
	{
		Name:      "Android Studio",
		BundleIDs: []string{"com.google.android.studio"},
		Vendors:   []string{"android", "google"},
		Paths:     []string{"Android Studio", "com.google.android.studio"},
	},
	{
		Name:      "Xcode",
		BundleIDs: []string{"com.apple.dt.Xcode"},
		Vendors:   []string{"apple"},
		Paths:     []string{"Xcode", "com.apple.dt.Xcode"},
	},
	{
		Name:      "Xcode Derived Data",
		BundleIDs: []string{"com.apple.dt.Xcode"},
		Paths:     []string{"DerivedData"},
	},
	{
		Name:      "Playwright",
		BundleIDs: []string{},
		Vendors:   []string{"microsoft", "playwright"},
		Paths:     []string{"ms-playwright", "ms-playwright-go"},
	},
	{
		Name:      "MetaTrader",
		BundleIDs: []string{"com.metatrader.*"},
		Paths:     []string{"MetaTrader 5", "MetaTrader 4", "MetaQuotes"},
	},
	{
		Name:      "Setapp",
		BundleIDs: []string{"com.setapp.*"},
		Vendors:   []string{"setapp"},
		Paths:     []string{"Setapp", "com.setapp.*"},
	},
	{
		Name:      "CleanMyMac",
		BundleIDs: []string{"com.macpaw.CleanMyMac"},
		Vendors:   []string{"macpaw"},
		Paths:     []string{"CleanMyMac", "com.macpaw.CleanMyMac"},
	},
	{
		Name:      "OpenWhispr",
		BundleIDs: []string{"com.openwhispr.app"},
		Vendors:   []string{"openwhispr"},
		Paths:     []string{"openwhispr", "open-whispr"},
	},
	{
		Name:      "HuggingFace Hub",
		BundleIDs: []string{},
		Vendors:   []string{"huggingface"},
		Paths:     []string{"huggingface"},
	},
	{
		Name:      "uv (Python)",
		BundleIDs: []string{},
		Vendors:   []string{},
		Paths:     []string{"uv"},
	},
	{
		Name:      "Wireshark",
		BundleIDs: []string{"org.wireshark.*"},
		Vendors:   []string{"wireshark"},
		Paths:     []string{"Wireshark", "org.wireshark.*"},
	},
	{
		Name:      "VLC",
		BundleIDs: []string{"org.videolan.vlc"},
		Vendors:   []string{"videolan"},
		Paths:     []string{"VLC", "org.videolan.vlc"},
	},
	{
		Name:      "Firefox",
		BundleIDs: []string{"org.mozilla.firefox"},
		Vendors:   []string{"mozilla", "firefox"},
		Paths:     []string{"Firefox", "org.mozilla.firefox"},
	},
	{
		Name:      "Sublime Text",
		BundleIDs: []string{"com.sublimetext.*"},
		Vendors:   []string{"sublimetext"},
		Paths:     []string{"Sublime Text", "com.sublimetext.*", "Sublime Text 3", "Sublime Text 4"},
	},
}

func matchFingerprint(folderName string) *AppFingerprint {
	for i := range Fingerprints {
		fp := &Fingerprints[i]
		for _, p := range fp.Paths {
			if strings.EqualFold(p, folderName) {
				return fp
			}
			if matched, _ := simpleMatch(p, folderName); matched {
				return fp
			}
		}
	}
	return nil
}

func simpleMatch(pattern, name string) (bool, error) {
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		lower := strings.ToLower(name)
		if len(parts) == 2 && parts[0] == "" {
			return strings.HasSuffix(lower, strings.ToLower(parts[1])), nil
		}
		if len(parts) == 2 && parts[1] == "" {
			return strings.HasPrefix(lower, strings.ToLower(parts[0])), nil
		}
	}
	return strings.EqualFold(pattern, name), nil
}

func ageScore(modTime time.Time) []core.Signal {
	age := time.Since(modTime)
	if age < 7*24*time.Hour {
		return nil
	}
	if age >= 180*24*time.Hour {
		return []core.Signal{{Kind: "age", Detail: "Last modified over 6 months ago", Weight: 0.10}}
	}
	if age >= 30*24*time.Hour {
		return []core.Signal{{Kind: "age", Detail: "Last modified over 30 days ago", Weight: 0.05}}
	}
	return nil
}
