package vulkango

/*
#cgo CFLAGS: -I/usr/include
#cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lvulkan
#cgo windows LDFLAGS: -lvulkan-1
#cgo darwin LDFLAGS: -lvulkan

#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>

// Include the video codec headers if available
// These define the StdVideoH264* types
#ifdef VK_ENABLE_BETA_EXTENSIONS
#include <vk_video/vulkan_video_codec_h264std.h>
#include <vk_video/vulkan_video_codec_h264std_encode.h>
#endif

*/
import "C"
import (
	"encoding/binary"
	"unsafe"
)

// ============================================================================
// H.264 Standard Video Types (from vk_video_codec_h264std.h)
// ============================================================================

// H264ProfileIdc - H.264 profile identifiers
type H264ProfileIdc uint32

const (
	H264_PROFILE_IDC_BASELINE            H264ProfileIdc = 66
	H264_PROFILE_IDC_MAIN                H264ProfileIdc = 77
	H264_PROFILE_IDC_HIGH                H264ProfileIdc = 100
	H264_PROFILE_IDC_HIGH_444_PREDICTIVE H264ProfileIdc = 244
)

// H264LevelIdc - H.264 level identifiers
type H264LevelIdc uint32

const (
	H264_LEVEL_IDC_1_0 H264LevelIdc = 10
	H264_LEVEL_IDC_1_1 H264LevelIdc = 11
	H264_LEVEL_IDC_1_2 H264LevelIdc = 12
	H264_LEVEL_IDC_1_3 H264LevelIdc = 13
	H264_LEVEL_IDC_2_0 H264LevelIdc = 20
	H264_LEVEL_IDC_2_1 H264LevelIdc = 21
	H264_LEVEL_IDC_2_2 H264LevelIdc = 22
	H264_LEVEL_IDC_3_0 H264LevelIdc = 30
	H264_LEVEL_IDC_3_1 H264LevelIdc = 31
	H264_LEVEL_IDC_3_2 H264LevelIdc = 32
	H264_LEVEL_IDC_4_0 H264LevelIdc = 40
	H264_LEVEL_IDC_4_1 H264LevelIdc = 41
	H264_LEVEL_IDC_4_2 H264LevelIdc = 42
	H264_LEVEL_IDC_5_0 H264LevelIdc = 50
	H264_LEVEL_IDC_5_1 H264LevelIdc = 51
	H264_LEVEL_IDC_5_2 H264LevelIdc = 52
	H264_LEVEL_IDC_6_0 H264LevelIdc = 60
	H264_LEVEL_IDC_6_1 H264LevelIdc = 61
	H264_LEVEL_IDC_6_2 H264LevelIdc = 62
)

// H264ChromaFormatIdc - Chroma format
type H264ChromaFormatIdc uint32

const (
	H264_CHROMA_FORMAT_IDC_MONOCHROME H264ChromaFormatIdc = 0
	H264_CHROMA_FORMAT_IDC_420        H264ChromaFormatIdc = 1
	H264_CHROMA_FORMAT_IDC_422        H264ChromaFormatIdc = 2
	H264_CHROMA_FORMAT_IDC_444        H264ChromaFormatIdc = 3
)

// H264SliceType - Slice types
type H264SliceType uint32

const (
	H264_SLICE_TYPE_P  H264SliceType = 0
	H264_SLICE_TYPE_B  H264SliceType = 1
	H264_SLICE_TYPE_I  H264SliceType = 2
	H264_SLICE_TYPE_SP H264SliceType = 3
	H264_SLICE_TYPE_SI H264SliceType = 4
)

// H264PictureType - Picture types for encoding
type H264PictureType uint32

const (
	H264_PICTURE_TYPE_P   H264PictureType = 0
	H264_PICTURE_TYPE_B   H264PictureType = 1
	H264_PICTURE_TYPE_I   H264PictureType = 2
	H264_PICTURE_TYPE_IDR H264PictureType = 5
)

// H264NalUnitType - NAL unit types
type H264NalUnitType uint32

const (
	H264_NAL_UNIT_TYPE_UNSPECIFIED     H264NalUnitType = 0
	H264_NAL_UNIT_TYPE_CODED_SLICE     H264NalUnitType = 1
	H264_NAL_UNIT_TYPE_CODED_SLICE_IDR H264NalUnitType = 5
	H264_NAL_UNIT_TYPE_SEI             H264NalUnitType = 6
	H264_NAL_UNIT_TYPE_SPS             H264NalUnitType = 7
	H264_NAL_UNIT_TYPE_PPS             H264NalUnitType = 8
	H264_NAL_UNIT_TYPE_AUD             H264NalUnitType = 9
	H264_NAL_UNIT_TYPE_END_OF_SEQ      H264NalUnitType = 10
	H264_NAL_UNIT_TYPE_END_OF_STREAM   H264NalUnitType = 11
	H264_NAL_UNIT_TYPE_FILLER          H264NalUnitType = 12
)

