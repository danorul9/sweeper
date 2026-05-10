package matcher

import (
	"strings"
)

type MatchStrategy int

const (
	StrategyBundleID MatchStrategy = iota
	StrategyFuzzy
	StrategyReverseDomain
	StrategyCamelCase
	StrategyVendorPrefix
	StrategyFingerprint
)

func StrategyName(s MatchStrategy) string {
	switch s {
	case StrategyBundleID:
		return "bundle_id"
	case StrategyFuzzy:
		return "fuzzy"
	case StrategyReverseDomain:
		return "reverse_domain"
	case StrategyCamelCase:
		return "camel_case"
	case StrategyVendorPrefix:
		return "vendor_prefix"
	case StrategyFingerprint:
		return "fingerprint"
	default:
		return "unknown"
	}
}

func camelCaseSplit(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	current := []rune{rune(s[0])}

	for i := 1; i < len(s); i++ {
		ch := rune(s[i])
		prev := rune(s[i-1])

		if ch >= 'A' && ch <= 'Z' && prev >= 'a' && prev <= 'z' {
			parts = append(parts, string(current))
			current = []rune{ch}
		} else if ch == '-' || ch == '_' || ch == ' ' {
			parts = append(parts, string(current))
			current = nil
		} else {
			current = append(current, ch)
		}
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}

func vendorFromBundleID(bundleID string) string {
	parts := splitReverseDomain(bundleID)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return ""
}

func extractTokens(name string) []string {
	tokens := camelCaseSplit(name)

	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == '.'
	})
	tokens = append(tokens, words...)

	seen := make(map[string]bool)
	var unique []string
	for _, t := range tokens {
		lower := strings.ToLower(t)
		if !seen[lower] && len(t) > 1 {
			seen[lower] = true
			unique = append(unique, t)
		}
	}
	return unique
}
