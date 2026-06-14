package matcher

import (
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/core"
	"github.com/sahilm/fuzzy"
)

type Matcher struct {
	index *appindex.AppIndex
}

func New(index *appindex.AppIndex) *Matcher {
	return &Matcher{index: index}
}

func (m *Matcher) Match(folderName, folderPath string, modTime time.Time) *core.MatchResult {
	signals := []core.Signal{}
	verdict := core.VerdictUncertain
	confidence := 0.0

	layer1 := m.exactMatch(folderName)
	if layer1 != nil {
		signals = append(signals, *layer1)
		verdict = core.VerdictInstalled
		confidence = 1.0
	}

	if verdict == core.VerdictUncertain || confidence < 1.0 {
		if fp := matchFingerprint(folderName); fp != nil {
			signals = append(signals, core.Signal{
				Kind:   "fingerprint",
				Detail: "Fingerprint match: " + fp.Name,
				Weight: 0.85,
			})
			verdict = core.VerdictLeftover
			confidence = 0.85
		}
	}

	if verdict == core.VerdictUncertain {
		fuzzyResult, fuzzyScore := m.fuzzyMatch(folderName)
		if fuzzyResult != "" && fuzzyScore > 0.5 {
			signals = append(signals, core.Signal{
				Kind:   "fuzzy",
				Detail: "Fuzzy match: " + folderName + " → " + fuzzyResult,
				Weight: fuzzyScore,
			})
			verdict = core.VerdictUncertain
			confidence = fuzzyScore
		}
	}

	if verdict == core.VerdictUncertain {
		heuristicSignals := m.heuristicMatch(folderName)
		signals = append(signals, heuristicSignals...)
		for _, s := range heuristicSignals {
			if s.Weight > confidence {
				confidence = s.Weight
			}
		}
		if confidence > 0.6 {
			verdict = core.VerdictUncertain
		}
	}

	if verdict != core.VerdictLeftover {
		procSignals := m.processMatch(folderName)
		signals = append(signals, procSignals...)
		for _, s := range procSignals {
			confidence += s.Weight
			if verdict != core.VerdictInstalled {
				verdict = core.VerdictInstalled
			}
		}
	}

	ageSignals := ageScore(modTime)
	if len(ageSignals) > 0 {
		signals = append(signals, ageSignals...)
		for _, s := range ageSignals {
			confidence += s.Weight
		}
		if confidence > 1.0 {
			confidence = 1.0
		}
	}

	if len(signals) == 0 {
		signals = append(signals, core.Signal{
			Kind:   "no_match",
			Detail: "No matching app found — folder is orphaned",
			Weight: 0.30,
		})
		verdict = core.VerdictLeftover
		confidence = 0.30
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return &core.MatchResult{
		Verdict:    verdict,
		Confidence: confidence,
		Signals:    signals,
	}
}

func (m *Matcher) processMatch(folderName string) []core.Signal {
	if appindex.IsProcessRunning(folderName) {
		return []core.Signal{{
			Kind:   "running_process",
			Detail: "Running process \"" + folderName + "\" is running on this system",
			Weight: 0.95,
		}}
	}

	if m.index == nil {
		return nil
	}

	for bid := range m.index.BundleIDs {
		if strings.EqualFold(bid, folderName) && appindex.IsProcessRunning(shorterBundleID(bid)) {
			return []core.Signal{{
				Kind:   "running_process",
				Detail: "App with bundle ID \"" + bid + "\" is running on this system",
				Weight: 0.95,
			}}
		}
	}

	return nil
}

func (m *Matcher) exactMatch(folderName string) *core.Signal {
	if sig := m.matchDirect(folderName); sig != nil {
		return sig
	}

	baseName := stripSuffixes(folderName)
	if baseName != folderName {
		if sig := m.matchDirect(baseName); sig != nil {
			return sig
		}
	}

	cleaned := stripTeamIDPrefix(folderName)
	if cleaned != folderName {
		if sig := m.matchDirect(cleaned); sig != nil {
			return sig
		}
		if cleaned2 := stripSuffixes(cleaned); cleaned2 != cleaned {
			if sig := m.matchDirect(cleaned2); sig != nil {
				return sig
			}
		}
	}

	folderLower := strings.ToLower(folderName)
	for bid := range m.index.BundleIDs {
		if strings.HasPrefix(folderLower, strings.ToLower(bid)+".") ||
			strings.HasPrefix(folderLower, strings.ToLower(bid)+"-") {
			return &core.Signal{
				Kind:   "bundle_id",
				Detail: "Bundle ID \"" + bid + "\" is prefix of folder \"" + folderName + "\"",
				Weight: 1.0,
			}
		}
	}

	return nil
}

func (m *Matcher) matchDirect(name string) *core.Signal {
	for bid := range m.index.BundleIDs {
		if strings.EqualFold(bid, name) {
			return &core.Signal{
				Kind:   "bundle_id",
				Detail: "Bundle ID " + bid + " matches installed app exactly",
				Weight: 1.0,
			}
		}
		shortID := shorterBundleID(bid)
		if shortID != "" && strings.EqualFold(shortID, name) {
			return &core.Signal{
				Kind:   "bundle_id",
				Detail: "Bundle ID " + bid + " matches installed app (short form)",
				Weight: 1.0,
			}
		}
	}

	for appName := range m.index.Names {
		if strings.EqualFold(appName, name) {
			return &core.Signal{
				Kind:   "app_name",
				Detail: "App name \"" + appName + "\" matches folder exactly",
				Weight: 1.0,
			}
		}
	}

	if parts := splitReverseDomain(name); len(parts) >= 3 {
		last := strings.ToLower(parts[len(parts)-1])
		for appName := range m.index.Names {
			if strings.EqualFold(appName, last) {
				return &core.Signal{
					Kind:   "reverse_domain",
					Detail: "Reverse-domain \"" + name + "\" last component \"" + last + "\" matches installed app",
					Weight: 1.0,
				}
			}
		}
		combined := strings.ToLower(parts[len(parts)-2]) + " " + last
		for appName := range m.index.Names {
			if strings.EqualFold(appName, combined) {
				return &core.Signal{
					Kind:   "reverse_domain",
					Detail: "Reverse-domain \"" + name + "\" combined \"" + combined + "\" matches installed app",
					Weight: 1.0,
				}
			}
		}
	}

	return nil
}

var knownSuffixes = []string{
	".savedState",
	".ShipIt",
	"-ShipIt",
	".ShipIt",
	".appstore",
	".loginhelper",
	".Intents",
	".ServiceExtension",
	".ShareExtension",
	".SafariExtension",
	".SafariExtension2",
	".WAAppKitBridgeService",
	".RaycastAppIntents",
	".iTermAI",
	".Shared",
	".findersync",
	".finderhelper",
	".fpext",
	".photos-extension",
	".quicklookpreviewextension",
	".remove-background",
	".thumbnailextension",
	".PacketTunnel-DNSProxy",
	".PacketTunnel-Dausos",
	".PacketTunnel-OpenVPN",
	".PacketTunnel-WireGuard",
	".shared",
	".private",
}

func stripSuffixes(name string) string {
	lower := strings.ToLower(name)
	for _, s := range knownSuffixes {
		if strings.HasSuffix(lower, strings.ToLower(s)) {
			return name[:len(name)-len(s)]
		}
	}
	return name
}

func stripTeamIDPrefix(name string) string {
	parts := strings.SplitN(name, ".", 3)
	if len(parts) >= 2 && isTeamID(parts[0]) {
		if len(parts) >= 3 && parts[1] == "group" {
			return parts[2]
		}
		return strings.Join(parts[1:], ".")
	}
	if len(parts) >= 2 && parts[0] == "group" {
		return strings.Join(parts[1:], ".")
	}
	if strings.HasPrefix(name, "--") {
		if idx := strings.Index(name[2:], "-"); idx >= 0 {
			return name[2+idx+1:]
		}
		return strings.TrimPrefix(name, "--AppIdentifierPrefix-")
	}
	return name
}

func isTeamID(s string) bool {
	if len(s) != 10 {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func (m *Matcher) fuzzyMatch(folderName string) (string, float64) {
	if m.index == nil || len(m.index.Names) == 0 {
		return "", 0
	}

	names := make([]string, 0, len(m.index.Names))
	for name := range m.index.Names {
		names = append(names, name)
	}

	matches := fuzzy.Find(folderName, names)
	if len(matches) == 0 {
		return "", 0
	}

	best := matches[0]
	score := float64(best.Score) / 100.0
	if score > 1.0 {
		score = 1.0
	}
	// Length-ratio guard: reject when candidate is >2x the query length.
	// Catches short queries matching much longer names via substring overlap
	// (e.g. "virt-manager" → "Karabiner-VirtualHIDDevice-Manager" at 2.83x).
	q := strings.ToLower(folderName)
	c := strings.ToLower(best.Str)
	minLen := len(q)
	if minLen > len(c) {
		minLen = len(c)
	}
	maxLen := len(q)
	if maxLen < len(c) {
		maxLen = len(c)
	}
	if maxLen > minLen*2 {
		return "", 0
	}
	if score > 0.5 {
		return best.Str, score
	}
	return "", 0
}

func (m *Matcher) heuristicMatch(folderName string) []core.Signal {
	var signals []core.Signal

	if parts := splitReverseDomain(folderName); len(parts) >= 2 && len(parts) <= 6 {
		for _, part := range parts {
			if part == "" {
				continue
			}
			for name := range m.index.Names {
				if strings.Contains(strings.ToLower(name), strings.ToLower(part)) {
					signals = append(signals, core.Signal{
						Kind:   "reverse_domain",
						Detail: "Reverse-domain \"" + folderName + "\" contains token \"" + part + "\" matching installed app \"" + name + "\"",
						Weight: 0.65,
					})
					return signals
				}
			}
		}
	}

	for name := range m.index.Names {
		if strings.Contains(strings.ToLower(name), strings.ToLower(folderName)) ||
			strings.Contains(strings.ToLower(folderName), strings.ToLower(name)) {
			signals = append(signals, core.Signal{
				Kind:   "partial_name",
				Detail: "Partial name match: \"" + folderName + "\" ↔ \"" + name + "\"",
				Weight: 0.50,
			})
			return signals
		}
	}

	return signals
}

func shorterBundleID(bid string) string {
	parts := strings.Split(bid, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

func splitReverseDomain(s string) []string {
	return strings.Split(s, ".")
}