// StdVideoH264SpsFlags - SPS flags
type StdVideoH264SpsFlags struct {
	ConstraintSet0Flag                   uint32
	ConstraintSet1Flag                   uint32
	ConstraintSet2Flag                   uint32
	ConstraintSet3Flag                   uint32
	ConstraintSet4Flag                   uint32
	ConstraintSet5Flag                   uint32
	Direct8x8InferenceFlag               uint32
	MbAdaptiveFrameFieldFlag             uint32
	FrameMbsOnlyFlag                     uint32
	DeltaPicOrderAlwaysZeroFlag          uint32
	SeparateColourPlaneFlag              uint32
	GapsInFrameNumValueAllowedFlag       uint32
	QpprimeYZeroTransformBypassFlag      uint32
	FrameCroppingFlag                    uint32
	SeqScalingMatrixPresentFlag          uint32
	VuiParametersPresentFlag             uint32
}

// StdVideoH264SequenceParameterSet - Sequence Parameter Set
type StdVideoH264SequenceParameterSet struct {
	Flags                        StdVideoH264SpsFlags
	ProfileIdc                   H264ProfileIdc
	LevelIdc                     H264LevelIdc
	ChromaFormatIdc              H264ChromaFormatIdc
	SeqParameterSetId            uint8
	BitDepthLumaMinus8           uint8
	BitDepthChromaMinus8         uint8
	Log2MaxFrameNumMinus4        uint8
	PicOrderCntType              uint8
	OffsetForNonRefPic           int32
	OffsetForTopToBottomField    int32
	Log2MaxPicOrderCntLsbMinus4  uint8
	NumRefFramesInPicOrderCntCycle uint8
	MaxNumRefFrames              uint8
	PicWidthInMbsMinus1          uint32
	PicHeightInMapUnitsMinus1    uint32
	FrameCropLeftOffset          uint32
	FrameCropRightOffset         uint32
	FrameCropTopOffset           uint32
	FrameCropBottomOffset        uint32
}

// StdVideoH264PpsFlags - PPS flags
type StdVideoH264PpsFlags struct {
	Transform8x8ModeFlag                   uint32
	RedundantPicCntPresentFlag             uint32
	ConstrainedIntraPredFlag               uint32
	DeblockingFilterControlPresentFlag     uint32
	WeightedPredFlag                       uint32
	BottomFieldPicOrderInFramePresentFlag  uint32
	EntropyCodingModeFlag                  uint32
	PicScalingMatrixPresentFlag            uint32
}

// StdVideoH264PictureParameterSet - Picture Parameter Set
type StdVideoH264PictureParameterSet struct {
	Flags                          StdVideoH264PpsFlags
	SeqParameterSetId              uint8
	PicParameterSetId              uint8
	NumRefIdxL0DefaultActiveMinus1 uint8
	NumRefIdxL1DefaultActiveMinus1 uint8
	WeightedBipredIdc              uint8
	PicInitQpMinus26               int8
	PicInitQsMinus26               int8
	ChromaQpIndexOffset            int8
	SecondChromaQpIndexOffset      int8
}

// ============================================================================
// Vulkan Video Encode H.264 Types
// ============================================================================

// VideoEncodeH264ProfileInfoKHR - H.264 encode profile info
type VideoEncodeH264ProfileInfoKHR struct {
	SType         StructureType
	PNext         unsafe.Pointer
	StdProfileIdc H264ProfileIdc
}

// VideoEncodeH264CapabilitiesKHR is defined in video.go

// VideoEncodeH264SessionParametersAddInfoKHR - Add SPS/PPS to session
type VideoEncodeH264SessionParametersAddInfoKHR struct {
	SType         StructureType
	PNext         unsafe.Pointer
	StdSPSCount   uint32
	PStdSPSs      *StdVideoH264SequenceParameterSet
	StdPPSCount   uint32
	PStdPPSs      *StdVideoH264PictureParameterSet
}

// VideoEncodeH264SessionParametersCreateInfoKHR - Create session parameters
type VideoEncodeH264SessionParametersCreateInfoKHR struct {
	SType             StructureType
	PNext             unsafe.Pointer
	MaxStdSPSCount    uint32
	MaxStdPPSCount    uint32
	PParametersAddInfo *VideoEncodeH264SessionParametersAddInfoKHR
}

// VideoEncodeH264NaluSliceInfoKHR - NALU slice info
type VideoEncodeH264NaluSliceInfoKHR struct {
	SType           StructureType
	PNext           unsafe.Pointer
	ConstantQp      int32
	PStdSliceHeader unsafe.Pointer // *StdVideoEncodeH264SliceHeader
}

// VideoEncodeH264PictureInfoKHR - Per-picture encode info
type VideoEncodeH264PictureInfoKHR struct {
	SType              StructureType
	PNext              unsafe.Pointer
	NaluSliceEntryCount uint32
	PNaluSliceEntries   *VideoEncodeH264NaluSliceInfoKHR
	PStdPictureInfo     unsafe.Pointer // *StdVideoEncodeH264PictureInfo
	GeneratePrefixNalu  bool
}

// VideoEncodeH264DpbSlotInfoKHR - DPB slot info for reference frames
type VideoEncodeH264DpbSlotInfoKHR struct {
	SType              StructureType
	PNext              unsafe.Pointer
	PStdReferenceInfo  unsafe.Pointer // *StdVideoEncodeH264ReferenceInfo
}

// ============================================================================
// H.264 Bitstream Writer - Pure Go implementation
// ============================================================================

