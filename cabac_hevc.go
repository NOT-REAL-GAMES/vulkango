package vulkango

// HEVC CABAC Encoder
// Implements Context-Adaptive Binary Arithmetic Coding for HEVC
// Based on HM (HEVC Test Model) reference implementation

// CABAC probability state transition tables (from HEVC spec Table 9-45)
var transIdxMPS = [64]uint8{
	1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 62, 63,
}

var transIdxLPS = [64]uint8{
	0, 0, 1, 2, 2, 4, 4, 5, 6, 7, 8, 9, 9, 11, 11, 12,
	13, 13, 15, 15, 16, 16, 18, 18, 19, 19, 21, 21, 22, 22, 23, 24,
	24, 25, 26, 26, 27, 27, 28, 29, 29, 30, 30, 30, 31, 32, 32, 33,
	33, 33, 34, 34, 35, 35, 35, 36, 36, 36, 37, 37, 37, 38, 38, 63,
}

// Renormalization table - determines number of bits to shift based on LPS range
// From HM reference: sm_aucRenormTable indexed by (uiLPS >> 3)
var renormTable = [32]uint8{
	6, 5, 4, 4, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
}

// LPS range table (from HEVC spec Table 9-46)
// Indexed by [state][((range >> 6) & 3)]
var rangeTabLPS = [64][4]uint16{
	{128, 176, 208, 240}, {128, 167, 197, 227}, {128, 158, 187, 216}, {123, 150, 178, 205},
	{116, 142, 169, 195}, {111, 135, 160, 185}, {105, 128, 152, 175}, {100, 122, 144, 166},
	{95, 116, 137, 158}, {90, 110, 130, 150}, {85, 104, 123, 142}, {81, 99, 117, 135},
	{77, 94, 111, 128}, {73, 89, 105, 122}, {69, 85, 100, 116}, {66, 80, 95, 110},
	{62, 76, 90, 104}, {59, 72, 86, 99}, {56, 69, 81, 94}, {53, 65, 77, 89},
	{51, 62, 73, 85}, {48, 59, 69, 80}, {46, 56, 66, 76}, {43, 53, 63, 72},
	{41, 50, 59, 69}, {39, 48, 56, 65}, {37, 45, 54, 62}, {35, 43, 51, 59},
	{33, 41, 48, 56}, {32, 39, 46, 53}, {30, 37, 43, 50}, {29, 35, 41, 48},
	{27, 33, 39, 45}, {26, 31, 37, 43}, {24, 30, 35, 41}, {23, 28, 33, 39},
	{22, 27, 32, 37}, {21, 26, 30, 35}, {20, 24, 29, 33}, {19, 23, 27, 31},
	{18, 22, 26, 30}, {17, 21, 25, 28}, {16, 20, 23, 27}, {15, 19, 22, 25},
	{14, 18, 21, 24}, {14, 17, 20, 23}, {13, 16, 19, 22}, {12, 15, 18, 21},
	{12, 14, 17, 20}, {11, 14, 16, 19}, {11, 13, 15, 18}, {10, 12, 15, 17},
	{10, 12, 14, 16}, {9, 11, 13, 15}, {9, 11, 12, 14}, {8, 10, 12, 14},
	{8, 9, 11, 13}, {7, 9, 11, 12}, {7, 9, 10, 12}, {7, 8, 10, 11},
	{6, 8, 9, 11}, {6, 7, 9, 10}, {6, 7, 8, 9}, {2, 2, 2, 2},
}

// HEVC context indices
const (
	HEVC_CTX_SPLIT_CU_FLAG    = 0  // 3 contexts (0-2)
	HEVC_CTX_CU_SKIP_FLAG     = 3  // 3 contexts (3-5)
	HEVC_CTX_PRED_MODE_FLAG   = 6  // 1 context
	HEVC_CTX_PART_MODE        = 7  // 4 contexts (7-10)
	HEVC_CTX_PCM_FLAG         = 11 // 1 context
	HEVC_CTX_PREV_INTRA_LUMA  = 12 // 1 context
	HEVC_CTX_INTRA_CHROMA     = 13 // 1 context
	HEVC_CTX_RQT_ROOT_CBF     = 14 // 1 context (for rqt_root_cbf)
	// ... more contexts as needed
	HEVC_NUM_CONTEXTS = 64 // Simplified - full HEVC has ~154
)

