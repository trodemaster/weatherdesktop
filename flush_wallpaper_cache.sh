#!/bin/bash
#
# Wallpaper extension image cache cleanup
#
# Deletes UUID-named JPG files that accumulate in the WallpaperImageExtension
# container cache (~1.8MB per file, grows at 1 file per wd run).
#
# Runs daily at 01:00 via launchd in system context (root), which bypasses
# the macOS App Sandbox restrictions that prevent the wd binary from deleting
# files inside ~/Library/Containers/* without a permission prompt.
#
# The wd command handles per-run plist cleanup (ChoiceRequests.ImageFiles)
# via `defaults write` — no plist manipulation is needed here.
#
# Install: see tv.jibb.weatherdesktop.cacheflush.plist

USER_HOME="/Users/blake"
CACHE_DIR="${USER_HOME}/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches"

echo "$(date '+%Y-%m-%d %H:%M:%S') Starting wallpaper cache cleanup..."

if [ ! -d "$CACHE_DIR" ]; then
    echo "Cache directory does not exist: $CACHE_DIR"
    echo "$(date '+%Y-%m-%d %H:%M:%S') Nothing to do."
    exit 0
fi

# Count and measure before deletion
FILE_COUNT=$(find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
SIZE_BEFORE=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)

echo "Found ${FILE_COUNT} cache file(s) using ${SIZE_BEFORE} — deleting..."

# Delete all top-level JPG files (UUID-named cached copies of wallpaper images)
find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" -delete 2>/dev/null

FILE_COUNT_AFTER=$(find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
SIZE_AFTER=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)

echo "Done. ${FILE_COUNT_AFTER} file(s) remaining (${SIZE_AFTER})"
echo "$(date '+%Y-%m-%d %H:%M:%S') Cache cleanup complete."

exit 0
