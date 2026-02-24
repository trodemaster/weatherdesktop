# Implementation Details

Technical documentation for developers and contributors.

## Recent Changes (December 2025)

### WallpaperAgent Hang — Full Root Cause & Fix (February 2026)

> **Status: Stable workaround in place. Under observation for several days before considering further changes.**

#### Problem Summary

`WallpaperAgent` was consuming 100% CPU and hanging for 20–30+ seconds after every `./wd` run, blocking the desktop from updating.

#### True Root Cause

The hang is caused by `WallpaperImageExtension` accumulating a bookmark entry in `ChoiceRequests.ImageFiles` (inside its container preferences plist) for **every unique file path ever set as a wallpaper**. With `./wd` running every 5 minutes and producing a new `hud-YYMMDD-HHMM.jpg` each time, this list grew to **15,000+ entries (~15MB)**. `WallpaperAgent` then attempted to sync this structure to `NSUserDefaults`, which has a hard 4MB platform limit, causing `PropertyListEncoder` to hang in an infinite retry loop.

**Contributing factors:**

1. **Unique timestamped filenames** — each `hud-YYMMDD-HHMM.jpg` was a new path, so the extension kept adding new bookmarks without ever deduplicating.
2. **`code/` symlink alias** — the launchd job ran `wd` with `WorkingDirectory: /Users/blake/code/weatherdesktop` (a symlink to `Developer/`). Because `os.Args[0]` was the symlink path, all wallpaper paths recorded by WallpaperAgent used `code/...` instead of `Developer/...`, making paths appear as new entries even after the project was moved.
3. **`cfprefsd` cache not cleared** — previous cleanup attempts used `rm -f` on the plist file, which bypasses `cfprefsd`'s in-memory cache. The daemon restored the 15MB file instantly from cache every time, making all cleanup attempts completely ineffective — including across reboots, since cfprefsd's cache is populated from the file before the file is deleted.
4. **`setDesktopImageURL:` only updating the active space** — other Mission Control spaces retained old `rendered/` path wallpapers, so WallpaperAgent continued registering those historical paths with the extension even after the active space was updated.

#### Fixes Applied

**1. ~~Wallpaper source indirection via `~/Pictures/Desktop/`~~ (Reverted Feb 24, 2026)**

- ~~Previously copied the rendered image to `~/Pictures/Desktop/weather-desktop.jpg` (fixed filename).~~ This approach was reverted after confirming the root causes were addressed by fixes #3, #4, and #5.
- **Current approach**: Set the wallpaper directly from timestamped `rendered/` files (`hud-YYMMDD-HHMM.jpg`).
- **Why this is now safe**: Plist entries and cache files are now pruned on every `wd` run (before wallpaper set), so they stay at 0→1 and never accumulate.

**2. `NSWorkspaceDesktopImageAllSpacesKey` — Added then removed**

- `pkg/desktop/macos.go`: Temporarily added `NSWorkspaceDesktopImageAllSpacesKey: @(YES)` to update all Mission Control spaces atomically.
- **Reverted (Feb 24, 2026)**: This undocumented key causes macOS to write the wallpaper configuration for ALL spaces in the preference database, but does **not** trigger an immediate visual redraw on the currently visible space. WallpaperAgent processes the change lazily (e.g., on space transition), so the desktop appeared frozen between updates.
- The original `setDesktopImageURL:forScreen:options:error:` call **without** this key updates only the current space's wallpaper and triggers an immediate visual refresh — which is the correct behavior.

**3. Fixed launchd job path — `Developer/` instead of `code/`**

- `Developer/machine-cfg/umac/tv.jibb.wd.plist`: Changed `ProgramArguments` and `WorkingDirectory` from `/Users/blake/code/weatherdesktop` to `/Users/blake/Developer/weatherdesktop`.
- `code → Developer/` is a symlink, so the binary was identical, but `os.Args[0]` returned the symlink path. This caused `scriptDir` to resolve as `code/...`, so WallpaperAgent recorded all wallpaper paths using the `code/` alias — building up a separate history from the `Developer/` paths.
- Launchd job reloaded with `launchctl bootout` + `launchctl bootstrap`.

**4. Fixed cache cleanup to use `defaults write` — `flush_wallpaper_cache.sh` (now a fallback)**

