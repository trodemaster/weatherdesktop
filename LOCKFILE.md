# Lock File Documentation

## Overview

The `wd` tool uses a lock file to prevent multiple instances from running simultaneously in production mode. This prevents:
- Asset file conflicts (same filenames)
- Render output collisions
- Desktop wallpaper race conditions
- Resource contention

## Lock File Location

The lock file is automatically placed in the system temp directory:
```
$TMPDIR/wd.lock
```

On macOS, this typically resolves to:
```
/var/folders/xx/xxxxx/T/wd.lock
```

## Behavior

### Production Mode (Lock File Active)

Normal runs use the lock file:
```bash
./wd                    # Lock acquired, exits if already running
./wd -s -d -c -r -p    # Lock acquired
make run               # Lock acquired
```

Output:
```
Lock acquired
Starting generation of hud-251102-1045.jpg
...
```

If another instance is running:
```
Failed to acquire lock: another instance is already running (PID: 12345)
Another instance may be running. Use -debug or -scrape-target for testing.
```

### Test Mode (Lock File Bypassed)

Test/debug runs bypass the lock file and are **safe to run alongside production**:

```bash
./wd -debug                              # No lock, safe to run anytime
./wd -s -scrape-target "NWAC" -debug     # No lock, safe to run anytime
./wd -scrape-target "Weather.gov"        # No lock (even without -debug)
make run-debug                           # No lock
```

Output:
```
🧪 Test mode: bypassing lock file (safe to run alongside production)
🧪 Test mode: Starting generation of hud-TEST-251102-104523.jpg
...
```

### Test Mode File Naming

Test mode uses unique filenames to avoid conflicts:

**Screenshots (debug mode):**
```
assets/weather_gov_hourly_forecast-DEBUG-20251102-1045.png
```

**Rendered output (test mode):**
```
rendered/hud-TEST-251102-104523.jpg  # Includes seconds for uniqueness
```

**Production output:**
```
rendered/hud-251102-1045.jpg         # Standard format
```

## Lock File Management

### Automatic Cleanup

The lock file is automatically removed when:
- Program exits normally
- Deferred cleanup executes

### Stale Lock Detection

If the program crashes or is killed, the lock file may remain. The next run will:
1. Check if PID in lock file is still running
2. If process is dead, remove stale lock and continue
3. If process is alive, exit with error

### Manual Lock Removal

If you need to manually remove a stale lock:
```bash
# Find the lock file
ls $TMPDIR/wd.lock

# Remove it
rm $TMPDIR/wd.lock
```

## Use Cases

### Scheduled Production Runs (cron/launchd)

Production runs will never conflict:
```bash
# In crontab or launchd plist
*/15 * * * * /path/to/wd    # Runs every 15 minutes, skipped if still running
```

### Manual Testing While Production Runs

Debug/test runs are safe alongside production:
```bash
# Terminal 1: Production run (takes 5 minutes)
./wd

# Terminal 2: Test scraper (safe, no conflict)
./wd -s -scrape-target "NWAC" -debug -keep-browser
```

### Multiple Test Runs

Multiple test runs can run simultaneously:
```bash
# Terminal 1: Test Weather.gov
./wd -s -scrape-target "Weather.gov" -debug

# Terminal 2: Test NWAC (safe, no conflict)
./wd -s -scrape-target "NWAC" -debug

# Terminal 3: Test with different wait time (safe)
./wd -s -scrape-target "NWAC Stevens" -wait 10000 -debug
```

Each gets unique screenshot filenames with timestamps.

## Implementation Details

### Lock File Contents

The lock file contains just the PID:
```
12345
```

### Process Detection

Uses Unix `kill(pid, 0)` to check if process exists without actually sending a signal.

### Error Handling

- Lock acquisition failure → Fatal error (exits)
- Lock removal failure → Logged warning (not fatal)
- Stale lock → Automatically cleaned

## FAQ

### Q: Can I force multiple production runs?

No built-in force flag. Use test mode instead:
```bash
./wd -debug    # Bypasses lock
```

Or manually remove lock:
```bash
rm $TMPDIR/wd.lock && ./wd
```

### Q: What if I kill the process?

Stale locks are detected and automatically removed on next run.

### Q: Do test runs share Safari sessions?

No. Each run creates its own Safari WebDriver session with unique session ID.

### Q: Can test and production runs interfere?

Minimal interference:
- Different output filenames (TEST prefix, extra precision)
- Safari handles multiple sessions
- Lock file prevents production-to-production conflicts

### Q: How do I see the lock file location?

```bash
echo $TMPDIR
ls -la $TMPDIR/wd.lock
```

### Q: Does `-list-targets` use a lock?

No. It exits before lock acquisition.

## Best Practices

1. **Production (scheduled)**: Let lock file handle conflicts
2. **Testing/debugging**: Use `-debug` or `-scrape-target` flags
3. **Manual runs**: Just run `./wd` - it will exit cleanly if blocked
4. **Monitoring**: Check logs for "Lock acquired" vs "Test mode" messages

