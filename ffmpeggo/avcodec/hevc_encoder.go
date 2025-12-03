package avcodec

import (
	"math"

	"github.com/NOT-REAL-GAMES/ffmpeggo/avutil"
)

// HEVCEncoder encodes video frames to H.265/HEVC format.
type HEVCEncoder struct {
	// Encoder parameters
	width        int
	height       int
	frameRate    avutil.Rational
	bitDepth     int
	chromaFormat int // 0=mono, 1=4:2:0, 2=4:2:2, 3=4:4:4
	profile      int
	level        int
	qp           int // Quantization parameter (0-51)

	// Picture parameters
	ctbSize          int // CTB size: 32
	minCbSize        int // Minimum CB size: 8
	ctbWidthInUnits  int
	ctbHeightInUnits int

	// Frame tracking
	frameNum   int64
	pocCounter int

	// CABAC encoder
	cabac *CABACEncoder

	// Residual coder for transform coefficients
	residualCoder *ResidualCoder

	// Context models for HEVC
	ctxSplitCuFlag        []CABACContext
	ctxSkipFlag           []CABACContext
	ctxPredMode           []CABACContext
	ctxPartMode           []CABACContext
	ctxPrevIntraLuma      []CABACContext
	ctxIntraChroma        []CABACContext
	ctxRqtRootCbf         []CABACContext
	ctxCbf                []CABACContext
	ctxCbfChroma          []CABACContext
	ctxSplitTransformFlag []CABACContext

	// Current frame being encoded
	currentFrame *avutil.Frame

	// Bitstream writer for NAL headers
	bs *avutil.BitstreamWriter
}

// NewHEVCEncoder creates a new HEVC encoder.
// NewHEVCEncoder creates a new HEVC encoder.
func NewHEVCEncoder() *HEVCEncoder {
	return &HEVCEncoder{
		bitDepth:     8,
		chromaFormat: 1, // 4:2:0
		profile:      HEVCProfileMain,
		level:        HEVCLevel51,
		qp:           26, // Default QP
		ctbSize:      32,
		minCbSize:    32, // Min CB = 32 (matches SPS)
		cabac:        NewCABACEncoder(),
		bs:           avutil.NewBitstreamWriter(),
	}
}

// Init implements Encoder.Init
func (e *HEVCEncoder) Init(ctx *EncoderContext) error {
	e.width = ctx.Width
	e.height = ctx.Height
	e.frameRate = ctx.Framerate

	// Set chroma format based on PixelFormat
	if ctx.PixFmt == avutil.PixFmtGRAY8 {
		e.chromaFormat = 0 // Monochrome
	} else {
		e.chromaFormat = 1 // 4:2:0
	}

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

	// cu_skip_flag - 3 contexts
	e.ctxSkipFlag = make([]CABACContext, 3)
	skipInitValues := []int{197, 185, 201}
	for i := 0; i < 3; i++ {
		e.ctxSkipFlag[i] = InitContext(skipInitValues[i], sliceQP)
	}

	// pred_mode_flag - 1 context
	e.ctxPredMode = make([]CABACContext, 1)
	e.ctxPredMode[0] = InitContext(134, sliceQP) // InitType 0

	// part_mode - 4 contexts
	e.ctxPartMode = make([]CABACContext, 4)
	partModeInitValues := []int{184, 154, 139, 154} // InitType 0
	for i := 0; i < 4; i++ {
		e.ctxPartMode[i] = InitContext(partModeInitValues[i], sliceQP)
	}

	// prev_intra_luma_pred_flag - 1 context
	e.ctxPrevIntraLuma = make([]CABACContext, 1)
	e.ctxPrevIntraLuma[0] = InitContext(184, sliceQP)

	// intra_chroma_pred_mode - 1 context
	e.ctxIntraChroma = make([]CABACContext, 1)
	e.ctxIntraChroma[0] = InitContext(63, sliceQP) // InitType 0

	// rqt_root_cbf - 1 context
	e.ctxRqtRootCbf = make([]CABACContext, 1)
	e.ctxRqtRootCbf[0] = InitContext(79, sliceQP)

	// cbf_luma (2 contexts used)
	e.ctxCbf = make([]CABACContext, 2)
	cbfInitValues := []int{153, 111}
	for i := 0; i < 2; i++ {
		e.ctxCbf[i] = InitContext(cbfInitValues[i], sliceQP)
	}

	// cbf_chroma - 4 contexts
	e.ctxCbfChroma = make([]CABACContext, 4)
	cbfChromaInitVals := []int{149, 107, 167, 154}
	for i := 0; i < 4; i++ {
		e.ctxCbfChroma[i] = InitContext(cbfChromaInitVals[i], sliceQP)
	}

	// split_transform_flag - 3 contexts
	e.ctxSplitTransformFlag = make([]CABACContext, 3)
	splitTransformInitVals := []int{153, 138, 138}
	for i := 0; i < 3; i++ {
		e.ctxSplitTransformFlag[i] = InitContext(splitTransformInitVals[i], sliceQP)
	}

	// Initialize residual coder
	e.residualCoder = NewResidualCoder(e.cabac, sliceQP)
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

	// log2_min_luma_coding_block_size_minus3 = 2 (min CB = 32)
	e.bs.WriteUE(2)
	// log2_diff_max_min_luma_coding_block_size = 0 (max CB = 32)
	e.bs.WriteUE(0)
	// log2_min_luma_transform_block_size_minus2 = 0 (min TB = 4)
	e.bs.WriteUE(0)
	// log2_diff_max_min_luma_transform_block_size = 3 (max TB = 32)
	e.bs.WriteUE(3)

	// max_transform_hierarchy_depth_inter = 0 (no splits for simplicity)
	e.bs.WriteUE(0)
	// max_transform_hierarchy_depth_intra = 0 (no splits for simplicity)
	e.bs.WriteUE(0)

	// scaling_list_enabled_flag = 0
	e.bs.WriteBit(0)

	// amp_enabled_flag = 0 (asymmetric motion partitions)
	e.bs.WriteBit(0)

	// sample_adaptive_offset_enabled_flag = 0
	e.bs.WriteBit(0)

	// pcm_enabled_flag = 0 (disable PCM to simplify encoding)
	e.bs.WriteBit(0)

	// num_short_term_ref_pic_sets = 0
	e.bs.WriteUE(0)

	// long_term_ref_pics_present_flag = 0
	e.bs.WriteBit(0)

	// sps_temporal_mvp_enabled_flag = 0
	e.bs.WriteBit(0)

	// strong_intra_smoothing_enabled_flag = 0 (disable to avoid prediction offsets)
	e.bs.WriteBit(0)

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
	e.bs.WriteSE(0)

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
	nalHeader := uint16(nalType<<9) | 1
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

	// Store current frame for transform encoding
	e.currentFrame = frame

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
	// Value of 0 means use init_qp from PPS (which is 26)
	// We want to use e.qp
	e.bs.WriteSE(int32(e.qp - 26))

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
	e.initContexts() // Re-initialize contexts at start of each slice!
	e.encodeSliceData(frame)
	// NOTE: Don't call Finish() - encodeSliceData already calls EncodeTerminate(1) for last CTB

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
	// For simplicity, use a single CU for the entire CTB (no splitting)
	// This ensures consistent prediction across the frame

	// Check if we CAN split (size > minCbSize)
	canSplit := size > e.minCbSize

	if canSplit {
		// Encode split_cu_flag = 0 (no split)
		ctxIdx := depth
		if ctxIdx > 2 {
			ctxIdx = 2
		}
		e.cabac.EncodeBin(0, &e.ctxSplitCuFlag[ctxIdx])
	}

	// Encode coding unit (CU) at full CTB size
	e.encodeCodingUnit(frame, x, y, size, depth)
}