// BitstreamWriter writes H.264 bitstream data
type BitstreamWriter struct {
	data      []byte
	bitPos    int
	bytePos   int
}

// NewBitstreamWriter creates a new bitstream writer
func NewBitstreamWriter(capacity int) *BitstreamWriter {
	return &BitstreamWriter{
		data:    make([]byte, 0, capacity),
		bitPos:  0,
		bytePos: 0,
	}
}

// WriteBits writes n bits to the bitstream
func (bw *BitstreamWriter) WriteBits(value uint32, numBits int) {
	for numBits > 0 {
		if bw.bitPos == 0 {
			bw.data = append(bw.data, 0)
		}

		bitsToWrite := 8 - bw.bitPos
		if bitsToWrite > numBits {
			bitsToWrite = numBits
		}

		shift := numBits - bitsToWrite
		mask := uint32((1 << bitsToWrite) - 1)
		bits := (value >> shift) & mask

		bw.data[len(bw.data)-1] |= byte(bits << (8 - bw.bitPos - bitsToWrite))

		bw.bitPos += bitsToWrite
		if bw.bitPos >= 8 {
			bw.bitPos = 0
		}
		numBits -= bitsToWrite
	}
}

// WriteBit writes a single bit
func (bw *BitstreamWriter) WriteBit(value uint32) {
	bw.WriteBits(value&1, 1)
}

// WriteUE writes an unsigned Exp-Golomb coded value
func (bw *BitstreamWriter) WriteUE(value uint32) {
	// Exp-Golomb encoding: leadingZeroBits zeros, 1, (value+1) in binary minus leading 1
	value++ // Add 1 as per Exp-Golomb
	leadingZeroBits := 0
	temp := value
	for temp > 1 {
		temp >>= 1
		leadingZeroBits++
	}

	// Write leading zeros
	for i := 0; i < leadingZeroBits; i++ {
		bw.WriteBit(0)
	}
	// Write the value
	bw.WriteBits(value, leadingZeroBits+1)
}

// WriteSE writes a signed Exp-Golomb coded value
func (bw *BitstreamWriter) WriteSE(value int32) {
	var codeNum uint32
	if value <= 0 {
		codeNum = uint32(-value) * 2
	} else {
		codeNum = uint32(value)*2 - 1
	}
	bw.WriteUE(codeNum)
}

// WriteBytes writes raw bytes (must be byte-aligned)
func (bw *BitstreamWriter) WriteBytes(data []byte) {
	// Align to byte boundary first
	if bw.bitPos != 0 {
		bw.WriteBits(0, 8-bw.bitPos)
	}
	bw.data = append(bw.data, data...)
}

// ByteAlign writes RBSP trailing bits (stop bit + alignment zeros)
// This MUST always write at least the stop bit, even when byte-aligned
func (bw *BitstreamWriter) ByteAlign() {
	bw.WriteBit(1) // rbsp_stop_one_bit - ALWAYS written
	for bw.bitPos != 0 {
		bw.WriteBit(0) // rbsp_alignment_zero_bit
	}
}

// Data returns the written data
func (bw *BitstreamWriter) Data() []byte {
	return bw.data
}

// ============================================================================
// NAL Unit Helpers
// ============================================================================

// NALStartCode is the Annex B start code
var NALStartCode = []byte{0x00, 0x00, 0x00, 0x01}
var NALStartCode3 = []byte{0x00, 0x00, 0x01}

// WriteNALUnit wraps RBSP data in a NAL unit with start code
func WriteNALUnit(nalType H264NalUnitType, refIdc uint8, rbsp []byte) []byte {
	// Escape prevention: replace 00 00 00/01/02/03 with 00 00 03 00/01/02/03
	escaped := make([]byte, 0, len(rbsp)*2)
	zeroCount := 0
	for _, b := range rbsp {
		if zeroCount >= 2 && b <= 3 {
			escaped = append(escaped, 0x03) // Emulation prevention byte
			zeroCount = 0
		}
		escaped = append(escaped, b)
		if b == 0 {
			zeroCount++
		} else {
			zeroCount = 0
		}
	}

	// NAL header: forbidden_zero_bit (1) | nal_ref_idc (2) | nal_unit_type (5)
	nalHeader := (refIdc << 5) | uint8(nalType)

	result := make([]byte, 0, 4+1+len(escaped))
	result = append(result, NALStartCode...)
	result = append(result, nalHeader)
	result = append(result, escaped...)

	return result
}

// ============================================================================
// SPS/PPS Generation - Pure Go
// ============================================================================

// H264EncoderConfig holds encoder configuration
type H264EncoderConfig struct {
	Width           uint32
	Height          uint32
	FrameRateNum    uint32
	FrameRateDen    uint32
	Profile         H264ProfileIdc
	Level           H264LevelIdc
	BitRate         uint32 // bits per second
	GOPSize         uint32 // frames between I-frames
	MaxBFrames      uint32
	QP              int32  // Quantization parameter (0-51)
}

