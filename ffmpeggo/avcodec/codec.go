// Package avcodec provides audio/video codec interfaces and implementations.
// This is a Go port of FFmpeg's libavcodec.
package avcodec

import (
	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// CodecID identifies a specific codec.
type CodecID int

const (
	CodecIDNone CodecID = iota
	// Video codecs
	CodecIDH264
	CodecIDHEVC // H.265
	CodecIDAV1
	CodecIDVP8
	CodecIDVP9
	CodecIDMJPEG
	CodecIDPNG
	// Audio codecs
	CodecIDAAC
	CodecIDMP3
	CodecIDOpus
	CodecIDVorbis
	CodecIDFLAC
	CodecIDPCMS16LE
	CodecIDPCMS16BE
)

// Codec represents a codec's capabilities and metadata.
type Codec struct {
	ID        CodecID
	Name      string
	LongName  string
	MediaType avutil.MediaType
	// Capabilities flags
	Capabilities CodecCapability
}

// CodecCapability flags.
type CodecCapability int

const (
	CapDrawHorizBand   CodecCapability = 1 << 0
	CapDR1             CodecCapability = 1 << 1
	CapTruncated       CodecCapability = 1 << 3
	CapDelay           CodecCapability = 1 << 5
	CapSmallLastFrame  CodecCapability = 1 << 6
	CapSubFrames       CodecCapability = 1 << 8
	CapExperimental    CodecCapability = 1 << 9
	CapChannelConf     CodecCapability = 1 << 10
	CapFrameThreads    CodecCapability = 1 << 12
	CapSliceThreads    CodecCapability = 1 << 13
	CapParamChange     CodecCapability = 1 << 14
	CapAutoThreads     CodecCapability = 1 << 15
	CapVariableFrameSize CodecCapability = 1 << 16
	CapAvoidProbing    CodecCapability = 1 << 17
	CapIntraOnly       CodecCapability = 1 << 30
	CapLossless        CodecCapability = 1 << 31
)

// EncoderContext holds state for encoding.
type EncoderContext struct {
	Codec *Codec

	// Video parameters
	Width     int
	Height    int
	PixFmt    avutil.PixelFormat
	TimeBase  avutil.Rational
	Framerate avutil.Rational
	GopSize   int // Group of pictures size (keyframe interval)
	MaxBFrames int

	// Quality parameters
	BitRate   int64
	QMin      int
	QMax      int
	QCompress float64

	// Audio parameters
	SampleRate    int
	SampleFmt     avutil.SampleFormat
	ChannelLayout uint64
	Channels      int

	// Codec-specific data
	ExtraData []byte

	// Internal state
	FrameNum int64

	// Private implementation data
	priv interface{}
}

// Encoder interface for all encoders.
type Encoder interface {
	// Init initializes the encoder with the given context.
	Init(ctx *EncoderContext) error

	// Encode encodes a frame and returns packets.
	// Returns nil packets if more frames are needed.
	Encode(ctx *EncoderContext, frame *avutil.Frame) ([]*avutil.Packet, error)

	// Flush flushes any buffered frames.
	Flush(ctx *EncoderContext) ([]*avutil.Packet, error)

	// Close releases encoder resources.
	Close(ctx *EncoderContext) error
}

// DecoderContext holds state for decoding.
type DecoderContext struct {
	Codec *Codec

	// Video parameters
	Width  int
	Height int
	PixFmt avutil.PixelFormat

	// Audio parameters
	SampleRate    int
	SampleFmt     avutil.SampleFormat
	ChannelLayout uint64
	Channels      int

	// Codec-specific data from container
	ExtraData []byte

	// Private implementation data
	priv interface{}
}

// Decoder interface for all decoders.
type Decoder interface {
	// Init initializes the decoder with the given context.
	Init(ctx *DecoderContext) error

	// Decode decodes a packet and returns frames.
	// Returns nil frames if more packets are needed.
	Decode(ctx *DecoderContext, pkt *avutil.Packet) ([]*avutil.Frame, error)

	// Flush flushes any buffered packets.
	Flush(ctx *DecoderContext) ([]*avutil.Frame, error)

	// Close releases decoder resources.
	Close(ctx *DecoderContext) error
}

// Common codec constants

// Profile constants for H.265/HEVC
const (
	HEVCProfileMain   = 1
	HEVCProfileMain10 = 2
	HEVCProfileMainStillPicture = 3
	HEVCProfileRExt   = 4
)

// Level constants for H.265/HEVC (multiplied by 30)
const (
	HEVCLevel1   = 30  // Level 1
	HEVCLevel2   = 60  // Level 2
	HEVCLevel21  = 63  // Level 2.1
	HEVCLevel3   = 90  // Level 3
	HEVCLevel31  = 93  // Level 3.1
	HEVCLevel4   = 120 // Level 4
	HEVCLevel41  = 123 // Level 4.1
	HEVCLevel5   = 150 // Level 5
	HEVCLevel51  = 153 // Level 5.1
	HEVCLevel52  = 156 // Level 5.2
	HEVCLevel6   = 180 // Level 6
	HEVCLevel61  = 183 // Level 6.1
	HEVCLevel62  = 186 // Level 6.2
)

// NAL unit types for HEVC
const (
	HEVCNalTrailN       = 0
	HEVCNalTrailR       = 1
	HEVCNalTsaN         = 2
	HEVCNalTsaR         = 3
	HEVCNalStsaN        = 4
	HEVCNalStsaR        = 5
	HEVCNalRadlN        = 6
	HEVCNalRadlR        = 7
	HEVCNalRaslN        = 8
	HEVCNalRaslR        = 9
	HEVCNalBlaWLP       = 16
	HEVCNalBlaWRadl     = 17
	HEVCNalBlaNLP       = 18
	HEVCNalIdrWRadl     = 19
	HEVCNalIdrNLP       = 20
	HEVCNalCraNut       = 21
	HEVCNalVPS          = 32
	HEVCNalSPS          = 33
	HEVCNalPPS          = 34
	HEVCNalAUD          = 35
	HEVCNalEOSNut       = 36
	HEVCNalEOBNut       = 37
	HEVCNalFDNut        = 38
	HEVCNalSeiPrefix    = 39
	HEVCNalSeiSuffix    = 40
)

// Slice types for HEVC
const (
	HEVCSliceB = 0
	HEVCSliceP = 1
	HEVCSliceI = 2
)
