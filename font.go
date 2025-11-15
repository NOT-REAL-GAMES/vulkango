package vulkango

/*
#cgo pkg-config: vulkan
#cgo LDFLAGS: -lm

#define STB_TRUETYPE_IMPLEMENTATION
#define STBTT_STATIC
#include <stdlib.h>
#include "stb_truetype.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// BakedChar represents a baked character in the atlas
type BakedChar struct {
	X0, Y0, X1, Y1   uint16  // Atlas coordinates
	XOffset, YOffset float32 // Offset when rendering
	XAdvance         float32 // Horizontal advance
}

// FontInfo holds font metadata
type FontInfo struct {
	data   []byte
	handle *C.stbtt_fontinfo
}

// BakeFontBitmap bakes a font into a bitmap atlas
// Returns the baked character data and the atlas bitmap
func BakeFontBitmap(fontData []byte, pixelHeight float32, atlasWidth, atlasHeight int, firstChar, numChars int) ([]BakedChar, []byte, error) {
	atlas := make([]byte, atlasWidth*atlasHeight)
	charData := make([]C.stbtt_bakedchar, numChars)

	result := C.stbtt_BakeFontBitmap(
		(*C.uchar)(unsafe.Pointer(&fontData[0])),
		0,
		C.float(pixelHeight),
		(*C.uchar)(unsafe.Pointer(&atlas[0])),
		C.int(atlasWidth),
		C.int(atlasHeight),
		C.int(firstChar),
		C.int(numChars),
		&charData[0],
	)

	if result <= 0 {
		return nil, nil, fmt.Errorf("failed to bake font bitmap: not enough space in atlas")
	}

	// Convert C.stbtt_bakedchar to Go BakedChar
	bakedChars := make([]BakedChar, numChars)
	for i := 0; i < numChars; i++ {
		c := &charData[i]
		bakedChars[i] = BakedChar{
			X0:       uint16(c.x0),
			Y0:       uint16(c.y0),
			X1:       uint16(c.x1),
			Y1:       uint16(c.y1),
			XOffset:  float32(c.xoff),
			YOffset:  float32(c.yoff),
			XAdvance: float32(c.xadvance),
		}
	}

	return bakedChars, atlas, nil
}

// InitFont initializes a font from TTF data
func InitFont(fontData []byte) (*FontInfo, error) {
	info := &FontInfo{
		data:   fontData,
		handle: (*C.stbtt_fontinfo)(C.malloc(C.size_t(unsafe.Sizeof(C.stbtt_fontinfo{})))),
	}

	result := C.stbtt_InitFont(
		info.handle,
		(*C.uchar)(unsafe.Pointer(&fontData[0])),
		0,
	)

	if result == 0 {
		C.free(unsafe.Pointer(info.handle))
		return nil, fmt.Errorf("failed to initialize font")
	}

	return info, nil
}

// ScaleForPixelHeight calculates the scale factor for a given pixel height
func (f *FontInfo) ScaleForPixelHeight(pixelHeight float32) float32 {
	return float32(C.stbtt_ScaleForPixelHeight(f.handle, C.float(pixelHeight)))
}

// GetFontVMetrics returns vertical metrics (ascent, descent, line gap)
func (f *FontInfo) GetFontVMetrics() (ascent, descent, lineGap int) {
	var cAscent, cDescent, cLineGap C.int
	C.stbtt_GetFontVMetrics(f.handle, &cAscent, &cDescent, &cLineGap)
	return int(cAscent), int(cDescent), int(cLineGap)
}

// GetCodepointHMetrics returns horizontal metrics for a specific codepoint
// Returns: advanceWidth, leftSideBearing
func (f *FontInfo) GetCodepointHMetrics(codepoint int) (advanceWidth, leftSideBearing int) {
	var cAdvance, cLeftBearing C.int
	C.stbtt_GetCodepointHMetrics(f.handle, C.int(codepoint), &cAdvance, &cLeftBearing)
	return int(cAdvance), int(cLeftBearing)
}

// GetCodepointSDF generates an SDF bitmap for a character
// Returns: bitmap data, width, height, xOffset, yOffset
// The returned bitmap is a Go slice (C memory is automatically freed)
func (f *FontInfo) GetCodepointSDF(scale float32, codepoint int, padding int, onedgeValue byte, pixelDistScale float32) ([]byte, int, int, int, int) {
	var width, height, xoff, yoff C.int

	cBitmap := C.stbtt_GetCodepointSDF(
		f.handle,
		C.float(scale),
		C.int(codepoint),
		C.int(padding),
		C.uchar(onedgeValue),
		C.float(pixelDistScale),
		&width,
		&height,
		&xoff,
		&yoff,
	)

	if cBitmap == nil {
		return nil, 0, 0, 0, 0
	}

	// Convert C array to Go slice and free C memory
	w := int(width)
	h := int(height)
	size := w * h
	goSlice := make([]byte, size)

	// Copy data from C to Go
	cSlice := (*[1 << 30]byte)(unsafe.Pointer(cBitmap))[:size:size]
	copy(goSlice, cSlice)

	// Free the C-allocated bitmap
	C.stbtt_FreeSDF((*C.uchar)(cBitmap), nil)

	return goSlice, w, h, int(xoff), int(yoff)
}

// Free releases the font info memory
func (f *FontInfo) Free() {
	if f.handle != nil {
		C.free(unsafe.Pointer(f.handle))
		f.handle = nil
	}
}
