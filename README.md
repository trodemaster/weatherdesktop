# Weather Desktop

A tool that creates a composite desktop wallpaper with weather information, webcam feeds, and avalanche forecasts for Stevens Pass.

## Quick Start

```bash
# Build Docker image (first time only)
make docker-build

# Build host binary
make build

# Run full pipeline
./wd
```

## Prerequisites

- **macOS** - Required for desktop wallpaper setting
- **Go 1.23+** - [golang.org](https://golang.org)
- **Docker Desktop** - [docker.com](https://www.docker.com/products/docker-desktop)
- **Xcode Command Line Tools** - For CGO compilation

## Installation

```bash
cd /Users/blake/code/weatherdesktop

# Build Docker image (downloads Playwright/WebKit)
make docker-build

# Build host binary
make build

# Run
./wd
```

## Usage

### Default (Full Pipeline)

```bash
./wd
```

Runs all phases:
1. Scrape websites (weather forecasts, avalanche data)
2. Download images (satellite, webcams)
3. Crop/resize images
4. Render composite (3840x2160)
5. Set desktop wallpaper

### Individual Phases

```bash
./wd -s        # Scrape websites only
./wd -d        # Download images only
./wd -c        # Crop/resize images only
./wd -r        # Render composite only
./wd -p        # Set desktop wallpaper only
./wd -f        # Flush/clear assets directory
```

### Set Desktop from Specific Image

```bash
./wd -set-desktop rendered/hud-251102-1056.jpg
```

### Debug Mode

```bash
# Full pipeline with debug output
./wd -debug

# Test specific scrape target
./wd -s -scrape-target "NWAC Stevens" -debug
./wd -s -scrape-target "Weather.gov Hourly" -debug
```

### List Available Targets

```bash
./wd -list-targets
```

## Makefile Commands

### Build & Run
```bash
make build        # Build host binary
make docker-build # Build Docker image
make run          # Build and run full pipeline
make run-debug    # Build and run with debug
make clean        # Remove build artifacts
make rebuild      # Rebuild Docker image and host binary
```

### Docker Management
```bash
make docker-up      # Start container
make docker-down    # Stop container
make docker-restart # Restart container
make docker-logs    # View container logs
make docker-shell   # Open shell in container
make docker-ps      # List container status
```

## Output

Final composite image: `rendered/hud-YYMMDD-HHMM.jpg`
- Resolution: 3840x2160 (4K)
- Format: JPEG
- Sky blue background with layered weather data

## Data Sources

- **NOAA GOES-18** - North Pacific satellite imagery
- **Weather.gov** - Hourly and extended forecasts
- **NWAC** - Avalanche observations and forecasts
- **WSDOT** - Traffic cameras and mountain pass status
- **Stevens Pass** - Webcams and conditions

## Troubleshooting

### Docker Not Running
```bash
open -a Docker
# Wait for Docker to start, then retry
```

### Container Issues
```bash
# View logs
make docker-logs

# Restart container
make docker-restart

# Rebuild from scratch
docker compose down
docker compose build --no-cache
make docker-build
```

### Build Errors
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Check Go version
go version  # Should be 1.23+

# Clean rebuild
make clean
make rebuild
```

### Missing Assets
```bash
# Clear and start fresh
./wd -f
./wd
```

## Scheduled Runs

Add to crontab or launchd:
```bash
*/30 * * * * cd /Users/blake/code/weatherdesktop && ./wd >> /tmp/wd.log 2>&1
```

The Docker container stays running between executions for fast startup.

## Project Structure

```
weatherdesktop/
├── cmd/
│   ├── wd/          # Host orchestrator (Docker + desktop)
│   └── wd-worker/    # Container worker (scrape/render)
├── pkg/
│   ├── assets/       # Asset configuration
│   ├── downloader/   # HTTP downloads
│   ├── playwright/   # WebKit scraping
│   ├── image/        # Image processing
│   ├── desktop/      # macOS wallpaper (CGO)
│   └── docker/        # Docker orchestration
├── assets/           # Downloaded/scraped images
├── rendered/         # Final composites
├── Dockerfile        # Container definition
└── compose.yaml      # Docker Compose config
```

## See Also

- [IMPLEMENTATION.md](IMPLEMENTATION.md) - Technical details and architecture
