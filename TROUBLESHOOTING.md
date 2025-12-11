# WSDOT Pass Closure Logic - Troubleshooting & Fix Summary

## Problem
The WSDOT pass closure detection was not working because the CSS selector used to extract the pass status HTML was outdated.

**Symptom:** Stevens Pass showed as "closed in both directions" on the website but the system couldn't properly detect and display this status.

## Root Cause Analysis

The old CSS selector in `pkg/assets/manager.go` was:
```
#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1
```

This selector relied on:
1. An `#index` ID that no longer exists in the DOM
2. Specific nth-child positions that changed

The WSDOT website had been redesigned/refactored, changing the HTML structure from the old static layout to a Vue.js-based dynamic layout.

## Solution Implemented

### 1. Updated CSS Selector
**File:** `pkg/assets/manager.go`
```go
// OLD
Selector: "#index > div:nth-child(7) > div.full-width.column-container.mountain-pass > div.column-1"

// NEW - Uses simpler class-based selector
Selector: ".full-width.column-container.mountain-pass .column-1"
```

### 2. Improved HTML Scraping Function
**File:** `pkg/playwright/scraper.go`

Enhanced the `ScrapeHTML()` function to:
- Add proper navigation timeout (10 seconds)
- Use `domcontentloaded` wait strategy (faster, good for dynamic content)
- Add element visibility wait state
- Include debug logging for troubleshooting
- Better error handling

### 3. Increased Wait Time
**File:** `pkg/assets/manager.go`
```go
// Increased from 1000ms to 5000ms to allow Vue.js to render
WaitTime: 5000
```

### 4. Created Docker Build Optimization
**File:** `.dockerignore`

Added `.dockerignore` to exclude large directories from Docker build context:
- `rendered/` directory (16+ GB of historical images)
- `assets/` directory (temporary files)
- Build artifacts and documentation

This reduced build context from 16+ GB to 1.4 MB, fixing disk space issues.

## Parser Behavior

The parser (`pkg/parser/parser.go`) was already correct and properly handles:

### Closed Pass Detection
Looks for `conditionLabel` containing "Travel eastbound/westbound" with `conditionValue` containing "Closed"
```
Status: {
  East: "Pass Closed",
  West: "Pass Closed", 
  IsClosed: true,
  Conditions: "US 2 is closed from milepost..."
}
```

### Open Pass Detection
When status is not "Closed" (e.g., "Traction Tires Required"):
```
Status: {
  East: "Traction Tires Required, Chains required...",
  West: "Traction Tires Required, Chains required...",
  IsClosed: false,
  Conditions: ""
}
```

## Test Files Added

Created comprehensive test files in `testfiles/` directory with real HTML from WSDOT:

1. **`closed_wsdot_stevens_pass_2025_12_10_rain.html`**
   - Current status (Dec 10, 2025)
   - Closed due to rocks, trees, mud from rain
   - Both directions closed

2. **`closed_wsdot_stevens_pass.html`**
   - Historical status (Jan 9, 2024)
   - Closed due to high winds, poor visibility, heavy snow
   - Both directions closed

3. **`open_wsdot_stevens_pass_2024_01_10.html`**
   - Historical status (Jan 10, 2024)
   - Open with traction tires and chains required
   - Compact snow and ice conditions

## Test Results

All tests pass successfully:
```
=== RUN   TestParseWSDOTPassStatus_Closed_Rain
    ✓ Closed pass detected correctly
    ✓ East: Pass Closed
    ✓ West: Pass Closed
    ✓ IsClosed: true

=== RUN   TestParseWSDOTPassStatus_Closed_Snow
    ✓ Closed pass (snow) detected correctly
    ✓ East: Pass Closed
    ✓ West: Pass Closed

=== RUN   TestParseWSDOTPassStatus_Open
    ✓ Open pass detected correctly
    ✓ East: Traction Tires Required, Chains required...
    ✓ IsClosed: false

PASS
```

## HTML Structure Reference

The current WSDOT page uses Vue.js with this structure:
```html
<div class="full-width column-container mountain-pass">
  <div class="column-1">
    <h2>Pass report</h2>
    <div class="condition">
      <div class="conditionLabel">Travel eastbound</div>
      <div class="conditionValue">Pass Closed</div>
    </div>
    <div class="condition">
      <div class="conditionLabel">Travel westbound</div>
      <div class="conditionValue">Pass Closed</div>
    </div>
    <div class="condition">
      <div class="conditionLabel">Conditions</div>
      <div class="conditionValue">US 2 is closed from...</div>
    </div>
    <!-- Additional conditions... -->
  </div>
</div>
```

## Files Modified

1. `pkg/assets/manager.go` - Updated selector and wait time
2. `pkg/playwright/scraper.go` - Enhanced HTML scraping function
3. `IMPLEMENTATION.md` - Updated documentation with new selector
4. `.dockerignore` - Created to optimize Docker builds
5. `pkg/parser/parser_test.go` - Created comprehensive tests

## Verification

To test the fix:
```bash
cd /Users/blake/Developer/weatherdesktop

# Run tests
go test -v ./pkg/parser

# Full rebuild and test
make rebuild

# Run with debug
./wd -s -debug
```

The pass status detection now correctly identifies when Stevens Pass is closed and properly displays this on the composite image.

