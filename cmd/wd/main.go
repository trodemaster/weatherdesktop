package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
	"github.com/trodemaster/weatherdesktop/pkg/desktop"
	"github.com/trodemaster/weatherdesktop/pkg/docker"
)

var (
	scrapeFlag      = flag.Bool("s", false, "Scrape websites")
	downloadFlag    = flag.Bool("d", false, "Download images")
	cropFlag        = flag.Bool("c", false, "Crop/resize images")
	renderFlag      = flag.Bool("r", false, "Render composite image")
	desktopFlag     = flag.Bool("p", false, "Set desktop wallpaper")
	desktopImageFlag = flag.String("set-desktop", "", "Set desktop wallpaper from specified image file path")
	desktopMethodFlag = flag.String("desktop-method", "cgo", "Wallpaper setting method (default: 'cgo')")
	flushFlag       = flag.Bool("f", false, "Flush/clear assets directory")
	uploadFlag      = flag.Bool("upload", false, "Upload latest rendered image to remote server via SCP (requires SSH_TARGET env var)")
	debugFlag       = flag.Bool("debug", false, "Enable debug output")
	scrapeTargetFlag = flag.String("scrape-target", "", "Test specific scrape target by name")
	listTargetsFlag = flag.Bool("list-targets", false, "List all available scrape targets and exit")
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
		fmt.Fprintf(os.Stderr, "   -p                    Set Desktop (uses most recent rendered image)\n")
		fmt.Fprintf(os.Stderr, "   -set-desktop <path>   Set desktop wallpaper from specified image file\n")
		fmt.Fprintf(os.Stderr, "   -desktop-method <m>   Wallpaper method (default: 'cgo')\n")
		fmt.Fprintf(os.Stderr, "   -f                    Flush assets\n")
		fmt.Fprintf(os.Stderr, "   -upload               Upload latest rendered image to remote server via SCP\n")
		fmt.Fprintf(os.Stderr, "                         (requires SSH_TARGET environment variable)\n")
		fmt.Fprintf(os.Stderr, "\nDEBUG OPTIONS:\n")
		fmt.Fprintf(os.Stderr, "   -debug                Enable debug output\n")
		fmt.Fprintf(os.Stderr, "   -list-targets         List all available scrape targets\n")
		fmt.Fprintf(os.Stderr, "   -scrape-target <name> Test specific scrape target (e.g., \"Weather.gov Hourly\")\n")
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "   wd -s -scrape-target \"NWAC Stevens\" -debug\n")
		fmt.Fprintf(os.Stderr, "   wd -s -debug\n")
		fmt.Fprintf(os.Stderr, "   wd -set-desktop ./rendered/hud-251102-1056.jpg\n")
		fmt.Fprintf(os.Stderr, "   wd -set-desktop ./rendered/hud-251102-1056.jpg\n")
	}
	
	flag.Parse()

	// Handle list-targets flag (special case - exits after listing)
	if *listTargetsFlag {
		listScrapeTargets()
		return
	}

	// Handle set-desktop flag (special case - just set desktop from specified file)
	// Always allows desktop setting (even with -debug) since it's explicitly requested
	if *desktopImageFlag != "" {
		imagePath := *desktopImageFlag
		
		// Resolve to absolute path if relative
		if !filepath.IsAbs(imagePath) {
			wd, err := os.Getwd()
			if err != nil {
				log.Fatalf("Failed to get working directory: %v", err)
			}
			imagePath = filepath.Join(wd, imagePath)
		}
		
		// Normalize the path
		absPath, err := filepath.Abs(imagePath)
		if err != nil {
			log.Fatalf("Failed to resolve image path: %v", err)
		}
		
		if *debugFlag {
			log.Printf("Setting desktop wallpaper on all screens (debug mode) from: %s", absPath)
		} else {
			log.Printf("Setting desktop wallpaper on all screens from: %s", absPath)
		}
		if err := setDesktopWallpaper(absPath, *desktopMethodFlag); err != nil {
			log.Fatalf("Failed to set desktop: %v", err)
		}
		log.Println("✓ Desktop wallpaper set successfully on all screens")
		
		// Upload to remote server if requested (after desktop setting)
		if *uploadFlag {
			scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				scriptDir = "."
			}
			if err := uploadToRemote(absPath, scriptDir); err != nil {
				log.Printf("Warning: Failed to upload to remote server: %v", err)
			} else {
				log.Println("✓ Image uploaded to remote server successfully")
			}
		}
		return
	}

	// Handle upload-only flag (special case - just upload latest rendered image)
	// Check if only -upload is specified (no other phase flags)
	hasPhaseFlags := *scrapeFlag || *downloadFlag || *cropFlag || *renderFlag || *desktopFlag || *flushFlag
	if *uploadFlag && !hasPhaseFlags {
		// Get script directory
		scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Fatalf("Failed to get script directory: %v", err)
		}

		// Find the most recent rendered file
		renderedDir := filepath.Join(scriptDir, "rendered")
		renderedPath, err := findMostRecentRendered(renderedDir)
		if err != nil {
			log.Fatalf("Failed to find rendered file: %v", err)
		}

		// Upload to remote server
		if *debugFlag {
			log.Printf("Upload: Uploading latest rendered image: %s", renderedPath)
		} else {
			log.Printf("Uploading latest rendered image: %s", renderedPath)
		}
		if err := uploadToRemote(renderedPath, scriptDir); err != nil {
			log.Fatalf("Failed to upload to remote server: %v", err)
		}
		log.Println("✓ Image uploaded to remote server successfully")
		return
	}

	// Get script directory
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("Failed to get script directory: %v", err)
	}

	// Initialize Docker client
	dockerClient := docker.New(scriptDir)

	// Determine which phases to run
	// If no flags set, run all phases (same logic as bash script lines 82-84)
	// Note: desktopImageFlag and uploadFlag are handled separately, so we exclude them from runAll check
	// uploadFlag is also excluded because it can be used standalone or with other flags
	runAll := !(*scrapeFlag || *downloadFlag || *cropFlag || *renderFlag || *desktopFlag || *flushFlag)
	
	doScrape := runAll || *scrapeFlag
	doDownload := runAll || *downloadFlag
	doCrop := runAll || *cropFlag
	doRender := runAll || *renderFlag
	doDesktop := runAll || *desktopFlag
	doFlush := runAll || *flushFlag
	
	// Filename will be generated by container at render time to avoid timezone/timing issues
	log.Printf("Starting wallpaper generation...")

	// Phase 0: Flush assets if requested
	if doFlush {
		if err := flushAssets(scriptDir); err != nil {
			log.Printf("Warning: Failed to flush assets: %v", err)
		}
	}

	// Ensure Docker container is running for any Docker-based phases
	if doDownload || doScrape || doCrop || doRender {
		if err := dockerClient.EnsureRunning(); err != nil {
			log.Fatalf("Failed to ensure Docker container is running: %v", err)
		}
	}

	// Phase 1: Scrape websites
	if doScrape {
		log.Println("Scraping sites...")
		
		args := []string{"/app/wd-worker", "scrape"}
		if *debugFlag {
			args = append(args, "--debug")
		}
		if *scrapeTargetFlag != "" {
			args = append(args, "--target", *scrapeTargetFlag)
		}
		
		if err := dockerClient.Exec(args...); err != nil {
			log.Fatalf("Failed to scrape sites: %v", err)
		}
	}

	// Phase 2: Download images
	if doDownload {
		log.Println("Downloading images...")
		
		if err := dockerClient.Exec("/app/wd-worker", "download"); err != nil {
			log.Fatalf("Failed to download images: %v", err)
		}
	}

	// Wait for asset collection to complete
	if doDownload || doScrape {
		log.Println("Asset Collection Completed...")
	}

	// Phase 3: Crop and resize images
	if doCrop {
		log.Println("Cropping images...")
		
		if err := dockerClient.Exec("/app/wd-worker", "crop"); err != nil {
			log.Fatalf("Failed to crop images: %v", err)
		}
		
		log.Println("Cropping completed...")
	}

	// Phase 4: Render composite image
	if doRender {
		log.Println("Rendering...")
		
		if err := dockerClient.Exec("/app/wd-worker", "render"); err != nil {
			log.Fatalf("Failed to render composite: %v", err)
		}
		
		log.Println("Rendering completed...")
	}

	// Phase 5: Set desktop wallpaper
	if doDesktop {
		// Skip desktop setting in debug mode UNLESS desktop was explicitly requested (-p flag)
		// This allows troubleshooting desktop setting with -p -debug while preventing accidental
		// wallpaper changes during other debug operations
		skipDesktopInDebug := *debugFlag && !*desktopFlag
		if skipDesktopInDebug {
			log.Println("⚠️  Skipping desktop wallpaper setting (debug mode active)")
			log.Println("   Use -p -debug to set desktop with debug output, or remove -debug flag")
		} else {
			// Find the most recent rendered file
			renderedDir := filepath.Join(scriptDir, "rendered")
			renderedPath, err := findMostRecentRendered(renderedDir)
			if err != nil {
				log.Fatalf("Failed to find rendered file: %v", err)
			}
			
			if *debugFlag {
				log.Printf("Setting desktop wallpaper on all screens (debug mode): %s", renderedPath)
			} else {
				log.Printf("Setting desktop wallpaper on all screens: %s", renderedPath)
			}
			if err := setDesktopWallpaper(renderedPath, *desktopMethodFlag); err != nil {
				log.Fatalf("Failed to set desktop: %v", err)
			}
			log.Println("✓ Desktop wallpaper set successfully on all screens")
			
			// Upload to remote server if requested via flag or if SSH_TARGET env var is set (default behavior)
			shouldUpload := *uploadFlag || os.Getenv("SSH_TARGET") != ""
			if shouldUpload {
				if err := uploadToRemote(renderedPath, scriptDir); err != nil {
					log.Printf("Warning: Failed to upload to remote server: %v", err)
				} else {
					log.Println("✓ Image uploaded to remote server successfully")
				}
			}
		}
	} else {
		log.Printf("Skipping desktop wallpaper (doDesktop=%v, runAll=%v, desktopFlag=%v)", doDesktop, runAll, *desktopFlag)
		
		// If upload flag is set but desktop wasn't set, still try to upload latest image
		if *uploadFlag {
			renderedDir := filepath.Join(scriptDir, "rendered")
			renderedPath, err := findMostRecentRendered(renderedDir)
			if err != nil {
				log.Printf("Warning: Failed to find rendered file for upload: %v", err)
			} else {
				if err := uploadToRemote(renderedPath, scriptDir); err != nil {
					log.Printf("Warning: Failed to upload to remote server: %v", err)
				} else {
					log.Println("✓ Image uploaded to remote server successfully")
				}
			}
		}
	}

	// Optional: Copy to CDN if mounted
	cdnPath := "/Volumes/Bomb20/cdn"
	if doRender {
		if info, err := os.Stat(cdnPath); err == nil && info.IsDir() {
			renderedDir := filepath.Join(scriptDir, "rendered")
			renderedPath, err := findMostRecentRendered(renderedDir)
			if err == nil {
				destPath := filepath.Join(cdnPath, "stevens_pass.jpg")
				log.Printf("Copying %s to %s", renderedPath, destPath)
				if err := copyFile(renderedPath, destPath); err != nil {
					log.Printf("Warning: Failed to copy to CDN: %v", err)
				}
			}
		}
	}

	log.Println("End of Line...")
}

