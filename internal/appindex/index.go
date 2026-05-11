package appindex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/danorul9/sweeper/internal/config"
)

type CachedIndex struct {
	Timestamp time.Time `json:"timestamp"`
	Index     *AppIndex `json:"index"`
}

func Build() (*AppIndex, error) {
	idx := NewAppIndex()
	apps := ScanAllApplications()

	for _, appPath := range apps {
		info, err := ReadAppInfo(appPath)
		if err != nil {
			appName := AppNameFromPath(appPath)
			idx.Names[appName] = struct{}{}
			continue
		}

		if info.Name != "" {
			idx.Names[info.Name] = struct{}{}
		}
		if info.BundleID != "" {
			idx.BundleIDs[info.BundleID] = struct{}{}
		}
		if info.Version != "" {
			// store version info if needed
		}

		appName := AppNameFromPath(appPath)
		if info.Name == "" {
			idx.Names[appName] = struct{}{}
		}

		execName := ExecutableName(appPath)
		idx.Executables[execName] = struct{}{}

		parts := splitVendor(info.BundleID)
		for _, p := range parts {
			if p != "" {
				idx.Vendors[p] = struct{}{}
			}
		}
	}

	return idx, nil
}

func mergeKnownApps(idx *AppIndex) {
	for name := range KnownAppNames {
		idx.Names[name] = struct{}{}
	}
}

func splitVendor(bundleID string) []string {
	if bundleID == "" {
		return nil
	}
	parts := splitReverseDomain(bundleID)
	return parts
}

func splitReverseDomain(bundleID string) []string {
	return splitByDot(bundleID)
}

func splitByDot(s string) []string {
	if s == "" {
		return nil
	}
	result := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func (idx *AppIndex) Save() error {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return err
	}
	path := filepath.Join(cacheDir, "appindex.json")

	cached := CachedIndex{
		Timestamp: time.Now(),
		Index:     idx,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal appindex: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write appindex cache: %w", err)
	}
	return nil
}

func LoadCached() (*AppIndex, time.Time, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return nil, time.Time{}, err
	}
	path := filepath.Join(cacheDir, "appindex.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, fmt.Errorf("read appindex cache: %w", err)
	}

	var cached CachedIndex
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, time.Time{}, fmt.Errorf("unmarshal appindex cache: %w", err)
	}

	if cached.Index == nil {
		return nil, time.Time{}, nil
	}
	if cached.Index.Names == nil {
		cached.Index.Names = make(map[string]struct{})
	}
	if cached.Index.BundleIDs == nil {
		cached.Index.BundleIDs = make(map[string]struct{})
	}
	if cached.Index.Vendors == nil {
		cached.Index.Vendors = make(map[string]struct{})
	}
	if cached.Index.Executables == nil {
		cached.Index.Executables = make(map[string]struct{})
	}

	return cached.Index, cached.Timestamp, nil
}

func IsCacheStale(cachedAt time.Time) bool {
	if cachedAt.IsZero() {
		return true
	}
	if time.Since(cachedAt) > 24*time.Hour {
		return true
	}
	for _, dir := range []string{"/Applications", "/System/Applications"} {
		if modTimeChanged(dir, cachedAt) {
			return true
		}
	}
	return false
}

func modTimeChanged(dir string, since time.Time) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil {
			if info.ModTime().After(since) {
				return true
			}
		}
	}
	return false
}

func BuildOrLoadCached() (*AppIndex, error) {
	cached, cachedAt, err := LoadCached()
	if err == nil && cached != nil && !IsCacheStale(cachedAt) {
		mergeKnownApps(cached)
		return cached, nil
	}

	idx, err := Build()
	if err != nil {
		return nil, err
	}

	mergeKnownApps(idx)

	if err := idx.Save(); err != nil {
		return idx, nil
	}
	return idx, nil
}

func NormalizeAppName(name string) string {
	name = strings.ToLower(name)
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".app")
	return name
}

func SortStrings(s []string) []string {
	sort.Strings(s)
	return s
}
