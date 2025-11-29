package avcodec

// CABAC (Context-Adaptive Binary Arithmetic Coding) encoder for HEVC.
// This implementation follows the HEVC specification and HM reference encoder.

// CABACEncoder encodes binary symbols using arithmetic coding.
type CABACEncoder struct {
	// Output buffer
	buf []byte

	// Arithmetic coding state
	low       uint32 // Low end of the interval
	range_    uint32 // Current interval range
	bitsLeft  int    // Bits remaining before output
	bufferedByte byte
	numBufferedBytes int

	// Statistics
	bitsEncoded int
}

// Context model state for CABAC
type CABACContext struct {
	State uint8 // 6-bit state (0-63)
	MPS   uint8 // Most probable symbol (0 or 1)
}

// NewCABACEncoder creates a new CABAC encoder.
func NewCABACEncoder() *CABACEncoder {
	c := &CABACEncoder{
		buf: make([]byte, 0, 65536),
	}
	c.Reset()
	return c
}

// Reset resets the encoder state for a new slice.
func (c *CABACEncoder) Reset() {
	c.low = 0
	c.range_ = 510
	c.bitsLeft = 23
	c.bufferedByte = 0xFF
	c.numBufferedBytes = 0
	c.buf = c.buf[:0]
	c.bitsEncoded = 0
}

// InitContext initializes a context from initValue.
// initValue is derived from init_type and SliceQPY as per HEVC spec.
func InitContext(initValue int, sliceQP int) CABACContext {
	// HEVC context initialization:
	// slope = (initValue >> 4) * 5 - 45
	// offset = ((initValue & 15) << 3) - 16
	// state = Clip3(1, 126, ((slope * sliceQP) >> 4) + offset)
	// if state >= 64: MPS = 1, state = state - 64
	// else: MPS = 0

	slope := (initValue>>4)*5 - 45
	offset := ((initValue & 15) << 3) - 16
	state := ((slope * sliceQP) >> 4) + offset

	// Clip to [1, 126]
	if state < 1 {
		state = 1
	} else if state > 126 {
		state = 126
	}

	var ctx CABACContext
	if state >= 64 {
		ctx.MPS = 1
		ctx.State = uint8(state - 64)
	} else {
		ctx.MPS = 0
		ctx.State = uint8(63 - state)
	}

	return ctx
}

