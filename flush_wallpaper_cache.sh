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

# Clear wallpaper caches in ~/Library/Caches
echo "Step 5: Clearing wallpaper caches in ~/Library/Caches..."
safe_remove_dir "$HOME/Library/Caches/com.apple.wallpaper" "wallpaper cache directory"

echo "================================="
echo "Wallpaper cache flush complete!"
echo "System should now be responsive."

# Optional: Send notification if terminal-notifier is available
if command -v terminal-notifier >/dev/null 2>&1; then
    terminal-notifier -title "Wallpaper Cache" -message "Cache flush completed successfully" -sound default 2>/dev/null || true
fi

exit 0
