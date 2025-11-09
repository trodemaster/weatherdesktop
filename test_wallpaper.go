package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework IOKit -framework CoreGraphics
#import <Cocoa/Cocoa.h>
#import <unistd.h>
#import <dispatch/dispatch.h>
#import <string.h>

// Get screen UUID (used for wallpaper configuration)
const char* getScreenUUID(NSScreen *screen) {
    @autoreleasepool {
        // On macOS, screen UUIDs are typically stored in display preferences
        // Try to get it from the screen's device description
        NSDictionary *deviceDescription = [screen deviceDescription];
        NSNumber *screenNumber = [deviceDescription objectForKey:@"NSScreenNumber"];
        if (screenNumber) {
            CGDirectDisplayID displayID = [screenNumber unsignedIntValue];
            // Get UUID from IOKit
            CFUUIDRef uuid = CGDisplayCreateUUIDFromDisplayID(displayID);
            if (uuid) {
                CFStringRef uuidString = CFUUIDCreateString(kCFAllocatorDefault, uuid);
                NSString *uuidNS = (__bridge NSString *)uuidString;
                const char *result = [uuidNS UTF8String];
                CFRelease(uuidString);
                CFRelease(uuid);
                return result;
            }
        }
        return "UNKNOWN";
    }
}

// Check if we're on the main thread
int isMainThread() {
    return [NSThread isMainThread] ? 1 : 0;
}

// Get thread name for debugging
const char* getCurrentThreadName() {
    NSString *threadName = [[NSThread currentThread] name];
    if (threadName && [threadName length] > 0) {
        return [threadName UTF8String];
    }
    return [[NSThread currentThread] isMainThread] ? "Main Thread" : "Background Thread";
}

// Forward declaration
int setWallpaperOnMainThread(const char* imagePath);
char* getCurrentWallpaper(int screenIndex);

// Enhanced wallpaper setting with detailed error reporting
int setWallpaperEnhanced(const char* imagePath) {
    @autoreleasepool {
        // Initialize NSApplication if needed for GUI access
        if (NSApp == nil) {
            [NSApplication sharedApplication];
            NSLog(@"Initialized NSApplication for GUI access");
        }

        // Log thread information
        BOOL onMainThread = [NSThread isMainThread];
        NSString *threadName = [[NSThread currentThread] name];
        if (!threadName || [threadName length] == 0) {
            threadName = onMainThread ? @"Main Thread" : @"Background Thread";
        }
        NSLog(@"===========================================");
        NSLog(@"Wallpaper Test: Starting");
        NSLog(@"Thread: %@ (Main: %@)", threadName, onMainThread ? @"YES" : @"NO");
        NSLog(@"===========================================");
        
        if (!onMainThread) {
            NSLog(@"WARNING: Not on main thread! NSWorkspace methods may fail.");
            NSLog(@"Attempting to dispatch to main queue...");
            
            // Try to dispatch to main queue
            dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
            __block int result = -1;
            
            dispatch_async(dispatch_get_main_queue(), ^{
                result = setWallpaperOnMainThread(imagePath);
                dispatch_semaphore_signal(semaphore);
            });
            
            // Wait for completion (max 10 seconds)
            dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, 10 * NSEC_PER_SEC);
            if (dispatch_semaphore_wait(semaphore, timeout) != 0) {
                NSLog(@"ERROR: Timeout waiting for main thread execution");
                return -1;
            }
            
            return result;
        }
        
        return setWallpaperOnMainThread(imagePath);
    }
}

