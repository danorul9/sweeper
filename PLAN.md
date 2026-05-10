# Sweeper — Intelligent macOS Leftover Detector & Cleaner

> **Plan:** A transparent, explainable, developer-friendly macOS CLI for detecting and cleaning orphaned app leftovers. Differentiator: confidence-scored matching with bundle-ID intelligence, not heuristic guessing.

**Goal:** A single-binary CLI (`sweeper`) that scans macOS library paths, cross-references leftovers against a multi-source app index, scores confidence, explains its reasoning, and safely moves selected items to Trash.

**Product Angle:** _Intelligent leftover detection with explainable confidence scoring_ — not another "mac cleaner" that aggressively guesses.

**Architecture:**

```
Filesystem Scanner → Metadata Extractor → App Index Builder → Fingerprint Matcher → Confidence Scorer → Safety Filter → TUI
```

**Tech Stack:** Go 1.25+, bubbletea (TUI framework), lipgloss (styling), cobra (CLI), viper (config), **go-humanize** (readable sizes), **xxhash** (fast file hashing), fsnotify (watch)

---

## Design Principle: Smart Detection Tuned for Real Macs

Real test cases drawn from the author's machine:

**Must-Keep (installed apps that look like cruft):**
| Library Folder | Installed App | Why it's tricky |
|---|---|---|
| `Code` | VS Code | Generic noun, not an app name |
| `com.raycast.macos` | Raycast | Reverse-domain ≠ app name |
| `Google` | Google Drive | One folder spans multiple Google apps |
| `Blackmagic Design` | DaVinci Resolve | Folder name ≠ app name at all |
| `obs-studio` | OBS | App is "OBS", folder is "obs-studio" |
| `Capture One` | Capture One 22 | Version suffix divergence |

**Must-Catch (leftovers to flag):**
| Library Folder | Former App |
|---|---|
| `Docker Desktop` | Docker Desktop (user moved to OrbStack) |
| `ON1` | ON1 Photo Editor |
| `Epic` | Epic Games Launcher |
| `MetaTrader 5` | MetaTrader 5 |
| `ms-playwright` / `ms-playwright-go` | Playwright browser binaries |
| `us.zoom.xos` | Zoom |
| `DaisyDisk`, `EaseUS*`, `MacPhun`, `Skylum`, `CEF`, `io.sentry` | Various uninstalled tools |

---

## Architecture Components

### AppIndex — Multi-Source App Registry

**DO NOT rely on `mdfind` alone.** Build a composite index:

```go
type AppIndex struct {
    Names       map[string]struct{}       // "Visual Studio Code", "OBS"
    BundleIDs   map[string]struct{}       // "com.microsoft.VSCode", "com.obsproject.obs-studio"
    Vendors     map[string]struct{}       // "microsoft", "obsproject", "raycast"
    Executables map[string]struct{}       // binary names from /Applications/*.app/Contents/MacOS/
}
```

**Sources:**
1. `ls /Applications /System/Applications ~/Applications` — finds all .app bundles directly
2. **Mac App Store apps** — some live in `/Applications` but install via `/System/Library/Sandbox/` paths. Also check `mdfind 'kMDItemAppStoreCategory != ""'` for installed store apps.
3. **Setapp** — `~/Applications/Setapp` — check this directory separately; Setapp apps have unusual bundle paths
4. **Electron apps & auto-updaters** — Many Electron apps (Slack, Discord, VS Code) leave artifacts like `com.electron.*`, `CachedData`, `Code Cache`, `GPUCache` inside their support folders. Fingerprint these explicitly — the folder name often doesn't match the app name. Auto-updaters (Sparkle, Squirrel) also create "update" temp dirs.
5. `mdls -name kMDItemCFBundleIdentifier` on each .app — extracts exact bundle IDs
6. Homebrew Cask list — `brew list --cask 2>/dev/null`
7. Running processes — `ps aux` → match against app bundles
8. **Receipts database** — `/Library/Receipts/InstallHistory.plist` contains a history of all installed packages. Parse via plist to cross-reference deleted packages.
9. **system_profiler** — `system_profiler SPApplicationsDataType -json` as an extra signal source. Slow (2-3s), so cache the output or run only on first scan.

