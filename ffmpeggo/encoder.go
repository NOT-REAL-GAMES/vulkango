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
	formatCtx  *avformat.FormatContext
	muxer      *avformat.MP4Muxer
	
	// Main encoder (Luma/Chroma)
	encoder    *avcodec.HEVCEncoder
	encoderCtx *avcodec.EncoderContext

	// Alpha encoder (optional)
	alphaEncoder    *avcodec.HEVCEncoder
	alphaEncoderCtx *avcodec.EncoderContext
	hasAlpha        bool

	width      int
	height     int
	frameRate  avutil.Rational
	frameCount int64
}

// NewVideoEncoder creates a new video encoder.
// If hasAlpha is true, it enables alpha channel encoding.
func NewVideoEncoder(outputPath string, width, height int, fps float64, hasAlpha bool) (*VideoEncoder, error) {
	// Open output file
	formatCtx, err := avformat.OpenOutput(outputPath)
	if err != nil {
		return nil, err
	}

	// Create muxer
	muxer := avformat.NewMP4Muxer()

	// Add video stream (Main)
	stream := formatCtx.AddStream()
	stream.CodecType = avutil.MediaTypeVideo
	stream.Width = width
	stream.Height = height
	stream.TimeBase = avutil.Rational{1, 90000}
	stream.FrameRate = avutil.Rational{int(fps * 1000), 1000}

	// Create encoder (Main)
	encoder := avcodec.NewHEVCEncoder()

	// Initialize encoder context (Main)
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

	// Handle Alpha Channel
	var alphaEncoder *avcodec.HEVCEncoder
	var alphaEncoderCtx *avcodec.EncoderContext

	if hasAlpha {
		// Add alpha stream (Auxiliary)
		alphaStream := formatCtx.AddStream()
		alphaStream.CodecType = avutil.MediaTypeVideo
		alphaStream.Width = width
		alphaStream.Height = height
		alphaStream.TimeBase = avutil.Rational{1, 90000}
		alphaStream.FrameRate = avutil.Rational{int(fps * 1000), 1000}

		// Create encoder (Alpha) - Monochrome
		alphaEncoder = avcodec.NewHEVCEncoder()
		
		alphaEncoderCtx = &avcodec.EncoderContext{
			Width:     width,
			Height:    height,
			PixFmt:    avutil.PixFmtGRAY8, // Monochrome for Alpha
			TimeBase:  avutil.Rational{1, 90000},
			Framerate: avutil.Rational{int(fps * 1000), 1000},
			GopSize:   30,
			QMin:      20,
		}

		if err := alphaEncoder.Init(alphaEncoderCtx); err != nil {
			formatCtx.Close()
			return nil, err
		}

		// Set codec data
		alphaStream.CodecData = alphaEncoderCtx.ExtraData

		// Link tracks: Alpha track (1) references Main track (0) with 'auxl'
		// Or 'auxl' from Alpha to Main? "The track containing the auxiliary video stream shall contain a Track Reference Box..."
		// So Alpha Track (1) -> Main Track (0)
		formatCtx.AddTrackReference(1, 0, "auxl")
	}

	// Write header
	if err := muxer.WriteHeader(formatCtx); err != nil {
		formatCtx.Close()
		return nil, err
	}

	return &VideoEncoder{
		formatCtx:       formatCtx,
		muxer:           muxer,
		encoder:         encoder,
		encoderCtx:      encoderCtx,
		alphaEncoder:    alphaEncoder,
		alphaEncoderCtx: alphaEncoderCtx,
		hasAlpha:        hasAlpha,
		width:           width,
		height:          height,
		frameRate:       encoderCtx.Framerate,
	}, nil
}

