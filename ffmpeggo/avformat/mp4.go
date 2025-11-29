package avformat

import (
	"bytes"
	"encoding/binary"

	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// MP4Muxer writes MP4/MOV containers.
type MP4Muxer struct {
	// Collected samples for mdat
	samples []mp4Sample

	// mdat start position
	mdatStart int64

	// Track data
	tracks []mp4Track
}

type mp4Sample struct {
	trackIdx int
	data     []byte
	pts      int64
	dts      int64
	duration int64
	keyframe bool
}

type mp4Track struct {
	stream       *Stream
	sampleSizes  []uint32
	sampleDts    []int64
	chunkOffsets []int64
	syncSamples  []uint32 // Keyframe indices
}

// NewMP4Muxer creates a new MP4 muxer.
func NewMP4Muxer() *MP4Muxer {
	return &MP4Muxer{}
}

// WriteHeader writes the MP4 header (ftyp box).
func (m *MP4Muxer) WriteHeader(ctx *FormatContext) error {
	// Initialize tracks
	m.tracks = make([]mp4Track, len(ctx.Streams))
	for i, s := range ctx.Streams {
		m.tracks[i].stream = s
	}

	// Write ftyp box
	ftyp := m.createFtypBox()
	if err := ctx.write(ftyp); err != nil {
		return err
	}

	// Write free box (placeholder for moov update)
	free := m.createFreeBox(8)
	if err := ctx.write(free); err != nil {
		return err
	}

	// Start mdat box - we'll update the size later
	m.mdatStart, _ = ctx.tell()
	mdatHeader := make([]byte, 8)
	copy(mdatHeader[4:], []byte("mdat"))
	if err := ctx.write(mdatHeader); err != nil {
		return err
	}

	ctx.priv = m
	return nil
}

// WritePacket writes a packet to the mdat box.
func (m *MP4Muxer) WritePacket(ctx *FormatContext, pkt *avutil.Packet) error {
	if pkt == nil || len(pkt.Data) == 0 {
		return nil
	}

	trackIdx := pkt.StreamIndex
	if trackIdx < 0 || trackIdx >= len(m.tracks) {
		trackIdx = 0
	}

	track := &m.tracks[trackIdx]

	// Record current position as chunk offset
	pos, _ := ctx.tell()
	track.chunkOffsets = append(track.chunkOffsets, pos)

	// Convert Annex B (start codes) to length-prefixed for MP4
	data := annexBToMP4(pkt.Data)

	// Record sample info
	track.sampleSizes = append(track.sampleSizes, uint32(len(data)))
	track.sampleDts = append(track.sampleDts, pkt.Dts)

	if pkt.IsKeyframe() {
		track.syncSamples = append(track.syncSamples, uint32(len(track.sampleSizes)))
	}

	// Write the sample data
	return ctx.write(data)
}

// WriteTrailer writes the moov box and finalizes the file.
func (m *MP4Muxer) WriteTrailer(ctx *FormatContext) error {
	// Update mdat size
	endPos, _ := ctx.tell()
	mdatSize := endPos - m.mdatStart

	ctx.seek(m.mdatStart)
	ctx.writeU32BE(uint32(mdatSize))
	ctx.seek(endPos)

	// Write moov box
	moov := m.createMoovBox(ctx)
	return ctx.write(moov)
}

func (m *MP4Muxer) createFtypBox() []byte {
	buf := new(bytes.Buffer)

	// ftyp: file type box
	data := []byte{
		0x00, 0x00, 0x00, 0x18, // size = 24
		'f', 't', 'y', 'p',
		'i', 's', 'o', 'm', // major brand = isom
		0x00, 0x00, 0x02, 0x00, // minor version = 512
		'i', 's', 'o', 'm', // compatible brand
		'i', 's', 'o', '2', // compatible brand
	}
	buf.Write(data)

	return buf.Bytes()
}

func (m *MP4Muxer) createFreeBox(size int) []byte {
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf, uint32(size))
	copy(buf[4:], []byte("free"))
	return buf
}

