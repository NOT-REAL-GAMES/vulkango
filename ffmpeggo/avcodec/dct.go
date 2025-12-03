package avcodec

import "math"

// HEVC uses integer DCT transforms
// These are the standard HEVC transform matrices

// 4x4 DCT matrix (scaled by 64)
var dct4x4Matrix = [4][4]int16{
	{64, 64, 64, 64},
	{83, 36, -36, -83},
	{64, -64, -64, 64},
	{36, -83, 83, -36},
}

// 8x8 DCT matrix (scaled by 64)
var dct8x8Matrix = [8][8]int16{
	{64, 64, 64, 64, 64, 64, 64, 64},
	{89, 75, 50, 18, -18, -50, -75, -89},
	{83, 36, -36, -83, -83, -36, 36, 83},
	{75, -18, -89, -50, 50, 89, 18, -75},
	{64, -64, -64, 64, 64, -64, -64, 64},
	{50, -89, 18, 75, -75, -18, 89, -50},
	{36, -83, 83, -36, -36, 83, -83, 36},
	{18, -50, 75, -89, 89, -75, 50, -18},
}

// 16x16 DCT matrix
var dct16x16Matrix [16][16]int16

// 32x32 DCT matrix
var dct32x32Matrix [32][32]int16

func init() {
	// Initialize 16x16 DCT matrix
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			if i == 0 {
				dct16x16Matrix[i][j] = 64
			} else {
				dct16x16Matrix[i][j] = int16(64 * math.Cos(float64(i)*math.Pi*(2*float64(j)+1)/32))
			}
		}
	}

	// Initialize 32x32 DCT matrix
	for i := 0; i < 32; i++ {
		for j := 0; j < 32; j++ {
			if i == 0 {
				dct32x32Matrix[i][j] = 64
			} else {
				dct32x32Matrix[i][j] = int16(64 * math.Cos(float64(i)*math.Pi*(2*float64(j)+1)/64))
			}
		}
	}
}

// DCT4x4 performs a 4x4 integer DCT transform
func DCT4x4(block [4][4]int16) [4][4]int16 {
	var temp [4][4]int32
	var result [4][4]int16

	// Horizontal transform
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var sum int32
			for k := 0; k < 4; k++ {
				sum += int32(dct4x4Matrix[j][k]) * int32(block[i][k])
			}
			temp[i][j] = sum
		}
	}

	// Vertical transform
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var sum int32
			for k := 0; k < 4; k++ {
				sum += int32(dct4x4Matrix[i][k]) * temp[k][j]
			}
			// Scale down by 2^7 (matrix is scaled by 64, applied twice = 4096, divide by 128)
			result[i][j] = int16((sum + 2048) >> 12)
		}
	}

	return result
}

// DCT8x8 performs an 8x8 integer DCT transform
func DCT8x8(block [8][8]int16) [8][8]int16 {
	var temp [8][8]int32
	var result [8][8]int16

	// Horizontal transform
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			var sum int32
			for k := 0; k < 8; k++ {
				sum += int32(dct8x8Matrix[j][k]) * int32(block[i][k])
			}
			temp[i][j] = sum
		}
	}

	// Vertical transform
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			var sum int32
			for k := 0; k < 8; k++ {
				sum += int32(dct8x8Matrix[i][k]) * temp[k][j]
			}
			// Scale down
			result[i][j] = int16((sum + 2048) >> 12)
		}
	}

	return result
}

// IDCT4x4 performs a 4x4 inverse DCT transform
func IDCT4x4(block [4][4]int16) [4][4]int16 {
	var temp [4][4]int32
	var result [4][4]int16

	// Vertical inverse transform (transpose of forward matrix)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var sum int32
			for k := 0; k < 4; k++ {
				sum += int32(dct4x4Matrix[k][i]) * int32(block[k][j])
			}
			temp[i][j] = sum
		}
	}

	// Horizontal inverse transform
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var sum int32
			for k := 0; k < 4; k++ {
				sum += int32(dct4x4Matrix[k][j]) * temp[i][k]
			}
			// Scale down and clip
			val := (sum + 2048) >> 12
			if val < -32768 {
				val = -32768
			} else if val > 32767 {
				val = 32767
			}
			result[i][j] = int16(val)
		}
	}

	return result
}

// Quantization tables for HEVC
// QP ranges from 0 to 51
// Quantization step doubles every 6 QP values

// QuantMatrix holds per-size quantization scaling factors
type QuantMatrix struct {
	Scale4x4   [4][4]int32
	Scale8x8   [8][8]int32
	Scale16x16 [16][16]int32
	Scale32x32 [32][32]int32
}

// Default flat quantization matrix (all 16)
var defaultQuantMatrix = func() QuantMatrix {
	var m QuantMatrix
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			m.Scale4x4[i][j] = 16
		}
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			m.Scale8x8[i][j] = 16
		}
	}
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			m.Scale16x16[i][j] = 16
		}
	}
	for i := 0; i < 32; i++ {
		for j := 0; j < 32; j++ {
			m.Scale32x32[i][j] = 16
		}
	}
	return m
}()

// Quantization scaling factors (from HEVC spec table 8-4)
// These are the values for qp % 6
var quantScaleFactors = [6]int32{26214, 23302, 20560, 18396, 16384, 14564}
var dequantScaleFactors = [6]int32{40, 45, 51, 57, 64, 72}

// Quantize4x4 quantizes a 4x4 DCT block
func Quantize4x4(block [4][4]int16, qp int) [4][4]int16 {
	var result [4][4]int16

	qpDiv6 := qp / 6
	qpMod6 := qp % 6
	scale := quantScaleFactors[qpMod6]
	shift := 14 + qpDiv6

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			coeff := int32(block[i][j])
			// Apply quantization: coeff * scale >> shift
			sign := int32(1)
			if coeff < 0 {
				sign = -1
				coeff = -coeff
			}
			quantized := (coeff*scale + (1 << (shift - 1))) >> shift
			result[i][j] = int16(sign * quantized)
		}
	}

	return result
}

