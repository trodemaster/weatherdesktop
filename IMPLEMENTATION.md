# Implementation Details

Technical documentation for developers and contributors.

## Recent Changes (December 2025)

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

## Desktop Setting

### macOS Implementation

**CGO with Objective-C:**
- Direct NSWorkspace API calls (`[[NSWorkspace sharedWorkspace] setDesktopImageURL:forScreen:options:error:]`)
- Sets wallpaper on all screens automatically
- Merges existing screen options to preserve per-screen settings (fill color, etc.)
- Clears wallpaper cache for immediate update
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
