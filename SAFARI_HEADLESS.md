# Safari "Headless" Mode

## Important: Safari Doesn't Support True Headless

Unlike Chrome or Firefox, **Safari WebDriver does not support true headless mode**. The Safari browser window will always open when a WebDriver session is created.

## Solution: Window Minimization

Instead of headless mode, we minimize the Safari window in production to keep it out of sight:

### Production Mode (Default)
```bash
./wd                    # Safari opens but is immediately minimized
./wd -s -d -c -r -p    # Safari minimized
```

**Log Output:**
```
Safari WebDriver session created (minimized): <session-id>
```

**Behavior:**
- ✅ Safari window opens
- ✅ Window is immediately minimized to Dock
- ✅ Runs in background
- ✅ Not visible on screen
- ✅ No desktop interruption

### Debug Mode (Visible)
```bash
./wd -debug                              # Safari stays visible
./wd -s -scrape-target "Target" -debug  # Safari stays visible
```

**Log Output:**
```
🔍 Debug mode: Safari browser window will be visible
Window resized to 1920x1600 for full content capture
Safari WebDriver session created (DEBUG MODE - browser visible): <session-id>
```

**Behavior:**
- ✅ Safari window opens
- ✅ Window stays visible (NOT minimized)
- ✅ Resized to 1920x1600 for full content
- ✅ Can watch scraping in real-time
- ✅ Can manually open inspector if needed

## Implementation

### 1. Window Minimization (`MinimizeWindow`)

Added to `pkg/webdriver/client.go`:
```go
// MinimizeWindow minimizes the window
// Per W3C WebDriver spec: POST /session/{sessionId}/window/minimize
func (c *Client) MinimizeWindow(sessionID string) error {
	_, err := c.post(fmt.Sprintf("/session/%s/window/minimize", sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to minimize window: %w", err)
	}
	return nil
}
```

### 2. Conditional Minimization

In `pkg/scraper/scraper.go` `StartWithDebug()`:
```go
// Minimize window in production mode (Safari doesn't support true headless)
// In debug mode, keep window visible for inspection
if !debug {
	if err := s.client.MinimizeWindow(session.ID); err != nil {
		log.Printf("Warning: Failed to minimize window: %v", err)
		// Don't fail, just log - window will stay visible
	}
	log.Printf("Safari WebDriver session created (minimized): %s", session.ID)
} else {
	log.Printf("Safari WebDriver session created (DEBUG MODE - browser visible): %s", session.ID)
}
```

## When Window is Minimized vs Visible

| Mode | Safari Window | Use Case |
|------|---------------|----------|
| **Production** (`./wd`) | Minimized | Scheduled/automated runs |
| **Debug** (`./wd -debug`) | Visible | Manual testing, inspection |
| **Target Test** (`./wd -scrape-target "X"`) | Visible | Testing specific scrapers |

## Scheduled Runs (cron/launchd)

When running periodically, Safari will:
1. Open in a new window
2. Be immediately minimized to the Dock
3. Run scraping tasks
4. Close when session ends
5. **Not interrupt your work**

Example launchd configuration:
```xml
<key>ProgramArguments</key>
<array>
    <string>/path/to/wd</string>
</array>
<key>StartInterval</key>
<integer>900</integer>  <!-- Every 15 minutes -->
```

Safari will open minimized every 15 minutes, do its work, and close - all without showing on screen.

## User Experience

### Production (Minimized)
You'll see in the Dock:
- Safari icon appears briefly
- Window immediately minimizes
- Tool runs in background
- Safari closes when done
- **Zero desktop interruption**

### Debug (Visible)
You'll see on screen:
- Safari window opens at 1920x1600
- Can watch pages loading
- Can see elements being found
- Can manually inspect if needed
- Browser closes when done (unless `-keep-browser` used)

## Why Not True Headless?

Safari WebDriver limitations:
- No `headless` capability exists
- No command-line flag for headless
- Apple doesn't provide headless Safari
- Browser window always opens

This is by design - Safari is a GUI application and Apple hasn't implemented headless mode like other browsers.

## Alternative: Run on Separate Space/Desktop

If you want Safari completely out of sight, you can:

1. Create a separate desktop (Mission Control)
2. Move Safari to that desktop when it opens
3. Switch back to your main desktop
4. Safari runs on the other desktop

However, window minimization is simpler and works automatically.

## Comparison with Other Browsers

| Browser | True Headless | Method |
|---------|---------------|--------|
| Chrome | ✅ Yes | `--headless` flag |
| Firefox | ✅ Yes | `--headless` flag |
| Edge | ✅ Yes | `--headless` flag |
| Safari | ❌ No | Minimize window instead |

## Troubleshooting

### Safari Window Still Visible

**Possible causes:**
1. Debug mode is enabled (`-debug` or `-scrape-target` flag)
2. Minimize command failed (check logs for warning)
3. Safari preferences preventing minimize

**Check logs:**
```bash
./wd -s | grep "Safari WebDriver session"
```

Should show: `Safari WebDriver session created (minimized)`

### Can't See Safari in Debug Mode

Verify debug flag is set:
```bash
./wd -debug -s
```

Should show: `🔍 Debug mode: Safari browser window will be visible`

### Want to Keep Safari Visible for Longer

Use `-keep-browser` flag:
```bash
./wd -s -scrape-target "Target" -debug -keep-browser
```

Browser stays open after scraping completes.

## Summary

✅ **Production runs** - Safari minimized automatically  
✅ **Debug runs** - Safari visible for inspection  
✅ **No manual intervention** needed  
✅ **No desktop interruption** in production  
✅ **Clean separation** between testing and production  
✅ **Best possible UX** given Safari's limitations  

The implementation provides a "headless-like" experience by keeping Safari out of sight during automated runs while still allowing visibility for debugging.

