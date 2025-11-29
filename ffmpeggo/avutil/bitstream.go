package avutil

// BitstreamWriter writes bits to a byte buffer.
// This is essential for video encoding where precise bit-level control is needed.
type BitstreamWriter struct {
	buf      []byte
	bitPos   int  // Current bit position within current byte (0-7)
	bytePos  int  // Current byte position
	curByte  byte // Current byte being built
}

// NewBitstreamWriter creates a new bitstream writer.
func NewBitstreamWriter() *BitstreamWriter {
	return &BitstreamWriter{
		buf: make([]byte, 0, 4096),
	}
}

// NewBitstreamWriterWithBuffer creates a bitstream writer with a pre-allocated buffer.
func NewBitstreamWriterWithBuffer(capacity int) *BitstreamWriter {
	return &BitstreamWriter{
		buf: make([]byte, 0, capacity),
	}
}

// WriteBits writes n bits (1-32) from value to the bitstream.
// Bits are written MSB first.
func (w *BitstreamWriter) WriteBits(value uint32, n int) {
	if n <= 0 || n > 32 {
		return
	}

	for i := n - 1; i >= 0; i-- {
		bit := (value >> i) & 1
		w.curByte = (w.curByte << 1) | byte(bit)
		w.bitPos++

		if w.bitPos == 8 {
			w.buf = append(w.buf, w.curByte)
			w.curByte = 0
			w.bitPos = 0
		}
	}
}

// WriteBit writes a single bit.
func (w *BitstreamWriter) WriteBit(bit int) {
	w.WriteBits(uint32(bit&1), 1)
}

// WriteU8 writes 8 bits.
func (w *BitstreamWriter) WriteU8(value byte) {
	w.WriteBits(uint32(value), 8)
}

// WriteU16 writes 16 bits.
func (w *BitstreamWriter) WriteU16(value uint16) {
	w.WriteBits(uint32(value), 16)
}

// WriteU32 writes 32 bits.
func (w *BitstreamWriter) WriteU32(value uint32) {
	w.WriteBits(value, 32)
}

// WriteBytes writes whole bytes to the bitstream.
// If the current position is byte-aligned, this is efficient.
func (w *BitstreamWriter) WriteBytes(data []byte) {
	for _, b := range data {
		w.WriteU8(b)
	}
}

// WriteUE writes an unsigned Exp-Golomb coded value.
// Used extensively in H.264/H.265 for variable-length encoding.
func (w *BitstreamWriter) WriteUE(value uint32) {
	// Exp-Golomb encoding:
	// 1. value + 1 in binary
	// 2. Count leading zeros needed
	// 3. Write (leadingZeros) zeros, then the (leadingZeros+1) bit value

	v := value + 1
	leadingZeros := 0
	tmp := v

	// Count bits needed
	for tmp > 1 {
		tmp >>= 1
		leadingZeros++
	}

	// Write leading zeros
	for i := 0; i < leadingZeros; i++ {
		w.WriteBit(0)
	}

	// Write the value (leadingZeros + 1 bits)
	w.WriteBits(v, leadingZeros+1)
}

// WriteSE writes a signed Exp-Golomb coded value.
func (w *BitstreamWriter) WriteSE(value int32) {
	// SE mapping: 0 -> 0, 1 -> 1, -1 -> 2, 2 -> 3, -2 -> 4, ...
	var ue uint32
	if value <= 0 {
		ue = uint32(-value) * 2
	} else {
		ue = uint32(value)*2 - 1
	}
	w.WriteUE(ue)
}

// Flush pads the bitstream to byte alignment.
// Pads with zero bits.
func (w *BitstreamWriter) Flush() {
	if w.bitPos > 0 {
		// Pad remaining bits with zeros
		w.curByte <<= (8 - w.bitPos)
		w.buf = append(w.buf, w.curByte)
		w.curByte = 0
		w.bitPos = 0
	}
}

// FlushWithRBSP flushes with RBSP trailing bits (1 bit followed by zero padding).
// Used at the end of NAL units.
func (w *BitstreamWriter) FlushWithRBSP() {
	// Write RBSP stop bit (1)
	w.WriteBit(1)
	// Pad to byte alignment with zeros
	if w.bitPos > 0 {
		w.curByte <<= (8 - w.bitPos)
		w.buf = append(w.buf, w.curByte)
		w.curByte = 0
		w.bitPos = 0
	}
}

// Bytes returns the written bytes.
// Call Flush() first if you need byte alignment.
func (w *BitstreamWriter) Bytes() []byte {
	return w.buf
}