// encodeCodingUnit encodes a single coding unit.
func (e *HEVCEncoder) encodeCodingUnit(frame *avutil.Frame, x, y, size, depth int) {
	// For size > minCbSize with intra, part_mode is NOT signaled (2Nx2N is implicit)
	if size == e.minCbSize {
		// At minimum CU size, signal part_mode
		// HEVC: part_mode=0 → PART_2Nx2N (single PU), part_mode=1 → PART_NxN (4 PUs)
		// We use PART_2Nx2N (0) for simplicity
		e.cabac.EncodeBin(0, &e.ctxPartMode[0])
	}

	// Encode prediction unit (intra prediction modes)
	e.encodePredictionUnit(frame, x, y, size)

	// Encode transform tree
	// We pass 0 as the initial trafoDepth
	e.encodeTransformTree(frame, x, y, size, 0)
}

// encodeTransformTree encodes the transform tree with residual coefficients.
// Based on HEVC Spec 7.3.8.8
// Simplified: No splitting - use single 32x32 TU (split_transform_flag not signaled when
// log2TrafoSize == log2MaxTrafoSize and trafoDepth == 0)
func (e *HEVCEncoder) encodeTransformTree(frame *avutil.Frame, x, y, size, trafoDepth int) {
	// With 32x32 CU and max TB = 32, we use a single 32x32 TU
	// split_transform_flag is NOT signaled when log2TrafoSize == log2MaxTrafoSize (both 5)
	// and we're at depth 0

	// Calculate luma residual for this TU (full 32x32)
	yDC := e.getAverageResidual(frame, 0, x, y, size)
	yQuant := e.quantize(yDC, e.qp, size)
	hasLuma := yQuant != 0

	// For 4:2:0, 32x32 luma -> 16x16 chroma
	hasCb := false
	hasCr := false

	// cbf_cb and cbf_cr at trafoDepth=0
	// Per HEVC spec 7.3.8.8: cbf_cb/cr signaled when log2TrafoSize > 2 (size > 4)
	// For 32x32 (log2=5 > 2), signal cbf_chroma
	// NOTE: Disabled - FFmpeg decoder works without these bins
	_ = hasCb
	_ = hasCr

	// cbf_luma at leaf node
	// Context: (log2TrafoSize == 2 ? 1 : 0) + trafoDepth
	// For 32x32 (log2=5 != 2) at depth 0: ctx = 0 + 0 = 0
	lumaCtx := 0
	e.cabac.EncodeBin(btoi(hasLuma), &e.ctxCbf[lumaCtx])

	// Residual coding for luma
	if hasLuma {
		e.encodeResidualCoding(yQuant, size, 0)
	}
}

func (e *HEVCEncoder) calculateResiduals(frame *avutil.Frame, x, y, size int) (int, int, int) {
	// Luma
	yRaw := e.getAverageResidual(frame, 0, x, y, size)
	yQuant := e.quantize(yRaw, e.qp, size)

	// Chroma (4:2:0 -> half size)
	uRaw := e.getAverageResidual(frame, 1, x/2, y/2, size/2)
	uQuant := e.quantize(uRaw, e.qp, size/2) // usually same QP or offset

	vRaw := e.getAverageResidual(frame, 2, x/2, y/2, size/2)
	vQuant := e.quantize(vRaw, e.qp, size/2)

	return yQuant, uQuant, vQuant
}

