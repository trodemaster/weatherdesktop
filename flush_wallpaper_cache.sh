#!/bin/bash

# macOS Wallpaper Cache Flush Script
# Safely clears wallpaper caches and corrupted data to prevent CPU hangs
# Designed to run daily via launchd

set -e  # Exit on any error

# Configuration
INDEX_PLIST="$HOME/Library/Application Support/com.apple.wallpaper/Store/Index.plist"

# Error handling
error_exit() {
    echo "ERROR: $1"
    exit 1
}

echo "Starting wallpaper cache flush..."
echo "================================="

# Function to kill wallpaper-related processes
kill_wallpaper_processes() {
    echo "Terminating wallpaper processes..."

    pkill -9 WallpaperAgent 2>/dev/null || true
    pkill -9 WallpaperImageExtension 2>/dev/null || true
    pkill -9 WallpaperAerialsExtension 2>/dev/null || true

    sleep 2  # Allow system to settle
}

# Function to safely remove directory contents
safe_remove_dir() {
    local dir_path="$1"
    local description="$2"

    if [ -d "$dir_path" ]; then
        echo "Clearing $description: $dir_path"

        # Fix permissions first
        chmod -R u+rw "$dir_path" 2>/dev/null || true

        # Remove all files
        find "$dir_path" -type f -delete 2>/dev/null || true

        # Remove empty directories
        find "$dir_path" -type d -empty -delete 2>/dev/null || true

        echo "✓ Cleared $description"
    else
        echo "Directory $description does not exist: $dir_path"
    fi
}

# Function to safely remove file
safe_remove_file() {
    local file_path="$1"
    local description="$2"

    if [ -f "$file_path" ]; then
        echo "Removing $description: $file_path"
        rm -f "$file_path" 2>/dev/null || true
        if [ -f "$file_path" ]; then
            echo "Warning: Failed to remove $file_path"
        else
            echo "✓ Removed $description"
        fi
    else
        echo "File $description does not exist: $file_path"
    fi
}

# 1. Kill wallpaper processes before deleting cache files
echo "Step 1: Terminating wallpaper processes..."
kill_wallpaper_processes

# 2. Remove the corrupted Index.plist file
echo "Step 2: Removing corrupted Index.plist..."
safe_remove_file "$INDEX_PLIST" "corrupted wallpaper index"

# 3. Clear TMPDIR-based wallpaper cache (from Go code)
echo "Step 3: Clearing TMPDIR wallpaper cache..."
USER_TMP_DIR=$(getconf DARWIN_USER_TEMP_DIR 2>/dev/null || echo "")
if [ -n "$USER_TMP_DIR" ]; then
    TMPDIR_CACHE="${USER_TMP_DIR}../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image"
    safe_remove_dir "$TMPDIR_CACHE" "TMPDIR wallpaper cache"
else
    echo "Warning: Could not determine user TMPDIR"
fi

# 4. Clear Container-based wallpaper cache (from Go code)
echo "Step 4: Clearing Container wallpaper cache..."
CONTAINER_CACHE="$HOME/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image"
safe_remove_dir "$CONTAINER_CACHE" "Container wallpaper cache"

# Clear wallpaper agent's entire Data directory to reset all internal state
# This forces WallpaperAgent to rebuild its catalog from scratch
AGENT_DATA_DIR="$HOME/Library/Containers/com.apple.wallpaper.agent/Data"
if [ -d "$AGENT_DATA_DIR" ]; then
    echo "Clearing wallpaper agent internal state (Data directory)..."
    # Remove all contents but keep the directory structure
    find "$AGENT_DATA_DIR" -mindepth 1 -delete 2>/dev/null || true
    echo "✓ Cleared wallpaper agent internal state"
fi

# Clear wallpaper caches in ~/Library/Caches
echo "Step 5: Clearing wallpaper caches in ~/Library/Caches..."
safe_remove_dir "$HOME/Library/Caches/com.apple.wallpaper" "wallpaper cache directory"

# 6. Clear wallpaper preferences that might cause Index.plist recreation
echo "Step 6: Clearing wallpaper preferences..."
WALLPAPER_PREFS=(
    "$HOME/Library/Preferences/com.apple.wallpaper.plist"
    "$HOME/Library/Preferences/com.apple.wallpaper.aerial.plist"
    "$HOME/Library/Preferences/com.apple.Home.wallpaper.plist"
)

for pref_file in "${WALLPAPER_PREFS[@]}"; do
    if [ -f "$pref_file" ]; then
        echo "Removing wallpaper preference: $(basename "$pref_file")"
        rm -f "$pref_file" 2>/dev/null || true
    fi
done

# 7. Clear com.apple.spaces.plist to remove historical "Collapsed Space" entries
# These accumulate over time and cause WallpaperAgent to create massive Index.plist
echo "Step 7: Clearing Mission Control historical spaces..."
SPACES_PLIST="$HOME/Library/Preferences/com.apple.spaces.plist"
if [ -f "$SPACES_PLIST" ]; then
    echo "Removing spaces preferences (will regenerate with active spaces only): $SPACES_PLIST"
    rm -f "$SPACES_PLIST" 2>/dev/null || true
    if [ -f "$SPACES_PLIST" ]; then
        echo "Warning: Failed to remove $SPACES_PLIST"
    else
        echo "✓ Removed historical spaces data"
    fi
fi

