package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
	"github.com/trodemaster/weatherdesktop/pkg/downloader"
	pkgimage "github.com/trodemaster/weatherdesktop/pkg/image"
	"github.com/trodemaster/weatherdesktop/pkg/parser"
	"github.com/trodemaster/weatherdesktop/pkg/playwright"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: wd-worker <command> [options]\n")
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  scrape   Scrape websites\n")
		fmt.Fprintf(os.Stderr, "  download Download images\n")
		fmt.Fprintf(os.Stderr, "  crop     Crop and resize images\n")
		fmt.Fprintf(os.Stderr, "  render   Render composite image\n")
		os.Exit(1)
	}

	command := os.Args[1]
	
	switch command {
	case "scrape":
		if err := runScrape(); err != nil {
			log.Fatalf("Scrape failed: %v", err)
		}
	case "download":
		if err := runDownload(); err != nil {
			log.Fatalf("Download failed: %v", err)
		}
	case "crop":
		if err := runCrop(); err != nil {
			log.Fatalf("Crop failed: %v", err)
		}
	case "render":
		if err := runRender(); err != nil {
			log.Fatalf("Render failed: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func runScrape() error {
	// Parse flags specific to scrape command
	scrapeFlags := flag.NewFlagSet("scrape", flag.ExitOnError)
	debugFlag := scrapeFlags.Bool("debug", false, "Enable debug mode")
	targetFlag := scrapeFlags.String("target", "", "Filter specific target")
	
	if err := scrapeFlags.Parse(os.Args[2:]); err != nil {
		return err
	}
	
	workDir := "/app"
	mgr := assets.NewManager(workDir)
	
	// Create scraper
	scraper := playwright.New(*debugFlag)
	
	// Start Playwright
	if err := scraper.Start(); err != nil {
		return fmt.Errorf("failed to start playwright: %w", err)
	}
	defer scraper.Stop()
	
	log.Println("Scraping sites...")
	
	// Scrape targets
	if *targetFlag != "" {
		if err := scraper.ScrapeFiltered(mgr, *targetFlag); err != nil {
			return fmt.Errorf("filtered scrape failed: %w", err)
		}
	} else {
		if err := scraper.ScrapeAll(mgr); err != nil {
			return fmt.Errorf("scrape failed: %w", err)
		}
	}
	
	// Also scrape WSDOT HTML
	wsdotTarget := mgr.GetWSDOTHTMLTarget()
	if err := scraper.ScrapeHTML(wsdotTarget); err != nil {
		log.Printf("Warning: Failed to scrape WSDOT HTML: %v", err)
	}
	
	log.Println("Asset Collection Completed...")
	return nil
}

func runDownload() error {
	workDir := "/app"
	mgr := assets.NewManager(workDir)
	
	log.Println("Downloading images...")
	
	// Download concurrently
	dl := downloader.New(mgr)
	if err := dl.DownloadAll(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	
	log.Println("Downloads completed")
	return nil
}

func runCrop() error {
	workDir := "/app"
	mgr := assets.NewManager(workDir)
	
	log.Println("Cropping and resizing images...")
	
	// Process all crop assets
	processor := pkgimage.NewProcessor(mgr)
	if err := processor.ProcessAll(); err != nil {
		return fmt.Errorf("crop failed: %w", err)
	}
	
	log.Println("Image processing completed")
	return nil
}

func runRender() error {
	workDir := "/app"
	mgr := assets.NewManager(workDir)
	
	// Generate output filename with timestamp
	renderedFilename := fmt.Sprintf("hud-%s.jpg", time.Now().Format("060102-1504"))
	outputPath := filepath.Join(workDir, "rendered", renderedFilename)
	
	log.Printf("Rendering composite image: %s", renderedFilename)
	
	// Parse WSDOT HTML for pass status and render status graphic
	wsdotHTML := filepath.Join(mgr.AssetsDir, "wsdot_stevens_pass.html")
	prsr := parser.New()
	passStatus, err := prsr.ParseWSDOTPassStatus(wsdotHTML)
	if err != nil {
		log.Printf("Warning: Failed to parse WSDOT status: %v", err)
		// Create empty/transparent image if parsing fails
		passConditionsPath := mgr.GetPassConditionsImagePath()
		if err := pkgimage.CreateEmptyImage(250, 200, passConditionsPath); err != nil {
			log.Printf("Warning: Failed to create empty pass conditions image: %v", err)
		}
	} else {
		// Normalize status values for display
		eastStatus := passStatus.East
		westStatus := passStatus.West
		
		// Check if status indicates "Open" (no restrictions, etc.)
		if strings.Contains(strings.ToLower(eastStatus), "no restrictions") {
			eastStatus = "Open"
		}
		if strings.Contains(strings.ToLower(westStatus), "no restrictions") {
			westStatus = "Open"
		}
		
		log.Printf("Pass Status - East: %s, West: %s", eastStatus, westStatus)
		
		// Create updated PassStatus with normalized values
		normalizedStatus := &parser.PassStatus{
			East:       eastStatus,
			West:       westStatus,
			IsClosed:   passStatus.IsClosed,
			Conditions: passStatus.Conditions,
		}
		
		// Render pass status graphic
		passConditionsPath := mgr.GetPassConditionsImagePath()
		textRenderer := pkgimage.NewTextRenderer()
		if err := textRenderer.RenderPassStatus(normalizedStatus, 250, 200, passConditionsPath); err != nil {
			log.Printf("Warning: Failed to render pass status graphic: %v", err)
			// Fallback to empty image
			if err := pkgimage.CreateEmptyImage(250, 200, passConditionsPath); err != nil {
				log.Printf("Warning: Failed to create empty pass conditions image: %v", err)
			}
		} else {
			log.Printf("Pass status graphic rendered: %s", passConditionsPath)
		}
	}
	
	// Composite the image
	compositor := pkgimage.NewCompositor(mgr)
	if err := compositor.Render(outputPath); err != nil {
		return fmt.Errorf("composite failed: %w", err)
	}
	
	log.Printf("Composite image saved: %s", outputPath)
	return nil
}