func (m *MP4Muxer) createMoovBox(ctx *FormatContext) []byte {
	buf := new(bytes.Buffer)

	// mvhd (movie header)
	mvhd := m.createMvhdBox(ctx)

	// trak boxes for each stream
	var traks [][]byte
	for i := range m.tracks {
		trak := m.createTrakBox(ctx, i)
		traks = append(traks, trak)
	}

	// Calculate total size
	totalSize := 8 + len(mvhd)
	for _, trak := range traks {
		totalSize += len(trak)
	}

	// Write moov header
	binary.Write(buf, binary.BigEndian, uint32(totalSize))
	buf.WriteString("moov")

	// Write mvhd
	buf.Write(mvhd)

	// Write traks
	for _, trak := range traks {
		buf.Write(trak)
	}

	return buf.Bytes()
}

func (m *MP4Muxer) createMvhdBox(ctx *FormatContext) []byte {
	buf := new(bytes.Buffer)

	// Get duration from first track (in timescale units)
	duration := int64(0)
	if len(m.tracks) > 0 && len(m.tracks[0].sampleSizes) > 0 {
		duration = m.calculateTrackDuration(&m.tracks[0])
	}

	// mvhd version 0 (32-bit times)
	data := make([]byte, 108)
	binary.BigEndian.PutUint32(data[0:], 108) // size
	copy(data[4:], []byte("mvhd"))
	// version (1 byte) + flags (3 bytes) = 0
	// creation_time (4 bytes) = 0
	// modification_time (4 bytes) = 0
	binary.BigEndian.PutUint32(data[20:], 90000) // timescale
	binary.BigEndian.PutUint32(data[24:], uint32(duration)) // duration
	binary.BigEndian.PutUint32(data[28:], 0x00010000) // rate = 1.0
	binary.BigEndian.PutUint16(data[32:], 0x0100) // volume = 1.0
	// reserved (10 bytes)
	// matrix (36 bytes) - identity
	data[44] = 0x00; data[45] = 0x01; data[46] = 0x00; data[47] = 0x00 // a = 1.0
	data[56] = 0x00; data[57] = 0x01; data[58] = 0x00; data[59] = 0x00 // d = 1.0
	data[76] = 0x40; data[77] = 0x00; data[78] = 0x00; data[79] = 0x00 // w = 1.0
	// pre_defined (24 bytes) = 0
	binary.BigEndian.PutUint32(data[104:], uint32(len(m.tracks)+1)) // next_track_ID

	buf.Write(data)
	return buf.Bytes()
}

func (m *MP4Muxer) createTrakBox(ctx *FormatContext, trackIdx int) []byte {
	buf := new(bytes.Buffer)

	track := &m.tracks[trackIdx]
	stream := track.stream

	// tkhd (track header)
	tkhd := m.createTkhdBox(stream, trackIdx)

	// mdia (media)
	mdia := m.createMdiaBox(ctx, trackIdx)

	// Calculate total size
	totalSize := 8 + len(tkhd) + len(mdia)

	// Write trak header
	binary.Write(buf, binary.BigEndian, uint32(totalSize))
	buf.WriteString("trak")

	buf.Write(tkhd)
	buf.Write(mdia)

	return buf.Bytes()
}

func (m *MP4Muxer) createTkhdBox(stream *Stream, trackIdx int) []byte {
	data := make([]byte, 92)
	binary.BigEndian.PutUint32(data[0:], 92) // size
	copy(data[4:], []byte("tkhd"))
	data[8] = 0    // version
	data[11] = 0x03 // flags: track enabled + in movie

	// creation_time, modification_time = 0
	binary.BigEndian.PutUint32(data[20:], uint32(trackIdx+1)) // track_ID

	// Duration in timescale units (numSamples * sampleDelta)
	duration := m.calculateTrackDuration(&m.tracks[trackIdx])
	binary.BigEndian.PutUint32(data[28:], uint32(duration))

	// reserved, layer, alternate_group, volume
	if stream.CodecType == avutil.MediaTypeAudio {
		binary.BigEndian.PutUint16(data[44:], 0x0100) // volume = 1.0
	}

	// Matrix (identity)
	binary.BigEndian.PutUint32(data[48:], 0x00010000) // a
	binary.BigEndian.PutUint32(data[60:], 0x00010000) // d
	binary.BigEndian.PutUint32(data[84:], 0x40000000) // w

	// Width and height (16.16 fixed point)
	binary.BigEndian.PutUint32(data[84:], uint32(stream.Width)<<16)
	binary.BigEndian.PutUint32(data[88:], uint32(stream.Height)<<16)

	return data
}

