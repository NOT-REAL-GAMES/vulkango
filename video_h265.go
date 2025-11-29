package vulkango

import "os"

// HEVC/H.265 NAL unit types
const (
	HEVC_NAL_TRAIL_N        = 0
	HEVC_NAL_TRAIL_R        = 1
	HEVC_NAL_TSA_N          = 2
	HEVC_NAL_TSA_R          = 3
	HEVC_NAL_STSA_N         = 4
	HEVC_NAL_STSA_R         = 5
	HEVC_NAL_RADL_N         = 6
	HEVC_NAL_RADL_R         = 7
	HEVC_NAL_RASL_N         = 8
	HEVC_NAL_RASL_R         = 9
	HEVC_NAL_BLA_W_LP       = 16
	HEVC_NAL_BLA_W_RADL     = 17
	HEVC_NAL_BLA_N_LP       = 18
	HEVC_NAL_IDR_W_RADL     = 19
	HEVC_NAL_IDR_N_LP       = 20
	HEVC_NAL_CRA_NUT        = 21
	HEVC_NAL_VPS            = 32
	HEVC_NAL_SPS            = 33
	HEVC_NAL_PPS            = 34
	HEVC_NAL_AUD            = 35
	HEVC_NAL_EOS            = 36
	HEVC_NAL_EOB            = 37
	HEVC_NAL_FD             = 38
	HEVC_NAL_PREFIX_SEI     = 39
	HEVC_NAL_SUFFIX_SEI     = 40
)

// HEVC profiles
const (
	HEVC_PROFILE_MAIN       = 1
	HEVC_PROFILE_MAIN_10    = 2
	HEVC_PROFILE_MAIN_STILL = 3
	HEVC_PROFILE_REXT       = 4 // Range Extensions (for 4:4:4, alpha, etc.)
)

// HEVC levels (level_idc values)
const (
	HEVC_LEVEL_1   = 30  // Level 1
	HEVC_LEVEL_2   = 60  // Level 2
	HEVC_LEVEL_2_1 = 63  // Level 2.1
	HEVC_LEVEL_3   = 90  // Level 3
	HEVC_LEVEL_3_1 = 93  // Level 3.1
	HEVC_LEVEL_4   = 120 // Level 4
	HEVC_LEVEL_4_1 = 123 // Level 4.1
	HEVC_LEVEL_5   = 150 // Level 5
	HEVC_LEVEL_5_1 = 153 // Level 5.1
	HEVC_LEVEL_5_2 = 156 // Level 5.2
	HEVC_LEVEL_6   = 180 // Level 6
	HEVC_LEVEL_6_1 = 183 // Level 6.1
	HEVC_LEVEL_6_2 = 186 // Level 6.2
)

// HEVCEncoderConfig holds configuration for HEVC encoding
type HEVCEncoderConfig struct {
	Width        uint32
	Height       uint32
	FrameRateNum uint32
	FrameRateDen uint32
	Profile      uint32
	Level        uint32
	GOPSize      uint32
	BitDepth     uint32
	ChromaFormat uint32 // 1 = 4:2:0, 2 = 4:2:2, 3 = 4:4:4
	HasAlpha     bool
}

// DefaultHEVCConfig returns a default HEVC encoder configuration
func DefaultHEVCConfig(width, height uint32) HEVCEncoderConfig {
	return HEVCEncoderConfig{
		Width:        width,
		Height:       height,
		FrameRateNum: 24,
		FrameRateDen: 1,
		Profile:      HEVC_PROFILE_MAIN,
		Level:        HEVC_LEVEL_4_1,
		GOPSize:      30,
		BitDepth:     8,
		ChromaFormat: 1, // 4:2:0
		HasAlpha:     false,
	}
}

// DefaultHEVCConfigWithAlpha returns config for HEVC with alpha channel
func DefaultHEVCConfigWithAlpha(width, height uint32) HEVCEncoderConfig {
	return HEVCEncoderConfig{
		Width:        width,
		Height:       height,
		FrameRateNum: 24,
		FrameRateDen: 1,
		Profile:      HEVC_PROFILE_MAIN, // Main profile for basic 8-bit 4:2:0
		Level:        HEVC_LEVEL_4_1,
		GOPSize:      30,
		BitDepth:     8,
		ChromaFormat: 1, // 4:2:0 for main, alpha is monochrome
		HasAlpha:     true,
	}
}

// WriteHEVCNALUnit wraps data in an HEVC NAL unit with start code
// HEVC NAL header is 2 bytes: forbidden_zero_bit(1) + nal_unit_type(6) + nuh_layer_id(6) + nuh_temporal_id_plus1(3)
func WriteHEVCNALUnit(nalType uint8, temporalID uint8, data []byte) []byte {
	// Start code (4 bytes) + NAL header (2 bytes) + data
	result := make([]byte, 4+2+len(data))

	// Start code
	result[0] = 0x00
	result[1] = 0x00
	result[2] = 0x00
	result[3] = 0x01

	// NAL header byte 1: forbidden_zero_bit(0) + nal_unit_type(6 bits)
	result[4] = (nalType & 0x3F) << 1

	// NAL header byte 2: nuh_layer_id(6 bits, typically 0) + nuh_temporal_id_plus1(3 bits)
	result[5] = (temporalID + 1) & 0x07

	// Copy payload
	copy(result[6:], data)

	return result
}

// WriteHEVCNALUnitAVCC writes NAL unit with 4-byte length prefix (for MP4/MOV)
func WriteHEVCNALUnitAVCC(nalType uint8, temporalID uint8, data []byte) []byte {
	nalSize := 2 + len(data) // NAL header (2 bytes) + data
	result := make([]byte, 4+nalSize)

	// 4-byte length prefix (big-endian)
	result[0] = byte(nalSize >> 24)
	result[1] = byte(nalSize >> 16)
	result[2] = byte(nalSize >> 8)
	result[3] = byte(nalSize)

	// NAL header
	result[4] = (nalType & 0x3F) << 1
	result[5] = (temporalID + 1) & 0x07

	// Copy payload
	copy(result[6:], data)

	return result
}

