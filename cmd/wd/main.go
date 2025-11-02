package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
	"github.com/trodemaster/weatherdesktop/pkg/desktop"
	"github.com/trodemaster/weatherdesktop/pkg/downloader"
	pkgimage "github.com/trodemaster/weatherdesktop/pkg/image"
	"github.com/trodemaster/weatherdesktop/pkg/lockfile"
	"github.com/trodemaster/weatherdesktop/pkg/parser"
	"github.com/trodemaster/weatherdesktop/pkg/scraper"
)

var (
	scrapeFlag      = flag.Bool("s", false, "Scrape websites")
	downloadFlag    = flag.Bool("d", false, "Download images")
	cropFlag        = flag.Bool("c", false, "Crop/resize images")
	renderFlag      = flag.Bool("r", false, "Render composite image")
	desktopFlag     = flag.Bool("p", false, "Set desktop wallpaper")
	flushFlag       = flag.Bool("f", false, "Flush/clear assets directory")
	debugFlag       = flag.Bool("debug", false, "Show Safari browser window (debug mode)")
	scrapeTargetFlag = flag.String("scrape-target", "", "Test specific scrape target by name")
	listTargetsFlag = flag.Bool("list-targets", false, "List all available scrape targets and exit")
	waitFlag        = flag.Int("wait", 0, "Override wait time in milliseconds (0 = use smart wait)")
	keepBrowserFlag = flag.Bool("keep-browser", false, "Keep Safari session open after scraping (for inspection)")
	saveFullPageFlag = flag.Bool("save-full-page", false, "Save both full page and element screenshots")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE: wd [options]\n\n")
		fmt.Fprintf(os.Stderr, "Running wd without options will collect all assets,\n")
		fmt.Fprintf(os.Stderr, "render and set the desktop image to the output.\n\n")
		fmt.Fprintf(os.Stderr, "Individual options provided for debugging specific functions.\n\n")
		fmt.Fprintf(os.Stderr, "PHASE OPTIONS:\n")
		fmt.Fprintf(os.Stderr, "   -s                    Scrape Sites\n")
		fmt.Fprintf(os.Stderr, "   -d                    Download Images\n")
		fmt.Fprintf(os.Stderr, "   -c                    Crop Images\n")
		fmt.Fprintf(os.Stderr, "   -r                    Render Image\n")
		fmt.Fprintf(os.Stderr, "   -p                    Set Desktop\n")
		fmt.Fprintf(os.Stderr, "   -f                    Flush assets\n")
		fmt.Fprintf(os.Stderr, "\nDEBUG OPTIONS:\n")
		fmt.Fprintf(os.Stderr, "   -debug                Show Safari browser (headless by default)\n")
		fmt.Fprintf(os.Stderr, "   -list-targets         List all available scrape targets\n")
		fmt.Fprintf(os.Stderr, "   -scrape-target <name> Test specific scrape target (e.g., \"Weather.gov Hourly\")\n")
		fmt.Fprintf(os.Stderr, "   -wait <ms>            Override wait time in milliseconds (default: smart wait)\n")
		fmt.Fprintf(os.Stderr, "   -keep-browser         Keep Safari open after scraping (for inspection)\n")
		fmt.Fprintf(os.Stderr, "   -save-full-page       Save both full page and element screenshots\n")
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "   wd -s -scrape-target \"NWAC Stevens\" -wait 10000 -debug\n")
		fmt.Fprintf(os.Stderr, "   wd -s -debug -keep-browser\n")
	}
	
	flag.Parse()

	// Handle list-targets flag (special case - exits after listing)
	if *listTargetsFlag {
		listScrapeTargets()
		return
	}

	// Determine if we're in test/debug mode
	// Test mode: debug flag OR scrape-target specified
	testMode := *debugFlag || *scrapeTargetFlag != ""
	
	// Acquire lock file for normal operation (skip in test mode)
	var lock *lockfile.LockFile
	if !testMode {
		lock = lockfile.New()
		if err := lock.TryLock(); err != nil {
			log.Fatalf("Failed to acquire lock: %v\nAnother instance may be running. Use -debug or -scrape-target for testing.", err)
		}
		defer lock.Unlock()
		log.Println("Lock acquired")
	} else {
		log.Println("🧪 Test mode: bypassing lock file (safe to run alongside production)")
	}

	// Determine which phases to run
	// If no flags set, run all phases (same logic as bash script lines 82-84)
	runAll := !(*scrapeFlag || *downloadFlag || *cropFlag || *renderFlag || *desktopFlag || *flushFlag)
	
	doScrape := runAll || *scrapeFlag
	doDownload := runAll || *downloadFlag
	doCrop := runAll || *cropFlag
	doRender := runAll || *renderFlag
	doDesktop := runAll || *desktopFlag
	doFlush := runAll || *flushFlag

	// Get script directory
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("Failed to get script directory: %v", err)
	}
	
	// Generate output filename with timestamp
	// In test mode, add more precision to avoid conflicts
	var renderedFilename string
	if testMode {
		renderedFilename = fmt.Sprintf("hud-TEST-%s.jpg", time.Now().Format("060102-150405"))
		log.Printf("🧪 Test mode: Starting generation of %s", renderedFilename)
	} else {
		renderedFilename = fmt.Sprintf("hud-%s.jpg", time.Now().Format("060102-1504"))
		log.Printf("Starting generation of %s", renderedFilename)
	}

	// Phase 0: Flush assets if requested
	if doFlush {
		if err := flushAssets(scriptDir); err != nil {
			log.Printf("Warning: Failed to flush assets: %v", err)
		}
	}

	// Phase 1: Download images
	if doDownload {
		log.Println("Downloading images...")
		if err := downloadImages(scriptDir); err != nil {
			log.Printf("Warning: Some downloads failed: %v", err)
		}
	}

	// Phase 2: Scrape websites
	if doScrape {
		log.Println("Scraping sites...")
		scrapeOpts := ScrapeOptions{
			Debug:        *debugFlag,
			TargetFilter: *scrapeTargetFlag,
			WaitOverride: *waitFlag,
			KeepBrowser:  *keepBrowserFlag,
			SaveFullPage: *saveFullPageFlag,
		}
		if err := scrapeSites(scriptDir, scrapeOpts); err != nil {
			log.Printf("Warning: Some scrapes failed: %v", err)
		}
	}

	// Wait for asset collection to complete
	if doDownload || doScrape {
		log.Println("Asset Collection Completed...")
	}

	// Phase 3: Crop and resize images
	if doCrop {
		log.Println("Cropping images...")
		if err := cropImages(scriptDir); err != nil {
			log.Printf("Warning: Some crops failed: %v", err)
		}
		log.Println("Cropping completed...")
	}

	// Phase 4: Render composite image
	if doRender {
		log.Println("Rendering...")
		renderedPath := filepath.Join(scriptDir, "rendered", renderedFilename)
		if err := renderComposite(scriptDir, renderedPath); err != nil {
			log.Fatalf("Failed to render composite: %v", err)
		}
		log.Printf("Rendering %s completed...", renderedPath)
	}

	// Phase 5: Set desktop wallpaper
	if doDesktop {
		// Skip desktop setting in debug mode
		if *debugFlag {
			log.Println("⚠️  Skipping desktop wallpaper setting (debug mode active)")
			log.Println("   Remove -debug flag to set desktop wallpaper")
		} else {
			renderedPath := filepath.Join(scriptDir, "rendered", renderedFilename)
			if _, err := os.Stat(renderedPath); os.IsNotExist(err) {
				log.Fatalf("Rendered file %s not found. Run with render and set desktop at the same time", renderedPath)
			}
			
			log.Printf("Setting desktop to %s", renderedPath)
			if err := setDesktopWallpaper(renderedPath); err != nil {
				log.Fatalf("Failed to set desktop: %v", err)
			}
		}
	}

	// Optional: Copy to CDN if mounted
	cdnPath := "/Volumes/Bomb20/cdn"
	if doRender {
		if info, err := os.Stat(cdnPath); err == nil && info.IsDir() {
			renderedPath := filepath.Join(scriptDir, "rendered", renderedFilename)
			destPath := filepath.Join(cdnPath, "stevens_pass.jpg")
			log.Printf("Copying %s to %s", renderedPath, destPath)
			if err := copyFile(renderedPath, destPath); err != nil {
				log.Printf("Warning: Failed to copy to CDN: %v", err)
			}
		}
	}

	log.Println("End of Line...")
}

