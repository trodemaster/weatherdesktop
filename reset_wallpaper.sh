#!/bin/bash

# macOS Wallpaper System Reset Script
# Resets wallpaper preferences and caches to resolve extension system issues

echo "Resetting macOS Wallpaper System..."
echo "==================================="

# Kill wallpaper processes first
echo "Stopping wallpaper processes..."
killall WallpaperAgent 2>/dev/null || true
killall Wallpaper 2>/dev/null || true

# Reset user preferences
echo "Resetting wallpaper preferences..."
defaults delete com.apple.wallpaper 2>/dev/null || true
defaults delete com.apple.wallpaper.agent 2>/dev/null || true
defaults delete com.apple.desktop 2>/dev/null || true
defaults delete com.apple.screensaver 2>/dev/null || true

# Clear user-level wallpaper data
echo "Clearing wallpaper application support data..."
rm -rf ~/Library/Application\ Support/com.apple.wallpaper/

# Clear wallpaper container caches
echo "Clearing wallpaper container caches..."
rm -rf ~/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/

# Clear extension caches
echo "Clearing extension caches..."
find ~/Library/Caches -name "*wallpaper*" -type d -exec rm -rf {} + 2>/dev/null || true
find /private/var/folders -name "*wallpaper*" -type d -exec rm -rf {} + 2>/dev/null || true

# Clear XPC caches
echo "Clearing XPC caches..."
rm -rf ~/Library/Caches/com.apple.xpc/

# Rebuild launch services (requires sudo)
echo "Rebuilding launch services..."
/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister -seed -lint -r -v -gc


# Clear system wallpaper caches (requires sudo)
echo "Clearing system wallpaper caches..."
sudo rm -rf /Library/Caches/com.apple.wallpaper/
sudo rm -rf /Library/Caches/com.apple.WallpaperAgent/

echo "==================================="
echo "Wallpaper system reset complete!"
echo "Please restart your computer for full effect."
echo "==================================="
