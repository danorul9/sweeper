# Sweeper 🧹

**Intelligent macOS leftover detector & cleaner.**

Sweeper scans your Mac for files left behind by uninstalled applications — caches, support data, containers, saved state, and hidden dotdirs. It uses evidence-based "liveliness" scoring to distinguish between dead cruft and active data, with explainable confidence signals for every item.

> Not another "mac cleaner" that aggressively guesses. Sweeper shows you _why_ something is safe to delete.

## Features

- **Orphan Scanner** — scan `~/Library/Application Support`, `Caches`, `Containers`, `Saved Application State`, and more for leftover app files. Bundle ID matching, App Families (Google/Adobe/JetBrains groups), and fuzzy name matching.
- **Liveliness Detection** — evidence-based scoring for `~/.*` dotdirs. Checks file age, open handles (`lsof`), binary on PATH, newest child file age, and directory contents. Produces DEAD / STALE / COLD / ALIVE verdicts.
- **Explainable Results** — every item shows WHY it was flagged: "Modified 8 months ago", "No process has open handles", "Binary 'diffusionbee' not found on PATH".
- **Doctor** — detect orphaned LaunchAgents, dead symlinks, stale login items, corrupted plists, old Xcode derived data, and iOS backups.
- **Reclaim** — aggressively clean safe categories (caches, logs, temp files, saved state) without touching Application Support or Containers.
- **Large Files** — find files > 100 MB in user directories.
- **Duplicates** — find duplicate files using xxhash (fast first pass) + SHA-256 confirmation.
- **Apple System Protection** — `com.apple.*`, `App Store`, `CloudDocs`, `CallHistoryDB`, `SyncServices`, `.ssh`, `.cups`, `.aws`, `.gnupg` and other system paths are never shown. Hard-blocked at the scoring level.
- **Snapshot/Undo** — every deletion is recorded as a JSON snapshot. `sweeper undo` restores from Trash.
- **Interactive TUI** — bubbletea terminal UI with tabs, search, selection, and detail panel showing path, size, evidence signals, 3 biggest files, newest file age, and directory contents.
- **CLI mode** — all commands support `--json` for scripting and piping. Auto-detects when output is piped.

## Installation

```bash
brew install danorul9/tap/sweeper
```

Or build from source:

```bash
git clone https://github.com/danorul9/sweeper
cd sweeper
go build -o sweeper ./cmd/sweeper/
```

## Quick Start

```bash
# Interactive TUI hub
sweeper

# Scan for leftovers
sweeper scan

# Liveliness check for ~/.* dotdirs (from TUI menu)
sweeper scan --liveliness

# Explain why a specific folder is flagged
sweeper explain ~/Library/Application\ Support/ON1

# Run system diagnostics
sweeper doctor

# Reclaim safe caches and logs
sweeper reclaim

# Find large files
sweeper large

# Find duplicate files
sweeper dupes

# Undo last deletion
sweeper undo
```

## Commands

| Command | Description |
|---|---|
| `sweeper` | Interactive TUI hub (default) |
| `sweeper scan` | Scan for orphan app leftovers in ~/Library |
| `sweeper explain <path>` | Show why a folder is considered leftover |
| `sweeper doctor` | System diagnostics (zombie services, dead symlinks, etc.) |
| `sweeper reclaim` | Safe cleanup of caches, logs, temp files |
| `sweeper large` | Find files > 100 MB |
| `sweeper dupes` | Find duplicate files by checksum |
| `sweeper undo` | Restore files from last Trash snapshot |
| `sweeper stats` | Historical cleanup analytics |
| `sweeper watch` | Monitor ~/Library for new leftover creation |

All commands support `--json` for machine-readable output.

## Architecture

```
Filesystem Scanner
    ↓
Metadata Extractor
    ↓
App Index Builder (bundle IDs, /Applications, Homebrew, processes)
    ↓
3-Layer Matcher (exact → fuzzy → heuristic)
    ↓
Confidence Scorer (verdict + signals)
    ↓
Safety Filter (Apple system protection, blacklists)
    ↓
TUI (bubbletea) / CLI (JSON)
```

### Matching Pipeline

1. **Exact (bundle ID)** — `howett.net/plist` parses `Info.plist` from every installed `.app`. Exact `CFBundleIdentifier` match = 1.0 confidence.
2. **Fuzzy** — case-insensitive name similarity with multiple variants (lowercase, title case, `which -a` lookup).
3. **Heuristic** — reverse-domain parsing, camelCase splitting, vendor prefix matching, App Families (Google/Adobe/JetBrains groups).