// GenerateHEVCVPS generates a Video Parameter Set for HEVC
func GenerateHEVCVPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(64)

	// vps_video_parameter_set_id (4 bits)
	bw.WriteBits(0, 4)

	// vps_base_layer_internal_flag
	bw.WriteBit(1)

	// vps_base_layer_available_flag
	bw.WriteBit(1)

	// vps_max_layers_minus1 (6 bits)
	bw.WriteBits(0, 6)

	// vps_max_sub_layers_minus1 (3 bits)
	bw.WriteBits(0, 3)

	// vps_temporal_id_nesting_flag
	bw.WriteBit(1)

	// vps_reserved_0xffff_16bits
	bw.WriteBits(0xFFFF, 16)

	// profile_tier_level
	writeProfileTierLevel(bw, config, true, 0)

	// vps_sub_layer_ordering_info_present_flag
	bw.WriteBit(1)

	// For each sub-layer (just one for now)
	// vps_max_dec_pic_buffering_minus1[0]
	bw.WriteUE(4)
	// vps_max_num_reorder_pics[0]
	bw.WriteUE(0)
	// vps_max_latency_increase_plus1[0]
	bw.WriteUE(0)

	// vps_max_layer_id (6 bits)
	bw.WriteBits(0, 6)

	// vps_num_layer_sets_minus1
	bw.WriteUE(0)

	// vps_timing_info_present_flag
	bw.WriteBit(0)

	// vps_extension_flag
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// GenerateHEVCSPS generates a Sequence Parameter Set for HEVC
func GenerateHEVCSPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(256)

	// sps_video_parameter_set_id (4 bits)
	bw.WriteBits(0, 4)

	// sps_max_sub_layers_minus1 (3 bits)
	bw.WriteBits(0, 3)

	// sps_temporal_id_nesting_flag
	bw.WriteBit(1)

	// profile_tier_level
	writeProfileTierLevel(bw, config, true, 0)

	// sps_seq_parameter_set_id
	bw.WriteUE(0)

	// chroma_format_idc
	bw.WriteUE(uint32(config.ChromaFormat))

	// For 4:4:4, need separate_colour_plane_flag
	if config.ChromaFormat == 3 {
		bw.WriteBit(0) // separate_colour_plane_flag
	}

	// pic_width_in_luma_samples
	bw.WriteUE(config.Width)

	// pic_height_in_luma_samples
	bw.WriteUE(config.Height)

	// conformance_window_flag
	// Check if we need conformance window (dimensions not multiple of min CU size)
	needConformanceWindow := (config.Width%8 != 0) || (config.Height%8 != 0)
	if needConformanceWindow {
		bw.WriteBit(1)
		// Pad to multiple of 8
		padWidth := (8 - (config.Width % 8)) % 8
		padHeight := (8 - (config.Height % 8)) % 8
		bw.WriteUE(0)            // conf_win_left_offset
		bw.WriteUE(padWidth / 2) // conf_win_right_offset (in chroma samples)
		bw.WriteUE(0)            // conf_win_top_offset
		bw.WriteUE(padHeight / 2) // conf_win_bottom_offset
	} else {
		bw.WriteBit(0)
	}

	// bit_depth_luma_minus8
	bw.WriteUE(config.BitDepth - 8)

	// bit_depth_chroma_minus8
	bw.WriteUE(config.BitDepth - 8)

	// log2_max_pic_order_cnt_lsb_minus4
	bw.WriteUE(4) // 8 bits for POC

	// sps_sub_layer_ordering_info_present_flag
	bw.WriteBit(1)

	// For each sub-layer
	bw.WriteUE(4) // sps_max_dec_pic_buffering_minus1[0]
	bw.WriteUE(0) // sps_max_num_reorder_pics[0]
	bw.WriteUE(0) // sps_max_latency_increase_plus1[0]

	// log2_min_luma_coding_block_size_minus3
	bw.WriteUE(0) // MinCbLog2SizeY = 3, so min CB size = 8

	// log2_diff_max_min_luma_coding_block_size
	bw.WriteUE(3) // MaxCbLog2SizeY = 6, so max CB size = 64

	// log2_min_luma_transform_block_size_minus2
	bw.WriteUE(0) // MinTbLog2SizeY = 2, so min TB size = 4

	// log2_diff_max_min_luma_transform_block_size
	bw.WriteUE(3) // MaxTbLog2SizeY = 5, so max TB size = 32

	// max_transform_hierarchy_depth_inter
	bw.WriteUE(0)

	// max_transform_hierarchy_depth_intra
	bw.WriteUE(0)

	// scaling_list_enabled_flag
	bw.WriteBit(0)

	// amp_enabled_flag (asymmetric motion partitions)
	bw.WriteBit(0)

	// sample_adaptive_offset_enabled_flag
	bw.WriteBit(0)

	// pcm_enabled_flag - IMPORTANT: enables PCM mode for lossless
	bw.WriteBit(1)

	// PCM parameters (since pcm_enabled_flag = 1)
	// pcm_sample_bit_depth_luma_minus1 (4 bits)
	bw.WriteBits(uint32(config.BitDepth-1), 4)
	// pcm_sample_bit_depth_chroma_minus1 (4 bits)
	bw.WriteBits(uint32(config.BitDepth-1), 4)
	// log2_min_pcm_luma_coding_block_size_minus3
	bw.WriteUE(0) // min PCM CB size = 8
	// log2_diff_max_min_pcm_luma_coding_block_size
	bw.WriteUE(2) // max PCM CB size = 32
	// pcm_loop_filter_disabled_flag
	bw.WriteBit(1)

	// num_short_term_ref_pic_sets
	bw.WriteUE(0)

	// long_term_ref_pics_present_flag
	bw.WriteBit(0)

	// sps_temporal_mvp_enabled_flag
	bw.WriteBit(0)

	// strong_intra_smoothing_enabled_flag
	bw.WriteBit(0)

	// vui_parameters_present_flag
	bw.WriteBit(0)

	// sps_extension_present_flag
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// GenerateHEVCPPS generates a Picture Parameter Set for HEVC
func GenerateHEVCPPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(64)

	// pps_pic_parameter_set_id
	bw.WriteUE(0)

	// pps_seq_parameter_set_id
	bw.WriteUE(0)

	// dependent_slice_segments_enabled_flag
	bw.WriteBit(0)

	// output_flag_present_flag
	bw.WriteBit(0)

	// num_extra_slice_header_bits (3 bits)
	bw.WriteBits(0, 3)

	// sign_data_hiding_enabled_flag
	bw.WriteBit(0)

	// cabac_init_present_flag
	bw.WriteBit(0)

	// num_ref_idx_l0_default_active_minus1
	bw.WriteUE(0)

	// num_ref_idx_l1_default_active_minus1
	bw.WriteUE(0)

	// init_qp_minus26
	bw.WriteSE(0)

	// constrained_intra_pred_flag
	bw.WriteBit(0)

	// transform_skip_enabled_flag
	bw.WriteBit(0)

	// cu_qp_delta_enabled_flag
	bw.WriteBit(0)

	// pps_cb_qp_offset
	bw.WriteSE(0)

	// pps_cr_qp_offset
	bw.WriteSE(0)

	// pps_slice_chroma_qp_offsets_present_flag
	bw.WriteBit(0)

	// weighted_pred_flag
	bw.WriteBit(0)

	// weighted_bipred_flag
	bw.WriteBit(0)

	// transquant_bypass_enabled_flag
	bw.WriteBit(0)

	// tiles_enabled_flag
	bw.WriteBit(0)

	// entropy_coding_sync_enabled_flag
	bw.WriteBit(0)

	// pps_loop_filter_across_slices_enabled_flag
	bw.WriteBit(0)

	// deblocking_filter_control_present_flag
	bw.WriteBit(1)

	// deblocking_filter_override_enabled_flag
	bw.WriteBit(0)

	// pps_deblocking_filter_disabled_flag
	bw.WriteBit(1) // Disable deblocking

	// pps_scaling_list_data_present_flag
	bw.WriteBit(0)

	// lists_modification_present_flag
	bw.WriteBit(0)

	// log2_parallel_merge_level_minus2
	bw.WriteUE(0)

	// slice_segment_header_extension_present_flag
	bw.WriteBit(0)

	// pps_extension_present_flag
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// writeProfileTierLevel writes the profile_tier_level syntax element
func writeProfileTierLevel(bw *BitstreamWriter, config HEVCEncoderConfig, profilePresentFlag bool, maxNumSubLayersMinus1 uint32) {
	if profilePresentFlag {
		// general_profile_space (2 bits)
		bw.WriteBits(0, 2)

		// general_tier_flag
		bw.WriteBit(0)

		// general_profile_idc (5 bits)
		bw.WriteBits(config.Profile, 5)

		// general_profile_compatibility_flag[32]
		for i := 0; i < 32; i++ {
			if uint32(i) == config.Profile {
				bw.WriteBit(1)
			} else {
				bw.WriteBit(0)
			}
		}

		// general_progressive_source_flag
		bw.WriteBit(1)

		// general_interlaced_source_flag
		bw.WriteBit(0)

		// general_non_packed_constraint_flag
		bw.WriteBit(0)

		// general_frame_only_constraint_flag
		bw.WriteBit(1)

		// general_reserved_zero_43bits (for Main profile) or constraint flags for REXT
		if config.Profile == HEVC_PROFILE_REXT {
			// Range extension constraint flags
			bw.WriteBit(0) // general_max_12bit_constraint_flag
			bw.WriteBit(0) // general_max_10bit_constraint_flag
			bw.WriteBit(1) // general_max_8bit_constraint_flag
			bw.WriteBit(0) // general_max_422chroma_constraint_flag
			bw.WriteBit(0) // general_max_420chroma_constraint_flag
			bw.WriteBit(0) // general_max_monochrome_constraint_flag
			bw.WriteBit(0) // general_intra_constraint_flag
			bw.WriteBit(0) // general_one_picture_only_constraint_flag
			bw.WriteBit(0) // general_lower_bit_rate_constraint_flag
			// reserved_zero_34bits + general_inbld_flag = 35 bits
			bw.WriteBits(0, 32)
			bw.WriteBits(0, 3)
		} else {
			// 44 reserved zero bits for non-REXT profiles
			bw.WriteBits(0, 32)
			bw.WriteBits(0, 12)
		}
	}

	// general_level_idc (8 bits)
	bw.WriteBits(config.Level, 8)

	// Sub-layer flags (none for maxNumSubLayersMinus1 = 0)
	for i := uint32(0); i < maxNumSubLayersMinus1; i++ {
		bw.WriteBit(0) // sub_layer_profile_present_flag[i]
		bw.WriteBit(0) // sub_layer_level_present_flag[i]
	}

	// Alignment bits if maxNumSubLayersMinus1 > 0
	if maxNumSubLayersMinus1 > 0 {
		for i := maxNumSubLayersMinus1; i < 8; i++ {
			bw.WriteBits(0, 2) // reserved_zero_2bits
		}
	}
}

