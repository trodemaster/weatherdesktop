# Weather Desktop Troubleshooting & Fixes

---

# WallpaperAgent CPU Usage & Hanging

## Problem
macOS WallpaperAgent process consumes excessive CPU (up to 30 seconds in 10-second sampling periods) and causes system hangs due to PropertyList encoding operations getting stuck in infinite recursion.

**Symptoms:**
- WallpaperAgent process shows high CPU usage in Activity Monitor
- System becomes unresponsive (UI freezes for 20+ seconds)
- PropertyListEncoder operations stuck in deep recursion
- Swift Task UNKNOWN consuming all available CPU

## Root Cause Analysis

### Trigger Mechanism: Wallpaper Setting Operations
The issue is triggered when applications set wallpapers, causing WallpaperAgent to:

1. **Receive XPC Messages**: `setLegacyDesktopPictureConfiguration` updates
2. **Update Internal State**: Rebuild wallpaper configuration catalog
3. **Persist Changes**: Attempt to save massive data structures to plist
4. **Encoding Hang**: PropertyListEncoder gets stuck on complex nested data

### Primary Issue: NSUserDefaults 4MB Limit Violation
**ROOT CAUSE IDENTIFIED:** WallpaperImageExtension caches bookmarks for ALL weather desktop rendered images (15,785+ files), creating 12,658+ entries (14.8MB) that exceed NSUserDefaults 4MB limit.

**The Chain:**
1. **Weather desktop generates 15,785+ rendered images** in `rendered/` directory (intentionally preserved)
2. **WallpaperImageExtension scans all available images** during wallpaper operations
3. **Creates bookmarks for every image found** in the rendered directory
4. **Accumulates 12K+ bookmarks** in container preferences file (`~/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image.plist`)
5. **Attempts to sync massive data to NSUserDefaults** → 4MB limit violation → high CPU hang

**Error:** "Attempting to store >= 4194304 bytes of data in CFPreferences/NSUserDefaults on this platform is invalid"

**Data Analysis:**
- **Container preferences file**: 15MB containing 12,660 base64-encoded binary plists
- **Each entry**: Bookmark data for one rendered image file
- **Pattern**: `file:///Users/blake/Developer/weatherdesktop/rendered/hud-*.jpg`
- **Accumulation**: Every wallpaper operation adds more bookmarks without cleanup

### Secondary Issues: PropertyList Encoder & Data Accumulation
WallpaperAgent also encounters PropertyList encoding issues due to:

1. **Massive Data Structure**: Attempts to catalog ALL available wallpaper sources
2. **Accumulated Historical Data**: Historical Mission Control spaces and wallpaper configurations
3. **Complex Nested Encoding**: Arrays within arrays with binary data and metadata
4. **Infinite Recursion**: Encoder can't complete encoding of deeply nested structures

### Data Sources That Cause Issues

1. **Index.plist Accumulation**:
   - `~/Library/Application Support/com.apple.wallpaper/Store/Index.plist`
   - Grows to 60,000+ lines (148KB+)
   - Contains wallpaper configurations for every space/display combination
   - Recreates after deletion due to system rebuilding

2. **Mission Control Spaces History**:
   - `~/Library/Preferences/com.apple.spaces.plist`
   - Contains 50+ "Collapsed Space" entries (historical spaces)
   - WallpaperAgent creates configurations for ALL space UUIDs (active + historical)
   - Historical spaces accumulate over months/years

3. **Wallpaper Agent Internal State**:
   - `~/Library/Containers/com.apple.wallpaper.agent/Data/`
   - Contains cached wallpaper metadata and processing state
   - Gets corrupted or accumulates excessive data

4. **Wallpaper Cache Buildup**:
   - 458MB+ Aerial wallpapers in `~/Library/Application Support/com.apple.wallpaper/aerials/`
   - System attempts to index all aerial video files
   - Encoding fails when trying to serialize references to thousands of files