// Actual wallpaper setting (must be called from main thread)
int setWallpaperOnMainThread(const char* imagePath) {
    @autoreleasepool {
        NSString *path = [NSString stringWithUTF8String:imagePath];
        NSLog(@"Setting wallpaper from path: %@", path);
        
        // Verify we're on main thread
        if (![NSThread isMainThread]) {
            NSLog(@"ERROR: setWallpaperOnMainThread called from non-main thread!");
            return -1;
        }
        
        NSURL *imageURL = [NSURL fileURLWithPath:path];
        if (!imageURL) {
            NSLog(@"ERROR: Failed to create NSURL from path: %@", path);
            return -1;
        }
        NSLog(@"Created NSURL: %@", imageURL);
        
        // Check if file exists and get details
        NSFileManager *fileManager = [NSFileManager defaultManager];
        BOOL fileExists = [fileManager fileExistsAtPath:path];
        if (!fileExists) {
            NSLog(@"ERROR: File does not exist at path: %@", path);
            return -1;
        }
        
        NSError *attrError = nil;
        NSDictionary *fileAttrs = [fileManager attributesOfItemAtPath:path error:&attrError];
        if (fileAttrs) {
            NSNumber *fileSize = [fileAttrs objectForKey:NSFileSize];
            NSNumber *filePerms = [fileAttrs objectForKey:NSFilePosixPermissions];
            NSLog(@"File exists:");
            NSLog(@"  Size: %lld bytes", [fileSize longLongValue]);
            NSLog(@"  Permissions: %o", [filePerms intValue]);
            NSLog(@"  Readable: %@", [fileManager isReadableFileAtPath:path] ? @"YES" : @"NO");
        } else {
            NSLog(@"WARNING: Could not get file attributes: %@", attrError ? [attrError localizedDescription] : @"Unknown");
        }
        
        // Try to load image to verify it's valid
        NSImage *testImage = [[NSImage alloc] initWithContentsOfFile:path];
        if (testImage) {
            NSSize imageSize = [testImage size];
            NSLog(@"Image loaded successfully:");
            NSLog(@"  Size: %.0f x %.0f points", imageSize.width, imageSize.height);
            NSLog(@"  Valid: YES");
        } else {
            NSLog(@"WARNING: Could not load image with NSImage - may still work");
        }
        
        NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
        NSArray *screens = [NSScreen screens];
        NSUInteger screenCount = [screens count];

        NSLog(@"Found %lu screen(s)", (unsigned long)screenCount);
        NSLog(@"NSScreen.screens returned: %@", screens);
        NSLog(@"NSApp: %@", NSApp);
        NSLog(@"NSApp keyWindow: %@", [NSApp keyWindow]);
        NSLog(@"Current run loop mode: %@", [[NSRunLoop currentRunLoop] currentMode]);

        if (screenCount == 0) {
            NSLog(@"ERROR: No screens found");
            NSLog(@"This may indicate the program is running without GUI access");
            NSLog(@"Try running from a GUI environment or with proper application initialization");
            return -1;
        }
        
        int successCount = 0;
        int failureCount = 0;

        // Set wallpaper on only the primary screen (screen 0)
        if (screenCount > 0) {
            NSScreen *screen = [screens objectAtIndex:0];
            NSUInteger screenIndex = 0;
            NSRect frame = [screen frame];
            NSLog(@"-------------------------------------------");
            NSLog(@"Screen %lu:", (unsigned long)screenIndex);
            NSLog(@"  Frame: %.0f x %.0f @ %.0f, %.0f", 
                  frame.size.width, frame.size.height, frame.origin.x, frame.origin.y);
            NSLog(@"  Visible: %@", [screen visibleFrame].size.width > 0 ? @"YES" : @"NO");
            
            // Try to get screen UUID from IOKit
            // Note: CGDisplayCreateUUIDFromDisplayID is deprecated and may not be available
            // We'll skip UUID detection for now and match by screen properties instead
            NSDictionary *deviceDescription = [screen deviceDescription];
            NSNumber *screenNumber = [deviceDescription objectForKey:@"NSScreenNumber"];
            if (screenNumber) {
                CGDirectDisplayID displayID = [screenNumber unsignedIntValue];
                NSLog(@"  Display ID: %u", displayID);
            }
            NSLog(@"  Display UUID: (skipping - will match by screen properties)");
            
            // Check current wallpaper before setting
            NSURL *currentURL = [workspace desktopImageURLForScreen:screen];
            if (currentURL) {
                NSLog(@"  Current wallpaper: %@", [currentURL path]);
            } else {
                NSLog(@"  Current wallpaper: (none detected)");
            }
            
            // Try different option combinations
            // First try: Standard options
            NSDictionary *options1 = @{
                NSWorkspaceDesktopImageScalingKey: @(NSImageScaleProportionallyUpOrDown),
                NSWorkspaceDesktopImageAllowClippingKey: @(YES)
            };
            
            // Try alternative: Get current options and merge
            NSDictionary *currentOptions = [workspace desktopImageOptionsForScreen:screen];
            NSMutableDictionary *options = [NSMutableDictionary dictionaryWithDictionary:options1];
            if (currentOptions) {
                NSLog(@"  Current options: %@", currentOptions);
                // Merge current options to preserve any system settings
                [options addEntriesFromDictionary:currentOptions];
            }
            
            NSLog(@"  Using options: %@", options);
            
            NSError *error = nil;
            NSDate *startTime = [NSDate date];
            BOOL success = [workspace setDesktopImageURL:imageURL
                                               forScreen:screen
                                                 options:options
                                                   error:&error];
            NSTimeInterval elapsed = [[NSDate date] timeIntervalSinceDate:startTime];
            
            if (success) {
                successCount++;
                NSLog(@"  ✓ API call SUCCESS (took %.3f seconds)", elapsed);
                
                // Wait a moment then verify what's actually set
                usleep(200000); // 0.2 seconds
                NSURL *verifyURL = [workspace desktopImageURLForScreen:screen];
                if (verifyURL) {
                    NSString *verifyPath = [verifyURL path];
                    NSString *expectedPath = path;
                    if ([verifyPath isEqualToString:expectedPath]) {
                        NSLog(@"  ✓ Verification: Wallpaper matches expected path");
                    } else {
                        NSLog(@"  ⚠ Verification FAILED:");
                        NSLog(@"    Expected: %@", expectedPath);
                        NSLog(@"    Actual:   %@", verifyPath);
                        NSLog(@"    This indicates macOS extension system issue");
                    }
                } else {
                    NSLog(@"  ⚠ Verification: Could not query wallpaper URL");
                }
            } else {
                failureCount++;
                NSLog(@"  ✗ FAILED (took %.3f seconds)", elapsed);
                if (error) {
                    NSLog(@"  Error domain: %@", [error domain]);
                    NSLog(@"  Error code: %ld", (long)[error code]);
                    NSLog(@"  Error description: %@", [error localizedDescription]);
                    
                    // Log userInfo dictionary if available
                    NSDictionary *userInfo = [error userInfo];
                    if (userInfo && [userInfo count] > 0) {
                        NSLog(@"  Error userInfo:");
                        for (NSString *key in userInfo) {
                            NSLog(@"    %@: %@", key, [userInfo objectForKey:key]);
                        }
                    }
                } else {
                    NSLog(@"  No error object provided");
                }
            }
        } else {
            NSLog(@"ERROR: No screens available to test");
            return -1;
        }
        
        NSLog(@"===========================================");
        NSLog(@"Summary:");
        NSLog(@"  Testing: Primary screen only (screen 0)");
        NSLog(@"  Total screens available: %lu", (unsigned long)screenCount);
        NSLog(@"  API Success: %d", successCount);
        NSLog(@"  API Failed: %d", failureCount);
        NSLog(@"===========================================");
        
        // Small delay to allow system to catch up
        NSLog(@"Waiting 1 second for wallpaper extension system to process...");
        usleep(1000000); // 1 second
        
        // Verify what's actually set after delay (primary screen only)
        NSLog(@"===========================================");
        NSLog(@"POST-SET VERIFICATION:");
        if (screenCount > 0) {
            NSScreen *screen = [screens objectAtIndex:0];
            NSUInteger screenIndex = 0;
            NSURL *verifyURL = [workspace desktopImageURLForScreen:screen];
            if (verifyURL) {
                NSString *verifyPath = [verifyURL path];
                NSLog(@"Screen %lu wallpaper URL: %@", (unsigned long)screenIndex, verifyPath);
                if ([verifyPath isEqualToString:path]) {
                    NSLog(@"  ✓ URL matches expected path");
                } else {
                    NSLog(@"  ⚠ URL MISMATCH:");
                    NSLog(@"    Expected: %@", path);
                    NSLog(@"    Actual:   %@", verifyPath);
                    NSLog(@"    This indicates the API set a different wallpaper");
                }
            } else {
                NSLog(@"Screen %lu wallpaper URL: (nil - cannot query)", (unsigned long)screenIndex);
            }
        }
        NSLog(@"===========================================");
        
        // Read wallpaper configuration from Index.plist with enhanced analysis
        // Based on private API insights: WallpaperChoiceConfiguration structure
        NSLog(@"===========================================");
        NSLog(@"ENHANCED WALLPAPER CONFIGURATION ANALYSIS:");
        NSLog(@"Using private API insights for better analysis (primary screen test)");
        NSString *homeDir = NSHomeDirectory();
        NSString *indexPath = [homeDir stringByAppendingPathComponent:@"Library/Application Support/com.apple.wallpaper/Store/Index.plist"];
        NSFileManager *indexFileManager = [NSFileManager defaultManager];
        if ([indexFileManager fileExistsAtPath:indexPath]) {
            NSLog(@"Found wallpaper Index.plist at: %@", indexPath);

            NSDictionary *indexDict = [NSDictionary dictionaryWithContentsOfFile:indexPath];
            if (indexDict) {
                NSDictionary *displays = [indexDict objectForKey:@"Displays"];
                if (displays) {
                    NSLog(@"Found %lu display configurations (testing primary screen)", (unsigned long)[displays count]);

                    for (NSString *displayUUID in displays) {
                        NSDictionary *displayConfig = [displays objectForKey:displayUUID];
                        NSDictionary *desktop = [displayConfig objectForKey:@"Desktop"];
                        if (desktop) {
                            NSDictionary *content = [desktop objectForKey:@"Content"];
                            if (content) {
                                NSArray *choices = [content objectForKey:@"Choices"];
                                if (choices && [choices count] > 0) {
                                    NSDictionary *choice = [choices objectAtIndex:0];
                                    NSString *provider = [choice objectForKey:@"Provider"];
                                    NSData *configData = [choice objectForKey:@"Configuration"];

                                    NSLog(@"  Display UUID: %@", displayUUID);
                                    NSLog(@"    Provider: %@", provider);

                                    // Enhanced analysis using private API insights
                                    if ([provider isEqualToString:@"com.apple.wallpaper.choice.image"]) {
                                        // This matches WallpaperChoiceProviderID.image from private APIs
                                        NSLog(@"    ✓ Static image provider (matches private API WallpaperChoiceProviderID.image)");

                                        if (configData) {
                                            // Try to decode binary plist with enhanced error handling
                                            NSError *plistError = nil;
                                            NSPropertyListFormat format;
                                            NSDictionary *configDict = [NSPropertyListSerialization propertyListWithData:configData
                                                                                                                 options:NSPropertyListImmutable
                                                                                                                  format:&format
                                                                                                                   error:&plistError];

                                            if (configDict && !plistError) {
                                                NSLog(@"    ✓ Successfully decoded WallpaperChoiceConfiguration");

                                                // Analyze configuration structure based on private API insights
                                                NSDictionary *urlDict = [configDict objectForKey:@"url"];
                                                if (urlDict) {
                                                    NSString *relativePath = [urlDict objectForKey:@"relative"];
                                                    if (relativePath) {
                                                        NSLog(@"    Image path: %@", relativePath);

                                                        // Check if our test image is configured
                                                        if ([relativePath containsString:@"hud-251102-2153"]) {
                                                            NSLog(@"    🎯 SUCCESS: Configuration contains our test image!");
                                                            NSLog(@"    ✓ Private API insights confirmed: wallpaper should be set");
                                                        } else if ([relativePath containsString:@"DefaultDesktop.heic"]) {
                                                            NSLog(@"    ❌ FAILURE: Configuration shows default wallpaper");
                                                            NSLog(@"    ⚠️ This indicates extension system rejection despite configuration success");
                                                        }
                                                    }
                                                }

                                                // Check for placement settings (from private API DesktopPicturePlacement)
                                                NSNumber *placement = [configDict objectForKey:@"placement"];
                                                if (placement) {
                                                    NSInteger placementValue = [placement integerValue];
                                                    NSString *placementDesc = @"Unknown";
                                                    // Based on private API analysis of DesktopPicturePlacement
                                                    switch (placementValue) {
                                                        case 0: placementDesc = @"Centered"; break;
                                                        case 1: placementDesc = @"Tiled"; break;
                                                        case 2: placementDesc = @"Scaled to Fit"; break;
                                                        case 3: placementDesc = @"Scaled to Fill (default)"; break;
                                                        case 4: placementDesc = @"Stretched"; break;
                                                    }
                                                    NSLog(@"    Placement: %@ (%ld)", placementDesc, (long)placementValue);
                                                }

                                                // Check for animation settings (from private API analysis)
                                                NSNumber *animate = [configDict objectForKey:@"animate"];
                                                NSNumber *randomize = [configDict objectForKey:@"randomize"];
                                                if (animate || randomize) {
                                                    NSLog(@"    Animation: %@, Randomize: %@",
                                                          animate ? ([animate boolValue] ? @"YES" : @"NO") : @"Not set",
                                                          randomize ? ([randomize boolValue] ? @"YES" : @"NO") : @"Not set");
                                                }

                                                // Check for background color (from private API analysis)
                                                NSDictionary *bgColorDict = [configDict objectForKey:@"backgroundColor"];
                                                if (bgColorDict) {
                                                    NSLog(@"    Background color configured: %@", bgColorDict);
                                                }

                                            } else {
                                                NSLog(@"    ❌ Failed to decode configuration: %@", plistError ? [plistError localizedDescription] : @"Unknown error");
                                                NSLog(@"    ⚠️ This may indicate corrupted configuration from extension system failure");
                                            }
                                        } else {
                                            NSLog(@"    ⚠️ No configuration data found");
                                        }

                                    } else if ([provider isEqualToString:@"com.apple.wallpaper.choice.color"]) {
                                        NSLog(@"    Color provider (matches private API WallpaperChoiceProviderID.color)");
                                    } else if ([provider hasPrefix:@"com.apple.wallpaper.choice."]) {
                                        NSLog(@"    Other Apple provider: %@", provider);
                                    } else {
                                        NSLog(@"    ⚠️ Unknown provider: %@", provider);
                                    }
                                } else {
                                    NSLog(@"    ⚠️ No choices found in content");
                                }
                            } else {
                                NSLog(@"    ⚠️ No Content found in Desktop");
                            }
                        } else {
                            NSLog(@"    ⚠️ No Desktop found in display config");
                        }
                    }
                } else {
                    NSLog(@"⚠️ No Displays found in Index.plist");
                }
            } else {
                NSLog(@"❌ Failed to read Index.plist dictionary");
            }
        } else {
            NSLog(@"Wallpaper Index.plist not found at: %@", indexPath);
            NSLog(@"This may indicate no wallpapers have been set or extension system failure");
        }

        // Additional analysis based on private API insights
        NSLog(@"===========================================");
        NSLog(@"PRIVATE API INSIGHTS ANALYSIS:");
        NSLog(@"Based on reverse engineering of Wallpaper.framework (primary screen test):");

        // Check if extension system processed our request
        BOOL foundOurImage = NO;
        if ([indexFileManager fileExistsAtPath:indexPath]) {
            NSDictionary *indexDict = [NSDictionary dictionaryWithContentsOfFile:indexPath];
            if (indexDict) {
                NSDictionary *displays = [indexDict objectForKey:@"Displays"];
                for (NSString *displayUUID in displays) {
                    NSDictionary *displayConfig = [displays objectForKey:displayUUID];
                    NSDictionary *desktop = [displayConfig objectForKey:@"Desktop"];
                    if (desktop) {
                        NSDictionary *content = [desktop objectForKey:@"Content"];
                        if (content) {
                            NSArray *choices = [content objectForKey:@"Choices"];
                            if (choices && [choices count] > 0) {
                                NSDictionary *choice = [choices objectAtIndex:0];
                                NSString *provider = [choice objectForKey:@"Provider"];
                                if ([provider isEqualToString:@"com.apple.wallpaper.choice.image"]) {
                                    NSData *configData = [choice objectForKey:@"Configuration"];
                                    if (configData) {
                                        NSError *plistError = nil;
                                        NSDictionary *configDict = [NSPropertyListSerialization propertyListWithData:configData
                                                                                                             options:NSPropertyListImmutable
                                                                                                              format:nil
                                                                                                               error:&plistError];
                                        if (configDict) {
                                            NSDictionary *urlDict = [configDict objectForKey:@"url"];
                                            if (urlDict) {
                                                NSString *relativePath = [urlDict objectForKey:@"relative"];
                                                if ([relativePath containsString:@"hud-251102-2153"]) {
                                                    foundOurImage = YES;
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        if (foundOurImage) {
            NSLog(@"✅ Extension system accepted our configuration");
            NSLog(@"❌ But wallpaper visually failed - indicates extension export/snapshot failure");
            NSLog(@"🔍 Check Console.app for WallpaperExtensionKit.WallpaperExtensionError (3)");
        } else {
            NSLog(@"❌ Extension system rejected our configuration entirely");
            NSLog(@"🔍 This suggests the API call never reached the extension system");
        }
        NSLog(@"===========================================");
        
        // Return error if primary screen failed
        if (successCount == 0) {
            NSLog(@"ERROR: Failed to set wallpaper on primary screen");
            return -1;
        }

        if (failureCount > 0) {
            NSLog(@"WARNING: Failed to set wallpaper on primary screen");
        } else {
            NSLog(@"API SUCCESS: setDesktopImageURL returned YES for primary screen");
            NSLog(@"NOTE: If wallpaper doesn't visually change, this is a macOS Sequoia extension system issue");
        }
        
        return 0;
    }
}

// Get current wallpaper URL for a screen (for verification)
// Returns a malloc'd string that MUST be freed by the caller
char* getCurrentWallpaper(int screenIndex) {
    @autoreleasepool {
        NSArray *screens = [NSScreen screens];
        if (screenIndex < 0 || (NSUInteger)screenIndex >= [screens count]) {
            return strdup("INVALID_SCREEN");
        }
        
        NSScreen *screen = [screens objectAtIndex:screenIndex];
        NSWorkspace *workspace = [NSWorkspace sharedWorkspace];
        NSURL *url = [workspace desktopImageURLForScreen:screen];
        
        if (url) {
            NSString *path = [url path];
            return strdup([path UTF8String]);
        }
        
        return strdup("NONE");
    }
}
*/
import "C"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <image_path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSets the desktop wallpaper on the primary screen only.\n")
		fmt.Fprintf(os.Stderr, "Example: %s /path/to/image.jpg\n", os.Args[0])
		os.Exit(1)
	}

	imagePath := os.Args[1]

	// Log Go-side information
	log.Printf("===========================================")
	log.Printf("Wallpaper Test Program")
	log.Printf("===========================================")
	log.Printf("Image path: %s", imagePath)
	log.Printf("Go version: %s", runtime.Version())
	log.Printf("OS: %s", runtime.GOOS)
	log.Printf("Arch: %s", runtime.GOARCH)
	log.Printf("Goroutines: %d", runtime.NumGoroutine())

	// Verify file exists
	info, err := os.Stat(imagePath)
	if os.IsNotExist(err) {
		log.Fatalf("ERROR: File does not exist: %s", imagePath)
	}
	if err != nil {
		log.Fatalf("ERROR: Failed to stat file: %v", err)
	}
	if info.IsDir() {
		log.Fatalf("ERROR: Path is a directory, not a file: %s", imagePath)
	}

	log.Printf("File verified:")
	log.Printf("  Size: %d bytes", info.Size())
	log.Printf("  Mode: %s", info.Mode())
	log.Printf("  ModTime: %s", info.ModTime())

	// Get absolute path
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		log.Fatalf("ERROR: Failed to get absolute path: %v", err)
	}
	log.Printf("Absolute path: %s", absPath)

	// Skip cache copy test for now - use original path
	log.Printf("===========================================")
	log.Printf("Using original image path (not copying to cache)")
	log.Printf("Path: %s", absPath)

	// Check thread information before calling CGO
	log.Printf("Current goroutine: %d", runtime.NumGoroutine())
	log.Printf("Calling CGO function...")

	// Call Objective-C function via CGO
	cPath := C.CString(absPath)
	defer C.free(unsafe.Pointer(cPath))
	
	isMain := C.isMainThread()
	threadName := C.GoString(C.getCurrentThreadName())
	log.Printf("Thread check (from Objective-C):")
	log.Printf("  Is main thread: %d", isMain)
	log.Printf("  Thread name: %s", threadName)

	log.Printf("Invoking setWallpaperEnhanced...")
	log.Printf("Using path: %s", absPath)
	result := C.setWallpaperEnhanced(cPath)

	log.Printf("===========================================")
	if result != 0 {
		log.Printf("RESULT: FAILED (error code: %d)", result)
		log.Printf("Check Console.app logs for detailed error information")
		os.Exit(1)
	} else {
		log.Printf("RESULT: API call reported SUCCESS")
		
		// CRITICAL: WallpaperAgent processing takes ~32 seconds according to logs
		// The program must stay alive long enough for XPC communication to complete
		// Otherwise we get: "invalidated after the last release of the connection object"
		log.Printf("===========================================")
		log.Printf("IMPORTANT: Keeping process alive for wallpaper extension system")
		log.Printf("Based on testing, WallpaperAgent takes ~32-45 seconds to process")
		log.Printf("===========================================")
		
		// Wait in intervals and check periodically
		checkIntervals := []int{5, 10, 15, 20, 25, 30, 35, 40, 45}
		for i, seconds := range checkIntervals {
			if i == 0 {
				log.Printf("Waiting %d seconds...", seconds)
			} else {
				prevSeconds := checkIntervals[i-1]
				log.Printf("Waiting %d more seconds (total: %d)...", seconds-prevSeconds, seconds)
			}
			
			if i == 0 {
				time.Sleep(time.Duration(seconds) * time.Second)
			} else {
				time.Sleep(time.Duration(seconds-checkIntervals[i-1]) * time.Second)
			}
			
			// Check wallpaper at this interval
			log.Printf("--- Check at %d seconds ---", seconds)
			cWallpaper := C.getCurrentWallpaper(0)
			currentWallpaper := C.GoString(cWallpaper)
			C.free(unsafe.Pointer(cWallpaper))
			log.Printf("Screen 0 wallpaper: %s", currentWallpaper)
			
			// Check if it matches our target
			if currentWallpaper == absPath {
				log.Printf("✓ SUCCESS: Wallpaper matches target image!")
				log.Printf("Total time to apply: %d seconds", seconds)
				break
			} else if currentWallpaper == "/System/Library/CoreServices/DefaultDesktop.heic" {
				log.Printf("⚠ Still showing default wallpaper, continuing to wait...")
			} else {
				log.Printf("⚠ Showing different wallpaper: %s", currentWallpaper)
			}
		}
		
		// Final verification
		log.Printf("===========================================")
		log.Printf("FINAL VERIFICATION:")
		cWallpaper := C.getCurrentWallpaper(0)
		currentWallpaper := C.GoString(cWallpaper)
		C.free(unsafe.Pointer(cWallpaper))
		log.Printf("Screen 0 wallpaper: %s", currentWallpaper)
		
		if currentWallpaper == absPath {
			log.Printf("✓ SUCCESS: Wallpaper successfully set!")
		} else if currentWallpaper == "/System/Library/CoreServices/DefaultDesktop.heic" {
			log.Printf("✗ FAILED: Still showing default wallpaper after 45 seconds")
			log.Printf("This indicates macOS Sequoia/Tahoe extension system bug")
		} else {
			log.Printf("? UNEXPECTED: Showing different wallpaper than expected")
		}
		
		log.Printf("===========================================")
		log.Printf("DIAGNOSIS:")
		log.Printf("Check Console.app logs for detailed error information:")
		log.Printf("  log show --predicate 'subsystem == \"com.apple.wallpaper\"' --last 1m")
		log.Printf("Look for:")
		log.Printf("  - WallpaperExtensionKit.WallpaperExtensionError")
		log.Printf("  - NSCocoaErrorDomain (4099)")
		log.Printf("  - 'Failed to create snapshot to export'")
		log.Printf("  - XPC connection errors")
		os.Exit(0)
	}
}

