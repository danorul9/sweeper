#!/usr/bin/env python3
"""
scan_leftovers.py — Legacy quick scanner (logic now in sweeper Go binary).

Scans ~/Library/Application Support/, Caches/, and Saved Application State/
for folders left behind by uninstalled applications. Cross-references against
installed apps via mdfind and a known-app lookup table.

Prefer 'sweeper scan' for the full-featured Go implementation.

Usage:
    sweeper scan                           # Full-featured Go version (preferred)
    python3 scan_leftovers.py              # Show all leftovers (legacy)
    python3 scan_leftovers.py --delete     # Move confirmed to Trash
    python3 scan_leftovers.py --json       # JSON output for scripting
"""

import subprocess
import argparse
import json
import os
import sys

# ── Known installed apps (apps that are definitely installed) ──
KNOWN_APPS = {x.lower() for x in [
    "safari", "terminal", "finder", "mail", "messages", "notes", "calendar",
    "contacts", "reminders", "photos", "music", "tv", "books",
    "maps", "facetime", "preview", "textedit", "calculator", "chess",
    "activity monitor", "disk utility", "console", "system settings",
    "screenshot", "image capture", "stickies", "time machine", "automator",
    "font book", "iina", "arc", "iterm", "iterm2", "jdownloader2", "obs",
    "localsend", "orbstack", "ollama", "lm studio", "jan", "raycast",
    "zoom.us", "surfshark", "bitwarden", "brave browser",
    "google chrome", "pixelmator pro", "capture one 22", "davinci resolve",
    "karabiner-elements", "keyclu", "mailspring",
    "microsoft word", "microsoft excel", "microsoft powerpoint",
    "microsoft defender", "google drive", "google docs",
    "google sheets", "google slides", "whatsapp", "tidal", "calibre",
    "balenaetcher", "pasty", "runcat", "appcleaner", "zerotier",
    "omnidisksweeper", "onyx", "iphone mirroring", "image playground",
    "passwords", "freeform", "journal", "voice memos",
    "photo booth", "vnc viewer", "windows app",
    "microsoft remote desktop", "loops", "openswarm", "remio",
    "adobe acrobat reader", "siri", "shortcuts",
    "home", "clock", "weather", "stocks", "news", "tips",
    "visual studio code", "code", "vscode",
    "adobe", "adobe creative cloud", "notion",
    "msty", "firefox", "spotify",
]}

# ── Apple system folders to always skip ──
APPLE_PREFIXES = ("com.apple.", "apple.")


def get_installed_apps() -> set:
    """Get currently installed app names via mdfind (Spotlight)."""
    installed = set()
    try:
        result = subprocess.run(
            ["mdfind", 'kMDItemKind == "Application"',
             "-onlyin", "/Applications",
             "-onlyin", "/System/Applications",
             "-onlyin", os.path.expanduser("~/Applications")],
            capture_output=True, text=True, timeout=15
        )
        for line in result.stdout.strip().split("\n"):
            name = line.strip()
            if name:
                # Extract app name from path, strip .app
                app_name = os.path.basename(name)
                if app_name.lower().endswith(".app"):
                    app_name = app_name[:-4]
                installed.add(app_name.lower())
    except Exception:
        pass
    return installed


def is_installed(folder_name: str, installed_apps: set) -> bool:
    """Check if a library folder name corresponds to an installed app."""
    n = folder_name.lower().strip()

    # Direct match
    if n in installed_apps or n in KNOWN_APPS:
        return True

    # Reverse domain: com.company.AppName -> try AppName, Company AppName
    parts = n.split(".")
    if len(parts) >= 3:
        last = parts[-1]
        if last in installed_apps or last in KNOWN_APPS:
            return True
        combined = f"{parts[-2]} {parts[-1]}"
        if combined in installed_apps or combined in KNOWN_APPS:
            return True

    return False


def get_folder_size(path: str) -> int:
    """Get folder size using du (bytes)."""
    try:
        result = subprocess.run(
            ["du", "-sk", path],
            capture_output=True, text=True, timeout=30
        )
        size_kb = int(result.stdout.strip().split()[0])
        return size_kb * 1024
    except Exception:
        return 0