# 8. Clear Metal shader caches that WallpaperImageExtension uses
echo "Step 8: Clearing Metal shader caches..."
# Find and clear wallpaper extension Metal caches
find "$HOME/Library/Caches" -path "*wallpaper*metal*" -type d -exec rm -rf {} + 2>/dev/null || true
find "$TMPDIR" -path "*wallpaper*metal*" -type d -exec rm -rf {} + 2>/dev/null || true
# Clear the specific Metal cache directory from process inspector
METAL_CACHE_DIR="$TMPDIR../C/com.apple.wallpaper.extension.image/com.apple.wallpaper.extension.image/com.apple.metal"
if [ -d "$METAL_CACHE_DIR" ]; then
    rm -rf "$METAL_CACHE_DIR" 2>/dev/null || true
    echo "✓ Cleared Metal shader caches"
fi

# 9. Clear LaunchServices caches that may be corrupted
echo "Step 9: Clearing LaunchServices caches..."
LAUNCH_SERVICES_CACHE="$TMPDIR../0/com.apple.LaunchServices.dv"
if [ -d "$LAUNCH_SERVICES_CACHE" ]; then
    find "$LAUNCH_SERVICES_CACHE" -name "*.csstore" -delete 2>/dev/null || true
    echo "✓ Cleared LaunchServices caches"
fi

# 10. CRITICAL: Clear WallpaperImageExtension container cache via cfprefsd
#
# Root cause: WallpaperImageExtension stores a bookmark for every wallpaper ever
# set in ChoiceRequests.ImageFiles (accumulated 15K+ entries / 15MB over months).
# Deleting the plist file alone (rm -f) does NOT work because cfprefsd holds the
# data in its in-memory cache and restores the file immediately.
#
# Fix: use `defaults write` to overwrite the keys with empty arrays. This updates
# cfprefsd's cache directly so WallpaperImageExtension sees the clean state when
# it restarts. WallpaperAgent then only registers the current wallpaper
# (~/Pictures/Desktop/weather-desktop.jpg), keeping the plist tiny.
echo "Step 10: Clearing WallpaperImageExtension container cache via cfprefsd..."
# NOTE: As of Feb 24, 2026, plist clearing happens per-run in the `wd` command
# (pkg/desktop/macos.go clearContainerCache). This step is now a belt-and-suspenders
# fallback for backward compatibility or if the per-run cleanup is disabled.
EXT_DOMAIN="$HOME/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image"
EXT_PLIST_FILE="${EXT_DOMAIN}.plist"

if [ -f "$EXT_PLIST_FILE" ]; then
    FILE_SIZE=$(stat -f%z "$EXT_PLIST_FILE" 2>/dev/null || echo "0")
    echo "WallpaperImageExtension container preferences: ${FILE_SIZE} bytes"
fi

# Write empty arrays through cfprefsd (not rm -f, which bypasses the cache)
defaults write "$EXT_DOMAIN" "ChoiceRequests.ImageFiles"             -array 2>/dev/null || true
defaults write "$EXT_DOMAIN" "ChoiceRequests.Assets"                 -array 2>/dev/null || true
defaults write "$EXT_DOMAIN" "ChoiceRequests.CollectionIdentifiers"  -array 2>/dev/null || true
echo "✓ Cleared WallpaperImageExtension ChoiceRequests via cfprefsd"

# Also clear the main (non-container) preferences file if it exists
MAIN_PREFS="$HOME/Library/Preferences/com.apple.wallpaper.extension.image.plist"
if [ -f "$MAIN_PREFS" ]; then
    rm -f "$MAIN_PREFS" 2>/dev/null || true
fi

# 8. Ensure wallpaper directory exists but stays empty
echo "Step 8: Ensuring clean wallpaper directory structure..."
mkdir -p "$HOME/Library/Application Support/com.apple.wallpaper/Store" 2>/dev/null || true

echo "================================="
echo "Wallpaper cache flush complete!"
echo "System should now be responsive."
echo ""
echo "=== POST-FLUSH FILE SIZE REPORT ==="

# Report sizes of key plist files
echo "Wallpaper-related plist file sizes:"

# WallpaperImageExtension container plist (should be tiny after cleanup)
EXT_PLIST="$HOME/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image.plist"
if [ -f "$EXT_PLIST" ]; then
    SIZE=$(stat -f%z "$EXT_PLIST" 2>/dev/null || echo "0")
    if [ "$SIZE" -gt 1048576 ]; then
        SIZE_MB=$((SIZE / 1048576))
        echo "  WallpaperImageExtension container plist: ${SIZE_MB}MB (${SIZE} bytes) ⚠️  still large"
    else
        echo "  WallpaperImageExtension container plist: ${SIZE} bytes ✓"
    fi
else
    echo "  WallpaperImageExtension container plist: NOT FOUND"
fi

# System wallpaper Index.plist
INDEX_PLIST="$HOME/Library/Application Support/com.apple.wallpaper/Store/Index.plist"
if [ -f "$INDEX_PLIST" ]; then
    SIZE=$(stat -f%z "$INDEX_PLIST" 2>/dev/null || echo "0")
    echo "  System wallpaper Index.plist: ${SIZE} bytes"
else
    echo "  System wallpaper Index.plist: NOT FOUND"
fi

# Aerial wallpaper preferences
AERIAL_PLIST="$HOME/Library/Preferences/com.apple.wallpaper.aerial.plist"
if [ -f "$AERIAL_PLIST" ]; then
    SIZE=$(stat -f%z "$AERIAL_PLIST" 2>/dev/null || echo "0")
    echo "  Aerial wallpaper preferences: ${SIZE} bytes"
else
    echo "  Aerial wallpaper preferences: NOT FOUND"
fi

echo "================================="

# Optional: Send notification if terminal-notifier is available
if command -v terminal-notifier >/dev/null 2>&1; then
    terminal-notifier -title "Wallpaper Cache" -message "Cache flush completed successfully" -sound default 2>/dev/null || true
fi

exit 0
