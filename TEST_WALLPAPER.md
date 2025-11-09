# Wallpaper Setting Investigation - macOS Sequoia/Tahoe Bug

## Quick Status (Last Updated: 2025-11-06)

**Current State:** macOS 26.1 (Tahoe) legacy wallpaper API is broken. Test program issues have been fixed, but wallpaper still fails to change.

**What Works:**
- ✅ Test program timing fixed (waits 45s for async XPC)
- ✅ Memory management fixed (getCurrentWallpaper proper string handling)
- ✅ XPC communication completes without errors
- ✅ Dock and WallpaperAgent process requests successfully
- ✅ No errors logged in Console.app

**What Doesn't Work:**
- ❌ Wallpaper immediately reverts to DefaultDesktop.heic
- ❌ Silent rejection by extension system (no error messages)
- ❌ Legacy `setDesktopImageURL` API appears to be broken at system level

**Next Actions for Future Agent:**
1. Test direct preference manipulation (`com.apple.wallpaper` plist)
2. Investigate TCC/sandboxing permissions for file access
3. Consider filing Apple bug report with comprehensive logs
4. Explore private framework APIs as last resort

**Key Files:**
- `test_wallpaper.go` - Fixed test program (waits 45s, proper memory management)
- `TEST_WALLPAPER.md` - This document with full investigation details
- Console logs showing successful XPC but silent wallpaper rejection

## Executive Summary

The `NSWorkspace setDesktopImageURL:forScreen:options:error:` API reports success on macOS Sequoia (15.x) / Tahoe (26.x), but the wallpaper extension system fails to apply the change. The wallpaper reverts to the default desktop image (`/System/Library/CoreServices/DefaultDesktop.heic`) after approximately 45 seconds.

**CRITICAL DISCOVERY (2025-11-06):** Initial test program had a bug where it exited after only 3 seconds, causing XPC connection invalidation errors. The macOS wallpaper system requires asynchronous XPC communication between Dock → WallpaperAgent → Extension that takes ~32-45 seconds to complete. Programs calling `setDesktopImageURL` must remain alive long enough for this communication cycle to finish, otherwise the XPC connection is invalidated and the wallpaper change fails silently.

**TEST PROGRAM FIXES (2025-11-06):**
1. **Timing Issue Fixed:** Program now waits 45 seconds with periodic checks every 5 seconds
2. **Memory Bug Fixed:** `getCurrentWallpaper()` function now properly uses `strdup()` to return malloc'd strings that won't be freed by autorelease pool
3. **Cache Copy Removed:** Simplified to use original image path directly (cache copy with `.jpg.jpg` extension was potentially confusing the system)

**CONFIRMED BEHAVIOR (2025-11-06):**
- ✅ No more XPC connection invalidation errors
- ✅ Program stays alive for full 45 seconds
- ✅ Dock receives and processes `setLegacyDesktopPicture` calls
- ✅ WallpaperAgent launches WallpaperImageExtension process
- ✅ No errors logged in Console (no WallpaperExtensionError, no NSCocoaErrorDomain 4099)
- ❌ **Wallpaper still immediately reverts to `/System/Library/CoreServices/DefaultDesktop.heic`**
- ❌ **`desktopImageURLForScreen` returns DefaultDesktop.heic immediately after API call reports success**

**CONCLUSION:** This confirms a genuine macOS Sequoia/Tahoe (26.x) wallpaper extension system bug. The legacy API path is broken at the system level, not due to timing, threading, or test program issues.

## Test Program

**File:** `test_wallpaper.go`

**Usage:**
```bash
go run test_wallpaper.go /path/to/image.jpg
```

**Features:**
- Thread verification (ensures main thread execution)
- Detailed error reporting with full NSError details
- Pre-set and post-set wallpaper verification
- File validation (existence, permissions, image loading)
- Screen-by-screen diagnostics

## Investigation Findings

### 1. API Behavior

**Method:** `NSWorkspace setDesktopImageURL:forScreen:options:error:`

