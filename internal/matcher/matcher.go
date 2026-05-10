package matcher

import (
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/core"
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
			Detail: "No matching signals found. Unknown folder.",
			Weight: 0.1,
		})
		confidence = 0.1
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
	for bid := range m.index.BundleIDs {
		if strings.EqualFold(bid, folderName) {
			return &core.Signal{
				Kind:   "bundle_id",
				Detail: "Bundle ID " + bid + " matches installed app exactly",
				Weight: 1.0,
			}
		}
		shortID := shorterBundleID(bid)
		if shortID != "" && strings.EqualFold(shortID, folderName) {
			return &core.Signal{
				Kind:   "bundle_id",
				Detail: "Bundle ID " + bid + " matches installed app (short form)",
				Weight: 1.0,
			}
		}
	}

	for name := range m.index.Names {
		if strings.EqualFold(name, folderName) {
			return &core.Signal{
				Kind:   "app_name",
				Detail: "App name \"" + name + "\" matches folder exactly",
				Weight: 1.0,
			}
		}
	}

	return nil
}

func (m *Matcher) fuzzyMatch(folderName string) (string, float64) {
	bestScore := 0.0
	bestName := ""

	for name := range m.index.Names {
		score := fuzzyScore(folderName, name)
		if score > bestScore {
			bestScore = score
			bestName = name
		}
	}

	if bestScore > 0.5 {
		return bestName, bestScore
	}
	return "", 0
}

func (m *Matcher) heuristicMatch(folderName string) []core.Signal {
	var signals []core.Signal

	if parts := splitReverseDomain(folderName); len(parts) >= 2 && len(parts) <= 6 {
		for _, part := range parts {
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

func fuzzyScore(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return 1.0
	}

	if strings.Contains(a, b) || strings.Contains(b, a) {
		return 0.8
	}

	common := commonPrefix(a, b)
	if len(common) >= 3 {
		return float64(len(common)) / float64(max(len(a), len(b)))
	}

	return 0.0
}

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return a[:i]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
