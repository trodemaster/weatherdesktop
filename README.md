# Weather Desktop (Go)

A Go implementation of the weather desktop script that creates a composite desktop wallpaper with weather information, webcam feeds, and avalanche forecasts for Stevens Pass.

## Features

- ✅ **Pure Go & stdlib** - Uses only standard library and `golang.org/x/*` packages
- ✅ **Playwright WebKit** - Modern browser automation in Docker for reliable scraping
- ✅ **Docker Compose** - Isolated execution environment with persistent container
- ✅ **CGO Desktop Setting** - Direct NSWorkspace API calls for wallpaper setting (macOS host)
- ✅ **Concurrent Downloads** - Fast parallel image downloads with fallback handling
- ✅ **Image Processing** - Crop, resize, and composite using stdlib `image/draw`
- ✅ **Flag-based CLI** - Simple command-line interface matching original bash script

## Architecture

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
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **Go 1.21+** - Install from [golang.org](https://golang.org)
2. **Docker Desktop** - Install from [docker.com](https://www.docker.com/products/docker-desktop)
3. **Docker Compose v2** - Included with Docker Desktop
4. **macOS** - Required for desktop wallpaper setting (Cocoa APIs)

## Installation

### Quick Start

```bash
cd /Users/blake/code/weatherdesktop

# Build Docker image (first time only)
make docker-build

# Build host binary
make build

# Run full pipeline
make run
```

### Manual Build

```bash
# Build Docker image
docker compose build

# Build host binary
CGO_ENABLED=1 go build -o wd ./cmd/wd

# Run
./wd
```

### Makefile Targets

#### Build & Run
```bash
make build        # Build host binary (default)
make docker-build # Build Docker image with Playwright
make run          # Build and run full pipeline
make run-debug    # Build and run with debug
make clean        # Remove build artifacts
```

#### Docker Management
```bash
make docker-up      # Start wd-worker container
make docker-down    # Stop and remove container
make docker-restart # Restart container
make docker-logs    # Follow container logs
make docker-shell   # Open shell in container
make docker-ps      # List container status
```

#### Development
```bash
make deps         # Install/update dependencies
make info         # Show build information
make help         # Show all targets
```

## Usage

### Run Full Pipeline (Default)
```bash
./wd
```
This will:
1. Ensure Docker container is running
2. Download weather satellite and webcam images (in container)
3. Scrape weather forecasts and avalanche data (Playwright/WebKit in container)
4. Crop and resize all images (in container)
5. Composite into final 3840x2160 image (in container)
6. Set as desktop wallpaper (on host)

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

By default, Playwright WebKit runs **headless** in Docker. Debug mode shows the browser:

```bash
# Basic debug (show browser in container)
./wd -s -debug        # Scrape with visible WebKit browser
./wd -debug           # Full pipeline with visible browser

# Test specific scrape target
./wd -s -scrape-target "NWAC Stevens" -debug
./wd -s -scrape-target "Weather.gov Hourly" -debug

# Combine options
./wd -s -scrape-target "NWAC" -debug
```

#### Debug Features

- **Visible Browser**: `-debug` shows WebKit window in Docker
- **Verbose Logging**: Shows URL, selector, timing, element detection status
- **Target Filtering**: Test individual scrapers without running full pipeline
- **Real-time Logs**: Container logs stream to console in debug mode
- **Safety**: Debug mode automatically skips desktop wallpaper setting

## Project Structure

```
weatherdesktop/
├── cmd/
│   ├── wd/main.go              # Host orchestrator (Docker + desktop)
│   └── wd-worker/main.go       # Container worker (scrape/render)
├── pkg/
│   ├── assets/                 # Asset configuration & paths
│   ├── downloader/             # HTTP downloads with retry
│   ├── playwright/             # Playwright-Go WebKit scraping
│   ├── parser/                 # HTML parsing with x/net/html
│   ├── image/                  # Image processing & compositing
│   │   ├── processor.go        # Crop & resize
│   │   ├── compositor.go       # Layer images
│   │   └── text.go             # Text rendering
│   ├── desktop/                # macOS wallpaper setting (CGO)
│   └── docker/                 # Docker Compose client
├── assets/                     # Downloaded/scraped images (shared)
├── rendered/                   # Final composite outputs (shared)
├── Dockerfile                  # Worker container definition
└── compose.yaml                # Docker Compose v2 config
```

## Dependencies

### Host Binary (cmd/wd)
- `golang.org/x/image/draw` - Image scaling with interpolation
- `golang.org/x/image/font` - Text rendering
- `golang.org/x/net/html` - HTML parsing
- CGO - macOS Cocoa APIs for desktop setting

### Container (cmd/wd-worker)
- `github.com/playwright-community/playwright-go` - Browser automation
- WebKit browser binaries (installed in Docker image)
- Standard library packages for image processing

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

## How It Works

1. **Host `wd` binary** checks Docker container status
2. If not running, starts `wd-worker` container via Docker Compose
3. Executes commands in container via `docker compose exec`:
   - `wd-worker scrape` - Playwright WebKit scraping
   - `wd-worker download` - HTTP image downloads
   - `wd-worker crop` - Image processing
   - `wd-worker render` - Composite generation
4. **Host `wd` binary** reads rendered image from shared volume
5. Uses CGO to set macOS desktop wallpaper

## Differences from Bash Version

1. **Playwright WebKit in Docker** instead of shot-scraper (Python)
2. **Pure Go image processing** instead of ImageMagick
3. **CGO NSWorkspace** instead of desktoppr binary
4. **stdlib flag** instead of custom argument parsing
5. **Concurrent downloads** with goroutines instead of background jobs
6. **Better error handling** with fallback to empty images
7. **Container isolation** for reliable, reproducible scraping environment

## Troubleshooting

### Docker Container Not Starting

```bash
# Check Docker is running
docker info

# Check container status
make docker-ps
docker compose ps

# View logs
make docker-logs
docker compose logs -f wd-worker

# Restart container
make docker-restart
```

### Build Errors with CGO (Host Binary)

Make sure Xcode Command Line Tools are installed:
```bash
xcode-select --install
```

### Container Build Errors

```bash
# Clean rebuild
docker compose build --no-cache

# Check Docker resources (Settings > Resources)
# Playwright needs at least 2GB RAM
```

### Missing Assets

Run with `-f` flag to clear assets and start fresh:
```bash
./wd -f
./wd
```

### Playwright Browser Issues

If WebKit fails to launch in container:
```bash
# Rebuild with fresh Playwright install
docker compose down
docker compose build --no-cache
docker compose up -d
```

## Scheduled Runs

For periodic execution via cron or launchd:

```bash
# Add to crontab
*/30 * * * * cd /Users/blake/code/weatherdesktop && ./wd >> /tmp/wd.log 2>&1
```

The Docker container will stay running between executions, providing fast startup times.

## License

Same as original bash script.