// CABACEncoder implements CABAC encoding for HEVC
type CABACEncoder struct {
	low           uint32
	range_        uint32
	bufferedByte  int
	numBuffered   int
	bitsLeft      int
	buffer        []byte
	contexts      [HEVC_NUM_CONTEXTS]uint8 // state << 1 | mps
}

// NewCABACEncoder creates a new CABAC encoder
func NewCABACEncoder() *CABACEncoder {
	enc := &CABACEncoder{
		low:          0,
		range_:       510,
		bufferedByte: -1,
		numBuffered:  0,
		bitsLeft:     23,
		buffer:       make([]byte, 0, 1024),
	}
	enc.initContexts()
	return enc
}

// initContexts initializes context models for I-slice
// Based on HEVC spec 9.3.2.2 initialization process
func (c *CABACEncoder) initContexts() {
	// For I-slice with QP=26 (init_qp_minus26=0, slice_qp_delta=0)
	// SliceQpY = 26
	// Formula: preCtxState = Clip3(1, 126, ((initValue>>4)*5 - 45 + (SliceQpY+26))>>1)
	// Then: state = (valMPS == 1) ? preCtxState-1 : 126-preCtxState

	// split_cu_flag contexts (0-2)
	// initValues from HM: 139, 141, 157
	// For initValue=139, QP=26: preCtxState=((8)*5-45+52)>>1 = 47>>1 = 23, MPS=1, state=22
	// For initValue=141, QP=26: preCtxState=((8)*5-45+52)>>1 = 47>>1 = 23, MPS=1, state=22
	// For initValue=157, QP=26: preCtxState=((9)*5-45+52)>>1 = 52>>1 = 26, MPS=1, state=25
	c.contexts[0] = (22 << 1) | 1 // ctxIdx 0: depth 0
	c.contexts[1] = (22 << 1) | 1 // ctxIdx 1: depth 1
	c.contexts[2] = (25 << 1) | 1 // ctxIdx 2: depth 2+

	// pred_mode_flag context (6): For I-slice, this isn't used (inferred MODE_INTRA)
	// initValue = 149 for I-slice
	c.contexts[HEVC_CTX_PRED_MODE_FLAG] = (26 << 1) | 1

	// pcm_flag uses terminating bin, not context-based, so this isn't used
	c.contexts[HEVC_CTX_PCM_FLAG] = (10 << 1) | 0

	// prev_intra_luma_pred_flag context (12)
	// initValue = 184 for I-slice
	// preCtxState = ((11)*5-45+52)>>1 = 62>>1 = 31, MPS=1, state=30
	c.contexts[HEVC_CTX_PREV_INTRA_LUMA] = (30 << 1) | 1

	// intra_chroma_pred_mode context (13)
	// initValue = 63 for I-slice (from HM INTRA_CHROMA_PRED_MODE)
	// preCtxState = ((3)*5-45+52)>>1 = 22>>1 = 11, MPS=0, state=63-11=52
	c.contexts[HEVC_CTX_INTRA_CHROMA] = (11 << 1) | 0

	// rqt_root_cbf context (14)
	// initValue = 79 for I-slice (from HM CU_QT_ROOT_CBF)
	// preCtxState = ((4)*5-45+52)>>1 = 27>>1 = 13, MPS=0, state=126-13=113 (clamped)
	// Actually for I-slice initValue=79: slope=4, offset=15
	// preCtxState = (4*5-45+(26+26))>>1 = (20-45+52)>>1 = 27>>1 = 13
	// valMPS = (13 >= 64) ? 1 : 0 = 0, state = 63-13 = 50
	c.contexts[HEVC_CTX_RQT_ROOT_CBF] = (13 << 1) | 0
}

// Reset resets the encoder for a new slice
func (c *CABACEncoder) Reset() {
	c.low = 0
	c.range_ = 510
	c.bufferedByte = -1
	c.numBuffered = 0
	c.bitsLeft = 23
	c.buffer = c.buffer[:0]
	c.initContexts()
}

