package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// winterImagePath matches the image field of latest.json. The name becomes a local
// filename, so anything else is rejected.
var winterImagePath = regexp.MustCompile(`^/img/hud-[0-9]{6}-[0-9]{4}\.jpg$`)

// winterLatest is the latest.json published by the winter.jibb.tv generator.
type winterLatest struct {
	Image     string `json:"image"`
	Generated string `json:"generated"`
}

// fetchFromServer is the client half of the darkstar split: instead of rendering
// locally, poll <baseURL>/latest.json and, when the image changed, download it into
// rendered/ and set it as the wallpaper. Image URLs are immutable, so the local
// filename doubles as the cache key.
//
// A server that can't be reached is not an error: the current wallpaper stays and the
// next run tries again. Only a failed wallpaper set returns an error. Downloaded
// images accumulate in rendered/ and are never deleted (they are kept for making movies).
func fetchFromServer(baseURL string, scriptDir string) error {
	baseURL = strings.TrimSuffix(baseURL, "/")
	client := &http.Client{Timeout: 60 * time.Second}

	latest, err := fetchLatest(client, baseURL)
	if err != nil {
		log.Printf("Fetch: Warning: %v (keeping current wallpaper)", err)
		return nil
	}

	renderedDir := filepath.Join(scriptDir, "rendered")
	if err := os.MkdirAll(renderedDir, 0755); err != nil {
		return fmt.Errorf("failed to create rendered directory: %w", err)
	}

	name := path.Base(latest.Image)
	dest := filepath.Join(renderedDir, name)
	if _, err := os.Stat(dest); err == nil {
		log.Printf("Fetch: %s is current (generated %s), nothing to do", name, latest.Generated)
		return nil
	}

	log.Printf("Fetch: Downloading %s%s", baseURL, latest.Image)
	if err := downloadImage(client, baseURL+latest.Image, dest); err != nil {
		log.Printf("Fetch: Warning: %v (keeping current wallpaper)", err)
		return nil
	}

	log.Printf("Setting desktop wallpaper on all screens: %s", dest)
	if err := setDesktopWallpaper(dest, *desktopMethodFlag, *clearCacheFlag); err != nil {
		// The file's presence means "already set", so drop it and let the next run retry.
		os.Remove(dest)
		return err
	}
	log.Println("✓ Desktop wallpaper set successfully on all screens")
	return nil
}

func fetchLatest(client *http.Client, baseURL string) (*winterLatest, error) {
	// latest.json is the only mutable URL; ask any proxy in the way (off-LAN) to
	// revalidate. Image URLs are unique per render and need no such header.
	req, err := http.NewRequest(http.MethodGet, baseURL+"/latest.json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build latest.json request: %w", err)
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest.json: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("latest.json returned %s", resp.Status)
	}

	var latest winterLatest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&latest); err != nil {
		return nil, fmt.Errorf("failed to parse latest.json: %w", err)
	}
	if !winterImagePath.MatchString(latest.Image) {
		return nil, fmt.Errorf("unexpected image path in latest.json: %q", latest.Image)
	}

	if *debugFlag {
		log.Printf("Fetch: latest.json image=%s generated=%s", latest.Image, latest.Generated)
	}
	return &latest, nil
}

// downloadImage writes url to dest via a temp file and rename, so the wallpaper is
// never set from a partial download.
func downloadImage(client *http.Client, url string, dest string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download returned %s", resp.Status)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/jpeg") {
		return fmt.Errorf("image download returned Content-Type %q", ct)
	}

	tmp := filepath.Join(filepath.Dir(dest), "."+filepath.Base(dest)+".tmp")
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", tmp, err)
	}
	n, err := io.Copy(f, resp.Body)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil && n == 0 {
		err = fmt.Errorf("empty response")
	}
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to download image: %w", err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to move image into place: %w", err)
	}

	if *debugFlag {
		log.Printf("Fetch: Wrote %s (%d bytes)", dest, n)
	}
	return nil
}
