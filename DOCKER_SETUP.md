# Docker Setup Guide

This document explains the Docker-based architecture of the Weather Desktop tool.

## Architecture Overview

The Weather Desktop tool uses a **hybrid architecture**:
- **Host binary** (`wd`): Orchestrates Docker and sets the macOS desktop wallpaper
- **Container** (`wd-worker`): Handles web scraping, downloads, and image processing

This design isolates the complex browser automation (Playwright + WebKit) in a container while keeping the macOS-specific desktop setting on the host.

## Docker Compose Configuration

### compose.yaml

```yaml
services:
  wd-worker:
    build:
      context: .
      dockerfile: Dockerfile
    volumes:
      - ./assets:/app/assets      # Shared asset storage
      - ./rendered:/app/rendered  # Shared rendered images
    init: true                    # Use Docker's tini for process management
    command: sh -c "while true; do sleep 3600; done"  # Keep running
```

**Key Points:**
- **Persistent container**: Stays running to avoid startup overhead on each execution
- **Volume mounts**: Share `assets/` and `rendered/` directories with host
- **init: true**: Uses Docker's built-in Tini for proper signal handling and zombie reaping
- **Long-running command**: Keeps container alive; actual work is done via `docker compose exec`

### Dockerfile

```dockerfile
FROM golang:1.21-bookworm

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    fonts-liberation \
    && rm -rf /var/lib/apt/lists/*

# Install Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install Playwright and WebKit browser
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1 install --with-deps webkit

# Build worker binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wd-worker ./cmd/wd-worker/main.go

ENTRYPOINT ["/app/wd-worker"]
```

**Key Points:**
- **Base image**: `golang:1.21-bookworm` (Debian-based)
- **Playwright installation**: Installs WebKit browser and system dependencies
- **Static binary**: `CGO_ENABLED=0` for portability within container
- **Layer caching**: Dependencies installed before source copy for efficient rebuilds

## How Commands Flow

### Example: Full Pipeline (`./wd`)

```
┌──────────────────────────────────────────────────────┐
│ Host: ./wd                                           │
│                                                      │
│ 1. Check if container is running                    │
│    └─> docker compose ps -q wd-worker               │
│                                                      │
│ 2. Start if needed                                  │
│    └─> docker compose up -d --wait                  │
│                                                      │
│ 3. Execute scraping in container                    │
│    └─> docker compose exec wd-worker \              │
│         wd-worker scrape                             │
│        ┌──────────────────────────────┐             │
│        │ Container: Playwright WebKit │             │
│        │ - Navigate to URLs           │             │
│        │ - Wait for elements          │             │
│        │ - Take screenshots           │             │
│        │ - Save to /app/assets/       │             │
│        └──────────────────────────────┘             │
│                                                      │
│ 4. Execute downloads in container                   │
│    └─> docker compose exec wd-worker \              │
│         wd-worker download                           │
│        ┌──────────────────────────────┐             │
│        │ Container: HTTP downloads    │             │
│        │ - Concurrent downloads       │             │
│        │ - Save to /app/assets/       │             │
│        └──────────────────────────────┘             │
│                                                      │
│ 5. Execute cropping in container                    │
│    └─> docker compose exec wd-worker \              │
│         wd-worker crop                               │
│        ┌──────────────────────────────┐             │
│        │ Container: Image processing  │             │
│        │ - Crop and resize images     │             │
│        │ - Update /app/assets/        │             │
│        └──────────────────────────────┘             │
│                                                      │
│ 6. Execute rendering in container                   │
│    └─> docker compose exec wd-worker \              │
│         wd-worker render                             │
│        ┌──────────────────────────────┐             │
│        │ Container: Compositing       │             │
│        │ - Layer images onto canvas   │             │
│        │ - Save to /app/rendered/     │             │
│        └──────────────────────────────┘             │
│                                                      │
│ 7. Set desktop wallpaper (host only)               │
│    └─> CGO call to NSWorkspace API                  │
│        ┌──────────────────────────────┐             │
│        │ Host: macOS API              │             │
│        │ - Read rendered/hud-*.jpg    │             │
│        │ - Set as desktop wallpaper   │             │
│        └──────────────────────────────┘             │
└──────────────────────────────────────────────────────┘
```

## Volume Mounts

### ./assets ↔ /app/assets

This directory contains:
- Downloaded images (satellite, webcams)
- Scraped screenshots (forecasts, avalanche data)
- Processed/cropped images

**Flow:**
1. Container writes scraped/downloaded images to `/app/assets/`
2. Host reads from `./assets/` (same directory via volume mount)
3. Both host and container can access the same files

### ./rendered ↔ /app/rendered

This directory contains:
- Final composite images with timestamps (`hud-YYMMDD-HHMM.jpg`)

**Flow:**
1. Container writes composite to `/app/rendered/hud-*.jpg`
2. Host reads from `./rendered/hud-*.jpg` to set wallpaper
3. Old rendered images persist for history/debugging

## Container Lifecycle

### Starting the Container

```bash
# Manual start
docker compose up -d --wait

# Or let wd binary handle it
./wd  # Automatically starts if not running
```

**What happens:**
1. Docker builds image if not cached
2. Starts container in background (`-d`)
3. Waits for container to be healthy (`--wait`)
4. Container runs infinite sleep loop to stay alive

### Checking Status

```bash
# Via Makefile
make docker-ps

# Via Docker Compose
docker compose ps

# Expected output:
NAME        IMAGE               COMMAND                  SERVICE      STATUS
wd-worker   weatherdesktop...   "sh -c 'while true..."   wd-worker    Up 5 minutes
```

### Stopping the Container

