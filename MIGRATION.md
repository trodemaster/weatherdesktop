# Migration: Safari WebDriver → Playwright + Docker

This document explains the architectural migration from Safari WebDriver to Playwright-Go with Docker containerization.

## Summary

**Before:** Single `wd` binary using Safari WebDriver for scraping  
**After:** Hybrid architecture with `wd` host orchestrator + `wd-worker` container (Playwright/WebKit)

## Motivation

### Why Move Away from Safari WebDriver?

1. **No True Headless Mode**: Safari WebDriver doesn't support headless operation
   - Workaround (window minimization) was unreliable
   - Safari window still visible in Dock during runs
   
2. **System-Level Dependency**: Required `safaridriver` running on host
   - Manual startup: `safaridriver --port=4444 &`
   - Process management complexity
   - Conflicts when multiple instances run
   
3. **Lock File Complexity**: Needed custom lock file to prevent concurrent runs
   - PID-based locking
   - Stale lock detection
   - Special bypass for debug mode
   
4. **Fragile Browser State**: Safari session could get into bad states
   - Inspector panel appearing unexpectedly
   - Window size not persisting
   - Session pairing conflicts

### Why Playwright + Docker?

1. **True Headless**: Playwright WebKit supports genuine headless mode
   - No visible windows unless debug mode
   - No Dock icons
   
2. **Isolated Environment**: Docker container provides:
   - Self-contained browser binaries
   - Consistent dependencies
   - No host pollution
   
3. **Simpler Process Management**: Docker Compose handles:
   - Container lifecycle
   - Process supervision (via `init: true`)
   - No manual daemon startup
   
4. **Better Concurrency**: Docker allows:
   - Multiple containers if needed
   - No lock file needed (container provides isolation)
   - Safer parallel execution

## Architectural Changes

### Old Architecture (Safari WebDriver)

```
┌─────────────────────────────────────────┐
│ macOS Host                              │
│                                         │
│  ┌────────────────────────────────┐    │
│  │ safaridriver --port=4444       │    │
│  │ (manual start, background)     │    │
│  └────────────────────────────────┘    │
│                 ▲                       │
│                 │                       │
│  ┌──────────────┴─────────────────┐    │
│  │ wd binary (single executable)  │    │
│  │ - WebDriver HTTP client         │    │
│  │ - Safari browser automation     │    │
│  │ - Image downloads               │    │
│  │ - Image processing              │    │
│  │ - Composite rendering           │    │
│  │ - Desktop wallpaper (CGO)       │    │
│  │ - Lock file management          │    │
│  └─────────────────────────────────┘    │
│                                         │
│  Lock file: $TMPDIR/wd.lock             │
└─────────────────────────────────────────┘
```

### New Architecture (Playwright + Docker)

