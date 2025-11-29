package vulkango

// HEVCEncoder manages H.265/HEVC encoding
type HEVCEncoder struct {
	device         Device
	physicalDevice PhysicalDevice
	config         HEVCEncoderConfig

	// Frame tracking
	frameNum    uint32
	gopFrameNum uint32
	pocLsb      uint32

	// Parameter sets
	vps []byte
	sps []byte
	pps []byte

	// MP4 writer
	mp4Writer *MP4WriterHEVC
}

// NewHEVCEncoder creates a new HEVC encoder
func NewHEVCEncoder(device Device, physicalDevice PhysicalDevice, config HEVCEncoderConfig) (*HEVCEncoder, error) {
	enc := &HEVCEncoder{
		device:         device,
		physicalDevice: physicalDevice,
		config:         config,
	}

	// Generate VPS, SPS, PPS - simple single-layer for now
	enc.vps = GenerateSimpleHEVCVPS(config)
	enc.sps = GenerateSimpleHEVCSPS(config)
	enc.pps = GenerateSimpleHEVCPPS(config)

	// Initialize MP4 writer
	enc.mp4Writer = NewMP4WriterHEVC(config, enc.vps, enc.sps, enc.pps)

	return enc, nil
}

// EncodeFrame encodes a single RGBA frame
func (enc *HEVCEncoder) EncodeFrame(rgbaData []byte, width, height uint32) ([]byte, error) {
	isIDR := enc.frameNum == 0 || enc.gopFrameNum >= enc.config.GOPSize

	if isIDR {
		enc.gopFrameNum = 0
		enc.pocLsb = 0
	}

	encodedData := enc.encodeCPU(rgbaData, width, height, isIDR)
	enc.mp4Writer.AddFrame(encodedData, isIDR)

	enc.frameNum++
	enc.gopFrameNum++
	enc.pocLsb = (enc.pocLsb + 1) % 256

	return encodedData, nil
}

// encodeCPU performs software encoding using PCM coding units
func (enc *HEVCEncoder) encodeCPU(rgbaData []byte, width, height uint32, isIDR bool) []byte {
	var result []byte

	// For PCM encoding without reference picture management, use IDR for all frames
	// This makes every frame independently decodable
	isIDR = true

	// Generate slice with PCM CTUs
	sliceData := enc.generatePCMSlice(rgbaData, width, height, isIDR)

	// Always use IDR NAL type for simplicity
	nalType := uint8(HEVC_NAL_IDR_W_RADL)

	// Write NAL unit with 4-byte length prefix (AVCC format for MP4)
	sliceNAL := writeHEVCNALAVCC(nalType, sliceData)
	result = append(result, sliceNAL...)

	return result
}

// writeHEVCNALAVCC writes NAL with 4-byte length prefix
func writeHEVCNALAVCC(nalType uint8, rbsp []byte) []byte {
	nalSize := 2 + len(rbsp) // 2-byte NAL header + RBSP
	result := make([]byte, 4+nalSize)

	// 4-byte length prefix (big-endian)
	result[0] = byte(nalSize >> 24)
	result[1] = byte(nalSize >> 16)
	result[2] = byte(nalSize >> 8)
	result[3] = byte(nalSize)

	// NAL header (2 bytes)
	// Byte 0: forbidden_zero_bit(1) + nal_unit_type(6) + nuh_layer_id[5](1)
	// Byte 1: nuh_layer_id[4:0](5) + nuh_temporal_id_plus1(3)
	result[4] = (nalType & 0x3F) << 1 // type shifted left, layer_id MSB = 0
	result[5] = 0x01                  // layer_id = 0, temporal_id = 1

	copy(result[6:], rbsp)
	return result
}

