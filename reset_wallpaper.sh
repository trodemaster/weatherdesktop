#!/bin/bash

# macOS Wallpaper System Reset Script
# Resets wallpaper preferences and caches to resolve extension system issues

echo "Resetting macOS Wallpaper System..."
echo "==================================="

# Stop wallpaper services using launchctl
echo "Stopping wallpaper services..."
# Unload user domain services (try common service identifiers)
#launchctl bootout gui/$UID/com.apple.WallpaperAgent 2>/dev/null || true
#launchctl bootout gui/$UID/com.apple.wallpaper.agent 2>/dev/null || true
#launchctl bootout gui/$UID/com.apple.wallpaper.extension.aerials 2>/dev/null || true
#launchctl bootout gui/$UID/com.apple.wallpaper.extension.image 2>/dev/null || true
# Find and unload any other wallpaper-related services in user domain
#launchctl list | grep -i wallpaper | awk '{print $3}' | while read service; do
#    launchctl bootout gui/$UID/"$service" 2>/dev/null || true
#done
# Also try system domain (may require sudo)
# sudo launchctl bootout system/com.apple.WallpaperAgent 2>/dev/null || true
#sudo launchctl bootout system/com.apple.wallpaper.agent 2>/dev/null || true
#sudo launchctl bootout system/com.apple.wallpaper.export 2>/dev/null || true
#sudo launchctl bootout system/com.apple.wallpaper.extension.aerials 2>/dev/null || true
#sudo launchctl bootout system/com.apple.wallpaper.extension.image 2>/dev/null || true
# Find and unload any other wallpaper-related services in system domain
#sudo launchctl list | grep -i wallpaper | awk '{print $3}' | while read service; do
#    sudo launchctl bootout system/"$service" 2>/dev/null || true
#done
# Kill any remaining processes as fallback
killall WallpaperAgent 2>/dev/null || true
killall Wallpaper 2>/dev/null || true
killall WallpaperAerialsExtension 2>/dev/null || true
killall WallpaperImageExtension 2>/dev/null || true
# Give processes time to fully terminate
sleep 2

# Reset user preferences
echo "Resetting wallpaper preferences..."
# Delete all wallpaper-related preference files
find ~/Library/Preferences -name "com.apple.wallpaper*.plist" -delete 2>/dev/null || true
# Also reset desktop and screensaver preferences
defaults delete com.apple.desktop 2>/dev/null || true
defaults delete com.apple.screensaver 2>/dev/null || true

# Clear user-level wallpaper data
echo "Clearing wallpaper application support data..."
# Remove files first, then directories, with permission fixes
if [ -d ~/Library/Application\ Support/com.apple.wallpaper/ ]; then
    find ~/Library/Application\ Support/com.apple.wallpaper/ -type f -exec chmod 644 {} + 2>/dev/null || true
    find ~/Library/Application\ Support/com.apple.wallpaper/ -type d -exec chmod 755 {} + 2>/dev/null || true
    rm -rf ~/Library/Application\ Support/com.apple.wallpaper/ 2>/dev/null || true
    # Force remove if still exists
    [ -d ~/Library/Application\ Support/com.apple.wallpaper/ ] && sudo rm -rf ~/Library/Application\ Support/com.apple.wallpaper/ 2>/dev/null || true
fi

# Clear all wallpaper container data (full directories)
echo "Clearing wallpaper container data..."
# First fix permissions, then remove - only search for wallpaper containers, don't traverse others
find ~/Library/Containers -maxdepth 1 -name "com.apple.wallpaper*" -type d 2>/dev/null | while read dir; do
    if [ -d "$dir" ] && [ -r "$dir" ]; then
        find "$dir" -type f -exec chmod 644 {} + 2>/dev/null || true
        find "$dir" -type d -exec chmod 755 {} + 2>/dev/null || true
        rm -rf "$dir" 2>/dev/null || true
        # Force remove with sudo if still exists
        [ -d "$dir" ] && sudo rm -rf "$dir" 2>/dev/null || true
    fi