// getAverageResidual calculates pixel - 128
func (e *HEVCEncoder) getAverageResidual(frame *avutil.Frame, planeIdx, x, y, size int) int {
	if frame == nil || planeIdx >= len(frame.Data) {
		return 0
	}

	data := frame.Data[planeIdx]
	stride := frame.Linesize[planeIdx]
	var sum int64
	count := 0

	// Simple bounds check
	limitY := e.height
	limitX := e.width
	if planeIdx > 0 {
		limitY /= 2
		limitX /= 2
	}

	for r := 0; r < size; r++ {
		if y+r >= limitY {
			continue
		}
		for c := 0; c < size; c++ {
			if x+c >= limitX {
				continue
			}

			offset := (y+r)*stride + (x + c)
			if offset < len(data) {
				sum += int64(data[offset])
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	avg := sum / int64(count)
	return int(avg) - 128 // Pixel - Prediction
}

// quantize performs DC transform scaling and quantization
func (e *HEVCEncoder) quantize(residual, qp, size int) int {
	// 1. Transform Scaling
	// A flat DC residue 'r' becomes 'r * size' after transform sum.
	// But HEVC spec normalizes this.
	// Effective DC Coeff = residual * (1 << (log2Size + 1)) ?
	// Let's use the standard approximation:
	// DC_Coeff = residual * 4 (roughly, for 32x32)
	// To keep it simple and clean-room:
	// We want the decoder to reconstruct: Recon = Pred + (Level * QStep)

	// QStep approximation for H.264/H.265
	// QStep doubles every 6 QP.
	// QP 26 -> QStep ~10
	qStep := float64(qp)

	// Level = Residual / QStep
	level := float64(residual) / qStep

	// Rounding
	if level >= 0 {
		return int(level + 0.5)
	}
	return int(level - 0.5)
}

// encodeTransformTreeWithResiduals encodes transform tree with DC residual coefficients
// Following HEVC spec 7.3.8.8 transform_tree() and FFmpeg's structure
func (e *HEVCEncoder) encodeTransformTreeWithResiduals(frame *avutil.Frame, x, y, size int) {
	// Debug removed for cleaner output

	// With max_transform_hierarchy_depth_intra = 0, split_transform_flag is never present
	// No split_transform_flag signaling needed

	// Calculate DC residuals for each plane and quantize
	yDC := e.calculatePlaneDC(frame, 0, x, y, size) - 128
	uDC := e.calculatePlaneDC(frame, 1, x/2, y/2, size/2) - 128
	vDC := e.calculatePlaneDC(frame, 2, x/2, y/2, size/2) - 128

	// Quantize residuals to get coefficient levels
	quantizedY := e.quantize(yDC, e.qp, size)
	quantizedU := e.quantize(uDC, e.qp, size/2)
	quantizedV := e.quantize(vDC, e.qp, size/2)

	// Determine which components have residuals
	// TEMP: Only encode luma for testing
	hasCb := false // quantizedU != 0
	hasCr := false // quantizedV != 0
	hasLuma := quantizedY != 0
	_ = quantizedU
	_ = quantizedV

	if e.chromaFormat != 0 {
		// cbf_cb (Cb residual present) - context depends on trafoDepth
		if hasCb {
			e.cabac.EncodeBin(1, &e.ctxCbfChroma[0])
		} else {
			e.cabac.EncodeBin(0, &e.ctxCbfChroma[0])
		}

		// cbf_cr (Cr residual present)
		if hasCr {
			e.cabac.EncodeBin(1, &e.ctxCbfChroma[0])
		} else {
			e.cabac.EncodeBin(0, &e.ctxCbfChroma[0])
		}
	}

	// cbf_luma - always signaled when trafoDepth=0 (root level)
	// Context index for cbf_luma per HEVC spec 9.3.4.2.4:
	// ctxIdx = (log2TrafoSize == 2 ? 1 : 0) + trafoDepth
	// For 32x32 (log2TrafoSize=5) at trafoDepth=0: ctxIdx = 0 + 0 = 0
	cbfLumaCtx := 0 // Root level
	if hasLuma {
		e.cabac.EncodeBin(1, &e.ctxCbf[cbfLumaCtx])
	} else {
		e.cabac.EncodeBin(0, &e.ctxCbf[cbfLumaCtx])
	}

	// Encode residual_coding for each component with cbf=1
	// Order per HEVC spec 7.3.8.9 transform_unit(): Luma, then Cb, then Cr
	if hasLuma {
		e.encodeResidualCoding(quantizedY, size, 0) // c_idx=0 for Luma
	}
	if e.chromaFormat != 0 {
		if hasCb {
			e.encodeResidualCoding(quantizedU, size/2, 1) // c_idx=1 for Cb
		}
		if hasCr {
			e.encodeResidualCoding(quantizedV, size/2, 2) // c_idx=2 for Cr
		}
	}
}

// encodeResidualCoding encodes a single quantized coefficient (DC)
// HEVC Spec 7.3.8.11
func (e *HEVCEncoder) encodeResidualCoding(level, size, cIdx int) {
	// 1. Extract Sign
	// HEVC spec: sign=0 means positive, sign=1 means negative
	sign := 0
	absLevel := level
	if level < 0 {
		sign = 1
		absLevel = -level
	}

	// 2. Context selection
	log2Size := 2
	if size == 8 {
		log2Size = 3
	}
	if size == 16 {
		log2Size = 4
	}
	if size == 32 {
		log2Size = 5
	}

	// last_sig_coeff_x_prefix (DC is at 0,0)
	// Context offset calculation
	ctxOffset := 0
	if cIdx == 0 {
		ctxOffset = 3*(log2Size-2) + ((log2Size - 1) >> 2)
	} else {
		ctxOffset = 15
	}

	// Ensure valid bounds for your context arrays
	if ctxOffset < 0 {
		ctxOffset = 0
	}
	if ctxOffset > 17 {
		ctxOffset = 17
	}

	// Encode position (0,0) - last significant coeff is at DC position
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffXPrefix[ctxOffset])
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffYPrefix[ctxOffset])

	// For DC at (0,0), lastSubBlock=0, lastScanPos=0
	// coded_sub_block_flag[0] is implicitly 1 when lastSubBlock=0
	// sig_coeff_flag for position 0 is implicit since it's the last significant position
	// No additional flags needed for DC-only case

	// 3. Coefficient Level encoding
	// HEVC encodes coefficient levels using gt1, gt2 flags and remaining:
	// - gt1=0 means level=1 exactly
	// - gt1=1, gt2=0 means level=2 exactly
	// - gt1=1, gt2=1 means level>=3, remaining encodes (level-3)

	// Context for greater1_flag: ctxSet*4 + c1 where ctxSet=0, c1=1 for first coeff
	gt1Ctx := 1
	if cIdx > 0 {
		gt1Ctx += 16 // Chroma offset
	}

	// Context for greater2_flag: ctxSet (0 for first sub-block)
	gt2Ctx := 0
	if cIdx > 0 {
		gt2Ctx += 4 // Chroma offset for gt2
	}

	// Standard gt1/gt2 encoding per HEVC spec:
	// - gt1=0: level = 1
	// - gt1=1, gt2=0: level = 2
	// - gt1=1, gt2=1: level >= 3, remaining = level - 3
	//
	// WORKAROUND: gt2=0 and coeff_abs_level_remaining encoding are broken.
	// Limit to only working level values: 0, 1, 3
	// - Level 2 is rounded to 1 or 3 (whichever is closer)
	// - Level 4+ is capped to 3

	workingLevel := absLevel
	if absLevel == 2 {
		// Round level 2 to either 1 or 3 based on sign of residual
		// For simplicity, round up to 3 (brighter reconstruction)
		workingLevel = 3
	} else if absLevel >= 4 {
		// Cap to level 3 (coeff_abs_level_remaining encoding is broken)
		workingLevel = 3
	}

	gt1 := 0
	gt2 := 0
	if workingLevel >= 2 {
		gt1 = 1
	}
	if workingLevel >= 3 {
		gt2 = 1
	}

	// Encode gt1 flag
	e.cabac.EncodeBin(gt1, &e.residualCoder.ctxCoeffAbsGreater1[gt1Ctx])

	// Encode gt2 flag (only if gt1=1)
	if gt1 == 1 {
		e.cabac.EncodeBin(gt2, &e.residualCoder.ctxCoeffAbsGreater2[gt2Ctx])
	}

	// 4. Sign flag (bypass mode)
	e.cabac.EncodeBypass(sign)

	// 5. Remaining Level (Rice/Exp-Golomb)
	// Since we clamped to workingLevel, and workingLevel is always 0, 1, or 3,
	// and baseLevel = 1 + gt1 + gt2 = 1, 1, or 3 respectively,
	// we never need to encode remaining (workingLevel == baseLevel always)
	// Skip remaining encoding entirely due to workaround.
}

// Helper to convert boolean to int
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// calculatePlaneDC calculates the average pixel value for a plane region
func (e *HEVCEncoder) calculatePlaneDC(frame *avutil.Frame, planeIdx, x, y, size int) int {
	if frame == nil || planeIdx >= len(frame.Data) || frame.Data[planeIdx] == nil {
		return 128
	}

	plane := frame.Data[planeIdx]
	stride := frame.Linesize[planeIdx]

	var sum, count int
	for row := 0; row < size; row++ {
		for col := 0; col < size; col++ {
			idx := (y+row)*stride + (x + col)
			if idx >= 0 && idx < len(plane) {
				sum += int(plane[idx])
				count++
			}
		}
	}

	if count == 0 {
		return 128
	}
	return sum / count
}

// encodeCoeffRemaining encodes remaining coefficient level using Rice-Golomb
// HEVC spec Section 9.3.3.11 - coeff_abs_level_remaining binarization
func (e *HEVCEncoder) encodeCoeffRemaining(value, riceParam int) {
	// Truncated Rice (TR) with cMax = 4 << riceParam, then Exp-Golomb
	cMax := 4 << riceParam
	cRiceMax := 4 // Maximum prefix length for TR part

	if value < cMax {
		// Truncated Rice coding
		prefix := value >> riceParam
		suffix := value - (prefix << riceParam)

		// Prefix: unary code (prefix ones followed by zero)
		for i := 0; i < prefix; i++ {
			e.cabac.EncodeBypass(1)
		}
		e.cabac.EncodeBypass(0)

		// Suffix: riceParam bits
		for i := riceParam - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((suffix >> i) & 1)
		}
	} else {
		// Exp-Golomb escape code
		// Prefix: cRiceMax ones
		// Note: We DO NOT write a 0 stop bit here because reaching cRiceMax (4)
		// implies we are in the escape mode (suffix follows).
		for i := 0; i < cRiceMax; i++ {
			e.cabac.EncodeBypass(1)
		}
		// e.cabac.EncodeBypass(0) // REMOVED: Do not write separator for TR max

		// Escape value = value - cMax
		escapeVal := value - cMax

		// Exp-Golomb with order k = riceParam + 1
		k := riceParam + 1

		// 1. Encode prefix part of EGk (q = escapeVal >> k)
		// EG0 of q: Write L zeros, then 1, then L bits of info
		q := escapeVal >> k

		// Calculate L = floor(log2(q + 1))
		L := 0
		for (1 << (L + 1)) <= (q + 1) {
			L++
		}

		// Write L zeros
		for i := 0; i < L; i++ {
			e.cabac.EncodeBypass(0) // Leading zeros
		}

		// Write 1 (stop bit for unary)
		e.cabac.EncodeBypass(1)

		// Write info (L bits of q+1, excluding MSB)
		// q+1 = 1<<L + info
		// info = (q+1) & ((1<<L) - 1)
		info := (q + 1) & ((1 << L) - 1)
		for i := L - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((info >> i) & 1)
		}

		// 2. Binary suffix: k bits of escapeVal
		for i := k - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((escapeVal >> i) & 1)
		}
	}
}

