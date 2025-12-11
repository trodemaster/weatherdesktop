package playwright

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/trodemaster/weatherdesktop/pkg/assets"
)

// Scraper handles web scraping using Playwright WebKit
type Scraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	debug   bool
}

// New creates a new Playwright scraper
func New(debug bool) *Scraper {
	return &Scraper{
		debug: debug,
	}
}

// Start initializes Playwright and launches WebKit
func (s *Scraper) Start() error {
	var err error
	
	// Start Playwright
	s.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start playwright: %w", err)
	}
	
	// Launch WebKit browser (always headless in Docker - no X server)
	s.browser, err = s.pw.WebKit.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true), // Always headless in container
	})
	if err != nil {
		return fmt.Errorf("failed to launch webkit: %w", err)
	}
	
	if s.debug {
		log.Println("WebKit browser started (headless mode)")
	}
	
	return nil
}

// Stop closes the browser and Playwright
func (s *Scraper) Stop() error {
	if s.browser != nil {
		if err := s.browser.Close(); err != nil {
			log.Printf("Warning: Failed to close browser: %v", err)
		}
	}
	
	if s.pw != nil {
		if err := s.pw.Stop(); err != nil {
			log.Printf("Warning: Failed to stop playwright: %v", err)
		}
	}
	
	return nil
}

// ScrapeAll scrapes all configured targets
func (s *Scraper) ScrapeAll(mgr *assets.Manager) error {
	targets := mgr.GetScrapeTargets()
	
	for _, target := range targets {
		if s.debug {
			log.Printf("\n🌐 Scraping: %s", target.Name)
		}
		
		if err := s.scrapeTarget(target); err != nil {
			log.Printf("❌ Failed to scrape %s: %v", target.Name, err)
			// Create fallback image
			if err := s.createFallbackImage(target.OutputPath); err != nil {
				log.Printf("Warning: Failed to create fallback image: %v", err)
			}
			continue
		}
		
		if !s.debug {
			log.Printf("Saved screenshot to %s", target.OutputPath)
		}
	}
	
	return nil
}

// ScrapeFiltered scrapes only targets matching the filter
func (s *Scraper) ScrapeFiltered(mgr *assets.Manager, filter string) error {
	targets := mgr.GetScrapeTargets()
	filterLower := strings.ToLower(filter)
	
	var matched []assets.ScrapeTarget
	for _, target := range targets {
		if strings.Contains(strings.ToLower(target.Name), filterLower) {
			matched = append(matched, target)
		}
	}
	
	if len(matched) == 0 {
		return fmt.Errorf("no targets match filter: %s", filter)
	}
	
	if s.debug {
		log.Printf("🎯 Testing specific target: %s", filter)
		log.Printf("📋 Found %d target(s) matching '%s':", len(matched), filterLower)
		for _, t := range matched {
			log.Printf("   - %s", t.Name)
		}
		log.Println()
	}
	
	for _, target := range matched {
		if s.debug {
			log.Printf("\n🌐 Scraping: %s", target.Name)
		}
		
		if err := s.scrapeTarget(target); err != nil {
			return fmt.Errorf("failed to scrape %s: %w", target.Name, err)
		}
		
		if !s.debug {
			log.Printf("Saved screenshot to %s", target.OutputPath)
		}
	}
	
	return nil
}

