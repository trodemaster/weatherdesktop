package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <unistd.h>

int setWallpaper(const char* imagePath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:imagePath];
        NSLog(@"Desktop: Setting wallpaper from path: %@", path);

        NSURL *imageURL = [NSURL fileURLWithPath:path];
        if (!imageURL) {
            NSLog(@"Desktop: ERROR - Failed to create NSURL from path: %@", path);
            return -1;
        }
        NSLog(@"Desktop: Created NSURL: %@", imageURL);

        // Check if file exists
        NSFileManager *fileManager = [NSFileManager defaultManager];
        BOOL fileExists = [fileManager fileExistsAtPath:path];
        if (!fileExists) {
            NSLog(@"Desktop: ERROR - File does not exist at path: %@", path);
            return -1;
        }
        NSLog(@"Desktop: File exists, size: %lld bytes", [[fileManager attributesOfItemAtPath:path error:nil] fileSize]);

        NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
        NSArray *screens = [NSScreen screens];
        NSUInteger screenCount = [screens count];

        NSLog(@"Desktop: Found %lu screen(s)", (unsigned long)screenCount);

        if (screenCount == 0) {
            NSLog(@"Desktop: ERROR - No screens found");
            return -1;
        }

        int successCount = 0;
        int failureCount = 0;

        // Set wallpaper on all screens (continue even if one fails)
        for (NSScreen *screen in screens) {
            NSUInteger screenIndex = [screens indexOfObject:screen];
            NSRect frame = [screen frame];
            NSLog(@"Desktop: Setting wallpaper for screen %lu (frame: %.0f x %.0f @ %.0f, %.0f)",
                  (unsigned long)screenIndex, frame.size.width, frame.size.height, frame.origin.x, frame.origin.y);

            // Standard options: scale proportionally, allow clipping.
            // Stale spaces are handled by the daily launchd flush job
            // (flush_wallpaper_cache.sh) which uses `defaults write` to clear
            // ChoiceRequests.ImageFiles through cfprefsd.
            NSDictionary *options = @{
                NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown),
                NSWorkspaceDesktopImageAllowClippingKey: @(YES)
            };

            NSError *error = nil;
            BOOL success = [workspace setDesktopImageURL:imageURL
                                               forScreen:screen
                                                 options:options
                                                   error:&error];
            if (success) {
                successCount++;
                NSLog(@"Desktop: ✓ Successfully set wallpaper for screen %lu", (unsigned long)screenIndex);
            } else {
                failureCount++;
                NSLog(@"Desktop: ✗ Failed to set wallpaper for screen %lu: %@",
                      (unsigned long)screenIndex, error ? [error localizedDescription] : @"Unknown error");
                if (error) {
                    NSLog(@"Desktop:   Error domain: %@, code: %ld", [error domain], (long)[error code]);
                }
            }
        }

        NSLog(@"Desktop: Summary - %d succeeded, %d failed out of %lu screens",
              successCount, failureCount, (unsigned long)screenCount);

        // Small delay to allow system to catch up (as recommended by desktoppr)
        NSLog(@"Desktop: Waiting 0.5 seconds for system to process...");
        usleep(500000); // 0.5 seconds

        // Return error only if ALL screens failed
        if (successCount == 0) {
            NSLog(@"Desktop: ERROR - Failed to set wallpaper on all %lu screens", (unsigned long)screenCount);
            return -1;
        }

        // Success if at least one screen succeeded
        if (failureCount > 0) {
            NSLog(@"Desktop: WARNING - Failed to set wallpaper on %d of %lu screens",
                  failureCount, (unsigned long)screenCount);
        } else {
            NSLog(@"Desktop: Successfully set wallpaper on all %lu screens", (unsigned long)screenCount);
        }

        return 0;
    }
}
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"
)

// SetWallpaper sets the desktop wallpaper on all screens
// verbose enables detailed logging
func SetWallpaper(imagePath string, verbose bool) error {
	if verbose {
		log.Printf("Desktop: Starting wallpaper setting process")
		log.Printf("Desktop: Input path: %s", imagePath)
	}

	// Verify file exists
	info, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		if verbose {
			log.Printf("Desktop: ERROR - File does not exist")
		}
		return fmt.Errorf("image file not found: %s", imagePath)
	}
	if err != nil {
		if verbose {
			log.Printf("Desktop: ERROR - Failed to stat file: %v", err)
		}
		return fmt.Errorf("failed to stat image file: %w", err)
	}
	if info.IsDir() {
		if verbose {
			log.Printf("Desktop: ERROR - Path is a directory")
		}
		return fmt.Errorf("path is a directory, not a file: %s", imagePath)
	}

	if verbose {
		log.Printf("Desktop: File verified - size: %d bytes, mode: %s", info.Size(), info.Mode())
	}

	// Get absolute path (required for NSURL fileURLWithPath)
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		if verbose {
			log.Printf("Desktop: ERROR - Failed to get absolute path: %v", err)
		}
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if verbose {
		log.Printf("Desktop: Absolute path: %s", absPath)
		log.Printf("Desktop: Calling Objective-C setWallpaper function...")
	}

	// Call Objective-C function via CGO
	cPath := C.CString(absPath)
	defer C.free(unsafe.Pointer(cPath))

	result := C.setWallpaper(cPath)
	if result != 0 {
		if verbose {
			log.Printf("Desktop: ERROR - Objective-C function returned error code: %d", result)
		}
		return fmt.Errorf("failed to set wallpaper on all screens")
	}

	if verbose {
		log.Printf("Desktop: Objective-C function completed successfully")
	}

	return nil
}