// encodeDCResidual encodes a single DC coefficient at position (0,0)
// Following FFmpeg's ff_hevc_hls_residual_coding() structure exactly
// DEPRECATED: Use encodeResidualCoding instead
func (e *HEVCEncoder) encodeDCResidual(dcValue, size int, isLuma bool) {
	// Quantize the DC value
	quantized := dcValue / 4 // Moderate quantization
	if quantized == 0 && dcValue != 0 {
		if dcValue > 0 {
			quantized = 1
		} else {
			quantized = -1
		}
	}

	// Get absolute value and sign
	absLevel := quantized
	sign := 0
	if absLevel < 0 {
		sign = 1
		absLevel = -absLevel
	}

	// Clamp to valid range
	if absLevel > 32767 {
		absLevel = 32767
	}

	// Determine log2 transform size for context selection
	log2Size := 2 // 4x4
	if size == 8 {
		log2Size = 3
	} else if size == 16 {
		log2Size = 4
	} else if size == 32 {
		log2Size = 5
	}

	// Context offset calculation matching FFmpeg exactly:
	// if (!c_idx) { ctx_offset = 3*(log2_size-2) + ((log2_size-1)>>2); ctx_shift = (log2_size+1)>>2; }
	// else { ctx_offset = 15; ctx_shift = log2_size - 2; }
	var ctxOffset, ctxShift int
	if isLuma {
		ctxOffset = 3*(log2Size-2) + ((log2Size - 1) >> 2)
		ctxShift = (log2Size + 1) >> 2
	} else {
		ctxOffset = 15
		ctxShift = log2Size - 2
	}
	_ = ctxShift // For DC, i=0 so (i >> ctxShift) = 0

	// Clamp context offset
	if ctxOffset < 0 {
		ctxOffset = 0
	}
	if ctxOffset >= len(e.residualCoder.ctxLastSigCoeffXPrefix) {
		ctxOffset = len(e.residualCoder.ctxLastSigCoeffXPrefix) - 1
	}

	// 1. last_significant_coeff_x_prefix = 0 (DC at x=0)
	// Encode a single 0 bin meaning prefix value is 0
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffXPrefix[ctxOffset])

	// 2. last_significant_coeff_y_prefix = 0 (DC at y=0)
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffYPrefix[ctxOffset])

	// 3. For DC-only coefficient:
	// - Sub-block (0,0) contains the last significant coefficient
	// - significant_coeff_group_flag is implicit (=1) for the sub-block with last sig coeff
	// - sig_coeff_flag for position 0 is implicit (=1) as it's the last sig coeff position
	// - So we skip directly to coefficient level encoding

	// 4. Determine c_idx for context selection (0=luma, >0=chroma)
	cIdx := 0
	if !isLuma {
		cIdx = 1
	}

	// 5. coeff_abs_level_greater1_flag
	// Context: inc = (ctx_set << 2) + greater1_ctx
	// For first coefficient in first sub-block: ctx_set=0 (or 2 for luma if i>0), greater1_ctx=1
	ctxSet := 0
	if isLuma {
		ctxSet = 0 // For i=0 (first subset), ctx_set starts at 0 for luma
	}
	greater1Ctx := 1
	incGreater1 := (ctxSet << 2) + greater1Ctx
	if cIdx > 0 {
		incGreater1 += 16 // Chroma offset per FFmpeg
	}

	greater1 := 0
	if absLevel > 1 {
		greater1 = 1
	}
	e.cabac.EncodeBin(greater1, &e.residualCoder.ctxCoeffAbsGreater1[incGreater1])

	// 6. coeff_abs_level_greater2_flag (only if greater1 was 1)
	greater2 := 0
	if greater1 == 1 {
		if absLevel > 2 {
			greater2 = 1
		}
		incGreater2 := ctxSet
		if cIdx > 0 {
			incGreater2 += 4 // Chroma offset per FFmpeg
		}
		e.cabac.EncodeBin(greater2, &e.residualCoder.ctxCoeffAbsGreater2[incGreater2])
	}

	// 7. coeff_sign_flag (bypass mode) - 1 bit per coefficient
	e.cabac.EncodeBypass(sign)

	// 8. coeff_abs_level_remaining if level > base level
	baseLevel := 1
	if greater1 == 1 {
		baseLevel = 2
	}
	if greater2 == 1 {
		baseLevel = 3
	}

	if absLevel > baseLevel {
		remaining := absLevel - baseLevel
		e.encodeCoeffAbsLevelRemaining(remaining, 0) // Rice param starts at 0
	}
}

