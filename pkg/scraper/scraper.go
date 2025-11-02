package scraper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"strings"
	"time"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
	"github.com/trodemaster/weatherdesktop/pkg/webdriver"
)

// Scraper handles web scraping via Safari WebDriver
type Scraper struct {
	client         *webdriver.Client
	sessionManager *webdriver.SessionManager
	session        *webdriver.Session
	manager        *assets.Manager
	debug          bool
	saveFullPage   bool
	waitOverride   int
}

// New creates a new scraper
func New(manager *assets.Manager, webdriverURL string) *Scraper {
	client := webdriver.NewClient(webdriverURL)
	sessionManager := webdriver.NewSessionManager(client)
	
	return &Scraper{
		client:         client,
		sessionManager: sessionManager,
		manager:        manager,
	}
}

// Start creates a Safari WebDriver session (headless by default)
func (s *Scraper) Start() error {
	return s.StartWithDebug(false)
}

// StartWithDebug creates a Safari WebDriver session with optional debug mode
func (s *Scraper) StartWithDebug(debug bool) error {
	session, err := s.sessionManager.CreateSession(debug)
	if err != nil {
		return fmt.Errorf("failed to create Safari session: %w", err)
	}
	s.session = session
	
	// Set a large window size to ensure full content is visible
	// This prevents element screenshots from being cropped
	if err := s.client.SetWindowRect(session.ID, 0, 0, 1920, 1600); err != nil {
		log.Printf("Warning: Failed to set window size: %v", err)
		// Don't fail the session, just log the warning
	} else if debug {
		log.Printf("Window resized to 1920x1600 for full content capture")
	}
	
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
	return nil
}

// Stop destroys the Safari WebDriver session
func (s *Scraper) Stop() error {
	if s.session != nil {
		if s.debug {
			log.Printf("🛑 Closing Safari WebDriver session: %s", s.session.ID)
		}
		return s.sessionManager.DeleteSession(s.session.ID)
	}
	return nil
}

// SetDebugOptions configures debug options for the scraper
func (s *Scraper) SetDebugOptions(debug, saveFullPage bool, waitOverride int) {
	s.debug = debug
	s.saveFullPage = saveFullPage
	s.waitOverride = waitOverride
}

// ScrapeAll scrapes all configured targets
func (s *Scraper) ScrapeAll() error {
	// Ensure session is started
	if s.session == nil {
		if err := s.Start(); err != nil {
			return err
		}
		defer s.Stop()
	}
	
	targets := s.manager.GetScrapeTargets()
	
	for _, target := range targets {
		if s.debug {
			log.Printf("\n🌐 Scraping: %s", target.Name)
		} else {
			log.Printf("Scraping %s", target.Name)
		}
		
		if err := s.scrapeTarget(target); err != nil {
			log.Printf("❌ Failed to scrape %s: %v, creating fallback image", target.Name, err)
			if err := s.createFallbackImage(target.OutputPath); err != nil {
				log.Printf("Failed to create fallback for %s: %v", target.Name, err)
			}
		}
		
		// Small delay between scrapes
		time.Sleep(500 * time.Millisecond)
	}
	
	// Also scrape WSDOT HTML
	htmlTarget := s.manager.GetWSDOTHTMLTarget()
	if s.debug {
		log.Printf("\n📄 Extracting HTML from %s", htmlTarget.Name)
	} else {
		log.Printf("Extracting HTML from %s", htmlTarget.Name)
	}
	if err := s.scrapeHTML(htmlTarget); err != nil {
		log.Printf("❌ Failed to extract HTML from %s: %v", htmlTarget.Name, err)
		// Create empty HTML file
		if err := os.WriteFile(htmlTarget.OutputPath, []byte("<div></div>"), 0644); err != nil {
			log.Printf("Failed to create fallback HTML: %v", err)
		}
	}
	
	return nil
}

