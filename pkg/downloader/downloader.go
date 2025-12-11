package downloader

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
)

// Downloader handles HTTP downloads with retry logic
type Downloader struct {
	client  *http.Client
	manager *assets.Manager
}

// New creates a new downloader
func New(manager *assets.Manager) *Downloader {
	// Always load CA certificates from bundle file
	// SystemCertPool() may not work properly in containers without CGO
	systemCertPool := x509.NewCertPool()
	
	// Try to load from standard Ubuntu/Debian location first
	certPaths := []string{
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
		"/etc/ssl/ca-bundle.pem",
	}
	
	loaded := false
	for _, path := range certPaths {
		if certs, err := os.ReadFile(path); err == nil {
			if systemCertPool.AppendCertsFromPEM(certs) {
				log.Printf("Loaded CA certificates from %s", path)
				loaded = true
				break
			}
		}
	}
	
	// Fallback to SystemCertPool if direct loading failed
	if !loaded {
		if pool, err := x509.SystemCertPool(); err == nil && pool != nil {
			systemCertPool = pool
			log.Printf("Using system cert pool")
		} else {
			log.Printf("Warning: Failed to load CA certificates")
		}
	}
	
	// Create HTTP client with system certificates
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            systemCertPool,
			InsecureSkipVerify: false,
		},
	}
	
	return &Downloader{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		manager: manager,
	}
}

// DownloadAll downloads all configured assets concurrently
func (d *Downloader) DownloadAll() error {
	downloadTargets := d.manager.GetDownloadTargets()

	var wg sync.WaitGroup
	errorsChan := make(chan error, len(downloadTargets))

	for _, target := range downloadTargets {
		wg.Add(1)
		go func(t assets.DownloadTarget) {
			defer wg.Done()

			log.Printf("Downloading %s from %s", t.Name, t.URL)

			if err := d.downloadWithRetry(t.URL, t.OutputPath, 3); err != nil {
				log.Printf("Failed to download %s: %v, creating fallback image", t.Name, err)
				if err := d.createFallbackImage(t.OutputPath); err != nil {
					errorsChan <- fmt.Errorf("failed to create fallback for %s: %w", t.Name, err)
					return
				}
			}
		}(target)
	}
	
	wg.Wait()
	close(errorsChan)
	
	// Collect any errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("download errors: %v", errors)
	}
	
	return nil
}

// downloadWithRetry attempts to download a file with retry logic
func (d *Downloader) downloadWithRetry(url, destPath string, maxRetries int) error {
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(time.Second * time.Duration(attempt))
			log.Printf("Retry attempt %d for %s", attempt+1, url)
		}
		
		err := d.download(url, destPath)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// download performs a single HTTP download
func (d *Downloader) download(url, destPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Create output file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()
	
	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// createFallbackImage creates a 1x1 transparent PNG as fallback
func (d *Downloader) createFallbackImage(destPath string) error {
	// Create a 1x1 transparent image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{0, 0, 0, 0}) // Transparent
	
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create fallback file: %w", err)
	}
	defer out.Close()
	
	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("failed to encode fallback image: %w", err)
	}
	
	log.Printf("Created 1x1 transparent fallback image at %s", destPath)
	return nil
}

