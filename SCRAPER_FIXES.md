# Scraper Target Fixes

## ✅ FIXED: Window Size Issue

**Problem:** Safari WebDriver's default window size was too small, causing element screenshots to be cropped.

**Solution:** Added window resizing to 1920x1200 when creating Safari sessions.

### Implementation

Added to `pkg/webdriver/client.go`:
- `SetWindowRect()` - Sets window position and size
- `MaximizeWindow()` - Maximizes window

Modified `pkg/scraper/scraper.go`:
- Automatically resizes window to 1920x1200 on session start
- Applies to both headless and debug modes

### Weather.gov Hourly Forecast - ✅ FIXED

**Before:**
- Screenshot: 60KB, only captured top ~400px
- Missing: Wind, precipitation, humidity charts

**After:**  
- Screenshot: 196KB (3x larger)
- Captures: Full 800x870px meteogram with all weather elements
- Selector: `img[src*="meteograms/Plotter.php"]` ✅ Works perfectly

**Solution Options:**

#### Option 1: Download Image Directly (RECOMMENDED)
Instead of screenshotting, extract the img src and download it:

```go
// In scraper, detect if selector targets an <img> element
// Extract src attribute
// Download image directly using downloader package
imgSrc := element.GetAttribute("src")
// Download imgSrc to OutputPath
```

**Benefits:**
- Gets complete image at full resolution
- Faster (no screenshot encoding/decoding)
- More reliable

**Changes needed:**
1. Add method to WebDriver client to get element attribute
2. Modify scraper to detect img elements
3. Use downloader for direct image fetch

#### Option 2: Scroll Element Into View + Resize Viewport
Scroll the element fully into view and ensure viewport is large enough:

```go
// JavaScript to scroll element and get dimensions
script := `
var el = document.querySelector('selector');
el.scrollIntoView();
return {height: el.offsetHeight, width: el.offsetWidth};
`
// Resize Safari window to accommodate element
// Take screenshot
```

**Drawbacks:**
- More complex
- Viewport resizing might cause layout shifts
- Still encoding/decoding overhead

#### Option 3: Full Page Screenshot + Crop
Take full page screenshot, then crop to element bounds:

```go
// Get element position
bounds := getElementBounds(selector)
// Take full page screenshot
fullScreenshot := client.GetScreenshot(sessionID)
// Crop image to bounds
croppedImg := cropImage(fullScreenshot, bounds)
```

**Drawbacks:**
- Very slow for long pages
- Large memory usage
- Requires image manipulation in Go

### Recommendation

**Implement Option 1** for the Weather.gov Hourly Forecast:

1. Create a new scrape target type that directly downloads images
2. Add image URL extraction capability to scraper
3. Use the existing downloader for fetching

**Temporary Workaround:**

Add the meteogram as a download target instead of scrape target:

```go
// In pkg/assets/manager.go GetDownloadAssets()
{
    Name:      "Weather.gov Hourly Forecast",
    URL:       "https://forecast.weather.gov/meteograms/Plotter.php?lat=47.7456&lon=-121.0892&wfo=SEW&zcode=WAZ302&gset=20&gdiff=3&unit=0&tinfo=PY8&ahour=0&pcmd=11011111111110000000000000000000000000000000000000000000000&lg=en&indu=1!1!1!&dd=&bw=&hrspan=48&pqpfhr=6&psnwhr=6",
    LocalPath: filepath.Join(m.AssetsDir, "weather_gov_hourly_forecast.png"),
},
```

**Note:** The URL parameters control what's displayed:
- `hrspan=48` - 48 hour forecast
- `pcmd=11011111111110000...` - which elements to show (temp, wind, precip, etc.)
- `unit=0` - imperial units

### Other Targets to Verify

#### ✅ Weather.gov Extended Forecast
- Selector: `#seven-day-forecast`
- Element: `<div>` panel, 327px tall
- Status: Works correctly (fits in viewport)

#### ⚠️  NWAC Stevens Observations
- Selector: `#post-146 > div`
- URL: `https://nwac.us/data-portal/graph/21/`
- Status: Unknown, needs browser verification
- Action: Test with browser tools

#### ⚠️  NWAC Avalanche Forecast
- Selector: Complex nth-child path
- URL: `https://nwac.us/avalanche-forecast/#/stevens-pass`
- Status: Unknown, needs simplification
- Action: Use browser tools to find simpler selector

#### ⚠️  NWAC Avalanche Forecast Map
- Selector: `#danger-map-widget`
- URL: `https://nwac.us`
- Status: Unknown, needs verification
- Action: Test with browser tools

#### ⚠️  WSDOT Stevens Pass Status
- Selector: Complex nth-child path for HTML extraction
- URL: `https://wsdot.com/travel/real-time/mountainpasses/stevens`
- Status: Unknown, needs verification
- Action: Test with browser tools

## Testing Workflow

For each scrape target:

```bash
# 1. Test current selector
./wd -s -scrape-target "Target Name" -debug -keep-browser

# 2. Inspect output
open assets/*-DEBUG-*.png

# 3. Use browser tools to find better selector
# (Navigate to URL, evaluate selectors, test)

# 4. Update selector in pkg/assets/manager.go

# 5. Retest
make build && ./wd -s -scrape-target "Target Name" -debug
```

## Priority

1. **HIGH**: Weather.gov Hourly Forecast - Currently broken (incomplete capture)
2. **MEDIUM**: NWAC targets - Complex selectors, may break with site updates
3. **LOW**: WSDOT target - Working, but selector could be simplified

## Next Steps

1. Implement Option 1 (direct image download) for Weather.gov meteogram
2. Use browser tools to verify and fix other scrape targets
3. Document all working selectors with comments explaining what they target
4. Add validation to scraper to detect incomplete element captures