// scrapeTarget scrapes a single target
func (s *Scraper) scrapeTarget(target assets.ScrapeTarget) error {
	if s.debug {
		log.Printf("   URL: %s", target.URL)
		log.Printf("   Selector: %s", target.Selector)
	}
	
	// Create new page
	page, err := s.browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	
	// Set up console logging in debug mode
	if s.debug {
		page.On("console", func(msg playwright.ConsoleMessage) {
			log.Printf("   [Browser Console] %s: %s", msg.Type(), msg.Text())
		})
		page.On("pageerror", func(err error) {
			log.Printf("   [Page Error] %v", err)
		})
	}
	
	// Navigate to URL
	startNav := time.Now()
	if s.debug {
		log.Printf("⏳ Navigating to URL...")
	}
	
	// Navigate with 'domcontentloaded' - fastest option, good for slow sites
	if _, err := page.Goto(target.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded, // Wait for DOMContentLoaded event
		Timeout:   playwright.Float(10000),                    // 10 second timeout
	}); err != nil {
		// Log page content on failure for debugging
		if s.debug {
			if content, contentErr := page.Content(); contentErr == nil {
				log.Printf("   [Page Content Preview] %s", content[:min(200, len(content))])
			}
		}
		return fmt.Errorf("navigation failed: %w", err)
	}
	
	if s.debug {
		log.Printf("✓ Navigation complete (%.2fs)", time.Since(startNav).Seconds())
	}
	
	// Wait for element
	waitTime := target.WaitTime
	if waitTime == 0 {
		waitTime = 1000 // Default 1 second
	}

	if s.debug {
		log.Printf("⏰ Waiting %dms for element...", waitTime)
	}

	locator := page.Locator(target.Selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(waitTime)),
		State:   playwright.WaitForSelectorStateVisible,
	}); err != nil {
		if s.debug {
			log.Printf("⚠️  Element not found after %dms, taking screenshot anyway", waitTime)
		}
	} else if s.debug {
		log.Printf("✓ Element found: %s", target.Selector)
	}

	// For NWAC sites, wait for actual content to load (charts/graphs)
	// This is more reliable than simple time-based waits
	if strings.Contains(target.URL, "nwac.us") {
		if s.debug {
			log.Printf("⏰ Waiting for NWAC content to fully render...")
		}

		// Strategy 1: Wait for canvas or SVG elements (charts are usually rendered with these)
		contentSelector := target.Selector + " canvas, " + target.Selector + " svg"
		contentLocator := page.Locator(contentSelector)

		if err := contentLocator.First().WaitFor(playwright.LocatorWaitForOptions{
			Timeout: playwright.Float(20000), // 20 second timeout for chart rendering
			State:   playwright.WaitForSelectorStateVisible,
		}); err != nil {
			if s.debug {
				log.Printf("⚠️  Chart elements (canvas/svg) not found: %v", err)
				log.Printf("⏰ Falling back to network idle wait...")
			}

			// Fallback: Wait for network idle
			if err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
				State:   playwright.LoadStateNetworkidle,
				Timeout: playwright.Float(15000),
			}); err != nil {
				if s.debug {
					log.Printf("⚠️  Network idle timeout: %v", err)
				}
			}
		} else if s.debug {
			log.Printf("✓ Chart content detected")
		}

		// Additional wait for any animations/transitions
		extraWait := 1500 // 1.5 seconds
		if s.debug {
			log.Printf("⏰ Additional %dms wait for animations...", extraWait)
		}
		time.Sleep(time.Duration(extraWait) * time.Millisecond)
	}
	
	// Take screenshot of the element
	if s.debug {
		log.Printf("📸 Taking screenshot with 10s timeout...")
	}
	screenshot, err := locator.Screenshot(playwright.LocatorScreenshotOptions{
		Timeout: playwright.Float(10000), // 10 second timeout
	})
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}
	
	if s.debug {
		log.Printf("📸 Screenshot captured: %d bytes", len(screenshot))
	}
	
	// Determine output path
	outputPath := target.OutputPath
	if s.debug {
		// Add timestamp to debug screenshots
		ext := ".png"
		base := strings.TrimSuffix(outputPath, ext)
		timestamp := time.Now().Format("20060102-1504")
		outputPath = fmt.Sprintf("%s-DEBUG-%s%s", base, timestamp, ext)
	}
	
	// Save screenshot
	if err := os.WriteFile(outputPath, screenshot, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}
	
	if s.debug {
		log.Printf("✓ Saved to: %s", outputPath)
	}
	
	return nil
}