// EncodeBin encodes a single bin with context
// Based on HM reference: TEncBinCABAC::encodeBin
func (c *CABACEncoder) EncodeBin(bin int, ctxIdx int) {
	state := c.contexts[ctxIdx] >> 1
	mps := int(c.contexts[ctxIdx] & 1)

	// Get LPS range from table (same as HM sm_aucLPSTable)
	qIdx := (c.range_ >> 6) & 3
	lpsRange := uint32(rangeTabLPS[state][qIdx])
	c.range_ -= lpsRange

	if bin != mps {
		// LPS (Least Probable Symbol) - matches HM exactly
		numBits := int(renormTable[lpsRange>>3])
		c.low = (c.low + c.range_) << uint(numBits)
		c.range_ = lpsRange << uint(numBits)

		// State transition for LPS
		if state == 0 {
			c.contexts[ctxIdx] ^= 1 // Toggle MPS
		}
		c.contexts[ctxIdx] = (transIdxLPS[state] << 1) | (c.contexts[ctxIdx] & 1)

		c.bitsLeft -= numBits
		c.testAndWriteOut()
	} else {
		// MPS (Most Probable Symbol) - matches HM exactly
		c.contexts[ctxIdx] = (transIdxMPS[state] << 1) | uint8(mps)

		if c.range_ < 256 {
			c.low <<= 1
			c.range_ <<= 1
			c.bitsLeft--
			c.testAndWriteOut()
		}
	}
}

// testAndWriteOut checks if we need to output bytes
// Matches HM: TEncBinCABAC::testAndWriteOut
func (c *CABACEncoder) testAndWriteOut() {
	if c.bitsLeft < 12 {
		c.writeOut()
	}
}

// EncodeBypass encodes a bin in bypass mode (equiprobable)
// Matches HM: TEncBinCABAC::encodeBinEP
func (c *CABACEncoder) EncodeBypass(bin int) {
	c.low <<= 1
	if bin != 0 {
		c.low += c.range_
	}
	c.bitsLeft--
	c.testAndWriteOut()
}

// EncodeTerminate encodes the terminating bin
// Matches HM: TEncBinCABAC::encodeBinTrm
func (c *CABACEncoder) EncodeTerminate(bin int) {
	c.range_ -= 2

	if bin != 0 {
		// Terminating with 1 - end of slice
		c.low += c.range_
		c.low <<= 7
		c.range_ = 2 << 7
		c.bitsLeft -= 7
		c.testAndWriteOut()
	} else {
		// Terminating with 0 - continue
		if c.range_ < 256 {
			c.low <<= 1
			c.range_ <<= 1
			c.bitsLeft--
			c.testAndWriteOut()
		}
	}
}

// renormalize renormalizes after encoding
func (c *CABACEncoder) renormalize() {
	for c.range_ < 256 {
		c.bitsLeft--
		c.range_ <<= 1
		c.low <<= 1

		if c.bitsLeft < 12 {
			c.writeOut()
		}
	}
}

// writeOut outputs buffered bytes
// Matches HM: TEncBinCABAC::writeOut exactly
func (c *CABACEncoder) writeOut() {
	leadByte := c.low >> uint(24-c.bitsLeft)
	c.bitsLeft += 8
	c.low &= 0xFFFFFFFF >> uint(c.bitsLeft) // Critical fix: was wrong mask calculation

	if leadByte == 0xFF {
		c.numBuffered++
	} else {
		if c.numBuffered > 0 {
			// We have buffered bytes to output
			carry := leadByte >> 8
			byteVal := byte(uint32(c.bufferedByte) + carry)
			c.bufferedByte = int(leadByte & 0xFF)
			c.buffer = append(c.buffer, byteVal)

			byteVal = byte((0xFF + carry) & 0xFF)
			for c.numBuffered > 1 {
				c.buffer = append(c.buffer, byteVal)
				c.numBuffered--
			}
			c.numBuffered = 0
		} else {
			c.numBuffered = 1
			c.bufferedByte = int(leadByte)
		}
	}
}

