#!/bin/bash
#
# Wallpaper extension image cache cleanup
#
# Runs daily at 01:00 via launchd in system context (root). Running as root:
#  - Bypasses the macOS TCC/App Sandbox prompt that fires when a user process
#    calls `defaults write` on another app's preference domain.
#  - Allows deleting files from ~/Library/Containers/* without permission dialogs.
#
# Two jobs performed:
#  1. Clear ChoiceRequests.ImageFiles plist entries (prevents WallpaperAgent hang)
#  2. Delete UUID-named JPG cache files (prevents 22GB+ disk accumulation)
#
# Install: see tv.jibb.weatherdesktop.cacheflush.plist

USER="blake"
USER_HOME="/Users/${USER}"
EXT_DOMAIN="${USER_HOME}/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image"
CACHE_DIR="${USER_HOME}/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches"

echo "$(date '+%Y-%m-%d %H:%M:%S') Starting wallpaper cache cleanup..."

# --- Step 1: Clear ChoiceRequests plist entries via cfprefsd ---
# Must run as the target user so cfprefsd updates the right user session's cache.
echo "Clearing ChoiceRequests plist entries..."
sudo -u "${USER}" defaults write "${EXT_DOMAIN}" "ChoiceRequests.ImageFiles"            -array 2>/dev/null && echo "  ✓ ChoiceRequests.ImageFiles cleared" || echo "  ✗ Failed to clear ChoiceRequests.ImageFiles"
sudo -u "${USER}" defaults write "${EXT_DOMAIN}" "ChoiceRequests.Assets"                -array 2>/dev/null && echo "  ✓ ChoiceRequests.Assets cleared"     || echo "  ✗ Failed"
sudo -u "${USER}" defaults write "${EXT_DOMAIN}" "ChoiceRequests.CollectionIdentifiers" -array 2>/dev/null && echo "  ✓ ChoiceRequests.CollectionIdentifiers cleared" || echo "  ✗ Failed"

# --- Step 2: Delete UUID-named JPG cache files ---
if [ ! -d "$CACHE_DIR" ]; then
    echo "Cache directory does not exist: $CACHE_DIR"
else
    FILE_COUNT=$(find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_BEFORE=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)
    echo "Deleting ${FILE_COUNT} cache file(s) using ${SIZE_BEFORE}..."
    find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" -delete 2>/dev/null
    FILE_COUNT_AFTER=$(find "$CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_AFTER=$(du -sh "$CACHE_DIR" 2>/dev/null | cut -f1)
    echo "  ✓ ${FILE_COUNT_AFTER} file(s) remaining (${SIZE_AFTER})"
fi

echo "$(date '+%Y-%m-%d %H:%M:%S') Cache cleanup complete."
exit 0