// flushAssets removes all files from the assets directory
func findMostRecentRendered(renderedDir string) (string, error) {
	pattern := filepath.Join(renderedDir, "hud-*.jpg")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to glob rendered files: %w", err)
	}
	
	if len(files) == 0 {
		return "", fmt.Errorf("no rendered files found in %s", renderedDir)
	}
	
	// Find the most recently modified file
	var mostRecent string
	var mostRecentTime time.Time
	
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if mostRecent == "" || info.ModTime().After(mostRecentTime) {
			mostRecent = file
			mostRecentTime = info.ModTime()
		}
	}
	
	if mostRecent == "" {
		return "", fmt.Errorf("failed to determine most recent rendered file")
	}
	
	return mostRecent, nil
}

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

// setDesktopWallpaper sets the desktop wallpaper using CGO method
func setDesktopWallpaper(imagePath string, method string) error {
	verbose := *debugFlag
	log.Printf("Desktop: ========================================")
	log.Printf("Desktop: Setting desktop wallpaper")
	log.Printf("Desktop: Image path: %s", imagePath)
	log.Printf("Desktop: Method: %s", method)
	log.Printf("Desktop: Verbose logging: %v", verbose)
	log.Printf("Desktop: ========================================")

	// Only CGO method is supported
	if method != "cgo" {
		return fmt.Errorf("unsupported desktop method: %s (only 'cgo' is supported)", method)
	}

	if err := desktop.SetWallpaper(imagePath, verbose); err != nil {
		log.Printf("Desktop: ERROR - CGO SetWallpaper failed: %v", err)
		return err
	}
	log.Printf("Desktop: CGO SetWallpaper completed successfully")

	// Also clear wallpaper cache
	log.Printf("Desktop: Clearing wallpaper cache...")
	if err := desktop.ClearWallpaperCache(verbose); err != nil {
		log.Printf("Desktop: Warning: %v", err)
	}

	log.Printf("Desktop: ========================================")
	log.Printf("Desktop: Desktop wallpaper setting process complete")
	log.Printf("Desktop: ========================================")

	return nil
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}