// MP4WriterHEVC writes HEVC data to MP4 container
type MP4WriterHEVC struct {
	config      HEVCEncoderConfig
	vps         []byte
	sps         []byte
	pps         []byte
	alphaSps    []byte // Alpha layer SPS (layer 1)
	alphaPps    []byte // Alpha layer PPS (layer 1)
	frames      [][]byte
	isKeyFrame  []bool
}

// NewMP4WriterHEVC creates a new HEVC MP4 writer
func NewMP4WriterHEVC(config HEVCEncoderConfig, vps, sps, pps []byte) *MP4WriterHEVC {
	return &MP4WriterHEVC{
		config:     config,
		vps:        vps,
		sps:        sps,
		pps:        pps,
		frames:     make([][]byte, 0),
		isKeyFrame: make([]bool, 0),
	}
}

// NewMP4WriterHEVCWithAlpha creates a new HEVC MP4 writer with alpha layer support
func NewMP4WriterHEVCWithAlpha(config HEVCEncoderConfig, vps, sps, pps, alphaSps, alphaPps []byte) *MP4WriterHEVC {
	return &MP4WriterHEVC{
		config:     config,
		vps:        vps,
		sps:        sps,
		pps:        pps,
		alphaSps:   alphaSps,
		alphaPps:   alphaPps,
		frames:     make([][]byte, 0),
		isKeyFrame: make([]bool, 0),
	}
}

