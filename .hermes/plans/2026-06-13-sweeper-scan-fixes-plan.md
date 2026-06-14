# Sweeper Scan Output Fixes — Implementation Plan

> **Goal:** Fix all identified bugs in the orphan scanner output — data safety gaps, scoring bugs, and missing protection — found during the evaluation of scan results on this machine.

**Architecture:** Three layers need changes: (1) the matcher pipeline (`matcher.go` — empty-string heuristic bug, fuzzy overflow), (2) Apple system path protection (`system.go`, `scanner.go` — missing entries), and (3) fingerprint matching (`fingerprints.go` — missing variant). Plus a `--json` output fix for `scan.go`.

**Tech Stack:** Go 1.26, bubbletea, cobra.

---

### Task 1: Fix empty-string bug in `heuristicMatch`

**Objective:** Stop the empty string produced by `splitReverseDomain(".foldername")` from matching every installed app via `strings.Contains(name, "")`.

**Files:**
- Modify: `internal/matcher/matcher.go:329-361`

**Step 1: Read the file to confirm current state**

Run: `read_file("internal/matcher/matcher.go", offset=329, limit=35)`

Expected: The `heuristicMatch` function with the `for _, part := range parts` loop.

**Step 2: Add empty-string guard**

Insert at the top of the loop body (after `for _, part := range parts {`):
```go
if part == "" {
    continue
}
```

This prevents `strings.Contains(name, "")` which is always true for any string.

**Step 3: Add test for empty-string guard**

In `internal/matcher/matcher_test.go`, add:
```go
func TestHeuristicEmptyPart(t *testing.T) {
    name, score := heuristicMatch(".hidden", []string{"SomeApp"})
    if score > 0 {
        t.Errorf("expected empty part to be skipped, got match '%s' at %.2f", name, score)
    }
}
```

**Step 4: Verify**

Run: `go build ./...`
Expected: success

Run: `go test ./internal/matcher/... -v -run TestHeuristic`
Expected: tests pass, including new test

**Step 5: Commit**

```bash
git add internal/matcher/matcher.go internal/matcher/matcher_test.go
git commit -m "fix: skip empty string parts in heuristicMatch (fixes false positive on hidden dirs)"
```

---

### Task 2: Cap fuzzy match score + add length-ratio guard

**Objective:** Fuzzy scores can exceed `1.0` (e.g., `virt-manager` → `Karabiner-VirtualHIDDevice-Manager` scores 1476). Cap at `1.0` AND add a length-ratio check so a 36-char name matching an 11-char query at `score=1.0` is still rejected.

**Files:**
- Modify: `internal/matcher/matcher.go:316-327`

**Step 1: Read current code**

Run: `read_file("internal/matcher/matcher.go", offset=316, limit=15)`

**Step 2: Add score capping + length ratio check**

Change:
```go
score := float64(best.Score) / 100.0
if score > 0.5 {
    return best.Str, score
}
```

To:
```go
score := float64(best.Score) / 100.0
if score > 1.0 {
    score = 1.0
}
// Length-ratio guard: reject when candidate is >3x the query length.
// Prevents short query (e.g. "virt-manager", 11 chars) from matching
// a much longer name (e.g. "Karabiner-VirtualHIDDevice-Manager", 36 chars)
// even if substring overlap produces a high raw score.
minLen := len(query)
if minLen > len(best.Str) {
    minLen = len(best.Str)
}
maxLen := len(query)
if maxLen < len(best.Str) {
    maxLen = len(best.Str)
}
if maxLen > minLen*3 {
    return "", 0
}
if score > 0.5 {
    return best.Str, score
}
```

**Step 3: Add test for length-ratio rejection**