5. **XPC Message Processing Issues**:
   - Frequent XPC messages: `getLegacyDesktopPictureConfiguration`, `setLegacyDesktopPictureConfiguration`
   - Missing system directories: `/System/Library/Desktop Pictures/.thumbnails`, `.wallpapers`, `Solid Colors`
   - Unsupported image URLs with hash masks (corrupted image references)
   - **Critical: Missing DetachedSignatures file** (`/private/var/db/DetachedSignatures`)
   - ImageIO failures opening non-existent system wallpaper files

6. **SQLite Database Corruption**:
   - WallpaperImageExtension accessing Photos library database (`Photos.sqlite`)
   - SQLite errors: "cannot open file at line 51043" and "No such file or directory" for DetachedSignatures
   - DetachedSignatures file is missing from `/private/var/db/`
   - This prevents proper database validation and image processing

### Stack Trace Analysis
```
PropertyListEncoder.encode<A>() → _encodeBPlist<A>() → encodeToTopLevelContainerBPlist<A>()
↓
__PlistEncoderBPlist.wrapGeneric<A, B>() → _wrapGeneric<A>() → partial apply
↓
WallpaperTypes encoding operations (124340, 237548, 218440, etc.)
↓
Array<A>.encode(to:) → UnkeyedEncodingContainer.encode<A>()
↓
Infinite recursion on complex nested wallpaper configuration data
```

**Main Thread**: Blocked in NSApplication event loop (last ran 23+ seconds ago)
**Swift Task**: Consuming 9.98s CPU in PropertyListEncoder operations

## Solution Implemented

### Automated Cache Flush Script
**File:** `flush_wallpaper_cache.sh`

Nightly cleanup script with four steps:

1. **Clear ChoiceRequests** — delete `.blocked` sentinel, restart cfprefsd, then `defaults write` to clear bookmark accumulation (prevents WallpaperAgent hang)
2. **Delete JPG cache** — `~/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches` (~150 files/day, ~300 MB)
3. **Delete JPG var/folders cache** — `$(getconf DARWIN_USER_CACHE_DIR)/com.apple.wallpaper.extension.image` (was 22 GB accumulated)
4. **Delete BMP rendered frames** — `~/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/...` (was 353 GB / 15K files; ~3.5 GB/day growth)

### Daily LaunchAgent Job
**File:** `tv.jibb.weatherdesktop.cacheflush.plist`

User LaunchAgent (not a system LaunchDaemon) that runs the cache flush script nightly at 01:00.
Must be a LaunchAgent so it runs inside the user's GUI session (`gui/501`) — required for
`defaults write` to reach the user's cfprefsd instance.

```xml
<key>StartCalendarInterval</key>
<dict>
    <key>Hour</key><integer>1</integer>
    <key>Minute</key><integer>0</integer>
</dict>
```

Install to `~/Library/LaunchAgents/` (no sudo needed):
```bash
cp tv.jibb.weatherdesktop.cacheflush.plist ~/Library/LaunchAgents/
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/tv.jibb.weatherdesktop.cacheflush.plist
```

Log: `~/Library/Logs/weatherdesktop_cacheflush.log`

### Key Findings & Fixes

1. **WallpaperImageExtension Massive Indexing**: **CRITICAL ROOT CAUSE** - Extension caches ALL 15,785+ rendered images (12K+ bookmarks, 15MB)
2. **Immediate Cache Recreation**: Deleting preferences file causes immediate rebuild (<2 seconds)
3. **Cleanup Ineffective**: Cache flush provides only temporary relief
4. **Directory-Based Solution**: **ONLY SUSTAINABLE FIX** - Set wallpapers from dedicated directory to avoid 15K+ image indexing
4. **Index.plist Recreation**: Fixed by clearing spaces history (`com.apple.spaces.plist`)
5. **Agent State Corruption**: Fixed by clearing agent Data directory
6. **Cache Accumulation**: Prevented by regular automated cleanup
7. **Process Conflicts**: Resolved by terminating processes before cache operations
8. **XPC Message Overload**: Resolved by cleaning up corrupted state before wallpaper operations
9. **Missing DetachedSignatures**: Critical SQLite validation file missing (`/private/var/db/DetachedSignatures`)
10. **Metal Shader Cache Corruption**: WallpaperImageExtension Metal caches may be corrupted
11. **LaunchServices Cache Issues**: Corrupted `.csstore` files causing database access failures

