#!/bin/bash
#
# Wallpaper extension image cache cleanup
#
# Runs daily at 01:00 via launchd as the logged-in user (LaunchAgent). Running
# as the user allows defaults write to reach cfprefsd in the active session,
# and all cache files are user-owned so no elevated privileges are needed.
#
# Four jobs performed:
#  1. Clear ChoiceRequests plist entries via cfprefsd (prevents WallpaperAgent hang)
#  2. Delete UUID-named JPG cache files in ~/Library/Containers
#  3. Delete UUID-named JPG cache files in ~/Library/Caches (var/folders)
#  4. Delete rendered BMP cache files from wallpaper.agent (prevents 350GB+ accumulation)
#
# Install: see tv.jibb.weatherdesktop.cacheflush.plist

USER_HOME="${HOME}"
EXT_DOMAIN="${USER_HOME}/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image"
CONTAINERS_CACHE_DIR="${USER_HOME}/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches"
VAR_CACHE_DIR="$(getconf DARWIN_USER_CACHE_DIR)/com.apple.wallpaper.extension.image"
BMP_CACHE_DIR="${USER_HOME}/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image"

echo "$(date '+%Y-%m-%d %H:%M:%S') Starting wallpaper cache cleanup..."

# --- Step 1: Clear ChoiceRequests plist entries via cfprefsd ---
# cfprefsd loads a .blocked sentinel at startup that prevents external defaults write calls.
# Deleting the sentinel and restarting cfprefsd clears the in-memory block; cfprefsd does not
# recreate the sentinel on restart. The 1s sleep lets cfprefsd fully restart before writing.
echo "Clearing ChoiceRequests plist entries..."
BLOCKED_FILE="${EXT_DOMAIN}.plist.blocked"
if [ -f "${BLOCKED_FILE}" ]; then
    rm "${BLOCKED_FILE}" && echo "  ✓ Removed .blocked sentinel" || echo "  ✗ Failed to remove .blocked sentinel"
fi
killall cfprefsd 2>/dev/null; sleep 1
defaults write "${EXT_DOMAIN}" "ChoiceRequests.ImageFiles"            -array 2>/dev/null && echo "  ✓ ChoiceRequests.ImageFiles cleared" || echo "  ✗ Failed to clear ChoiceRequests.ImageFiles"
defaults write "${EXT_DOMAIN}" "ChoiceRequests.Assets"                -array 2>/dev/null && echo "  ✓ ChoiceRequests.Assets cleared"     || echo "  ✗ Failed"
defaults write "${EXT_DOMAIN}" "ChoiceRequests.CollectionIdentifiers" -array 2>/dev/null && echo "  ✓ ChoiceRequests.CollectionIdentifiers cleared" || echo "  ✗ Failed"

# --- Step 2: Delete UUID-named JPG cache files (Containers) ---
if [ ! -d "$CONTAINERS_CACHE_DIR" ]; then
    echo "Containers cache directory does not exist: $CONTAINERS_CACHE_DIR"
else
    FILE_COUNT=$(find "$CONTAINERS_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_BEFORE=$(du -sh "$CONTAINERS_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "Deleting ${FILE_COUNT} JPG cache file(s) using ${SIZE_BEFORE} (Containers)..."
    find "$CONTAINERS_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" -delete 2>/dev/null
    FILE_COUNT_AFTER=$(find "$CONTAINERS_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_AFTER=$(du -sh "$CONTAINERS_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "  ✓ ${FILE_COUNT_AFTER} file(s) remaining (${SIZE_AFTER})"
fi

# --- Step 3: Delete UUID-named JPG cache files (var/folders) ---
if [ ! -d "$VAR_CACHE_DIR" ]; then
    echo "Var cache directory does not exist: $VAR_CACHE_DIR"
else
    FILE_COUNT=$(find "$VAR_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_BEFORE=$(du -sh "$VAR_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "Deleting ${FILE_COUNT} JPG cache file(s) using ${SIZE_BEFORE} (var/folders)..."
    find "$VAR_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" -delete 2>/dev/null
    FILE_COUNT_AFTER=$(find "$VAR_CACHE_DIR" -maxdepth 1 -type f -name "*.jpg" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_AFTER=$(du -sh "$VAR_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "  ✓ ${FILE_COUNT_AFTER} file(s) remaining (${SIZE_AFTER})"
fi

# --- Step 4: Delete rendered BMP cache files from wallpaper.agent ---
# These 25MB BMPs are rendered wallpaper frames; macOS regenerates them on demand.
# They accumulate at ~25MB per weather update and can exceed 350GB+ if unchecked.
if [ ! -d "$BMP_CACHE_DIR" ]; then
    echo "BMP cache directory does not exist: $BMP_CACHE_DIR"
else
    FILE_COUNT=$(find "$BMP_CACHE_DIR" -maxdepth 1 -type f -name "*.bmp" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_BEFORE=$(du -sh "$BMP_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "Deleting ${FILE_COUNT} BMP cache file(s) using ${SIZE_BEFORE}..."
    find "$BMP_CACHE_DIR" -maxdepth 1 -type f -name "*.bmp" -delete 2>/dev/null
    FILE_COUNT_AFTER=$(find "$BMP_CACHE_DIR" -maxdepth 1 -type f -name "*.bmp" 2>/dev/null | wc -l | tr -d ' ')
    SIZE_AFTER=$(du -sh "$BMP_CACHE_DIR" 2>/dev/null | cut -f1)
    echo "  ✓ ${FILE_COUNT_AFTER} BMP file(s) remaining (${SIZE_AFTER})"
fi

echo "$(date '+%Y-%m-%d %H:%M:%S') Cache cleanup complete."
exit 0