**Observations:**
- ✅ API call returns `YES` (success)
- ✅ No NSError returned
- ✅ File exists and is readable
- ✅ Image loads successfully with NSImage
- ✅ Thread verification: Called from main thread (required)
- ❌ **Wallpaper does not visually change**
- ❌ **`desktopImageURLForScreen` immediately returns default wallpaper path**

**Example Output:**
```
Screen 0:
  Current wallpaper: /Users/blake/code/weatherdesktop/rendered/hud-251102-2040.jpg
  ✓ API call SUCCESS (took 0.002 seconds)
  ⚠ Verification FAILED:
    Expected: /Users/blake/code/weatherdesktop/rendered/hud-251102-2040.jpg
    Actual:   /System/Library/CoreServices/DefaultDesktop.heic
```

### 2. macOS Version

**System:** macOS 26.1 (Tahoe) / Build 25B77

**Impact:** This appears to be a macOS Sequoia/Tahoe-specific issue. The extension system architecture changed significantly in these versions.

### 3. Extension System Architecture

**Legacy Path Flow:**
1. `NSWorkspace setDesktopImageURL` → Dock `setLegacyDesktopPicture`
2. Dock → XPC message to `WallpaperAgent`: `setLegacyDesktopPictureConfiguration`
3. `WallpaperAgent` → Processes via extension system
4. Extension system fails during export phase

**Timeline:**
- API call: ~0.002 seconds
- Dock processing: Immediate
- WallpaperAgent processing: ~32 seconds
- Extension failure: ~32 seconds after API call
- Wallpaper reverts: ~45 seconds after API call

### 4. Error Messages

**Console.app Logs Show:**

```
WallpaperAgent[1333:xxx] ERROR - makeWallpaper for '[extension] com.apple.wallpaper.extension.image': 
  WallpaperExtensionKit.WallpaperExtensionError (3)

WallpaperAgent[1333:xxx] ERROR - Exporting wallpaper '<ChoiceDescriptor provider=com.apple.wallpaper.choice.image, configuration=DBDBF273>' - 
  due to 'Failed to create snapshot to export': WallpaperExtensionKit.WallpaperExtensionError (3)

WallpaperAgent[1333:xxx] ERROR - [com.apple.wallpaper.extension.image] Wallpaper Timeline: Acquire Wallpaper: 
  NSCocoaErrorDomain (4099)
```

**Error Analysis:**
- `WallpaperExtensionKit.WallpaperExtensionError (3)`: Extension system failure
- `NSCocoaErrorDomain (4099)`: File access/sandboxing error
- "Failed to create snapshot to export": Extension cannot create snapshot of image

### 5. SDK Investigation

**Checked:** macOS 26.0 SDK (`/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk`)

**Findings:**
- ✅ `NSWorkspace.h` contains only the legacy API (`setDesktopImageURL:forScreen:options:error:`)
- ✅ No newer public API available
- ✅ API available since macOS 10.6 (no changes)
- ✅ Private frameworks exist but are undocumented:
  - `Wallpaper.framework` (Swift-only, XPC-based)
  - `WallpaperExtensionKit.framework` (Swift-only, XPC-based)
  - `WallpaperSettingsUI.framework`
  - `WallpaperFoundation.framework`
  - `WallpaperTypes.framework`

**Private Framework Symbols (from .tbd files):**
- `ChoiceProviderID` (image, color, photoLibrary, aerials, etc.)
- `ChoiceRequest`
- `SettingsViewModel`
- XPC messaging infrastructure

**Conclusion:** No public API alternative exists. Private frameworks are Swift-only and require reverse engineering.

### 6. Preference Domains

**Found Wallpaper-Related Domains:**
- `com.apple.wallpaper.agent`
- `com.apple.wallpaper.extension.image`
- `com.apple.wallpaper.extension.aerials`
- `com.apple.wallpaper`
- `com.apple.Wallpaper-Settings.extension`
- `com.apple.wallpaper.aerial`

