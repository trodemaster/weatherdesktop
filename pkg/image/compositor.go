package image

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"log"
	"os"

	"github.com/trodemaster/weatherdesktop/pkg/assets"
)

// Compositor handles compositing multiple images into a single output
type Compositor struct {
	manager *assets.Manager
}

// NewCompositor creates a new compositor
func NewCompositor(manager *assets.Manager) *Compositor {
	return &Compositor{
		manager: manager,
	}
}

// Render creates the final composite image
func (c *Compositor) Render(outputPath string) error {
	// Create canvas: 3840x2160 with sky blue background
	canvas := image.NewRGBA(image.Rect(0, 0, 3840, 2160))
	
	// Fill with sky blue color
	skyBlue := color.RGBA{135, 206, 235, 255} // RGB(135, 206, 235)
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{skyBlue}, image.Point{}, draw.Src)
	
	// Get composite layout
	layers := c.manager.GetCompositeLayout()
	
	// Composite each layer
	for _, layer := range layers {
		if err := c.compositeLayer(canvas, layer); err != nil {
			log.Printf("Warning: Failed to composite %s: %v", layer.ImagePath, err)
			// Continue with other layers even if one fails
		}
	}
	
	// Save the final composite
	if err := c.saveJPEG(canvas, outputPath); err != nil {
		return fmt.Errorf("failed to save composite: %w", err)
	}
	
	log.Printf("Composite image saved to %s", outputPath)
	return nil
}

// compositeLayer adds a single layer to the canvas
func (c *Compositor) compositeLayer(canvas *image.RGBA, layer assets.CompositeLayer) error {
	// Check if file exists
	if _, err := os.Stat(layer.ImagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file not found: %s", layer.ImagePath)
	}
	
	// Load the layer image
	layerImg, err := LoadImageForComposite(layer.ImagePath)
	if err != nil {
		return fmt.Errorf("failed to load layer image: %w", err)
	}
	
	// Calculate destination rectangle
	bounds := layerImg.Bounds()
	destRect := image.Rectangle{
		Min: layer.Position,
		Max: layer.Position.Add(image.Point{X: bounds.Dx(), Y: bounds.Dy()}),
	}
	
	// Composite the image onto the canvas using Over operation (alpha blending)
	draw.Draw(canvas, destRect, layerImg, bounds.Min, draw.Over)
	
	log.Printf("Composited %s at position (%d, %d)", layer.ImagePath, layer.Position.X, layer.Position.Y)
	return nil
}

// saveJPEG saves an image as JPEG with high quality
func (c *Compositor) saveJPEG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	
	opts := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(f, img, opts); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}
	
	return nil
}

