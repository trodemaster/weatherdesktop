package desktop

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int setWallpaper(const char* imagePath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:imagePath];
        NSURL *imageURL = [NSURL fileURLWithPath:path];
        
        NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
        NSArray *screens = [NSScreen screens];
        
        for (NSScreen *screen in screens) {
            NSDictionary *options = @{
                NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown),
                NSWorkspaceDesktopImageAllowClippingKey: @(YES)
            };
            
            NSError *error = nil;
            BOOL success = [workspace setDesktopImageURL:imageURL
                                               forScreen:screen
                                                 options:options
                                                   error:&error];
            if (!success) {
                NSLog(@"Error setting wallpaper for screen: %@", error);
                return -1;
            }
        }
        return 0;
    }
}
*/
import "C"

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"
)

// SetWallpaper sets the desktop wallpaper on all screens
func SetWallpaper(imagePath string) error {
	// Verify file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file not found: %s", imagePath)
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Call Objective-C function via CGO
	cPath := C.CString(absPath)
	defer C.free(unsafe.Pointer(cPath))
	
	result := C.setWallpaper(cPath)
	if result != 0 {
		return fmt.Errorf("failed to set wallpaper (error code: %d)", result)
	}
	
	return nil
}

// ClearWallpaperCache clears macOS wallpaper cache
// Replicates: find "${TMPDIR}../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image" -name "*.png" -type f -delete
func ClearWallpaperCache() error {
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	
	// Construct cache path
	cachePath := filepath.Join(tmpDir, "../C/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image")
	
	// Check if cache directory exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// Cache doesn't exist, nothing to clear
		return nil
	}
	
	// Use find command to delete PNG files
	cmd := exec.Command("find", cachePath, "-name", "*.png", "-type", "f", "-delete")
	if err := cmd.Run(); err != nil {
		// Don't fail if cache clearing fails, just log it
		return fmt.Errorf("warning: failed to clear wallpaper cache: %w", err)
	}
	
	return nil
}