// DefaultH264Config returns a default encoder configuration
func DefaultH264Config(width, height uint32) H264EncoderConfig {
	return H264EncoderConfig{
		Width:        width,
		Height:       height,
		FrameRateNum: 24,
		FrameRateDen: 1,
		Profile:      H264_PROFILE_IDC_BASELINE, // Baseline uses CAVLC, simpler I_PCM encoding
		Level:        H264_LEVEL_IDC_4_1,
		BitRate:      5000000, // 5 Mbps
		GOPSize:      30,
		MaxBFrames:   0, // No B-frames for simplicity
		QP:           23,
	}
}

// GenerateSPS generates an H.264 Sequence Parameter Set
func GenerateSPS(config H264EncoderConfig, spsId uint8) []byte {
	bw := NewBitstreamWriter(256)

	// profile_idc
	bw.WriteBits(uint32(config.Profile), 8)

	// constraint_set flags (6 bits) + reserved (2 bits)
	constraints := uint32(0)
	if config.Profile == H264_PROFILE_IDC_BASELINE {
		constraints |= 0x80 // constraint_set0_flag
	}
	if config.Profile == H264_PROFILE_IDC_MAIN || config.Profile == H264_PROFILE_IDC_HIGH {
		constraints |= 0x40 // constraint_set1_flag
	}
	bw.WriteBits(constraints, 8)

	// level_idc
	bw.WriteBits(uint32(config.Level), 8)

	// seq_parameter_set_id
	bw.WriteUE(uint32(spsId))

	// High profile specific
	if config.Profile == H264_PROFILE_IDC_HIGH || config.Profile == H264_PROFILE_IDC_HIGH_444_PREDICTIVE {
		// chroma_format_idc
		bw.WriteUE(1) // 4:2:0
		// bit_depth_luma_minus8
		bw.WriteUE(0)
		// bit_depth_chroma_minus8
		bw.WriteUE(0)
		// qpprime_y_zero_transform_bypass_flag
		bw.WriteBit(0)
		// seq_scaling_matrix_present_flag
		bw.WriteBit(0)
	}

	// log2_max_frame_num_minus4
	bw.WriteUE(0) // max_frame_num = 16

	// pic_order_cnt_type
	bw.WriteUE(2) // Type 2: POC derived from frame_num (simplest, no POC LSB field)

	// max_num_ref_frames
	bw.WriteUE(1)

	// gaps_in_frame_num_value_allowed_flag
	bw.WriteBit(0)

	// pic_width_in_mbs_minus1
	mbWidth := (config.Width + 15) / 16
	bw.WriteUE(mbWidth - 1)

	// pic_height_in_map_units_minus1
	mbHeight := (config.Height + 15) / 16
	bw.WriteUE(mbHeight - 1)

	// frame_mbs_only_flag
	bw.WriteBit(1) // Progressive only

	// direct_8x8_inference_flag
	bw.WriteBit(1)

	// frame_cropping_flag
	cropWidth := mbWidth*16 - config.Width
	cropHeight := mbHeight*16 - config.Height
	if cropWidth > 0 || cropHeight > 0 {
		bw.WriteBit(1)
		bw.WriteUE(0)             // left
		bw.WriteUE(cropWidth / 2) // right (in chroma samples for 4:2:0)
		bw.WriteUE(0)             // top
		bw.WriteUE(cropHeight / 2) // bottom
	} else {
		bw.WriteBit(0)
	}

	// vui_parameters_present_flag - disable VUI for simplicity
	bw.WriteBit(0)

	// rbsp_trailing_bits
	bw.ByteAlign()

	return WriteNALUnit(H264_NAL_UNIT_TYPE_SPS, 3, bw.Data())
}

// GeneratePPS generates an H.264 Picture Parameter Set
func GeneratePPS(config H264EncoderConfig, spsId, ppsId uint8) []byte {
	bw := NewBitstreamWriter(64)

	// pic_parameter_set_id
	bw.WriteUE(uint32(ppsId))

	// seq_parameter_set_id
	bw.WriteUE(uint32(spsId))

	// entropy_coding_mode_flag (0=CAVLC, 1=CABAC)
	if config.Profile == H264_PROFILE_IDC_HIGH || config.Profile == H264_PROFILE_IDC_MAIN {
		bw.WriteBit(1) // CABAC for better compression
	} else {
		bw.WriteBit(0) // CAVLC for baseline
	}

	// bottom_field_pic_order_in_frame_present_flag
	bw.WriteBit(0)

	// num_slice_groups_minus1
	bw.WriteUE(0)

	// num_ref_idx_l0_default_active_minus1
	bw.WriteUE(0)

	// num_ref_idx_l1_default_active_minus1
	bw.WriteUE(0)

	// weighted_pred_flag
	bw.WriteBit(0)

	// weighted_bipred_idc
	bw.WriteBits(0, 2)

	// pic_init_qp_minus26
	bw.WriteSE(config.QP - 26)

	// pic_init_qs_minus26
	bw.WriteSE(0)

	// chroma_qp_index_offset
	bw.WriteSE(0)

	// deblocking_filter_control_present_flag
	bw.WriteBit(1)

	// constrained_intra_pred_flag
	bw.WriteBit(0)

	// redundant_pic_cnt_present_flag
	bw.WriteBit(0)

	// High profile additions
	if config.Profile == H264_PROFILE_IDC_HIGH || config.Profile == H264_PROFILE_IDC_HIGH_444_PREDICTIVE {
		// transform_8x8_mode_flag
		bw.WriteBit(1)
		// pic_scaling_matrix_present_flag
		bw.WriteBit(0)
		// second_chroma_qp_index_offset
		bw.WriteSE(0)
	}

	// rbsp_trailing_bits
	bw.ByteAlign()

	return WriteNALUnit(H264_NAL_UNIT_TYPE_PPS, 3, bw.Data())
}

