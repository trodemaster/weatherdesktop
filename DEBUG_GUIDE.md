# Weather Desktop Debug Guide

## Troubleshooting Scraping Issues

This guide helps you debug problems with web scraping when selectors don't work or pages load too slowly.

## Quick Start

```bash
# Start Safari WebDriver
safaridriver --port=4444 &

# Test a specific problematic scraper
./wd -s -scrape-target "NWAC Stevens" -debug
```

## Debug Flags

### `-debug`
Shows Safari browser window and enables verbose logging.

**Safety feature:** Automatically skips desktop wallpaper setting to prevent changing your wallpaper during debugging.

**Output includes:**
- 🌐 Target name
- URL being scraped
- CSS selector being used
- ⏳ Navigation timing
- ⏰ Wait strategy (smart vs. fixed)
- ✓ Element detection status
- 📸 Screenshot size
- ✓ File save location

**Example:**
```bash
./wd -s -debug
```

### `-scrape-target <name>`
Test only scrapers matching the name (case-insensitive, partial match).

**Example:**
```bash
# Test all NWAC scrapers
./wd -s -scrape-target "NWAC" -debug

# Test specific one
./wd -s -scrape-target "Weather.gov Hourly" -debug
```

**Available targets:**
- "Weather.gov Hourly Forecast"
- "Weather.gov Extended Forecast"
- "NWAC Stevens Observations"
- "NWAC Avalanche Forecast"
- "NWAC Avalanche Forecast Map"

### `-wait <milliseconds>`
Override wait time for slow-loading pages.

**Default behavior (smart wait):**
- Polls for element every 100ms
- Proceeds immediately when element found
- Times out after configured WaitTime

**With `-wait` flag:**
- Forces fixed wait time
- Useful when element detection fails
- Recommended for slow/complex pages

**Example:**
```bash
# Wait 10 seconds for slow page
./wd -s -scrape-target "NWAC Stevens" -wait 10000 -debug

# Test with 5 second override
./wd -s -wait 5000 -debug
```

### `-keep-browser`
Keeps Safari window open after scraping completes.

**Use case:**
- Manually inspect the page
- Check if element is actually there
- Verify selector in Dev Tools
- Test different selectors

**Example:**
```bash
./wd -s -scrape-target "NWAC Avalanche" -debug -keep-browser

# Then manually in Safari:
# 1. Open Developer Tools (Cmd+Opt+I)
# 2. Try the selector in Console:
#    document.querySelector('#nac-tab-resizer > div')
# 3. Adjust selector if needed
```

### `-save-full-page`
Saves both full page screenshot and element crop.

**Output files:**
- `screenshot.png` - Element only (normal)
- `screenshot-FULLPAGE.png` - Entire page

**Use case:**
- Compare full page vs. cropped
- See if element is outside viewport
- Check page layout issues

**Example:**
```bash
./wd -s -scrape-target "Weather.gov" -debug -save-full-page
```

## Common Issues

### Issue: Element Not Found

**Symptoms:**
```
⚠️  Element not found after 5000ms, proceeding with screenshot
```

**Debug steps:**
```bash
# 1. Keep browser open to inspect
./wd -s -scrape-target "NWAC" -debug -keep-browser

# 2. Check Developer Console for selector
# In Safari Dev Tools Console:
document.querySelector('#your-selector')

# 3. Try longer wait time
./wd -s -scrape-target "NWAC" -wait 10000 -debug

# 4. Save full page to see what's captured
./wd -s -scrape-target "NWAC" -debug -save-full-page
```

**Common causes:**
- Selector changed (website updated)
- Element loads via JavaScript (needs longer wait)
- Element in iframe (requires different approach)
- Element uses dynamic IDs/classes

### Issue: Page Loads Too Slowly

**Symptoms:**
```
✓ Element found after 4800ms (48 attempts)
```

**Solution:**
```bash
# Increase wait time override
./wd -s -scrape-target "Slow Page" -wait 15000 -debug
```

### Issue: Wrong Content Captured

**Debug:**
```bash
# Save full page to see entire capture
./wd -s -scrape-target "Problem" -debug -save-full-page

# Keep browser open to inspect
./wd -s -scrape-target "Problem" -debug -keep-browser
```

**Check:**
- Is selector too broad? (capturing parent element)
- Is selector too specific? (missing dynamic parts)
- Does element exist multiple times? (need more specific selector)

### Issue: Selector Changed After Website Update

**Workflow:**
```bash
# 1. Keep browser open
./wd -s -scrape-target "Updated Site" -debug -keep-browser

# 2. In Safari Dev Tools:
#    - Right-click element → Inspect
#    - Copy selector
#    - Test in Console:
document.querySelector('new-selector')

# 3. Update selector in pkg/assets/manager.go

# 4. Test new selector
./wd -s -scrape-target "Updated Site" -debug
```

## Debug Output Example

```bash
$ ./wd -s -scrape-target "NWAC Stevens" -wait 8000 -debug

🔍 Debug mode: Safari browser window will be visible
⏰ Wait time override: 8000ms
Safari WebDriver session created (DEBUG MODE - browser visible): xxx
Scraping sites...
🎯 Testing specific target: NWAC Stevens
📋 Found 1 target(s) matching 'nwac stevens':
   - NWAC Stevens Observations

🌐 Scraping: NWAC Stevens Observations
   URL: https://nwac.us/data-portal/graph/21/
   Selector: #post-146 > div
⏳ Navigating to URL...
✓ Navigation complete (1.23s)
⏰ Using override wait time: 8000ms
✓ Element found after 2400ms (24 attempts)
📸 Screenshot captured: 245681 bytes
✓ Saved to: assets/nwac_stevens_observations-DEBUG-20251102-1045.png
```

## Performance Tips

1. **Use smart wait by default** - Don't specify `-wait` unless needed
2. **Test individual targets** - Use `-scrape-target` instead of full scrape
3. **Remove `-debug` in production** - Faster without browser window
4. **Adjust WaitTime in code** - Edit `pkg/assets/manager.go` for permanent changes

## Debug Mode Safety

**Desktop wallpaper is NOT set when `-debug` is active.**

This prevents your desktop from constantly changing while debugging scrapers. When you see:
```
⚠️  Skipping desktop wallpaper setting (debug mode active)
   Remove -debug flag to set desktop wallpaper
```

This is expected behavior. To actually set the desktop wallpaper:
```bash
# Debug scraping (no wallpaper change)
./wd -s -debug

# Production run (sets wallpaper)
./wd
```

## Updating Selectors

**File:** `pkg/assets/manager.go`

**Method:** `GetScrapeTargets()`

```go
{
    Name:       "NWAC Stevens Observations",
    URL:        "https://nwac.us/data-portal/graph/21/",
    Selector:   "#post-146 > div",  // <-- Update this
    OutputPath: filepath.Join(m.AssetsDir, "nwac_stevens_observations.png"),
    WaitTime:   5000,  // <-- Or adjust default wait
},
```

## Tips for Finding Selectors

1. **Use Safari Dev Tools**
   - Right-click element → Inspect Element
   - Look for unique ID or class
   - Prefer IDs over complex CSS paths

2. **Test in Console**
   ```javascript
   // Should return exactly one element
   document.querySelector('#your-selector')
   
   // Should return array with one element
   document.querySelectorAll('#your-selector')
   ```

3. **Make selectors resilient**
   - Avoid dynamically generated IDs
   - Use stable class names or IDs
   - Keep selectors as simple as possible

4. **Handle dynamic content**
   - Increase WaitTime for AJAX-loaded content
   - Use smart wait (default) to detect when ready
   - Consider `-wait` override for very slow pages

