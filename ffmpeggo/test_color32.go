//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/NOT-REAL-GAMES/ffmpeggo/avcodec"
	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

func main() {
	// Test multiple Y values to understand residual encoding
	testValues := []struct {
		name string
		y    byte
	}{
		{"black", 0},
		{"dark", 64},
		{"midgray", 128},
		{"light", 192},
		{"white", 255},
		{"red76", 76},
	}

	for _, tv := range testValues {
		fmt.Printf("\n=== Testing %s (Y=%d) ===\n", tv.name, tv.y)
		testYValue(tv.name, tv.y)
	}
}

func testYValue(name string, yVal byte) {
	frame := &avutil.Frame{
		Width:  32,
		Height: 32,
		Format: 0, // YUV420P
	}

	// Y plane
	ySize := 32 * 32
	frame.Data[0] = make([]byte, ySize)
	for i := range frame.Data[0] {
		frame.Data[0][i] = yVal
	}
	frame.Linesize[0] = 32

	// U and V planes (neutral gray)
	uvSize := 16 * 16
	frame.Data[1] = make([]byte, uvSize)
	frame.Data[2] = make([]byte, uvSize)
	for i := 0; i < uvSize; i++ {
		frame.Data[1][i] = 128 // Neutral U
		frame.Data[2][i] = 128 // Neutral V
	}
	frame.Linesize[1] = 16
	frame.Linesize[2] = 16

	// Create encoder
	encoder := avcodec.NewHEVCEncoder()
	ctx := &avcodec.EncoderContext{
		Width:     32,
		Height:    32,
		Framerate: avutil.Rational{Num: 30, Den: 1},
		QMin:      0,
	}

	err := encoder.Init(ctx)
	if err != nil {
		fmt.Printf("Init error: %v\n", err)
		return
	}

	// Encode one frame
	packets, err := encoder.Encode(ctx, frame)
	if err != nil {
		fmt.Printf("Encode error: %v\n", err)
		return
	}

	// Write raw H.265 bitstream
	h265File := fmt.Sprintf("/tmp/test_%s.h265", name)
	f, _ := os.Create(h265File)
	f.Write(ctx.ExtraData)
	for _, pkt := range packets {
		f.Write(pkt.Data)
	}
	f.Close()

	// Decode to raw Y values
	yuvFile := fmt.Sprintf("/tmp/test_%s.yuv", name)
	cmd := exec.Command("ffmpeg", "-y", "-i", h265File, "-pix_fmt", "gray", "-f", "rawvideo", yuvFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("FFmpeg error: %s\n", output)
		return
	}

	// Read decoded Y value
	data, _ := os.ReadFile(yuvFile)
	if len(data) > 0 {
		avgY := int(data[0]) // All pixels should be same for flat color
		fmt.Printf("Input Y=%d -> Decoded Y=%d (diff=%d)\n", yVal, avgY, avgY-int(yVal))
	}
}