// ScrapeHTML extracts HTML from a page element
func (s *Scraper) ScrapeHTML(target assets.ScrapeTarget) error {
	if s.debug {
		log.Printf("\n🌐 Scraping HTML: %s", target.Name)
		log.Printf("   URL: %s", target.URL)
		log.Printf("   Selector: %s", target.Selector)
	}
	
	page, err := s.browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	
	// Set up console logging in debug mode
	if s.debug {
		page.On("console", func(msg playwright.ConsoleMessage) {
			log.Printf("   [Browser Console] %s: %s", msg.Type(), msg.Text())
		})
		page.On("pageerror", func(err error) {
			log.Printf("   [Page Error] %v", err)
		})
	}
	
	// Navigate with timeout and wait strategy
	startNav := time.Now()
	if s.debug {
		log.Printf("⏳ Navigating to URL...")
	}
	
	// Use domcontentloaded like the image scraper - faster and more reliable
	if _, err := page.Goto(target.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded, // Same as image scraper
		Timeout:   playwright.Float(30000), // 30 second timeout for slow WSDOT page
	}); err != nil {
		// Log page content on failure for debugging
		if s.debug {
			if content, contentErr := page.Content(); contentErr == nil {
				log.Printf("   [Page Content Preview] %s", content[:min(200, len(content))])
			}
		}
		return fmt.Errorf("navigation failed: %w", err)
	}
	
	if s.debug {
		log.Printf("✓ Navigation complete (%.2fs)", time.Since(startNav).Seconds())
	}
	
	// Add extra wait time for Vue.js to render (WSDOT is a Vue.js app)
	additionalWait := 3000 // 3 seconds for Vue to hydrate
	if s.debug {
		log.Printf("⏰ Waiting additional %dms for Vue.js to render...", additionalWait)
	}
	time.Sleep(time.Duration(additionalWait) * time.Millisecond)
	
	// Wait for element
	waitTime := target.WaitTime
	if waitTime == 0 {
		waitTime = 1000 // Default 1 second
	}
	
	if s.debug {
		log.Printf("⏰ Waiting %dms for element...", waitTime)
	}
	
	locator := page.Locator(target.Selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(waitTime)),
		State:   playwright.WaitForSelectorStateVisible,
	}); err != nil {
		if s.debug {
			log.Printf("⚠️  Element not found after %dms, attempting extraction anyway: %v", waitTime, err)
		}
		// Don't fail immediately - try to extract anyway
	}
	
	if s.debug {
		log.Printf("✓ Proceeding with HTML extraction")
	}
	
	// Get inner HTML using evaluate (more reliable than InnerHTML for Vue.js pages)
	if s.debug {
		log.Printf("📄 Extracting HTML with evaluate method...")
	}
	
	// Use Page.Evaluate to extract HTML directly from DOM
	result, err := page.Evaluate(fmt.Sprintf(`() => {
		const el = document.querySelector('%s');
		if (!el) {
			console.log('Element not found with selector: %s');
			return null;
		}
		console.log('Element found, extracting HTML...');
		return el.innerHTML;
	}`, target.Selector, target.Selector))
	
	if err != nil {
		if s.debug {
			log.Printf("⚠️  Evaluate failed: %v", err)
		}
		return fmt.Errorf("failed to evaluate: %w", err)
	}
	
	if result == nil {
		return fmt.Errorf("selector '%s' did not match any element", target.Selector)
	}
	
	html, ok := result.(string)
	if !ok || html == "" {
		return fmt.Errorf("extracted HTML is empty or invalid type")
	}
	
	if s.debug {
		log.Printf("📄 HTML extracted: %d bytes", len(html))
		if len(html) < 500 {
			log.Printf("📄 Content preview: %s", html[:min(len(html), 200)])
		}
	}
	
	// Save HTML
	if err := os.WriteFile(target.OutputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to save HTML: %w", err)
	}
	
	if s.debug {
		log.Printf("✓ Saved to: %s", target.OutputPath)
	} else {
		log.Printf("Saved HTML to %s", target.OutputPath)
	}
	
	return nil
}

// createFallbackImage creates an empty placeholder image
func (s *Scraper) createFallbackImage(destPath string) error {
	// Create a 1x1 transparent PNG
	transparentPNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
	data, err := base64.StdEncoding.DecodeString(transparentPNG)
	if err != nil {
		return err
	}
	
	return os.WriteFile(destPath, data, 0644)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