// BitCount returns the total number of bits written.
func (w *BitstreamWriter) BitCount() int {
	return len(w.buf)*8 + w.bitPos
}

// ByteCount returns the number of complete bytes written.
func (w *BitstreamWriter) ByteCount() int {
	return len(w.buf)
}

// IsByteAligned returns true if the current position is byte-aligned.
func (w *BitstreamWriter) IsByteAligned() bool {
	return w.bitPos == 0
}

// Reset resets the writer for reuse.
func (w *BitstreamWriter) Reset() {
	w.buf = w.buf[:0]
	w.bitPos = 0
	w.curByte = 0
}

// BitstreamReader reads bits from a byte buffer.
type BitstreamReader struct {
	buf     []byte
	bitPos  int // Bit position in current byte (0-7, 0 = MSB)
	bytePos int
}

// NewBitstreamReader creates a new bitstream reader.
func NewBitstreamReader(data []byte) *BitstreamReader {
	return &BitstreamReader{
		buf: data,
	}
}

// ReadBits reads n bits (1-32) from the bitstream.
func (r *BitstreamReader) ReadBits(n int) (uint32, error) {
	if n <= 0 || n > 32 {
		return 0, ErrInvalidData
	}

	var value uint32
	for i := 0; i < n; i++ {
		if r.bytePos >= len(r.buf) {
			return 0, ErrEOF
		}

		bit := (r.buf[r.bytePos] >> (7 - r.bitPos)) & 1
		value = (value << 1) | uint32(bit)
		r.bitPos++

		if r.bitPos == 8 {
			r.bitPos = 0
			r.bytePos++
		}
	}

	return value, nil
}

// ReadBit reads a single bit.
func (r *BitstreamReader) ReadBit() (int, error) {
	v, err := r.ReadBits(1)
	return int(v), err
}

// ReadU8 reads 8 bits.
func (r *BitstreamReader) ReadU8() (byte, error) {
	v, err := r.ReadBits(8)
	return byte(v), err
}

// ReadU16 reads 16 bits.
func (r *BitstreamReader) ReadU16() (uint16, error) {
	v, err := r.ReadBits(16)
	return uint16(v), err
}

// ReadU32 reads 32 bits.
func (r *BitstreamReader) ReadU32() (uint32, error) {
	return r.ReadBits(32)
}

// ReadUE reads an unsigned Exp-Golomb coded value.
func (r *BitstreamReader) ReadUE() (uint32, error) {
	// Count leading zeros
	leadingZeros := 0
	for {
		bit, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit == 1 {
			break
		}
		leadingZeros++
		if leadingZeros > 31 {
			return 0, ErrInvalidData
		}
	}

	if leadingZeros == 0 {
		return 0, nil
	}

	// Read the remaining bits
	suffix, err := r.ReadBits(leadingZeros)
	if err != nil {
		return 0, err
	}

	return (1 << leadingZeros) - 1 + suffix, nil
}

// ReadSE reads a signed Exp-Golomb coded value.
func (r *BitstreamReader) ReadSE() (int32, error) {
	ue, err := r.ReadUE()
	if err != nil {
		return 0, err
	}

	// SE mapping: 0 -> 0, 1 -> 1, 2 -> -1, 3 -> 2, 4 -> -2, ...
	if ue&1 == 1 {
		return int32((ue + 1) / 2), nil
	}
	return -int32(ue / 2), nil
}

// BitsRemaining returns the number of bits remaining to read.
func (r *BitstreamReader) BitsRemaining() int {
	return (len(r.buf)-r.bytePos)*8 - r.bitPos
}

// BytesRemaining returns the number of bytes remaining (rounded down).
func (r *BitstreamReader) BytesRemaining() int {
	return len(r.buf) - r.bytePos
}

// IsByteAligned returns true if the current position is byte-aligned.
func (r *BitstreamReader) IsByteAligned() bool {
	return r.bitPos == 0
}

// SkipBits skips n bits.
func (r *BitstreamReader) SkipBits(n int) error {
	for n >= 8 && r.bitPos == 0 {
		r.bytePos++
		n -= 8
		if r.bytePos > len(r.buf) {
			return ErrEOF
		}
	}

	for i := 0; i < n; i++ {
		_, err := r.ReadBit()
		if err != nil {
			return err
		}
	}
	return nil
}

// Align aligns the reader to the next byte boundary.
func (r *BitstreamReader) Align() {
	if r.bitPos != 0 {
		r.bitPos = 0
		r.bytePos++
	}
}