// Finish finalizes the CABAC encoding
// Based on HM: TEncBinCABAC::finish
func (c *CABACEncoder) Finish() []byte {
	// The terminating bin should already be encoded by the caller
	// Now flush the remaining bits

	// Flush remaining bits from low register
	if c.low > 0 || c.numBuffered > 0 || c.bufferedByte >= 0 {
		// Output any remaining buffered bytes
		if c.numBuffered > 0 || c.bufferedByte >= 0 {
			// Force output of remaining state
			c.low <<= uint(c.bitsLeft)
			c.writeOut()
			c.low <<= uint(c.bitsLeft)
			c.writeOut()
		}

		// Output the buffered byte
		if c.bufferedByte >= 0 {
			c.buffer = append(c.buffer, byte(c.bufferedByte))
			for c.numBuffered > 0 {
				c.buffer = append(c.buffer, 0xFF)
				c.numBuffered--
			}
		}
	}

	return c.buffer
}

// GetBytes returns the current encoded bytes without finishing
func (c *CABACEncoder) GetBytes() []byte {
	return c.buffer
}

// EncodeSplitCuFlag encodes split_cu_flag with proper context selection
func (c *CABACEncoder) EncodeSplitCuFlag(split int, depth int) {
	// Context selection based on depth (simplified)
	// Full HEVC uses neighbor availability
	ctxIdx := HEVC_CTX_SPLIT_CU_FLAG
	if depth < 3 {
		ctxIdx += depth
	} else {
		ctxIdx += 2
	}
	c.EncodeBin(split, ctxIdx)
}

// EncodePredModeFlag encodes pred_mode_flag (1 = intra)
func (c *CABACEncoder) EncodePredModeFlag(intra int) {
	c.EncodeBin(intra, HEVC_CTX_PRED_MODE_FLAG)
}

// EncodePCMFlag encodes pcm_flag
// IMPORTANT: pcm_flag uses TERMINATING bin encoding, NOT context-based!
// From HM reference: m_pcBinIf->encodeBinTrm(uiIPCM)
func (c *CABACEncoder) EncodePCMFlag(pcm int) {
	c.EncodeTerminate(pcm)
}

// FlushForPCM flushes CABAC and prepares for raw PCM samples
// Based on HM: TEncBinCABAC::finish() + pcmAlignmentBits()
// Returns byte-aligned output with proper pcm_alignment bits
func (c *CABACEncoder) FlushForPCM() []byte {
	// This matches HM's finish() function
	// Check for carry in high bits of low register
	carry := c.low >> uint(32-c.bitsLeft)

	if carry != 0 {
		// Carry occurred - output bufferedByte+1, then zeros for remaining
		if c.bufferedByte >= 0 {
			c.buffer = append(c.buffer, byte(c.bufferedByte+1))
		}
		for c.numBuffered > 1 {
			c.buffer = append(c.buffer, 0x00)
			c.numBuffered--
		}
		c.low -= uint32(1) << uint(32-c.bitsLeft)
	} else {
		// No carry - output bufferedByte, then 0xFFs for remaining
		if c.numBuffered > 0 && c.bufferedByte >= 0 {
			c.buffer = append(c.buffer, byte(c.bufferedByte))
		}
		for c.numBuffered > 1 {
			c.buffer = append(c.buffer, 0xFF)
			c.numBuffered--
		}
	}

	// Output remaining bits from low register with proper alignment
	// HM sequence: finish() writes bits, then pcmAlignmentBits() writes 1 + zeros
	remainingBits := 24 - c.bitsLeft
	if remainingBits > 0 {
		remainingValue := c.low >> 8

		// Write full bytes first
		for remainingBits >= 8 {
			shift := remainingBits - 8
			c.buffer = append(c.buffer, byte(remainingValue>>uint(shift)))
			remainingBits -= 8
		}

		if remainingBits > 0 {
			// Partial byte case: we have remainingBits of CABAC data
			// Need to add pcm_alignment: 1 bit followed by zeros
			// Format: [CABAC bits][1][zeros to fill byte]
			cabacPart := (remainingValue << uint(8-remainingBits)) & 0xFF
			alignmentBit := uint32(1) << uint(7-remainingBits)
			c.buffer = append(c.buffer, byte(cabacPart|alignmentBit))
		} else {
			// Full byte boundary - need separate alignment byte
			c.buffer = append(c.buffer, 0x80)
		}
	} else {
		// No remaining bits - need alignment byte
		c.buffer = append(c.buffer, 0x80)
	}

	// Get bytes and clear state
	cabacBytes := make([]byte, len(c.buffer))
	copy(cabacBytes, c.buffer)
	c.buffer = c.buffer[:0]
	c.bufferedByte = -1
	c.numBuffered = 0

	return cabacBytes
}