// generatePCMSlice generates a slice using PCM coding units with proper quad-tree structure and CABAC
func (enc *HEVCEncoder) generatePCMSlice(rgbaData []byte, width, height uint32, isIDR bool) []byte {
	// For PCM-only HEVC, we use a simpler approach:
	// Since EVERY CU uses PCM, the CABAC bitstream is predictable.
	// We encode split flags and PCM flags with CABAC, then raw PCM samples.

	result := NewBitstreamWriter(int(width*height) * 3)

	// === Slice Segment Header (Exp-Golomb coded) ===

	// first_slice_segment_in_pic_flag
	result.WriteBit(1)

	// For IDR, no_output_of_prior_pics_flag
	if isIDR {
		result.WriteBit(0)
	}

	// slice_pic_parameter_set_id
	result.WriteUE(0)

	// slice_type (I = 2)
	result.WriteUE(2)

	// For non-IDR, pic_order_cnt_lsb
	if !isIDR {
		// pic_order_cnt_lsb (8 bits, since log2_max_pic_order_cnt_lsb_minus4 = 4)
		result.WriteBits(enc.pocLsb, 8)
		// short_term_ref_pic_set_sps_flag = 1
		result.WriteBit(1)
	}

	// slice_qp_delta
	result.WriteSE(0)

	// Byte-align before CABAC slice data
	result.ByteAlign()

	// === Slice Segment Data ===
	// For PCM mode, we need interleaved CABAC and raw PCM data.
	// Since this is complex, let's use a streaming approach with InterleavedCABACEncoder

	ctuSize := uint32(64)
	log2CtuSize := uint32(6)
	log2MinCuSize := uint32(3)

	ctuWidth := (width + ctuSize - 1) / ctuSize
	ctuHeight := (height + ctuSize - 1) / ctuSize

	// Use interleaved encoder that handles CABAC + PCM properly
	ice := NewInterleavedCABACEncoder()

	for ctuY := uint32(0); ctuY < ctuHeight; ctuY++ {
		for ctuX := uint32(0); ctuX < ctuWidth; ctuX++ {
			x0 := ctuX * ctuSize
			y0 := ctuY * ctuSize

			// Don't reset contexts at CTU boundaries - contexts persist across the slice
			// Only the arithmetic engine is reinitialized after PCM blocks (in ResumeAfterPCM)
			// ice.ResetContexts()

			enc.writeCodingQuadtreeInterleaved(ice, rgbaData, width, height, x0, y0, log2CtuSize, log2MinCuSize, 0)

			isLast := (ctuY == ctuHeight-1) && (ctuX == ctuWidth-1)
			if isLast {
				ice.EncodeTerminate(1)
			} else {
				ice.EncodeTerminate(0)
			}
		}
	}

	cabacData := ice.Finish()
	for _, b := range cabacData {
		result.WriteBits(uint32(b), 8)
	}

	return result.Data()
}

// InterleavedCABACEncoder handles CABAC encoding with interleaved PCM samples
type InterleavedCABACEncoder struct {
	cabac     *CABACEncoder
	output    []byte
	inPCMMode bool
}

// NewInterleavedCABACEncoder creates a new interleaved encoder
func NewInterleavedCABACEncoder() *InterleavedCABACEncoder {
	return &InterleavedCABACEncoder{
		cabac:     NewCABACEncoder(),
		output:    make([]byte, 0, 1024),
		inPCMMode: false,
	}
}

// EncodeSplitCuFlag encodes split_cu_flag
func (ice *InterleavedCABACEncoder) EncodeSplitCuFlag(split int, depth int) {
	ice.cabac.EncodeSplitCuFlag(split, depth)
}

// EncodePredModeFlag encodes pred_mode_flag
func (ice *InterleavedCABACEncoder) EncodePredModeFlag(intra int) {
	ice.cabac.EncodePredModeFlag(intra)
}

// EncodePCMFlag encodes pcm_flag and flushes for PCM data
func (ice *InterleavedCABACEncoder) EncodePCMFlag(pcm int) {
	ice.cabac.EncodePCMFlag(pcm)
	if pcm == 1 {
		// Flush CABAC arithmetic coder with alignment bits included
		// FlushForPCM() outputs CABAC data + pcm_alignment_one_bit + zeros
		cabacBytes := ice.cabac.FlushForPCM()
		ice.output = append(ice.output, cabacBytes...)

		ice.inPCMMode = true
	}
}

