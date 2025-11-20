package image

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/trodemaster/weatherdesktop/pkg/parser"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TextRenderer renders text to images
type TextRenderer struct {
	boldFace font.Face
}

// NewTextRenderer creates a new text renderer
func NewTextRenderer() *TextRenderer {
	tr := &TextRenderer{}
	
	// Try to load OpenType font, fallback to basicfont if not available
	if face, err := loadFont("fonts/Roboto-Bold.ttf", 20); err == nil {
		tr.boldFace = face
	} else {
		// Fallback to basicfont if font file not found
		// This allows the code to work even if font file isn't included
		fmt.Printf("Warning: Could not load font: %v, using basicfont fallback\n", err)
		tr.boldFace = nil
	}
	
	return tr
}

// loadFont loads an OpenType font file and creates a font face
func loadFont(fontPath string, size float64) (font.Face, error) {
	// Try current directory first, then check common locations
	paths := []string{
		fontPath,
		filepath.Join("/app", fontPath),
		filepath.Join(".", fontPath),
	}
	
	var fontData []byte
	var err error
	for _, path := range paths {
		if fontData, err = os.ReadFile(path); err == nil {
			break
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("could not read font file: %w", err)
	}
	
	tt, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("could not parse font: %w", err)
	}
	
	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create font face: %w", err)
	}
	
	return face, nil
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

// RenderPassStatus creates a pass status graphic showing East/West status
// Shows visual indicators for Open (black) and Closed (red) status
func (tr *TextRenderer) RenderPassStatus(status *parser.PassStatus, width, height int, outputPath string) error {
	// Use OpenType font if available, otherwise fallback to basicfont
	var face font.Face
	if tr.boldFace != nil {
		face = tr.boldFace
	} else {
		// Fallback to basicfont (no bold variant available in basicfont)
		face = basicfont.Face7x13
	}
	
	// Helper function to draw text normally
	drawText := func(d *font.Drawer, x, y int, text string, col color.Color) {
		d.Src = image.NewUniform(col)
		d.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
		d.DrawString(text)
	}
	
	// First pass: measure all text to determine bounds
	d := &font.Drawer{Face: face}
	
	// Helper function to determine if status is closed
	isEastClosed := strings.Contains(status.East, "Closed")
	isWestClosed := strings.Contains(status.West, "Closed")
	
	title := "Stevens Pass Status"
	eastLabel := "East:"
	eastStatus := "Open"
	if isEastClosed {
		eastStatus = "Closed"
	}
	westLabel := "West:"
	westStatus := "Open"
	if isWestClosed {
		westStatus = "Closed"
	}
	
	// Measure text widths
	titleWidth := d.MeasureString(title).Ceil()
	eastLabelWidth := d.MeasureString(eastLabel).Ceil()
	eastStatusWidth := d.MeasureString(eastStatus).Ceil()
	westLabelWidth := d.MeasureString(westLabel).Ceil()
	westStatusWidth := d.MeasureString(westStatus).Ceil()
	
	// Calculate max width
	maxWidth := titleWidth
	if eastLabelWidth+eastStatusWidth+10 > maxWidth {
		maxWidth = eastLabelWidth + eastStatusWidth + 10
	}
	if westLabelWidth+westStatusWidth+10 > maxWidth {
		maxWidth = westLabelWidth + westStatusWidth + 10
	}
	
	// Measure text height
	metrics := face.Metrics()
	lineHeight := (metrics.Height + metrics.Ascent).Ceil()
	// For OpenType fonts, the metrics already provide good spacing, no need to scale
	if tr.boldFace == nil {
		// Only scale for basicfont
		lineHeight = int(float64(lineHeight) * 1.5)
	}
	
	// Calculate content dimensions with tight padding
	padding := 6 // Tight padding around text
	contentWidth := maxWidth + padding*2
	contentHeight := lineHeight*3 + padding*2 + 4 // Title + 2 status lines + spacing
	
	// Add space for conditions if closed
	if status.IsClosed && status.Conditions != "" {
		conditionsWidth := contentWidth - padding*2
		lines := tr.wordWrap(status.Conditions, conditionsWidth, face)
		contentHeight += len(lines) * lineHeight
	}
	
	// Create image with calculated dimensions
	img := image.NewRGBA(image.Rect(0, 0, contentWidth, contentHeight))
	
	// Fill with white background at 50% transparency (alpha = 128)
	semiTransparentWhite := color.RGBA{255, 255, 255, 128}
	for y := 0; y < contentHeight; y++ {
		for x := 0; x < contentWidth; x++ {
			img.Set(x, y, semiTransparentWhite)
		}
	}
	
	// Set up font drawer for rendering
	d = &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: face,
		Dot:  fixed.Point26_6{},
	}
	
	// Define colors
	red := color.RGBA{200, 0, 0, 255}      // Red for "Closed"
	black := color.RGBA{0, 0, 0, 255}     // Black for "Open" and labels
	
	// Title (centered, bold)
	titleX := (contentWidth - titleWidth) / 2
	titleY := padding + lineHeight
	drawText(d, titleX, titleY, title, black)
	
	// East direction status (bold)
	eastY := titleY + lineHeight + 8
	eastLabelX := padding
	statusColor := black
	if isEastClosed {
		statusColor = red
	}
	
	drawText(d, eastLabelX, eastY, eastLabel, black)
	eastStatusX := eastLabelX + eastLabelWidth + 8
	drawText(d, eastStatusX, eastY, eastStatus, statusColor)
	
	// West direction status (bold)
	westY := eastY + lineHeight
	westLabelX := padding
	statusColor = black
	if isWestClosed {
		statusColor = red
	}
	
	drawText(d, westLabelX, westY, westLabel, black)
	westStatusX := westLabelX + westLabelWidth + 8
	drawText(d, westStatusX, westY, westStatus, statusColor)
	
	// Conditions text if closed (word wrapped)
	if status.IsClosed && status.Conditions != "" {
		conditionsY := westY + lineHeight + 5
		conditionsWidth := contentWidth - padding*2
		
		// Word wrap the conditions text
		lines := tr.wordWrap(status.Conditions, conditionsWidth, face)
		
		// Draw each line
		d.Src = image.NewUniform(black)
		for i, line := range lines {
			if conditionsY+i*lineHeight > contentHeight-padding {
				break // Don't draw beyond image bounds
			}
			lineWidth := d.MeasureString(line).Ceil()
			lineX := (contentWidth - lineWidth) / 2
			if lineX < padding {
				lineX = padding
			}
			d.Dot = fixed.Point26_6{
				X: fixed.I(lineX),
				Y: fixed.I(conditionsY + i*lineHeight),
			}
			d.DrawString(line)
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