**Cache for subsequent runs:**
- Store the built AppIndex at `~/.cache/sweeper/appindex.json` with a timestamp
- On subsequent scans, if no new .app bundles found (check modification times on /Applications), reuse cache
- Use a **bloom filter** (`github.com/willf/bloom`) for fast "is this bundle ID known?" lookups during matching, avoiding map overhead

This resolves the "Code → VS Code", "com.raycast.macos → Raycast", "obs-studio → OBS" problems via bundle-ID matching, which is exact.

---

### Fingerprint Matcher — Curated Rule Base

For apps where bundle ID is unknown or the library folder doesn't match:

```go
type AppFingerprint struct {
    Name      string
    BundleIDs []string
    Vendors   []string
    Paths     []string     // Library folder names this app creates
}
```

Example fingerprint for Docker Desktop:

```go
{
    Name: "Docker Desktop",
    BundleIDs: []string{"com.docker.docker"},
    Paths: []string{"Docker", "Docker Desktop", "com.electron.dockerdesktop"},
}
```

Provides deterministic accuracy for known apps while the heuristic matcher handles unknown ones.

---

### Confidence Scorer — Verdict + Score + Signal List

The scorer produces a structured result, not a single blended number:

```go
type MatchResult struct {
    Verdict    Verdict       // installed | leftover | uncertain
    Confidence float64       // 0.0 – 1.0
    Signals    []Signal      // ordered list of what contributed
}

type Verdict int
const (
    Installed  Verdict = iota // app IS installed, do NOT flag
    Leftover                  // app is gone, safe to delete
    Uncertain                 // cannot decide, user decides
)

type Signal struct {
    Kind    string // "bundle_id" | "running_process" | "age" | "fuzzy" | etc.
    Detail  string // human-readable: "Bundle ID com.docker.docker matches installed Docker.app"
    Weight  float64 // how much this signal shifts confidence
}
```

**Signal strength rules:**
- **Exact bundle ID match** → `Installed, 1.0` — conclusive, overrides everything
- **Running process detected** → `Installed, 0.95` — mostly suppresses deletion, not a minor signal
- **Fuzzy similarity** → `Uncertain, 0.50-0.65` — never overrides strong evidence, only used when bundle ID and process checks are inconclusive
- **Last modified age** → used as a **secondary hint only** — adjusts confidence ±0.10 within an existing verdict, never flips it alone
- **Path fingerprint match** → `Leftover, 0.85` when fingerprint points to a known-uninstalled app
- **Generic token match** → `Uncertain, 0.20` — user must decide

**TUI display:**

```
  Docker Desktop                       1.2 GB     LEFTOVER (0.92)
  ✓ Bundle ID com.docker.docker not found
  ✓ No running Docker processes
  ✓ Fingerprint match: Docker Desktop
  ✓ Last modified 6 months ago

  Code                               1.0 GB     INSTALLED (0.95)
  ✓ Bundle ID com.microsoft.VSCode found
  ✓ Process "code" is running
  ⚠ Name "Code" is generic (low confidence if signals absent)
```

---

### Bundle-ID Intelligence (Critical)

Extract `CFBundleIdentifier` from every installed .app via:

```bash
mdls -name kMDItemCFBundleIdentifier /Applications/Raycast.app
# or
/usr/libexec/PlistBuddy -c "Print CFBundleIdentifier" /Applications/Raycast.app/Contents/Info.plist
```

Then matching is exact: folder `com.raycast.macos` → bundle ID `com.raycast.macos` → installed. No heuristic needed for apps that ship their bundle ID in a library folder name (which is most of them).

This alone fixes 80%+ of false positives.

---

### App Families — Model Overlapping Suites

Some vendors leave multiple folders that all belong to the same installed suite. Model this explicitly to avoid flagging pieces of an installed product as leftovers:

```go
type AppFamily struct {
    Vendor   string   // "Adobe", "Google", "JetBrains", "Microsoft"
    BundleIDs []string
    Folders   []string // library folder names across all locations
}

var AppFamilies = []AppFamily{
    {
        Vendor: "Google",
        BundleIDs: []string{"com.google.drive", "com.google.Chrome", "com.google.BackupAndSync"},
        Folders: []string{"Google", "Google Drive", "com.google.drive", "Google/Chrome"},
    },
    {
        Vendor: "Adobe",
        BundleIDs: []string{"com.adobe.*"},
        Folders: []string{"Adobe", "Adobe Creative Cloud", "com.adobe.*"},
    },
}
```

When any app in the family is installed, ALL folders belonging to that family are treated as "kept" — not flagged as leftovers. When NO app in the family is installed, all folders are collectively flagged as leftovers.

---

### Parallel File Walking

**NEVER call `du -sh` per item.** Use native Go:

```go
func dirSize(path string) int64 {
    var size int64
    filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        info, _ := d.Info()
        size += info.Size()
        return nil
    })
    return size
}
```

**APFS-aware:** Use `syscall.Stat_t.Blocks * 512` instead of `Info().Size()` for real disk usage (sparse files, clones).

**Format with go-humanize:**
```go
import "github.com/dustin/go-humanize"

func displaySize(path string) string {
    size := dirSize(path)
    return humanize.Bytes(uint64(size))
}
// Example: "1.2 GB" instead of "1234567890"
```

**Parallelize with errgroup + semaphore:**

```go
import (
    "golang.org/x/sync/errgroup"
    "golang.org/x/sync/semaphore"
)

func scanLibrary(location string, items []string, maxWorkers int64) ([]Leftover, error) {
    sem := semaphore.NewWeighted(maxWorkers)
    g, ctx := errgroup.WithContext(context.Background())
    results := make(chan Leftover, len(items))

    for _, item := range items {
        item := item
        if err := sem.Acquire(ctx, 1); err != nil {
            return nil, err
        }
        g.Go(func() error {
            defer sem.Release(1)
            leftover, err := analyze(item)
            if err != nil {
                return err
            }
            if leftover != nil {
                results <- *leftover
            }
            return nil
        })
    }
    // ...collect results
}
```

Workers = `runtime.NumCPU() * 2`. Pipeline: enumerate → match → size calc → aggregate.

**Safety limits:**
- Configurable timeout per worker: default 60s (via `context.WithTimeout`)
- Configurable depth limit: default 10 levels deep (skip `.git/`, `node_modules/` pyramids)
- First scan on a large Library may take 10-40s — cache the AppIndex + use bloom filter for subsequent runs

---

### Why-Explainable UI

Before showing an item, explain _why_ it's flagged:

```
  Docker Desktop                       1.2 GB     SAFE to delete
  └─ ✓ App not found in /Applications
  └─ ✓ Bundle ID "com.docker.docker" missing
  └─ ✓ No running Docker processes
  └─ ✓ Last modified 6 months ago
  └─ ✓ Fingerprint match "Docker Desktop"

  Code                               1.0 GB     UNCERTAIN
  └─ ⚠ "Code" is a generic word
  └─ ✗ App "Visual Studio Code" IS installed
  └─ ⚠ Bundle ID "com.microsoft.VSCode" seen
  └─ ✓ Process "code" is running (suggest keeping)
```

This is what builds user trust.

---

### Matcher — Fuzzy + Exact + Heuristic

Three-layer matching pipeline:

**Layer 1 — Exact (bundle ID):** `howett.net/plist` to parse `Info.plist` files and extract `CFBundleIdentifier` from every installed `.app`. Exact match on bundle ID is 1.0 confidence — no guessing needed.