func (m *MP4Muxer) createMdiaBox(ctx *FormatContext, trackIdx int) []byte {
	buf := new(bytes.Buffer)

	track := &m.tracks[trackIdx]

	// mdhd (media header)
	mdhd := m.createMdhdBox(track)

	// hdlr (handler)
	hdlr := m.createHdlrBox(track)

	// minf (media info)
	minf := m.createMinfBox(ctx, trackIdx)

	// Calculate total size
	totalSize := 8 + len(mdhd) + len(hdlr) + len(minf)

	// Write mdia header
	binary.Write(buf, binary.BigEndian, uint32(totalSize))
	buf.WriteString("mdia")

	buf.Write(mdhd)
	buf.Write(hdlr)
	buf.Write(minf)

	return buf.Bytes()
}

func (m *MP4Muxer) createMdhdBox(track *mp4Track) []byte {
	data := make([]byte, 32)
	binary.BigEndian.PutUint32(data[0:], 32) // size
	copy(data[4:], []byte("mdhd"))
	// version + flags = 0

	binary.BigEndian.PutUint32(data[20:], uint32(track.stream.TimeBase.Den)) // timescale

	// Duration in timescale units (numSamples * sampleDelta)
	duration := m.calculateTrackDuration(track)
	binary.BigEndian.PutUint32(data[24:], uint32(duration))

	// language (und = 0x55C4)
	binary.BigEndian.PutUint16(data[28:], 0x55C4)

	return data
}

// calculateTrackDuration returns duration in timescale units.
func (m *MP4Muxer) calculateTrackDuration(track *mp4Track) int64 {
	if len(track.sampleDts) == 0 {
		return 0
	}

	// Calculate sample duration (sampleDelta)
	sampleDelta := int64(3000) // Default for 30fps at 90000 timescale
	if track.stream.FrameRate.Num > 0 && track.stream.FrameRate.Den > 0 {
		fps := float64(track.stream.FrameRate.Num) / float64(track.stream.FrameRate.Den)
		if fps > 0 {
			sampleDelta = int64(float64(track.stream.TimeBase.Den) / fps)
		}
	}

	return int64(len(track.sampleDts)) * sampleDelta
}

func (m *MP4Muxer) createHdlrBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	handlerType := "vide"
	handlerName := "VideoHandler"
	if track.stream.CodecType == avutil.MediaTypeAudio {
		handlerType = "soun"
		handlerName = "SoundHandler"
	}

	size := uint32(33 + len(handlerName))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("hdlr")
	buf.Write(make([]byte, 4)) // version + flags
	buf.Write(make([]byte, 4)) // pre_defined
	buf.WriteString(handlerType)
	buf.Write(make([]byte, 12)) // reserved
	buf.WriteString(handlerName)
	buf.WriteByte(0) // null terminator

	return buf.Bytes()
}

func (m *MP4Muxer) createMinfBox(ctx *FormatContext, trackIdx int) []byte {
	buf := new(bytes.Buffer)

	track := &m.tracks[trackIdx]

	// vmhd or smhd (video/sound media header)
	var xmhd []byte
	if track.stream.CodecType == avutil.MediaTypeVideo {
		xmhd = m.createVmhdBox()
	} else {
		xmhd = m.createSmhdBox()
	}

	// dinf (data information)
	dinf := m.createDinfBox()

	// stbl (sample table)
	stbl := m.createStblBox(ctx, trackIdx)

	// Calculate total size
	totalSize := 8 + len(xmhd) + len(dinf) + len(stbl)

	// Write minf header
	binary.Write(buf, binary.BigEndian, uint32(totalSize))
	buf.WriteString("minf")

	buf.Write(xmhd)
	buf.Write(dinf)
	buf.Write(stbl)

	return buf.Bytes()
}

