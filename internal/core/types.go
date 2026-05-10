package core

import "time"

type Verdict int

const (
	VerdictInstalled Verdict = iota
	VerdictLeftover
	VerdictUncertain
)

func (v Verdict) String() string {
	switch v {
	case VerdictInstalled:
		return "INSTALLED"
	case VerdictLeftover:
		return "LEFTOVER"
	case VerdictUncertain:
		return "UNCERTAIN"
	default:
		return "UNKNOWN"
	}
}

type Signal struct {
	Kind   string  `json:"kind"`
	Detail string  `json:"detail"`
	Weight float64 `json:"weight"`
}

type MatchResult struct {
	Verdict    Verdict  `json:"verdict"`
	Confidence float64  `json:"confidence"`
	Signals    []Signal `json:"signals"`
}

type Leftover struct {
	Path     string       `json:"path"`
	Name     string       `json:"name"`
	Size     int64        `json:"size"`
	Location string       `json:"location"`
	ModTime  time.Time    `json:"mod_time"`
	Match    *MatchResult `json:"match,omitempty"`
}

type ScanResult struct {
	Items     []Leftover `json:"items"`
	TotalSize int64      `json:"total_size"`
	Duration  string     `json:"duration"`
}

type LocationType string

const (
	LocAppSupport       LocationType = "Application Support"
	LocCaches           LocationType = "Caches"
	LocState            LocationType = "Saved Application State"
	LocContainers       LocationType = "Containers"
	LocPreferences      LocationType = "Preferences"
	LocLogs             LocationType = "Logs"
	LocTemp             LocationType = "TemporaryItems"
	LocLaunchAgents     LocationType = "LaunchAgents"
	LocCrashes          LocationType = "Crashes"
	LocCookies          LocationType = "Cookies"
	LocWebKit           LocationType = "WebKit"
	LocFonts            LocationType = "Fonts"
	LocSyncedPrefs      LocationType = "SyncedPreferences"
	LocGroupContainers  LocationType = "Group Containers"
)

type Location struct {
	Type LocationType
	Path string
}