// AddFrame adds an encoded frame to the MP4
func (w *MP4WriterHEVC) AddFrame(data []byte, isKey bool) {
	w.frames = append(w.frames, data)
	w.isKeyFrame = append(w.isKeyFrame, isKey)
}

// WriteToFile writes the complete MP4 to a file
func (w *MP4WriterHEVC) WriteToFile(path string) error {
	// Calculate total mdat size
	mdatSize := uint64(8) // mdat header
	for _, frame := range w.frames {
		mdatSize += uint64(len(frame))
	}

	// Build file
	var file []byte

	// ftyp box
	file = append(file, w.writeFtyp()...)

	// mdat box (raw frame data)
	file = append(file, w.writeMdat()...)

	// moov box (metadata)
	file = append(file, w.writeMoov(uint64(len(w.writeFtyp())))...)

	// Write to file
	return os.WriteFile(path, file, 0644)
}

func (w *MP4WriterHEVC) writeFtyp() []byte {
	// ftyp: isom + compatible brands
	ftyp := []byte{
		0x00, 0x00, 0x00, 0x18, // size = 24
		'f', 't', 'y', 'p',
		'i', 's', 'o', 'm', // major brand
		0x00, 0x00, 0x02, 0x00, // minor version
		'i', 's', 'o', 'm', // compatible brand
		'i', 's', 'o', '6', // compatible brand (for HEVC)
	}
	return ftyp
}

func (w *MP4WriterHEVC) writeMdat() []byte {
	// Calculate size
	size := uint32(8) // header
	for _, frame := range w.frames {
		size += uint32(len(frame))
	}

	mdat := make([]byte, 8)
	mdat[0] = byte(size >> 24)
	mdat[1] = byte(size >> 16)
	mdat[2] = byte(size >> 8)
	mdat[3] = byte(size)
	mdat[4] = 'm'
	mdat[5] = 'd'
	mdat[6] = 'a'
	mdat[7] = 't'

	for _, frame := range w.frames {
		mdat = append(mdat, frame...)
	}

	return mdat
}

func (w *MP4WriterHEVC) writeMoov(mdatOffset uint64) []byte {
	mvhd := w.writeMvhd()
	trak := w.writeTrak(mdatOffset)

	size := uint32(8 + len(mvhd) + len(trak))
	moov := make([]byte, 8)
	moov[0] = byte(size >> 24)
	moov[1] = byte(size >> 16)
	moov[2] = byte(size >> 8)
	moov[3] = byte(size)
	moov[4] = 'm'
	moov[5] = 'o'
	moov[6] = 'o'
	moov[7] = 'v'

	moov = append(moov, mvhd...)
	moov = append(moov, trak...)

	return moov
}

func (w *MP4WriterHEVC) writeMvhd() []byte {
	duration := uint32(len(w.frames))
	timescale := w.config.FrameRateNum / w.config.FrameRateDen

	mvhd := make([]byte, 108)
	// Size
	size := uint32(108)
	mvhd[0] = byte(size >> 24)
	mvhd[1] = byte(size >> 16)
	mvhd[2] = byte(size >> 8)
	mvhd[3] = byte(size)
	// Type
	mvhd[4] = 'm'
	mvhd[5] = 'v'
	mvhd[6] = 'h'
	mvhd[7] = 'd'
	// Version and flags
	mvhd[8] = 0
	// Creation/modification time (skip)
	// Timescale at offset 20
	mvhd[20] = byte(timescale >> 24)
	mvhd[21] = byte(timescale >> 16)
	mvhd[22] = byte(timescale >> 8)
	mvhd[23] = byte(timescale)
	// Duration at offset 24
	mvhd[24] = byte(duration >> 24)
	mvhd[25] = byte(duration >> 16)
	mvhd[26] = byte(duration >> 8)
	mvhd[27] = byte(duration)
	// Rate (1.0) at offset 28
	mvhd[28] = 0x00
	mvhd[29] = 0x01
	mvhd[30] = 0x00
	mvhd[31] = 0x00
	// Volume (1.0) at offset 32
	mvhd[32] = 0x01
	mvhd[33] = 0x00
	// Matrix at offset 44 (3x3 transformation matrix, identity)
	// a = 1.0 (16.16 fixed point) at offset 44
	mvhd[44] = 0x00
	mvhd[45] = 0x01
	mvhd[46] = 0x00
	mvhd[47] = 0x00
	// b = 0 at offset 48 (already 0)
	// u = 0 at offset 52 (already 0)
	// c = 0 at offset 56 (already 0)
	// d = 1.0 (16.16 fixed point) at offset 60
	mvhd[60] = 0x00
	mvhd[61] = 0x01
	mvhd[62] = 0x00
	mvhd[63] = 0x00
	// v = 0 at offset 64 (already 0)
	// tx = 0 at offset 68 (already 0)
	// ty = 0 at offset 72 (already 0)
	// w = 1.0 (2.30 fixed point = 0x40000000) at offset 76
	mvhd[76] = 0x40
	mvhd[77] = 0x00
	mvhd[78] = 0x00
	mvhd[79] = 0x00
	// Next track ID at offset 104
	mvhd[104] = 0x00
	mvhd[105] = 0x00
	mvhd[106] = 0x00
	mvhd[107] = 0x02

	return mvhd
}

