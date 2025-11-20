package canvas

import (
	vk "github.com/NOT-REAL-GAMES/vulkango"
)

// PageSize is the size of each virtual memory page in pixels
// Common choices: 128x128, 256x256, or 512x512
// Smaller = more granular memory control, Larger = less overhead
const PageSize = 64

// Canvas represents a virtual texture that can be backed by either
// dense allocation (all memory allocated upfront) or sparse binding
// (memory allocated on-demand per page).
//
// This abstraction allows us to start with simple dense textures and
// later upgrade to megatexture-style sparse binding without changing
// the rest of the rendering code.
type Canvas interface {
	// GetImage returns the underlying Vulkan image
	GetImage() vk.Image

	// GetView returns the image view for sampling
	GetView() vk.ImageView

	// GetWidth returns the canvas width in pixels
	GetWidth() uint32

	// GetHeight returns the canvas height in pixels
	GetHeight() uint32

	// GetFormat returns the pixel format
	GetFormat() vk.Format

	// AllocatePage allocates physical memory for a virtual page
	// For dense canvases, this is a no-op (already allocated)
	// For sparse canvases, this binds a memory page
	AllocatePage(pageX, pageY int) error

	// DeallocatePage releases physical memory for a virtual page
	// For dense canvases, this is a no-op
	// For sparse canvases, this unbinds the memory page
	DeallocatePage(pageX, pageY int) error

	// IsPageAllocated checks if a page has physical memory bound
	IsPageAllocated(pageX, pageY int) bool

	// MarkDirty marks a region as modified (for potential optimizations)
	// x, y, width, height are in pixels
	MarkDirty(x, y, width, height uint32)

	// AllocateAll pre-allocates all sparse pages for the entire canvas
	// For dense canvases, this is a no-op (already allocated)
	// For sparse canvases, this allocates all virtual pages
	AllocateAll() error

	// Upload uploads pixel data to a region
	// data should be in the canvas's pixel format
	// This handles staging buffers and transfers
	Upload(x, y, width, height uint32, data []byte) error

	// Download reads pixel data from a region
	// Returns data in the canvas's pixel format
	// This handles staging buffers and transfers from GPU to CPU
	Download(x, y, width, height uint32) ([]byte, error)

	// Clear fills a region with a color
	Clear(x, y, width, height uint32, r, g, b, a float32) error

	// Destroy releases all resources
	Destroy()
}

// Config holds canvas creation parameters
type Config struct {
	Device         vk.Device
	PhysicalDevice vk.PhysicalDevice
	Width          uint32
	Height         uint32
	Format         vk.Format

	// UsageFlagswill typically include:
	// - TRANSFER_DST (for uploads)
	// - SAMPLED (for reading in shaders)
	// - COLOR_ATTACHMENT (for rendering to)
	Usage vk.ImageUsageFlags

	// UseSparseBinding enables megatexture mode
	// If false, creates a dense canvas (all memory allocated upfront)
	// If true, creates a sparse canvas (allocate pages on-demand)
	UseSparseBinding bool

	// InitiallyAllocateAll for sparse canvases, allocate all pages immediately
	// Useful for testing sparse pipeline without actual sparsity
	InitiallyAllocateAll bool
}

// PageCoord represents a virtual page coordinate
type PageCoord struct {
	X, Y int
}

// PixelToPage converts pixel coordinates to page coordinates
func PixelToPage(x, y uint32) PageCoord {
	return PageCoord{
		X: int(x) / PageSize,
		Y: int(y) / PageSize,
	}
}

// PageToPixel converts page coordinates to top-left pixel coordinates
func PageToPixel(pageX, pageY int) (uint32, uint32) {
	return uint32(pageX * PageSize), uint32(pageY * PageSize)
}

// GetPagesInRect returns all page coordinates that intersect a pixel rectangle
func GetPagesInRect(x, y, width, height uint32) []PageCoord {
	if width == 0 || height == 0 {
		return nil
	}

	startPage := PixelToPage(x, y)
	endPage := PixelToPage(x+width-1, y+height-1)

	var pages []PageCoord
	for py := startPage.Y; py <= endPage.Y; py++ {
		for px := startPage.X; px <= endPage.X; px++ {
			pages = append(pages, PageCoord{X: px, Y: py})
		}
	}
	return pages
}

// New creates a canvas using the appropriate implementation based on config
// This is the recommended way to create canvases.
//
// Example:
//
//	canvas, err := canvas.New(canvas.Config{
//	    Device: device,
//	    Width: 8192,
//	    Height: 8192,
//	    Format: vk.FORMAT_R8G8B8A8_UNORM,
//	    Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
//	           vk.IMAGE_USAGE_SAMPLED_BIT |
//	           vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
//	    UseSparseBinding: false, // Use dense for now
//	}, commandPool, queue)
func New(cfg Config, commandPool vk.CommandPool, queue vk.Queue) (Canvas, error) {
	if cfg.UseSparseBinding {
		return NewSparseCanvas(cfg, commandPool, queue)
	}
	return NewDenseCanvas(cfg, commandPool, queue)
}