// GenerateSliceHeader generates a slice header for I or P frame
func GenerateSliceHeader(config H264EncoderConfig, frameNum uint32, isIDR bool, ppsId uint8) []byte {
	bw := NewBitstreamWriter(64)

	// first_mb_in_slice
	bw.WriteUE(0)

	// slice_type
	if isIDR {
		bw.WriteUE(7) // I slice (all macroblocks are I)
	} else {
		bw.WriteUE(5) // P slice
	}

	// pic_parameter_set_id
	bw.WriteUE(uint32(ppsId))

	// frame_num (log2_max_frame_num_minus4 = 0, so 4 bits)
	bw.WriteBits(frameNum&0xF, 4)

	// IDR-specific
	if isIDR {
		// idr_pic_id
		bw.WriteUE(0)
	}

	// pic_order_cnt_type = 0, so write pic_order_cnt_lsb
	// log2_max_pic_order_cnt_lsb_minus4 = 0, so 4 bits
	poc := frameNum * 2
	bw.WriteBits(poc&0xF, 4)

	// P-slice reference list
	if !isIDR {
		// num_ref_idx_active_override_flag
		bw.WriteBit(0)
		// ref_pic_list_modification_flag_l0
		bw.WriteBit(0)
	}

	// dec_ref_pic_marking
	if isIDR {
		// no_output_of_prior_pics_flag
		bw.WriteBit(0)
		// long_term_reference_flag
		bw.WriteBit(0)
	} else {
		// adaptive_ref_pic_marking_mode_flag
		bw.WriteBit(0)
	}

	// CABAC: cabac_init_idc for P/B slices
	if !isIDR && (config.Profile == H264_PROFILE_IDC_HIGH || config.Profile == H264_PROFILE_IDC_MAIN) {
		bw.WriteUE(0)
	}

	// slice_qp_delta
	bw.WriteSE(0)

	// deblocking_filter_control_present_flag = 1, so:
	// disable_deblocking_filter_idc
	bw.WriteUE(0) // Enabled
	// slice_alpha_c0_offset_div2
	bw.WriteSE(0)
	// slice_beta_offset_div2
	bw.WriteSE(0)

	return bw.Data()
}

// ============================================================================
// MP4/MOV Container Writer (Simple version)
// ============================================================================

// MP4Writer writes H.264 data to an MP4 container
type MP4Writer struct {
	data           []byte
	sps            []byte
	pps            []byte
	samples        []mp4Sample
	width          uint32
	height         uint32
	frameRateNum   uint32
	frameRateDen   uint32
	timescale      uint32
}

type mp4Sample struct {
	data     []byte
	duration uint32
	isKey    bool
}

// NewMP4Writer creates a new MP4 writer
func NewMP4Writer(config H264EncoderConfig, sps, pps []byte) *MP4Writer {
	return &MP4Writer{
		data:         make([]byte, 0, 1024*1024),
		sps:          sps,
		pps:          pps,
		samples:      make([]mp4Sample, 0),
		width:        config.Width,
		height:       config.Height,
		frameRateNum: config.FrameRateNum,
		frameRateDen: config.FrameRateDen,
		timescale:    config.FrameRateNum,
	}
}

// AddFrame adds an encoded frame to the MP4
func (w *MP4Writer) AddFrame(nalData []byte, isKeyFrame bool) {
	// Convert Annex B to AVCC format (length-prefixed)
	avccData := annexBToAVCC(nalData)

	w.samples = append(w.samples, mp4Sample{
		data:     avccData,
		duration: w.frameRateDen,
		isKey:    isKeyFrame,
	})
}

// Finalize writes the complete MP4 file
func (w *MP4Writer) Finalize() []byte {
	var buf []byte

	// ftyp box
	buf = append(buf, w.writeFtyp()...)

	// mdat box (media data)
	mdatStart := len(buf)
	buf = append(buf, w.writeMdat()...)
	mdatSize := len(buf) - mdatStart

	// moov box (movie metadata)
	buf = append(buf, w.writeMoov(uint32(mdatStart), uint32(mdatSize))...)

	return buf
}

func (w *MP4Writer) writeFtyp() []byte {
	box := make([]byte, 0, 32)
	box = append(box, 0, 0, 0, 0x18) // size = 24
	box = append(box, 'f', 't', 'y', 'p')
	box = append(box, 'i', 's', 'o', 'm') // major brand
	box = append(box, 0, 0, 0, 1)         // minor version
	box = append(box, 'i', 's', 'o', 'm') // compatible brands
	box = append(box, 'a', 'v', 'c', '1')
	return box
}

func (w *MP4Writer) writeMdat() []byte {
	// Calculate total size
	totalSize := 8 // box header
	for _, s := range w.samples {
		totalSize += len(s.data)
	}

	box := make([]byte, 0, totalSize)
	// Size (32-bit)
	box = append(box, byte(totalSize>>24), byte(totalSize>>16), byte(totalSize>>8), byte(totalSize))
	box = append(box, 'm', 'd', 'a', 't')

	for _, s := range w.samples {
		box = append(box, s.data...)
	}

	return box
}