// LPS probability table (scaled by 2^15)
// Index is the state (0-63)
var cabacLPSTable = [64][4]uint16{
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

// State transition table for LPS
var cabacStateLPS = [64]uint8{
	0, 0, 1, 2, 2, 4, 4, 5, 6, 7, 8, 9, 9, 11, 11, 12,
	13, 13, 15, 15, 16, 16, 18, 18, 19, 19, 21, 21, 22, 22, 23, 24,
	24, 25, 26, 26, 27, 27, 28, 29, 29, 30, 30, 30, 31, 32, 32, 33,
	33, 33, 34, 34, 35, 35, 35, 36, 36, 36, 37, 37, 37, 38, 38, 63,
}

// State transition table for MPS
var cabacStateMPS = [64]uint8{
	1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 62, 63,
}

// EncodeBin encodes a single binary symbol using the given context.
func (c *CABACEncoder) EncodeBin(bin int, ctx *CABACContext) {
	state := ctx.State
	mps := ctx.MPS

	// Get range index based on current range
	rangeIdx := (c.range_ >> 6) & 3

	// Get LPS range from table
	lpsRange := uint32(cabacLPSTable[state][rangeIdx])

	// Update range for MPS
	c.range_ -= lpsRange

	if bin != int(mps) {
		// LPS occurred
		c.low += c.range_
		c.range_ = lpsRange

		// Update context state for LPS
		if state == 0 {
			ctx.MPS ^= 1 // Flip MPS
		}
		ctx.State = cabacStateLPS[state]
	} else {
		// MPS occurred - update context state for MPS
		ctx.State = cabacStateMPS[state]
	}

	// Renormalize
	c.renormalize()
}

// EncodeBypass encodes a binary symbol in bypass mode (equiprobable).
func (c *CABACEncoder) EncodeBypass(bin int) {
	c.low <<= 1
	if bin != 0 {
		c.low += c.range_
	}

	c.bitsLeft--
	if c.bitsLeft < 12 {
		c.outputBits()
	}
}

// EncodeBypassBins encodes multiple bypass bins from a value.
func (c *CABACEncoder) EncodeBypassBins(value uint32, numBins int) {
	for i := numBins - 1; i >= 0; i-- {
		c.EncodeBypass(int((value >> i) & 1))
	}
}

// EncodeTerminate encodes the terminating bin.
// bin=1 indicates end of slice/tile.
func (c *CABACEncoder) EncodeTerminate(bin int) {
	c.range_ -= 2

	if bin != 0 {
		// End of slice - encode 1 in the terminate position
		c.low += c.range_
		c.range_ = 2

		// Renormalize to flush remaining bits
		c.renormalize()

		// Output remaining bits
		c.outputBits()

		// Write final bits
		c.buf = append(c.buf, byte((c.low>>15)&0xFF))
		c.buf = append(c.buf, byte((c.low>>7)&0xFF))

		c.bitsEncoded += 8

		// Reset to prevent further encoding
		c.low = 0
		c.range_ = 510
	} else {
		// Not end - continue encoding
		c.renormalize()
	}
}

// renormalize performs range renormalization.
func (c *CABACEncoder) renormalize() {
	for c.range_ < 256 {
		c.bitsLeft--
		if c.bitsLeft < 12 {
			c.outputBits()
		}
		c.range_ <<= 1
		c.low <<= 1
	}
}

// outputBits outputs bits when buffer is full.
func (c *CABACEncoder) outputBits() {
	leadByte := c.low >> (24 - c.bitsLeft)

	c.bitsLeft += 8

	if c.numBufferedBytes > 0 {
		if leadByte == 0xFF {
			c.numBufferedBytes++
		} else {
			// Output buffered bytes
			carry := leadByte >> 8
			byteToWrite := c.bufferedByte + byte(carry)
			c.buf = append(c.buf, byteToWrite)

			// Output any stacked 0xFF bytes
			byteToWrite = byte(leadByte)
			for c.numBufferedBytes > 1 {
				stackByte := byte(0xFF + carry)
				c.buf = append(c.buf, stackByte)
				c.numBufferedBytes--
			}

			c.numBufferedBytes = 1
			c.bufferedByte = byteToWrite
		}
	} else {
		c.numBufferedBytes = 1
		c.bufferedByte = byte(leadByte)
	}

	c.low &= (1 << (24 - c.bitsLeft)) - 1
}

// Finish finalizes the CABAC stream.
func (c *CABACEncoder) Finish() {
	// Terminate with bin=1
	c.EncodeTerminate(1)
}

// Bytes returns the encoded byte stream.
func (c *CABACEncoder) Bytes() []byte {
	return c.buf
}

// BitsEncoded returns the number of bits encoded.
func (c *CABACEncoder) BitsEncoded() int {
	return c.bitsEncoded + len(c.buf)*8
}

// AppendBytes appends raw bytes to the output (for PCM data).
func (c *CABACEncoder) AppendBytes(data []byte) {
	c.buf = append(c.buf, data...)
}

// EncodeUnaryMax encodes a value using truncated unary coding.
// Value must be in range [0, max].
func (c *CABACEncoder) EncodeUnaryMax(value int, max int, ctx *CABACContext) {
	for i := 0; i < value; i++ {
		c.EncodeBin(1, ctx)
	}
	if value < max {
		c.EncodeBin(0, ctx)
	}
}

// EncodeExpGolombBypass encodes a value using Exp-Golomb coding in bypass mode.
func (c *CABACEncoder) EncodeExpGolombBypass(value uint32, k int) {
	// Rice-Golomb coding
	prefix := value >> k
	suffix := value & ((1 << k) - 1)

	// Unary prefix
	for prefix > 0 {
		c.EncodeBypass(1)
		prefix--
	}
	c.EncodeBypass(0)

	// Binary suffix
	c.EncodeBypassBins(suffix, k)
}

// Helper function to get the number of bits needed to represent a value
func bitsNeeded(value uint32) int {
	if value == 0 {
		return 1
	}
	bits := 0
	for value > 0 {
		bits++
		value >>= 1
	}
	return bits
}