// flushAssets removes all files from the assets directory
func flushAssets(scriptDir string) error {
	assetsDir := filepath.Join(scriptDir, "assets")
	
	// Read directory
	files, err := os.ReadDir(assetsDir)
	if err != nil {
		return fmt.Errorf("failed to read assets directory: %w", err)
	}
	
	// Remove each file
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(assetsDir, file.Name())
			log.Printf("Removing %s", filePath)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Warning: Failed to remove %s: %v", filePath, err)
			}
		}
	}
	
	return nil
}

// downloadImages downloads all configured images
func downloadImages(scriptDir string) error {
	mgr := assets.NewManager(scriptDir)
	dl := downloader.New(mgr)
	return dl.DownloadAll()
}

// ScrapeOptions contains options for scraping
type ScrapeOptions struct {
	Debug        bool
	TargetFilter string
	WaitOverride int
	KeepBrowser  bool
	SaveFullPage bool
}

// scrapeSites scrapes all configured websites
func scrapeSites(scriptDir string, opts ScrapeOptions) error {
	mgr := assets.NewManager(scriptDir)
	
	// Create scraper with Safari WebDriver at localhost:4444
	webdriverURL := "http://localhost:4444"
	scrpr := scraper.New(mgr, webdriverURL)
	
	// Configure scraper with options
	scrpr.SetDebugOptions(opts.Debug, opts.SaveFullPage, opts.WaitOverride)
	
	// Start Safari session (headless by default, visible when debug=true)
	if opts.Debug {
		log.Println("🔍 Debug mode: Safari browser window will be visible")
		if opts.SaveFullPage {
			log.Println("📸 Full page screenshots will be saved")
		}
		if opts.WaitOverride > 0 {
			log.Printf("⏰ Wait time override: %dms", opts.WaitOverride)
		} else {
			log.Println("⏰ Using smart wait (polls for element)")
		}
	}
	
	if err := scrpr.StartWithDebug(opts.Debug); err != nil {
		return fmt.Errorf("failed to start Safari WebDriver session: %w", err)
	}
	
	// Conditionally defer Stop based on keep-browser flag
	if !opts.KeepBrowser {
		defer scrpr.Stop()
	} else {
		log.Println("⚠️  Browser will be kept open (use -keep-browser=false or kill process to close)")
	}
	
	// Scrape all targets (or filtered target)
	if opts.TargetFilter != "" {
		log.Printf("🎯 Testing specific target: %s", opts.TargetFilter)
		if err := scrpr.ScrapeFiltered(opts.TargetFilter); err != nil {
			return fmt.Errorf("failed to scrape target: %w", err)
		}
	} else {
		if err := scrpr.ScrapeAll(); err != nil {
			return fmt.Errorf("failed to scrape sites: %w", err)
		}
	}
	
	// Parse WSDOT HTML and create pass conditions image
	htmlPath := filepath.Join(scriptDir, "assets", "wsdot_stevens_pass.html")
	prsr := parser.New()
	status, err := prsr.ParseWSDOTPassStatus(htmlPath)
	if err != nil {
		log.Printf("Warning: Failed to parse WSDOT status: %v", err)
		// Create empty pass conditions image
		passCondPath := mgr.GetPassConditionsImagePath()
		return pkgimage.CreateEmptyImage(250, 200, passCondPath)
	}
	
	// Create pass conditions image based on status
	passCondPath := mgr.GetPassConditionsImagePath()
	if status.IsClosed {
		log.Printf("Pass is CLOSED: %s", status.Conditions)
		renderer := pkgimage.NewTextRenderer()
		return renderer.RenderCaption(status.Conditions, 250, 200, passCondPath)
	} else {
		log.Println("Pass is OPEN")
		return pkgimage.CreateEmptyImage(250, 200, passCondPath)
	}
}