func (w *MP4Writer) writeMoov(mdatOffset, mdatSize uint32) []byte {
	// Build moov box with mvhd, trak, etc.
	// This is a simplified version

	mvhd := w.writeMvhd()
	trak := w.writeTrak(mdatOffset)

	moovSize := 8 + len(mvhd) + len(trak)
	box := make([]byte, 0, moovSize)
	box = append(box, byte(moovSize>>24), byte(moovSize>>16), byte(moovSize>>8), byte(moovSize))
	box = append(box, 'm', 'o', 'o', 'v')
	box = append(box, mvhd...)
	box = append(box, trak...)

	return box
}

func (w *MP4Writer) writeMvhd() []byte {
	duration := uint32(len(w.samples)) * w.frameRateDen

	box := make([]byte, 108)
	// Size
	binary.BigEndian.PutUint32(box[0:4], 108)
	copy(box[4:8], "mvhd")
	// Version and flags
	box[8] = 0
	// Creation/modification time (skip)
	// Timescale
	binary.BigEndian.PutUint32(box[20:24], w.timescale)
	// Duration
	binary.BigEndian.PutUint32(box[24:28], duration)
	// Rate (1.0 = 0x00010000)
	binary.BigEndian.PutUint32(box[28:32], 0x00010000)
	// Volume (1.0 = 0x0100)
	binary.BigEndian.PutUint16(box[32:34], 0x0100)
	// Matrix (identity)
	binary.BigEndian.PutUint32(box[48:52], 0x00010000)
	binary.BigEndian.PutUint32(box[64:68], 0x00010000)
	binary.BigEndian.PutUint32(box[80:84], 0x40000000)
	// Next track ID
	binary.BigEndian.PutUint32(box[104:108], 2)

	return box
}

func (w *MP4Writer) writeTrak(mdatOffset uint32) []byte {
	tkhd := w.writeTkhd()
	mdia := w.writeMdia(mdatOffset)

	trakSize := 8 + len(tkhd) + len(mdia)
	box := make([]byte, 0, trakSize)
	box = append(box, byte(trakSize>>24), byte(trakSize>>16), byte(trakSize>>8), byte(trakSize))
	box = append(box, 't', 'r', 'a', 'k')
	box = append(box, tkhd...)
	box = append(box, mdia...)

	return box
}

func (w *MP4Writer) writeTkhd() []byte {
	duration := uint32(len(w.samples)) * w.frameRateDen

	box := make([]byte, 92)
	binary.BigEndian.PutUint32(box[0:4], 92)
	copy(box[4:8], "tkhd")
	box[8] = 0                                   // Version
	box[9], box[10], box[11] = 0, 0, 3           // Flags (track enabled)
	binary.BigEndian.PutUint32(box[20:24], 1)    // Track ID
	binary.BigEndian.PutUint32(box[28:32], duration)
	// Matrix
	binary.BigEndian.PutUint32(box[48:52], 0x00010000)
	binary.BigEndian.PutUint32(box[64:68], 0x00010000)
	binary.BigEndian.PutUint32(box[80:84], 0x40000000)
	// Width/height (16.16 fixed point)
	binary.BigEndian.PutUint32(box[84:88], w.width<<16)
	binary.BigEndian.PutUint32(box[88:92], w.height<<16)

	return box
}

func (w *MP4Writer) writeMdia(mdatOffset uint32) []byte {
	mdhd := w.writeMdhd()
	hdlr := w.writeHdlr()
	minf := w.writeMinf(mdatOffset)

	mdiaSize := 8 + len(mdhd) + len(hdlr) + len(minf)
	box := make([]byte, 0, mdiaSize)
	box = append(box, byte(mdiaSize>>24), byte(mdiaSize>>16), byte(mdiaSize>>8), byte(mdiaSize))
	box = append(box, 'm', 'd', 'i', 'a')
	box = append(box, mdhd...)
	box = append(box, hdlr...)
	box = append(box, minf...)

	return box
}

func (w *MP4Writer) writeMdhd() []byte {
	duration := uint32(len(w.samples)) * w.frameRateDen

	box := make([]byte, 32)
	binary.BigEndian.PutUint32(box[0:4], 32)
	copy(box[4:8], "mdhd")
	binary.BigEndian.PutUint32(box[20:24], w.timescale)
	binary.BigEndian.PutUint32(box[24:28], duration)
	box[28], box[29] = 0x55, 0xC4 // Language: und

	return box
}

func (w *MP4Writer) writeHdlr() []byte {
	box := make([]byte, 45)
	binary.BigEndian.PutUint32(box[0:4], 45)
	copy(box[4:8], "hdlr")
	copy(box[16:20], "vide")
	copy(box[32:45], "VideoHandler")

	return box
}

