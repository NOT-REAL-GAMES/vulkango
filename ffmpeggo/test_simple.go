// +build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"os"

	"github.com/NOT-REAL-GAMES/ffmpeggo"
)

func main() {
	width := 64
	height := 64
	fps := 30.0
	frames := 10

	outputPath := "/tmp/test_simple.mp4"

	encoder, err := ffmpeggo.NewVideoEncoder(outputPath, width, height, fps, false)
	if err != nil {
		fmt.Printf("Failed to create encoder: %v\n", err)
		os.Exit(1)
	}
	defer encoder.Close()

	fmt.Printf("Encoding %d simple frames at %dx%d\n", frames, width, height)

	// Create simple solid color frames - red, green, blue, etc.
	colors := []color.RGBA{
		{255, 0, 0, 255},   // Red
		{0, 255, 0, 255},   // Green
		{0, 0, 255, 255},   // Blue
		{255, 255, 0, 255}, // Yellow
		{255, 0, 255, 255}, // Magenta
		{0, 255, 255, 255}, // Cyan
		{255, 255, 255, 255}, // White
		{128, 128, 128, 255}, // Gray
		{0, 0, 0, 255},     // Black
		{128, 64, 192, 255}, // Purple
	}

	for i := 0; i < frames; i++ {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		c := colors[i % len(colors)]
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.SetRGBA(x, y, c)
			}
		}

		err = encoder.EncodeFrame(img)
		if err != nil {
			fmt.Printf("Failed to encode frame %d: %v\n", i, err)
			os.Exit(1)
		}
		fmt.Printf("Frame %d: %s\n", i, colorName(c))
	}

	fmt.Println("Done!")
}

func colorName(c color.RGBA) string {
	switch {
	case c.R == 255 && c.G == 0 && c.B == 0:
		return "Red"
	case c.R == 0 && c.G == 255 && c.B == 0:
		return "Green"
	case c.R == 0 && c.G == 0 && c.B == 255:
		return "Blue"
	case c.R == 255 && c.G == 255 && c.B == 0:
		return "Yellow"
	case c.R == 255 && c.G == 0 && c.B == 255:
		return "Magenta"
	case c.R == 0 && c.G == 255 && c.B == 255:
		return "Cyan"
	case c.R == 255 && c.G == 255 && c.B == 255:
		return "White"
	case c.R == 0 && c.G == 0 && c.B == 0:
		return "Black"
	default:
		return "Other"
	}
}
