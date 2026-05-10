# Sweeper — Intelligent macOS Leftover Detector & Cleaner

Sweeper scans `~/Library` for files left behind by uninstalled applications and scores each item with explainable confidence — so you know exactly *why* something is safe to delete.

**Differentiator:** Bundle-ID intelligence (not heuristic guessing). If a folder has the same bundle ID as an installed app, it's kept. If the app is gone, it's flagged. 80%+ of false positives eliminated before fuzzy matching even runs.

## Install

```bash
# Via Homebrew
brew install danorul9/tap/sweeper

# Or build from source
git clone https://github.com/danorul9/sweeper.git
cd sweeper
make build
sudo make install          # installs to /usr/local/bin
# or: cp ./bin/sweeper ~/bin/   # user-local install
```

## Usage

```
sweeper scan                    # Interactive TUI — browse, select, delete
sweeper scan --dry-run          # Terminal list with signals
sweeper scan --aggressive       # Also scan containers, prefs, app support
sweeper scan --json             # Machine-readable output
sweeper scan --share-telemetry  # Opt-in: submit unknown folder + bundle ID pairs

sweeper large                   # Find files > 100MB in user directories
sweeper dupes                   # Find duplicate files (xxhash + SHA-256)
sweeper doctor                  # Zombie services, dead symlinks, Xcode/iOS cruft
sweeper reclaim                 # Safe caches & logs only
sweeper explain <path>          # Why is this folder flagged?
sweeper undo                    # Restore last cleanup from Trash
sweeper stats                   # Historical cleanup analytics
```

All commands support `--json` and `--dry-run`.

## How It Works

```
Filesystem Scanner → App Index → Fingerprint Matcher → Confidence Scorer → TUI
```

1. **App Index** — Scans `/Applications`, Mac App Store, Setapp, Homebrew casks, running processes. Extracts `CFBundleIdentifier` from every `.app` via `howett.net/plist`. Cached at `~/.cache/sweeper/appindex.json`.
2. **Matcher** — 3-layer pipeline: exact (bundle ID) → fuzzy (`sahilm/fuzzy`) → heuristic (reverse-domain, camelCase, vendor prefixes). Each produces a `Verdict` (`Installed` / `Leftover` / `Uncertain`) + `Confidence` (0.0–1.0) + ordered `Signal[]` list.
3. **Scorer** — Signals combine: bundle ID match (1.0), running process (0.95), fingerprint match (0.85), fuzzy similarity (0.50–0.65), age (±0.10).
4. **TUI** — Bubbletea with lipgloss. Tabs per location type (Caches, Logs, App Support, Containers, etc.), right-panel explainer, vim keys, search/filter (`/`), multi-select + delete to Trash.

## Key Features

- **Bundle-ID Intelligence** — Parses `Info.plist` from every installed app. Exact match = conclusive. No guessing for most apps.
- **App Families** — Google, Adobe, JetBrains, etc. If any family app is installed, all its folders are kept.
- **Fingerprint Database** — 30+ curated fingerprints for apps like Docker, Slack, Discord, VS Code, OBS, Electron patterns.
- **App Families** — Google, Adobe, JetBrains: any family app installed → all family folders kept.
- **Process Correlation** — Checks `ps` before flagging. Running process → verdict is Installed.
- **Age Scoring** — mtime tiers: <7d (suspicious), 30d+ (likely), 180d+ (highly likely). Secondary hint only.
- **Zombie Service Detection** — Scans LaunchAgents/Daemons plists for references to deleted apps.
- **Snapshot Undo** — JSON snapshots before every delete. `sweeper undo` restores from Trash.
- **Telemetry (opt-in)** — `--share-telemetry` records unknown folder + bundle ID pairs locally for future fingerprint improvements.

## Configuration

`~/.config/sweeper/config.yaml`:

```yaml
ignore:
  - Google
  - Adobe
safe_mode: true
```

## Development

```bash
make build     # Compile to ./bin/sweeper
make test      # Run all 42+ tests
make vet       # go vet
make install   # Build + copy to /usr/local/bin
make dist      # Universal binary (arm64 + amd64 via lipo)
```

Built with Go 1.26, [bubbletea](https://github.com/charmbracelet/bubbletea), [lipgloss](https://github.com/charmbracelet/lipgloss), [cobra](https://github.com/spf13/cobra), [viper](https://github.com/spf13/viper), [xxhash](https://github.com/cespare/xxhash), [howett.net/plist](https://howett.net/plist).