// WritePCMAlignmentBits writes the pcm_alignment_one_bit and zero padding
// This should be called after FlushForPCM, before writing raw PCM samples
// Returns the alignment byte(s) to append to output
func WritePCMAlignmentBits() []byte {
	// In HEVC, after pcm_flag=1 and CABAC flush:
	// - Write pcm_alignment_one_bit (1)
	// - Write pcm_alignment_zero_bit until byte-aligned
	// Since we're starting fresh after CABAC flush, we need 1 + 7 zeros = 0x80
	return []byte{0x80}
}

// ResumeAfterPCM reinitializes CABAC state after PCM samples
// The contexts are preserved but the arithmetic coder is reset
func (c *CABACEncoder) ResumeAfterPCM() {
	c.low = 0
	c.range_ = 510
	c.bufferedByte = -1
	c.numBuffered = 0
	c.bitsLeft = 23
	// Note: contexts are NOT reset - they persist across PCM
}

// EncodePrevIntraLumaPredFlag encodes prev_intra_luma_pred_flag
// flag=1 means the mode is in the MPM list
func (c *CABACEncoder) EncodePrevIntraLumaPredFlag(flag int) {
	c.EncodeBin(flag, HEVC_CTX_PREV_INTRA_LUMA)
}

// EncodeMPMIdx encodes mpm_idx (truncated unary, bypass coded)
// idx is 0, 1, or 2 selecting which MPM to use
func (c *CABACEncoder) EncodeMPMIdx(idx int) {
	// Truncated unary with max 2
	// 0 -> 0
	// 1 -> 10
	// 2 -> 11
	if idx == 0 {
		c.EncodeBypass(0)
	} else {
		c.EncodeBypass(1)
		if idx == 1 {
			c.EncodeBypass(0)
		} else {
			c.EncodeBypass(1)
		}
	}
}

// EncodeRemIntraLumaPredMode encodes rem_intra_luma_pred_mode (5 bypass bins)
// mode is the remaining mode (0-31) after removing MPM candidates
func (c *CABACEncoder) EncodeRemIntraLumaPredMode(mode int) {
	// 5-bit fixed length, bypass coded
	for i := 4; i >= 0; i-- {
		c.EncodeBypass((mode >> uint(i)) & 1)
	}
}

// EncodeIntraChromaPredMode encodes intra_chroma_pred_mode
// HEVC spec: first bin is context-coded, remaining are bypass
// mode: 0=planar, 1=vertical, 2=horizontal, 3=DC, 4=derived from luma (DM_CHROMA)
func (c *CABACEncoder) EncodeIntraChromaPredMode(mode int) {
	// First bin is context-coded:
	// - 0 means DM_CHROMA (mode 4 - derive from luma)
	// - 1 means explicit mode, followed by 2 bypass bins
	if mode == 4 {
		// DM_CHROMA - just encode 0
		c.EncodeBin(0, HEVC_CTX_INTRA_CHROMA)
	} else {
		// Explicit chroma mode
		c.EncodeBin(1, HEVC_CTX_INTRA_CHROMA)
		// 2 bypass bins for mode 0-3
		c.EncodeBypass((mode >> 1) & 1)
		c.EncodeBypass(mode & 1)
	}
}

// EncodeRqtRootCbf encodes rqt_root_cbf (coded block flag for residual)
// cbf=0 means no residual, cbf=1 means residual present
func (c *CABACEncoder) EncodeRqtRootCbf(cbf int) {
	c.EncodeBin(cbf, HEVC_CTX_RQT_ROOT_CBF)
}