done

# Clear extension caches
echo "Clearing extension caches..."
# Remove from user caches - only search for wallpaper, skip inaccessible dirs
find ~/Library/Caches -name "*wallpaper*" -type d 2>/dev/null | while read dir; do
    [ -d "$dir" ] && [ -r "$dir" ] && chmod -R u+rw "$dir" 2>/dev/null && rm -rf "$dir" 2>/dev/null || true
done
# Remove from temp folders - only search current user's folder if accessible
USER_TMP_DIR=$(getconf DARWIN_USER_TEMP_DIR 2>/dev/null || echo "")
if [ -n "$USER_TMP_DIR" ] && [ -d "$(dirname "$USER_TMP_DIR")" ]; then
    find "$(dirname "$USER_TMP_DIR")" -name "*wallpaper*" -type d 2>/dev/null | while read dir; do
        if [ -d "$dir" ] && [ -r "$dir" ]; then
            chmod -R u+rw "$dir" 2>/dev/null || true
            rm -rf "$dir" 2>/dev/null || true
            [ -d "$dir" ] && sudo rm -rf "$dir" 2>/dev/null || true
        fi
    done
fi
# Also try the current user's home temp folder pattern
find /private/var/folders -name "*wallpaper*" -type d -user "$(whoami)" 2>/dev/null | while read dir; do
    if [ -d "$dir" ] && [ -r "$dir" ]; then
        chmod -R u+rw "$dir" 2>/dev/null || true
        rm -rf "$dir" 2>/dev/null || true
        [ -d "$dir" ] && sudo rm -rf "$dir" 2>/dev/null || true
    fi
done

# Clear Metal shader caches for wallpaper extensions
echo "Clearing Metal shader caches..."
# Use prune to skip inaccessible directories and only search user-owned paths
find /private/var/folders -type d ! -readable -prune -o -path "*/C/com.apple.wallpaper.extension.*/com.apple.metal/*" -print 2>/dev/null | while read item; do
    if [ -e "$item" ] && [ -r "$(dirname "$item")" ]; then
        chmod -R u+rw "$item" 2>/dev/null || true
        rm -rf "$item" 2>/dev/null || true
        [ -e "$item" ] && sudo rm -rf "$item" 2>/dev/null || true
    fi
done

# Clear XPC caches
echo "Clearing XPC caches..."
if [ -d ~/Library/Caches/com.apple.xpc/ ]; then
    chmod -R u+rw ~/Library/Caches/com.apple.xpc/ 2>/dev/null || true
    rm -rf ~/Library/Caches/com.apple.xpc/ 2>/dev/null || true
    [ -d ~/Library/Caches/com.apple.xpc/ ] && sudo rm -rf ~/Library/Caches/com.apple.xpc/ 2>/dev/null || true
fi

# Clear LaunchServices cache files in temp folders
echo "Clearing LaunchServices cache files..."
# Use prune to skip inaccessible directories
find /private/var/folders -type d ! -readable -prune -o -path "*/0/com.apple.LaunchServices.dv/com.apple.LaunchServices-*" -print 2>/dev/null | while read item; do
    if [ -e "$item" ] && [ -r "$(dirname "$item")" ]; then
        chmod -R u+rw "$item" 2>/dev/null || true
        rm -rf "$item" 2>/dev/null || true
        [ -e "$item" ] && sudo rm -rf "$item" 2>/dev/null || true
    fi
done

# Clear system wallpaper caches (requires sudo)
echo "Clearing system wallpaper caches..."
sudo rm -rf /Library/Caches/com.apple.wallpaper/
sudo rm -rf /Library/Caches/com.apple.WallpaperAgent/

# Clear system logging preference cache (requires sudo)
echo "Clearing system logging preference cache..."
sudo find /Library/Preferences/Logging -name ".plist-cache.*" -type f -delete 2>/dev/null || true

echo "==================================="
echo "Wallpaper system reset complete!"
echo "Please restart your computer for full effect."
echo "==================================="