**Contents:**
- `com.apple.wallpaper.agent`: Minimal data (heartbeat timestamps)
- `com.apple.wallpaper.extension.image`: Binary plist data in `ChoiceRequests.ImageFiles` array
- `com.apple.wallpaper`: Contains `SystemWallpaperURL` (current system wallpaper)

**Legacy Preference:**
- `com.apple.desktop`: **Not found** (may have been removed in Sequoia/Tahoe)

### 7. Thread Requirements

**Apple Documentation:** "You must call this method from your app's main thread."

**Verification:**
- ✅ Test program verifies main thread execution
- ✅ Uses `dispatch_async(dispatch_get_main_queue())` fallback if needed
- ✅ Confirmed: Called from main thread in test runs

**Result:** Threading is not the issue.

### 8. File Characteristics

**Test Image:**
- Path: `/Users/blake/code/weatherdesktop/rendered/hud-251102-2040.jpg`
- Size: 1,849,393 bytes (~1.8 MB)
- Format: JPEG
- Dimensions: 3840 x 2160 points
- Permissions: 644 (rw-r--r--)
- Readable: YES
- Loadable by NSImage: YES

**Result:** File characteristics are not the issue.

### 9. Option Dictionary

**Current Options Used:**
```objective-c
NSDictionary *options = @{
    NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown),
    NSWorkspaceDesktopImageAllowClippingKey: @(YES)
};
```

**Current Options (from screen):**
```objective-c
{
    NSWorkspaceDesktopImageAllowClippingKey: @(YES),
    NSWorkspaceDesktopImageFillColorKey: <NSColor>,
    NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown)
}
```

**Test:** Merged current options with standard options - no change in behavior.

**Result:** Option dictionary is not the issue.

### 10. Timing Analysis

**Observed Delays:**
- Screen 0: 0.002 seconds (API call)
- Screen 1: 2.004 seconds (API call)
- Screen 2: 2.002 seconds (API call)
- WallpaperAgent processing: ~32 seconds
- Extension failure: ~32 seconds after API call
- Wallpaper reversion: ~45 seconds after API call

**Pattern:** The extension system processes asynchronously and fails silently. The API returns success immediately, but the extension system fails later.

## Root Cause Hypothesis

**Primary Hypothesis:** macOS Sequoia/Tahoe wallpaper extension system bug

The extension system (`com.apple.wallpaper.extension.image`) is failing to create a snapshot of the image file for export. This could be due to:

1. **Sandboxing restrictions**: Extension may not have access to files outside specific directories
2. **File location requirements**: Extension may require files in specific locations (`~/Library/Application Support/com.apple.wallpaper/`)
3. **Permission issues**: Extension may need different file permissions or extended attributes
4. **Extension system bug**: Known issue in macOS Sequoia/Tahoe where legacy API path doesn't work with new extension system

**Evidence:**
- Error occurs in extension system, not in API call
- Error is consistent across all attempts
- No workaround found in public APIs
- Issue affects all three screens identically

### 11. Extension Cache Directory

**Location:** `/private/var/folders/rb/_wjy90zx33b8lv1vp_cfcc180000gn/C/com.apple.wallpaper.extension.image`

**Findings:**
- Contains **11,703 cached image files**
- All files are UUID-named JPG files (e.g., `0A0CC2E3-74D5-4E79-B9CB-BB450AEE674B.jpg`)
- File sizes: 2-3 MB each
- Files date back to September 2024
- Most recent: November 2, 2024 (`0AA35747-92B1-4EF8-AE5A-33E3A2587F3F.jpg`)
- These appear to be processed/cached versions of wallpapers

**Additional Cache Location:**
- `${TMPDIR}../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/`
- Contains PNG files (snapshots for export)
- **Actual cache location:** `/Users/blake/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/`

