// Package ffmpeggo provides pure Go audio/video encoding/decoding.
// This is a Go port of core FFmpeg functionality.
package ffmpeggo

import (
	"image"
	"image/color"

	"github.com/NOT-REAL-GAMES/ffmpeggo/avcodec"
	"github.com/NOT-REAL-GAMES/ffmpeggo/avformat"
	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// VideoEncoder provides a high-level interface for video encoding.
type VideoEncoder struct {
	formatCtx   *avformat.FormatContext
	muxer       *avformat.MP4Muxer
	encoder     *avcodec.HEVCEncoder
	encoderCtx  *avcodec.EncoderContext

	width       int
	height      int
	frameRate   avutil.Rational
	frameCount  int64
}

// NewVideoEncoder creates a new video encoder.
func NewVideoEncoder(outputPath string, width, height int, fps float64) (*VideoEncoder, error) {
	// Open output file
	formatCtx, err := avformat.OpenOutput(outputPath)
	if err != nil {
		return nil, err
	}

	// Create muxer
	muxer := avformat.NewMP4Muxer()

	// Add video stream
	stream := formatCtx.AddStream()
	stream.CodecType = avutil.MediaTypeVideo
	stream.Width = width
	stream.Height = height
	stream.TimeBase = avutil.Rational{1, 90000}
	stream.FrameRate = avutil.Rational{int(fps * 1000), 1000}

	// Create encoder
	encoder := avcodec.NewHEVCEncoder()

	// Initialize encoder context
	encoderCtx := &avcodec.EncoderContext{
		Width:     width,
		Height:    height,
		PixFmt:    avutil.PixFmtYUV420P,
		TimeBase:  avutil.Rational{1, 90000},
		Framerate: avutil.Rational{int(fps * 1000), 1000},
		GopSize:   30, // Keyframe every 30 frames
		QMin:      20, // Quality (lower = better)
	}

	if err := encoder.Init(encoderCtx); err != nil {
		formatCtx.Close()
		return nil, err
	}

	// Set codec data on stream
	stream.CodecData = encoderCtx.ExtraData

	// Write header
	if err := muxer.WriteHeader(formatCtx); err != nil {
		formatCtx.Close()
		return nil, err
	}

	return &VideoEncoder{
		formatCtx:  formatCtx,
		muxer:      muxer,
		encoder:    encoder,
		encoderCtx: encoderCtx,
		width:      width,
		height:     height,
		frameRate:  encoderCtx.Framerate,
	}, nil
}

// EncodeFrame encodes an RGBA image as a video frame.
func (e *VideoEncoder) EncodeFrame(img image.Image) error {
	// Convert image to YUV420P
	frame := imageToYUV420P(img, e.width, e.height)
	frame.Pts = e.frameCount
	frame.PktDts = e.frameCount
	frame.KeyFrame = e.frameCount%int64(e.encoderCtx.GopSize) == 0
	if frame.KeyFrame {
		frame.PictType = avutil.PictureTypeI
	} else {
		frame.PictType = avutil.PictureTypeP
	}

	// Encode
	packets, err := e.encoder.Encode(e.encoderCtx, frame)
	if err != nil {
		return err
	}

	// Write packets
	for _, pkt := range packets {
		if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
			return err
		}
	}

	e.frameCount++
	return nil
}

// EncodeRGBA encodes raw RGBA pixel data as a video frame.
func (e *VideoEncoder) EncodeRGBA(rgba []byte) error {
	// Create RGBA image from bytes
	img := &image.RGBA{
		Pix:    rgba,
		Stride: e.width * 4,
		Rect:   image.Rect(0, 0, e.width, e.height),
	}
	return e.EncodeFrame(img)
}

// Close finalizes the video and closes the output file.
func (e *VideoEncoder) Close() error {
	// Flush encoder
	packets, err := e.encoder.Flush(e.encoderCtx)
	if err != nil {
		return err
	}

	for _, pkt := range packets {
		if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
			return err
		}
	}

	// Write trailer
	if err := e.muxer.WriteTrailer(e.formatCtx); err != nil {
		return err
	}

	// Close encoder
	e.encoder.Close(e.encoderCtx)

	// Close file
	return e.formatCtx.Close()
}

// FrameCount returns the number of frames encoded.
func (e *VideoEncoder) FrameCount() int64 {
	return e.frameCount
}

// imageToYUV420P converts an image to YUV420P format.
func imageToYUV420P(img image.Image, width, height int) *avutil.Frame {
	frame := avutil.NewFrame()
	frame.Width = width
	frame.Height = height
	frame.Format = int(avutil.PixFmtYUV420P)
	frame.AllocBuffer()

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Y plane
	for y := 0; y < height; y++ {
		srcY := y * srcHeight / height
		for x := 0; x < width; x++ {
			srcX := x * srcWidth / width
			r, g, b, _ := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()

			// Convert to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// RGB to Y (BT.601)
			yVal := (66*int(r8) + 129*int(g8) + 25*int(b8) + 128) >> 8
			yVal += 16
			if yVal < 16 {
				yVal = 16
			} else if yVal > 235 {
				yVal = 235
			}

			frame.Data[0][y*frame.Linesize[0]+x] = byte(yVal)
		}
	}

	// U and V planes (subsampled 2x2)
	chromaWidth := width / 2
	chromaHeight := height / 2

	for cy := 0; cy < chromaHeight; cy++ {
		srcY := (cy * 2) * srcHeight / height
		for cx := 0; cx < chromaWidth; cx++ {
			srcX := (cx * 2) * srcWidth / width

			// Sample 2x2 block and average
			var rSum, gSum, bSum int
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					px := bounds.Min.X + min(srcX+dx, srcWidth-1)
					py := bounds.Min.Y + min(srcY+dy, srcHeight-1)
					r, g, b, _ := img.At(px, py).RGBA()
					rSum += int(r >> 8)
					gSum += int(g >> 8)
					bSum += int(b >> 8)
				}
			}

			r8 := rSum / 4
			g8 := gSum / 4
			b8 := bSum / 4

			// RGB to U (Cb)
			uVal := (-38*r8 - 74*g8 + 112*b8 + 128) >> 8
			uVal += 128
			if uVal < 16 {
				uVal = 16
			} else if uVal > 240 {
				uVal = 240
			}

			// RGB to V (Cr)
			vVal := (112*r8 - 94*g8 - 18*b8 + 128) >> 8
			vVal += 128
			if vVal < 16 {
				vVal = 16
			} else if vVal > 240 {
				vVal = 240
			}

			frame.Data[1][cy*frame.Linesize[1]+cx] = byte(uVal)
			frame.Data[2][cy*frame.Linesize[2]+cx] = byte(vVal)
		}
	}

	return frame
}

// RGBAToYUV420P converts RGBA bytes to YUV420P bytes.
func RGBAToYUV420P(rgba []byte, width, height int) (y, u, v []byte) {
	img := &image.RGBA{
		Pix:    rgba,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	frame := imageToYUV420P(img, width, height)
	return frame.Data[0], frame.Data[1], frame.Data[2]
}

// Helper to create solid color frame (for testing)
func CreateSolidColorFrame(width, height int, c color.Color) *avutil.Frame {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return imageToYUV420P(img, width, height)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