func (w *MP4WriterHEVC) writeTrak(mdatOffset uint64) []byte {
	tkhd := w.writeTkhd()
	mdia := w.writeMdia(mdatOffset)

	size := uint32(8 + len(tkhd) + len(mdia))
	trak := make([]byte, 8)
	trak[0] = byte(size >> 24)
	trak[1] = byte(size >> 16)
	trak[2] = byte(size >> 8)
	trak[3] = byte(size)
	trak[4] = 't'
	trak[5] = 'r'
	trak[6] = 'a'
	trak[7] = 'k'

	trak = append(trak, tkhd...)
	trak = append(trak, mdia...)

	return trak
}

func (w *MP4WriterHEVC) writeTkhd() []byte {
	duration := uint32(len(w.frames))

	tkhd := make([]byte, 92)
	size := uint32(92)
	tkhd[0] = byte(size >> 24)
	tkhd[1] = byte(size >> 16)
	tkhd[2] = byte(size >> 8)
	tkhd[3] = byte(size)
	tkhd[4] = 't'
	tkhd[5] = 'k'
	tkhd[6] = 'h'
	tkhd[7] = 'd'
	// Version and flags (track enabled)
	tkhd[11] = 0x03
	// Track ID at offset 20
	tkhd[20] = 0x00
	tkhd[21] = 0x00
	tkhd[22] = 0x00
	tkhd[23] = 0x01
	// Duration at offset 28
	tkhd[28] = byte(duration >> 24)
	tkhd[29] = byte(duration >> 16)
	tkhd[30] = byte(duration >> 8)
	tkhd[31] = byte(duration)
	// Matrix at offset 48 (3x3 transformation matrix, identity)
	// a = 1.0 (16.16 fixed point) at offset 48
	tkhd[48] = 0x00
	tkhd[49] = 0x01
	tkhd[50] = 0x00
	tkhd[51] = 0x00
	// b = 0 at offset 52 (already 0)
	// u = 0 at offset 56 (already 0)
	// c = 0 at offset 60 (already 0)
	// d = 1.0 (16.16 fixed point) at offset 64
	tkhd[64] = 0x00
	tkhd[65] = 0x01
	tkhd[66] = 0x00
	tkhd[67] = 0x00
	// v = 0 at offset 68 (already 0)
	// tx = 0 at offset 72 (already 0)
	// ty = 0 at offset 76 (already 0)
	// w = 1.0 (2.30 fixed point = 0x40000000) at offset 80
	tkhd[80] = 0x40
	tkhd[81] = 0x00
	tkhd[82] = 0x00
	tkhd[83] = 0x00
	// Width at offset 84 (16.16 fixed point)
	tkhd[84] = byte(w.config.Width >> 8)
	tkhd[85] = byte(w.config.Width)
	// Height at offset 88
	tkhd[88] = byte(w.config.Height >> 8)
	tkhd[89] = byte(w.config.Height)

	return tkhd
}

func (w *MP4WriterHEVC) writeMdia(mdatOffset uint64) []byte {
	mdhd := w.writeMdhd()
	hdlr := w.writeHdlr()
	minf := w.writeMinf(mdatOffset)

	size := uint32(8 + len(mdhd) + len(hdlr) + len(minf))
	mdia := make([]byte, 8)
	mdia[0] = byte(size >> 24)
	mdia[1] = byte(size >> 16)
	mdia[2] = byte(size >> 8)
	mdia[3] = byte(size)
	mdia[4] = 'm'
	mdia[5] = 'd'
	mdia[6] = 'i'
	mdia[7] = 'a'

	mdia = append(mdia, mdhd...)
	mdia = append(mdia, hdlr...)
	mdia = append(mdia, minf...)

	return mdia
}

func (w *MP4WriterHEVC) writeMdhd() []byte {
	duration := uint32(len(w.frames))
	timescale := w.config.FrameRateNum / w.config.FrameRateDen

	mdhd := make([]byte, 32)
	size := uint32(32)
	mdhd[0] = byte(size >> 24)
	mdhd[1] = byte(size >> 16)
	mdhd[2] = byte(size >> 8)
	mdhd[3] = byte(size)
	mdhd[4] = 'm'
	mdhd[5] = 'd'
	mdhd[6] = 'h'
	mdhd[7] = 'd'
	// Timescale at offset 20
	mdhd[20] = byte(timescale >> 24)
	mdhd[21] = byte(timescale >> 16)
	mdhd[22] = byte(timescale >> 8)
	mdhd[23] = byte(timescale)
	// Duration at offset 24
	mdhd[24] = byte(duration >> 24)
	mdhd[25] = byte(duration >> 16)
	mdhd[26] = byte(duration >> 8)
	mdhd[27] = byte(duration)
	// Language at offset 28 (undetermined)
	mdhd[28] = 0x55
	mdhd[29] = 0xC4

	return mdhd
}

func (w *MP4WriterHEVC) writeHdlr() []byte {
	hdlr := []byte{
		0x00, 0x00, 0x00, 0x21, // size = 33
		'h', 'd', 'l', 'r',
		0x00, 0x00, 0x00, 0x00, // version and flags
		0x00, 0x00, 0x00, 0x00, // pre_defined
		'v', 'i', 'd', 'e', // handler_type
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, 0x00, 0x00, 0x00, // reserved
		0x00, // name (null-terminated)
	}
	return hdlr
}

func (w *MP4WriterHEVC) writeMinf(mdatOffset uint64) []byte {
	vmhd := w.writeVmhd()
	dinf := w.writeDinf()
	stbl := w.writeStbl(mdatOffset)

	size := uint32(8 + len(vmhd) + len(dinf) + len(stbl))
	minf := make([]byte, 8)
	minf[0] = byte(size >> 24)
	minf[1] = byte(size >> 16)
	minf[2] = byte(size >> 8)
	minf[3] = byte(size)
	minf[4] = 'm'
	minf[5] = 'i'
	minf[6] = 'n'
	minf[7] = 'f'

	minf = append(minf, vmhd...)
	minf = append(minf, dinf...)
	minf = append(minf, stbl...)

	return minf
}

func (w *MP4WriterHEVC) writeVmhd() []byte {
	return []byte{
		0x00, 0x00, 0x00, 0x14, // size = 20
		'v', 'm', 'h', 'd',
		0x00, 0x00, 0x00, 0x01, // version and flags
		0x00, 0x00, // graphics mode
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // opcolor
	}
}

