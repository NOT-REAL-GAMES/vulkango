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
	width := 320
	height := 240
	fps := 30.0
	frames := 30

	outputPath := "/tmp/test_transform.mp4"

	encoder, err := ffmpeggo.NewVideoEncoder(outputPath, width, height, fps, true)
	if err != nil {
		fmt.Printf("Failed to create encoder: %v\n", err)
		os.Exit(1)
	}
	defer encoder.Close()

	fmt.Printf("Encoding %d frames at %dx%d @ %.1f fps\n", frames, width, height, fps)

	// Create test frames with a gradient pattern
	for i := 0; i < frames; i++ {
		// Create RGBA image
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		// Create a moving gradient pattern
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// Gradient with animation
				r := uint8((x + i*3) % 256)
				g := uint8((y + i*2) % 256)
				b := uint8((x + y + i) % 256)
				
				// Alpha gradient
				a := uint8((x * 255) / width)
				
				img.SetRGBA(x, y, color.RGBA{r, g, b, a})
			}
		}

		err = encoder.EncodeFrame(img)
		if err != nil {
			fmt.Printf("Failed to encode frame %d: %v\n", i, err)
			os.Exit(1)
		}

		if (i+1)%10 == 0 {
			fmt.Printf("Encoded %d/%d frames\n", i+1, frames)
		}
	}

	fmt.Println("Encoding complete!")
	fmt.Printf("Output: %s\n", outputPath)
}