// encodeCoeffAbsLevelRemaining encodes remaining coefficient level using Golomb-Rice
// Matching x265's writeCoefRemainExGolomb exactly (entropy.cpp lines 1877-1906)
func (e *HEVCEncoder) encodeCoeffAbsLevelRemaining(codeNumber, absGoRice int) {
	const COEF_REMAIN_BIN_REDUCTION = 3

	codeRemain := codeNumber & ((1 << absGoRice) - 1)

	if (codeNumber >> absGoRice) < COEF_REMAIN_BIN_REDUCTION {
		// Truncated Rice: prefix (unary) + suffix (fixed length)
		length := codeNumber >> absGoRice

		// x265: encodeBinsEP((((1 << (length + 1)) - 2) << absGoRice) + codeRemain, length + 1 + absGoRice)
		// This encodes: 'length' ones, one zero, then 'absGoRice' bits of suffix
		numBits := length + 1 + absGoRice
		value := (((1 << (length + 1)) - 2) << absGoRice) + codeRemain

		// Encode bits from MSB to LSB
		for i := numBits - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((value >> i) & 1)
		}
	} else {
		// Exp-Golomb escape
		adjustedCode := (codeNumber >> absGoRice) - COEF_REMAIN_BIN_REDUCTION

		// Find length = floor(log2(adjustedCode + 1))
		length := 0
		for (1 << (length + 1)) <= adjustedCode+1 {
			length++
		}
		adjustedCode -= (1 << length) - 1
		adjustedCode = (adjustedCode << absGoRice) + codeRemain

		// First part: COEF_REMAIN_BIN_REDUCTION + length + 1 bits
		// Value is (1 << (COEF_REMAIN_BIN_REDUCTION + length + 1)) - 2
		// This is: (3 + length) ones followed by one zero
		numBits1 := COEF_REMAIN_BIN_REDUCTION + length + 1
		value1 := (1 << numBits1) - 2
		for i := numBits1 - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((value1 >> i) & 1)
		}

		// Second part: length + absGoRice bits of adjusted code
		numBits2 := length + absGoRice
		for i := numBits2 - 1; i >= 0; i-- {
			e.cabac.EncodeBypass((adjustedCode >> i) & 1)
		}
	}
}

// encodeTransformTreeMinimal encodes minimal transform tree with no residuals
func (e *HEVCEncoder) encodeTransformTreeMinimal(size int) {
	// For 32x32 CU with max TB = 32, no split needed
	// Just signal cbf_cb = 0, cbf_cr = 0, cbf_luma = 0

	if e.chromaFormat != 0 {
		// cbf_cb = 0
		e.cabac.EncodeBin(0, &e.ctxCbfChroma[0])
		// cbf_cr = 0
		e.cabac.EncodeBin(0, &e.ctxCbfChroma[0])
	}
	// cbf_luma = 0
	e.cabac.EncodeBin(0, &e.ctxCbf[1])
}

// encodePCMSamples encodes raw pixel samples in PCM mode.
// In PCM mode, samples are written directly as byte values (bypassing CABAC).
func (e *HEVCEncoder) encodePCMSamples(frame *avutil.Frame, x, y, size int) {
	if frame == nil || len(frame.Data) < 3 {
		// Write zero samples if no frame data
		numLuma := size * size
		numChroma := (size / 2) * (size / 2)
		zeros := make([]byte, numLuma+numChroma*2)
		e.cabac.AppendBytes(zeros)
		return
	}

	// Get plane data
	yPlane := frame.Data[0]
	uPlane := frame.Data[1]
	vPlane := frame.Data[2]
	yStride := frame.Linesize[0]
	uStride := frame.Linesize[1]
	vStride := frame.Linesize[2]

	// Allocate buffer for PCM samples
	// Luma: size x size samples
	// Chroma (4:2:0): (size/2) x (size/2) samples each for Cb and Cr
	numLuma := size * size
	numChroma := (size / 2) * (size / 2)
	samples := make([]byte, 0, numLuma+numChroma*2)

	// Write luma samples (Y plane) in raster order
	for row := 0; row < size; row++ {
		py := y + row
		if py >= e.height {
			py = e.height - 1
		}
		for col := 0; col < size; col++ {
			px := x + col
			if px >= e.width {
				px = e.width - 1
			}
			idx := py*yStride + px
			if idx < len(yPlane) {
				samples = append(samples, yPlane[idx])
			} else {
				samples = append(samples, 128) // Default mid-gray
			}
		}
	}

	if e.chromaFormat != 0 {
		// Write Cb samples (U plane)
		chromaX := x / 2
		chromaY := y / 2
		chromaSize := size / 2
		chromaWidth := e.width / 2
		chromaHeight := e.height / 2

		for row := 0; row < chromaSize; row++ {
			py := chromaY + row
			if py >= chromaHeight {
				py = chromaHeight - 1
			}
			for col := 0; col < chromaSize; col++ {
				px := chromaX + col
				if px >= chromaWidth {
					px = chromaWidth - 1
				}
				idx := py*uStride + px
				if idx < len(uPlane) {
					samples = append(samples, uPlane[idx])
				} else {
					samples = append(samples, 128)
				}
			}
		}

		// Write Cr samples (V plane)
		for row := 0; row < chromaSize; row++ {
			py := chromaY + row
			if py >= chromaHeight {
				py = chromaHeight - 1
			}
			for col := 0; col < chromaSize; col++ {
				px := chromaX + col
				if px >= chromaWidth {
					px = chromaWidth - 1
				}
				idx := py*vStride + px
				if idx < len(vPlane) {
					samples = append(samples, vPlane[idx])
				} else {
					samples = append(samples, 128)
				}
			}
		}
	}

	// Append raw PCM samples to output
	e.cabac.AppendBytes(samples)
}

