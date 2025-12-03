package avcodec

// ResidualCoder handles CABAC encoding of transform coefficients
type ResidualCoder struct {
	cabac *CABACEncoder

	// Context models for residual coding
	ctxLastSigCoeffXPrefix []CABACContext // 18 contexts
	ctxLastSigCoeffYPrefix []CABACContext // 18 contexts
	ctxCodedSubBlockFlag   []CABACContext // 4 contexts
	ctxSigCoeffFlag        []CABACContext // 42 contexts for luma
	ctxCoeffAbsGreater1    []CABACContext // 24 contexts
	ctxCoeffAbsGreater2    []CABACContext // 6 contexts
}

// NewResidualCoder creates a new residual coder
func NewResidualCoder(cabac *CABACEncoder, qp int) *ResidualCoder {
	rc := &ResidualCoder{
		cabac: cabac,
	}
	rc.initContexts(qp)
	return rc
}

// initContexts initializes CABAC contexts for residual coding
func (rc *ResidualCoder) initContexts(qp int) {
	// last_sig_coeff_x_prefix - 18 contexts
	// Using init_type=0 (I-slice)
	rc.ctxLastSigCoeffXPrefix = make([]CABACContext, 18)
	lastSigXInitVals := []int{110, 110, 124, 125, 140, 153, 125, 127, 140, 109, 111, 143, 127, 111, 79, 108, 123, 63}
	for i := 0; i < 18; i++ {
		rc.ctxLastSigCoeffXPrefix[i] = InitContext(lastSigXInitVals[i], qp)
	}

	// last_sig_coeff_y_prefix - 18 contexts
	// Using init_type=0 (I-slice)
	rc.ctxLastSigCoeffYPrefix = make([]CABACContext, 18)
	lastSigYInitVals := []int{110, 110, 124, 125, 140, 153, 125, 127, 140, 109, 111, 143, 127, 111, 79, 108, 123, 63}
	for i := 0; i < 18; i++ {
		rc.ctxLastSigCoeffYPrefix[i] = InitContext(lastSigYInitVals[i], qp)
	}

	// coded_sub_block_flag - 4 contexts
	// Using init_type=0 (I-slice)
	rc.ctxCodedSubBlockFlag = make([]CABACContext, 4)
	codedSBFlagInitVals := []int{91, 176, 93, 154}
	for i := 0; i < 4; i++ {
		rc.ctxCodedSubBlockFlag[i] = InitContext(codedSBFlagInitVals[i], qp)
	}

	// sig_coeff_flag - 44 contexts (42 + 2 for DC)
	// Using init_type=0 (I-slice)
	rc.ctxSigCoeffFlag = make([]CABACContext, 44)
	sigCoeffInitVals := []int{
		111, 141, 112, 154, 124, 154, 139, 153, 139, 123,
		123, 63, 124, 166, 183, 140, 136, 153, 154, 166,
		183, 140, 136, 153, 154, 170, 153, 123, 123, 107,
		121, 107, 121, 167, 151, 183, 140, 151, 183, 140,
		140, 140, 140, 140,
	}
	for i := 0; i < 44; i++ {
		rc.ctxSigCoeffFlag[i] = InitContext(sigCoeffInitVals[i], qp)
	}

	// coeff_abs_level_greater1_flag - 24 contexts
	// Using init_type=0 (I-slice)
	rc.ctxCoeffAbsGreater1 = make([]CABACContext, 24)
	greater1InitVals := []int{
		140, 92, 126, 154, 140, 123, 154, 154, 154, 107,
		126, 154, 154, 154, 154, 154, 140, 92, 126, 154,
		140, 123, 154, 154,
	}
	for i := 0; i < 24; i++ {
		rc.ctxCoeffAbsGreater1[i] = InitContext(greater1InitVals[i], qp)
	}

	// coeff_abs_level_greater2_flag - 6 contexts
	// Using init_type=0 (I-slice)
	rc.ctxCoeffAbsGreater2 = make([]CABACContext, 6)
	greater2InitVals := []int{138, 153, 136, 167, 152, 152}
	for i := 0; i < 6; i++ {
		rc.ctxCoeffAbsGreater2[i] = InitContext(greater2InitVals[i], qp)
	}
}