// sshTargetInfo contains parsed information from SSH_TARGET
type sshTargetInfo struct {
	Host string // e.g., "wx"
	Dir  string // e.g., "/var/www/html/weewx/stevenspass/"
}

// parseSSHTarget parses SSH_TARGET env var into host and directory
// Format: "host:/path/to/dir" or "host:/path/to/dir/"
func parseSSHTarget(sshTarget string) (*sshTargetInfo, error) {
	parts := strings.SplitN(sshTarget, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("SSH_TARGET must be in format 'host:/path/to/dir', got: %s", sshTarget)
	}

	host := parts[0]
	dir := parts[1]
	
	// Normalize directory path - ensure it ends with /
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	return &sshTargetInfo{
		Host: host,
		Dir:  dir,
	}, nil
}

// uploadToRemote uploads the latest rendered image to a remote server via SCP
// Uses SSH_TARGET environment variable to construct the scp command
// Uploads with original filename, then runs sed to update HTML template, and cleans up old files
func uploadToRemote(imagePath string, scriptDir string) error {
	sshTarget := os.Getenv("SSH_TARGET")
	if sshTarget == "" {
		return fmt.Errorf("SSH_TARGET environment variable is not set")
	}

	// Parse SSH_TARGET
	targetInfo, err := parseSSHTarget(sshTarget)
	if err != nil {
		return fmt.Errorf("failed to parse SSH_TARGET: %w", err)
	}

	// Get original filename
	originalFilename := filepath.Base(imagePath)
	remotePath := targetInfo.Host + ":" + targetInfo.Dir + originalFilename

	// Step 1: Upload with original filename
	log.Printf("Upload: Executing: scp %s %s", imagePath, remotePath)
	cmd := exec.Command("scp", imagePath, remotePath)
	
	if *debugFlag {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Printf("Upload: Preparing to upload %s to %s", imagePath, remotePath)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp command failed: %w", err)
	}
	log.Printf("Upload: Successfully uploaded %s to %s", originalFilename, remotePath)

	// Step 2: Run sed command to update HTML template (idempotent)
	// Update /etc/weewx/skins/stevenspass/index.html.tmpl to use the new image filename
	// This is idempotent: it will replace any existing image filename (stevenspass.jpg or hud-*.jpg) with the current filename
	templatePath := "/etc/weewx/skins/stevenspass/index.html.tmpl"
	// Replace any image filename in src attribute: handles hud-*.jpg pattern
	// Assumes template has already been manually edited to use hud-YYMMDD-HHMM.jpg format
	// Simple pattern: just match hud-*.jpg and replace with new filename
	sedExpr := fmt.Sprintf("s/hud-.*\\.jpg/%s/g", originalFilename)
	// Construct command: pass directly to SSH with proper quoting
	fullCmd := fmt.Sprintf("sed -i '%s' %s", sedExpr, templatePath)
	sshCmd := exec.Command("ssh", targetInfo.Host, fullCmd)
	
	log.Printf("Upload: Executing: ssh %s '%s'", targetInfo.Host, fullCmd)
	if *debugFlag {
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr
	}

	if err := sshCmd.Run(); err != nil {
		log.Printf("Upload: Warning: sed command failed (may be expected if no HTML files found): %v", err)
		// Don't fail the entire upload if sed fails
	} else {
		log.Printf("Upload: Successfully updated HTML template(s)")
	}

	// Step 3: Cleanup old files (keep only N and N-1)
	if err := cleanupOldRemoteFiles(targetInfo); err != nil {
		log.Printf("Upload: Warning: Failed to cleanup old files: %v", err)
		// Don't fail the entire upload if cleanup fails
	}

	return nil
}