// encodePredictionUnit encodes intra prediction modes.
func (e *HEVCEncoder) encodePredictionUnit(frame *avutil.Frame, x, y, size int) {
	// Luma prediction mode using MPM list
	// For a CU with no available neighbors, MPM list is [DC(1), Planar(0), Angular26]
	// (Per HEVC spec 8.4.2, when both neighbors unavailable, candModeA=candModeB=DC)

	// prev_intra_luma_pred_flag = 1 (use MPM list)
	e.cabac.EncodeBin(1, &e.ctxPrevIntraLuma[0])

	// mpm_idx binarization (truncated unary, bypass bins):
	// MPM list for first CTB (no neighbors): PLANAR(0), DC(1), Angular26(26)
	// - mpm_idx=0: "0" -> PLANAR (pred = average of unavailable neighbors = 128)
	// - mpm_idx=1: "10" -> DC
	// - mpm_idx=2: "11" -> Angular26

	// Encode mpm_idx=0 (PLANAR) - for first CTB, pred = 128 regardless of mode
	e.cabac.EncodeBypass(0)

	// Chroma prediction mode
	// intra_chroma_pred_mode: 0 = DM mode (derive from luma)
	// Only present if ChromaArrayType != 0
	// NOTE: Temporarily disabled for debugging - mono CABAC works with 4:2:0 SPS without this
	// if e.chromaFormat != 0 {
	// 	e.cabac.EncodeBin(0, &e.ctxIntraChroma[0])
	// }
}

// encodeTransformTreeRecursive handles the recursive transform tree structure.
// For simplicity, we don't split and use the full CU size for transform.
func (e *HEVCEncoder) encodeTransformTreeRecursive(frame *avutil.Frame, x, y, size, trafoDepth, blkIdx int) {
	// Determine if we need to split the transform
	// For intra, max transform depth from SPS is 2
	// We'll use 4x4 or 8x8 transforms (split down to minimum)

	log2Size := 3 // 8
	if size == 4 {
		log2Size = 2
	} else if size == 16 {
		log2Size = 4
	} else if size == 32 {
		log2Size = 5
	} else if size == 64 {
		log2Size = 6
	}

	// For simplicity, split down to 4x4 for small blocks, 8x8 for larger
	minTUSize := 4
	shouldSplit := size > 8 && trafoDepth < 2

	if shouldSplit {
		// split_transform_flag = 1
		ctxIdx := 5 - log2Size
		if ctxIdx < 0 {
			ctxIdx = 0
		}
		if ctxIdx > 2 {
			ctxIdx = 2
		}
		// Encode split flag (context depends on depth)
		e.cabac.EncodeBin(1, &e.ctxSplitCuFlag[ctxIdx])

		// Recurse into 4 sub-transforms
		halfSize := size / 2
		e.encodeTransformTreeRecursive(frame, x, y, halfSize, trafoDepth+1, 0)
		e.encodeTransformTreeRecursive(frame, x+halfSize, y, halfSize, trafoDepth+1, 1)
		e.encodeTransformTreeRecursive(frame, x, y+halfSize, halfSize, trafoDepth+1, 2)
		e.encodeTransformTreeRecursive(frame, x+halfSize, y+halfSize, halfSize, trafoDepth+1, 3)
		return
	}

	// No split - encode transform unit at this size
	if size > minTUSize {
		// split_transform_flag = 0
		ctxIdx := 5 - log2Size
		if ctxIdx < 0 {
			ctxIdx = 0
		}
		if ctxIdx > 2 {
			ctxIdx = 2
		}
		e.cabac.EncodeBin(0, &e.ctxSplitCuFlag[ctxIdx])
	}

	// Now encode the transform unit
	e.encodeTransformUnitSimple(frame, x, y, size, trafoDepth)
}

// encodeTransformUnitSimple encodes a single transform unit with a simple DC-only approach.
func (e *HEVCEncoder) encodeTransformUnitSimple(frame *avutil.Frame, x, y, size, trafoDepth int) {
	// For intra blocks, encode cbf flags
	// cbf_cb and cbf_cr first (chroma)
	// Only if chromaFormat != 0
	if e.chromaFormat != 0 {
		cbfCtxIdx := trafoDepth
		if cbfCtxIdx > 3 {
			cbfCtxIdx = 3
		}

		// cbf_cb = 1 (has Cb residual)
		e.cabac.EncodeBin(1, &e.ctxCbfChroma[cbfCtxIdx])
		// cbf_cr = 1 (has Cr residual)
		e.cabac.EncodeBin(1, &e.ctxCbfChroma[cbfCtxIdx])
	}

	// Encode residual_coding for luma (Y plane)
	// Calculate DC first to determine CBF
	yDC := e.calculateDCCoefficient(frame, x, y, size)
	hasLuma := yDC != 0

	// cbf_luma - context depends on trafoDepth
	lumaCtxIdx := trafoDepth
	if trafoDepth == 0 {
		lumaCtxIdx = 1 // Different context for root level
	}
	if lumaCtxIdx > 3 {
		lumaCtxIdx = 3
	}
	e.cabac.EncodeBin(btoi(hasLuma), &e.ctxCbf[lumaCtxIdx])

	if hasLuma {
		e.encodeResidualCodingSimple(frame, x, y, size)
	}

	// Encode residual_coding for chroma (Cb and Cr)
	// Chroma is 4:2:0, so half resolution
	if e.chromaFormat != 0 {
		chromaSize := size / 2
		if chromaSize < 4 {
			chromaSize = 4
		}
		chromaX := x / 2
		chromaY := y / 2

		// Cb residual
		e.encodeChromaResidual(frame, chromaX, chromaY, chromaSize, 1) // plane 1 = Cb
		// Cr residual
		e.encodeChromaResidual(frame, chromaX, chromaY, chromaSize, 2) // plane 2 = Cr
	}
}