```
┌──────────────────────────────────────────────────┐
│ macOS Host                                       │
│                                                  │
│  ┌────────────────────────────────────────────┐ │
│  │ wd binary (orchestrator)                   │ │
│  │ - Docker Compose client                    │ │
│  │ - Container lifecycle management           │ │
│  │ - Desktop wallpaper (CGO, host only)       │ │
│  └───────────────┬────────────────────────────┘ │
│                  │                               │
│                  ▼                               │
│  ┌──────────────────────────────────────────┐   │
│  │ Docker Compose v2                        │   │
│  │                                          │   │
│  │  ┌────────────────────────────────────┐ │   │
│  │  │ wd-worker container (persistent)   │ │   │
│  │  │                                    │ │   │
│  │  │  ┌──────────────────────────────┐ │ │   │
│  │  │  │ Playwright-Go + WebKit       │ │ │   │
│  │  │  │ - Web scraping (headless)    │ │ │   │
│  │  │  │ - Image downloads            │ │ │   │
│  │  │  │ - Image processing           │ │ │   │
│  │  │  │ - Composite rendering        │ │ │   │
│  │  │  └──────────────────────────────┘ │ │   │
│  │  │                                    │ │   │
│  │  │  Volumes:                          │ │   │
│  │  │  - ./assets ↔ /app/assets         │ │   │
│  │  │  - ./rendered ↔ /app/rendered     │ │   │
│  │  └────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

## Code Changes

### Removed Components

| Component | Path | Reason |
|-----------|------|--------|
| Safari WebDriver client | `pkg/webdriver/` | Replaced by Playwright |
| Safari-based scraper | `pkg/scraper/` | Replaced by `pkg/playwright/` |
| Lock file manager | `pkg/lockfile/` | No longer needed with Docker isolation |
| Safari configuration docs | `SAFARI_*.md` | No longer relevant |
| Lock file docs | `LOCKFILE.md` | No longer relevant |
| Debug guide (Safari) | `DEBUG_GUIDE.md` | Replaced by Docker documentation |

### New Components

| Component | Path | Purpose |
|-----------|------|---------|
| Playwright scraper | `pkg/playwright/scraper.go` | WebKit automation |
| Docker client | `pkg/docker/client.go` | Docker Compose orchestration |
| Worker binary | `cmd/wd-worker/main.go` | Container entry point |
| Dockerfile | `Dockerfile` | Container image definition |
| Docker Compose config | `compose.yaml` | Service configuration |
| Docker docs | `DOCKER_SETUP.md` | Setup and troubleshooting guide |
| Migration docs | `MIGRATION.md` | This document |

### Modified Components

#### `cmd/wd/main.go`

**Before:**
```go
// Direct execution
scrpr := scraper.New(mgr)
scrpr.Start()
scrpr.ScrapeAll()

dl := downloader.New(mgr)
dl.DownloadAll()

proc := pkgimage.NewProcessor(mgr)
proc.ProcessAll()

comp := pkgimage.NewCompositor(mgr)
comp.Render(outputPath)

setDesktopWallpaper(outputPath)
```

**After:**
```go
// Docker orchestration
dockerClient := docker.New(scriptDir)
dockerClient.EnsureRunning()

// Execute in container
dockerClient.Exec("wd-worker", "scrape")
dockerClient.Exec("wd-worker", "download")
dockerClient.Exec("wd-worker", "crop")
dockerClient.Exec("wd-worker", "render")

