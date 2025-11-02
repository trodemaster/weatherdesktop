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
	page, err := s.browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()
	
	if _, err := page.Goto(target.URL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	
	// Wait for element
	locator := page.Locator(target.Selector)
	if err := locator.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(float64(target.WaitTime)),
	}); err != nil {
		log.Printf("Warning: Element not found: %v", err)
	}
	
	// Get inner HTML
	html, err := locator.InnerHTML()
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}
	
	// Save HTML
	if err := os.WriteFile(target.OutputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to save HTML: %w", err)
	}
	
	log.Printf("Saved HTML to %s", target.OutputPath)
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