// EncodeFrame encodes an RGBA image as a video frame.
func (e *VideoEncoder) EncodeFrame(img image.Image) error {
	var mainFrame, alphaFrame *avutil.Frame

	if e.hasAlpha {
		// Convert image to YUVA420P (Main + Alpha)
		mainFrame, alphaFrame = imageToYUVA420P(img, e.width, e.height)
	} else {
		// Convert image to YUV420P (Main only)
		mainFrame = imageToYUV420P(img, e.width, e.height)
	}
	
	// Setup Main Frame
	mainFrame.Pts = e.frameCount
	mainFrame.PktDts = e.frameCount
	mainFrame.KeyFrame = e.frameCount%int64(e.encoderCtx.GopSize) == 0
	if mainFrame.KeyFrame {
		mainFrame.PictType = avutil.PictureTypeI
	} else {
		mainFrame.PictType = avutil.PictureTypeP
	}

	// Encode Main
	packets, err := e.encoder.Encode(e.encoderCtx, mainFrame)
	if err != nil {
		return err
	}

	// Write Main packets
	for _, pkt := range packets {
		pkt.StreamIndex = 0
		if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
			return err
		}
	}

	// Encode Alpha if present
	if e.hasAlpha && alphaFrame != nil {
		// Setup Alpha Frame
		alphaFrame.Pts = e.frameCount
		alphaFrame.PktDts = e.frameCount
		alphaFrame.KeyFrame = mainFrame.KeyFrame // Sync keyframes
		if alphaFrame.KeyFrame {
			alphaFrame.PictType = avutil.PictureTypeI
		} else {
			alphaFrame.PictType = avutil.PictureTypeP
		}

		// Encode Alpha
		alphaPackets, err := e.alphaEncoder.Encode(e.alphaEncoderCtx, alphaFrame)
		if err != nil {
			return err
		}

		// Write Alpha packets
		for _, pkt := range alphaPackets {
			pkt.StreamIndex = 1 // Alpha track
			if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
				return err
			}
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
	// Flush main encoder
	packets, err := e.encoder.Flush(e.encoderCtx)
	if err != nil {
		return err
	}

	for _, pkt := range packets {
		pkt.StreamIndex = 0
		if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
			return err
		}
	}

	// Flush alpha encoder
	if e.hasAlpha {
		alphaPackets, err := e.alphaEncoder.Flush(e.alphaEncoderCtx)
		if err != nil {
			return err
		}
		for _, pkt := range alphaPackets {
			pkt.StreamIndex = 1
			if err := e.muxer.WritePacket(e.formatCtx, pkt); err != nil {
				return err
			}
		}
	}

	// Write trailer
	if err := e.muxer.WriteTrailer(e.formatCtx); err != nil {
		return err
	}

	// Close encoders
	e.encoder.Close(e.encoderCtx)
	if e.hasAlpha {
		e.alphaEncoder.Close(e.alphaEncoderCtx)
	}

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

	// Helper function to get RGB at a specific coordinate
	// This abstracts the specific image type implementation
	getRGB := func(x, y int) (uint8, uint8, uint8) {
		// Fast path: Check if image is standard RGBA to avoid interface overhead
		if rgbaImg, ok := img.(*image.RGBA); ok {
			offset := rgbaImg.PixOffset(x, y)
			// Check bounds to be safe, though logical mapping usually prevents this
			if offset < len(rgbaImg.Pix)-3 {
				return rgbaImg.Pix[offset], rgbaImg.Pix[offset+1], rgbaImg.Pix[offset+2]
			}
		}

		// Fallback for other image types
		r, g, b, _ := img.At(x, y).RGBA()
		return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
	}

	// --- Y Plane Processing ---
	// Y represents Luma (Brightness). Resolution matches the output video size.
	for y := 0; y < height; y++ {
		// Simple Nearest Neighbor scaling for Y
		srcY := bounds.Min.Y + (y * bounds.Dy() / height)

		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + (x * bounds.Dx() / width)

			r, g, b := getRGB(srcX, srcY)

			// BT.601 coefficients for Studio Swing (16-235)
			// Y = 0.257R + 0.504G + 0.098B + 16
			yVal := (66*int(r) + 129*int(g) + 25*int(b) + 128) >> 8
			yVal += 16

			// Clamp Y to 16-235
			if yVal < 16 {
				yVal = 16
			}
			if yVal > 235 {
				yVal = 235
			}

			// Write to Y plane (Data[0])
			// Note: Use Linesize to calculate offset, not Width
			frame.Data[0][y*frame.Linesize[0]+x] = byte(yVal)
		}
	}

	// --- U and V Plane Processing ---
	// 4:2:0 subsampling means chroma is 1/2 width and 1/2 height.
	chromaWidth := width / 2
	chromaHeight := height / 2

	for cy := 0; cy < chromaHeight; cy++ {
		for cx := 0; cx < chromaWidth; cx++ {
			// Map destination chroma coordinate back to source image coordinate
			// We look at the top-left of the 2x2 block in the source
			srcX := bounds.Min.X + ((cx * 2) * bounds.Dx() / width)
			srcY := bounds.Min.Y + ((cy * 2) * bounds.Dy() / height)

			// Sample 2x2 block and average (Box Filter)
			var rSum, gSum, bSum int
			samples := 0

			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					// Check source bounds to prevent panic on edge pixels
					if srcX+dx < bounds.Max.X && srcY+dy < bounds.Max.Y {
						r, g, b := getRGB(srcX+dx, srcY+dy)
						rSum += int(r)
						gSum += int(g)
						bSum += int(b)
						samples++
					}
				}
			}

			// Avoid division by zero if something went wrong with bounds
			if samples == 0 {
				samples = 1
			}

			r := rSum / samples
			g := gSum / samples
			b := bSum / samples

			// BT.601 Coefficients
			// U = -0.148R - 0.291G + 0.439B + 128
			// V =  0.439R - 0.368G - 0.071B + 128

			uVal := (-38*r - 74*g + 112*b + 128) >> 8
			uVal += 128

			vVal := (112*r - 94*g - 18*b + 128) >> 8
			vVal += 128

			// Clamp UV to 16-240
			if uVal < 16 {
				uVal = 16
			}
			if uVal > 240 {
				uVal = 240
			}
			if vVal < 16 {
				vVal = 16
			}
			if vVal > 240 {
				vVal = 240
			}

			// Write to U (Data[1]) and V (Data[2]) planes
			frame.Data[1][cy*frame.Linesize[1]+cx] = byte(uVal)
			frame.Data[2][cy*frame.Linesize[2]+cx] = byte(vVal)
		}
	}

	return frame
}

