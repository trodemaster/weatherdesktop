# Scrape Target Selector Status

## ✅ Safari Window Configuration

**Applied automatically on session start:**

1. **Window Size:** 1920x1200 pixels
   - Ensures all page elements fit in viewport
   - Enables complete screenshot capture
   - Prevents element cropping

2. **Inspector Disabled:** `safari:automaticInspection = false`
   - Maximizes viewport space for content
   - Prevents inspector panel from taking up screen real estate
   - Can be manually opened if needed: Safari → Develop → Show Web Inspector (⌥⌘I)

3. **Profiling Disabled:** `safari:automaticProfiling = false`
   - Better performance during scraping
   - Reduces resource usage

## Verified Targets

### ✅ Weather.gov Hourly Forecast
**Status:** Working perfectly  
**Selector:** `img[src*="meteograms/Plotter.php"]`  
**Output:** 196KB, full 800x870px meteogram  
**Captures:**
- Temperature, Dewpoint, Wind Chill charts
- Wind speed (gusts and surface wind)
- Sky Cover, Precipitation Potential, Relative Humidity
- Precipitation amount bars

**Test command:**
```bash
./wd -s -scrape-target "Weather.gov Hourly" -debug
```

### ✅ Weather.gov Extended Forecast
**Status:** Working perfectly  
**Selector:** `#seven-day-forecast`  
**Output:** 528KB, complete 7-day forecast panel  
**Captures:**
- All 9 forecast periods (Today through Thursday Night)
- Weather icons with precipitation percentages
- High/Low temperatures
- Detailed text forecasts

**Test command:**
```bash
./wd -s -scrape-target "Weather.gov Extended" -debug
```

## Targets Needing Verification

### ⚠️  NWAC Stevens Observations
**Selector:** `#post-146 > div`  
**URL:** `https://nwac.us/data-portal/graph/21/`  
**Wait Time:** 5000ms  
**Notes:** Needs browser-based verification to ensure selector is current

**Test command:**
```bash
./wd -s -scrape-target "NWAC Stevens Obs" -debug
```

### ⚠️  NWAC Avalanche Forecast
**Selector:** `#nac-tab-resizer > div > div:nth-child(1) > div > div.nac-danger.nac-mb-4 > div.nac-row > div.nac-dangerToday.nac-col-lg-8.nac-mb-3 > div.nac-dangerGraphic`  
**URL:** `https://nwac.us/avalanche-forecast/#/stevens-pass`  
**Wait Time:** 1000ms (may need increase for JS loading)  
**Notes:** 
- Very complex nth-child selector (fragile to site changes)
- Should simplify using browser tools
- Wait time may be insufficient for React app

**Test command:**
```bash
./wd -s -scrape-target "NWAC Avalanche Forecast" -debug -wait 5000
```

### ⚠️  NWAC Avalanche Forecast Map
**Selector:** `#danger-map-widget`  
**URL:** `https://nwac.us`  
**Wait Time:** 1000ms  
**Notes:** Simple ID selector (good), may need longer wait for widget to load

**Test command:**
```bash
./wd -s -scrape-target "NWAC Avalanche Forecast Map" -debug -wait 3000
```

### ⚠️  WSDOT Stevens Pass Status (HTML)
**Selector:** `#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1`  
**URL:** `https://wsdot.com/travel/real-time/mountainpasses/stevens`  
**Wait Time:** 1000ms  
**Type:** HTML extraction (not screenshot)  
**Notes:** 
- Complex nth-child selector
- Used for pass status parsing
- Should verify with browser tools

## Recommendations

### Priority 1: Test NWAC Targets
Use browser tools to verify NWAC selectors and simplify them:

```bash
# Navigate to each NWAC page in the browser extension
# Use browser snapshot to find simpler, more stable selectors
# Test with current selectors first
```

### Priority 2: Increase Wait Times for Dynamic Content
NWAC sites use React/JavaScript rendering - may need longer waits:
- Stevens Observations: Keep 5000ms
- Avalanche Forecast: Increase to 5000ms
- Forecast Map: Increase to 3000ms

### Priority 3: Simplify Complex Selectors
Replace nth-child selectors with:
- ID selectors (`#element-id`)
- Class selectors (`.class-name`)
- Attribute selectors (`[data-*="value"]`)

## Testing Workflow

For each unverified target:

1. **Test current selector**
   ```bash
   ./wd -s -scrape-target "Target Name" -debug -keep-browser
   ```

2. **Inspect output**
   ```bash
   open assets/*-DEBUG-*.png
   ```

3. **If incomplete/missing, use browser tools**
   - Navigate to URL in browser extension
   - Use `browser_snapshot` to examine page structure
   - Use `browser_evaluate` to test selectors
   - Find simpler, more reliable selector

4. **Update selector in code**
   ```go
   // pkg/assets/manager.go
   Selector: "new-better-selector",
   ```

5. **Retest**
   ```bash
   make build && ./wd -s -scrape-target "Target Name" -debug
   ```

## Browser Tool Examples

### Test if element exists
```javascript
() => {
  const el = document.querySelector('selector-here');
  return {
    found: el !== null,
    visible: el?.offsetParent !== null,
    width: el?.offsetWidth,
    height: el?.offsetHeight
  };
}
```

### Find all matching elements
```javascript
() => {
  return Array.from(document.querySelectorAll('selector-pattern'))
    .map(el => ({
      tag: el.tagName,
      id: el.id,
      classes: Array.from(el.classList),
      text: el.innerText?.substring(0, 50)
    }));
}
```

### Get element's CSS path
```javascript
() => {
  function getPath(el) {
    if (el.id) return '#' + el.id;
    if (el === document.body) return 'body';
    
    const parent = el.parentElement;
    const tag = el.tagName.toLowerCase();
    const siblings = Array.from(parent.children).filter(e => e.tagName === el.tagName);
    
    if (siblings.length === 1) {
      return getPath(parent) + ' > ' + tag;
    }
    
    const index = siblings.indexOf(el) + 1;
    return getPath(parent) + ' > ' + tag + ':nth-child(' + index + ')';
  }
  
  const el = document.querySelector('selector-to-test');
  return el ? getPath(el) : 'Not found';
}
```

## Current Status Summary

- ✅ **2 of 5** screenshot targets verified and working
- ✅ **Window resizing** implemented and tested
- ⚠️  **3 targets** need verification with browser tools
- ⚠️  **2 complex selectors** should be simplified
- ⚠️  **Wait times** may need adjustment for JS-heavy sites

## Next Steps

1. Test NWAC targets with `-debug -keep-browser`
2. Use browser tools to verify/improve selectors
3. Update wait times if needed
4. Document final working selectors
5. Add comments explaining what each selector targets