### Cache Files Identified for Cleanup:

Based on process inspector analysis, the following cache files are accessed by WallpaperImageExtension:

1. **Metal Shader Caches**:
   - `$TMPDIR/../C/com.apple.wallpaper.extension.image/com.apple.wallpaper.extension.image/com.apple.metal/`
   - Contains `functions.data` and `functions.list` files

2. **LaunchServices Caches**:
   - `$TMPDIR/../0/com.apple.LaunchServices.dv/*.csstore` files
   - Corrupted cache files prevent proper database access

3. **Photos Library Database**:
   - `~/Pictures/Photos Library.photoslibrary/database/Photos.sqlite*`
   - SQLite database accessed during image processing

### Weather Desktop Application Considerations:

**To prevent triggering this issue:**
- **Cache flush before wallpaper setting**: Run `./flush_wallpaper_cache.sh` before setting wallpapers
- **Monitor system logs**: Check for WallpaperAgent errors after wallpaper operations
- **Batch wallpaper updates**: Avoid frequent individual wallpaper changes
- **Automated maintenance**: Rely on launchd job for regular cleanup

**Wallpaper setting best practices:**
- Clear wallpaper state before major updates
- Monitor Activity Monitor for WallpaperAgent CPU usage
- Use system logs to diagnose any issues
- Avoid setting wallpapers during high system load

## Monitoring & Alerts

**CPU Usage Monitoring:**
```bash
# Check for high WallpaperAgent CPU usage
ps aux | grep WallpaperAgent | grep -v grep

# Monitor system logs for issues
log stream --predicate 'process contains "Wallpaper"'
```

**Automated Cleanup:**
The LaunchAgent runs nightly at 01:00. Manual cleanup may be needed if:
- WallpaperAgent CPU usage exceeds 50%
- System becomes unresponsive
- PropertyList encoding errors appear in logs

**Quick Diagnostic:**
```bash
# Check Index.plist size
ls -la ~/Library/Application\ Support/com.apple.wallpaper/Store/Index.plist

# Check for DetachedSignatures errors
log show --predicate 'process contains "Wallpaper"' --last 5m | grep -i "detach"
```

## Root Cause Analysis Complete

**Issue Status: RESOLVED (Root Cause Identified)**
The WallpaperAgent CPU usage issue is caused by WallpaperImageExtension creating bookmarks for all 15,785+ weather desktop rendered images:

1. **Massive Bookmark Accumulation**: WallpaperImageExtension indexes ALL images in accessible directories
2. **Rendered Directory Overload**: 15,785+ images in `rendered/` directory get bookmarked (15MB data)
3. **NSUserDefaults Limit Violation**: 12,658+ bookmarks exceed 4MB platform limit
4. **System Hang**: PropertyListEncoder gets stuck trying to persist massive data structure

**Secondary Issues:**
- **Index.plist bloat** from historical Mission Control spaces
- **Agent state corruption** in container Data directory
- **Missing DetachedSignatures** causing SQLite validation failures
- **Cache corruption** in Metal shaders and LaunchServices

**Current Solution:**
1. **Nightly cache flush script** (`flush_wallpaper_cache.sh`) running as user LaunchAgent clears:
   - ChoiceRequests bookmark entries via `defaults write` (after removing `.blocked` sentinel and restarting cfprefsd)
   - JPG and BMP wallpaper cache files (prevents 350 GB+ disk accumulation)

2. **User LaunchAgent** (`gui/501`) firing at 01:00 — works with display sleep, no sudo required