func (m *MP4Muxer) createVmhdBox() []byte {
	data := make([]byte, 20)
	binary.BigEndian.PutUint32(data[0:], 20)
	copy(data[4:], []byte("vmhd"))
	data[11] = 0x01 // flags
	return data
}

func (m *MP4Muxer) createSmhdBox() []byte {
	data := make([]byte, 16)
	binary.BigEndian.PutUint32(data[0:], 16)
	copy(data[4:], []byte("smhd"))
	return data
}

func (m *MP4Muxer) createDinfBox() []byte {
	// dinf contains dref
	dref := []byte{
		0x00, 0x00, 0x00, 0x1C, // size = 28
		'd', 'r', 'e', 'f',
		0x00, 0x00, 0x00, 0x00, // version + flags
		0x00, 0x00, 0x00, 0x01, // entry_count = 1
		0x00, 0x00, 0x00, 0x0C, // url size = 12
		'u', 'r', 'l', ' ',
		0x00, 0x00, 0x00, 0x01, // flags = self contained
	}

	buf := new(bytes.Buffer)
	size := uint32(8 + len(dref))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("dinf")
	buf.Write(dref)

	return buf.Bytes()
}

func (m *MP4Muxer) createStblBox(ctx *FormatContext, trackIdx int) []byte {
	buf := new(bytes.Buffer)

	track := &m.tracks[trackIdx]

	// stsd (sample description)
	stsd := m.createStsdBox(ctx, trackIdx)

	// stts (time to sample)
	stts := m.createSttsBox(track)

	// stsc (sample to chunk)
	stsc := m.createStscBox(track)

	// stsz (sample sizes)
	stsz := m.createStszBox(track)

	// stco (chunk offsets)
	stco := m.createStcoBox(track)

	// stss (sync samples) - only for video with keyframes
	var stss []byte
	if track.stream.CodecType == avutil.MediaTypeVideo && len(track.syncSamples) > 0 {
		stss = m.createStssBox(track)
	}

	// Calculate total size
	totalSize := 8 + len(stsd) + len(stts) + len(stsc) + len(stsz) + len(stco) + len(stss)

	// Write stbl header
	binary.Write(buf, binary.BigEndian, uint32(totalSize))
	buf.WriteString("stbl")

	buf.Write(stsd)
	buf.Write(stts)
	buf.Write(stsc)
	buf.Write(stsz)
	buf.Write(stco)
	if len(stss) > 0 {
		buf.Write(stss)
	}

	return buf.Bytes()
}

func (m *MP4Muxer) createStsdBox(ctx *FormatContext, trackIdx int) []byte {
	buf := new(bytes.Buffer)

	track := &m.tracks[trackIdx]

	// Create sample entry based on codec
	var sampleEntry []byte
	if track.stream.CodecType == avutil.MediaTypeVideo {
		sampleEntry = m.createHvcSampleEntry(track)
	}

	size := uint32(16 + len(sampleEntry))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stsd")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(1)) // entry_count
	buf.Write(sampleEntry)

	return buf.Bytes()
}

func (m *MP4Muxer) createHvcSampleEntry(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	stream := track.stream

	// hvc1 sample entry
	// Base visual sample entry: 86 bytes
	// hvcC box: variable

	hvcC := m.createHvcCBox(track)

	size := uint32(86 + len(hvcC))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("hvc1")

	// Reserved (6 bytes)
	buf.Write(make([]byte, 6))
	// Data reference index
	binary.Write(buf, binary.BigEndian, uint16(1))
	// Pre-defined, reserved
	buf.Write(make([]byte, 16))
	// Width
	binary.Write(buf, binary.BigEndian, uint16(stream.Width))
	// Height
	binary.Write(buf, binary.BigEndian, uint16(stream.Height))
	// Horizontal resolution (72 dpi as 16.16)
	binary.Write(buf, binary.BigEndian, uint32(0x00480000))
	// Vertical resolution (72 dpi as 16.16)
	binary.Write(buf, binary.BigEndian, uint32(0x00480000))
	// Reserved
	buf.Write(make([]byte, 4))
	// Frame count
	binary.Write(buf, binary.BigEndian, uint16(1))
	// Compressor name (32 bytes)
	compressor := make([]byte, 32)
	copy(compressor[1:], "HEVC Coding")
	buf.Write(compressor)
	// Depth
	binary.Write(buf, binary.BigEndian, uint16(0x0018))
	// Pre-defined
	binary.Write(buf, binary.BigEndian, int16(-1))

	// hvcC box
	buf.Write(hvcC)

	return buf.Bytes()
}