func (w *MP4WriterHEVC) writeDinf() []byte {
	return []byte{
		0x00, 0x00, 0x00, 0x24, // size = 36
		'd', 'i', 'n', 'f',
		0x00, 0x00, 0x00, 0x1C, // dref size = 28
		'd', 'r', 'e', 'f',
		0x00, 0x00, 0x00, 0x00, // version and flags
		0x00, 0x00, 0x00, 0x01, // entry count
		0x00, 0x00, 0x00, 0x0C, // url size = 12
		'u', 'r', 'l', ' ',
		0x00, 0x00, 0x00, 0x01, // flags (self-contained)
	}
}

func (w *MP4WriterHEVC) writeStbl(mdatOffset uint64) []byte {
	stsd := w.writeStsd()
	stts := w.writeStts()
	stss := w.writeStss()
	stsc := w.writeStsc()
	stsz := w.writeStsz()
	stco := w.writeStco(mdatOffset)

	size := uint32(8 + len(stsd) + len(stts) + len(stss) + len(stsc) + len(stsz) + len(stco))
	stbl := make([]byte, 8)
	stbl[0] = byte(size >> 24)
	stbl[1] = byte(size >> 16)
	stbl[2] = byte(size >> 8)
	stbl[3] = byte(size)
	stbl[4] = 's'
	stbl[5] = 't'
	stbl[6] = 'b'
	stbl[7] = 'l'

	stbl = append(stbl, stsd...)
	stbl = append(stbl, stts...)
	stbl = append(stbl, stss...)
	stbl = append(stbl, stsc...)
	stbl = append(stbl, stsz...)
	stbl = append(stbl, stco...)

	return stbl
}

func (w *MP4WriterHEVC) writeStsd() []byte {
	hvc1 := w.writeHvc1()

	size := uint32(16 + len(hvc1))
	stsd := make([]byte, 16)
	stsd[0] = byte(size >> 24)
	stsd[1] = byte(size >> 16)
	stsd[2] = byte(size >> 8)
	stsd[3] = byte(size)
	stsd[4] = 's'
	stsd[5] = 't'
	stsd[6] = 's'
	stsd[7] = 'd'
	// version and flags
	// entry count = 1
	stsd[15] = 0x01

	stsd = append(stsd, hvc1...)
	return stsd
}

func (w *MP4WriterHEVC) writeHvc1() []byte {
	hvcC := w.writeHvcC()

	// Build auxC box if alpha is present
	var auxC []byte
	if w.config.HasAlpha {
		// auxC box: signals that auxiliary layer is alpha
		// urn:mpeg:mpegB:cicp:systems:auxiliary:alpha
		auxTypeUrn := "urn:mpeg:mpegB:cicp:systems:auxiliary:alpha"
		auxC = make([]byte, 8+1+len(auxTypeUrn)+1)
		auxCSize := uint32(len(auxC))
		auxC[0] = byte(auxCSize >> 24)
		auxC[1] = byte(auxCSize >> 16)
		auxC[2] = byte(auxCSize >> 8)
		auxC[3] = byte(auxCSize)
		auxC[4] = 'a'
		auxC[5] = 'u'
		auxC[6] = 'x'
		auxC[7] = 'C'
		auxC[8] = 0 // version
		copy(auxC[9:], auxTypeUrn)
		auxC[9+len(auxTypeUrn)] = 0 // null terminator
	}

	// hvc1 sample entry: 86 bytes base + hvcC + optional auxC
	size := uint32(86 + len(hvcC) + len(auxC))
	hvc1 := make([]byte, 86)

	// Size
	hvc1[0] = byte(size >> 24)
	hvc1[1] = byte(size >> 16)
	hvc1[2] = byte(size >> 8)
	hvc1[3] = byte(size)

	// Type
	hvc1[4] = 'h'
	hvc1[5] = 'v'
	hvc1[6] = 'c'
	hvc1[7] = '1'

	// Reserved (6 bytes) + data_reference_index (2 bytes)
	hvc1[14] = 0x00
	hvc1[15] = 0x01 // data_reference_index = 1

	// Width at offset 32
	hvc1[32] = byte(w.config.Width >> 8)
	hvc1[33] = byte(w.config.Width)

	// Height at offset 34
	hvc1[34] = byte(w.config.Height >> 8)
	hvc1[35] = byte(w.config.Height)

	// Horizontal resolution at offset 36 (72 dpi = 0x00480000)
	hvc1[36] = 0x00
	hvc1[37] = 0x48
	hvc1[38] = 0x00
	hvc1[39] = 0x00

	// Vertical resolution at offset 40
	hvc1[40] = 0x00
	hvc1[41] = 0x48
	hvc1[42] = 0x00
	hvc1[43] = 0x00

	// Frame count at offset 48
	hvc1[48] = 0x00
	hvc1[49] = 0x01

	// Compressor name at offset 50 (32 bytes, empty)

	// Depth at offset 82 (32-bit for RGBA with alpha)
	if w.config.HasAlpha {
		hvc1[82] = 0x00
		hvc1[83] = 0x20 // 32-bit
	} else {
		hvc1[82] = 0x00
		hvc1[83] = 0x18 // 24-bit
	}

	// Pre-defined at offset 84
	hvc1[84] = 0xFF
	hvc1[85] = 0xFF

	hvc1 = append(hvc1, hvcC...)
	if len(auxC) > 0 {
		hvc1 = append(hvc1, auxC...)
	}
	return hvc1
}

