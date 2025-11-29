package avutil

// Frame represents a decoded video or audio frame.
// This corresponds to FFmpeg's AVFrame structure.
type Frame struct {
	// For video frames: picture data and linesize.
	// For audio frames: sample data.
	// data[i] contains the pointer to the picture/audio plane.
	Data     [8][]byte
	Linesize [8]int // For video, size in bytes of each picture line.

	// Video dimensions
	Width  int
	Height int

	// Number of audio samples (per channel) described by this frame.
	NbSamples int

	// Format of the frame. -1 if unknown or unset.
	// For video: PixelFormat
	// For audio: SampleFormat
	Format int

	// Whether this is a keyframe.
	KeyFrame bool

	// Picture type of the frame.
	PictType PictureType

	// Sample aspect ratio for video frames.
	SampleAspectRatio Rational

	// Presentation timestamp in time_base units.
	Pts int64

	// DTS (decode timestamp) copied from the AVPacket that triggered
	// returning this frame.
	PktDts int64

	// Duration of the frame.
	Duration int64

	// Time base for the timestamps.
	TimeBase Rational

	// Quality (between 1 (good) and FF_LAMBDA_MAX (bad)).
	Quality int

	// Opaque user data for callbacks.
	Opaque interface{}

	// When decoding, signals how much the picture must be delayed.
	RepeatPict int

	// Interlaced frame flag.
	InterlacedFrame bool

	// If the content is interlaced, is top field displayed first.
	TopFieldFirst bool

	// Tell user application that palette has changed from previous frame.
	PaletteHasChanged bool

	// Reordered opaque.
	ReorderedOpaque int64

	// Audio sample rate.
	SampleRate int

	// Audio channel layout (bitmask).
	ChannelLayout uint64

	// Number of audio channels.
	Channels int

	// Buffer refs for the data planes.
	// Extended data is used for planar audio with more than 8 channels.
	ExtendedData [][]byte

	// Color properties
	ColorPrimaries ColorPrimaries
	ColorTrc       ColorTransferCharacteristic
	ColorSpace     ColorSpace
	ColorRange     ColorRange
	ChromaLocation ChromaLocation
}

// NewFrame creates a new empty Frame.
func NewFrame() *Frame {
	return &Frame{
		Format:     -1,
		Pts:        NoTimestamp,
		PktDts:     NoTimestamp,
		Quality:    -1,
		TimeBase:   Rational{0, 1},
		ColorRange: ColorRangeUnspecified,
	}
}

// NoTimestamp represents an undefined timestamp value.
const NoTimestamp int64 = -0x8000000000000000

// AllocBuffer allocates buffer for the frame based on format and dimensions.
// For video, it allocates based on pixel format, width, and height.
func (f *Frame) AllocBuffer() error {
	if f.Width <= 0 || f.Height <= 0 {
		return ErrInvalidData
	}

	pixFmt := PixelFormat(f.Format)
	switch pixFmt {
	case PixFmtYUV420P, PixFmtYUVJ420P:
		// Y plane
		f.Linesize[0] = f.Width
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)
		// U and V planes (half width, half height)
		f.Linesize[1] = f.Width / 2
		f.Linesize[2] = f.Width / 2
		chromaHeight := f.Height / 2
		f.Data[1] = make([]byte, f.Linesize[1]*chromaHeight)
		f.Data[2] = make([]byte, f.Linesize[2]*chromaHeight)

	case PixFmtYUV422P, PixFmtYUVJ422P:
		f.Linesize[0] = f.Width
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)
		f.Linesize[1] = f.Width / 2
		f.Linesize[2] = f.Width / 2
		f.Data[1] = make([]byte, f.Linesize[1]*f.Height)
		f.Data[2] = make([]byte, f.Linesize[2]*f.Height)

	case PixFmtYUV444P, PixFmtYUVJ444P:
		f.Linesize[0] = f.Width
		f.Linesize[1] = f.Width
		f.Linesize[2] = f.Width
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)
		f.Data[1] = make([]byte, f.Linesize[1]*f.Height)
		f.Data[2] = make([]byte, f.Linesize[2]*f.Height)

	case PixFmtNV12:
		f.Linesize[0] = f.Width
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)
		f.Linesize[1] = f.Width
		f.Data[1] = make([]byte, f.Linesize[1]*(f.Height/2))

	case PixFmtRGB24, PixFmtBGR24:
		f.Linesize[0] = f.Width * 3
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)

	case PixFmtRGBA, PixFmtBGRA, PixFmtARGB, PixFmtABGR:
		f.Linesize[0] = f.Width * 4
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)

	case PixFmtGRAY8:
		f.Linesize[0] = f.Width
		f.Data[0] = make([]byte, f.Linesize[0]*f.Height)

	default:
		return ErrUnknownFormat
	}

	return nil
}

// Unref unreferences the frame buffer.
func (f *Frame) Unref() {
	for i := range f.Data {
		f.Data[i] = nil
	}
	f.ExtendedData = nil
}

// Clone creates a new frame referencing the same data.
func (f *Frame) Clone() *Frame {
	dst := NewFrame()
	*dst = *f
	// Data slices are shared by default (reference)
	return dst
}

// CopyTo copies frame data to destination frame.
func (f *Frame) CopyTo(dst *Frame) error {
	dst.Width = f.Width
	dst.Height = f.Height
	dst.Format = f.Format
	dst.KeyFrame = f.KeyFrame
	dst.PictType = f.PictType
	dst.Pts = f.Pts
	dst.PktDts = f.PktDts
	dst.Duration = f.Duration
	dst.TimeBase = f.TimeBase

	if err := dst.AllocBuffer(); err != nil {
		return err
	}

	for i := 0; i < 8; i++ {
		if f.Data[i] != nil {
			copy(dst.Data[i], f.Data[i])
		}
	}

	return nil
}

// Error types
type Error int

const (
	ErrOK           Error = 0
	ErrInvalidData  Error = -1
	ErrUnknownFormat Error = -2
	ErrNoMem        Error = -3
	ErrEOF          Error = -4
	ErrAgain        Error = -5
)

func (e Error) Error() string {
	switch e {
	case ErrOK:
		return "success"
	case ErrInvalidData:
		return "invalid data"
	case ErrUnknownFormat:
		return "unknown format"
	case ErrNoMem:
		return "not enough memory"
	case ErrEOF:
		return "end of file"
	case ErrAgain:
		return "resource temporarily unavailable"
	default:
		return "unknown error"
	}
}
