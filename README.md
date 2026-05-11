# Sweeper — macOS App Leftover Detector

[![Release](https://img.shields.io/github/v/release/danorul9/sweeper?label=version)](https://github.com/danorul9/sweeper/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/danorul9/sweeper)](https://go.dev/)
[![License](https://img.shields.io/github/license/danorul9/sweeper)](LICENSE)

Sweeper scans `~/Library` for orphaned files left behind by uninstalled applications and scores each item with explainable confidence. An interactive TUI hub gives you access to all features from one screen, or use individual CLI commands.

**Differentiator:** Bundle-ID intelligence, not heuristic guessing. If a folder's bundle ID matches an installed app, it's kept. If the app is gone, it's flagged. Suffix stripping, team-ID prefix handling, and prefix matching eliminate false positives before fuzzy matching runs.

## Install

```bash
# Via Homebrew
brew install danorul9/tap/sweeper

# Or build from source
git clone https://github.com/danorul9/sweeper.git
cd sweeper
make build
sudo make install
```

## Usage

```
sweeper                         # Interactive TUI hub — menu with all features
sweeper scan                    # CLI: scan for orphaned app leftovers
sweeper large                   # CLI: find large files
sweeper dupes                   # CLI: find duplicate files
sweeper doctor                  # CLI: run system health checks
sweeper reclaim                 # CLI: safe caches & logs scan
sweeper undo                    # CLI: show last undo snapshot
sweeper stats                   # CLI: show cleanup statistics
sweeper watch                   # CLI: watch for new orphan files
sweeper explain <path>          # CLI: explain why a folder is considered leftover
sweeper --version               # Show version
```

The TUI hub has 8 features, each also available as a CLI subcommand:

| Feature | CLI | Description |
|---------|-----|-------------|
| **Detected Apps** | — (TUI only) | List user-installed apps with sizes, select to trash |
| **Orphan Scanner** | `sweeper scan` | Find leftover files from uninstalled apps |
| **Large Files** | `sweeper large` | Find files over 100MB |
| **Duplicates** | `sweeper dupes` | Find duplicate files by checksum |
| **Doctor** | `sweeper doctor` | Zombie services, dead symlinks, system cruft |
| **Reclaim** | `sweeper reclaim` | Safe caches & logs only |
| **Undo Last Cleanup** | `sweeper undo` | Show last trash snapshot |
| **Stats** | `sweeper stats` | Historical cleanup analytics |

In the TUI, select a feature with arrow keys, press enter to run, browse results with vim keys, and press `esc` to return to the menu. Features support multi-select (`space`, `a` all, `n` none) and deletion (`d`).

## How It Works

```
TUI Hub ─┬─ Detected Apps ─ App Index Builder → Plist parser (filters macOS defaults)
          ├─ Orphan Scanner ─ Filesystem Scanner → Matcher → Scorer
          ├─ Large Files
          ├─ Duplicates ─ xxhash → SHA-256
          ├─ Doctor → launchctl unload before trash
          ├─ Reclaim
          ├─ Undo
          └─ Stats
```

1. **App Index** — Scans `/Applications`, `~/Applications`, Setapp. macOS default apps (`/System/Applications`, `com.apple.*` bundle IDs) are filtered out. Extracts `CFBundleIdentifier` from every `.app` via `howett.net/plist`. Cached at `~/.cache/sweeper/appindex.json`.
2. **Matcher** — Multi-layer pipeline: exact (bundle ID) → suffix-stripped (`.savedState`, `.ShipIt`, etc.) → team-ID-prefix stripped → bundle-ID-prefix match → fingerprint → fuzzy (`sahilm/fuzzy` library) → heuristic (reverse-domain, camelCase, vendor prefixes). Each produces a `Verdict` (`Installed` / `Leftover` / `Uncertain`) + `Confidence` (0.0–1.0) + ordered `Signal[]` list.
3. **Scorer** — Signals combine: bundle ID match (1.0), running process (0.95), fingerprint match (0.85), fuzzy similarity (0.50–0.65), age (±0.10).
4. **Trash** — Moves files to `~/.Trash/` natively (no AppleScript). Falls back to `osascript` for cross-device moves. For launch agent plists, runs `launchctl unload` before trashing.
5. **TUI Hub** — Bubbletea with lipgloss. Menu-driven hub with scroll-aware list views that dynamically fit header, tabs, list, detail panel, and footer within terminal height.

## Features

### Detected Apps

Builds and displays the user-installed app index with real file sizes (recursive `Contents/` walk). macOS default apps (Safari, Mail, Calendar, etc.) are automatically filtered out. Supports selection and trashing — select apps with `space`, trash with `d`.

### Orphan Scanner

Walks `~/Library/{Application Support,Caches,Logs,Containers,...}` and matches each folder against the installed app index. Items belonging to installed apps are filtered out; everything else is scored with explainable signals.

- **Bundle-ID matching** — exact, suffix-stripped, team-ID-prefixed, and bundle-ID-prefix matching
- **App Families** — Google, Adobe, JetBrains, Microsoft. If any family app is installed, vendor folders are kept.
- **Fingerprint Database** — 30+ curated fingerprints for apps like Docker, Slack, Discord, VS Code, OBS
- **Fuzzy Matching** — Uses `sahilm/fuzzy` library for substring similarity scoring
- **Process Correlation** — Checks `ps` before flagging. Running process → installed.
- **Age Scoring** — mtime tiers: <7d (suspicious), 30d+ (likely), 180d+ (highly likely)
- **Explainable results** — each item shows the exact signals that produced its verdict and confidence

### Other Features

- **Large Files** — Scans `~/Downloads`, `~/Desktop`, `~/Documents`, `~/Movies` for files over 100MB (configurable threshold).
- **Duplicates** — xxhash first pass + SHA-256 verification. Groups duplicate files and shows reclaimable space.
- **Doctor** — Checks launch agents, dead symlinks, Xcode DerivedData, iOS backups. Uses `launchctl unload` before trashing plists to prevent service crashes.
- **Reclaim** — Safe scan mode: caches, logs, saved state, temp items only.
- **Undo** — Loads the most recent delete snapshot and restores files from Trash.
- **Stats** — Historical cleanup analytics: scans, deletes, space freed, recent activity.

## Configuration

`~/.config/sweeper/config.yaml`:

```yaml
ignore:
  - Google
  - Adobe
safe_mode: false   # set true for caches + saved state only
```

## Development

```bash
make build     # Compile to ./bin/sweeper
make test      # Run all tests
make vet       # go vet
make install   # Build + copy to /usr/local/bin
```

Built with Go 1.26, [bubbletea](https://github.com/charmbracelet/bubbletea), [lipgloss](https://github.com/charmbracelet/lipgloss), [cobra](https://github.com/spf13/cobra), [viper](https://github.com/spf13/viper), [xxhash](https://github.com/cespare/xxhash), [howett.net/plist](https://howett.net/plist), [sahilm/fuzzy](https://github.com/sahilm/fuzzy), [willf/bloom](https://github.com/willf/bloom).

Find a bug or have a feature request? [Open an issue](https://github.com/danorul9/sweeper/issues).