func (w *MP4Writer) writeMinf(mdatOffset uint32) []byte {
	vmhd := w.writeVmhd()
	dinf := w.writeDinf()
	stbl := w.writeStbl(mdatOffset)

	minfSize := 8 + len(vmhd) + len(dinf) + len(stbl)
	box := make([]byte, 0, minfSize)
	box = append(box, byte(minfSize>>24), byte(minfSize>>16), byte(minfSize>>8), byte(minfSize))
	box = append(box, 'm', 'i', 'n', 'f')
	box = append(box, vmhd...)
	box = append(box, dinf...)
	box = append(box, stbl...)

	return box
}

func (w *MP4Writer) writeVmhd() []byte {
	box := make([]byte, 20)
	binary.BigEndian.PutUint32(box[0:4], 20)
	copy(box[4:8], "vmhd")
	box[8] = 0
	box[9], box[10], box[11] = 0, 0, 1 // Flags

	return box
}

func (w *MP4Writer) writeDinf() []byte {
	// dref inside dinf
	dref := make([]byte, 28)
	binary.BigEndian.PutUint32(dref[0:4], 28)
	copy(dref[4:8], "dref")
	binary.BigEndian.PutUint32(dref[12:16], 1) // Entry count
	binary.BigEndian.PutUint32(dref[16:20], 12)
	copy(dref[20:24], "url ")
	dref[24] = 0
	dref[25], dref[26], dref[27] = 0, 0, 1 // Self-contained flag

	dinfSize := 8 + len(dref)
	box := make([]byte, 0, dinfSize)
	box = append(box, byte(dinfSize>>24), byte(dinfSize>>16), byte(dinfSize>>8), byte(dinfSize))
	box = append(box, 'd', 'i', 'n', 'f')
	box = append(box, dref...)

	return box
}

func (w *MP4Writer) writeStbl(mdatOffset uint32) []byte {
	stsd := w.writeStsd()
	stts := w.writeStts()
	stsc := w.writeStsc()
	stsz := w.writeStsz()
	stco := w.writeStco(mdatOffset)
	stss := w.writeStss()

	stblSize := 8 + len(stsd) + len(stts) + len(stsc) + len(stsz) + len(stco) + len(stss)
	box := make([]byte, 0, stblSize)
	box = append(box, byte(stblSize>>24), byte(stblSize>>16), byte(stblSize>>8), byte(stblSize))
	box = append(box, 's', 't', 'b', 'l')
	box = append(box, stsd...)
	box = append(box, stts...)
	box = append(box, stsc...)
	box = append(box, stsz...)
	box = append(box, stco...)
	box = append(box, stss...)

	return box
}

func (w *MP4Writer) writeStsd() []byte {
	// Build avcC box (AVC decoder configuration record)
	avcC := w.writeAvcC()

	// avc1 sample entry (VisualSampleEntry)
	// Structure: size(4) + type(4) + reserved(6) + data_ref_idx(2) +
	//            pre_defined(2) + reserved(2) + pre_defined(12) +
	//            width(2) + height(2) + hres(4) + vres(4) + reserved(4) +
	//            frame_count(2) + compressorname(32) + depth(2) + pre_defined(2)
	// Total fixed = 86 bytes
	avc1Size := 86 + len(avcC)
	avc1 := make([]byte, 86)
	binary.BigEndian.PutUint32(avc1[0:4], uint32(avc1Size))
	copy(avc1[4:8], "avc1")
	// reserved[6] at 8-13: zeros
	binary.BigEndian.PutUint16(avc1[14:16], 1) // data_reference_index
	// pre_defined at 16-17: zeros
	// reserved at 18-19: zeros
	// pre_defined[3] at 20-31: zeros
	binary.BigEndian.PutUint16(avc1[32:34], uint16(w.width))  // width
	binary.BigEndian.PutUint16(avc1[34:36], uint16(w.height)) // height
	binary.BigEndian.PutUint32(avc1[36:40], 0x00480000)       // horizresolution (72.0 dpi)
	binary.BigEndian.PutUint32(avc1[40:44], 0x00480000)       // vertresolution (72.0 dpi)
	// reserved at 44-47: zeros
	binary.BigEndian.PutUint16(avc1[48:50], 1) // frame_count
	// compressorname[32] at 50-81: zeros (no name)
	binary.BigEndian.PutUint16(avc1[82:84], 0x0018) // depth (24-bit)
	binary.BigEndian.PutUint16(avc1[84:86], 0xFFFF) // pre_defined (-1)

	avc1Full := append(avc1, avcC...)
	binary.BigEndian.PutUint32(avc1Full[0:4], uint32(len(avc1Full)))

	stsdSize := 16 + len(avc1Full)
	box := make([]byte, 16)
	binary.BigEndian.PutUint32(box[0:4], uint32(stsdSize))
	copy(box[4:8], "stsd")
	binary.BigEndian.PutUint32(box[12:16], 1) // Entry count

	return append(box, avc1Full...)
}

