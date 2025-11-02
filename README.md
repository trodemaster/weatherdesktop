# Weather Desktop (Go)

A Go implementation of the weather desktop script that creates a composite desktop wallpaper with weather information, webcam feeds, and avalanche forecasts for Stevens Pass.

## Features

- ✅ **Pure Go & stdlib** - Uses only standard library and `golang.org/x/*` packages
- ✅ **Safari WebDriver** - Native macOS browser automation for scraping (1920x1200 window)
- ✅ **CGO Desktop Setting** - Direct NSWorkspace API calls for wallpaper setting
- ✅ **Concurrent Downloads** - Fast parallel image downloads with fallback handling
- ✅ **Image Processing** - Crop, resize, and composite using stdlib `image/draw`
- ✅ **Flag-based CLI** - Simple command-line interface matching original bash script
- ✅ **Lock File Protection** - Prevents concurrent production runs; test mode bypasses lock
- ✅ **Smart Element Capture** - Large window size ensures full content screenshots

## Prerequisites

1. **Go 1.21+** - Install from [golang.org](https://golang.org)
2. **Safari WebDriver** - Enable and start:
   ```bash
   sudo safaridriver --enable
   safaridriver --port=4444 &
   ```
3. **macOS** - Required for desktop wallpaper setting (Cocoa APIs)

## Installation

### Using Make (Recommended)

```bash
cd /Users/blake/code/weatherdesktop

# Build for current OS/arch (auto-detected)
make build

# Or simply
make
```

### Using Go directly

```bash
go build -o wd ./cmd/wd
```

### Makefile Targets

```bash
make build        # Build binary (default)
make clean        # Remove build artifacts
make deps         # Install/update dependencies
make info         # Show build information
make run          # Build and run full pipeline
make run-debug    # Build and run with debug
make list-targets # List scrape targets
make help         # Show all targets
```

## Usage

### Run Full Pipeline (Default)
```bash
./wd
```
This will:
1. Download weather satellite and webcam images
2. Scrape weather forecasts and avalanche data
3. Crop and resize all images
4. Composite into final 3840x2160 image
5. Set as desktop wallpaper

### Individual Phases

```bash
./wd -s        # Scrape websites only
./wd -d        # Download images only
./wd -c        # Crop/resize images only
./wd -r        # Render composite only
./wd -p        # Set desktop wallpaper only
./wd -f        # Flush/clear assets directory
```

### Combined Phases

```bash
./wd -d -c -r  # Download, crop, and render (skip scraping)
./wd -r -p     # Render and set desktop
```

### Debug Mode

By default, Safari runs **headless** (no visible browser window). Enhanced debug capabilities:

```bash
# Basic debug (show browser)
./wd -s -debug        # Scrape with visible Safari browser
./wd -debug           # Full pipeline with visible browser

# Test specific scrape target
./wd -s -scrape-target "NWAC Stevens" -debug

# Increase wait time for slow pages (10 seconds)
./wd -s -scrape-target "NWAC Avalanche" -wait 10000 -debug

# Keep browser open for manual inspection
./wd -s -debug -keep-browser

# Save both full page and cropped element screenshots
./wd -s -scrape-target "Weather.gov Hourly" -debug -save-full-page

# Combine options
./wd -s -scrape-target "NWAC" -wait 8000 -debug -keep-browser -save-full-page
```

#### Debug Features

- **Smart Wait** (default): Polls for element every 100ms, proceeds when found
- **Manual Wait Override**: `-wait <ms>` forces fixed wait time
- **Verbose Logging**: Shows URL, selector, timing, element detection status
- **Timestamped Screenshots**: Debug mode adds timestamp to filenames
- **Full Page Capture**: `-save-full-page` saves both full page and element crops
- **Browser Persistence**: `-keep-browser` leaves Safari open for inspection
- **Target Filtering**: Test individual scrapers without running full pipeline
- **Safety**: Debug mode automatically skips desktop wallpaper setting

## Project Structure

```
weatherdesktop/
├── cmd/wd/main.go              # CLI entry point
├── pkg/
│   ├── assets/                 # Asset configuration & paths
│   ├── downloader/             # HTTP downloads with retry
│   ├── scraper/                # Safari WebDriver scraping
│   ├── parser/                 # HTML parsing with x/net/html
│   ├── image/                  # Image processing & compositing
│   │   ├── processor.go        # Crop & resize
│   │   ├── compositor.go       # Layer images
│   │   └── text.go             # Text rendering
│   ├── desktop/                # macOS wallpaper setting (CGO)
│   └── webdriver/              # Safari WebDriver client
├── assets/                     # Downloaded/scraped images
└── rendered/                   # Final composite outputs
```

## Dependencies

All dependencies are standard library or `golang.org/x/*`:

- `golang.org/x/image/draw` - Image scaling with interpolation
- `golang.org/x/image/font` - Text rendering
- `golang.org/x/net/html` - HTML parsing

## Data Sources

### Downloaded Images
- NOAA GOES-18 North Pacific Satellite
- WSDOT Traffic Cameras (5 locations)
- Stevens Pass Webcams (3 locations)

### Scraped Data
- Weather.gov hourly & extended forecasts
- NWAC avalanche observations & forecasts
- WSDOT mountain pass status

## Output

Final composite image: `rendered/hud-YYMMDD-HHMM.jpg`
- Resolution: 3840x2160 (4K)
- Format: JPEG (quality 90)
- Sky blue background with layered images

## Differences from Bash Version

1. **Safari WebDriver** instead of shot-scraper (Python)
2. **Pure Go image processing** instead of ImageMagick
3. **CGO NSWorkspace** instead of desktoppr binary
4. **stdlib flag** instead of custom argument parsing
5. **Concurrent downloads** with goroutines instead of background jobs
6. **Better error handling** with fallback to empty images

## Lock File & Concurrent Runs

### Production Mode (Lock Protected)

Normal runs use a lock file to prevent conflicts:
```bash
./wd              # Exits if another instance is running
make run          # Protected by lock file
```

### Test Mode (Safe Alongside Production)

Debug and target-specific runs bypass the lock:
```bash
./wd -debug                           # Safe to run anytime
./wd -scrape-target "NWAC" -debug     # Safe to run anytime
make run-debug                        # Safe to run anytime
```

Test runs use unique filenames (`hud-TEST-*.jpg`) to avoid conflicts.

**See [LOCKFILE.md](LOCKFILE.md) for complete documentation.**

## Troubleshooting

### Another Instance Already Running

```
Failed to acquire lock: another instance is already running (PID: 12345)
```

**Solutions:**
- Wait for other instance to finish
- Use test mode: `./wd -debug`
- Remove stale lock: `rm $TMPDIR/wd.lock`

### Safari WebDriver Not Running
```bash
# Check if running
ps aux | grep safaridriver

# Start it
safaridriver --port=4444 &
```

### Build Errors with CGO
Make sure Xcode Command Line Tools are installed:
```bash
xcode-select --install
```

### Missing Assets
Run with `-f` flag to clear assets and start fresh:
```bash
./wd -f
./wd
```

## License

Same as original bash script.