```bash
# Via Makefile
make docker-down

# Via Docker Compose
docker compose down

# Container and volumes are removed
# Host directories (./assets, ./rendered) persist
```

### Restarting After Changes

```bash
# Rebuild image after code changes
make docker-build
docker compose build

# Restart container with new image
make docker-restart
docker compose restart wd-worker
```

## Debugging

### View Container Logs

```bash
# Follow logs in real-time
make docker-logs

# Or directly
docker compose logs -f wd-worker
```

### Execute Commands Manually

```bash
# Open shell in container
make docker-shell
docker compose exec wd-worker sh

# Inside container:
/app $ ls -la /app/assets/
/app $ wd-worker scrape --debug
/app $ exit
```

### Debug Scraping with Visible Browser

```bash
# Run from host with debug flag
./wd -s -debug -scrape-target "NWAC"

# Playwright will show WebKit browser in container
# (Requires X11 forwarding or VNC for GUI, not practical on macOS)
# Instead, check logs and screenshots:
ls -lh assets/*-DEBUG-*.png
```

### Inspect Container Filesystem

```bash
# List assets in container
docker compose exec wd-worker ls -lh /app/assets/

# Check rendered images
docker compose exec wd-worker ls -lh /app/rendered/

# View container processes
docker compose exec wd-worker ps aux
```

## Performance Considerations

### Container Startup

- **Cold start** (first build): ~5-10 minutes (downloads WebKit)
- **Warm start** (cached image): ~2-3 seconds
- **Exec commands** (container running): <1 second

**Optimization**: Keep container running between `wd` executions

### Playwright/WebKit

- **Headless mode**: Fast, no GUI overhead
- **Visible mode** (`-debug`): Slightly slower, requires X11 setup
- **Network-dependent**: Scraping speed depends on target site response

### Resource Usage

Typical resource consumption:
- **Memory**: 500MB-1GB (Playwright + WebKit)
- **CPU**: Low idle, spikes during scraping
- **Disk**: ~2GB image size (includes WebKit browser)

## Troubleshooting

### "Cannot connect to Docker daemon"

**Problem**: Docker Desktop not running

**Solution**:
```bash
# Check Docker is running
docker info

# Start Docker Desktop (macOS)
open -a Docker
```

### "Container exited with code 1"

**Problem**: Container crashed or failed to start

**Solution**:
```bash
# View logs
docker compose logs wd-worker

# Common issues:
# - Playwright installation failed -> Rebuild: docker compose build --no-cache
# - Port conflict -> Change exposed ports in compose.yaml
# - Resource limits -> Increase Docker memory (Settings > Resources)
```

### "Playwright failed to launch WebKit"

**Problem**: WebKit browser not properly installed

**Solution**:
```bash
# Rebuild with clean slate
docker compose down
docker compose build --no-cache
docker compose up -d

# Verify installation
docker compose exec wd-worker wd-worker scrape --debug
```

### "Volume mount permission denied"

**Problem**: Docker doesn't have access to host directories

**Solution**:
```bash
# Check Docker Desktop settings
# Settings > Resources > File Sharing
# Ensure /Users/blake/code/weatherdesktop is allowed

# Or use absolute paths in compose.yaml (not recommended)
```

### Stale Container State

**Problem**: Container has old code or cached data

**Solution**:
```bash
# Complete reset
docker compose down
docker volume prune -f  # Remove any named volumes (if used)
docker compose build --no-cache
docker compose up -d

# Or just restart
make docker-restart
```

## Best Practices

### Development Workflow

1. **Make code changes** to `cmd/wd-worker/` or `pkg/`
2. **Rebuild container**: `make docker-build`
3. **Restart**: `make docker-restart`
4. **Test**: `./wd -s -debug`

### Production Workflow

1. **Build once**: `make docker-build`
2. **Start container**: `docker compose up -d`
3. **Schedule runs**: Add `./wd` to cron or launchd
4. **Container stays running**: No restart needed between runs

### Maintenance

```bash
# Weekly: Check for updates
docker compose pull
go get -u ./...
make docker-build

# Monthly: Clean up unused images
docker image prune -f
docker system prune -f
```

## Advanced Configuration

### Custom Docker Compose File

Create `compose.override.yaml` for local customization:
```yaml
services:
  wd-worker:
    # Add environment variables
    environment:
      - DEBUG=true
    
    # Expose Playwright debugging port
    ports:
      - "9222:9222"
    
    # Custom resource limits
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '1.5'
```

### Multi-Platform Builds

Build for different architectures:
```bash
# Intel/AMD
docker compose build --build-arg GOARCH=amd64

# ARM (Apple Silicon)
docker compose build --build-arg GOARCH=arm64
```

## Why Docker?

### Problems Solved

1. **Playwright Dependencies**: Complex system dependencies for WebKit
2. **Browser Isolation**: Avoid polluting host with browser binaries
3. **Reproducibility**: Same environment every time, regardless of host OS
4. **Version Locking**: Pin Playwright/WebKit versions independently
5. **Testing**: Easy to test scraping without affecting host

### Trade-offs

**Pros:**
- ✅ Isolated execution environment
- ✅ Easy dependency management
- ✅ Consistent across machines
- ✅ Simple rollback (old images)

**Cons:**
- ❌ Additional Docker Desktop requirement
- ❌ Slightly slower initial setup
- ❌ More complex debugging (logs instead of direct access)
- ❌ Cannot use Docker inside container (desktop setting requires host)

## See Also

- [README.md](README.md) - General usage and installation
- [Docker Compose v2 Documentation](https://docs.docker.com/compose/)
- [Playwright-Go Documentation](https://pkg.go.dev/github.com/playwright-community/playwright-go)