- Previous cleanup used `rm -f` on the extension plist. **This never worked.** `cfprefsd` holds preferences in an in-memory cache and restores the file immediately on next access, regardless of deletion.
- `flush_wallpaper_cache.sh` step 10 now uses `defaults write <domain> "ChoiceRequests.ImageFiles" -array` (and similarly for `ChoiceRequests.Assets` and `ChoiceRequests.CollectionIdentifiers`). Writing through `defaults` updates cfprefsd's cache directly, so the old data is truly gone.
- **Now a fallback**: As of Feb 24, 2026, this cleanup happens on every `wd` run (see #5 below), so the nightly launchd job is no longer the primary cleanup mechanism.

**5. Per-run plist and cache cleanup (Feb 24, 2026)**

- `pkg/desktop/macos.go` — `clearContainerCache()` now:
  1. Clears plist entries via `defaults write ChoiceRequests.ImageFiles -array` (and Assets, CollectionIdentifiers).
  2. Deletes all files from both cache directories:
     - `~/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/...` (old path)
     - `~/Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches/` (UUID-named JPGs)
- `cmd/wd/main.go` — `setDesktopWallpaper()` now **always** calls `ClearWallpaperCache(verbose, true)` before setting the wallpaper. The `-clear-cache` flag is retained for backward compat but is now a no-op.
- **Result**: Plist entries stay at **0→1 per run** (no daily accumulation). Cache files stay at **0→1 per run** (freed **22GB** immediately on first run).
- The nightly `flush_wallpaper_cache.sh` remains as a belt-and-suspenders fallback but is no longer the primary mechanism.

#### SDK Investigation Notes

Searched the macOS SDK (`MacOSX.sdk`) for wallpaper internals:

- **`NSWorkspaceDesktopImageAllSpacesKey`** — exported in `AppKit.tbd` but absent from `NSWorkspace.h`. Usable via `extern NSString * const NSWorkspaceDesktopImageAllSpacesKey;`.
- **`Wallpaper.framework` (private)** — Swift XPC framework. Relevant symbols: `AgentXPCMessage.addChoiceRequest`, `removeChoiceRequest`, `snapshotAllSpaces()`, `DisplaySpacesInfo`, `LegacyDesktopPictureConfiguration`.
- **`DesktopPictureSetDisplayForSpace` / `DesktopPictureCopyDisplayForSpace`** — private C APIs in `HIServices.framework` for per-space wallpaper operations (not used, but available if finer-grained space control is needed in future).
- **`ChoiceRequests.ImageFiles`** — the specific NSUserDefaults key that accumulates wallpaper history. Clearing via `defaults write ... -array` is the effective reset method.

#### Test Results (February 22, 2026)

| Metric | Before | After |
|---|---|---|
| Extension plist size | ~15.2MB, ~15K entries | ~1.5KB, 1 entry |
| Plist after daily flush | Rebuilt to 15MB instantly | Stays at ~357 bytes |
| WallpaperAgent CPU | 100% hang, 20–30s | Normal |
| Desktop update | Blocked until process killed | Updates cleanly each run |

#### TODO

- ~~**Consider reverting `~/Pictures/Desktop/` indirection.**~~ **Reverted (Feb 24, 2026)**.
- ~~**Nightly cleanup via `flush_wallpaper_cache.sh` launchd job.**~~ **Replaced with per-run cleanup (Feb 24, 2026)** — Plist entries and cache files are now cleared before each wallpaper set, keeping both at 0→1 and freeing 22GB immediately. The nightly flush remains as a fallback.

#### Files Modified

| File | Change |
|---|---|
| `cmd/wd/main.go` | Removed `syncDesktopPicturesDir()` and fixed filename logic; `setDesktopWallpaper()` now sets directly from timestamped `rendered/` paths; always calls full cleanup before every wallpaper set |
| `pkg/desktop/macos.go` | `NSWorkspaceDesktopImageAllSpacesKey` added then removed; `clearContainerCache()` now clears plist entries (via `defaults write`) and both cache directories (0→1 per run) |
| `flush_wallpaper_cache.sh` | Updated with comment noting step 10 is now a fallback; per-run cleanup is the primary mechanism |
| `machine-cfg/umac/tv.jibb.wd.plist` | `ProgramArguments` and `WorkingDirectory` updated from `code/` → `Developer/` |

---

### Desktop Pictures Directory Isolation (February 2026) — REVERTED

> **Superseded and reverted Feb 24, 2026.** This section documents the earlier (now-removed) approach to isolating wallpaper to a fixed filename in `~/Pictures/Desktop/`. The revert was safe because the actual root causes of the hang were addressed by the other three fixes (#2, #3, #4).

**Initial approach (now reverted):**
- Copied the current rendered image to `~/Pictures/Desktop/weather-desktop.jpg` (fixed filename — later revised from the original timestamped approach).
- `setDesktopWallpaper()` set the wallpaper from this copy instead of directly from `rendered/`.
- Stale `.jpg` files in `~/Pictures/Desktop/` were removed each run.
- `rendered/` files were never deleted.

**Why it was effective but unnecessary:** Using a fixed filename prevented `WallpaperImageExtension` from adding new history entries, which would have slowed the plist growth. However, the real problem-solvers were:
1. **`defaults write` cache cleanup** — actually clears the existing 15MB plist through `cfprefsd`.
2. **`NSWorkspaceDesktopImageAllSpacesKey`** — eliminates per-space references that would otherwise persist as stale paths.
3. **Canonical launchd paths** — prevents duplicate `code/` vs `Developer/` entries.

With these three fixes in place, the indirection becomes unnecessary because:
- New paths no longer accumulate indefinitely (all spaces updated each run).
- The daily flush actually works, keeping plist size under control.
- Paths can safely accumulate through the day and be wiped nightly.

### Cache Cleanup Implementation (January 2026)

**Changes Made:**
1. **Enhanced Cache Cleanup**: Updated `ClearWallpaperCache()` to clear both cache locations:
   - TMPDIR-based cache: `${TMPDIR}../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/` (always cleared)
   - Container-based cache: `~/Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image/` (optional, requires `-clear-cache` flag)
2. **Optional Container Cache Cleanup**: Added `-clear-cache` flag to enable Container cache cleanup
   - Default behavior: Only clears TMPDIR cache (no security prompts)
   - With `-clear-cache`: Clears both caches (may trigger macOS security prompt on first use)
3. **Timing Change**: Cache clearing now happens **before** setting wallpaper (was previously after)
4. **Complete File Removal**: Now removes all files in cache directories (not just PNGs)
5. **Split Functions**: Separated into `clearTMPDIRCache()` and `clearContainerCache()` for maintainability
6. **Graceful Degradation**: If one cache location fails to clear, the other is still attempted

**Root Cause**: macOS wallpaper system creates cached copies in Container-based storage that accumulate over time, causing disk space issues. Container cache access requires special permissions, so it's now opt-in.

**Usage:**
```bash
# Normal run (TMPDIR cache only, no prompts)
./wd

# Full cache cleanup (includes Container cache, may prompt first time)
./wd -clear-cache

# Desktop setting with full cleanup
./wd -p -clear-cache
```

### Image Cropping and Positioning Updates (December 15, 2025)

**Changes Made:**
1. **Fixed NWAC Avalanche Forecast Cropping Regression**:
   - Corrected crop parameters that were causing search bar to show and bottom clipping
   - NWAC Avalanche Forecast Map: Updated from `Rect(0, 0, 500, 500)` to `Rect(65, 110, 465, 630)` with target size `400x520` (maintains aspect ratio)
   - NWAC Stevens Observations: Corrected from `Rect(0, 0, 870, 870)` to `Rect(0, 0, 1140, 1439)` with 75% resize
   - Weather.gov Extended Forecast: Fixed from `Rect(0, 0, 1150, 250)` to `Rect(0, 100, 1146, 400)` to skip header

2. **NWAC Stevens Avalanche Forecast**:
   - Removed processed `_s.jpg` version (no longer needed)
   - Now uses raw PNG directly in composite at position (3100, 60)
   - Eliminated unnecessary cropping/resizing step

3. **Highway 2 Pass Status Graphic**:
   - Moved from X:3150 to X:3050 (100 pixels left) to prevent overlap with avalanche forecast
   - Added logic to skip graphic display when pass is completely open (both directions)
   - When pass is open, removes any existing `pass_conditions.png` file

4. **Weather.gov Extended Forecast Positioning**:
   - Moved up 50 pixels from Y:1860 to Y:1810 in composite layout

5. **Stevens Pass Skyline Camera Added**:
   - Added download target: `https://streamer8.brownrice.com/cam-images/stevenspassskyline.jpg`
   - Positioned at (900, 1055) - below Jupiter camera with 50px clearance
   - No cropping or resizing applied (uses raw 1280x720 image)

6. **GOES18 Background Satellite Crop Reverted**:
   - Reverted from custom crop `Rect(1200, 50, 5040, 2210)` back to original `Rect(0, 0, 7200, 4050)`
   - Returns to full wide-view background satellite image

7. **Stevens Pass School Camera Added**:
   - Added download target: `https://streamer3.brownrice.com/cam-images/stevenspassschool.jpg`
   - Positioned at (900, 1820) - below Skyline camera with 50px clearance
   - No cropping or resizing applied (uses raw 1280x720 image)

8. **Camera Layout Reorganization**:
   - Moved Snow Stake from (910, 1730) to (2010, 285) - new column right of Jupiter
   - Moved Courtyard from (1600, 1730) to (2010, 697) - below Snow Stake in new column
   - Created organized column structure: Column 1 (Jupiter/Skyline/School at X=905) and Column 2 (Snow Stake/Courtyard at X=2010)
   - Scaled Jupiter, Skyline, School cameras to 84% (1075x605) with 30px vertical spacing
   - Eliminated 40-pixel overlap between Skyline and Snow Stake cameras

9. **Weather.gov Hourly Forecast Scaling**:
   - Scaled from original 800x871 to 855x930 to match NWAC observations width (855px)
   - Maintains aspect ratio: 1.0688x scale factor
   - Positioned at (20, 1130) aligned below NWAC observations
   - Scaled version uses `weather_gov_hourly_forecast_s.jpg`

**Root Cause of Regressions:**
- When converting from old `CropParams{X, Y, Width, Height}` to new `image.Rect(x0, y0, x1, y1)` system, some coordinates were incorrectly specified
- Old system used starting position + dimensions; new system uses top-left and bottom-right corner coordinates
- Incorrect conversion caused cropping from wrong positions and aspect ratio distortion

### WSDOT Pass Status Scraping Fixes

**Issue**: WSDOT pass closure detection was not working due to outdated CSS selector and timeout issues.

**Root Cause**: 
- Old selector `#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1` relied on DOM structure that changed
- WSDOT website migrated to Vue.js, requiring different wait strategies
- `networkidle` wait strategy timed out due to continuous network activity

**Solutions Implemented**:
1. **Updated CSS Selector**: Changed to `.full-width.column-container.mountain-pass .column-1` (class-based, more reliable)
2. **Improved Wait Strategy**: 
   - Use `domcontentloaded` instead of `networkidle` (same as image scrapers)
   - Added 3-second additional wait for Vue.js hydration
   - Increased element wait time to 10 seconds
3. **Better HTML Extraction**: Use `Page.Evaluate()` instead of `Locator.InnerHTML()` for more reliable extraction from Vue.js-rendered DOM
4. **Graphics-Based Rendering**: Replaced text rendering with pre-rendered PNG graphics (hw2_*.png files)
5. **Docker Optimization**: Added `.dockerignore` to exclude large `rendered/` directory from build context

**Test Files**: Created comprehensive test HTML files in `testfiles/` directory:
- `closed_wsdot_stevens_pass_2025_12_10_rain.html` - Current closed status
- `closed_wsdot_stevens_pass.html` - Historical closed status
- `open_wsdot_stevens_pass_2024_01_10.html` - Open status

**Result**: Pass closure detection now works correctly, displaying appropriate graphics based on east/west closure status.

## Architecture

### Hybrid Docker + Host Design

The tool uses a hybrid architecture:
- **Host binary** (`wd`): Orchestrates Docker and sets macOS desktop wallpaper (CGO)
- **Container** (`wd-worker`): Handles web scraping, downloads, and image processing

This design isolates browser automation (Playwright + WebKit) in a container while keeping macOS-specific desktop setting on the host.

```
┌─────────────────────────────────────────────────────────┐
│  macOS Host                                             │
│  ┌───────────────────────────────────────────────────┐  │
│  │  wd binary (orchestrates Docker + sets wallpaper) │  │
│  └──────────────────┬────────────────────────────────┘  │
│                     │                                    │
│  ┌──────────────────▼──────────────────────────────┐    │
│  │  Docker Compose v2                              │    │
│  │  ┌────────────────────────────────────────────┐ │    │
│  │  │  wd-worker container                       │ │    │
│  │  │  ┌──────────────────────────────────────┐  │ │    │
│  │  │  │  Playwright-Go + WebKit              │  │ │    │
│  │  │  │  - Web scraping                      │  │ │    │
│  │  │  │  - Image downloads                   │  │ │    │
│  │  │  │  - Image cropping/resizing           │  │ │    │
│  │  │  │  - Composite rendering               │  │ │    │
│  │  │  └──────────────────────────────────────┘  │ │    │
│  │  └────────────────────────────────────────────┘ │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
│  Shared Volumes:                                         │
│  ./assets   ←→  /app/assets   (scraped/downloaded)      │
│  ./rendered ←→  /app/rendered (final composites)        │
│  ./graphics ←→  /app/graphics (pass status graphics)    │
└─────────────────────────────────────────────────────────┘
```

### Why Docker?

**Problems Solved:**
1. **True Headless Mode**: Playwright WebKit supports genuine headless operation (unlike Safari WebDriver)
2. **Isolated Environment**: Self-contained browser binaries, consistent dependencies
3. **Simpler Process Management**: Docker Compose handles lifecycle (no manual daemon startup)
4. **Better Concurrency**: Container isolation eliminates need for lock files

**Trade-offs:**
- ✅ Isolated execution environment
- ✅ Easy dependency management
- ✅ Consistent across machines
- ❌ Additional Docker Desktop requirement
- ❌ ~500MB memory overhead

## Migration from Safari WebDriver

### Previous Architecture

**Before:** Single `wd` binary using Safari WebDriver for scraping
- Required `safaridriver` running on host
- No true headless mode (window minimization workaround)
- Lock file needed to prevent concurrent runs
- System-level dependency complexity

### Current Architecture

**After:** Hybrid architecture with `wd` host orchestrator + `wd-worker` container (Playwright/WebKit)
- True headless mode
- No manual daemon startup
- No lock file needed (container isolation)
- Better reliability and reproducibility

### Code Changes

**Removed Components:**
- `pkg/webdriver/` - Safari WebDriver client
- `pkg/scraper/` - Safari-based scraper
- `pkg/lockfile/` - Lock file management (no longer needed)

**New Components:**
- `pkg/playwright/scraper.go` - Playwright WebKit automation
- `pkg/docker/client.go` - Docker Compose orchestration
- `cmd/wd-worker/main.go` - Container entry point
- `Dockerfile` - Container image definition
- `compose.yaml` - Docker Compose v2 configuration

**Modified Components:**
- `cmd/wd/main.go` - Now orchestrates Docker instead of direct execution
- `Makefile` - Added Docker management targets

## Docker Configuration

### compose.yaml

```yaml
services:
  wd-worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: wd-worker
    volumes:
      - ./assets:/app/assets
      - ./rendered:/app/rendered
      - ./graphics:/app/graphics
    working_dir: /app
    init: true                    # Tini for process management
    restart: unless-stopped
    environment:
      - TZ=America/Los_Angeles    # PST timezone for filenames
    command: sh -c "while true; do sleep 3600; done"  # Keep running
```

**Key Points:**
- **Persistent container**: Stays running to avoid startup overhead
- **Volume mounts**: Share `assets/`, `rendered/`, and `graphics/` directories with host
- **init: true**: Uses Docker's built-in Tini for signal handling and zombie reaping
- **Timezone**: Set to PST for consistent filename timestamps

### Dockerfile

```dockerfile
FROM ubuntu:24.04

# Install Go 1.23.3
RUN apt update && apt install -y wget ca-certificates && \
    wget -q https://go.dev/dl/go1.23.3.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.3.linux-amd64.tar.gz && \
    rm go1.23.3.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

WORKDIR /app

# Install Playwright and WebKit
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps webkit

# Install CA certificates for TLS
RUN update-ca-certificates && \
    wget -q -O /usr/local/share/ca-certificates/rapidssl-tls-rsa-ca-g1.crt \
    https://cacerts.digicert.com/RapidSSLTLSRSACAG1.crt.pem && \
    chmod 644 /usr/local/share/ca-certificates/rapidssl-tls-rsa-ca-g1.crt && \
    update-ca-certificates

# Build worker binary
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wd-worker ./cmd/wd-worker/main.go

ENTRYPOINT ["/app/wd-worker"]
```

**Key Points:**
- **Ubuntu 24.04**: Latest LTS base image
- **Playwright installation**: Installs WebKit browser and system dependencies via `--with-deps`
- **TLS certificates**: Installs RapidSSL intermediate CA for `brownrice.com` downloads
- **Static binary**: `CGO_ENABLED=0` for portability

## Command Flow

### Example: Full Pipeline (`./wd`)

1. **Host `wd` binary** checks Docker container status
2. If not running, starts `wd-worker` container via Docker Compose
3. Executes commands in container via `docker compose exec`:
   - `wd-worker scrape` - Playwright WebKit scraping
   - `wd-worker download` - HTTP image downloads
   - `wd-worker crop` - Image processing
   - `wd-worker render` - Composite generation
4. **Host `wd` binary** reads rendered image from shared volume
5. Uses CGO to set macOS desktop wallpaper

### Volume Mounts

**`./assets ↔ /app/assets`**
- Downloaded images (satellite, webcams)
- Scraped screenshots (forecasts, avalanche data)
- Processed/cropped images

**`./rendered ↔ /app/rendered`**
- Final composite images with timestamps (`hud-YYMMDD-HHMM.jpg`)
- **⚠️ CRITICAL: Never delete files from the rendered directory**
  - These are archived historical images that serve as a record
  - The `findMostRecentRendered()` function relies on these files to find the latest wallpaper
  - Manual deletion or automated cleanup scripts should NEVER target this directory
  - The `clean` Makefile target has been updated to preserve rendered images

**`./graphics ↔ /app/graphics`**
- Pre-rendered pass status graphics (hw2_*.png files)
- Used for pass closure status display
- Graphics are selected based on parsed WSDOT status and copied to `assets/pass_conditions.png`

## Scraping Implementation

### Playwright Configuration

**Headless Mode:**
- Always runs headless in Docker (`Headless: playwright.Bool(true)`)
- Debug mode shows browser logs but not GUI (no X11 forwarding)

**Timeouts:**
- Page navigation: 10 seconds
- Element screenshot: 10 seconds
- Network requests: 10 seconds (via downloader)

**Wait Strategy:**
- Uses `domcontentloaded` for NWAC sites (React apps)
- Uses `networkidle` for static sites

### Scrape Targets

**Weather.gov Hourly Forecast:**
- Selector: `img[src*="meteograms/Plotter.php"]`
- Wait time: 5000ms
- Output: 800x870px meteogram image

**Weather.gov Extended Forecast:**
- Selector: `#seven-day-forecast`
- Wait time: 1000ms
- Output: Complete 7-day forecast panel

**NWAC Sites:**
- Wait time: 15000ms (JavaScript-heavy React apps)
- Navigation strategy: `domcontentloaded` (don't wait for network idle)
- Selectors configured per target in `pkg/assets/manager.go`

**WSDOT Pass Status:**
- Selector: `.full-width.column-container.mountain-pass .column-1`
- Type: HTML extraction (not screenshot)
- Wait time: 10000ms (10 seconds for Vue.js to render)
- Navigation: Uses `domcontentloaded` wait strategy (same as image scrapers)
- Additional wait: 3000ms after navigation for Vue.js hydration
- Extraction: Uses `Page.Evaluate()` for reliable HTML extraction from Vue.js-rendered DOM
- HTML structure: Uses `class="condition"` wrappers with `class="conditionLabel"` and `class="conditionValue"` children
- Parsed for eastbound/westbound closure status
- **Graphics-based rendering**: Uses pre-rendered PNG graphics instead of text

## Image Processing

### Downloader

- **Concurrent downloads**: Uses goroutines for parallel HTTP requests
- **Retry logic**: Exponential backoff on failures
- **Fallback**: Creates 1x1 transparent PNG on failure
- **TLS**: Uses system CA certificates loaded from `/etc/ssl/certs/ca-certificates.crt`
- **Timeout**: 10 seconds per request

### Processor

- **Cropping**: Uses `image.SubImage()` for precise region extraction
- **Resizing**: Uses `golang.org/x/image/draw.CatmullRom` for high-quality scaling
- **No ImageMagick**: Pure Go implementation

### Compositor

- **Canvas**: 3840x2160 (4K) sky blue background
- **Layering**: Uses stdlib `image/draw.Draw()` for compositing
- **15 layers**: Positioned at precise coordinates from `pkg/assets/manager.go`

### Pass Status Graphics

- **Graphics-based system**: Uses pre-rendered PNG graphics instead of text rendering
- **Graphics location**: `graphics/` directory (mounted in Docker container)
- **Graphic selection logic**:
  - `hw2_open.png` - Pass is open (neither direction closed)
  - `hw2_closed.png` - Both directions closed
  - `hw2_closed_e.png` - Only eastbound closed
  - `hw2_closed_w.png` - Only westbound closed
- **Closure detection**: Parses WSDOT HTML to detect "Closed" status in eastbound/westbound conditions
- **File copying**: Selected graphic is copied to `assets/pass_conditions.png` for compositing
- **Fallback**: Uses `hw2_open.png` if parsing fails or graphic not found

## Image Composite Layout

### Canvas Specifications
- **Resolution**: 3840x2160 (4K UHD)
- **Background**: Sky blue RGB(135, 206, 235)
- **Total Layers**: 17

### Layer Map

#### Background Layer
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| background_s.jpg | (0, 0) | 3840x2160 | GOES18 satellite image (full canvas) |

#### Top Row - WSDOT Road Cameras (Y=20)
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| wsdot_us2_skykomish.jpg | (900, 20) | 335x249 | US2 at Skykomish |
| wsdot_w_stevens.jpg | (1250, 20) | 335x249 | West Stevens |
| wsdot_big_windy.jpg | (1600, 20) | 335x249 | Big Windy |
| wsdot_stevens_pass_b.jpg | (1950, 20) | 400x225 | Stevens Pass (main) |
| wsdot_e_stevens_summit.jpg | (2360, 20) | 335x249 | East Stevens Summit |

#### Left Column 1 - NWAC Data
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| nwac_stevens_observations_s.jpg | (20, 20) | 855x1079 | Weather observations graph |
| weather_gov_hourly_forecast_s.jpg | (20, 1130) | 855x930 | Hourly meteogram (scaled to match observations width) |

#### Center Column 1 - Stevens Pass Cameras (X=905)
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| stevenspassjupiter_s.jpg | (905, 285) | 1075x605 | Jupiter camera view (84% scaled) |
| stevenspassskyline_s.jpg | (905, 920) | 1075x605 | Skyline camera view (84% scaled) |
| stevenspassschool_s.jpg | (905, 1555) | 1075x605 | School camera view (84% scaled) |

#### Center Column 2 - Processed Cameras (X=2010)
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| stevenspasssnowstake_s.jpg | (2010, 285) | 680x382 | Snow stake camera |
| stevenspasscourtyard_s.jpg | (2010, 697) | 680x382 | Courtyard camera |

#### Bottom Center - Extended Forecast
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| weather_gov_extended_forecast_s.jpg | (2680, 1810) | 1146x300 | 7-day forecast panel |

#### Right Column - Avalanche Info
| Image | Position | Dimensions | Description |
|-------|----------|------------|-------------|
| nwac_stevens_avalanche_forcast.png | (3100, 60) | 718x281 | Current danger rating |
| pass_conditions.png | (3050, 420) | 342x342 | Highway 2 pass status |
| nwac_avalanche_forcast_s.jpg | (3420, 420) | 400x520 | Regional forecast map |

### Layout Principles
- **Column-based organization**: Related cameras grouped vertically
- **Top row alignment**: All WSDOT road cameras at Y=20
- **50px minimum spacing**: Between stacked images to prevent overlap
- **Right alignment**: Critical avalanche/road status in rightmost column
- **Z-order**: Layers applied bottom-to-top in order listed in `GetCompositeLayout()`

## Desktop Setting

### macOS Implementation

**CGO with Objective-C:**
- Direct NSWorkspace API calls (`[[NSWorkspace sharedWorkspace] setDesktopImageURL:forScreen:options:error:]`)
- Sets wallpaper on all screens automatically
- Merges existing screen options to preserve per-screen settings (fill color, etc.)
- Clears wallpaper caches **before** setting new wallpaper (prevents cache buildup)
- TMPDIR cache: Always cleared (no permissions needed)
- Container cache: Only cleared with `-clear-cache` flag (may prompt for permissions)
- 0.5 second delay after setting to allow system to process
- 0.5 second delay after setting to allow system to process

**Host-only:**
- Desktop setting must run on macOS host (cannot use Docker)
- CGO requires Xcode Command Line Tools

**Multi-Display Support:**
- Automatically detects all connected displays via `[NSScreen screens]`
- Loops through all screens and sets wallpaper on each
- Preserves existing screen-specific options (like fill color) when merging with new options
- Continues even if one screen fails (logs warning but doesn't abort)

### Filename Handling

- Container generates filenames with PST timezone (`hud-YYMMDD-HHMM.jpg`)
- Host finds most recent rendered file using `findMostRecentRendered()`
- Ensures correct file is used for desktop setting even if timing differs

## Performance

### Container Startup

- **Cold start** (first build): ~5-10 minutes (downloads WebKit)
- **Warm start** (cached image): ~2-3 seconds
- **Exec commands** (container running): <1 second overhead

**Optimization**: Keep container running between executions.

### Memory Usage

- Docker Desktop: ~400MB baseline
- `wd-worker` container: ~500MB-1GB
- `wd` binary: ~20MB
- **Total: ~920MB-1.4GB**

### Pipeline Execution

- **Full pipeline**: ~1-2 minutes (network dependent)
- **Scraping**: ~30-60 seconds (depends on site response)
- **Downloads**: ~5-10 seconds (concurrent)
- **Processing**: ~1-2 seconds
- **Rendering**: ~1-2 seconds

## Development Workflow

### File Management

**⚠️ IMPORTANT: Rendered Directory Preservation**
- The `./rendered/` directory contains archived composite images
- **Never delete files from this directory** - they serve as historical records
- The `clean` Makefile target preserves rendered images
- Only `assets/` directory files are temporary and can be cleaned

### Making Changes

**Worker code** (`cmd/wd-worker/`, `pkg/playwright/`, etc.):
```bash
# Edit files
vim pkg/playwright/scraper.go

# Rebuild container
make docker-build

# Restart container
make docker-restart

# Test
./wd -s -debug
```

**Host code** (`cmd/wd/`, `pkg/desktop/`, etc.):
```bash
# Edit files
vim cmd/wd/main.go

# Rebuild host binary
make build

# Test
./wd -debug
```

**Full rebuild:**
```bash
make rebuild  # Builds Docker image + host binary
```

### Testing

**Individual phases:**
```bash
./wd -s        # Scrape only
./wd -d        # Download only
./wd -c        # Crop only
./wd -r        # Render only
./wd -p        # Set desktop only
```

**Debug mode:**
```bash
./wd -s -debug                    # Scrape with debug logging
./wd -s -scrape-target "NWAC" -debug  # Test specific target
```

**Manual container testing:**
```bash
make docker-shell
# Inside container:
/app $ wd-worker scrape --debug
/app $ exit
```

## Troubleshooting

### Docker Issues

**Container not starting:**
```bash
# Check Docker is running
docker info

# View logs
make docker-logs

# Restart
make docker-restart
```

**Playwright failed to launch:**
```bash
# Rebuild with clean slate
docker compose down
docker compose build --no-cache
docker compose up -d
```

**Volume mount issues:**
```bash
# Check Docker Desktop settings
# Settings > Resources > File Sharing
# Ensure project directory is allowed
```

### Scraping Issues

**Timeout errors:**
- Check network connectivity
- Verify target URLs are accessible
- Increase wait times in `pkg/assets/manager.go` if needed
- For Vue.js/React sites, use `domcontentloaded` wait strategy (not `networkidle`)

**WSDOT Pass Status scraping:**
- **Issue**: WSDOT website uses Vue.js and requires additional wait time for hydration
- **Solution**: Added 3-second wait after `domcontentloaded` for Vue.js to render
- **Selector**: Updated from `#index > div:nth-child(7)...` to `.full-width.column-container.mountain-pass .column-1`
- **Extraction**: Uses `Page.Evaluate()` instead of `Locator.InnerHTML()` for more reliable extraction
- **Timeout**: 30 seconds for navigation, 10 seconds for element wait
- **Test files**: See `testfiles/` directory for HTML samples (closed/open states)

**Selector failures:**
- Use debug mode to see what's being captured
- Check browser logs in container: `make docker-logs`
- Update selectors in `pkg/assets/manager.go`
- For dynamic sites, verify element exists after JavaScript renders

**TLS certificate errors:**
- CA certificates are installed in Dockerfile
- If new site fails, add its CA certificate to Dockerfile

### Build Issues

**CGO errors (host binary):**
```bash
# Install Xcode Command Line Tools
xcode-select --install
```

**Go version mismatch:**
```bash
# Check go.mod requires Go 1.23
# Update Dockerfile if needed
```

## Dependencies

### Host Binary (`cmd/wd`)
- `golang.org/x/image/draw` - Image scaling
- `golang.org/x/image/font` - Text rendering
- `golang.org/x/net/html` - HTML parsing
- CGO - macOS Cocoa APIs

### Container (`cmd/wd-worker`)
- `github.com/playwright-community/playwright-go` - Browser automation
- WebKit browser binaries (installed in Docker image)
- Standard library packages for image processing

### Standard Library Focus

**Replaced external tools:**
- `wget` → `net/http.Client`
- `shot-scraper` → Playwright-Go
- `pup` → `x/net/html` parser
- `jq` → Native Go structs
- `ImageMagick convert` → `image/draw` + `x/image/draw`
- `desktoppr` → CGO NSWorkspace
- `pass_status.sh` → Go-based parser with graphics-based rendering

## Upcoming Tasks

### Next Items
- **Add Stevens Pass School Camera**: Add download target for `https://streamer3.brownrice.com/cam-images/stevenspassschool.jpg` and position in composite layout

## Future Enhancements

### Short Term
- Health check endpoint for container
- Graceful shutdown handling
- Configurable timeout and retry logic

### Medium Term
- Support multiple browser engines (Chromium, Firefox)
- Parallel scraping with multiple containers
- Metrics and monitoring

### Long Term
- Cloud deployment (AWS ECS, Google Cloud Run)
- CI/CD integration (GitHub Actions)
- Historical data tracking

## See Also

- [README.md](README.md) - User documentation
- [Docker Compose v2 Documentation](https://docs.docker.com/compose/)
- [Playwright-Go Documentation](https://pkg.go.dev/github.com/playwright-community/playwright-go)