// Quantize8x8 quantizes an 8x8 DCT block
func Quantize8x8(block [8][8]int16, qp int) [8][8]int16 {
	var result [8][8]int16

	qpDiv6 := qp / 6
	qpMod6 := qp % 6
	scale := quantScaleFactors[qpMod6]
	shift := 14 + qpDiv6

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			coeff := int32(block[i][j])
			sign := int32(1)
			if coeff < 0 {
				sign = -1
				coeff = -coeff
			}
			quantized := (coeff*scale + (1 << (shift - 1))) >> shift
			result[i][j] = int16(sign * quantized)
		}
	}

	return result
}

// Dequantize4x4 dequantizes a 4x4 coefficient block
func Dequantize4x4(block [4][4]int16, qp int) [4][4]int16 {
	var result [4][4]int16

	qpDiv6 := qp / 6
	qpMod6 := qp % 6
	scale := dequantScaleFactors[qpMod6]

	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			coeff := int32(block[i][j])
			// Dequantization: (coeff * scale) << qpDiv6
			dequantized := (coeff * scale) << qpDiv6
			// Clip to 16-bit range
			if dequantized < -32768 {
				dequantized = -32768
			} else if dequantized > 32767 {
				dequantized = 32767
			}
			result[i][j] = int16(dequantized)
		}
	}

	return result
}

// HEVC diagonal scan patterns
// For 4x4 block
var diagonalScan4x4 = [16][2]int{
	{0, 0}, {0, 1}, {1, 0}, {2, 0},
	{1, 1}, {0, 2}, {0, 3}, {1, 2},
	{2, 1}, {3, 0}, {3, 1}, {2, 2},
	{1, 3}, {2, 3}, {3, 2}, {3, 3},
}

// For 8x8 block
var diagonalScan8x8 [64][2]int

func init() {
	// Generate 8x8 diagonal scan
	idx := 0
	for diag := 0; diag < 15; diag++ {
		if diag%2 == 0 {
			// Scan down-right
			for i := min8(diag, 7); i >= max8(0, diag-7); i-- {
				j := diag - i
				diagonalScan8x8[idx] = [2]int{i, j}
				idx++
			}
		} else {
			// Scan up-left
			for i := max8(0, diag-7); i <= min8(diag, 7); i++ {
				j := diag - i
				diagonalScan8x8[idx] = [2]int{i, j}
				idx++
			}
		}
	}
}

func min8(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max8(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ScanCoeffs4x4 scans a 4x4 block in diagonal order and returns coefficients
func ScanCoeffs4x4(block [4][4]int16) []int16 {
	result := make([]int16, 16)
	for i := 0; i < 16; i++ {
		pos := diagonalScan4x4[i]
		result[i] = block[pos[0]][pos[1]]
	}
	return result
}

// ScanCoeffs8x8 scans an 8x8 block in diagonal order and returns coefficients
func ScanCoeffs8x8(block [8][8]int16) []int16 {
	result := make([]int16, 64)
	for i := 0; i < 64; i++ {
		pos := diagonalScan8x8[i]
		result[i] = block[pos[0]][pos[1]]
	}
	return result
}

// FindLastSigCoeff finds the last significant (non-zero) coefficient position
func FindLastSigCoeff(coeffs []int16) int {
	for i := len(coeffs) - 1; i >= 0; i-- {
		if coeffs[i] != 0 {
			return i
		}
	}
	return -1 // All zeros
}

// TransformUnit holds data for a transform unit
type TransformUnit struct {
	Size      int       // 4, 8, 16, or 32
	Coeffs    []int16   // Quantized coefficients in scan order
	LastSigX  int       // X position of last significant coeff
	LastSigY  int       // Y position of last significant coeff
	HasCoeffs bool      // True if any non-zero coefficients
}

// ProcessBlock4x4 processes a 4x4 pixel block through DCT and quantization
func ProcessBlock4x4(pixels [4][4]int16, qp int) TransformUnit {
	// Forward DCT
	dctBlock := DCT4x4(pixels)

	// Quantize
	quantBlock := Quantize4x4(dctBlock, qp)

	// Scan in diagonal order
	coeffs := ScanCoeffs4x4(quantBlock)

	// Find last significant coefficient
	lastSig := FindLastSigCoeff(coeffs)

	tu := TransformUnit{
		Size:      4,
		Coeffs:    coeffs,
		HasCoeffs: lastSig >= 0,
	}

	if lastSig >= 0 {
		pos := diagonalScan4x4[lastSig]
		tu.LastSigX = pos[1]
		tu.LastSigY = pos[0]
	}

	return tu
}

// ProcessBlock8x8 processes an 8x8 pixel block through DCT and quantization
func ProcessBlock8x8(pixels [8][8]int16, qp int) TransformUnit {
	// Forward DCT
	dctBlock := DCT8x8(pixels)

	// Quantize
	quantBlock := Quantize8x8(dctBlock, qp)

	// Scan in diagonal order
	coeffs := ScanCoeffs8x8(quantBlock)

	// Find last significant coefficient
	lastSig := FindLastSigCoeff(coeffs)

	tu := TransformUnit{
		Size:      8,
		Coeffs:    coeffs,
		HasCoeffs: lastSig >= 0,
	}

	if lastSig >= 0 {
		pos := diagonalScan8x8[lastSig]
		tu.LastSigX = pos[1]
		tu.LastSigY = pos[0]
	}

	return tu
}