// ClearWallpaperCache clears macOS wallpaper caches and plist entries
// Includes TMPDIR cache, Container cache, and WallpaperImageExtension plist entries
// verbose enables detailed logging
// clearContainer controls whether to clear Container cache paths (requires permissions)
func ClearWallpaperCache(verbose bool, clearContainer bool) error {
	if verbose {
		log.Printf("Desktop: Clearing wallpaper caches...")
	}

	// Always clear TMPDIR-based cache (doesn't require special permissions)
	if err := clearTMPDIRCache(verbose); err != nil {
		if verbose {
			log.Printf("Desktop: WARNING - Failed to clear TMPDIR cache: %v", err)
		}
		// Continue even if this fails
	}

	// Only clear Container-based cache if explicitly requested (may trigger security prompt)
	if clearContainer {
		if err := clearContainerCache(verbose); err != nil {
			if verbose {
				log.Printf("Desktop: WARNING - Failed to clear Container cache: %v", err)
			}
			// Continue even if this fails
		}
	} else {
		if verbose {
			log.Printf("Desktop: Skipping Container cache cleanup (use -clear-cache flag to enable)")
		}
	}

	if verbose {
		log.Printf("Desktop: Cache clearing complete")
	}

	return nil
}

// clearTMPDIRCache clears TMPDIR-based wallpaper cache
// Path: ${TMPDIR}../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image
func clearTMPDIRCache(verbose bool) error {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}

	// Construct cache path
	cachePath := filepath.Join(tmpDir, "../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image")

	if verbose {
		log.Printf("Desktop: TMPDIR cache path: %s", cachePath)
	}

	// Check if cache directory exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		if verbose {
			log.Printf("Desktop: TMPDIR cache directory does not exist, skipping")
		}
		return nil
	}

	if verbose {
		log.Printf("Desktop: TMPDIR cache directory exists, clearing files...")
	}

	// Remove all files in the cache directory (not just PNGs)
	cmd := exec.Command("find", cachePath, "-type", "f", "-delete")
	if err := cmd.Run(); err != nil {
		if verbose {
			log.Printf("Desktop: WARNING - Failed to clear TMPDIR cache: %v", err)
		}
		return fmt.Errorf("failed to clear TMPDIR cache: %w", err)
	}

	if verbose {
		log.Printf("Desktop: TMPDIR cache cleared successfully")
	}

	return nil
}

// clearContainerCache clears WallpaperImageExtension plist entries via cfprefsd.
// Uses `defaults write` which routes through cfprefsd's in-memory cache — the
// only effective way to clear these entries (rm -f on the plist is ignored by cfprefsd).
//
// Note: Deleting files inside ~/Library/Containers/* triggers macOS's App Sandbox
// permission dialog on every run. The plist entry clearing below is sufficient to
// prevent the WallpaperAgent hang; the UUID cache JPGs in the extension container
// are a disk-space concern only and are managed by macOS independently.
func clearContainerCache(verbose bool) error {
	// Clear plist entries via defaults write (cfprefsd-aware, no sandbox prompt)
	if verbose {
		log.Printf("Desktop: Clearing WallpaperImageExtension plist entries via defaults...")
	}
	domain := "com.apple.wallpaper.extension.image"
	for _, key := range []string{
		"ChoiceRequests.ImageFiles",
		"ChoiceRequests.Assets",
		"ChoiceRequests.CollectionIdentifiers",
	} {
		cmd := exec.Command("defaults", "write", domain, key, "-array")
		if err := cmd.Run(); err != nil {
			if verbose {
				log.Printf("Desktop: WARNING - Failed to clear %s: %v", key, err)
			}
			// Continue even if this fails
		}
	}
	if verbose {
		log.Printf("Desktop: Plist entries cleared successfully")
	}

	return nil
}