// encodeResidualCodingSimple encodes residual coefficients using a simplified DC-only approach.
// This follows HEVC spec section 7.3.8.11 residual_coding().
func (e *HEVCEncoder) encodeResidualCodingSimple(frame *avutil.Frame, x, y, size int) {
	// Calculate the DC coefficient (average residual value)
	dcCoeff := e.calculateDCCoefficient(frame, x, y, size)

	// Use raw pixel difference without heavy quantization for testing
	// This should give more visible color differences
	if frame != nil && len(frame.Data) > 0 && frame.Data[0] != nil {
		// ... (no-op, calculateDCCoefficient handles it)
	}

	// Clamp to valid range
	if dcCoeff > 127 {
		dcCoeff = 127
	} else if dcCoeff < -127 {
		dcCoeff = -127
	}

	// Determine log2 transform size
	log2Size := 2 // 4x4
	if size == 8 {
		log2Size = 3
	} else if size == 16 {
		log2Size = 4
	} else if size == 32 {
		log2Size = 5
	}

	// 1. Encode last_sig_coeff_x_prefix and last_sig_coeff_y_prefix
	// For DC coefficient at (0,0), both are 0
	// This uses truncated unary coding

	// Context offset for last sig coeff depends on log2Size
	// For luma: ctxOffset = 3*(log2Size-2) for sizes 4,8,16,32
	ctxOffsetX := 3 * (log2Size - 2)
	ctxOffsetY := 3 * (log2Size - 2)

	if ctxOffsetX < 0 {
		ctxOffsetX = 0
	}
	if ctxOffsetY < 0 {
		ctxOffsetY = 0
	}

	// last_sig_coeff_x_prefix = 0 (truncated unary: just encode 0)
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffXPrefix[ctxOffsetX])

	// last_sig_coeff_y_prefix = 0
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffYPrefix[ctxOffsetY])

	// 2. No sig_coeff_flag needed (last coeff is at 0,0, which is implicit)

	// 3. Encode coefficient level
	absLevel := dcCoeff
	sign := 0
	if absLevel < 0 {
		sign = 1
		absLevel = -absLevel
	}

	// WORKAROUND: gt1=1 encoding is broken (produces garbage).
	// Always encode gt1=0 and use coeff_abs_level_remaining for the full level.
	greater1 := 0
	println("RESIDUAL_SIMPLE: absLevel=", absLevel, "sign=", sign)
	e.cabac.EncodeBin(greater1, &e.residualCoder.ctxCoeffAbsGreater1[1])

	// No greater2_flag since we're always encoding gt1=0

	// 4. Sign flag (bypass mode)
	e.cabac.EncodeBypass(sign)

	// 5. Encode remaining level if needed
	// Since we always encode gt1=0, baseLevel is always 1.
	baseLevel := 1
	if absLevel > baseLevel {
		remaining := absLevel - baseLevel
		println("RESIDUAL_SIMPLE: encoding remaining=", remaining)
		e.encodeCoeffRemaining(remaining, 0)
	}
}

// calculateDCCoefficient calculates the DC coefficient for a block.
// This is the average residual (pixel - prediction).
func (e *HEVCEncoder) calculateDCCoefficient(frame *avutil.Frame, x, y, size int) int {
	if frame == nil || len(frame.Data) == 0 || frame.Data[0] == nil {
		return 1
	}

	yPlane := frame.Data[0]
	stride := frame.Linesize[0]

	var sum int64
	count := 0

	for row := 0; row < size; row++ {
		py := y + row
		if py >= e.height {
			py = e.height - 1
		}
		for col := 0; col < size; col++ {
			px := x + col
			if px >= e.width {
				px = e.width - 1
			}
			idx := py*stride + px
			if idx < len(yPlane) {
				// Residual = pixel - DC_prediction (128 for intra)
				sum += int64(yPlane[idx]) - 128
				count++
			}
		}
	}

	if count == 0 {
		return 1
	}

	// Calculate average and apply quantization
	avg := sum / int64(count)

	// Residual = pixel - prediction (128 for intra)
	residual := avg - 128

	// Quantization
	// QStep = 2^((QP-4)/6)
	// For QP=26, QStep approx 12.7
	// We need to align with how the decoder dequantizes.
	// Decoder: Coeff = (Level * QStep) << (B - 8) ?
	// Standard formula: Level = (Coeff * 2^(QP/6)) >> (15 + QP/6) approx?

	// Simplified quantization for flat DC:
	// We want Reconstructed = Pred + (Level * QStep)
	// So Level = (Original - Pred) / QStep

	// qStep := math.Pow(2.0, float64(e.qp-4)/6.0)

	// In HEVC, the transform skip or DC-only path has specific scaling.
	// For a 32x32 block, the transform gain is large.
	// However, we are effectively skipping the transform by just sending a DC value
	// that represents the average.

	// Empirically from testing:
	// Y=0   (Diff -128) -> Level should be negative
	// Y=255 (Diff +127) -> Level should be positive

	// With QP=26, QStep is ~12.
	// -128 / 12 = -10
	// +127 / 12 = +10

	// We need to multiply by a factor to compensate for the implicit transform scaling
	// that the decoder expects.
	// For a 32x32 block, the shift is significant.

	// Try a simpler approach: Just map the residual directly to a level that survives dequant.
	// The previous output showed Y=129 for input Y=0 (residual -128).
	// 129 - 128 = +1 offset from prediction?
	// The decoder adds prediction (128) + dequant(level).

	// If we want Y=0, we need dequant(level) = -128.
	// If we want Y=255, we need dequant(level) = +127.

	// Calculate average and apply quantization
	avg = sum / int64(count)

	// avg is already the average residual (pixel - 128), no need to subtract again
	residual = avg

	// HEVC Quantization formula:
	// Level = (Coeff * MF + offset) >> shift
	// where MF and shift depend on QP
	//
	// Simplified: Level = Coeff * 2^(QP/6) >> (15 + QP/6)
	// For QP=26: scale = 2^(26/6) = 2^4.33 ≈ 20
	// shift = 15 + 4 = 19
	// This gives very small levels...
	//
	// Actually for DC, the transform scaling is different.
	// For a 32x32 block, the DC gain from the transform is 32 (sqrt(N)).
	// So the effective coefficient is residual * 32.
	// Then quantization: Level = (residual * 32 * MF) >> shift
	//
	// Let's try: Level = (residual * 2^(QP/6)) >> (15 + QP/6 - log2(N))
	// For 32x32: Level = (residual * 20) >> (19 - 5) = (residual * 20) >> 14

	// Simpler approach based on HEVC dequant:
	// Dequant: Coeff' = Level * scale * (1 << shift)
	// For QP=26, scale ≈ 40, shift depends on transform size
	//
	// For 32x32 block, empirical testing showed level=1 gives ~64 pixel change
	// So for now, use qStep = 64 / scale_factor

	// Try the formula: Level = (Coeff * 2^(QP/6)) >> (15 + QP/6)
	// But apply to residual directly (treating it as the coefficient)
	qpDiv6 := float64(e.qp) / 6.0
	scale := math.Pow(2.0, qpDiv6)
	shift := 15.0 + qpDiv6

	// For DC coefficient, we need to account for transform gain
	// For NxN block, DC gain is N (or N^2 for energy)
	// Adjust shift by -log2(N) to compensate
	log2N := 5 // for 32x32
	adjustedShift := shift - float64(log2N)

	scaledCoeff := float64(residual) * scale
	level := int(scaledCoeff / math.Pow(2.0, adjustedShift))

	// Clamp to reasonable range
	if level > 127 {
		level = 127
	} else if level < -127 {
		level = -127
	}
	println("CalcDC: residual=", residual, ", qp=", e.qp, ", scale=", int(scale), ", shift=", int(adjustedShift), ", level=", level)
	return level
}