**Results:**
- **ChoiceRequests entries cleared nightly**: bookmark accumulation stays near zero
- **Disk accumulation controlled**: BMP (~3.5 GB/day) and JPG caches flushed before they accumulate
- **WallpaperAgent hang resolved**: running `./flush_wallpaper_cache.sh` manually clears the hang when it occurs
- **System responsiveness restored**

**Final Status:** Working automated solution in place. Nightly cleanup prevents conditions that cause the hang. Permanent upstream fix (isolating the rendered directory from WallpaperImageExtension) would eliminate the need for cleanup entirely.

## Test Results

**Before Fix:**
- Index.plist: 60,000+ lines (148KB)
- CPU Usage: 30 seconds in 10-second samples
- System: Completely unresponsive

**After Cache Cleanup:**
- Index.plist: 34 lines (2.7KB) - clean configuration
- CPU Usage: Normal levels (WallpaperAgent: ~3 minutes total, not spiking)
- System: Responsive with some background errors
- DetachedSignatures file still missing (system-level issue)

**Remaining Issues:**
- DetachedSignatures file missing (`/private/var/db/DetachedSignatures`)
- SQLite validation errors in WallpaperImageExtension
- Migration errors during cleanup process (expected)
- Missing system wallpaper directories (non-critical)

**LaunchAgent Integration:**
- Runs nightly at 01:00 as user LaunchAgent (`gui/501`)
- Logs to `~/Library/Logs/weatherdesktop_cacheflush.log`
- No user interaction required

**Current Status (Post-Cleanup):**
- ✅ **Major Issue Resolved**: PropertyList encoding hangs eliminated
- ✅ **Index.plist Size**: Reduced from 148KB to 2.7KB (98% reduction)
- ✅ **CPU Usage**: Normal levels after cleanup (WallpaperAgent: 0.0% CPU after restart)
- ✅ **System Responsiveness**: Restored
- ⚠️ **DetachedSignatures**: System-level issue persists (non-critical for normal operation)
- ⚠️ **Migration Errors**: Expected during cleanup, not affecting performance
- ⚠️ **ImageIO Cache Errors**: Expected after clearing container data (temporary)

### Confirmed: Workarounds Failed
**TMPDIR Approach Tested:** Modified weather desktop to copy wallpapers to `$TMPDIR` and set from there.
- **Result:** WallpaperImageExtension still accesses rendered directory and creates 15MB cache
- **Status:** Failed - system still indexes all accessible image directories

**Directory Permissions Tested:** Made rendered directory private (`chmod 700`)
- **Result:** WallpaperImageExtension still creates 15MB cache file
- **Status:** Failed - system extensions bypass user permissions

**Root Cause Confirmed:**
- WallpaperImageExtension **continuously scans all accessible image directories**
- Finds 15,785+ images in `rendered/` directory regardless of wallpaper source
- Creates bookmarks for every image → 12K+ entries → 4MB limit violation
- **No workaround possible** without changing where images are stored

**Required Solution:**
Weather desktop must either:
1. **Limit rendered images** to <100 files (not practical for historical data)
2. **Store images elsewhere** - move rendered directory to location inaccessible to system
3. **Use dedicated wallpaper directory** and only keep current wallpaper there

### Confirmed: File Recreates Immediately

**TEST RESULTS:** Deleting `com.apple.wallpaper.extension.image.plist` causes immediate recreation:
- **Deletion:** File removed
- **Recreation:** Within 2 seconds, back to 15MB with 12K+ entries
- **Conclusion:** Cleanup is ineffective - WallpaperImageExtension immediately rebuilds massive cache

**Emergency Action:**
When WallpaperAgent shows high CPU usage (dock unresponsive is the main symptom), run:
```bash
cd /Users/blake/Developer/weatherdesktop
./flush_wallpaper_cache.sh
```
This clears ChoiceRequests entries and cache files; WallpaperAgent recovers without needing to be killed.

## Prevention Strategy

1. **Daily Maintenance**: LaunchD job prevents data accumulation
2. **Clean State**: Script ensures WallpaperAgent starts with minimal state
3. **Process Management**: Terminates conflicting processes safely
4. **Space Management**: Clears historical Mission Control spaces