// WritePCMSample writes a raw PCM sample byte
func (ice *InterleavedCABACEncoder) WritePCMSample(sample uint8) {
	ice.output = append(ice.output, sample)
}

// EndPCMSamples signals end of PCM samples, resumes CABAC
func (ice *InterleavedCABACEncoder) EndPCMSamples() {
	if ice.inPCMMode {
		ice.cabac.ResumeAfterPCM()
		ice.inPCMMode = false
	}
}

// ResetContexts reinitializes CABAC contexts for CTU boundary
func (ice *InterleavedCABACEncoder) ResetContexts() {
	ice.cabac.initContexts()
}

// EncodeIntraDCMode encodes an intra CU using DC prediction with no residual
// This is much simpler than PCM and produces smaller output
func (ice *InterleavedCABACEncoder) EncodeIntraDCMode() {
	// pcm_flag = 0 (not using PCM mode)
	ice.cabac.EncodeTerminate(0)

	// prev_intra_luma_pred_flag = 1 (DC is in MPM list)
	// DC mode (1) is always MPM[1] when no neighbors available
	ice.cabac.EncodePrevIntraLumaPredFlag(1)

	// mpm_idx = 1 (select DC which is typically MPM[1])
	// For first CU with no neighbors: MPM = {0, 1, 26} (Planar, DC, Angular26)
	ice.cabac.EncodeMPMIdx(1)

	// intra_chroma_pred_mode = 4 (derived from luma - DM_CHROMA)
	ice.cabac.EncodeIntraChromaPredMode(4)

	// rqt_root_cbf = 0 (no residual)
	ice.cabac.EncodeRqtRootCbf(0)
}

// EncodeTerminate encodes end_of_slice_segment_flag
func (ice *InterleavedCABACEncoder) EncodeTerminate(bin int) {
	ice.cabac.EncodeTerminate(bin)
}

// Finish finalizes and returns all encoded data
func (ice *InterleavedCABACEncoder) Finish() []byte {
	// Finalize CABAC
	finalBytes := ice.cabac.Finish()
	ice.output = append(ice.output, finalBytes...)
	return ice.output
}