// imageToYUVA420P converts an image to YUV420P frame and an Alpha (Gray) frame.
func imageToYUVA420P(img image.Image, width, height int) (yuv *avutil.Frame, alpha *avutil.Frame) {
	// 1. Convert to YUV420P (ignoring alpha for YUV part)
	// Note: Standard imageToYUV420P should handle ignoring alpha or premultiplying.
	// Here we assume straight alpha for simple separation.
	yuv = imageToYUV420P(img, width, height)
	
	// 2. Create Alpha frame (Monochrome / Gray8)
	alpha = avutil.NewFrame()
	alpha.Width = width
	alpha.Height = height
	alpha.Format = int(avutil.PixFmtGRAY8)
	alpha.AllocBuffer()

	bounds := img.Bounds()

	// Helper to get Alpha
	getAlpha := func(x, y int) uint8 {
		if rgbaImg, ok := img.(*image.RGBA); ok {
			offset := rgbaImg.PixOffset(x, y)
			if offset < len(rgbaImg.Pix)-3 {
				return rgbaImg.Pix[offset+3]
			}
		}
		_, _, _, a := img.At(x, y).RGBA()
		return uint8(a >> 8)
	}

	// Extract Alpha to Y plane of alpha frame
	for y := 0; y < height; y++ {
		srcY := bounds.Min.Y + (y * bounds.Dy() / height)
		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + (x * bounds.Dx() / width)
			a := getAlpha(srcX, srcY)
			alpha.Data[0][y*alpha.Linesize[0]+x] = a
		}
	}

	return yuv, alpha
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