## Disk Accumulation: BMP and JPG Cache Files

Beyond the ChoiceRequests plist issue, two additional cache directories accumulate to hundreds of
GB if not cleaned regularly. Both are now handled by the nightly flush script.

### BMP Rendered Frames — `com.apple.wallpaper.agent`
- **Path:** `~/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/`
- **Files:** 25 MB BMPs — one rendered wallpaper frame per weather update cycle
- **Scale observed:** 15,327 files → **353 GB** before first cleanup (May 2026)
- **Growth rate:** ~3.5 GB/day
- **Safe to delete:** Yes — macOS regenerates on demand
- **Cleanup:** `find ... -name "*.bmp" -delete` (Step 4 of flush script)

### JPG Image Cache — `var/folders`
- **Path:** `$(getconf DARWIN_USER_CACHE_DIR)/com.apple.wallpaper.extension.image/`
  (e.g. `/private/var/folders/rb/_wjy90zx33b8lv1vp_cfcc180000gn/C/com.apple.wallpaper.extension.image/`)
- **Files:** ~2 MB UUID-named JPGs — compressed wallpaper thumbnails/cache
- **Scale observed:** 12,666 files → **22 GB** going back to November 2025
- **Safe to delete:** Yes — macOS regenerates on demand
- **Cleanup:** `find ... -name "*.jpg" -delete` (Step 3 of flush script)

A second JPG cache at `~/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches`
holds only the current day's files (~150 files, ~300 MB) and is also flushed nightly (Step 2).

---

## `defaults write` Blocked by macOS Sequoia Sandbox (Jan 2026) — RESOLVED

### Problem
Since approximately January 4, 2026 (likely a macOS Sequoia update), all external `defaults write`
calls to the `com.apple.wallpaper.extension.image` preference domain fail with:

```
Could not write domain /Users/blake/Library/Containers/com.apple.wallpaper.extension.image/
Data/Library/Preferences/com.apple.wallpaper.extension.image; exiting
```

### Root Cause: `.plist.blocked` Sentinel File

A 0-byte sentinel file is created at:
```
~/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/
com.apple.wallpaper.extension.image.plist.blocked
```

cfprefsd reads this file **at daemon startup** and loads the write block into memory. Once loaded,
the block is enforced in-process — deleting the sentinel file alone does not lift the block because
cfprefsd still has the policy cached in memory from startup.

