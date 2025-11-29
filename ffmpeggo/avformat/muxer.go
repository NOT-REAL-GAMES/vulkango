// Package avformat provides container format muxing/demuxing.
// This is a Go port of FFmpeg's libavformat.
package avformat

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// FormatContext holds muxer/demuxer state.
type FormatContext struct {
	// Output destination
	output io.WriteSeeker

	// Streams in this container
	Streams []*Stream

	// Format metadata
	Duration int64

	// Private data for format-specific state
	priv interface{}
}

// Stream represents a media stream in the container.
type Stream struct {
	Index     int
	CodecType avutil.MediaType
	CodecID   int
	TimeBase  avutil.Rational

	// Video parameters
	Width  int
	Height int

	// Audio parameters
	SampleRate int
	Channels   int

	// Codec-specific data (VPS/SPS/PPS for HEVC)
	CodecData []byte

	// Frame rate (for video)
	FrameRate avutil.Rational
}

// Muxer interface for container formats.
type Muxer interface {
	WriteHeader(ctx *FormatContext) error
	WritePacket(ctx *FormatContext, pkt *avutil.Packet) error
	WriteTrailer(ctx *FormatContext) error
}

// OpenOutput opens a file for muxing.
func OpenOutput(filename string) (*FormatContext, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &FormatContext{
		output: f,
	}, nil
}

// AddStream adds a stream to the format context.
func (ctx *FormatContext) AddStream() *Stream {
	s := &Stream{
		Index:    len(ctx.Streams),
		TimeBase: avutil.Rational{1, 90000}, // Default time base
	}
	ctx.Streams = append(ctx.Streams, s)
	return s
}

// Close closes the output file.
func (ctx *FormatContext) Close() error {
	if f, ok := ctx.output.(*os.File); ok {
		return f.Close()
	}
	return nil
}

// Write helpers
func (ctx *FormatContext) write(data []byte) error {
	_, err := ctx.output.Write(data)
	return err
}

func (ctx *FormatContext) writeU8(v byte) error {
	return ctx.write([]byte{v})
}

func (ctx *FormatContext) writeU16BE(v uint16) error {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	return ctx.write(buf)
}

func (ctx *FormatContext) writeU24BE(v uint32) error {
	buf := []byte{byte(v >> 16), byte(v >> 8), byte(v)}
	return ctx.write(buf)
}

func (ctx *FormatContext) writeU32BE(v uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	return ctx.write(buf)
}

func (ctx *FormatContext) writeU64BE(v uint64) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	return ctx.write(buf)
}

func (ctx *FormatContext) tell() (int64, error) {
	return ctx.output.Seek(0, io.SeekCurrent)
}

func (ctx *FormatContext) seek(offset int64) error {
	_, err := ctx.output.Seek(offset, io.SeekStart)
	return err
}
