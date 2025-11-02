# Safari WebDriver Configuration

## Automatic Configuration

The `wd` tool automatically configures Safari WebDriver for optimal screenshot capture:

### 1. Window Size: 1920x1200

```go
// pkg/scraper/scraper.go - StartWithDebug()
client.SetWindowRect(session.ID, 0, 0, 1920, 1200)
```

**Why?**
- Default Safari window is too small (~800x600)
- Elements extending beyond viewport get cropped in screenshots
- 1920x1200 ensures full content capture for most pages

**Result:**
- Weather.gov meteogram: 60KB → 211KB (complete capture)
- All weather elements visible: temp, wind, precip, humidity

### 2. Inspector Disabled

```go
// pkg/webdriver/session.go - CreateSession()
capabilities["safari:automaticInspection"] = false
```

**Why?**
- Inspector panel takes up ~400-600px of horizontal space
- Reduces available viewport for content
- Not needed for automated screenshot capture

**Result:**
- Clean viewport maximizes screenshot content
- Full 1920px width available for page rendering

**Manual Override:**
If you need the inspector during debugging:
- Safari menu → Develop → Show Web Inspector (⌥⌘I)
- Right-click → Inspect Element
- Won't affect screenshot quality (window is still 1920x1200)

### 3. Profiling Disabled

```go
// pkg/webdriver/session.go - CreateSession()
capabilities["safari:automaticProfiling"] = false
```

**Why?**
- Profiling adds overhead
- Not needed for web scraping
- Improves session creation speed

## Safari Capabilities Reference

These Safari-specific WebDriver capabilities are set:

| Capability | Value | Purpose |
|------------|-------|---------|
| `browserName` | `"safari"` | Specify Safari browser |
| `safari:automaticInspection` | `false` | Disable auto-opening inspector |
| `safari:automaticProfiling` | `false` | Disable profiling overhead |

### Additional Available Capabilities (Not Used)

Safari supports other capabilities that we don't currently use:

- `safari:useSimulator` - Run in iOS Simulator (not needed for desktop)
- `safari:platformVersion` - Specify iOS version (iOS only)
- `safari:diagnose` - Enable diagnostic logging (not needed)

## Debug vs Production Modes

### Production Mode (Headless)
```bash
./wd
```
- Safari runs invisible (headless)
- Full 1920x1200 window size
- No inspector
- Optimal for scheduled runs

### Debug Mode (Visible)
```bash
./wd -debug
```
- Safari window visible
- Full 1920x1200 window size
- No inspector (can manually open)
- Watch scraping in real-time

**Both modes use identical configuration for consistent results.**

## Troubleshooting

### Window Size Not Applied

**Symptom:** Screenshots still cropped

**Check:**
```bash
./wd -s -scrape-target "Target" -debug -keep-browser
```

Look for log message:
```
Window resized to 1920x1200 for full content capture
```

If missing, check for warning:
```
Warning: Failed to set window size: <error>
```

**Solution:** Safari WebDriver may not support window resizing. Fallback: maximize window manually.

### Inspector Still Appears

**Symptom:** Inspector panel visible in debug mode

**Verify capability:**
```bash
# Check session.go has:
"safari:automaticInspection": false
```

**Note:** If you manually opened the inspector in a previous session, Safari may remember this preference. Close inspector and restart Safari Driver.

### Element Still Cropped

**Possible causes:**
1. Element is larger than 1920x1200 (rare)
2. Element loads after screenshot (increase wait time)
3. Element is hidden/collapsed by default

**Debug:**
```bash
./wd -s -scrape-target "Target" -wait 5000 -debug -keep-browser
```

Use `browser_evaluate` to check element dimensions:
```javascript
() => {
  const el = document.querySelector('selector');
  return {
    width: el.offsetWidth,
    height: el.offsetHeight,
    visible: el.offsetParent !== null
  };
}
```

## Performance Impact

### Window Resize
- **Cost:** ~50-100ms per session
- **Benefit:** Complete screenshot capture
- **Worth it:** Yes, prevents re-runs for missing content

### Disabled Inspector
- **Cost:** 0ms (actually saves time)
- **Benefit:** Clean viewport, no layout shifts
- **Worth it:** Absolutely

### Disabled Profiling
- **Cost:** 0ms (saves CPU cycles)
- **Benefit:** Faster session creation
- **Worth it:** Yes for production scraping

## Summary

✅ **1920x1200 window** - Full content capture  
✅ **No inspector** - Maximizes viewport space  
✅ **No profiling** - Better performance  
✅ **Consistent** - Same config for debug and production  
✅ **Manual override** - Can open inspector if needed  

These settings ensure reliable, complete screenshot capture without manual intervention.