func (w *MP4WriterHEVC) writeHvcC() []byte {
	// HEVCDecoderConfigurationRecord with optional alpha layer parameter sets

	// Calculate size - each NAL array has 5-byte header + 2-byte NAL header + RBSP
	headerSize := 23
	vpsArraySize := 5 + 2 + len(w.vps)  // array header + NAL header + RBSP
	spsArraySize := 5 + 2 + len(w.sps)
	ppsArraySize := 5 + 2 + len(w.pps)

	numArrays := 3
	totalSize := 8 + headerSize + vpsArraySize + spsArraySize + ppsArraySize

	// Add alpha layer arrays if present
	if len(w.alphaSps) > 0 {
		numArrays += 2
		totalSize += 5 + 2 + len(w.alphaSps) // Alpha SPS
		totalSize += 5 + 2 + len(w.alphaPps) // Alpha PPS
	}

	hvcC := make([]byte, 8+headerSize)

	// Box size
	hvcC[0] = byte(totalSize >> 24)
	hvcC[1] = byte(totalSize >> 16)
	hvcC[2] = byte(totalSize >> 8)
	hvcC[3] = byte(totalSize)

	// Box type
	hvcC[4] = 'h'
	hvcC[5] = 'v'
	hvcC[6] = 'c'
	hvcC[7] = 'C'

	// configurationVersion
	hvcC[8] = 1

	// general_profile_space(2) + general_tier_flag(1) + general_profile_idc(5)
	hvcC[9] = byte(w.config.Profile & 0x1F)

	// general_profile_compatibility_flags - set bit for the profile
	// Profile bit position: bit N from MSB corresponds to profile N
	profileCompat := uint32(1) << (31 - w.config.Profile)
	hvcC[10] = byte(profileCompat >> 24)
	hvcC[11] = byte(profileCompat >> 16)
	hvcC[12] = byte(profileCompat >> 8)
	hvcC[13] = byte(profileCompat)

	// general_constraint_indicator_flags (6 bytes = 48 bits)
	// Bits 0-3: progressive, interlaced, non_packed, frame_only
	// Bits 4-47: profile-specific constraint flags
	if w.config.Profile == HEVC_PROFILE_REXT {
		// REXT byte 0: progressive(1) + interlaced(0) + non_packed(0) + frame_only(1) +
		//              max_12bit(0) + max_10bit(0) + max_8bit(1) + max_422chroma(0)
		hvcC[14] = 0x92 // 1001 0010
		// REXT byte 1: max_420chroma(0) + max_mono(0) + intra(0) + one_pic_only(0) + lower_bitrate(0) + reserved
		hvcC[15] = 0x00
		hvcC[16] = 0x00
		hvcC[17] = 0x00
		hvcC[18] = 0x00
		hvcC[19] = 0x00
	} else {
		// Main profile: progressive=1, interlaced=0, non_packed=0, frame_only=1, reserved
		hvcC[14] = 0x90
		hvcC[15] = 0x00
		hvcC[16] = 0x00
		hvcC[17] = 0x00
		hvcC[18] = 0x00
		hvcC[19] = 0x00
	}

	// general_level_idc
	hvcC[20] = byte(w.config.Level)

	// min_spatial_segmentation_idc
	hvcC[21] = 0xF0
	hvcC[22] = 0x00

	// parallelismType
	hvcC[23] = 0xFC

	// chromaFormat
	hvcC[24] = 0xFC | byte(w.config.ChromaFormat&0x03)

	// bitDepthLumaMinus8
	hvcC[25] = 0xF8 | byte((w.config.BitDepth-8)&0x07)

	// bitDepthChromaMinus8
	hvcC[26] = 0xF8 | byte((w.config.BitDepth-8)&0x07)

	// avgFrameRate
	hvcC[27] = 0x00
	hvcC[28] = 0x00

	// constantFrameRate(2) + numTemporalLayers(3) + temporalIdNested(1) + lengthSizeMinusOne(2)
	// constantFrameRate=0, numTemporalLayers=1, temporalIdNested=1, lengthSizeMinusOne=3
	hvcC[29] = (0 << 6) | (1 << 3) | (1 << 2) | 3 // = 0x0F

	// numOfArrays
	hvcC[30] = byte(numArrays)

	// VPS array - NAL units in hvcC include 2-byte NAL header
	vpsNalSize := 2 + len(w.vps)
	vpsArray := make([]byte, 5+vpsNalSize)
	vpsArray[0] = 0xA0 | HEVC_NAL_VPS // array_completeness=1, reserved=0, NAL_unit_type
	vpsArray[1] = 0x00
	vpsArray[2] = 0x01 // numNalus = 1
	vpsArray[3] = byte(vpsNalSize >> 8)
	vpsArray[4] = byte(vpsNalSize)
	// NAL header: forbidden(0) + type(6) + layerId(6) + tid(3)
	vpsArray[5] = (HEVC_NAL_VPS << 1) // type in bits [6:1]
	vpsArray[6] = 0x01                // layerId=0, tid=1
	copy(vpsArray[7:], w.vps)
	hvcC = append(hvcC, vpsArray...)

	// SPS array (main layer)
	spsNalSize := 2 + len(w.sps)
	spsArray := make([]byte, 5+spsNalSize)
	spsArray[0] = 0xA0 | HEVC_NAL_SPS
	spsArray[1] = 0x00
	spsArray[2] = 0x01
	spsArray[3] = byte(spsNalSize >> 8)
	spsArray[4] = byte(spsNalSize)
	spsArray[5] = (HEVC_NAL_SPS << 1)
	spsArray[6] = 0x01
	copy(spsArray[7:], w.sps)
	hvcC = append(hvcC, spsArray...)

	// PPS array (main layer)
	ppsNalSize := 2 + len(w.pps)
	ppsArray := make([]byte, 5+ppsNalSize)
	ppsArray[0] = 0xA0 | HEVC_NAL_PPS
	ppsArray[1] = 0x00
	ppsArray[2] = 0x01
	ppsArray[3] = byte(ppsNalSize >> 8)
	ppsArray[4] = byte(ppsNalSize)
	ppsArray[5] = (HEVC_NAL_PPS << 1)
	ppsArray[6] = 0x01
	copy(ppsArray[7:], w.pps)
	hvcC = append(hvcC, ppsArray...)

	// Alpha layer SPS/PPS if present
	if len(w.alphaSps) > 0 {
		// Alpha SPS array
		alphaSpsNalSize := 2 + len(w.alphaSps)
		alphaSpsArray := make([]byte, 5+alphaSpsNalSize)
		alphaSpsArray[0] = 0xA0 | HEVC_NAL_SPS
		alphaSpsArray[1] = 0x00
		alphaSpsArray[2] = 0x01
		alphaSpsArray[3] = byte(alphaSpsNalSize >> 8)
		alphaSpsArray[4] = byte(alphaSpsNalSize)
		alphaSpsArray[5] = (HEVC_NAL_SPS << 1)
		alphaSpsArray[6] = 0x09 // layerId=1, tid=1
		copy(alphaSpsArray[7:], w.alphaSps)
		hvcC = append(hvcC, alphaSpsArray...)

		// Alpha PPS array
		alphaPpsNalSize := 2 + len(w.alphaPps)
		alphaPpsArray := make([]byte, 5+alphaPpsNalSize)
		alphaPpsArray[0] = 0xA0 | HEVC_NAL_PPS
		alphaPpsArray[1] = 0x00
		alphaPpsArray[2] = 0x01
		alphaPpsArray[3] = byte(alphaPpsNalSize >> 8)
		alphaPpsArray[4] = byte(alphaPpsNalSize)
		alphaPpsArray[5] = (HEVC_NAL_PPS << 1)
		alphaPpsArray[6] = 0x09 // layerId=1, tid=1
		copy(alphaPpsArray[7:], w.alphaPps)
		hvcC = append(hvcC, alphaPpsArray...)
	}

	return hvcC
}

