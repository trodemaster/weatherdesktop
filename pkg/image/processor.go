package image

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
	"golang.org/x/image/draw"
)

// Processor handles image cropping and resizing
type Processor struct {
	manager *assets.Manager
}

// NewProcessor creates a new image processor
func NewProcessor(manager *assets.Manager) *Processor {
	return &Processor{
		manager: manager,
	}
}

// ProcessAll crops and resizes all configured assets
func (p *Processor) ProcessAll() error {
	cropAssets := p.manager.GetCropAssets()
	
	for _, asset := range cropAssets {
		log.Printf("Processing %s", asset.Name)
		
		if err := p.processAsset(asset); err != nil {
			log.Printf("Failed to process %s: %v", asset.Name, err)
			// Continue with other assets even if one fails
		}
	}
	
	return nil
}

// processAsset crops and/or resizes a single asset
func (p *Processor) processAsset(asset assets.Asset) error {
	// Load source image
	img, err := p.loadImage(asset.InputPath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	// Crop to specified rectangle
	img = p.crop(img, asset.CropRect)

	// Resize to target size
	img = p.resize(img, asset.TargetSize)

	// Save processed image
	if err := p.saveImage(img, asset.OutputPath); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	log.Printf("Saved processed image to %s", asset.OutputPath)
	return nil
}

// loadImage loads an image from file
func (p *Processor) loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	
	// Decode image (supports JPEG, PNG, etc.)
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	return img, nil
}

// crop crops an image to the specified rectangle
func (p *Processor) crop(img image.Image, cropRect image.Rectangle) image.Image {
	// Use SubImage if available (most image types support it)
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := img.(subImager); ok {
		return si.SubImage(cropRect)
	}

	// Fallback: manually copy pixels (should rarely be needed)
	width := cropRect.Dx()
	height := cropRect.Dy()
	cropped := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(cropped, cropped.Bounds(), img, cropRect.Min, draw.Src)
	return cropped
}

// resize resizes an image to the target size
func (p *Processor) resize(img image.Image, targetSize image.Point) image.Image {
	bounds := img.Bounds()
	newWidth := targetSize.X
	newHeight := targetSize.Y

	// If target size is zero, return original image
	if newWidth == 0 && newHeight == 0 {
		return img
	}

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Use CatmullRom interpolation for high-quality resizing
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

// saveImage saves an image to file as JPEG
func (p *Processor) saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	
	// Determine output format based on extension
	// For now, default to JPEG with quality 90
	opts := &jpeg.Options{Quality: 90}
	
	if err := jpeg.Encode(f, img, opts); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}
	
	return nil
}

// LoadImageForComposite loads an image for compositing (with error handling)
func LoadImageForComposite(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	
	// Try to decode as any supported format
	img, format, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", format, err)
	}
	
	return img, nil
}

