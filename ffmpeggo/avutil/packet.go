package avutil

// Packet stores compressed data for encoding/decoding.
// This corresponds to FFmpeg's AVPacket structure.
type Packet struct {
	// Presentation timestamp in AVStream.time_base units.
	Pts int64

	// Decompression timestamp in AVStream.time_base units.
	Dts int64

	// Packet data.
	Data []byte

	// Size of Data in bytes.
	Size int

	// Stream index this packet belongs to.
	StreamIndex int

	// Combination of packet flags.
	Flags PacketFlags

	// Additional packet data that can be provided by the container.
	SideData []PacketSideData

	// Duration of this packet in AVStream.time_base units, 0 if unknown.
	Duration int64

	// Byte position in stream, -1 if unknown.
	Pos int64

	// Time base for the timestamps.
	TimeBase Rational
}

// PacketFlags represents packet flags.
type PacketFlags int

const (
	PacketFlagKey     PacketFlags = 0x0001 // The packet contains a keyframe.
	PacketFlagCorrupt PacketFlags = 0x0002 // The packet content is corrupted.
	PacketFlagDiscard PacketFlags = 0x0004 // Flag to discard this packet.
	PacketFlagTrusted PacketFlags = 0x0008 // Packet from a trusted source.
	PacketFlagDisposable PacketFlags = 0x0010 // Can be dropped after decoding.
)

// PacketSideDataType represents the type of side data.
type PacketSideDataType int

const (
	PacketSideDataTypePalette PacketSideDataType = iota
	PacketSideDataTypeNewExtradata
	PacketSideDataTypeParamChange
	PacketSideDataTypeH263MbInfo
	PacketSideDataTypeReplayGain
	PacketSideDataTypeDisplayMatrix
	PacketSideDataTypeStereo3D
	PacketSideDataTypeAudioServiceType
	PacketSideDataTypeQualityStats
	PacketSideDataTypeFallbackTrack
	PacketSideDataTypeCBDInfo
	PacketSideDataTypeSkipSamples
	PacketSideDataTypeJpDualmono
	PacketSideDataTypeStringsMetadata
	PacketSideDataTypeSubtitlePosition
	PacketSideDataTypeMatroskaBlockadditional
	PacketSideDataTypeWebVttIdentifier
	PacketSideDataTypeWebVttSettings
	PacketSideDataTypeMetadataUpdate
	PacketSideDataTypeMpegTsStreamID
	PacketSideDataTypeMasteringDisplayMetadata
	PacketSideDataTypeContentLightLevel
	PacketSideDataTypeA53CC
	PacketSideDataTypeEncryptionInitInfo
	PacketSideDataTypeEncryptionInfo
	PacketSideDataTypeAFD
)

// PacketSideData represents side data attached to a packet.
type PacketSideData struct {
	Data []byte
	Type PacketSideDataType
}

// NewPacket creates a new empty Packet.
func NewPacket() *Packet {
	return &Packet{
		Pts:         NoTimestamp,
		Dts:         NoTimestamp,
		Pos:         -1,
		StreamIndex: -1,
		TimeBase:    Rational{1, 1},
	}
}

// Alloc allocates the packet's data buffer.
func (p *Packet) Alloc(size int) error {
	if size < 0 {
		return ErrInvalidData
	}
	p.Data = make([]byte, size)
	p.Size = size
	return nil
}

// Grow grows the packet's data buffer.
func (p *Packet) Grow(growBy int) error {
	if growBy < 0 {
		return ErrInvalidData
	}
	newData := make([]byte, len(p.Data)+growBy)
	copy(newData, p.Data)
	p.Data = newData
	p.Size = len(newData)
	return nil
}

// Shrink shrinks the packet's data buffer to the given size.
func (p *Packet) Shrink(size int) {
	if size < 0 {
		size = 0
	}
	if size > len(p.Data) {
		return
	}
	p.Data = p.Data[:size]
	p.Size = size
}

// Unref unreferences the packet buffer.
func (p *Packet) Unref() {
	p.Data = nil
	p.Size = 0
	p.SideData = nil
	p.Pts = NoTimestamp
	p.Dts = NoTimestamp
	p.Pos = -1
	p.Duration = 0
	p.Flags = 0
}

// Clone creates a new packet referencing the same data.
func (p *Packet) Clone() *Packet {
	dst := NewPacket()
	*dst = *p
	// Copy the data
	if p.Data != nil {
		dst.Data = make([]byte, len(p.Data))
		copy(dst.Data, p.Data)
	}
	return dst
}

// IsKeyframe returns true if this packet is a keyframe.
func (p *Packet) IsKeyframe() bool {
	return p.Flags&PacketFlagKey != 0
}

// SetKeyframe sets or clears the keyframe flag.
func (p *Packet) SetKeyframe(keyframe bool) {
	if keyframe {
		p.Flags |= PacketFlagKey
	} else {
		p.Flags &^= PacketFlagKey
	}
}

// AddSideData adds side data to the packet.
func (p *Packet) AddSideData(dataType PacketSideDataType, data []byte) {
	p.SideData = append(p.SideData, PacketSideData{
		Type: dataType,
		Data: data,
	})
}

// GetSideData retrieves side data of a specific type.
func (p *Packet) GetSideData(dataType PacketSideDataType) []byte {
	for _, sd := range p.SideData {
		if sd.Type == dataType {
			return sd.Data
		}
	}
	return nil
}

// RescaleTs rescales timestamps from one time base to another.
func (p *Packet) RescaleTs(srcTb, dstTb Rational) {
	if p.Pts != NoTimestamp {
		p.Pts = RescaleQ(p.Pts, srcTb, dstTb)
	}
	if p.Dts != NoTimestamp {
		p.Dts = RescaleQ(p.Dts, srcTb, dstTb)
	}
	if p.Duration > 0 {
		p.Duration = RescaleQ(p.Duration, srcTb, dstTb)
	}
}

// RescaleQ rescales a value from one time base to another.
func RescaleQ(a int64, bq, cq Rational) int64 {
	if bq.Den == 0 || cq.Den == 0 {
		return a
	}
	// a * bq.Num / bq.Den * cq.Den / cq.Num
	// = a * bq.Num * cq.Den / (bq.Den * cq.Num)
	num := int64(bq.Num) * int64(cq.Den)
	den := int64(bq.Den) * int64(cq.Num)
	if den == 0 {
		return a
	}
	return a * num / den
}