// ScrapeFiltered scrapes only targets matching the filter string
func (s *Scraper) ScrapeFiltered(filter string) error {
	// Ensure session is started
	if s.session == nil {
		if err := s.Start(); err != nil {
			return err
		}
		defer s.Stop()
	}
	
	targets := s.manager.GetScrapeTargets()
	filter = strings.ToLower(filter)
	
	var matched []assets.ScrapeTarget
	for _, target := range targets {
		if strings.Contains(strings.ToLower(target.Name), filter) {
			matched = append(matched, target)
		}
	}
	
	if len(matched) == 0 {
		return fmt.Errorf("no targets match filter: %s", filter)
	}
	
	if s.debug {
		log.Printf("📋 Found %d target(s) matching '%s':", len(matched), filter)
		for _, t := range matched {
			log.Printf("   - %s", t.Name)
		}
		log.Println()
	}
	
	for _, target := range matched {
		if s.debug {
			log.Printf("\n🌐 Scraping: %s", target.Name)
		} else {
			log.Printf("Scraping %s", target.Name)
		}
		
		if err := s.scrapeTarget(target); err != nil {
			log.Printf("❌ Failed to scrape %s: %v, creating fallback image", target.Name, err)
			if err := s.createFallbackImage(target.OutputPath); err != nil {
				log.Printf("Failed to create fallback for %s: %v", target.Name, err)
			}
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return nil
}

// scrapeTarget scrapes a single target and saves a screenshot
func (s *Scraper) scrapeTarget(target assets.ScrapeTarget) error {
	if s.debug {
		log.Printf("   URL: %s", target.URL)
		log.Printf("   Selector: %s", target.Selector)
	}
	
	// Navigate to URL
	startNav := time.Now()
	if s.debug {
		log.Printf("⏳ Navigating to URL...")
	}
	if err := s.session.NavigateTo(target.URL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	if s.debug {
		log.Printf("✓ Navigation complete (%.2fs)", time.Since(startNav).Seconds())
	}
	
	// Determine wait time
	waitTime := target.WaitTime
	if s.waitOverride > 0 {
		waitTime = s.waitOverride
	}
	
	// Smart wait for element (or use fixed wait as fallback)
	if s.debug {
		if s.waitOverride > 0 {
			log.Printf("⏰ Using override wait time: %dms", waitTime)
		} else {
			log.Printf("⏰ Using smart wait with timeout: %dms", waitTime)
		}
	}
	
	elementFound, err := s.waitForElement(target.Selector, waitTime)
	if err != nil {
		if s.debug {
			log.Printf("⚠️  Element detection error: %v, proceeding anyway", err)
		}
	} else if elementFound {
		if s.debug {
			log.Printf("✓ Element found: %s", target.Selector)
		}
	} else {
		if s.debug {
			log.Printf("⚠️  Element not found after %dms, proceeding with screenshot", waitTime)
		}
	}
	
	// Take screenshot of the specific element
	screenshot, err := s.takeElementScreenshot(target.Selector)
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
	
	// Save full page screenshot if requested
	if s.saveFullPage {
		fullPagePath := strings.Replace(outputPath, ".png", "-FULLPAGE.png", 1)
		if err := os.WriteFile(fullPagePath, screenshot, 0644); err != nil {
			log.Printf("⚠️  Failed to save full page screenshot: %v", err)
		} else if s.debug {
			log.Printf("✓ Saved full page screenshot: %s", fullPagePath)
		}
	}
	
	// Save screenshot
	if err := os.WriteFile(outputPath, screenshot, 0644); err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}
	
	if s.debug {
		log.Printf("✓ Saved to: %s", outputPath)
	} else {
		log.Printf("Saved screenshot to %s", outputPath)
	}
	return nil
}

// scrapeHTML extracts HTML from a page element
func (s *Scraper) scrapeHTML(target assets.ScrapeTarget) error {
	// Navigate to URL
	if err := s.session.NavigateTo(target.URL); err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}
	
	// Wait for page load
	time.Sleep(time.Duration(target.WaitTime) * time.Millisecond)
	
	// Get page source
	source, err := s.session.GetPageSource()
	if err != nil {
		return fmt.Errorf("failed to get page source: %w", err)
	}
	
	// For simplicity, save the entire page source
	// In a full implementation, we'd parse and extract just the selector
	if err := os.WriteFile(target.OutputPath, []byte(source), 0644); err != nil {
		return fmt.Errorf("failed to save HTML: %w", err)
	}
	
	log.Printf("Saved HTML to %s", target.OutputPath)
	return nil
}

// takeElementScreenshot takes a screenshot of a specific element
func (s *Scraper) takeElementScreenshot(selector string) ([]byte, error) {
	// Find the element
	elementID, err := s.client.FindElement(s.session.ID, selector)
	if err != nil {
		return nil, fmt.Errorf("failed to find element: %w", err)
	}
	
	// Take screenshot of the element
	base64Data, err := s.client.GetElementScreenshot(s.session.ID, elementID)
	if err != nil {
		return nil, fmt.Errorf("element screenshot request failed: %w", err)
	}
	
	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot data: %w", err)
	}
	
	return imageData, nil
}