In `internal/matcher/matcher_test.go`, add a test case:
```go
### Task 3: Add missing system paths — split into `AppleSystemFolders` + `ProtectedDotDirs`

**Objective:** Protect Apple system directories that are listed in the README but not in `IsSystemPath()`. Also protect user home dotdirs (`.ssh`, `.gnupg`) but in a separate list — mixing dotdirs into `AppleSystemFolders` is semantically odd since the original list was Apple's own App Support paths.

**Files:**
- Modify: `internal/matcher/system.go`

**Step 1: Read file**

Run: `read_file("internal/matcher/system.go")`

**Step 2: Add `ProtectedDotDirs` variable and update `IsSystemPath`**

After the existing `AppleSystemFolders` variable, add:
```go
var ProtectedDotDirs = []string{
    ".cups",
    ".ssh",
    ".gnupg",
    ".aws",
    ".identityservice",
    ".servicehub",
    ".localized",
}
```

Update `IsSystemPath` to check both lists:
```go
func IsSystemPath(name string) bool {
    for _, folder := range AppleSystemFolders {
        if filepathMatch(folder, name) {
            return true
        }
        if strings.EqualFold(folder, name) {
            return true
        }
    }
    for _, folder := range ProtectedDotDirs {
        if strings.EqualFold(folder, name) {
            return true
        }
    }
    return false
}
```

Add the missing Apple system paths to `AppleSystemFolders`:
```
"App Store",
"Automator",
"CloudDocs",
"CallHistoryDB",
"CallHistoryTransactions",
"SyncServices",
"AddressBook",
"Spotlight",
"Knowledge",
"Dock",
"DiskImages",
"Assistant",
"Baseband",
"ControlCenter",
"Mobile Sync",
"Summary-Events",
"HomeEnergyD",
"TipsD",
"NetworkServiceProxy",
"CloudKit",
"HomeKit",
"iCloudMailAgent",
"IdentityServicesD",
"LocationAccessStored",
"PrivateCloudComputeD",
"DifferentialPrivacy",
"StickersD",
"ContactsD",
"ICDD",
```

And:
```
"askpermissiond",
"PrivacyPreservingMeasurement",
"SiriTTS",
"SiriTTSService",
"ssu",
```

**Step 3: Verify**

Run: `go build ./...`
Expected: success

Run: `go test ./internal/matcher/... -v`
Expected: tests pass

**Step 4: Commit**

```bash
git add internal/matcher/system.go
git commit -m "refactor: split IsSystemPath into AppleSystemFolders + ProtectedDotDirs; add missing entries"
```
### Task 4: Add `open-whispr` variant to OpenWhispr fingerprint

**Objective:** The fingerprint has `"openwhispr"` but the actual folder is `"open-whispr"` (hyphenated). Both variants should match.

**Files:**
- Modify: `internal/matcher/fingerprints.go:174`

**Step 1: Read file**

Run: `read_file("internal/matcher/fingerprints.go", offset=168, limit=15)`

**Step 2: Add hyphen variant**

Change:
```go
Paths: []string{"openwhispr"},
```

To:
```go
Paths: []string{"openwhispr", "open-whispr"},
```

**Step 3: Build and verify**

Run: `go build ./...`
Expected: success

**Step 4: Commit**

```bash
git add internal/matcher/fingerprints.go
git commit -m "fix: add open-whispr variant to OpenWhispr fingerprint"
```

---

### Task 5: Add root-level `--json` flag (all subcommands)

**Objective:** The README states all commands support `--json`, but `sweeper scan --json` has no effect. Add `--json` as a root-level persistent flag so it works on every subcommand (scan, live, doctor, stats, etc.).

**Files:**
- Modify: `cmd/sweeper/cmd/root.go` (add flag definition + shared JSON output helper)
- Modify: `cmd/sweeper/cmd/scan.go` (use the shared helper)

**Step 1: Read files**

Run: `read_file("cmd/sweeper/cmd/root.go")`
Run: `read_file("cmd/sweeper/cmd/scan.go")`

**Step 2: Add `--json` persistent flag on root**

In `root.go`'s `init()`:
```go
rootCmd.PersistentFlags().Bool("json", false, "Output as JSON")
```

**Step 3: Add JSON output helper in root.go**

```go
func maybeJSON(cmd *cobra.Command, data any) (printed bool, err error) {
    if jsonOut, _ := cmd.Flags().GetBool("json"); !jsonOut {
        return false, nil
    }
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    if err := enc.Encode(data); err != nil {
        return true, fmt.Errorf("marshal output: %w", err)
    }
    return true, nil
}
```

**Step 4: Use in scan.go's RunE**

After obtaining `result`:
```go
if printed, err := maybeJSON(cmd, result); printed || err != nil {
    return err
}
```

BEFORE the table-format output section.

**Step 5: Build and verify**

Run: `go build ./...`
Expected: success

Run scan with JSON:
```bash
./bin/sweeper scan --json 2>/dev/null | head -5
```
Expected: valid JSON with `"items": [...]`

Verify other commands reject --json gracefully (no output currently, but no crash):
```bash
./bin/sweeper live --json 2>/dev/null
```
Expected: exits cleanly (even if JSON output is empty for unimplemented commands)

**Step 6: Commit**

```bash
git add cmd/sweeper/cmd/
git commit -m "feat: add root-level --json flag with shared output helper"
```

---

### Task 6: Re-evaluate and run full scan to verify fixes
**Objective:** Run a full scan after all fixes to verify the improvements.

**Step 1: Rebuild**

```bash
cd /Users/danojose/00Code/sweeper && make build
```
Expected: builds successfully

**Step 2: Run scan**

```bash
cd /Users/danojose/00Code/sweeper && ./bin/sweeper scan 2>/dev/null
```

Expected:
- `.hermes`, `.local` no longer at 100% (should be at 40% hidden_home baseline or filtered)
- `.cups`, `.IdentityService`, `.ServiceHub` no longer shown
- `SyncServices`, `CloudDocs` no longer shown
- `homeenergyd`, `tipsd`, `icloudmailagent` no longer shown
- `Assistant`, `SiriTTSService` no longer shown
- open-whispr at 85% (fingerprint match)
- No item with confidence > 100%

**Step 3: Verify `--json` works**

```bash
cd /Users/danojose/00Code/sweeper && ./bin/sweeper scan --json 2>/dev/null | python3 -m json.tool | head -10
```
Expected: valid JSON output with items array

**Step 4: Commit any remaining changes**

```bash
git add -A
git commit -m "fix: resolve scanner false positives after evaluation"
```

---

### Verification Checklist

	| # | Check | Expected |
	|---|-------|----------|
	| 1 | `go build ./...` | success |
	| 2 | `go test ./...` | all pass |
	| 3 | `TestHeuristicEmptyPart` | passes |
	| 4 | `TestFuzzyLengthRatio` | passes |
	| 5 | `.hermes` in scan output | confidence ≤ 40% or filtered |
	| 6 | `.local` in scan output | confidence ≤ 40% or filtered |
	| 7 | `.cups` in scan output | not shown |
	| 8 | `.IdentityService` in scan output | not shown |
	| 9 | `.ServiceHub` in scan output | not shown |
	| 10 | `CloudDocs` in scan output | not shown |
	| 11 | `SyncServices` in scan output | not shown |
	| 12 | `homeenergyd` in scan output | not shown |
	| 13 | `tipsd` in scan output | not shown |
	| 14 | `open-whispr` confidence | 85% (fingerprint match) |
	| 15 | `--json` flag on scan | produces valid JSON |
	| 16 | Fuzzy match score | never > 1.0 |
	| 17 | Root-level `--json` on non-scan commands | no crash |