// Desktop setting on host (CGO)
setDesktopWallpaper(outputPath)
```

#### `pkg/assets/manager.go`

**No changes needed** - Asset configurations remain the same. Playwright uses the same selectors and URLs.

#### `Makefile`

**Added targets:**
```makefile
docker-build      # Build container image
docker-up         # Start container
docker-down       # Stop container
docker-restart    # Restart container
docker-logs       # View container logs
docker-shell      # Open shell in container
docker-ps         # List container status
```

**Modified targets:**
```makefile
build:            # Now only builds host binary
run:              # Now orchestrates Docker
run-debug:        # Debug mode with container logs
```

## Migration Checklist

### For Existing Users

- [x] Stop `safaridriver` (no longer needed)
  ```bash
  pkill safaridriver
  ```

- [ ] Install Docker Desktop
  ```bash
  # Download from docker.com or
  brew install --cask docker
  ```

- [ ] Remove old binary
  ```bash
  rm ./wd
  ```

- [ ] Build Docker image
  ```bash
  make docker-build
  ```

- [ ] Build new host binary
  ```bash
  make build
  ```

- [ ] Test new pipeline
  ```bash
  make run-debug
  ```

### For Fresh Install

Follow the updated [README.md](README.md) - no migration needed!

## Feature Comparison

| Feature | Safari WebDriver | Playwright + Docker |
|---------|------------------|---------------------|
| **Headless Mode** | ❌ Minimization only | ✅ True headless |
| **Manual Daemon Start** | ✅ Required | ❌ Auto-managed |
| **Lock File** | ✅ Required | ❌ Not needed |
| **Browser Visibility** | 🟡 Minimized in Dock | ✅ Completely hidden |
| **Concurrent Runs** | ❌ Lock prevents | 🟡 Container isolation |
| **Setup Complexity** | 🟡 Medium | 🟡 Medium (Docker required) |
| **Debugging** | ✅ Good (`-debug`) | ✅ Good (`-debug` + logs) |
| **Performance** | ✅ Fast | ✅ Fast (cached) |
| **Container Overhead** | ❌ None | 🟡 ~500MB memory |
| **Reproducibility** | 🟡 Host-dependent | ✅ Containerized |

## Backward Compatibility

### CLI Flags (Unchanged)

All existing flags work identically:
```bash
./wd -s -d -c -r -p -f     # Same behavior
./wd -debug                # Same behavior
./wd -scrape-target "NWAC" # Same behavior
```

### Removed Flags

These Safari-specific flags no longer exist:
- `-wait <ms>` - Not needed with Playwright's built-in waits
- `-keep-browser` - Not needed with Docker isolation
- `-save-full-page` - Not needed with Playwright's better screenshots

### Output (Unchanged)

- Same asset directory structure (`./assets/`)
- Same rendered filenames (`hud-YYMMDD-HHMM.jpg`)
- Same image resolution (3840x2160)
- Same composite layout

### Scheduled Runs (Unchanged)

Cron/launchd configurations work the same:
```bash
*/30 * * * * cd /path/to/weatherdesktop && ./wd
```

The container stays running, so no startup overhead.

## Performance Impact

### Startup Time

**Safari WebDriver:**
- Cold: ~2-3 seconds (Safari launch)
- Warm: ~1-2 seconds (session creation)

**Playwright + Docker:**
- Cold (first build): ~5-10 minutes (one-time)
- Cold (container start): ~2-3 seconds
- Warm (container running): <1 second (exec)

**Optimization:** Keep container running between executions.

### Memory Usage

**Safari WebDriver:**
- `safaridriver`: ~50MB
- Safari.app: ~200-400MB
- `wd` binary: ~20MB
- **Total: ~270-470MB**

**Playwright + Docker:**
- Docker Desktop: ~400MB (baseline)
- `wd-worker` container: ~500MB-1GB
- `wd` binary: ~20MB
- **Total: ~920MB-1.4GB**

**Trade-off:** ~500MB more memory for isolation and reliability.

## Troubleshooting Migration

### "Cannot connect to Docker daemon"

**Cause:** Docker Desktop not running

**Solution:**
```bash
open -a Docker
# Wait for Docker to start, then retry
```

### "Playwright failed to launch"

**Cause:** Incomplete Docker build

**Solution:**
```bash
docker compose down
docker compose build --no-cache
docker compose up -d
```

### "Old wd binary still using Safari"

**Cause:** Old binary in PATH

**Solution:**
```bash
which wd
rm $(which wd)
# Rebuild
make build
```

### "Assets not visible in container"

**Cause:** Volume mount not working

**Solution:**
```bash
# Check Docker Desktop settings
# Settings > Resources > File Sharing
# Ensure project directory is allowed

# Or restart Docker
docker compose down
docker compose up -d
```

## Rollback Plan

If you need to revert to Safari WebDriver:

1. Check out previous commit:
   ```bash
   git log --oneline  # Find last Safari commit
   git checkout <commit-hash>
   ```

2. Rebuild old version:
   ```bash
   make build
   ```

3. Restart safaridriver:
   ```bash
   safaridriver --port=4444 &
   ```

**Note:** Consider keeping both versions in separate directories if testing.

## Future Improvements

With Docker architecture in place, we can now:

1. **Multi-browser testing**: Easy to test with Chromium or Firefox
2. **Parallel scraping**: Run multiple containers for faster execution
3. **Cloud deployment**: Container can run on any Docker host
4. **CI/CD integration**: Automated testing in GitHub Actions
5. **Version pinning**: Lock specific Playwright/WebKit versions

## Questions?

- Setup issues → See [DOCKER_SETUP.md](DOCKER_SETUP.md)
- Usage questions → See [README.md](README.md)
- Architecture details → See this document

## Timeline

- **2025-11-02**: Migration completed
  - Safari WebDriver removed
  - Playwright + Docker implemented
  - Documentation updated
  - All tests passing ✅

