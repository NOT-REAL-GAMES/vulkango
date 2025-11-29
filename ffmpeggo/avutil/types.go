// Package avutil provides common types and utilities for audio/video processing.
// This is a Go port of FFmpeg's libavutil.
package avutil

// PixelFormat describes the pixel format of a frame.
type PixelFormat int

const (
	PixFmtNone PixelFormat = iota - 1
	PixFmtYUV420P          // Planar YUV 4:2:0 (1 Cr & Cb sample per 2x2 Y samples)
	PixFmtYUYV422          // Packed YUV 4:2:2
	PixFmtRGB24            // Packed RGB 8:8:8
	PixFmtBGR24            // Packed BGR 8:8:8
	PixFmtYUV422P          // Planar YUV 4:2:2
	PixFmtYUV444P          // Planar YUV 4:4:4
	PixFmtYUV410P          // Planar YUV 4:1:0
	PixFmtYUV411P          // Planar YUV 4:1:1
	PixFmtGRAY8            // Y, 8bpp
	PixFmtMonoWhite        // Y, 1bpp, 0 is white
	PixFmtMonoBlack        // Y, 1bpp, 0 is black
	PixFmtPAL8             // 8-bit with palette
	PixFmtYUVJ420P         // Full-range YUV 4:2:0
	PixFmtYUVJ422P         // Full-range YUV 4:2:2
	PixFmtYUVJ444P         // Full-range YUV 4:4:4
	PixFmtNV12             // Planar YUV 4:2:0, interleaved UV
	PixFmtNV21             // Planar YUV 4:2:0, interleaved VU
	PixFmtARGB             // Packed ARGB 8:8:8:8
	PixFmtRGBA             // Packed RGBA 8:8:8:8
	PixFmtABGR             // Packed ABGR 8:8:8:8
	PixFmtBGRA             // Packed BGRA 8:8:8:8
	PixFmtGRAY16BE         // Y, 16bpp, big-endian
	PixFmtGRAY16LE         // Y, 16bpp, little-endian
	PixFmtYUV420P10LE      // Planar YUV 4:2:0, 10bpp, little-endian
	PixFmtYUV420P10BE      // Planar YUV 4:2:0, 10bpp, big-endian
	PixFmtYUV422P10LE      // Planar YUV 4:2:2, 10bpp, little-endian
	PixFmtYUV444P10LE      // Planar YUV 4:4:4, 10bpp, little-endian
)

// SampleFormat describes the sample format of audio data.
type SampleFormat int

const (
	SampleFmtNone SampleFormat = iota - 1
	SampleFmtU8                // Unsigned 8 bits
	SampleFmtS16               // Signed 16 bits
	SampleFmtS32               // Signed 32 bits
	SampleFmtFLT               // Float
	SampleFmtDBL               // Double
	SampleFmtU8P               // Unsigned 8 bits, planar
	SampleFmtS16P              // Signed 16 bits, planar
	SampleFmtS32P              // Signed 32 bits, planar
	SampleFmtFLTP              // Float, planar
	SampleFmtDBLP              // Double, planar
	SampleFmtS64               // Signed 64 bits
	SampleFmtS64P              // Signed 64 bits, planar
)

// Rational represents a rational number (numerator/denominator).
type Rational struct {
	Num int // Numerator
	Den int // Denominator
}

// NewRational creates a new Rational with reduced form.
func NewRational(num, den int) Rational {
	if den == 0 {
		return Rational{0, 0}
	}
	g := gcd(abs(num), abs(den))
	if den < 0 {
		num, den = -num, -den
	}
	return Rational{num / g, den / g}
}

// Float64 returns the rational as a float64.
func (r Rational) Float64() float64 {
	if r.Den == 0 {
		return 0
	}
	return float64(r.Num) / float64(r.Den)
}

// Invert returns the inverse of the rational.
func (r Rational) Invert() Rational {
	return Rational{r.Den, r.Num}
}