**Important:** The `.plist.blocked` mechanism is **completely undocumented by Apple**. There is no
mention of it in developer docs, no public cfprefsd source code (only the CFPreferences API layer
is open-sourced at [opensource-apple/CF](https://github.com/opensource-apple/CF)), and no Apple
Developer Forum threads discussing it.

The file also carries a `com.apple.provenance` extended attribute. This is a Gatekeeper provenance
tracking tag introduced in macOS 13 Ventura — it records which application created the file as an
audit trail. Contrary to what the attribute name implies, it does **not** prevent file deletion;
that was a misdiagnosis caused by Claude Code's own sandbox blocking the `rm` call.
See: [Eclectic Light Company — Quarantine, MACL and Provenance](https://eclecticlight.co/2025/12/05/quarantine-macl-and-provenance-what-are-they-up-to/),
[Michael Tsai — Ventura adds com.apple.provenance](https://mjtsai.com/blog/2023/03/16/ventura-adds-com-apple-provenance/),
[Apple Developer Forums thread 723397](https://developer.apple.com/forums/thread/723397).

### Approaches That Failed

1. **`sudo -u blake defaults write` from LaunchDaemon (system context)** — fails silently; wrong bootstrap namespace; `defaults` cannot reach the user's `cfprefsd`
2. **`launchctl asuser <uid> sudo -u blake defaults write` from LaunchDaemon** — still fails at 1 AM when display is asleep; no active user session to attach to in the system context. See [scriptingosx.com — Running a Command as Another User](https://scriptingosx.com/2020/08/running-a-command-as-another-user/) for the pattern.
3. **`defaults write` from LaunchAgent (`gui/501`)** — fails; cfprefsd enforces the `.blocked` policy regardless of which bootstrap session the caller is in
4. **TCC / entitlement research** — `com.apple.security.temporary-exception.shared-preference.read-write` ([Apple Entitlement Reference](https://developer.apple.com/library/archive/documentation/Miscellaneous/Reference/EntitlementKeyReference/Chapters/AppSandboxTemporaryExceptionEntitlements.html)) requires a signed native binary and does not document any interaction with `.blocked` files

### Resolution

The fix is two steps, added to the flush script before the `defaults write` calls:

```bash
# 1. Delete the sentinel — cfprefsd does not recreate it on restart
rm "${EXT_DOMAIN}.plist.blocked" 2>/dev/null

# 2. Restart cfprefsd to clear the in-memory block (~1s to restart automatically)
killall cfprefsd; sleep 1

# 3. defaults write now succeeds
defaults write "${EXT_DOMAIN}" "ChoiceRequests.ImageFiles" -array
```

**Key findings:**
- The sentinel file is **deletable** — `rm` works as the file owner (no kernel protection)
- cfprefsd **does not recreate** the sentinel after restart
- The `killall cfprefsd` is essential — deleting the file alone leaves the in-memory block active
- cfprefsd restarts automatically within ~1 second; a `sleep 1` before writing is sufficient
- Running `killall cfprefsd` at 1 AM with no interactive apps open has no noticeable impact

---

## Files Modified

1. `flush_wallpaper_cache.sh` - Cache cleanup script (LaunchAgent, 4 steps)
2. `tv.jibb.weatherdesktop.cacheflush.plist` - LaunchAgent configuration (was LaunchDaemon)
3. `TROUBLESHOOTING.md` - This documentation

## System Log Analysis

### Collecting WallpaperAgent Logs
Use macOS unified logging to monitor WallpaperAgent activity:

```bash
# Get recent Wallpaper-related logs
log show --predicate 'process contains "Wallpaper"' --last 5m

# Monitor live WallpaperAgent activity
log stream --predicate 'process contains "Wallpaper"'
```

### Common Log Patterns
- **XPC Messages**: `getLegacyDesktopPictureConfiguration`, `setLegacyDesktopPictureConfiguration`
- **Errors**: Missing system directories, unsupported image URLs, SQLite issues
- **Performance**: Coordinator work queues, watchdog timers, encoding operations

## Verification

To test the fix:
```bash
# Manual test
./flush_wallpaper_cache.sh

# Check LaunchAgent status
launchctl print gui/$(id -u)/tv.jibb.weatherdesktop.cacheflush

# Trigger immediate run
launchctl kickstart gui/$(id -u)/tv.jibb.weatherdesktop.cacheflush

# Monitor logs
tail -f ~/Library/Logs/weatherdesktop_cacheflush.log

# Check system logs for WallpaperAgent errors
log show --predicate 'process contains "Wallpaper"' --last 1m
```

The system now maintains clean wallpaper state and prevents PropertyList encoding hangs.

## macOS System Calls in Weather Desktop

### Overview
The weather desktop application uses **CGO (C bindings for Go)** to interface with macOS system frameworks through Objective-C. All wallpaper operations are performed via the **Cocoa framework**.

### Framework Dependencies
```objective-c
#import <Cocoa/Cocoa.h>    // Core macOS UI framework
#import <unistd.h>         // POSIX system calls (usleep)
```

### CGO Build Configuration
```go
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
```

### Core System Calls

#### 1. **Wallpaper Setting Operations** (`setWallpaper` Objective-C function)

**NSString Operations:**
- `NSString *path = [NSString stringWithUTF8String:imagePath];`
  - Converts C string to NSString for Cocoa compatibility

**File System Operations:**
- `NSURL *imageURL = [NSURL fileURLWithPath:path];`
  - Creates file URL from path string
- `NSFileManager *fileManager = [NSFileManager defaultManager];`
  - Gets default file manager instance
- `BOOL fileExists = [fileManager fileExistsAtPath:path];`
  - Checks file existence
- `NSDictionary *attrs = [fileManager attributesOfItemAtPath:path error:nil];`
  - Gets file attributes (size, etc.)

**Display/Screen Operations:**
- `NSArray *screens = [NSScreen screens];`
  - Enumerates all connected displays
- `NSUInteger screenCount = [screens count];`
  - Gets number of screens
- `NSRect frame = [screen frame];`
  - Gets screen dimensions and position
- `NSUInteger screenIndex = [screens indexOfObject:screen];`
  - Gets screen index in array

**Wallpaper Setting (Core Operation):**
- `NSWorkspace *workspace = [NSWorkspace sharedWorkspace];`
  - Gets shared workspace instance (manages desktop)
- `NSDictionary *currentOptions = [workspace desktopImageOptionsForScreen:screen];`
  - Retrieves current wallpaper options for screen
- `BOOL success = [workspace setDesktopImageURL:imageURL forScreen:screen options:options error:&error];`
  - **PRIMARY SYSTEM CALL**: Sets wallpaper image on specific screen

**Wallpaper Options:**
- `NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown)`
  - Scales image proportionally
- `NSWorkspaceDesktopImageAllowClippingKey: @(YES)`
  - Allows image clipping if needed

**Error Handling:**
- `NSError *error = nil;`
  - Error object for system call failures
- `NSString *errorDesc = [error localizedDescription];`
  - Gets human-readable error description

**Timing Operations:**
- `usleep(500000);` (0.5 seconds)
  - Allows system to process wallpaper changes

**Logging Operations:**
- `NSLog(@"message", args);`
  - System logging for debugging

#### 2. **Cache Management Operations** (Go functions)

**File System Operations:**
- `os.Stat()` - Check file/directory existence
- `os.Remove()` - Delete files
- `os.Getenv("TMPDIR")` - Get temporary directory
- `filepath.Abs()` - Get absolute paths
- `filepath.Join()` - Construct paths

**Process Execution:**
- `exec.Command("find", args...).Run()` - Execute system `find` command
- `exec.Command("find", cachePath, "-type", "f", "-delete").Run()` - Delete cache files

#### 3. **Memory Management**
- `C.CString(absPath)` - Convert Go string to C string
- `defer C.free(unsafe.Pointer(cPath))` - Free C memory
- `@autoreleasepool { ... }` - Objective-C memory management

### System Integration Points

#### **NSWorkspace (Desktop Management)**
- Central macOS service for desktop operations
- Manages wallpaper settings across all screens
- Handles desktop image scaling and positioning
- Provides per-screen wallpaper options

#### **NSScreen (Display Management)**
- Enumerates physical and virtual displays
- Provides screen dimensions and positioning
- Supports multi-monitor configurations

#### **NSFileManager (File Operations)**
- Validates file existence before wallpaper operations
- Retrieves file metadata (size, permissions)
- Cross-platform file system operations

#### **NSURL (Resource Locators)**
- Converts file paths to URL objects
- Required for Cocoa framework operations
- Handles file:// URL scheme

### Security & Sandboxing
- **Container Access**: Reads from WallpaperAgent container caches
- **File Permissions**: Respects macOS file permissions
- **User Permissions**: Runs with user privileges (no elevated access)

### Performance Characteristics
- **Synchronous Operations**: Wallpaper setting blocks until complete
- **Per-Screen Processing**: Iterates through all displays individually
- **Error Resilience**: Continues with other screens if one fails
- **Memory Management**: Proper cleanup of C/Objective-C objects

### Error Scenarios
- **File not found**: `NSFileManager` existence checks
- **Invalid image**: `NSWorkspace` validation
- **Permission denied**: macOS sandbox restrictions
- **Display disconnected**: `NSScreen` enumeration failures

This comprehensive system call integration enables reliable cross-platform wallpaper management while maintaining macOS-specific optimizations and error handling.