def format_size(bytes_: int) -> str:
    """Human-readable size."""
    if bytes_ >= 1_000_000_000:
        return f"{bytes_ / 1_000_000_000:.1f} GB"
    elif bytes_ >= 1_000_000:
        return f"{bytes_ / 1_000_000:.1f} MB"
    elif bytes_ >= 1_000:
        return f"{bytes_ / 1_000:.0f} KB"
    return f"{bytes_} B"


def scan_location(loc_name: str, loc_path: str, installed_apps: set) -> list[dict]:
    """Scan a ~/Library subdirectory for leftover folders."""
    leftovers = []
    full_path = os.path.expanduser(f"~/Library/{loc_path}")

    if not os.path.isdir(full_path):
        return leftovers

    for entry in sorted(os.listdir(full_path)):
        entry_path = os.path.join(full_path, entry)

        if not os.path.isdir(entry_path):
            continue

        # Skip Apple system folders
        if entry.lower().startswith(APPLE_PREFIXES):
            continue

        # Check if this corresponds to an installed app
        if is_installed(entry, installed_apps):
            continue

        size = get_folder_size(entry_path)
        leftovers.append({
            "name": entry,
            "path": entry_path,
            "size": size,
            "size_str": format_size(size),
            "location": loc_name,
        })

    return leftovers


def move_to_trash(path: str) -> bool:
    """Move a file/folder to Trash using osascript."""
    try:
        script = f'tell app "Finder" to delete POSIX file "{path}"'
        subprocess.run(["osascript", "-e", script], capture_output=True, timeout=10)
        return True
    except Exception:
        return False


def main():
    parser = argparse.ArgumentParser(
        description="Scan for leftover app folders from uninstalled apps"
    )
    parser.add_argument("--delete", action="store_true",
                       help="Move confirmed leftovers to Trash")
    parser.add_argument("--json", action="store_true",
                       help="Output as JSON")
    parser.add_argument("--min-size", type=str, default="0",
                       help="Minimum file size to show (e.g., '1MB', '100KB')")
    args = parser.parse_args()

    # Parse min size
    min_size = 0
    if args.min_size != "0":
        size_str = args.min_size.upper()
        multipliers = {"KB": 1_000, "MB": 1_000_000, "GB": 1_000_000_000, "B": 1}
        for suffix, mult in multipliers.items():
            if size_str.endswith(suffix):
                try:
                    min_size = float(size_str.replace(suffix, "")) * mult
                except ValueError:
                    pass
                break

    print("🔍 Scanning for leftover app folders...", file=sys.stderr)
    installed_apps = get_installed_apps() | KNOWN_APPS
    print(f"   Found {len(installed_apps)} installed apps in index", file=sys.stderr)

    locations = [
        ("Application Support", "Application Support"),
        ("Caches", "Caches"),
        ("Saved Application State", "Saved Application State"),
        ("Containers", "Containers"),
    ]

    all_leftovers = []
    for loc_name, loc_path in locations:
        results = scan_location(loc_name, loc_path, installed_apps)
        # Filter by min size
        results = [r for r in results if r["size"] >= min_size]
        all_leftovers.extend(results)

    # Sort by size (largest first)
    all_leftovers.sort(key=lambda x: x["size"], reverse=True)

    total_size = sum(r["size"] for r in all_leftovers)

    if args.json:
        output = {
            "total_items": len(all_leftovers),
            "total_size": total_size,
            "total_size_str": format_size(total_size),
            "items": all_leftovers,
        }
        print(json.dumps(output, indent=2))
        return

    # ── Display ──
    print(f"\n{'=' * 70}")
    print(f"  LEFTOVER FOLDERS — {len(all_leftovers)} items, {format_size(total_size)}")
    print(f"{'=' * 70}")

    current_loc = ""
    for item in all_leftovers:
        if item["location"] != current_loc:
            print(f"\n📁 ~/Library/{item['location']}/")
            current_loc = item["location"]
        print(f"  {item['size_str']:>8}  {item['name']}")

    print(f"\n{'─' * 70}")
    print(f"  Total: {len(all_leftovers)} items — {format_size(total_size)}")
    print()

    # ── Delete mode ──
    if args.delete:
        print("🧹 Moving leftovers to Trash...")
        for item in all_leftovers:
            if move_to_trash(item["path"]):
                print(f"  ✓ {item['name']}")
            else:
                print(f"  ✗ {item['name']} — failed")
        print("Done.")


if __name__ == "__main__":
    main()
