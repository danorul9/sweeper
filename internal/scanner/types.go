package scanner

import "github.com/danorul9/sweeper/internal/core"

type Verdict = core.Verdict
type Signal = core.Signal
type MatchResult = core.MatchResult
type Leftover = core.Leftover
type ScanResult = core.ScanResult
type LocationType = core.LocationType
type Location = core.Location

const (
	VerdictInstalled Verdict = core.VerdictInstalled
	VerdictLeftover          = core.VerdictLeftover
	VerdictUncertain         = core.VerdictUncertain
)

const (
	LocAppSupport      LocationType = core.LocAppSupport
	LocCaches          LocationType = core.LocCaches
	LocState           LocationType = core.LocState
	LocContainers      LocationType = core.LocContainers
	LocPreferences     LocationType = core.LocPreferences
	LocLogs            LocationType = core.LocLogs
	LocTemp            LocationType = core.LocTemp
	LocLaunchAgents    LocationType = core.LocLaunchAgents
	LocCrashes         LocationType = core.LocCrashes
	LocCookies         LocationType = core.LocCookies
	LocWebKit          LocationType = core.LocWebKit
	LocFonts           LocationType = core.LocFonts
	LocSyncedPrefs     LocationType = core.LocSyncedPrefs
	LocGroupContainers LocationType = core.LocGroupContainers
)