// cleanupOldRemoteFiles removes old image files on remote, keeping only the newest and second-newest
func cleanupOldRemoteFiles(targetInfo *sshTargetInfo) error {
	// List all hud-*.jpg files on remote, sorted by modification time (newest first)
	// Format: full path with modification time for sorting
	listCmd := fmt.Sprintf("ls -t %shud-*.jpg 2>/dev/null", targetInfo.Dir)
	sshCmd := exec.Command("ssh", targetInfo.Host, listCmd)
	
	output, err := sshCmd.Output()
	if err != nil {
		// If no files found or command fails, that's okay
		if *debugFlag {
			log.Printf("Upload: Cleanup: No files found or command failed: %v", err)
		}
		return nil
	}

	// Parse file list
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var allFiles []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			allFiles = append(allFiles, line)
		}
	}

	if len(allFiles) <= 2 {
		if *debugFlag {
			log.Printf("Upload: Cleanup: Only %d file(s) found, nothing to remove", len(allFiles))
		}
		return nil
	}

	// Files are already sorted by modification time (newest first) from ls -t
	// Keep the first 2 (newest and second-newest), remove the rest
	filesToKeep := allFiles[:2]
	filesToRemove := allFiles[2:]

	if *debugFlag {
		log.Printf("Upload: Cleanup: Keeping files: %v", filesToKeep)
		log.Printf("Upload: Cleanup: Removing files: %v", filesToRemove)
	}

	// Remove old files
	removedCount := 0
	for _, file := range filesToRemove {
		rmCmd := fmt.Sprintf("rm -f %s", file)
		rmSSH := exec.Command("ssh", targetInfo.Host, rmCmd)
		if *debugFlag {
			log.Printf("Upload: Cleanup: Removing old file: %s", file)
		}
		if err := rmSSH.Run(); err != nil {
			if *debugFlag {
				log.Printf("Upload: Cleanup: Warning: Failed to remove %s: %v", file, err)
			}
		} else {
			removedCount++
		}
	}

	if removedCount > 0 {
		log.Printf("Upload: Cleanup: Removed %d old file(s), kept 2 most recent", removedCount)
	} else {
		log.Printf("Upload: Cleanup: No old files to remove")
	}

	return nil
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

