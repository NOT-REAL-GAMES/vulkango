package main

import (
	"fmt"
	"image"
	"image/color"
	"log"

	ffmpeggo "github.com/NOT-REAL-GAMES/ffmpeggo"
)

func main() {
	width := 512
	height := 512
	fps := 30.0
	duration := 2 // seconds

	fmt.Println("Creating video encoder...")
	enc, err := ffmpeggo.NewVideoEncoder("/tmp/ffmpeggo_test.mp4", width, height, fps, false)
	if err != nil {
		log.Fatal(err)
	}
	defer enc.Close()

	totalFrames := int(fps) * duration
	fmt.Printf("Encoding %d frames (%dx%d @ %.0f fps)...\n", totalFrames, width, height, fps)

	for i := 0; i < totalFrames; i++ {
		// Create a frame with color gradient animation
		img := createTestFrame(width, height, i, totalFrames)

		if err := enc.EncodeFrame(img); err != nil {
			log.Fatal(err)
		}

		if (i+1)%10 == 0 {
			fmt.Printf("  Encoded frame %d/%d\n", i+1, totalFrames)
		}
	}

	fmt.Printf("Finalizing video...\n")
	if err := enc.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done! Output: /tmp/ffmpeggo_test.mp4\n")
	fmt.Printf("Total frames encoded: %d\n", enc.FrameCount())
}

func createTestFrame(width, height, frame, totalFrames int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Animate hue over time
	hueOffset := float64(frame) / float64(totalFrames)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a color gradient
			hue := (float64(x)/float64(width) + hueOffset)
			if hue >= 1 {
				hue -= 1
			}
			sat := float64(y) / float64(height)
			val := 0.8 + 0.2*float64(y)/float64(height)

			r, g, b := hsvToRGB(hue, sat, val)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

func hsvToRGB(h, s, v float64) (r, g, b uint8) {
	var rf, gf, bf float64

	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)

	switch i % 6 {
	case 0:
		rf, gf, bf = v, t, p
	case 1:
		rf, gf, bf = q, v, p
	case 2:
		rf, gf, bf = p, v, t
	case 3:
		rf, gf, bf = p, q, v
	case 4:
		rf, gf, bf = t, p, v
	case 5:
		rf, gf, bf = v, p, q
	}

	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}