// EncodeResidual encodes a transform unit's coefficients
func (rc *ResidualCoder) EncodeResidual(tu *TransformUnit, isLuma bool) {
	if !tu.HasCoeffs {
		return
	}

	log2Size := 2 // Default for 4x4
	switch tu.Size {
	case 8:
		log2Size = 3
	case 16:
		log2Size = 4
	case 32:
		log2Size = 5
	}

	// Encode last significant coefficient position
	rc.encodeLastSigCoeffPos(tu.LastSigX, tu.LastSigY, log2Size, isLuma)

	// For simplicity, encode all coefficients in a single sub-block for 4x4
	// Larger blocks would need sub-block processing
	if tu.Size == 4 {
		rc.encodeCoeffs4x4(tu.Coeffs)
	} else if tu.Size == 8 {
		rc.encodeCoeffs8x8(tu.Coeffs)
	}
}

// encodeLastSigCoeffPos encodes the position of the last significant coefficient
func (rc *ResidualCoder) encodeLastSigCoeffPos(lastX, lastY, log2Size int, isLuma bool) {
	// Encode X prefix using truncated unary
	ctxOffset := 0
	if !isLuma {
		ctxOffset = 15
	}

	maxPos := (1 << log2Size) - 1

	// Encode last_sig_coeff_x_prefix
	rc.encodeTruncatedUnary(lastX, maxPos, rc.ctxLastSigCoeffXPrefix, ctxOffset, log2Size)

	// Encode last_sig_coeff_y_prefix
	rc.encodeTruncatedUnary(lastY, maxPos, rc.ctxLastSigCoeffYPrefix, ctxOffset, log2Size)
}