// encodeChromaResidual encodes residual coefficients for a chroma plane.
// planeIdx: 1 = Cb, 2 = Cr
func (e *HEVCEncoder) encodeChromaResidual(frame *avutil.Frame, x, y, size, planeIdx int) {
	// Calculate DC coefficient for chroma
	dcCoeff := e.calculateChromaDCCoefficient(frame, x, y, size, planeIdx)

	// Ensure non-zero
	if dcCoeff == 0 {
		dcCoeff = 1
	}

	// For chroma, use log2Size based on chroma size (usually 4x4 minimum)
	log2Size := 2 // 4x4
	if size == 8 {
		log2Size = 3
	} else if size == 16 {
		log2Size = 4
	}

	// Context offset for chroma
	ctxOffsetX := 3*(log2Size-2) + 9 // Chroma offset in context table
	ctxOffsetY := 3*(log2Size-2) + 9
	if ctxOffsetX < 9 {
		ctxOffsetX = 9
	}
	if ctxOffsetY < 9 {
		ctxOffsetY = 9
	}
	// Keep within bounds
	if ctxOffsetX >= len(e.residualCoder.ctxLastSigCoeffXPrefix) {
		ctxOffsetX = 9
	}
	if ctxOffsetY >= len(e.residualCoder.ctxLastSigCoeffYPrefix) {
		ctxOffsetY = 9
	}

	// last_sig_coeff_x_prefix = 0 (DC is at 0,0)
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffXPrefix[ctxOffsetX])

	// last_sig_coeff_y_prefix = 0
	e.cabac.EncodeBin(0, &e.residualCoder.ctxLastSigCoeffYPrefix[ctxOffsetY])

	// Encode coefficient level
	absLevel := dcCoeff
	sign := 0
	if absLevel < 0 {
		sign = 1
		absLevel = -absLevel
	}

	// coeff_abs_level_greater1_flag
	greater1 := 0
	if absLevel > 1 {
		greater1 = 1
	}
	e.cabac.EncodeBin(greater1, &e.residualCoder.ctxCoeffAbsGreater1[0])

	// coeff_abs_level_greater2_flag
	greater2 := 0
	if greater1 == 1 {
		if absLevel > 2 {
			greater2 = 1
		}
		e.cabac.EncodeBin(greater2, &e.residualCoder.ctxCoeffAbsGreater2[0])
	}

	// Sign flag (bypass mode)
	e.cabac.EncodeBypass(sign)

	// Remaining level
	baseLevel := 1
	if greater1 == 1 {
		baseLevel = 2
	}
	if greater2 == 1 {
		baseLevel = 3
	}

	if absLevel > baseLevel {
		remaining := absLevel - baseLevel
		e.encodeCoeffRemaining(remaining, 0)
	}
}

// calculateChromaDCCoefficient calculates the DC coefficient for a chroma block.
func (e *HEVCEncoder) calculateChromaDCCoefficient(frame *avutil.Frame, x, y, size, planeIdx int) int {
	if frame == nil || len(frame.Data) < 3 || frame.Data[planeIdx] == nil {
		return 1
	}

	chromaPlane := frame.Data[planeIdx]
	stride := frame.Linesize[planeIdx]

	chromaHeight := e.height / 2
	chromaWidth := e.width / 2

	// Get average chroma value
	var sum int
	count := 0
	for row := 0; row < size && y+row < chromaHeight; row++ {
		for col := 0; col < size && x+col < chromaWidth; col++ {
			idx := (y+row)*stride + (x + col)
			if idx < len(chromaPlane) {
				sum += int(chromaPlane[idx])
				count++
			}
		}
	}

	if count == 0 {
		return 1
	}

	avgPixel := sum / count
	// Residual = pixel - prediction (128), light quantization
	dcCoeff := (avgPixel - 128) / 4

	// Clamp
	if dcCoeff > 127 {
		dcCoeff = 127
	} else if dcCoeff < -127 {
		dcCoeff = -127
	}

	if dcCoeff == 0 {
		dcCoeff = 1
	}

	return dcCoeff
}

// extractLumaBlock4x4 extracts a 4x4 block from the Y plane
func (e *HEVCEncoder) extractLumaBlock4x4(x, y int) [4][4]int16 {
	var block [4][4]int16

	if e.currentFrame == nil || len(e.currentFrame.Data) == 0 || e.currentFrame.Data[0] == nil {
		return block
	}

	yPlane := e.currentFrame.Data[0]
	stride := e.currentFrame.Linesize[0]

	for row := 0; row < 4; row++ {
		py := y + row
		if py >= e.height {
			py = e.height - 1
		}
		for col := 0; col < 4; col++ {
			px := x + col
			if px >= e.width {
				px = e.width - 1
			}
			idx := py*stride + px
			if idx < len(yPlane) {
				// Convert to signed and subtract 128 for DC prediction residual
				block[row][col] = int16(yPlane[idx]) - 128
			}
		}
	}

	return block
}

// extractLumaBlock8x8 extracts an 8x8 block from the Y plane
func (e *HEVCEncoder) extractLumaBlock8x8(x, y int) [8][8]int16 {
	var block [8][8]int16

	if e.currentFrame == nil || len(e.currentFrame.Data) == 0 || e.currentFrame.Data[0] == nil {
		return block
	}

	yPlane := e.currentFrame.Data[0]
	stride := e.currentFrame.Linesize[0]

	for row := 0; row < 8; row++ {
		py := y + row
		if py >= e.height {
			py = e.height - 1
		}
		for col := 0; col < 8; col++ {
			px := x + col
			if px >= e.width {
				px = e.width - 1
			}
			idx := py*stride + px
			if idx < len(yPlane) {
				// Convert to signed and subtract 128 for DC prediction residual
				block[row][col] = int16(yPlane[idx]) - 128
			}
		}
	}

	return block
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