// writeCodingQuadtreeInterleaved recursively writes CTU structure with interleaved CABAC+PCM
func (enc *HEVCEncoder) writeCodingQuadtreeInterleaved(ice *InterleavedCABACEncoder, rgbaData []byte, imgWidth, imgHeight, x0, y0, log2CbSize, log2MinCbSize uint32, depth int) {
	cbSize := uint32(1) << log2CbSize

	// Skip if completely outside image
	if x0 >= imgWidth || y0 >= imgHeight {
		return
	}

	// Check if CU fits within picture (HEVC spec 7.3.8.4)
	cuFitsInPicture := (x0+cbSize <= imgWidth) && (y0+cbSize <= imgHeight)

	if log2CbSize > log2MinCbSize {
		// split_cu_flag is only encoded if CU fits within picture AND not at min size
		// If CU doesn't fit, split is INFERRED to be 1 (no encoding)
		if cuFitsInPicture {
			ice.EncodeSplitCuFlag(1, depth)
		}
		// If !cuFitsInPicture, split_cu_flag is inferred to be 1, no encoding needed

		halfSize := cbSize / 2
		log2Half := log2CbSize - 1

		enc.writeCodingQuadtreeInterleaved(ice, rgbaData, imgWidth, imgHeight, x0, y0, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeInterleaved(ice, rgbaData, imgWidth, imgHeight, x0+halfSize, y0, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeInterleaved(ice, rgbaData, imgWidth, imgHeight, x0, y0+halfSize, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeInterleaved(ice, rgbaData, imgWidth, imgHeight, x0+halfSize, y0+halfSize, log2Half, log2MinCbSize, depth+1)
	} else {
		// Use Intra DC mode with no residual (simpler than PCM)
		enc.writeIntraDCCodingUnit(ice)
	}
}

// writeIntraDCCodingUnit writes a CU using Intra DC prediction with zero residual
// This produces a gray/DC value output which is then predicted by the decoder
func (enc *HEVCEncoder) writeIntraDCCodingUnit(ice *InterleavedCABACEncoder) {
	ice.EncodeIntraDCMode()
}

// writePCMCodingUnitInterleaved writes a CU with PCM using interleaved encoder
func (enc *HEVCEncoder) writePCMCodingUnitInterleaved(ice *InterleavedCABACEncoder, rgbaData []byte, imgWidth, imgHeight, x, y, size uint32) {
	// NOTE: pred_mode_flag is NOT present for I-slices (HEVC spec 7.3.8.5)
	// It's inferred to be MODE_INTRA for I-slices

	// pcm_flag = 1 (this also flushes CABAC for PCM samples)
	ice.EncodePCMFlag(1)

	// Write raw PCM luma samples
	for py := uint32(0); py < size; py++ {
		for px := uint32(0); px < size; px++ {
			imgX := x + px
			imgY := y + py
			var luma uint8 = 128
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					luma = uint8(clamp(16.0+65.481*r/255.0+128.553*g/255.0+24.966*b/255.0, 16, 235))
				}
			}
			ice.WritePCMSample(luma)
		}
	}

	// Write PCM chroma samples (4:2:0)
	chromaSize := size / 2

	// Cb
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cb uint8 = 128
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cb = uint8(clamp(128.0-37.797*r/255.0-74.203*g/255.0+112.0*b/255.0, 16, 240))
				}
			}
			ice.WritePCMSample(cb)
		}
	}

	// Cr
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cr uint8 = 128
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cr = uint8(clamp(128.0+112.0*r/255.0-93.786*g/255.0-18.214*b/255.0, 16, 240))
				}
			}
			ice.WritePCMSample(cr)
		}
	}

	// Resume CABAC after PCM samples
	ice.EndPCMSamples()
}

// writeCodingQuadtreeCABAC recursively writes the quad-tree CTU structure using CABAC
func (enc *HEVCEncoder) writeCodingQuadtreeCABAC(cabac *CABACEncoder, pcmData *BitstreamWriter, rgbaData []byte, imgWidth, imgHeight, x0, y0, log2CbSize, log2MinCbSize uint32, depth int) {
	cbSize := uint32(1) << log2CbSize

	// Check if this CU is completely outside the image
	if x0 >= imgWidth || y0 >= imgHeight {
		return
	}

	// Check if we can split (CU is larger than minimum)
	if log2CbSize > log2MinCbSize {
		// split_cu_flag = 1 (always split until minimum size for PCM)
		cabac.EncodeSplitCuFlag(1, depth)

		halfSize := cbSize / 2
		log2Half := log2CbSize - 1

		// Recursively process four quadrants
		enc.writeCodingQuadtreeCABAC(cabac, pcmData, rgbaData, imgWidth, imgHeight, x0, y0, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeCABAC(cabac, pcmData, rgbaData, imgWidth, imgHeight, x0+halfSize, y0, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeCABAC(cabac, pcmData, rgbaData, imgWidth, imgHeight, x0, y0+halfSize, log2Half, log2MinCbSize, depth+1)
		enc.writeCodingQuadtreeCABAC(cabac, pcmData, rgbaData, imgWidth, imgHeight, x0+halfSize, y0+halfSize, log2Half, log2MinCbSize, depth+1)
	} else {
		// At minimum CU size - write the coding unit with PCM
		enc.writePCMCodingUnitCABAC(cabac, pcmData, rgbaData, imgWidth, imgHeight, x0, y0, cbSize)
	}
}

// writePCMCodingUnitCABAC writes a single CU using PCM mode with CABAC
func (enc *HEVCEncoder) writePCMCodingUnitCABAC(cabac *CABACEncoder, pcmData *BitstreamWriter, rgbaData []byte, imgWidth, imgHeight, x, y, size uint32) {
	// For I-slice CU:
	// cu_transquant_bypass_flag - not present (transquant_bypass_enabled_flag = 0 in PPS)
	// cu_skip_flag - not present for I-slice
	// pred_mode_flag - NOT present for I-slice (HEVC spec 7.3.8.5), inferred MODE_INTRA
	// part_mode - inferred to be PART_2Nx2N for intra at min CU size (no bits)

	// pcm_flag = 1
	cabac.EncodePCMFlag(1)

	// After pcm_flag=1, PCM samples are written as raw bytes
	// The PCM samples are collected separately and appended after CABAC

	// Write PCM luma samples (size x size)
	for py := uint32(0); py < size; py++ {
		for px := uint32(0); px < size; px++ {
			imgX := x + px
			imgY := y + py
			var luma uint8 = 128 // Default gray for out-of-bounds
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					// BT.601 RGB to Y conversion
					luma = uint8(clamp(16.0+65.481*r/255.0+128.553*g/255.0+24.966*b/255.0, 16, 235))
				}
			}
			pcmData.WriteBits(uint32(luma), 8)
		}
	}

	// Write PCM chroma samples (4:2:0: size/2 x size/2 for Cb and Cr)
	chromaSize := size / 2

	// Cb samples
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cb uint8 = 128 // Neutral chroma
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cb = uint8(clamp(128.0-37.797*r/255.0-74.203*g/255.0+112.0*b/255.0, 16, 240))
				}
			}
			pcmData.WriteBits(uint32(cb), 8)
		}
	}

	// Cr samples
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cr uint8 = 128 // Neutral chroma
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cr = uint8(clamp(128.0+112.0*r/255.0-93.786*g/255.0-18.214*b/255.0, 16, 240))
				}
			}
			pcmData.WriteBits(uint32(cr), 8)
		}
	}
}

