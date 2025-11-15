package font

import (
	"embed"
	"fmt"
	"math"

	vk "github.com/NOT-REAL-GAMES/vulkango"
)

//go:embed OpenSans_SemiCondensed-Light.ttf
var embeddedFonts embed.FS

// SDFChar holds SDF character metrics for rendering
type SDFChar struct {
	// Atlas coordinates (normalized 0-1)
	U0, V0, U1, V1 float32
	// Character metrics
	Width, Height    int
	XOffset, YOffset int
	XAdvance         int
}

// SDFAtlas holds an SDF font atlas texture and character data
type SDFAtlas struct {
	Width, Height int
	Pixels        []byte
	Chars         map[rune]SDFChar
	FontSize      float32
}

// GenerateSDFAtlas generates an SDF atlas for basic ASCII characters (32-126)
// Uses stb_truetype's built-in SDF generation
func GenerateSDFAtlas(fontData []byte, fontSize float32, padding int, onedgeValue byte, pixelDistScale float32) (*SDFAtlas, error) {
	// Initialize font
	fontInfo, err := vk.InitFont(fontData)
	if err != nil {
		return nil, fmt.Errorf("failed to init font: %v", err)
	}
	defer fontInfo.Free()

	// Calculate scale for desired font size
	scale := fontInfo.ScaleForPixelHeight(fontSize)

	// For MVP: simple grid layout
	// ASCII printable: 32-126 (95 characters)
	firstChar := 32
	numChars := 95

	// Estimate cell size: fontSize + padding on all sides
	cellSize := int(math.Ceil(float64(fontSize))) + padding*2

	// Calculate atlas dimensions (square grid)
	gridSize := int(math.Ceil(math.Sqrt(float64(numChars))))
	atlasWidth := gridSize * cellSize
	atlasHeight := gridSize * cellSize

	// Create atlas bitmap
	atlas := make([]byte, atlasWidth*atlasHeight)

	// Character map
	chars := make(map[rune]SDFChar)

	// Generate SDF for each character
	gridX := 0
	gridY := 0

	for i := 0; i < numChars; i++ {
		codepoint := firstChar + i

		// Generate SDF for this glyph
		// Note: stbtt_GetCodepointSDF allocates memory that we need to free
		sdfBitmap, width, height, xoff, yoff := fontInfo.GetCodepointSDF(
			scale,
			codepoint,
			padding,
			onedgeValue,
			pixelDistScale,
		)

		if sdfBitmap == nil {
			// For whitespace characters (like space), store metrics even without bitmap
			if codepoint == 32 { // Space character
				advanceWidth, leftSideBearing := fontInfo.GetCodepointHMetrics(codepoint)
				_ = leftSideBearing

				chars[rune(codepoint)] = SDFChar{
					U0: 0, V0: 0, U1: 0, V1: 0, // No texture coordinates
					Width:    0,
					Height:   0,
					XOffset:  0,
					YOffset:  0,
					XAdvance: int(float32(advanceWidth) * scale),
				}
			}

			// Move to next grid cell
			gridX++
			if gridX >= gridSize {
				gridX = 0
				gridY++
			}
			continue
		}

		// Calculate position in atlas
		atlasX := gridX * cellSize
		atlasY := gridY * cellSize

		// Copy SDF bitmap to atlas
		for y := 0; y < height && atlasY+y < atlasHeight; y++ {
			for x := 0; x < width && atlasX+x < atlasWidth; x++ {
				srcIdx := y*width + x
				dstIdx := (atlasY+y)*atlasWidth + (atlasX + x)
				atlas[dstIdx] = sdfBitmap[srcIdx]
			}
		}

		// SDF bitmap is already freed inside GetCodepointSDF

		// Store character metrics
		// Calculate normalized UV coordinates
		u0 := float32(atlasX) / float32(atlasWidth)
		v0 := float32(atlasY) / float32(atlasHeight)
		u1 := float32(atlasX+width) / float32(atlasWidth)
		v1 := float32(atlasY+height) / float32(atlasHeight)

		// Get actual horizontal metrics for this character
		advanceWidth, leftSideBearing := fontInfo.GetCodepointHMetrics(codepoint)
		_ = leftSideBearing // May use later for kerning

		chars[rune(codepoint)] = SDFChar{
			U0: u0, V0: v0, U1: u1, V1: v1,
			Width:    width,
			Height:   height,
			XOffset:  xoff,
			YOffset:  yoff,
			XAdvance: int(float32(advanceWidth) * scale), // Scale to font size
		}

		// Move to next grid cell
		gridX++
		if gridX >= gridSize {
			gridX = 0
			gridY++
		}
	}

	return &SDFAtlas{
		Width:    atlasWidth,
		Height:   atlasHeight,
		Pixels:   atlas,
		Chars:    chars,
		FontSize: fontSize,
	}, nil
}

// LoadEmbeddedFont loads the embedded Liberation Mono font
func LoadEmbeddedFont() ([]byte, error) {
	return embeddedFonts.ReadFile("OpenSans_SemiCondensed-Light.ttf")
}
