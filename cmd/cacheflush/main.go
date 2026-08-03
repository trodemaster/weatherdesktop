// cacheflush is a dedicated binary for the nightly wallpaper cache cleanup LaunchAgent.
//
// It exists as a standalone binary (rather than a shell script run via bash) so that
// Full Disk Access can be granted to this one executable specifically, instead of to
// /bin/bash — which every script on the system shares. See TROUBLESHOOTING.md for the
// TCC investigation that led here: launchd-spawned bash has no path to inherit Full
// Disk Access the way an interactive Terminal session does, so it silently can't see
// or modify files inside other apps' sandboxed Containers.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("cannot determine home directory: %v", err)
	}

	extDomain := filepath.Join(home, "Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Preferences/com.apple.wallpaper.extension.image")
	containersCacheDir := filepath.Join(home, "Library/Containers/com.apple.wallpaper.extension.image/Data/Library/Caches")
	bmpCacheDir := filepath.Join(home, "Library/Containers/com.apple.wallpaper.agent/Data/Library/Caches/com.apple.wallpaper.caches/extension-com.apple.wallpaper.extension.image")

	varCacheDir, err := darwinUserCacheDir()
	if err != nil {
		logf("warning: could not resolve DARWIN_USER_CACHE_DIR: %v", err)
	} else {
		varCacheDir = filepath.Join(varCacheDir, "com.apple.wallpaper.extension.image")
	}

	logf("Starting wallpaper cache cleanup...")

	clearChoiceRequests(extDomain)

	deleteFilesWithExt("Containers", containersCacheDir, ".jpg")
	if varCacheDir != "" {
		deleteFilesWithExt("var/folders", varCacheDir, ".jpg")
	}
	deleteFilesWithExt("BMP", bmpCacheDir, ".bmp")

	logf("Cache cleanup complete.")
}

// clearChoiceRequests restarts cfprefsd (to drop any in-memory write block) and clears
// the WallpaperImageExtension bookmark accumulation that otherwise grows past the 4MB
// NSUserDefaults limit and hangs WallpaperAgent.
func clearChoiceRequests(extDomain string) {
	logf("Clearing ChoiceRequests plist entries...")

	blockedFile := extDomain + ".plist.blocked"
	if _, err := os.Stat(blockedFile); err == nil {
		if err := os.Remove(blockedFile); err != nil {
			logf("  failed to remove .blocked sentinel: %v", err)
		} else {
			logf("  removed .blocked sentinel")
		}
	}

	if out, err := exec.Command("killall", "cfprefsd").CombinedOutput(); err != nil {
		logf("  killall cfprefsd: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	time.Sleep(1 * time.Second)

	for _, key := range []string{"ChoiceRequests.ImageFiles", "ChoiceRequests.Assets", "ChoiceRequests.CollectionIdentifiers"} {
		cmd := exec.Command("defaults", "write", extDomain, key, "-array")
		if out, err := cmd.CombinedOutput(); err != nil {
			logf("  failed to clear %s: %v (%s)", key, err, strings.TrimSpace(string(out)))
		} else {
			logf("  %s cleared", key)
		}
	}
}

// deleteFilesWithExt removes all top-level files matching ext in dir, logging counts
// and any access errors instead of silently swallowing them.
func deleteFilesWithExt(label, dir, ext string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logf("%s cache dir (%s) not accessible: %v", label, dir, err)
		return
	}

	var matched []string
	var totalSize int64
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ext) {
			continue
		}
		matched = append(matched, e.Name())
		if info, err := e.Info(); err == nil {
			totalSize += info.Size()
		}
	}

	logf("Deleting %d %s cache file(s) using %s (%s)...", len(matched), strings.TrimPrefix(ext, "."), humanSize(totalSize), label)

	var failed int
	for _, name := range matched {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			failed++
			logf("  failed to delete %s: %v", name, err)
		}
	}

	logf("  ✓ deleted %d, %d remaining (failed to delete)", len(matched)-failed, failed)
}

func darwinUserCacheDir() (string, error) {
	out, err := exec.Command("getconf", "DARWIN_USER_CACHE_DIR").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func logf(format string, args ...any) {
	fmt.Printf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
}

func fatalf(format string, args ...any) {
	logf("FATAL: "+format, args...)
	os.Exit(1)
}