// writeCodingQuadtree recursively writes the quad-tree CTU structure (non-CABAC, kept for reference)
func (enc *HEVCEncoder) writeCodingQuadtree(bw *BitstreamWriter, rgbaData []byte, imgWidth, imgHeight, x0, y0, log2CbSize, log2MinCbSize uint32) {
	cbSize := uint32(1) << log2CbSize

	// Check if this CU is completely outside the image
	if x0 >= imgWidth || y0 >= imgHeight {
		return
	}

	// Check if we can split (CU is larger than minimum)
	if log2CbSize > log2MinCbSize {
		// split_cu_flag = 1 (always split until minimum size for PCM)
		bw.WriteBit(1)

		halfSize := cbSize / 2
		log2Half := log2CbSize - 1

		// Recursively process four quadrants
		enc.writeCodingQuadtree(bw, rgbaData, imgWidth, imgHeight, x0, y0, log2Half, log2MinCbSize)
		enc.writeCodingQuadtree(bw, rgbaData, imgWidth, imgHeight, x0+halfSize, y0, log2Half, log2MinCbSize)
		enc.writeCodingQuadtree(bw, rgbaData, imgWidth, imgHeight, x0, y0+halfSize, log2Half, log2MinCbSize)
		enc.writeCodingQuadtree(bw, rgbaData, imgWidth, imgHeight, x0+halfSize, y0+halfSize, log2Half, log2MinCbSize)
	} else {
		// At minimum CU size - write the coding unit with PCM
		enc.writePCMCodingUnit(bw, rgbaData, imgWidth, imgHeight, x0, y0, cbSize)
	}
}