### Liveliness Scoring

Each scanned item receives a score based on evidence signals:

| Signal | Weight | What it measures |
|---|---|---|
| `recent_mod` | +0.4 | Modified within 90 days |
| `old_mod` | -0.3 | Not modified in 6+ months |
| `open_handles` | +0.5 | `lsof` finds running process using this folder |
| `no_open_handles` | -0.1 | No process has open handles |
| `recent_child` | +0.3 | Newest child file is < 90 days old |
| `all_children_old` | -0.3 | Newest child is 6+ months old |
| `binary_on_path` | +0.6 | Corresponding binary found on `$PATH` |
| `app_installed` | +0.5 | Matcher says app is currently installed |
| `empty` | -0.4 | Directory is empty (0 bytes) |
| `apple_system` | +1.5 | Apple system path — protected, never shown |

### Verdicts

| Score | Label | Meaning |
|---|---|---|
| > 0.5 | ALIVE | Actively used — hidden from results |
| 0.3 – 0.5 | COLD | Some signs of life, user decides |
| -0.1 – 0.3 | (hidden) | No signal — filtered out |
| -0.5 – -0.1 | STALE | Weak evidence of disuse |
| < -0.5 | DEAD | Strong evidence — safe to delete |

### Protected Paths

The following are **never shown** in results, regardless of evidence:

- `com.apple.*` — all Apple system namespaces
- `App Store`, `Automator`, `CloudDocs`, `CallHistoryDB`, `SyncServices`, `AddressBook`, `iCloud`, `Spotlight`, `Knowledge`, `Dock`
- macOS daemons: `tipsd`, `contactsd`, `homeenergyd`, `identityservicesd`, `locationaccessstored`, `privatecloudcomputed`, etc.
- Hidden dotfiles: `.ssh`, `.gnupg`, `.aws`, `.cups`, `.IdentityService`, `.ServiceHub`

## Tech Stack

- **Go 1.26** — single binary, no runtime dependencies
- **bubbletea** — TUI framework
- **lipgloss** — terminal styling
- **cobra** — CLI framework
- **viper** — configuration
- **howett.net/plist** — macOS plist parsing
- **xxhash** — fast file hashing (duplicates)
- **fsnotify** — file system watching
- **golang.org/x/sync** — errgroup + semaphore worker pool

## Configuration

`~/.config/sweeper/config.yaml`:

```yaml
ignore:
  - Google
  - Adobe
  - Steam
safe_mode: true
```

## Trash Mechanism

Sweeper moves files to `~/.Trash` (never permanently deletes by default). The trash pipeline uses a multi-layered fallback chain:

1. **`os.Rename`** — fastest, works for ~95% of files on the same volume
2. **`chflags`** — clears immutable flags (`uchg`, `uappnd`, `schg`) on SIP-protected paths, then retries rename
3. **`osascript` (Finder move to trash)** — AppleScript fallback via Finder for cross-device moves and permission-denied cases
4. **`osascript` (Finder delete)** — last resort when Finder returns `-43` (file not found) or `-1728` (item gone), performs a permanent delete via Finder

When trashing `LaunchAgents` or `LaunchDaemons` paths, `launchctl unload` is called first to stop the service before moving the file.

## Full Disk Access

Sweeper's **AppleScript fallback** (step 3–4 above) and **`chflags`** (step 2) require **Full Disk Access** on macOS 10.14+ to operate on protected paths. Without it, deletions will fail on:

- `~/Library/Application Support/*`
- `~/Library/Containers/*`
- `~/Library/Caches/*`
- `~/Library/Group Containers/*`
- `~/Library/Saved Application State/*`
- SIP-protected dotdirs and system-owned files

**To grant Full Disk Access:**

1. Open **System Settings → Privacy & Security → Full Disk Access**
2. Click **+** (or unlock to add)
3. Add your terminal emulator (Terminal, iTerm2, Warp, etc.) — the one you run `sweeper` from
4. Also add **Finder** (required for the AppleScript fallback path)
5. Restart your terminal session

> Most files use `os.Rename` and work without Full Disk Access. You'll only hit the fallback on cross-volume moves, SIP-protected containers, or permission-restricted paths.

## Safety

- All deletions move to Trash — nothing is permanently deleted
- Every deletion is saved as a JSON snapshot for `undo`
- Apple system paths are hard-blocked at the scoring level
- No telemetry — `--share-telemetry` is opt-in and only sends anonymous folder + bundle ID pairs

## License

MIT