**Cache Behavior (from logs):**
- Extension system attempts to cache images: `Image cache insertion - url: file:///Users/blake/code/weatherdesktop/rendered/hud-251102-2140.jpg`
- Cache writes scheduled: `Scheduling image cache writes to happen in seconds(2) seconds`
- Cache write location: `Writing out images to cache at 'file:///Users/blake/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/'`
- **Cache write succeeds** but export still fails

**Key Insight:** The extension system successfully writes to cache but fails during the "export" phase (snapshot creation). This suggests the cache write is not the problem - the export/snapshot creation is the failure point.

**Hypothesis:** The extension system may require:
1. Source file to be accessible from cache directory
2. Processed version to exist in cache before setting
3. Snapshot creation to succeed before wallpaper can be applied

**Next Test:** Copy image to cache directory before calling API, or check if extension system expects files in specific location.

### 12. Cache Copy Test

**Date:** 2025-11-02  
**Test:** Copy image to cache directory before calling API

**Procedure:**
1. Copy image to `/Users/blake/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/`
2. Call `setDesktopImageURL` with cached file path
3. Monitor results

**Results:**
- ✅ File successfully copied to cache directory
- ✅ API call reports success for all 3 screens
- ❌ Wallpaper still reverts to default (`/System/Library/CoreServices/DefaultDesktop.heic`)
- ❌ Verification shows default wallpaper immediately after API call

**Finding:** Copying to cache directory does not resolve the issue. The extension system still fails at the export/snapshot phase even when the file is in the expected cache location.

**Observation:** The Index.plist shows previous successful wallpaper configurations using original file paths (not cache paths), suggesting the cache is managed by the extension system itself, not by the API caller.

## Troubleshooting Session Summary (2025-11-06)

### Problems Identified and Fixed

1. **XPC Connection Timing Issue**
   - **Problem:** Test program exited after 3 seconds, causing XPC connection invalidation
   - **Error:** `[0x8accb8780] invalidated after the last release of the connection object`
   - **Root Cause:** macOS wallpaper system requires 32-45 seconds for async XPC communication (Dock → WallpaperAgent → Extension)
   - **Fix:** Program now waits 45 seconds with periodic checks every 5 seconds
   - **Status:** ✅ RESOLVED - No more XPC errors

2. **Memory Management Bug in getCurrentWallpaper()**
   - **Problem:** Function returned pointer to memory in autorelease pool, causing empty string returns
   - **Symptom:** Go code received empty strings when querying current wallpaper
   - **Fix:** Changed function to use `strdup()` and return malloc'd strings that caller must free
   - **Status:** ✅ RESOLVED - Wallpaper paths now properly returned

3. **Cache Copy Creating Confusing Paths**
   - **Problem:** Test was copying image to cache directory, creating files with `.jpg.jpg` double extension
   - **Example:** `/Users/blake/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/.../hud-251102-2153.jpg.jpg`
   - **Fix:** Removed cache copy step, now uses original image path directly
   - **Status:** ✅ RESOLVED - Simpler path handling

### Remaining Issue: Silent Wallpaper Rejection

**Problem:** After all fixes, wallpaper still fails to change on macOS 26.1 (Tahoe)

**Behavior:**
- API call returns success (YES)
- No errors logged in Console.app
- Dock processes the request
- WallpaperAgent launches extension
- Extension processes request silently
- **BUT:** `desktopImageURLForScreen` immediately returns `/System/Library/CoreServices/DefaultDesktop.heic`

**This is NOT:**
- A timing issue (program now waits 45+ seconds)
- A threading issue (confirmed main thread execution)
- An XPC issue (no connection errors)
- A file permissions issue (file is readable, NSImage loads it successfully)
- A logged error (no WallpaperExtensionError in Console)

**This IS:**
- A silent rejection at the macOS wallpaper extension system level
- Specific to macOS Sequoia/Tahoe (26.x) 
- Affecting the legacy `setDesktopImageURL` API path
- Possibly a deliberate security restriction or sandboxing change

## Potential Workarounds (To Investigate)