func (w *MP4WriterHEVC) writeStts() []byte {
	numFrames := uint32(len(w.frames))

	stts := make([]byte, 24)
	// Size
	stts[0] = 0x00
	stts[1] = 0x00
	stts[2] = 0x00
	stts[3] = 0x18 // 24
	// Type
	stts[4] = 's'
	stts[5] = 't'
	stts[6] = 't'
	stts[7] = 's'
	// Entry count
	stts[15] = 0x01
	// Sample count
	stts[16] = byte(numFrames >> 24)
	stts[17] = byte(numFrames >> 16)
	stts[18] = byte(numFrames >> 8)
	stts[19] = byte(numFrames)
	// Sample delta (1 frame per time unit)
	stts[23] = 0x01

	return stts
}

func (w *MP4WriterHEVC) writeStss() []byte {
	// Count keyframes
	keyframes := make([]uint32, 0)
	for i, isKey := range w.isKeyFrame {
		if isKey {
			keyframes = append(keyframes, uint32(i+1)) // 1-indexed
		}
	}

	size := uint32(16 + 4*len(keyframes))
	stss := make([]byte, 16)
	stss[0] = byte(size >> 24)
	stss[1] = byte(size >> 16)
	stss[2] = byte(size >> 8)
	stss[3] = byte(size)
	stss[4] = 's'
	stss[5] = 't'
	stss[6] = 's'
	stss[7] = 's'
	// Entry count
	count := uint32(len(keyframes))
	stss[12] = byte(count >> 24)
	stss[13] = byte(count >> 16)
	stss[14] = byte(count >> 8)
	stss[15] = byte(count)

	for _, kf := range keyframes {
		entry := make([]byte, 4)
		entry[0] = byte(kf >> 24)
		entry[1] = byte(kf >> 16)
		entry[2] = byte(kf >> 8)
		entry[3] = byte(kf)
		stss = append(stss, entry...)
	}

	return stss
}

func (w *MP4WriterHEVC) writeStsc() []byte {
	return []byte{
		0x00, 0x00, 0x00, 0x1C, // size = 28
		's', 't', 's', 'c',
		0x00, 0x00, 0x00, 0x00, // version and flags
		0x00, 0x00, 0x00, 0x01, // entry count
		0x00, 0x00, 0x00, 0x01, // first chunk
		0x00, 0x00, 0x00, 0x01, // samples per chunk
		0x00, 0x00, 0x00, 0x01, // sample description index
	}
}

func (w *MP4WriterHEVC) writeStsz() []byte {
	numFrames := uint32(len(w.frames))
	size := uint32(20 + 4*numFrames)

	stsz := make([]byte, 20)
	stsz[0] = byte(size >> 24)
	stsz[1] = byte(size >> 16)
	stsz[2] = byte(size >> 8)
	stsz[3] = byte(size)
	stsz[4] = 's'
	stsz[5] = 't'
	stsz[6] = 's'
	stsz[7] = 'z'
	// Sample size (0 = variable)
	// Sample count
	stsz[16] = byte(numFrames >> 24)
	stsz[17] = byte(numFrames >> 16)
	stsz[18] = byte(numFrames >> 8)
	stsz[19] = byte(numFrames)

	for _, frame := range w.frames {
		frameSize := uint32(len(frame))
		entry := make([]byte, 4)
		entry[0] = byte(frameSize >> 24)
		entry[1] = byte(frameSize >> 16)
		entry[2] = byte(frameSize >> 8)
		entry[3] = byte(frameSize)
		stsz = append(stsz, entry...)
	}

	return stsz
}

func (w *MP4WriterHEVC) writeStco(mdatOffset uint64) []byte {
	numFrames := uint32(len(w.frames))
	size := uint32(16 + 4*numFrames)

	stco := make([]byte, 16)
	stco[0] = byte(size >> 24)
	stco[1] = byte(size >> 16)
	stco[2] = byte(size >> 8)
	stco[3] = byte(size)
	stco[4] = 's'
	stco[5] = 't'
	stco[6] = 'c'
	stco[7] = 'o'
	// Entry count
	stco[12] = byte(numFrames >> 24)
	stco[13] = byte(numFrames >> 16)
	stco[14] = byte(numFrames >> 8)
	stco[15] = byte(numFrames)

	// Each frame is a separate chunk
	offset := mdatOffset + 8 // Skip mdat header
	for _, frame := range w.frames {
		entry := make([]byte, 4)
		entry[0] = byte(offset >> 24)
		entry[1] = byte(offset >> 16)
		entry[2] = byte(offset >> 8)
		entry[3] = byte(offset)
		stco = append(stco, entry...)
		offset += uint64(len(frame))
	}

	return stco
}