// encodeTruncatedUnary encodes a value using truncated unary coding
func (rc *ResidualCoder) encodeTruncatedUnary(val, maxVal int, contexts []CABACContext, offset, log2Size int) {
	// HEVC uses a specific context selection for last sig coeff
	// For simplicity, use a basic truncated unary encoding

	prefixLen := val
	if prefixLen > maxVal {
		prefixLen = maxVal
	}

	// Group index based on log2Size
	groupIdx := (log2Size - 2) * 3

	for i := 0; i < prefixLen && i < len(contexts)-offset; i++ {
		ctxIdx := offset + groupIdx
		if ctxIdx+i < len(contexts) {
			rc.cabac.EncodeBin(1, &contexts[ctxIdx+min(i, 2)])
		}
	}

	if prefixLen < maxVal {
		ctxIdx := offset + groupIdx
		if ctxIdx < len(contexts) {
			rc.cabac.EncodeBin(0, &contexts[ctxIdx+min(prefixLen, 2)])
		}
	}

	// If prefix indicates suffix is needed, encode suffix in bypass mode
	if val >= 4 && log2Size > 2 {
		suffix := val - 4
		suffixBits := log2Size - 2
		for i := suffixBits - 1; i >= 0; i-- {
			rc.cabac.EncodeBypass((suffix >> i) & 1)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// encodeCoeffs4x4 encodes coefficients for a 4x4 transform block
func (rc *ResidualCoder) encodeCoeffs4x4(coeffs []int16) {
	// Find last significant coefficient
	lastSig := -1
	for i := len(coeffs) - 1; i >= 0; i-- {
		if coeffs[i] != 0 {
			lastSig = i
			break
		}
	}

	if lastSig < 0 {
		return
	}

	// Collect significant coefficient info
	type coeffInfo struct {
		pos       int
		absLevel  int
		sign      int
		greater1  bool
		greater2  bool
		remaining int
	}

	sigCoeffs := make([]coeffInfo, 0, 16)

	// Process coefficients in reverse scan order (from last to first)
	for i := lastSig; i >= 0; i-- {
		coeff := coeffs[i]
		if coeff != 0 {
			absLevel := int(coeff)
			sign := 0
			if absLevel < 0 {
				sign = 1
				absLevel = -absLevel
			}

			info := coeffInfo{
				pos:      i,
				absLevel: absLevel,
				sign:     sign,
				greater1: absLevel > 1,
				greater2: absLevel > 2,
			}
			if absLevel > 2 {
				info.remaining = absLevel - 3
			} else if absLevel > 1 {
				info.remaining = 0
			}

			sigCoeffs = append(sigCoeffs, info)
		}
	}

	// Encode sig_coeff_flags for positions before lastSig
	for i := lastSig - 1; i >= 0; i-- {
		sigFlag := 0
		if coeffs[i] != 0 {
			sigFlag = 1
		}
		ctxIdx := rc.getSigCoeffCtxIdx(i, 4)
		rc.cabac.EncodeBin(sigFlag, &rc.ctxSigCoeffFlag[ctxIdx])
	}

	// Encode coeff_abs_level_greater1_flag (up to 8 coefficients)
	numGreater1 := 0
	firstGreater1 := -1
	for i, info := range sigCoeffs {
		if numGreater1 >= 8 {
			break
		}
		greater1 := 0
		if info.greater1 {
			greater1 = 1
			if firstGreater1 < 0 {
				firstGreater1 = i
			}
		}
		ctxIdx := min(numGreater1, 3)
		rc.cabac.EncodeBin(greater1, &rc.ctxCoeffAbsGreater1[ctxIdx])
		numGreater1++
	}

	// Encode coeff_abs_level_greater2_flag for first greater1 coefficient
	if firstGreater1 >= 0 && sigCoeffs[firstGreater1].greater2 {
		rc.cabac.EncodeBin(1, &rc.ctxCoeffAbsGreater2[0])
	} else if firstGreater1 >= 0 {
		rc.cabac.EncodeBin(0, &rc.ctxCoeffAbsGreater2[0])
	}

	// Encode sign flags in bypass mode
	for _, info := range sigCoeffs {
		rc.cabac.EncodeBypass(info.sign)
	}

	// Encode remaining absolute levels using Golomb-Rice coding in bypass mode
	for i, info := range sigCoeffs {
		remaining := 0
		if i == firstGreater1 && info.greater2 {
			remaining = info.absLevel - 3
		} else if info.greater1 && i < 8 {
			remaining = info.absLevel - 2
		} else if info.greater1 {
			remaining = info.absLevel - 1
		}

		if remaining > 0 || (info.greater1 && i >= 8) {
			rc.encodeCoeffAbsLevelRemaining(remaining)
		}
	}
}

// encodeCoeffs8x8 encodes coefficients for an 8x8 transform block
func (rc *ResidualCoder) encodeCoeffs8x8(coeffs []int16) {
	// For 8x8, we process 4x4 sub-blocks
	// For simplicity, use similar logic to 4x4 but with sub-block structure

	// Find last significant coefficient
	lastSig := -1
	for i := len(coeffs) - 1; i >= 0; i-- {
		if coeffs[i] != 0 {
			lastSig = i
			break
		}
	}

	if lastSig < 0 {
		return
	}

	// Process all coefficients similar to 4x4 (simplified)
	// A full implementation would use coded_sub_block_flag per 4x4 sub-block

	// Collect significant coefficients
	type coeffInfo struct {
		absLevel int
		sign     int
	}
	sigCoeffs := make([]coeffInfo, 0, 64)

	// Encode sig_coeff_flags and collect info
	for i := lastSig; i >= 0; i-- {
		if i < lastSig {
			sigFlag := 0
			if coeffs[i] != 0 {
				sigFlag = 1
			}
			ctxIdx := rc.getSigCoeffCtxIdx(i, 8) % 42
			rc.cabac.EncodeBin(sigFlag, &rc.ctxSigCoeffFlag[ctxIdx])
		}

		if coeffs[i] != 0 {
			absLevel := int(coeffs[i])
			sign := 0
			if absLevel < 0 {
				sign = 1
				absLevel = -absLevel
			}
			sigCoeffs = append(sigCoeffs, coeffInfo{absLevel: absLevel, sign: sign})
		}
	}

	// Encode greater1 flags (up to 8)
	numGreater1 := 0
	firstGreater1 := -1
	for i, info := range sigCoeffs {
		if numGreater1 >= 8 {
			break
		}
		greater1 := 0
		if info.absLevel > 1 {
			greater1 = 1
			if firstGreater1 < 0 {
				firstGreater1 = i
			}
		}
		ctxIdx := min(numGreater1, 3)
		rc.cabac.EncodeBin(greater1, &rc.ctxCoeffAbsGreater1[ctxIdx])
		numGreater1++
	}

	// Encode greater2 flag for first greater1 coeff
	if firstGreater1 >= 0 {
		greater2 := 0
		if sigCoeffs[firstGreater1].absLevel > 2 {
			greater2 = 1
		}
		rc.cabac.EncodeBin(greater2, &rc.ctxCoeffAbsGreater2[0])
	}

	// Encode signs
	for _, info := range sigCoeffs {
		rc.cabac.EncodeBypass(info.sign)
	}

	// Encode remaining levels
	for i, info := range sigCoeffs {
		var remaining int
		if i == firstGreater1 && info.absLevel > 2 {
			remaining = info.absLevel - 3
		} else if info.absLevel > 1 && i < 8 {
			remaining = info.absLevel - 2
		} else if info.absLevel > 1 {
			remaining = info.absLevel - 1
		}

		if remaining > 0 || (info.absLevel > 1 && i >= 8) {
			rc.encodeCoeffAbsLevelRemaining(remaining)
		}
	}
}

// getSigCoeffCtxIdx returns the context index for sig_coeff_flag
func (rc *ResidualCoder) getSigCoeffCtxIdx(scanPos, blockSize int) int {
	// Simplified context selection based on position
	// Full implementation would consider neighboring coefficients

	if blockSize == 4 {
		// For 4x4, use position-based context
		if scanPos == 0 {
			return 0
		}
		return min(scanPos, 8)
	}

	// For larger blocks, use sub-block position
	subBlockIdx := scanPos / 16
	posInSubBlock := scanPos % 16

	baseCtx := 0
	if subBlockIdx > 0 {
		baseCtx = 21
	}

	return baseCtx + min(posInSubBlock, 8)
}

// encodeCoeffAbsLevelRemaining encodes remaining absolute level using Golomb-Rice
func (rc *ResidualCoder) encodeCoeffAbsLevelRemaining(level int) {
	// Golomb-Rice parameter (simplified, using 0)
	riceParam := 0

	// Threshold for switching to Exp-Golomb
	threshold := 3 << riceParam

	if level < threshold {
		// Use truncated Rice code
		prefix := level >> riceParam
		for i := 0; i < prefix; i++ {
			rc.cabac.EncodeBypass(1)
		}
		rc.cabac.EncodeBypass(0)

		// Encode suffix
		for i := riceParam - 1; i >= 0; i-- {
			rc.cabac.EncodeBypass((level >> i) & 1)
		}
	} else {
		// Use Exp-Golomb for large values
		level -= threshold

		// Prefix of 1s
		for i := 0; i < threshold; i++ {
			rc.cabac.EncodeBypass(1)
		}

		// Exp-Golomb suffix
		length := 1
		for (1 << length) <= level {
			length++
		}

		// Write length-1 ones then a zero
		for i := 0; i < length-1; i++ {
			rc.cabac.EncodeBypass(1)
		}
		rc.cabac.EncodeBypass(0)

		// Write the value minus (1 << (length-1))
		suffix := level - (1 << (length - 1))
		for i := length - 2; i >= 0; i-- {
			rc.cabac.EncodeBypass((suffix >> i) & 1)
		}
	}
}