// Mul multiplies two rationals.
func (r Rational) Mul(other Rational) Rational {
	return NewRational(r.Num*other.Num, r.Den*other.Den)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// MediaType represents the type of media (video, audio, etc.).
type MediaType int

const (
	MediaTypeUnknown MediaType = iota - 1
	MediaTypeVideo
	MediaTypeAudio
	MediaTypeData
	MediaTypeSubtitle
	MediaTypeAttachment
)

// PictureType represents the type of picture in video.
type PictureType int

const (
	PictureTypeNone PictureType = iota
	PictureTypeI                // Intra
	PictureTypeP                // Predicted
	PictureTypeB                // Bi-directional predicted
	PictureTypeS                // S(GMC)-VOP MPEG-4
	PictureTypeSI               // Switching Intra
	PictureTypeSP               // Switching Predicted
	PictureTypeBI               // BI type
)

// ColorPrimaries represents color primaries.
type ColorPrimaries int

const (
	ColorPrimariesReserved0  ColorPrimaries = 0
	ColorPrimariesBT709      ColorPrimaries = 1
	ColorPrimariesUnspecif   ColorPrimaries = 2
	ColorPrimariesBT470M     ColorPrimaries = 4
	ColorPrimariesBT470BG    ColorPrimaries = 5
	ColorPrimariesSMPTE170M  ColorPrimaries = 6
	ColorPrimariesSMPTE240M  ColorPrimaries = 7
	ColorPrimariesFilm       ColorPrimaries = 8
	ColorPrimariesBT2020     ColorPrimaries = 9
	ColorPrimariesSMPTE428   ColorPrimaries = 10
	ColorPrimariesSMPTE431   ColorPrimaries = 11
	ColorPrimariesSMPTE432   ColorPrimaries = 12
	ColorPrimariesJEDECP22   ColorPrimaries = 22
)

// ColorTransferCharacteristic represents transfer characteristics.
type ColorTransferCharacteristic int

const (
	ColorTrcReserved0    ColorTransferCharacteristic = 0
	ColorTrcBT709        ColorTransferCharacteristic = 1
	ColorTrcUnspecified  ColorTransferCharacteristic = 2
	ColorTrcGamma22      ColorTransferCharacteristic = 4
	ColorTrcGamma28      ColorTransferCharacteristic = 5
	ColorTrcSMPTE170M    ColorTransferCharacteristic = 6
	ColorTrcSMPTE240M    ColorTransferCharacteristic = 7
	ColorTrcLinear       ColorTransferCharacteristic = 8
	ColorTrcLog          ColorTransferCharacteristic = 9
	ColorTrcLogSqrt      ColorTransferCharacteristic = 10
	ColorTrcIEC61966_2_4 ColorTransferCharacteristic = 11
	ColorTrcBT1361E      ColorTransferCharacteristic = 12
	ColorTrcIEC61966_2_1 ColorTransferCharacteristic = 13 // sRGB
	ColorTrcBT2020_10    ColorTransferCharacteristic = 14
	ColorTrcBT2020_12    ColorTransferCharacteristic = 15
	ColorTrcSMPTE2084    ColorTransferCharacteristic = 16 // PQ
	ColorTrcARIBSTDB67   ColorTransferCharacteristic = 18 // HLG
)

// ColorSpace represents YUV color space.
type ColorSpace int

const (
	ColorSpaceRGB        ColorSpace = 0
	ColorSpaceBT709      ColorSpace = 1
	ColorSpaceUnspecif   ColorSpace = 2
	ColorSpaceFCC        ColorSpace = 4
	ColorSpaceBT470BG    ColorSpace = 5
	ColorSpaceSMPTE170M  ColorSpace = 6
	ColorSpaceSMPTE240M  ColorSpace = 7
	ColorSpaceYCGCO      ColorSpace = 8
	ColorSpaceBT2020NCL  ColorSpace = 9
	ColorSpaceBT2020CL   ColorSpace = 10
	ColorSpaceSMPTE2085  ColorSpace = 11
	ColorSpaceChromaDNCL ColorSpace = 12
	ColorSpaceChromaDCL  ColorSpace = 13
	ColorSpaceICTCP      ColorSpace = 14
)

// ColorRange represents the color range.
type ColorRange int

const (
	ColorRangeUnspecified ColorRange = 0
	ColorRangeMPEG        ColorRange = 1 // Limited range (16-235 for Y, 16-240 for UV)
	ColorRangeJPEG        ColorRange = 2 // Full range (0-255)
)

// ChromaLocation represents chroma sample location.
type ChromaLocation int

const (
	ChromaLocationUnspecified ChromaLocation = 0
	ChromaLocationLeft        ChromaLocation = 1 // MPEG-2/4 4:2:0, H.264 default
	ChromaLocationCenter      ChromaLocation = 2 // MPEG-1 4:2:0, JPEG 4:2:0
	ChromaLocationTopLeft     ChromaLocation = 3
	ChromaLocationTop         ChromaLocation = 4
	ChromaLocationBottomLeft  ChromaLocation = 5
	ChromaLocationBottom      ChromaLocation = 6
)
