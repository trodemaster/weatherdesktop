package image

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextRenderer renders text to images
type TextRenderer struct{}

// NewTextRenderer creates a new text renderer
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

// RenderCaption creates an image with text (like ImageMagick's caption:)
func (tr *TextRenderer) RenderCaption(text string, width, height int, outputPath string) error {
	// Create a white background image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with white
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, white)
		}
	}
	
	// Draw 5px white border (fill will show through)
	// Actually, we want the background to be white, so the "border" is just maintaining white edges
	
	// Set up font drawer
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13, // Basic built-in font
		Dot:  fixed.Point26_6{},
	}
	
	// Word wrap the text to fit within bounds
	lines := tr.wordWrap(text, width-10, d.Face) // 5px margin on each side
	
	// Calculate starting Y position to center text vertically
	lineHeight := 16 // Approximate line height for basicfont
	totalHeight := len(lines) * lineHeight
	startY := (height - totalHeight) / 2
	if startY < 10 {
		startY = 10 // Minimum top margin
	}
	
	// Draw each line
	y := startY
	for _, line := range lines {
		// Center text horizontally
		lineWidth := d.MeasureString(line).Ceil()
		x := (width - lineWidth) / 2
		if x < 5 {
			x = 5 // Minimum left margin
		}
		
		d.Dot = fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y),
		}
		d.DrawString(line)
		y += lineHeight
	}
	
	// Save as PNG
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}
	
	return nil
}

// wordWrap wraps text to fit within maxWidth
func (tr *TextRenderer) wordWrap(text string, maxWidth int, face font.Face) []string {
	words := strings.Fields(text) // Split on whitespace
	if len(words) == 0 {
		return []string{}
	}
	
	var lines []string
	var currentLine strings.Builder
	
	d := &font.Drawer{Face: face}
	
	for i, word := range words {
		// Try adding the word to the current line
		testLine := currentLine.String()
		if testLine != "" {
			testLine += " "
		}
		testLine += word
		
		// Measure the test line
		width := d.MeasureString(testLine).Ceil()
		
		if width <= maxWidth {
			// Word fits, add it
			if currentLine.Len() > 0 {
				currentLine.WriteString(" ")
			}
			currentLine.WriteString(word)
		} else {
			// Word doesn't fit
			if currentLine.Len() == 0 {
				// Single word is too long, add it anyway
				currentLine.WriteString(word)
			} else {
				// Save current line and start new one
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
		
		// Last word
		if i == len(words)-1 && currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
		}
	}
	
	return lines
}

// CreateEmptyImage creates a transparent empty image (for when pass is open)
func CreateEmptyImage(width, height int, outputPath string) error {
	// Create a transparent image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with transparency (already zero-valued, but explicit)
	transparent := color.RGBA{0, 0, 0, 0}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, transparent)
		}
	}
	
	// Save as PNG
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()
	
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}
	
	return nil
}

