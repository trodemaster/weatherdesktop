# pluginkit Reference Guide

## Overview
`pluginkit` is a macOS command-line tool for managing the PlugInKit subsystem. It can query the plug-in database and make limited interventions for debugging and development.

## Basic Commands

### List All Plugins
```bash
pluginkit -m -v
```
- `-m` or `--match`: Scan all registered plug-ins
- `-v`: Verbose output (shows UUID, timestamp, and path)

### List Plugins Matching a Pattern
```bash
pluginkit -m -v | grep -i wallpaper
```

### List Plugins by Identifier
```bash
pluginkit -m -i com.apple.wallpaper.extension.aerials
```

## Status Indicators

When listing plugins, the first character indicates the user election state:

- **`-`** = User has elected to **ignore** (disabled)
- **`+`** = User has elected to **use** (explicitly enabled)
- **`!`** = User has elected to use for **debugger** use
- **`=`** = Plugin is **superseded** by another plugin
- **`?`** = **Unknown** user election state
- **(no prefix)** = **Default** state (enabled by default)

## Managing Plugin Elections

### Disable a Plugin
```bash
pluginkit -e ignore -i <identifier>
```

### Enable a Plugin
```bash
pluginkit -e use -i <identifier>
```

### Reset to Default
```bash
pluginkit -e default -i <identifier>
```

Where:
- `-e` = Perform election operation
- `-i` = Plugin identifier (shorthand for `NSExtensionIdentifier=identifier`)

## Example: Disabling WallpaperAerialsExtension

### 1. Find the Extension
```bash
pluginkit -m -v | grep -i aerials
```

Output:
```
com.apple.wallpaper.extension.aerials((null))	C2F5CC7C-2C83-5174-94CB-304B87DB0D3B	...	/System/Library/ExtensionKit/Extensions/WallpaperAerialsExtension.appex
```

### 2. Disable the Extension
```bash
pluginkit -e ignore -i com.apple.wallpaper.extension.aerials
```

### 3. Verify It's Disabled
```bash
pluginkit -m -i com.apple.wallpaper.extension.aerials
```

Output (with `-` prefix indicates disabled):
```
-    com.apple.wallpaper.extension.aerials((null))
```

### 4. Stop Running Process (if needed)
```bash
# Find the process
pgrep WallpaperAerialsExtension

# Kill it (replace PID with actual process ID)
kill <PID>

# Restart WallpaperAgent to ensure changes take effect
killall WallpaperAgent
```

## Important Notes

- **No reboot required**: After disabling via `pluginkit`, you can kill the running process and restart the related service (e.g., WallpaperAgent). The disabled plugin will not automatically restart.
- **Persistence**: The election setting persists across reboots. Once disabled, the plugin won't launch automatically.
- **System Extensions**: Some extensions are system extensions and may require additional steps or may be re-enabled by system updates.

## Finding All Wallpaper Extensions

```bash
pluginkit -m -v | grep -i wallpaper
```

Common wallpaper extensions:
- `com.apple.wallpaper.extension.aerials` - Aerial screensaver wallpapers
- `com.apple.wallpaper.extension.sonoma` - Sonoma wallpapers
- `com.apple.wallpaper.extension.monterey` - Monterey wallpapers
- `com.apple.wallpaper.extension.ventura` - Ventura wallpapers
- `com.apple.wallpaper.extension.sequoia` - Sequoia wallpapers
- `com.apple.wallpaper.extension.legacy` - Legacy wallpapers
- `com.apple.wallpaper.extension.macintosh` - Macintosh wallpapers
- `com.apple.wallpaper.extension.image` - Image wallpapers
- `com.apple.wallpaper.extension.gradient` - Gradient wallpapers
- `com.apple.wallpaper.extension.dynamic` - Dynamic wallpapers

## Additional Options

- `-A` or `--all-versions`: Match all versions of a plugin (not just latest)
- `-D` or `--duplicates`: Find all physical instances, even duplicates
- `-p` or `--protocol`: Match by protocol (shorthand for `NSExtensionPointName=protocol`)
- `-P` or `--platform`: Match by platform (macOS only: `native`, `maccatalyst`)
- `-a`: Explicitly add plugins (even if not normally eligible)
- `-r`: Explicitly remove plugins (may be re-added by automatic discovery)

## See Also

- `man pluginkit` - Full manual page
- `launchd(8)` - Service management
- `pkd(8)` - PluginKit daemon