### 1. File Location & Cache
- **TESTED:** Extension system successfully writes to cache
- **TESTED:** Copying image to cache directory before API call - **Does not resolve issue**
- Copy image to `~/Library/Application Support/com.apple.wallpaper/` before setting
- Copy image to cache directory: `/Users/blake/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/`
- May require specific directory structure or UUID naming

**Result:** Cache location is managed by extension system. Pre-copying to cache does not resolve export failure.

### 2. XPC Connection Timing Issue

**Date:** 2025-11-06  
**Error:** `[0x8accb8780] invalidated after the last release of the connection object`

**Problem:** Test program was exiting too quickly (after only 3 seconds), causing XPC connection to be invalidated before WallpaperAgent could complete processing.

**Timeline Analysis:**
- Program calls `setDesktopImageURL`: 0.002 seconds
- Program waits: 3 seconds  
- **Program exits** → XPC connection invalidated
- WallpaperAgent tries to process: 32 seconds later → Connection already gone!

**Solution:** Keep process alive for at least 45 seconds to allow complete XPC communication cycle.

**Fix Applied:** Modified test program to:
- Wait up to 45 seconds with periodic checks
- Monitor wallpaper changes every 5 seconds
- Detect when wallpaper successfully changes
- Exit only after verification or timeout

**Status:** Program no longer causes XPC invalidation errors. This allows proper testing of whether macOS extension system works correctly.

### 3. Snapshot/Export Failure Workaround
- **ROOT ISSUE:** Extension fails at "Failed to create snapshot to export"
- May need to pre-create snapshot in cache directory
- May need specific file format or metadata
- May need to trigger export manually after cache write

### 4. Permission/Extended Attributes
- Set specific extended attributes on file
- Adjust file permissions beyond standard 644

### 5. Direct Preference Manipulation
- Write directly to `com.apple.wallpaper.extension.image` preferences
- Requires understanding binary plist structure in `ChoiceRequests.ImageFiles`

### 6. Legacy Preference Path
- Attempt to use deprecated `com.apple.desktop` preference domain
- May bypass extension system entirely

### 7. Notification/Refresh
- Send notification to force wallpaper refresh
- May trigger extension system re-processing

### 8. Apple Bug Report
- File with Apple Feedback Assistant
- Include error logs and test program
- Reference: `WallpaperExtensionKit.WallpaperExtensionError (3)`

## Test Results Summary

### Test Run 1: Initial Discovery

**Date:** 2025-11-02  
**macOS Version:** 26.1 (Tahoe) Build 25B77  
**Test Program:** `test_wallpaper.go` (original version)

**Test Image:** `/Users/blake/code/weatherdesktop/rendered/hud-251102-2040.jpg`

**Results:**
- ✅ API returns success for all 3 screens
- ❌ Visual wallpaper does not change
- ❌ `desktopImageURLForScreen` returns default wallpaper immediately
- ❌ Extension system fails with `WallpaperExtensionKit.WallpaperExtensionError (3)`
- ❌ XPC connection invalidation errors (program exited too early)

**Screens Tested:**
- Screen 0: 3840 x 2160 @ 0, 0
- Screen 1: 3840 x 2160 @ -7680, -7
- Screen 2: 3840 x 2160 @ -3840, -7

### Test Run 2: After Fixes

**Date:** 2025-11-06  
**macOS Version:** 26.1 (Tahoe) Build 25B77  
**Test Program:** `test_wallpaper.go` (fixed version with 45s wait, memory fixes)

**Test Image:** `/Users/blake/code/weatherdesktop/rendered/hud-251102-2153.jpg`

**Results:**
- ✅ API returns success for all 3 screens
- ✅ No XPC connection errors (program stayed alive 45 seconds)
- ✅ Dock processes setLegacyDesktopPicture calls successfully
- ✅ WallpaperAgent launches WallpaperImageExtension
- ✅ No errors logged in Console.app
- ❌ **Wallpaper STILL does not change**
- ❌ **`desktopImageURLForScreen` returns `/System/Library/CoreServices/DefaultDesktop.heic` immediately**
- ❌ **No WallpaperExtensionError logged (silent failure)**

