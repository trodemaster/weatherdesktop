# Lock File Testing Guide

## Quick Test: Production vs Test Mode

### Test 1: Normal Production Run (with lock)

```bash
./wd -r    # Just render phase for quick test
```

Expected output:
```
Lock acquired
Starting generation of hud-251102-1045.jpg
...
```

### Test 2: Try Running Again (should fail)

While the first instance is still running, in another terminal:

```bash
./wd -r
```

Expected output:
```
Failed to acquire lock: another instance is already running (PID: 12345)
Another instance may be running. Use -debug or -scrape-target for testing.
```

### Test 3: Test Mode (bypasses lock)

Run alongside production:

```bash
./wd -r -debug
```

Expected output:
```
🧪 Test mode: bypassing lock file (safe to run alongside production)
🧪 Test mode: Starting generation of hud-TEST-251102-104523.jpg
...
⚠️  Skipping desktop wallpaper setting (debug mode active)
```

### Test 4: Target-Specific Test (also bypasses lock)

```bash
./wd -s -scrape-target "Weather.gov Hourly Forecast"
```

Expected output:
```
🧪 Test mode: bypassing lock file (safe to run alongside production)
...
```

## Lock File Inspection

### Check if lock exists

```bash
ls -la $TMPDIR/wd.lock
```

### View PID in lock file

```bash
cat $TMPDIR/wd.lock
```

Shows just the process ID:
```
12345
```

### Verify process is running

```bash
ps -p $(cat $TMPDIR/wd.lock) 2>/dev/null && echo "Process running" || echo "Stale lock"
```

### Manual lock cleanup (if needed)

```bash
rm $TMPDIR/wd.lock
```

## Testing Scenarios

### Scenario 1: Normal Scheduled Run

```bash
# In crontab or launchd
*/15 * * * * /path/to/wd
```

**Behavior:**
- First run: Lock acquired, runs normally
- Second run (before first finishes): Exits immediately with error
- After first completes: Lock released, next run proceeds

### Scenario 2: Manual Testing During Production

```bash
# Terminal 1: Production run
./wd
# (takes 5 minutes, lock is held)

# Terminal 2: Test specific scraper
./wd -s -scrape-target "NWAC" -debug -keep-browser
# ✓ Runs successfully, no conflict!
```

**Behavior:**
- Test mode bypasses lock
- Uses unique filename: `hud-TEST-251102-104523.jpg`
- Safari creates separate session
- No desktop wallpaper change
- Safe to inspect results while production continues

### Scenario 3: Multiple Debug Runs

```bash
# Terminal 1
./wd -s -scrape-target "Weather.gov" -debug

# Terminal 2
./wd -s -scrape-target "NWAC" -debug

# Terminal 3
./wd -s -scrape-target "NWAC Stevens" -debug
```

**Behavior:**
- All run simultaneously
- Each gets unique screenshot timestamps
- Each gets unique output filename
- No conflicts

### Scenario 4: Crash Recovery

```bash
# Start production run
./wd &
PID=$!

# Simulate crash
kill -9 $PID

# Lock file left behind, but next run handles it:
./wd
# ✓ Detects stale lock, removes it, continues
```

**Behavior:**
- Checks if PID in lock file still exists
- If dead process, removes stale lock
- Proceeds normally

## File Outputs

### Production Mode

```
rendered/
  hud-251102-1045.jpg     # Standard format
  hud-251102-1100.jpg
  hud-251102-1115.jpg
```

### Test Mode

```
rendered/
  hud-251102-1045.jpg         # Production (no conflicts)
  hud-TEST-251102-104512.jpg  # Test run 1
  hud-TEST-251102-104615.jpg  # Test run 2
  hud-TEST-251102-104734.jpg  # Test run 3
```

Note: Test filenames include seconds for uniqueness.

### Debug Screenshots (test mode)

```
assets/
  weather_gov_hourly_forecast-DEBUG-20251102-1045.png
  nwac_stevens_pass-DEBUG-20251102-1046.png
```

## Integration with Scheduling

### launchd (recommended for macOS)

Create `~/Library/LaunchAgents/com.user.weatherdesktop.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.weatherdesktop</string>
    <key>ProgramArguments</key>
    <array>
        <string>/path/to/wd</string>
    </array>
    <key>StartInterval</key>
    <integer>900</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/wd.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/wd.err</string>
</dict>
</plist>
```

**Benefits of lock file with launchd:**
- If run takes longer than interval, next run exits cleanly
- No pileup of processes
- Clean error logging
- Manual test runs still work: `./wd -debug`

### cron (alternative)

```bash
# Run every 15 minutes
*/15 * * * * /path/to/wd >> /tmp/wd.log 2>&1
```

**Lock file prevents:**
- Multiple simultaneous runs if one takes longer than 15 minutes
- Resource conflicts
- Safari session conflicts

## Troubleshooting

### Problem: Lock file won't release

**Check process:**
```bash
ps -p $(cat $TMPDIR/wd.lock)
```

**If process exists:**
- Wait for it to finish
- Or kill if needed: `kill $(cat $TMPDIR/wd.lock)`

**If process doesn't exist:**
- Just run again, stale lock will be auto-removed

### Problem: Need to force run despite lock

**Option 1: Use test mode**
```bash
./wd -debug
```

**Option 2: Remove lock**
```bash
rm $TMPDIR/wd.lock && ./wd
```

### Problem: Test runs conflicting with each other

This shouldn't happen! Test runs:
- Use timestamps with seconds precision
- Create unique filenames
- Don't interfere

If you see conflicts:
```bash
# Check timestamps
ls -la rendered/hud-TEST-*.jpg

# Should show unique filenames like:
# hud-TEST-251102-104512.jpg
# hud-TEST-251102-104513.jpg  (1 second later)
```

## Best Practices

1. **Production (scheduled)**: Always use normal mode (no flags)
   ```bash
   ./wd
   ```

2. **Testing (manual)**: Always use debug or scrape-target
   ```bash
   ./wd -debug
   ./wd -scrape-target "NWAC" -debug
   ```

3. **Monitoring**: Check lock file location
   ```bash
   echo $TMPDIR/wd.lock
   ```

4. **Cleanup**: Let the tool manage its lock automatically

5. **Debugging**: Use test mode for all manual testing
   - Prevents desktop changes
   - Avoids production conflicts
   - Creates unique outputs

## Summary

✅ **Lock file prevents** production run conflicts  
✅ **Test mode bypasses** lock for safe debugging  
✅ **Unique filenames** prevent output collisions  
✅ **Stale lock detection** auto-recovers from crashes  
✅ **No manual management** needed in normal use  
✅ **Safari handles** multiple sessions automatically  
✅ **Safe concurrent testing** during production runs  