func (m *MP4Muxer) createHvcCBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	stream := track.stream

	// Parse VPS, SPS, PPS from codec data
	vps, sps, pps := parseHEVCParameterSets(stream.CodecData)

	// Calculate size
	arraySize := 0
	if len(vps) > 0 {
		arraySize += 5 + len(vps)
	}
	if len(sps) > 0 {
		arraySize += 5 + len(sps)
	}
	if len(pps) > 0 {
		arraySize += 5 + len(pps)
	}

	numArrays := 0
	if len(vps) > 0 {
		numArrays++
	}
	if len(sps) > 0 {
		numArrays++
	}
	if len(pps) > 0 {
		numArrays++
	}

	size := uint32(8 + 23 + arraySize)
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("hvcC")

	// HEVCDecoderConfigurationRecord
	buf.WriteByte(1) // configurationVersion

	// general_profile_space (2) + general_tier_flag (1) + general_profile_idc (5)
	buf.WriteByte(1) // Main profile

	// general_profile_compatibility_flags (32 bits)
	buf.Write([]byte{0x60, 0x00, 0x00, 0x00})

	// general_constraint_indicator_flags (48 bits)
	buf.Write([]byte{0x90, 0x00, 0x00, 0x00, 0x00, 0x00})

	// general_level_idc
	buf.WriteByte(153) // Level 5.1

	// min_spatial_segmentation_idc
	binary.Write(buf, binary.BigEndian, uint16(0xF000))

	// parallelismType
	buf.WriteByte(0xFC)

	// chromaFormat
	buf.WriteByte(0xFD) // 4:2:0

	// bitDepthLumaMinus8
	buf.WriteByte(0xF8)

	// bitDepthChromaMinus8
	buf.WriteByte(0xF8)

	// avgFrameRate
	binary.Write(buf, binary.BigEndian, uint16(0))

	// constantFrameRate + numTemporalLayers + temporalIdNested + lengthSizeMinusOne
	buf.WriteByte(0x0F) // lengthSizeMinusOne = 3 (4 bytes)

	// numOfArrays
	buf.WriteByte(byte(numArrays))

	// Arrays
	if len(vps) > 0 {
		buf.WriteByte(0xA0 | 32) // array_completeness + NAL type (VPS)
		binary.Write(buf, binary.BigEndian, uint16(1)) // numNalus
		binary.Write(buf, binary.BigEndian, uint16(len(vps)))
		buf.Write(vps)
	}

	if len(sps) > 0 {
		buf.WriteByte(0xA0 | 33) // array_completeness + NAL type (SPS)
		binary.Write(buf, binary.BigEndian, uint16(1))
		binary.Write(buf, binary.BigEndian, uint16(len(sps)))
		buf.Write(sps)
	}

	if len(pps) > 0 {
		buf.WriteByte(0xA0 | 34) // array_completeness + NAL type (PPS)
		binary.Write(buf, binary.BigEndian, uint16(1))
		binary.Write(buf, binary.BigEndian, uint16(len(pps)))
		buf.Write(pps)
	}

	return buf.Bytes()
}

func (m *MP4Muxer) createSttsBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	// Simple case: all samples have the same duration
	numSamples := uint32(len(track.sampleSizes))

	// Calculate sample duration based on timescale and framerate
	// timescale / framerate = ticks per frame
	// For 30fps with timescale 90000: 90000/30 = 3000 ticks per frame
	sampleDelta := uint32(3000) // Default for 30fps
	if track.stream.FrameRate.Num > 0 && track.stream.FrameRate.Den > 0 {
		fps := float64(track.stream.FrameRate.Num) / float64(track.stream.FrameRate.Den)
		if fps > 0 {
			sampleDelta = uint32(float64(track.stream.TimeBase.Den) / fps)
		}
	}

	size := uint32(24)
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stts")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(1)) // entry_count
	binary.Write(buf, binary.BigEndian, numSamples) // sample_count
	binary.Write(buf, binary.BigEndian, sampleDelta) // sample_delta

	return buf.Bytes()
}