// cropImages crops and resizes all configured images
func cropImages(scriptDir string) error {
	mgr := assets.NewManager(scriptDir)
	proc := pkgimage.NewProcessor(mgr)
	return proc.ProcessAll()
}

// renderComposite creates the final composite image
func renderComposite(scriptDir, outputPath string) error {
	mgr := assets.NewManager(scriptDir)
	comp := pkgimage.NewCompositor(mgr)
	return comp.Render(outputPath)
}

// setDesktopWallpaper sets the desktop wallpaper using CGO
func setDesktopWallpaper(imagePath string) error {
	if err := desktop.SetWallpaper(imagePath); err != nil {
		return err
	}
	
	// Also clear wallpaper cache
	if err := desktop.ClearWallpaperCache(); err != nil {
		log.Printf("Warning: %v", err)
	}
	
	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

// listScrapeTargets lists all available scrape targets
func listScrapeTargets() {
	// Get current directory to create manager
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		scriptDir = "."
	}
	
	mgr := assets.NewManager(scriptDir)
	targets := mgr.GetScrapeTargets()
	
	fmt.Println("Available Scrape Targets:")
	fmt.Println()
	
	for i, target := range targets {
		fmt.Printf("%d. %s\n", i+1, target.Name)
		fmt.Printf("   URL: %s\n", target.URL)
		fmt.Printf("   Selector: %s\n", target.Selector)
		fmt.Printf("   Default Wait: %dms\n", target.WaitTime)
		fmt.Printf("   Output: %s\n", filepath.Base(target.OutputPath))
		fmt.Println()
	}
	
	htmlTarget := mgr.GetWSDOTHTMLTarget()
	fmt.Printf("HTML Extraction Target:\n")
	fmt.Printf("   %s\n", htmlTarget.Name)
	fmt.Printf("   URL: %s\n", htmlTarget.URL)
	fmt.Printf("   Output: %s\n", filepath.Base(htmlTarget.OutputPath))
	fmt.Println()
	
	fmt.Println("Usage:")
	fmt.Println("  wd -s -scrape-target \"<name>\" -debug")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  wd -s -scrape-target \"Weather.gov Hourly\" -debug")
	fmt.Println("  wd -s -scrape-target \"NWAC\" -debug     # Matches all NWAC targets")
}

