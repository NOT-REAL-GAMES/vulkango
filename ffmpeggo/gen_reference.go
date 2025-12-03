// +build ignore

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	width := 320
	height := 240
	frameNum := 15 // Middle frame

	// Create the same pattern used in the encoder test
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x + frameNum*3) % 256)
			g := uint8((y + frameNum*2) % 256)
			b := uint8((x + y + frameNum) % 256)
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}

	f, err := os.Create("/tmp/reference_frame.png")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	png.Encode(f, img)
	fmt.Println("Reference frame saved to /tmp/reference_frame.png")
}