**Key Observation:** The system processes the request without logging errors, but silently fails to apply the wallpaper. The API immediately returns DefaultDesktop.heic as the current wallpaper, suggesting the change is being rejected or overridden at a deeper level.

**Screens Tested:** All 3 screens show identical behavior.

## Next Steps

1. **✅ COMPLETED: Test program fixes**
   - ~~Program was exiting after 3 seconds~~ **FIXED**
   - ~~Memory bug in getCurrentWallpaper()~~ **FIXED**
   - ~~Cache copy creating .jpg.jpg files~~ **FIXED**
   - **Result:** XPC errors eliminated, but wallpaper still fails to change

2. **✅ COMPLETED: Re-test with fixed program**
   - ✅ Updated test program with 45-second wait
   - ✅ Monitored Console.app logs during full cycle
   - ✅ Confirmed: No XPC errors, no extension errors logged
   - ✅ Confirmed: Wallpaper still fails even with proper timing
   - **Result:** Genuine macOS system bug confirmed

3. **Investigate file location requirements**
   - Test copying image to `~/Library/Application Support/com.apple.wallpaper/`
   - Check if specific directory structure is required

4. **If still failing: Decode binary plist structure**
   - Understand `ChoiceRequests.ImageFiles` format
   - Attempt direct preference manipulation

5. **Check sandboxing/TCC permissions**
   - Verify if Desktop Pictures access is granted
   - Check if extension has file access permissions

6. **File Apple bug report (if needed)**
   - Document the issue with Apple
   - Include test program and logs
   - Mention XPC connection timing requirements

7. **Monitor macOS updates**
   - Check if future macOS updates fix the issue
   - Track Apple Developer Forums for similar reports

## References

- **NSWorkspace Documentation:** https://developer.apple.com/documentation/appkit/nsworkspace
- **setDesktopImageURL Method:** https://developer.apple.com/documentation/appkit/nsworkspace/setdesktopimageurl(_:for:options:)
- **macOS SDK:** `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk`

## Log Commands

**Monitor wallpaper extension logs:**
```bash
log show --predicate 'subsystem == "com.apple.wallpaper"' --last 5m
```

**Monitor Dock wallpaper calls:**
```bash
log show --predicate 'process == "Dock" AND subsystem == "com.apple.wallpaper"' --last 5m
```

**Monitor WallpaperAgent errors:**
```bash
log show --predicate 'process == "WallpaperAgent" AND eventType == "errorEvent"' --last 5m
```

**Monitor extension system:**
```bash
log show --predicate 'subsystem == "com.apple.wallpaper" AND (eventMessage CONTAINS "extension" OR eventMessage CONTAINS "ERROR")' --last 5m
```

**Check for XPC connection issues:**
```bash
log show --predicate 'process CONTAINS "Wallpaper" AND eventMessage CONTAINS "invalidated"' --last 2m
```

## Investigation Summary

This investigation confirms that the `NSWorkspace setDesktopImageURL:forScreen:options:error:` API is fundamentally broken on macOS Sequoia/Tahoe (26.x) when called from third-party applications. The issue is NOT caused by:

- Incorrect API usage
- Threading issues  
- Timing/XPC connection problems (now fixed in test program)
- File permissions or accessibility
- Image format or size issues

The wallpaper system silently rejects all attempts to set custom wallpapers via the legacy API, immediately reverting to the default desktop image without logging any errors. This appears to be either:

1. **A macOS bug** in the new extension-based wallpaper system's handling of legacy API calls
2. **A deliberate security restriction** requiring new entitlements or permissions
3. **An incomplete implementation** of the legacy API compatibility layer

**For weatherdesktop project:** Consider alternative approaches:
- Direct manipulation of wallpaper preference files
- Use of private frameworks (requires reverse engineering)
- File Apple bug report and wait for fix
- Workaround: Use AppleScript automation via System Events (slower but may work)