**Layer 2 — Fuzzy (name similarity):** [`sahilm/fuzzy`](https://github.com/sahilm/fuzzy) for approximate name matching between library folder names and installed app names. Covers typos, abbreviations, and partial matches like "Obs-Studio" → "OBS", "VSCode" → "Visual Studio Code".

```go
import "github.com/sahilm/fuzzy"

func fuzzyMatch(folderName string, appNames []string) (string, float64) {
    matches := fuzzy.Find(folderName, appNames)
    if len(matches) == 0 {
        return "", 0.0
    }
    best := matches[0]
    // Convert score to 0.0-1.0 range
    score := float64(best.Score) / 100.0
    return best.String, score
}
```

**Layer 3 — Heuristic (fallback):** Reverse-domain parsing, camelCase splitting, vendor prefix matching, known fingerprint database. Used when exact and fuzzy both fail.

All three feed into the Confidence Scorer.

---

### Plist Parsing — Foundational for macOS Metadata

Every macOS app ships an `Info.plist` containing its identity. Parsing these is **critical** for accurate detection.

Use [`howett.net/plist`](https://howett.net/plist) — a Go-native plist parser (no XML/JSON dependency).

```go
import "howett.net/plist"

type AppInfo struct {
    BundleID string `plist:"CFBundleIdentifier"`
    Name     string `plist:"CFBundleName"`
    Version  string `plist:"CFBundleShortVersionString"`
}

func readAppInfo(path string) (*AppInfo, error) {
    plistPath := filepath.Join(path, "Contents", "Info.plist")
    f, err := os.Open(plistPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var info AppInfo
    decoder := plist.NewDecoder(f)
    if err := decoder.Decode(&info); err != nil {
        return nil, err
    }
    return &info, nil
}
```

**Used for:**
- Extracting `CFBundleIdentifier` from all installed apps (the single most accurate signal)
- Parsing `~/Library/LaunchAgents/*.plist` and `/Library/LaunchDaemons/*.plist` for zombie service detection
- Reading app metadata for fingerprinting

---

### Move to Trash — Native macOS API

Use [`trash`](https://github.com/otiai10/trash) instead of AppleScript — a Go library that wraps `NSFileManager` trash operations natively.

```go
import "github.com/otiai10/trash"

func safeDelete(paths []string) error {
    for _, p := range paths {
        if err := trash.Trash(p); err != nil {
            return err
        }
    }
    return nil
}
```

**Why trash over AppleScript:**
- Native macOS API, no Finder dependency
- No permission dialog popups
- Much faster (no osascript fork per file)
- Works with paths containing special characters
- Returns errors properly instead of silent failures

---

### Ignore Rules / Config

Path: `~/.config/sweeper/config.yaml`

```yaml
ignore:
  - Google
  - Adobe
  - JetBrains
  - Steam
safe_mode: true   # Only caches, logs, temp files
```

Implemented via viper for zero-effort config loading.

---

### Snapshot / Undo Support

Before each delete operation, write to `~/Library/Application Support/Sweeper/snapshots/YYYYMMDDTHHMMSS.json`:

```json
{
  "timestamp": "2026-05-10T14:30:00Z",
  "items": [
    {"path": "~/Library/Application Support/Docker Desktop", "size": 12000000}
  ]
}
```

Then `sweeper undo` reads the latest snapshot and restores from Trash (or re-download instructions for caches).

---

### Doctor — Expand Into a Killer Feature

` sweeper doctor` detects deeper system issues beyond leftover app folders:

- **Orphaned LaunchAgents/Daemons** — `.plist` files referencing non-existent `.app` paths
- **Dead symlinks** — broken links in `~/Library` (common after app deletion)
- **Leftover kernel extensions (kexts)** — rare on modern macOS but possible with old audio/VPN drivers. Check `/Library/Extensions/` and compare against installed apps
- **Corrupted .plist files** — parse every `.plist` in `~/Library/Preferences/`, flag files that fail to decode (these cause app launch hangs)
- **Old Xcode derived data** — `~/Library/Developer/Xcode/DerivedData/` can be gigabytes. Flag projects no longer on disk.
- **iOS device backups** — `~/Library/Application Support/MobileSync/Backup/` — large, often forgotten
- **Broken login items** — entries pointing to deleted apps

All diagnostics grouped in the TUI with per-item explainability.

---

### Age-Based Scoring

| Last Modified | Score Adjustment |
|---|---|
| < 7 days | Suspicious — lower confidence |
| 30+ days | Likely leftover |
| 180+ days | Highly likely leftover |

Combine with other signals for final score.

---

### Running Process Correlation

Before flagging anything, check if any running process has a path matching the library folder or its associated app. If a process is running → don't flag, or drop confidence significantly.

Prevents false positives from apps running from outside /Applications (setapp, portable .apps, etc.).

---

### Safe Mode vs Aggressive Mode — With a Reclaim Variant

**Safe (default):** Only caches, logs, temp files, Saved Application State. Deletable with high confidence only.
**Aggressive (--aggressive):** Includes Application Support, Containers, Preferences. Shows more low-confidence items.
**Reclaim (--reclaim):** Extremely aggressive on **known safe categories** — Caches, Logs, TemporaryItems, Saved Application State, WebKit/Chromium caches, SwiftUI preview data, node_modules/.cache — while staying conservative on everything else (never delete Application Support or Containers in --reclaim mode). Designed as a "safe to blast" button for casual users.

```
$ sweeper reclaim
Reclaiming 2.1 GB of safe-to-delete caches and logs...
  ✓ User app caches: 1.4 GB
  ✓ System logs: 320 MB
  ✓ Saved Application State: 180 MB
  ✓ TemporaryItems: 90 MB
  ✓ Xcode preview data: 110 MB
Done. 2.1 GB reclaimed.
```

---

### Watch Mode

```bash
sweeper watch
```

Uses `fsnotify` to monitor ~/Library paths for new folder creation. Tracks what apps create over time:

```
[14:32] Adobe Creative Cloud created 1.4 GB in ~/Library/Application Support/Adobe
[14:35] Docker Desktop created 856 MB in ~/Library/Containers/com.docker.docker
```

Could eventually auto-suggest cleanup.

---

### Community Fingerprint Database (Future)

Long-term: Allow opt-in anonymous submission of "unknown folder + bundle ID" pairs to improve the global fingerprint database.

```bash
sweeper scan --share-telemetry
```

Submits:
- Library folder name
- Detected bundle ID (if any)
- Whether it was ultimately deleted or kept (user override)

Collected data feeds into the curated fingerprint list in future releases. No personal paths or filenames are uploaded.

---

## CLI Commands

| Command | Description |
|---|---|
| `sweeper scan` | Interactive TUI — browse, select, delete leftovers |
| `sweeper scan --json` | Machine-readable output |
| `sweeper scan --aggressive` | Also scan containers, prefs, app support |
| `sweeper doctor` | Detect broken launch agents, orphaned kexts, dead symlinks |
| `sweeper explain <path>` | Show why a folder is considered leftover |
| `sweeper undo` | Restore from last cleanup snapshot |
| `sweeper watch` | Monitor for new leftover creation in real-time |
| `sweeper reclaim` | Safe mode — only caches, logs, temp files |
| `sweeper stats` | Historical cleanup analytics |
| `sweeper large` | Find files > 100MB in user directories |
| `sweeper dupes` | Find duplicate files by checksum |

All support `--dry-run` and `--json`.

---

## Implementation Phases

### Phase 1 — Core Engine (60% effort)

**Task 1.1:** Project scaffold — Go module, cobra CLI skeleton, directory structure
**Task 1.2:** AppIndex — multi-source app registry (scan /Applications, Mac App Store, Setapp, parse Info.plist, extract bundle IDs)
**Task 1.3:** Bundle-ID extractor — `howett.net/plist` to parse Info.plist from every installed .app
**Task 1.4:** App Families — Google, Adobe, JetBrains family model to group overlapping folders
**Task 1.5:** Fingerprint library — curated rules for 30+ common apps (including Electron patterns)
**Task 1.6:** Confidence-scored matcher — verdict+signals model, 3-layer pipeline (exact → fuzzy → heuristic)
**Task 1.7:** Parallel file system walker — errgroup/semaphore worker pool, configurable timeout, depth limit, APFS-aware sizes
**Task 1.8:** AppIndex cache — JSON cache at `~/.cache/sweeper/`, bloom filter for fast bundle ID lookups
**Task 1.9:** Config loader — viper-based ignore rules + safe/aggressive/reclaim modes
**Task 1.10:** Trash integration — `github.com/otiai10/trash` for native macOS trash
**Task 1.11:** Snapshot manager — JSON snapshots at `~/Library/Application Support/Sweeper/snapshots/`, undo restoration
**Task 1.12:** Test suite — testdata fixtures for matching edge cases (bundle IDs, fuzzy, families, Electron artifacts)

### Phase 2 — TUI (20% effort)

Built with **bubbletea** (framework) + **lipgloss** (styling) + **go-humanize** (size formatting).

**Task 2.1:** Add bubbletea + lipgloss deps

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

**Task 2.2:** Scan results view — tabs, scrollable list, confidence badges

bubbletea model structure with go-humanize:

```go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/dustin/go-humanize"
)

type model struct {
    results  *scanner.ScanResult
    cursor   int
    tab      int            // which location tab
    selected map[int]bool
    width    int
    height   int
    err      error
}

func (m model) View() string {
    header := lipgloss.NewStyle().Bold(true).Render(
        fmt.Sprintf("Sweeper — %s found",
            humanize.Bytes(uint64(m.results.TotalSize))))

    items := m.results.Items
    var body strings.Builder
    for i, item := range items {
        sizeStr := humanize.Bytes(uint64(item.Size))
        cursor := " "  // or "▸" if selected
        row := fmt.Sprintf("  %s  %-40s  %10s", cursor, item.Name, sizeStr)
        body.WriteString(row + "\n")
    }

    footer := lipgloss.NewStyle().Italic(true).Render(
        "↑↓ nav | space toggle | d delete | tab switch | q quit")
    return lipgloss.JoinVertical(lipgloss.Top, header, body.String(), footer)
}
```

**Task 2.3:** Detail/explain panel — "Why is this safe?" for selected item (split pane: list left, explanation right)

**Task 2.4:** Delete flow — selection, confirmation dialog, dry-run diff preview showing "Will remove: N files / M dirs / X GB"

**Task 2.5:** Keyboard navigation — vim bindings (j/k), search/filter by name

### Phase 3 — Advanced Features (10% effort)

**Task 3.1:** Zombie service detection — LaunchAgents/LoginItems/LaunchDaemons scan
**Task 3.2:** Watch mode — fsnotify-based monitoring
**Task 3.3:** Running process correlation (ps aux scan)
**Task 3.4:** Age-based scoring integration (mtime thresholds)

### Phase 4 — Duplicates & Large Files (5% effort)

**Task 4.1:** Large files scan — `sweeper large`, finds files > 100MB, shows in TUI

**Task 4.2:** Duplicate detection — `sweeper dupes` using **xxhash** (fast first pass) + SHA-256 (confirm). Walk directories, group by xxhash, flag groups > 1, SHA-256 to eliminate collisions.

```go
import "github.com/cespare/xxhash/v2"

func fastHash(path string) (uint64, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return 0, err
    }
    return xxhash.Sum64(data), nil
}
```

### Phase 5 — Polish (10% effort)

**Task 5.1:** README, ASCII-cast demo, screenshots
**Task 5.2:** Homebrew formula
**Task 5.3:** goreleaser CI — arm64 + amd64, auto-update
**Task 5.4:** Benchmark suite — matcher, size walker, parallel scan

---

## File Map

```
sweeper/
├── cmd/
│   └── sweeper/
│       ├── main.go              # Entry point
│       ├── root.go               # cobra root command
│       ├── scan.go               # "sweeper scan"
│       ├── doctor.go             # "sweeper doctor"
│       ├── explain.go            # "sweeper explain"
│       ├── undo.go               # "sweeper undo"
│       ├── watch.go              # "sweeper watch"
│       ├── reclaim.go            # "sweeper reclaim"
│       ├── stats.go              # "sweeper stats"
│       ├── large.go              # "sweeper large"
│       └── dupes.go              # "sweeper dupes"
├── internal/
│   ├── appindex/
│   │   ├── index.go              # AppIndex builder
│   │   ├── mdfind.go             # App scanning via /Applications
│   │   ├── bundleid.go           # Info.plist / mdls bundle ID extractor
│   │   ├── homebrew.go           # Homebrew cask list
│   │   ├── processes.go          # Running process correlation
│   │   └── zombies.go            # LaunchAgent/LoginItem scanner
│   ├── scanner/
│   │   ├── types.go              # Leftover, ScanResult, MatchResult
│   │   ├── scanner.go            # Parallel scan orchestrator
│   │   ├── walker.go             # APFS-aware parallel size walker
│   │   ├── locations.go          # Library path definitions
│   │   └── scanner_test.go
│   ├── matcher/
│   │   ├── matcher.go            # Confidence-scored matching engine
│   │   ├── strategies.go         # Heuristic strategies (reverse-domain, camel, fuzzy)
│   │   ├── fingerprints.go       # Curated AppFingerprint database
│   │   ├── system.go             # Apple system skip list
│   │   └── matcher_test.go       # Testdata fixtures
│   ├── config/
│   │   └── config.go             # Viper config loader
│   ├── actions/
│   │   ├── trash.go              # Native macOS trash API
│   │   ├── snapshot.go           # Snapshot save/restore/undo
│   │   └── log.go                # Operation log
│   └── ui/
│       ├── model.go              # bubbletea model
│       ├── view.go               # Rendering + explain panel
│       ├── update.go             # Event handling
│       └── styles.go             # lipgloss styles
├── testdata/
│   ├── fixtures/
│   │   ├── library/              # Mock ~/Library structure
│   │   └── applications/         # Mock /Applications structure
│   └── fingerprints_test.yaml    # Test cases for matcher
├── scripts/
│   └── trash-helper/             # Swift helper for native trash
├── go.mod
├── go.sum
├── README.md
└── PLAN.md
```

---

## Dependencies

```
github.com/spf13/cobra              # CLI framework
github.com/spf13/viper              # Config management
github.com/charmbracelet/bubbletea  # TUI framework
github.com/charmbracelet/lipgloss   # TUI styling
github.com/charmbracelet/bubbles    # TUI components (table, progress)
github.com/dustin/go-humanize       # Human-readable sizes
github.com/cespare/xxhash/v2        # Fast hashing for dupes
github.com/sahilm/fuzzy             # Fuzzy app name matching
howett.net/plist                    # macOS plist parsing (bundle IDs, launch agents)
github.com/otiai10/trash            # Native macOS trash API (no AppleScript)
github.com/fsnotify/fsnotify        # File system watcher (watch mode)
golang.org/x/sync/semaphore         # Worker pool
```

---

## Biggest Technical Risk

**The matcher.** That's where trust, accuracy, reputation, and usefulness all live. If matcher quality is mediocre, users stop trusting the tool immediately.

Mitigation:
- Bundle ID matching covers 80%+ of false positives (exact, not heuristic)
- Confidence tiers prevent disasters (nothing < 0.85 deleted without explicit override)
- Explainability surfaces why a decision was made
- Fingerprint library provides deterministic accuracy for known apps
- Active process check prevents deleting a running app's files
- Configurable ignore rules give users control

---

## Quick Start

```bash
cd /Users/danojose/00Code/sweeper
go mod init github.com/danojose/sweeper

# Core CLI & config
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest

# TUI framework
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest

# Human-readable sizes
go get github.com/dustin/go-humanize@latest

# Fast hashing for dupes
go get github.com/cespare/xxhash/v2@latest

# Fuzzy name matching
go get github.com/sahilm/fuzzy@latest

# macOS plist parsing (bundle IDs, launch agents, login items)
go get howett.net/plist@latest

# Native macOS trash (no AppleScript)
go get github.com/otiai10/trash@latest

# File watcher for watch mode
go get github.com/fsnotify/fsnotify@latest

# Worker pool
go get golang.org/x/sync@latest

go run ./cmd/sweeper/ scan
go test ./...
```