// writePCMCodingUnit writes a single CU using PCM mode
func (enc *HEVCEncoder) writePCMCodingUnit(bw *BitstreamWriter, rgbaData []byte, imgWidth, imgHeight, x, y, size uint32) {
	// For I-slice CU:
	// cu_transquant_bypass_flag - not present (transquant_bypass_enabled_flag = 0 in PPS)
	// cu_skip_flag - not present for I-slice
	// pred_mode_flag - NOT present for I-slice (HEVC spec 7.3.8.5), inferred MODE_INTRA
	// part_mode - inferred to be PART_2Nx2N for intra at min CU size (no bits)

	// pcm_flag = 1
	bw.WriteBit(1)

	// Byte-align before PCM samples (pcm_alignment_zero_bit)
	bw.ByteAlign()

	// Write PCM luma samples (size x size)
	for py := uint32(0); py < size; py++ {
		for px := uint32(0); px < size; px++ {
			imgX := x + px
			imgY := y + py
			var luma uint8 = 128 // Default gray for out-of-bounds
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					// BT.601 RGB to Y conversion
					luma = uint8(clamp(16.0+65.481*r/255.0+128.553*g/255.0+24.966*b/255.0, 16, 235))
				}
			}
			bw.WriteBits(uint32(luma), 8)
		}
	}

	// Write PCM chroma samples (4:2:0: size/2 x size/2 for Cb and Cr)
	chromaSize := size / 2

	// Cb samples
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cb uint8 = 128 // Neutral chroma
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cb = uint8(clamp(128.0-37.797*r/255.0-74.203*g/255.0+112.0*b/255.0, 16, 240))
				}
			}
			bw.WriteBits(uint32(cb), 8)
		}
	}

	// Cr samples
	for py := uint32(0); py < chromaSize; py++ {
		for px := uint32(0); px < chromaSize; px++ {
			imgX := x + px*2
			imgY := y + py*2
			var cr uint8 = 128 // Neutral chroma
			if imgX < imgWidth && imgY < imgHeight {
				idx := (imgY*imgWidth + imgX) * 4
				if int(idx+2) < len(rgbaData) {
					r := float32(rgbaData[idx])
					g := float32(rgbaData[idx+1])
					b := float32(rgbaData[idx+2])
					cr = uint8(clamp(128.0+112.0*r/255.0-93.786*g/255.0-18.214*b/255.0, 16, 240))
				}
			}
			bw.WriteBits(uint32(cr), 8)
		}
	}
}

// WriteToFile writes the encoded video to a file
func (enc *HEVCEncoder) WriteToFile(path string) error {
	return enc.mp4Writer.WriteToFile(path)
}

// Destroy cleans up encoder resources
func (enc *HEVCEncoder) Destroy() {
	// Nothing to destroy for CPU encoder
}

