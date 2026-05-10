package matcher

import (
	"testing"
	"time"

	"github.com/danorul9/sweeper/internal/appindex"
	"github.com/danorul9/sweeper/internal/core"
)

func TestExactBundleIDMatch(t *testing.T) {
	idx := appindex.NewAppIndex()
	idx.BundleIDs["com.docker.docker"] = struct{}{}

	m := New(idx)
	result := m.Match("com.docker.docker", "/fake/path", time.Now())

	if result.Verdict != core.VerdictInstalled {
		t.Errorf("expected Installed, got %v", result.Verdict)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result.Confidence)
	}
}

func TestExactAppNameMatch(t *testing.T) {
	idx := appindex.NewAppIndex()
	idx.Names["Docker Desktop"] = struct{}{}

	m := New(idx)
	result := m.Match("Docker Desktop", "/fake/path", time.Now())

	if result.Verdict != core.VerdictInstalled {
		t.Errorf("expected Installed, got %v", result.Verdict)
	}
	if result.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", result.Confidence)
	}
}

func TestFingerprintMatch(t *testing.T) {
	idx := appindex.NewAppIndex()
	m := New(idx)

	result := m.Match("Docker Desktop", "/fake/path", time.Now())

	if result.Verdict != core.VerdictLeftover {
		t.Errorf("expected Leftover for Docker Desktop, got %v", result.Verdict)
	}
	if result.Confidence < 0.8 {
		t.Errorf("expected high confidence, got %f", result.Confidence)
	}
}

func TestFingerprintMatchObsStudio(t *testing.T) {
	idx := appindex.NewAppIndex()
	m := New(idx)

	result := m.Match("obs-studio", "/fake/path", time.Now())

	if result.Verdict != core.VerdictLeftover {
		t.Errorf("expected Leftover for obs-studio, got %v", result.Verdict)
	}
}

func TestFingerprintMatchCode(t *testing.T) {
	idx := appindex.NewAppIndex()
	m := New(idx)

	result := m.Match("Code", "/fake/path", time.Now())

	if result.Verdict != core.VerdictLeftover {
		t.Errorf("expected Leftover for Code, got %v", result.Verdict)
	}
}

func TestSystemApplePath(t *testing.T) {
	if !IsSystemPath("com.apple.Safari") {
		t.Error("expected com.apple.Safari to be system path")
	}
	if IsSystemPath("com.docker.docker") {
		t.Error("expected com.docker.docker NOT to be system path")
	}
}

func TestSystemAppleFolder(t *testing.T) {
	if !IsSystemPath("Preferences") {
		t.Error("expected Preferences to be system folder")
	}
	if IsSystemPath("Docker") {
		t.Error("expected Docker NOT to be system folder")
	}
}

func TestAgeScoreYoung(t *testing.T) {
	signals := ageScore(time.Now())
	if len(signals) != 0 {
		t.Error("expected no signals for young file")
	}
}

func TestAgeScoreOld(t *testing.T) {
	signals := ageScore(time.Now().Add(-200 * 24 * time.Hour))
	if len(signals) == 0 {
		t.Fatal("expected signals for old file")
	}
	if signals[0].Weight < 0.05 {
		t.Errorf("expected weight >= 0.05, got %f", signals[0].Weight)
	}
}

func TestFuzzyScoreExact(t *testing.T) {
	score := fuzzyScore("Docker", "Docker")
	if score != 1.0 {
		t.Errorf("expected 1.0 for exact match, got %f", score)
	}
}

func TestFuzzyScoreContains(t *testing.T) {
	score := fuzzyScore("VSCode", "Code")
	if score < 0.7 {
		t.Errorf("expected high score for containing match, got %f", score)
	}
}

func TestFuzzyScoreNoMatch(t *testing.T) {
	score := fuzzyScore("abc", "xyz")
	if score != 0.0 {
		t.Errorf("expected 0.0 for no match, got %f", score)
	}
}

func TestMatchFingerprintDocker(t *testing.T) {
	fp := matchFingerprint("Docker Desktop")
	if fp == nil {
		t.Fatal("expected fingerprint for Docker Desktop")
	}
	if fp.Name != "Docker Desktop" {
		t.Errorf("expected Docker Desktop, got %s", fp.Name)
	}
}

func TestMatchFingerprintCode(t *testing.T) {
	fp := matchFingerprint("Code")
	if fp == nil {
		t.Fatal("expected fingerprint for Code")
	}
	if fp.Name != "Visual Studio Code" {
		t.Errorf("expected Visual Studio Code, got %s", fp.Name)
	}
}

func TestMatchFingerprintUnknown(t *testing.T) {
	fp := matchFingerprint("SomeRandomFolder123")
	if fp != nil {
		t.Errorf("expected nil for unknown folder, got %v", fp.Name)
	}
}

func TestSimpleMatchExact(t *testing.T) {
	match, err := simpleMatch("Docker Desktop", "Docker Desktop")
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected exact match")
	}
}

func TestSimpleMatchWildcard(t *testing.T) {
	match, err := simpleMatch("com.epicgames.*", "com.epicgames.EpicGamesLauncher")
	if err != nil {
		t.Fatal(err)
	}
	if !match {
		t.Error("expected wildcard match")
	}
}

func TestMatchResultSignals(t *testing.T) {
	idx := appindex.NewAppIndex()
	m := New(idx)

	result := m.Match("Docker Desktop", "/fake/path", time.Now())

	if len(result.Signals) == 0 {
		t.Error("expected at least one signal")
	}

	found := false
	for _, s := range result.Signals {
		if s.Kind == "fingerprint" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected fingerprint signal")
	}
}

func TestCamelCaseSplit(t *testing.T) {
	parts := camelCaseSplit("obs-studio")
	if len(parts) == 0 {
		t.Fatal("expected parts")
	}
	if parts[0] != "obs" {
		t.Errorf("expected 'obs', got '%s'", parts[0])
	}
}

func TestCamelCaseSplitReverseDomain(t *testing.T) {
	parts := camelCaseSplit("com.raycast.macos")
	if len(parts) == 0 {
		t.Fatal("expected parts")
	}
}

func TestFingerprintPlaywright(t *testing.T) {
	fp := matchFingerprint("ms-playwright")
	if fp == nil {
		t.Fatal("expected fingerprint for ms-playwright")
	}
	if fp.Name != "Playwright" {
		t.Errorf("expected Playwright, got %s", fp.Name)
	}
}

func TestFingerprintEpic(t *testing.T) {
	fp := matchFingerprint("Epic")
	if fp == nil {
		t.Fatal("expected fingerprint for Epic")
	}
	if fp.Name != "Epic Games Launcher" {
		t.Errorf("expected Epic Games Launcher, got %s", fp.Name)
	}
}

func TestFingerprintNotion(t *testing.T) {
	fp := matchFingerprint("Notion")
	if fp == nil {
		t.Fatal("expected fingerprint for Notion")
	}
}