func (m *MP4Muxer) createStscBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	// Simple case: one sample per chunk
	size := uint32(28)
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stsc")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(1)) // entry_count
	binary.Write(buf, binary.BigEndian, uint32(1)) // first_chunk
	binary.Write(buf, binary.BigEndian, uint32(1)) // samples_per_chunk
	binary.Write(buf, binary.BigEndian, uint32(1)) // sample_description_index

	return buf.Bytes()
}

func (m *MP4Muxer) createStszBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	size := uint32(20 + 4*len(track.sampleSizes))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stsz")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(0)) // sample_size (0 = variable)
	binary.Write(buf, binary.BigEndian, uint32(len(track.sampleSizes))) // sample_count

	for _, sz := range track.sampleSizes {
		binary.Write(buf, binary.BigEndian, sz)
	}

	return buf.Bytes()
}

func (m *MP4Muxer) createStcoBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	size := uint32(16 + 4*len(track.chunkOffsets))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stco")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(len(track.chunkOffsets))) // entry_count

	for _, offset := range track.chunkOffsets {
		binary.Write(buf, binary.BigEndian, uint32(offset))
	}

	return buf.Bytes()
}

func (m *MP4Muxer) createStssBox(track *mp4Track) []byte {
	buf := new(bytes.Buffer)

	size := uint32(16 + 4*len(track.syncSamples))
	binary.Write(buf, binary.BigEndian, size)
	buf.WriteString("stss")
	buf.Write(make([]byte, 4)) // version + flags
	binary.Write(buf, binary.BigEndian, uint32(len(track.syncSamples))) // entry_count

	for _, idx := range track.syncSamples {
		binary.Write(buf, binary.BigEndian, idx)
	}

	return buf.Bytes()
}

// parseHEVCParameterSets extracts VPS, SPS, PPS from Annex B format.
func parseHEVCParameterSets(data []byte) (vps, sps, pps []byte) {
	// Find NAL units
	units := findNALUnits(data)

	for _, unit := range units {
		if len(unit) < 2 {
			continue
		}

		// HEVC NAL header: forbidden_zero_bit (1) + nal_unit_type (6) + nuh_layer_id (6) + nuh_temporal_id_plus1 (3)
		nalType := (unit[0] >> 1) & 0x3F

		switch nalType {
		case 32: // VPS
			vps = unit
		case 33: // SPS
			sps = unit
		case 34: // PPS
			pps = unit
		}
	}

	return
}

// findNALUnits finds NAL units in Annex B format.
func findNALUnits(data []byte) [][]byte {
	var units [][]byte

	i := 0
	for i < len(data) {
		// Find start code (0x000001 or 0x00000001)
		startCodeLen := 0
		if i+3 < len(data) && data[i] == 0 && data[i+1] == 0 {
			if data[i+2] == 1 {
				startCodeLen = 3
			} else if i+4 < len(data) && data[i+2] == 0 && data[i+3] == 1 {
				startCodeLen = 4
			}
		}

		if startCodeLen == 0 {
			i++
			continue
		}

		// Find end of NAL unit
		start := i + startCodeLen
		end := len(data)

		for j := start; j < len(data)-2; j++ {
			if data[j] == 0 && data[j+1] == 0 && (data[j+2] == 0 || data[j+2] == 1) {
				end = j
				break
			}
		}

		units = append(units, data[start:end])
		i = end
	}

	return units
}

// annexBToMP4 converts Annex B (start codes) to length-prefixed format.
func annexBToMP4(data []byte) []byte {
	units := findNALUnits(data)

	buf := new(bytes.Buffer)
	for _, unit := range units {
		binary.Write(buf, binary.BigEndian, uint32(len(unit)))
		buf.Write(unit)
	}

	return buf.Bytes()
}