// GenerateSimpleHEVCVPS generates a minimal VPS
func GenerateSimpleHEVCVPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(128)

	// vps_video_parameter_set_id (4 bits)
	bw.WriteBits(0, 4)

	// vps_base_layer_internal_flag, vps_base_layer_available_flag
	bw.WriteBit(1)
	bw.WriteBit(1)

	// vps_max_layers_minus1 (6 bits) = 0
	bw.WriteBits(0, 6)

	// vps_max_sub_layers_minus1 (3 bits) = 0
	bw.WriteBits(0, 3)

	// vps_temporal_id_nesting_flag = 1
	bw.WriteBit(1)

	// vps_reserved_0xffff_16bits
	bw.WriteBits(0xFFFF, 16)

	// profile_tier_level(1, vps_max_sub_layers_minus1=0)
	writeSimplePTL(bw, config)

	// vps_sub_layer_ordering_info_present_flag = 0
	bw.WriteBit(0)

	// vps_max_dec_pic_buffering_minus1[0], vps_max_num_reorder_pics[0], vps_max_latency_increase_plus1[0]
	bw.WriteUE(4)
	bw.WriteUE(0)
	bw.WriteUE(0)

	// vps_max_layer_id (6 bits) = 0
	bw.WriteBits(0, 6)

	// vps_num_layer_sets_minus1 = 0
	bw.WriteUE(0)

	// vps_timing_info_present_flag = 0
	bw.WriteBit(0)

	// vps_extension_flag = 0
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// GenerateSimpleHEVCSPS generates a minimal SPS
func GenerateSimpleHEVCSPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(256)

	// sps_video_parameter_set_id (4 bits)
	bw.WriteBits(0, 4)

	// sps_max_sub_layers_minus1 (3 bits) = 0
	bw.WriteBits(0, 3)

	// sps_temporal_id_nesting_flag = 1
	bw.WriteBit(1)

	// profile_tier_level
	writeSimplePTL(bw, config)

	// sps_seq_parameter_set_id
	bw.WriteUE(0)

	// chroma_format_idc = 1 (4:2:0)
	bw.WriteUE(1)

	// pic_width_in_luma_samples
	bw.WriteUE(config.Width)

	// pic_height_in_luma_samples
	bw.WriteUE(config.Height)

	// conformance_window_flag = 0
	bw.WriteBit(0)

	// bit_depth_luma_minus8 = 0
	bw.WriteUE(0)

	// bit_depth_chroma_minus8 = 0
	bw.WriteUE(0)

	// log2_max_pic_order_cnt_lsb_minus4 = 4 (8 bits for POC LSB)
	bw.WriteUE(4)

	// sps_sub_layer_ordering_info_present_flag = 0
	bw.WriteBit(0)

	// sps_max_dec_pic_buffering_minus1[0]
	bw.WriteUE(4)
	// sps_max_num_reorder_pics[0]
	bw.WriteUE(0)
	// sps_max_latency_increase_plus1[0]
	bw.WriteUE(0)

	// log2_min_luma_coding_block_size_minus3 = 0 (min CB = 8)
	bw.WriteUE(0)

	// log2_diff_max_min_luma_coding_block_size = 3 (max CB = 64)
	bw.WriteUE(3)

	// log2_min_luma_transform_block_size_minus2 = 0 (min TB = 4)
	bw.WriteUE(0)

	// log2_diff_max_min_luma_transform_block_size = 3 (max TB = 32)
	bw.WriteUE(3)

	// max_transform_hierarchy_depth_inter = 0
	bw.WriteUE(0)

	// max_transform_hierarchy_depth_intra = 0
	bw.WriteUE(0)

	// scaling_list_enabled_flag = 0
	bw.WriteBit(0)

	// amp_enabled_flag = 0
	bw.WriteBit(0)

	// sample_adaptive_offset_enabled_flag = 0
	bw.WriteBit(0)

	// pcm_enabled_flag = 1 (we use PCM)
	bw.WriteBit(1)

	// pcm_sample_bit_depth_luma_minus1 (4 bits) = 7
	bw.WriteBits(7, 4)
	// pcm_sample_bit_depth_chroma_minus1 (4 bits) = 7
	bw.WriteBits(7, 4)
	// log2_min_pcm_luma_coding_block_size_minus3 = 0 (min PCM = 8)
	bw.WriteUE(0)
	// log2_diff_max_min_pcm_luma_coding_block_size = 2 (max PCM = 32)
	bw.WriteUE(2)
	// pcm_loop_filter_disabled_flag = 1
	bw.WriteBit(1)

	// num_short_term_ref_pic_sets = 0
	bw.WriteUE(0)

	// long_term_ref_pics_present_flag = 0
	bw.WriteBit(0)

	// sps_temporal_mvp_enabled_flag = 0
	bw.WriteBit(0)

	// strong_intra_smoothing_enabled_flag = 0
	bw.WriteBit(0)

	// vui_parameters_present_flag = 0
	bw.WriteBit(0)

	// sps_extension_present_flag = 0
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// GenerateSimpleHEVCPPS generates a minimal PPS
func GenerateSimpleHEVCPPS(config HEVCEncoderConfig) []byte {
	bw := NewBitstreamWriter(64)

	// pps_pic_parameter_set_id
	bw.WriteUE(0)

	// pps_seq_parameter_set_id
	bw.WriteUE(0)

	// dependent_slice_segments_enabled_flag = 0
	bw.WriteBit(0)

	// output_flag_present_flag = 0
	bw.WriteBit(0)

	// num_extra_slice_header_bits (3 bits) = 0
	bw.WriteBits(0, 3)

	// sign_data_hiding_enabled_flag = 0
	bw.WriteBit(0)

	// cabac_init_present_flag = 0
	bw.WriteBit(0)

	// num_ref_idx_l0_default_active_minus1 = 0
	bw.WriteUE(0)

	// num_ref_idx_l1_default_active_minus1 = 0
	bw.WriteUE(0)

	// init_qp_minus26 = 0
	bw.WriteSE(0)

	// constrained_intra_pred_flag = 0
	bw.WriteBit(0)

	// transform_skip_enabled_flag = 0
	bw.WriteBit(0)

	// cu_qp_delta_enabled_flag = 0
	bw.WriteBit(0)

	// pps_cb_qp_offset = 0
	bw.WriteSE(0)

	// pps_cr_qp_offset = 0
	bw.WriteSE(0)

	// pps_slice_chroma_qp_offsets_present_flag = 0
	bw.WriteBit(0)

	// weighted_pred_flag = 0
	bw.WriteBit(0)

	// weighted_bipred_flag = 0
	bw.WriteBit(0)

	// transquant_bypass_enabled_flag = 0
	bw.WriteBit(0)

	// tiles_enabled_flag = 0
	bw.WriteBit(0)

	// entropy_coding_sync_enabled_flag = 0
	bw.WriteBit(0)

	// pps_loop_filter_across_slices_enabled_flag = 0
	bw.WriteBit(0)

	// deblocking_filter_control_present_flag = 0
	bw.WriteBit(0)

	// pps_scaling_list_data_present_flag = 0
	bw.WriteBit(0)

	// lists_modification_present_flag = 0
	bw.WriteBit(0)

	// log2_parallel_merge_level_minus2 = 0
	bw.WriteUE(0)

	// slice_segment_header_extension_present_flag = 0
	bw.WriteBit(0)

	// pps_extension_present_flag = 0
	bw.WriteBit(0)

	bw.ByteAlign()
	return bw.Data()
}