// takeScreenshot takes a full page screenshot (kept for compatibility)
func (s *Scraper) takeScreenshot() ([]byte, error) {
	// Use the WebDriver screenshot endpoint
	base64Data, err := s.client.GetScreenshot(s.session.ID)
	if err != nil {
		return nil, fmt.Errorf("screenshot request failed: %w", err)
	}
	
	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot data: %w", err)
	}
	
	return imageData, nil
}

// createFallbackImage creates an empty placeholder image
func (s *Scraper) createFallbackImage(destPath string) error {
	// Create a small transparent image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{0, 0, 0, 0})
	
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create fallback file: %w", err)
	}
	defer out.Close()
	
	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("failed to encode fallback image: %w", err)
	}
	
	log.Printf("Created fallback image at %s", destPath)
	return nil
}

// waitForElement waits for an element to appear using smart polling
// Returns true if element found, false if timeout reached
func (s *Scraper) waitForElement(selector string, timeoutMS int) (bool, error) {
	// JavaScript to check if element exists
	script := fmt.Sprintf(`
		var element = document.querySelector('%s');
		return element !== null;
	`, strings.ReplaceAll(selector, "'", "\\'"))
	
	pollInterval := 100 * time.Millisecond
	timeout := time.Duration(timeoutMS) * time.Millisecond
	deadline := time.Now().Add(timeout)
	
	attempts := 0
	for time.Now().Before(deadline) {
		attempts++
		
		// Execute JavaScript to check for element
		result, err := s.client.ExecuteScript(s.session.ID, script, []interface{}{})
		if err != nil {
			// If JavaScript fails, fall back to fixed wait
			if s.debug {
				log.Printf("   JavaScript execution failed (attempt %d): %v", attempts, err)
			}
			time.Sleep(timeout)
			return false, err
		}
		
		// Parse result
		var found bool
		if err := json.Unmarshal(result, &found); err != nil {
			if s.debug {
				log.Printf("   Failed to parse element check result: %v", err)
			}
			time.Sleep(timeout)
			return false, err
		}
		
		if found {
			if s.debug {
				log.Printf("✓ Element found after %dms (%d attempts)", time.Since(deadline.Add(-timeout)).Milliseconds(), attempts)
			}
			return true, nil
		}
		
		// Wait before next poll
		time.Sleep(pollInterval)
	}
	
	// Timeout reached
	if s.debug {
		log.Printf("   Element not detected after %dms (%d attempts)", timeoutMS, attempts)
	}
	return false, nil
}

// Helper to clean selector (remove CSS selector complexity for JS)
func selectorToJS(selector string) string {
	// Convert CSS selector to something usable in JavaScript
	// This is simplified - a full implementation would handle all CSS selector syntax
	selector = strings.TrimSpace(selector)
	if strings.HasPrefix(selector, "#") {
		// ID selector
		return fmt.Sprintf("document.getElementById('%s')", selector[1:])
	}
	// Default to querySelector
	return fmt.Sprintf("document.querySelector('%s')", selector)
}