func (w *MP4Writer) writeAvcC() []byte {
	// Strip start codes from SPS/PPS
	sps := stripStartCode(w.sps)
	pps := stripStartCode(w.pps)

	// Size = header(8) + config(5) + numSPS(1) + spsLen(2) + sps + numPPS(1) + ppsLen(2) + pps
	size := 8 + 5 + 3 + len(sps) + 3 + len(pps)
	box := make([]byte, 0, size)

	// Box header
	box = append(box, byte(size>>24), byte(size>>16), byte(size>>8), byte(size))
	box = append(box, 'a', 'v', 'c', 'C')

	// AVC decoder config
	box = append(box, 1)          // configurationVersion
	box = append(box, sps[1])     // AVCProfileIndication
	box = append(box, sps[2])     // profile_compatibility
	box = append(box, sps[3])     // AVCLevelIndication
	box = append(box, 0xFF)       // lengthSizeMinusOne (4 bytes)

	// SPS
	box = append(box, 0xE1) // numOfSequenceParameterSets (1)
	box = append(box, byte(len(sps)>>8), byte(len(sps)))
	box = append(box, sps...)

	// PPS
	box = append(box, 1) // numOfPictureParameterSets
	box = append(box, byte(len(pps)>>8), byte(len(pps)))
	box = append(box, pps...)

	return box
}

func (w *MP4Writer) writeStts() []byte {
	// All frames have same duration
	box := make([]byte, 24)
	binary.BigEndian.PutUint32(box[0:4], 24)
	copy(box[4:8], "stts")
	binary.BigEndian.PutUint32(box[12:16], 1) // Entry count
	binary.BigEndian.PutUint32(box[16:20], uint32(len(w.samples)))
	binary.BigEndian.PutUint32(box[20:24], w.frameRateDen)

	return box
}

func (w *MP4Writer) writeStsc() []byte {
	// One sample per chunk
	box := make([]byte, 28)
	binary.BigEndian.PutUint32(box[0:4], 28)
	copy(box[4:8], "stsc")
	binary.BigEndian.PutUint32(box[12:16], 1) // Entry count
	binary.BigEndian.PutUint32(box[16:20], 1) // First chunk
	binary.BigEndian.PutUint32(box[20:24], 1) // Samples per chunk
	binary.BigEndian.PutUint32(box[24:28], 1) // Sample description index

	return box
}

func (w *MP4Writer) writeStsz() []byte {
	box := make([]byte, 20+4*len(w.samples))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], "stsz")
	// sample_size = 0 (variable)
	binary.BigEndian.PutUint32(box[16:20], uint32(len(w.samples)))

	for i, s := range w.samples {
		binary.BigEndian.PutUint32(box[20+i*4:24+i*4], uint32(len(s.data)))
	}

	return box
}

func (w *MP4Writer) writeStco(mdatOffset uint32) []byte {
	box := make([]byte, 16+4*len(w.samples))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], "stco")
	binary.BigEndian.PutUint32(box[12:16], uint32(len(w.samples)))

	offset := mdatOffset + 8 // After mdat header
	for i, s := range w.samples {
		binary.BigEndian.PutUint32(box[16+i*4:20+i*4], offset)
		offset += uint32(len(s.data))
	}

	return box
}

func (w *MP4Writer) writeStss() []byte {
	// Sync sample table (keyframes)
	keyframes := make([]uint32, 0)
	for i, s := range w.samples {
		if s.isKey {
			keyframes = append(keyframes, uint32(i+1))
		}
	}

	box := make([]byte, 16+4*len(keyframes))
	binary.BigEndian.PutUint32(box[0:4], uint32(len(box)))
	copy(box[4:8], "stss")
	binary.BigEndian.PutUint32(box[12:16], uint32(len(keyframes)))

	for i, kf := range keyframes {
		binary.BigEndian.PutUint32(box[16+i*4:20+i*4], kf)
	}

	return box
}

// Helper functions

func stripStartCode(nal []byte) []byte {
	if len(nal) >= 4 && nal[0] == 0 && nal[1] == 0 && nal[2] == 0 && nal[3] == 1 {
		return nal[4:]
	}
	if len(nal) >= 3 && nal[0] == 0 && nal[1] == 0 && nal[2] == 1 {
		return nal[3:]
	}
	return nal
}

func annexBToAVCC(nalData []byte) []byte {
	// Find NAL units and convert to length-prefixed
	result := make([]byte, 0, len(nalData))

	i := 0
	for i < len(nalData) {
		// Find start code
		startCodeLen := 0
		if i+4 <= len(nalData) && nalData[i] == 0 && nalData[i+1] == 0 && nalData[i+2] == 0 && nalData[i+3] == 1 {
			startCodeLen = 4
		} else if i+3 <= len(nalData) && nalData[i] == 0 && nalData[i+1] == 0 && nalData[i+2] == 1 {
			startCodeLen = 3
		} else {
			i++
			continue
		}

		nalStart := i + startCodeLen

		// Find next start code or end
		nalEnd := len(nalData)
		for j := nalStart; j < len(nalData)-3; j++ {
			if nalData[j] == 0 && nalData[j+1] == 0 {
				if (j+2 < len(nalData) && nalData[j+2] == 1) ||
					(j+3 < len(nalData) && nalData[j+2] == 0 && nalData[j+3] == 1) {
					nalEnd = j
					break
				}
			}
		}

		nalLen := nalEnd - nalStart
		if nalLen > 0 {
			// Write 4-byte length prefix
			result = append(result, byte(nalLen>>24), byte(nalLen>>16), byte(nalLen>>8), byte(nalLen))
			result = append(result, nalData[nalStart:nalEnd]...)
		}

		i = nalEnd
	}

	return result
}