// writeSimplePTL writes profile_tier_level based on config
func writeSimplePTL(bw *BitstreamWriter, config HEVCEncoderConfig) {
	// general_profile_space (2 bits) = 0
	bw.WriteBits(0, 2)

	// general_tier_flag = 0
	bw.WriteBit(0)

	// general_profile_idc (5 bits)
	bw.WriteBits(config.Profile, 5)

	// general_profile_compatibility_flag[32]
	// Write as individual bits - set the bit corresponding to the profile
	for i := 0; i < 32; i++ {
		if uint32(i) == config.Profile {
			bw.WriteBit(1)
		} else {
			bw.WriteBit(0)
		}
	}

	// general_progressive_source_flag = 1
	bw.WriteBit(1)

	// general_interlaced_source_flag = 0
	bw.WriteBit(0)

	// general_non_packed_constraint_flag = 0
	bw.WriteBit(0)

	// general_frame_only_constraint_flag = 1
	bw.WriteBit(1)

	// Constraint flags depend on profile
	if config.Profile == HEVC_PROFILE_REXT {
		// Range extension constraint flags (9 bits) + 35 reserved zeros
		bw.WriteBit(0) // general_max_12bit_constraint_flag
		bw.WriteBit(0) // general_max_10bit_constraint_flag
		bw.WriteBit(1) // general_max_8bit_constraint_flag
		bw.WriteBit(0) // general_max_422chroma_constraint_flag
		bw.WriteBit(0) // general_max_420chroma_constraint_flag
		bw.WriteBit(0) // general_max_monochrome_constraint_flag
		bw.WriteBit(0) // general_intra_constraint_flag
		bw.WriteBit(0) // general_one_picture_only_constraint_flag
		bw.WriteBit(0) // general_lower_bit_rate_constraint_flag
		// 35 reserved zero bits
		bw.WriteBits(0, 32)
		bw.WriteBits(0, 3)
	} else {
		// 44 reserved zero bits for Main profile
		bw.WriteBits(0, 32)
		bw.WriteBits(0, 12)
	}

	// general_level_idc (8 bits)
	bw.WriteBits(config.Level, 8)

	// No sub-layer info since max_sub_layers_minus1 = 0
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
