package avcodec

import (
	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// HEVCEncoder encodes video frames to H.265/HEVC format.
type HEVCEncoder struct {
	// Encoder parameters
	width          int
	height         int
	frameRate      avutil.Rational
	bitDepth       int
	chromaFormat   int // 0=mono, 1=4:2:0, 2=4:2:2, 3=4:4:4
	profile        int
	level          int
	qp             int // Quantization parameter (0-51)

	// Picture parameters
	ctbSize        int // CTB size: 16, 32, or 64
	minCbSize      int // Minimum CB size: 8
	maxTransformSize int
	ctbWidthInUnits  int
	ctbHeightInUnits int

	// Frame tracking
	frameNum       int64
	pocCounter     int

	// CABAC encoder
	cabac          *CABACEncoder

	// Context models for HEVC
	ctxSplitCuFlag     []CABACContext
	ctxSkipFlag        []CABACContext
	ctxPredMode        []CABACContext
	ctxPartMode        []CABACContext
	ctxPrevIntraLuma   []CABACContext
	ctxIntraChroma     []CABACContext
	ctxRqtRootCbf      []CABACContext
	ctxCbf             []CABACContext

	// Bitstream writer for NAL headers
	bs *avutil.BitstreamWriter
}

// NewHEVCEncoder creates a new HEVC encoder.
func NewHEVCEncoder() *HEVCEncoder {
	return &HEVCEncoder{
		bitDepth:     8,
		chromaFormat: 1, // 4:2:0
		profile:      HEVCProfileMain,
		level:        HEVCLevel51,
		qp:           26,
		ctbSize:      64,
		minCbSize:    8,
		maxTransformSize: 32,
		cabac:        NewCABACEncoder(),
		bs:           avutil.NewBitstreamWriter(),
	}
}

// Init implements Encoder.Init
func (e *HEVCEncoder) Init(ctx *EncoderContext) error {
	e.width = ctx.Width
	e.height = ctx.Height
	e.frameRate = ctx.Framerate

	if ctx.QMin > 0 {
		e.qp = ctx.QMin
	}

	// Calculate CTB dimensions
	e.ctbWidthInUnits = (e.width + e.ctbSize - 1) / e.ctbSize
	e.ctbHeightInUnits = (e.height + e.ctbSize - 1) / e.ctbSize

	// Initialize context models
	e.initContexts()

	// Generate parameter sets as extradata
	ctx.ExtraData = e.generateParameterSets()

	return nil
}

// initContexts initializes CABAC context models for HEVC.
func (e *HEVCEncoder) initContexts() {
	sliceQP := e.qp

	// split_cu_flag - 3 contexts
	e.ctxSplitCuFlag = make([]CABACContext, 3)
	splitInitValues := []int{139, 141, 157}
	for i := 0; i < 3; i++ {
		e.ctxSplitCuFlag[i] = InitContext(splitInitValues[i], sliceQP)
	}

	// cu_skip_flag - 3 contexts (for P/B slices)
	e.ctxSkipFlag = make([]CABACContext, 3)
	skipInitValues := []int{197, 185, 201}
	for i := 0; i < 3; i++ {
		e.ctxSkipFlag[i] = InitContext(skipInitValues[i], sliceQP)
	}

	// pred_mode_flag - 1 context
	e.ctxPredMode = make([]CABACContext, 1)
	e.ctxPredMode[0] = InitContext(149, sliceQP)

	// part_mode - 4 contexts
	e.ctxPartMode = make([]CABACContext, 4)
	partModeInitValues := []int{154, 139, 154, 154}
	for i := 0; i < 4; i++ {
		e.ctxPartMode[i] = InitContext(partModeInitValues[i], sliceQP)
	}

	// prev_intra_luma_pred_flag - 1 context
	e.ctxPrevIntraLuma = make([]CABACContext, 1)
	e.ctxPrevIntraLuma[0] = InitContext(184, sliceQP)

	// intra_chroma_pred_mode - 1 context
	e.ctxIntraChroma = make([]CABACContext, 1)
	e.ctxIntraChroma[0] = InitContext(137, sliceQP)

	// rqt_root_cbf - 1 context
	e.ctxRqtRootCbf = make([]CABACContext, 1)
	e.ctxRqtRootCbf[0] = InitContext(79, sliceQP)

	// cbf_luma and cbf_cb/cr - 4 contexts each
	e.ctxCbf = make([]CABACContext, 8)
	cbfInitValues := []int{111, 141, 94, 138, 111, 141, 94, 138}
	for i := 0; i < 8; i++ {
		e.ctxCbf[i] = InitContext(cbfInitValues[i], sliceQP)
	}
}

// generateParameterSets generates VPS, SPS, and PPS as NAL units.
func (e *HEVCEncoder) generateParameterSets() []byte {
	var result []byte

	// VPS
	vps := e.generateVPS()
	result = append(result, vps...)

	// SPS
	sps := e.generateSPS()
	result = append(result, sps...)

	// PPS
	pps := e.generatePPS()
	result = append(result, pps...)

	return result
}

// generateVPS generates a Video Parameter Set NAL unit.
func (e *HEVCEncoder) generateVPS() []byte {
	e.bs.Reset()

	// vps_video_parameter_set_id = 0
	e.bs.WriteBits(0, 4)

	// vps_base_layer_internal_flag = 1
	e.bs.WriteBit(1)
	// vps_base_layer_available_flag = 1
	e.bs.WriteBit(1)

	// vps_max_layers_minus1 = 0
	e.bs.WriteBits(0, 6)

	// vps_max_sub_layers_minus1 = 0
	e.bs.WriteBits(0, 3)

	// vps_temporal_id_nesting_flag = 1
	e.bs.WriteBit(1)

	// vps_reserved_0xffff_16bits
	e.bs.WriteBits(0xFFFF, 16)

	// Profile tier level
	e.writeProfileTierLevel(true, 0)

	// vps_sub_layer_ordering_info_present_flag = 0
	e.bs.WriteBit(0)

	// For sub_layer = 0:
	// vps_max_dec_pic_buffering_minus1[0]
	e.bs.WriteUE(4)
	// vps_max_num_reorder_pics[0]
	e.bs.WriteUE(2)
	// vps_max_latency_increase_plus1[0]
	e.bs.WriteUE(0)

	// vps_max_layer_id = 0
	e.bs.WriteBits(0, 6)

	// vps_num_layer_sets_minus1 = 0
	e.bs.WriteUE(0)

	// vps_timing_info_present_flag = 1
	e.bs.WriteBit(1)

	// vps_num_units_in_tick
	e.bs.WriteU32(uint32(e.frameRate.Den))
	// vps_time_scale
	e.bs.WriteU32(uint32(e.frameRate.Num))

	// vps_poc_proportional_to_timing_flag = 0
	e.bs.WriteBit(0)

	// vps_num_hrd_parameters = 0
	e.bs.WriteUE(0)

	// vps_extension_flag = 0
	e.bs.WriteBit(0)

	e.bs.FlushWithRBSP()

	return e.wrapNAL(HEVCNalVPS, e.bs.Bytes())
}

// generateSPS generates a Sequence Parameter Set NAL unit.
func (e *HEVCEncoder) generateSPS() []byte {
	e.bs.Reset()

	// sps_video_parameter_set_id = 0
	e.bs.WriteBits(0, 4)

	// sps_max_sub_layers_minus1 = 0
	e.bs.WriteBits(0, 3)

	// sps_temporal_id_nesting_flag = 1
	e.bs.WriteBit(1)

	// Profile tier level
	e.writeProfileTierLevel(true, 0)

	// sps_seq_parameter_set_id = 0
	e.bs.WriteUE(0)

	// chroma_format_idc = 1 (4:2:0)
	e.bs.WriteUE(uint32(e.chromaFormat))

	// pic_width_in_luma_samples
	e.bs.WriteUE(uint32(e.width))
	// pic_height_in_luma_samples
	e.bs.WriteUE(uint32(e.height))

	// conformance_window_flag = 0
	e.bs.WriteBit(0)

	// bit_depth_luma_minus8 = 0
	e.bs.WriteUE(uint32(e.bitDepth - 8))
	// bit_depth_chroma_minus8 = 0
	e.bs.WriteUE(uint32(e.bitDepth - 8))

	// log2_max_pic_order_cnt_lsb_minus4 = 4 (max POC = 256)
	e.bs.WriteUE(4)

	// sps_sub_layer_ordering_info_present_flag = 0
	e.bs.WriteBit(0)

	// For sub_layer = 0:
	// sps_max_dec_pic_buffering_minus1
	e.bs.WriteUE(4)
	// sps_max_num_reorder_pics
	e.bs.WriteUE(2)
	// sps_max_latency_increase_plus1
	e.bs.WriteUE(0)

	// log2_min_luma_coding_block_size_minus3 = 0 (min CB = 8)
	e.bs.WriteUE(0)
	// log2_diff_max_min_luma_coding_block_size = 3 (max CB = 64)
	e.bs.WriteUE(3)
	// log2_min_luma_transform_block_size_minus2 = 0 (min TB = 4)
	e.bs.WriteUE(0)
	// log2_diff_max_min_luma_transform_block_size = 3 (max TB = 32)
	e.bs.WriteUE(3)

	// max_transform_hierarchy_depth_inter = 2
	e.bs.WriteUE(2)
	// max_transform_hierarchy_depth_intra = 2
	e.bs.WriteUE(2)

	// scaling_list_enabled_flag = 0
	e.bs.WriteBit(0)

	// amp_enabled_flag = 0 (asymmetric motion partitions)
	e.bs.WriteBit(0)

	// sample_adaptive_offset_enabled_flag = 0
	e.bs.WriteBit(0)

	// pcm_enabled_flag = 0
	e.bs.WriteBit(0)

	// num_short_term_ref_pic_sets = 0
	e.bs.WriteUE(0)

	// long_term_ref_pics_present_flag = 0
	e.bs.WriteBit(0)

	// sps_temporal_mvp_enabled_flag = 0
	e.bs.WriteBit(0)

	// strong_intra_smoothing_enabled_flag = 1
	e.bs.WriteBit(1)

	// vui_parameters_present_flag = 0
	e.bs.WriteBit(0)

	// sps_extension_present_flag = 0
	e.bs.WriteBit(0)

	e.bs.FlushWithRBSP()

	return e.wrapNAL(HEVCNalSPS, e.bs.Bytes())
}

// generatePPS generates a Picture Parameter Set NAL unit.
func (e *HEVCEncoder) generatePPS() []byte {
	e.bs.Reset()

	// pps_pic_parameter_set_id = 0
	e.bs.WriteUE(0)

	// pps_seq_parameter_set_id = 0
	e.bs.WriteUE(0)

	// dependent_slice_segments_enabled_flag = 0
	e.bs.WriteBit(0)

	// output_flag_present_flag = 0
	e.bs.WriteBit(0)

	// num_extra_slice_header_bits = 0
	e.bs.WriteBits(0, 3)

	// sign_data_hiding_enabled_flag = 0
	e.bs.WriteBit(0)

	// cabac_init_present_flag = 0
	e.bs.WriteBit(0)

	// num_ref_idx_l0_default_active_minus1 = 0
	e.bs.WriteUE(0)
	// num_ref_idx_l1_default_active_minus1 = 0
	e.bs.WriteUE(0)

	// init_qp_minus26 = qp - 26
	e.bs.WriteSE(int32(e.qp - 26))

	// constrained_intra_pred_flag = 0
	e.bs.WriteBit(0)

	// transform_skip_enabled_flag = 0
	e.bs.WriteBit(0)

	// cu_qp_delta_enabled_flag = 0
	e.bs.WriteBit(0)

	// pps_cb_qp_offset = 0
	e.bs.WriteSE(0)
	// pps_cr_qp_offset = 0
	e.bs.WriteSE(0)

	// pps_slice_chroma_qp_offsets_present_flag = 0
	e.bs.WriteBit(0)

	// weighted_pred_flag = 0
	e.bs.WriteBit(0)

	// weighted_bipred_flag = 0
	e.bs.WriteBit(0)

	// transquant_bypass_enabled_flag = 0
	e.bs.WriteBit(0)

	// tiles_enabled_flag = 0
	e.bs.WriteBit(0)

	// entropy_coding_sync_enabled_flag = 0
	e.bs.WriteBit(0)

	// pps_loop_filter_across_slices_enabled_flag = 0
	// (set to 0 to avoid needing slice_loop_filter_across_slices_enabled_flag in slice header)
	e.bs.WriteBit(0)

	// deblocking_filter_control_present_flag = 0
	e.bs.WriteBit(0)

	// pps_scaling_list_data_present_flag = 0
	e.bs.WriteBit(0)

	// lists_modification_present_flag = 0
	e.bs.WriteBit(0)

	// log2_parallel_merge_level_minus2 = 0
	e.bs.WriteUE(0)

	// slice_segment_header_extension_present_flag = 0
	e.bs.WriteBit(0)

	// pps_extension_present_flag = 0
	e.bs.WriteBit(0)

	e.bs.FlushWithRBSP()

	return e.wrapNAL(HEVCNalPPS, e.bs.Bytes())
}

// writeProfileTierLevel writes the profile_tier_level syntax.
func (e *HEVCEncoder) writeProfileTierLevel(profilePresentFlag bool, maxSubLayersMinus1 int) {
	if profilePresentFlag {
		// general_profile_space = 0
		e.bs.WriteBits(0, 2)
		// general_tier_flag = 0
		e.bs.WriteBit(0)
		// general_profile_idc
		e.bs.WriteBits(uint32(e.profile), 5)

		// general_profile_compatibility_flag[j] for j = 0..31
		// Set flag for Main profile (index 1)
		profileCompat := uint32(1 << (31 - e.profile))
		e.bs.WriteU32(profileCompat)

		// general_progressive_source_flag = 1
		e.bs.WriteBit(1)
		// general_interlaced_source_flag = 0
		e.bs.WriteBit(0)
		// general_non_packed_constraint_flag = 0
		e.bs.WriteBit(0)
		// general_frame_only_constraint_flag = 1
		e.bs.WriteBit(1)

		// Constraint flags (44 bits of zeros)
		e.bs.WriteBits(0, 32)
		e.bs.WriteBits(0, 12)
	}

	// general_level_idc
	e.bs.WriteU8(uint8(e.level))

	// Sub-layer flags (none for maxSubLayersMinus1 = 0)
}

// wrapNAL wraps payload in a NAL unit with start code.
func (e *HEVCEncoder) wrapNAL(nalType int, payload []byte) []byte {
	// HEVC NAL header format:
	// forbidden_zero_bit (1) + nal_unit_type (6) + nuh_layer_id (6) + nuh_temporal_id_plus1 (3)

	result := make([]byte, 0, len(payload)+6)

	// Start code
	result = append(result, 0x00, 0x00, 0x00, 0x01)

	// NAL header (2 bytes)
	// forbidden_zero_bit = 0, nal_unit_type = nalType, nuh_layer_id = 0, nuh_temporal_id_plus1 = 1
	nalHeader := uint16(nalType << 9) | 1
	result = append(result, byte(nalHeader>>8), byte(nalHeader&0xFF))

	// Add RBSP with emulation prevention
	result = append(result, addEmulationPrevention(payload)...)

	return result
}

// addEmulationPrevention adds emulation prevention bytes (0x03) as needed.
func addEmulationPrevention(data []byte) []byte {
	result := make([]byte, 0, len(data)+len(data)/256)

	zeroCount := 0
	for _, b := range data {
		if zeroCount >= 2 && b <= 3 {
			result = append(result, 0x03)
			zeroCount = 0
		}

		result = append(result, b)

		if b == 0 {
			zeroCount++
		} else {
			zeroCount = 0
		}
	}

	return result
}

// Encode implements Encoder.Encode
func (e *HEVCEncoder) Encode(ctx *EncoderContext, frame *avutil.Frame) ([]*avutil.Packet, error) {
	if frame == nil {
		return nil, nil
	}

	// Reinitialize contexts for each frame (for I-frames)
	e.initContexts()

	// Generate slice NAL
	sliceData := e.encodeSlice(frame)

	// All frames are IDR for simplicity (intra-only encoding)
	nalType := HEVCNalIdrNLP

	nalUnit := e.wrapNAL(nalType, sliceData)

	// Create packet
	pkt := avutil.NewPacket()
	pkt.Data = nalUnit
	pkt.Size = len(nalUnit)
	pkt.Pts = e.frameNum
	pkt.Dts = e.frameNum
	pkt.Duration = 1
	pkt.SetKeyframe(true) // All frames are keyframes

	e.frameNum++
	e.pocCounter++

	return []*avutil.Packet{pkt}, nil
}

// encodeSlice encodes a slice (single slice for entire frame).
func (e *HEVCEncoder) encodeSlice(frame *avutil.Frame) []byte {
	e.bs.Reset()

	// Slice header
	// Make ALL frames IDR for simplicity (no inter-prediction needed)
	isIDR := true

	// first_slice_segment_in_pic_flag = 1
	e.bs.WriteBit(1)

	if isIDR {
		// no_output_of_prior_pics_flag = 0
		e.bs.WriteBit(0)
	}

	// slice_pic_parameter_set_id = 0
	e.bs.WriteUE(0)

	// For non-first slice in pic: slice_segment_address (not needed here)

	// slice_type = I (2)
	e.bs.WriteUE(uint32(HEVCSliceI))

	// pic_output_flag = 1 (implicit since output_flag_present_flag = 0)

	// For non-IDR: pic_order_cnt_lsb would be needed
	// But since all frames are IDR, we skip this

	// slice_qp_delta is ALWAYS required (se(v))
	// Value of 0 means use init_qp from PPS
	e.bs.WriteSE(0)

	// slice_cb_qp_offset and slice_cr_qp_offset only if pps_slice_chroma_qp_offsets_present_flag = 1
	// Our PPS has it as 0, so skip

	// deblocking filter stuff skipped (deblocking_filter_control_present_flag = 0)

	// slice_loop_filter_across_slices_enabled_flag not needed
	// because pps_loop_filter_across_slices_enabled_flag = 0

	// byte_alignment(): alignment_bit_equal_to_one followed by alignment_bit_equal_to_zero
	e.bs.FlushWithRBSP()

	sliceHeader := e.bs.Bytes()

	// CABAC slice data
	e.cabac.Reset()
	e.encodeSliceData(frame)
	e.cabac.Finish()

	// Combine header and CABAC data
	result := make([]byte, 0, len(sliceHeader)+len(e.cabac.Bytes()))
	result = append(result, sliceHeader...)
	result = append(result, e.cabac.Bytes()...)

	return result
}

// encodeSliceData encodes the slice data (CTBs) using CABAC.
func (e *HEVCEncoder) encodeSliceData(frame *avutil.Frame) {
	// Process each CTB
	for ctbY := 0; ctbY < e.ctbHeightInUnits; ctbY++ {
		for ctbX := 0; ctbX < e.ctbWidthInUnits; ctbX++ {
			// SAO parameters (disabled in SPS, so skip)

			// Encode coding tree for this CTB
			e.encodeCodingTree(frame, ctbX*e.ctbSize, ctbY*e.ctbSize, e.ctbSize, 0)

			// End of slice segment flag
			isLastCTB := (ctbX == e.ctbWidthInUnits-1) && (ctbY == e.ctbHeightInUnits-1)
			if isLastCTB {
				e.cabac.EncodeTerminate(1)
			} else {
				e.cabac.EncodeTerminate(0)
			}
		}
	}
}

// encodeCodingTree recursively encodes the coding tree.
func (e *HEVCEncoder) encodeCodingTree(frame *avutil.Frame, x, y, size, depth int) {
	// Check if we need to split
	canSplit := size > e.minCbSize && (x+size <= e.width) && (y+size <= e.height)
	mustSplit := (x+size > e.width) || (y+size > e.height)

	if canSplit && !mustSplit {
		// Encode split_cu_flag
		// Context depends on depth
		ctxIdx := depth
		if ctxIdx > 2 {
			ctxIdx = 2
		}

		// For simplest encoding, don't split at minimum sizes
		shouldSplit := size > 8

		if shouldSplit {
			e.cabac.EncodeBin(1, &e.ctxSplitCuFlag[ctxIdx])

			// Recurse into 4 sub-CUs
			halfSize := size / 2
			e.encodeCodingTree(frame, x, y, halfSize, depth+1)
			e.encodeCodingTree(frame, x+halfSize, y, halfSize, depth+1)
			e.encodeCodingTree(frame, x, y+halfSize, halfSize, depth+1)
			e.encodeCodingTree(frame, x+halfSize, y+halfSize, halfSize, depth+1)
			return
		}

		e.cabac.EncodeBin(0, &e.ctxSplitCuFlag[ctxIdx])
	}

	// Encode coding unit (CU)
	e.encodeCodingUnit(frame, x, y, size, depth)
}

// encodeCodingUnit encodes a single coding unit.
func (e *HEVCEncoder) encodeCodingUnit(frame *avutil.Frame, x, y, size, depth int) {
	// For I-slice, skip cu_skip_flag

	// pred_mode_flag (1 = intra) - only for non-I slices, implicit for I-slice

	// part_mode (0 = 2Nx2N for intra)
	// Only signal if CU size > min CU size
	if size > e.minCbSize {
		e.cabac.EncodeBin(1, &e.ctxPartMode[0]) // 2Nx2N
	}

	// Encode prediction unit
	e.encodePredictionUnit(frame, x, y, size)

	// pcm_flag = 0 (implied, we use intra prediction)

	// Transform tree
	e.encodeTransformTree(frame, x, y, size, 0)
}

// encodePredictionUnit encodes intra prediction modes.
func (e *HEVCEncoder) encodePredictionUnit(frame *avutil.Frame, x, y, size int) {
	// Luma prediction mode
	// prev_intra_luma_pred_flag = 1 (use MPM)
	e.cabac.EncodeBin(1, &e.ctxPrevIntraLuma[0])

	// mpm_idx (0 = planar, 1 = DC, 2 = angular)
	// Use DC mode (index 1) for simplicity
	// mpm_idx is signaled as: if idx == 0, write 0; else write 1, then idx-1 in bypass
	e.cabac.EncodeBin(1, &e.ctxPrevIntraLuma[0]) // Not planar
	e.cabac.EncodeBypass(0)                       // DC (index 1)

	// Chroma prediction mode
	// intra_chroma_pred_mode = 4 (derived from luma / DM_CHROMA)
	// Encoded as: 0 means mode 4, else 1 followed by 2 bypass bits
	e.cabac.EncodeBin(0, &e.ctxIntraChroma[0])
}

// encodeTransformTree encodes the transform tree.
func (e *HEVCEncoder) encodeTransformTree(frame *avutil.Frame, x, y, size, depth int) {
	// For simplest case with zero residuals:
	// rqt_root_cbf = 0 (no residual data)
	e.cabac.EncodeBin(0, &e.ctxRqtRootCbf[0])

	// With rqt_root_cbf = 0, no transform data follows
}

// Flush implements Encoder.Flush
func (e *HEVCEncoder) Flush(ctx *EncoderContext) ([]*avutil.Packet, error) {
	// No buffered frames in this simple implementation
	return nil, nil
}

// Close implements Encoder.Close
func (e *HEVCEncoder) Close(ctx *EncoderContext) error {
	return nil
}

// GetExtraData returns VPS+SPS+PPS for container.
func (e *HEVCEncoder) GetExtraData() []byte {
	return e.generateParameterSets()
}
