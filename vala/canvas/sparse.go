package canvas

import (
	"fmt"

	vk "github.com/NOT-REAL-GAMES/vulkango"
)

// SparseCanvas implements megatexture-style virtual texturing using
// Vulkan sparse binding. This allows massive texture sizes with
// on-demand memory allocation per page.
//
// IMPLEMENTATION STATUS: Stub/Placeholder
// This is the architectural skeleton for future implementation.
//
// To complete this implementation, you'll need to:
// 1. Check for sparse binding support in VkPhysicalDeviceFeatures
// 2. Create image with VK_IMAGE_CREATE_SPARSE_BINDING_BIT
// 3. Implement vkQueueBindSparse for page binding/unbinding
// 4. Track which pages are resident in memory
// 5. Handle page fault detection (reading unbound pages)
// 6. Implement feedback buffer for automatic page loading
// 7. Consider implementing a page cache with LRU eviction
type SparseCanvas struct {
	device         vk.Device
	physicalDevice vk.PhysicalDevice
	image          vk.Image
	imageView      vk.ImageView
	width          uint32
	height         uint32
	format         vk.Format
	commandPool    vk.CommandPool
	queue          vk.Queue

	// Page tracking
	// Map from page coordinate (x,y) to memory block
	// Key format: (pageY << 16) | pageX
	boundPages map[uint32]vk.DeviceMemory

	// Memory pool for pages
	// In a real implementation, you'd want a sophisticated allocator
	// that reuses freed pages, handles fragmentation, etc.
	pageMemories []vk.DeviceMemory

	// Page size information (from sparse memory requirements)
	pageWidth      uint32 // Page width in pixels
	pageHeight     uint32 // Page height in pixels
	sparsePageSize uint64 // Page size in bytes
}

// NewSparseCanvas creates a new sparse canvas with on-demand memory binding
func NewSparseCanvas(cfg Config, commandPool vk.CommandPool, queue vk.Queue) (*SparseCanvas, error) {
	canvas := &SparseCanvas{
		device:         cfg.Device,
		physicalDevice: cfg.PhysicalDevice,
		width:          cfg.Width,
		height:         cfg.Height,
		format:         cfg.Format,
		commandPool:    commandPool,
		queue:          queue,
		boundPages:     make(map[uint32]vk.DeviceMemory),
	}

	// Create sparse image with sparse binding flags
	image, err := cfg.Device.CreateImage(&vk.ImageCreateInfo{
		Flags:     vk.IMAGE_CREATE_SPARSE_BINDING_BIT | vk.IMAGE_CREATE_SPARSE_RESIDENCY_BIT,
		ImageType: vk.IMAGE_TYPE_2D,
		Format:    cfg.Format,
		Extent: vk.Extent3D{
			Width:  cfg.Width,
			Height: cfg.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       vk.SAMPLE_COUNT_1_BIT,
		Tiling:        vk.IMAGE_TILING_OPTIMAL,
		Usage:         cfg.Usage,
		SharingMode:   vk.SHARING_MODE_EXCLUSIVE,
		InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sparse image: %w", err)
	}
	canvas.image = image

	// Query sparse memory requirements to get page size
	sparseReqs := cfg.Device.GetImageSparseMemoryRequirements(image)
	if len(sparseReqs) == 0 {
		canvas.Destroy()
		return nil, fmt.Errorf("no sparse memory requirements returned")
	}

	// Store page size (granularity)
	granularity := sparseReqs[0].FormatProperties.ImageGranularity
	canvas.pageWidth = granularity.Width
	canvas.pageHeight = granularity.Height

	// Calculate page size in bytes (for a single page)
	bytesPerPixel := getBytesPerPixel(cfg.Format)
	canvas.sparsePageSize = uint64(granularity.Width) * uint64(granularity.Height) * uint64(bytesPerPixel)

	fmt.Printf("Sparse canvas created: %dx%d, page size: %dx%d (%d bytes)\n",
		cfg.Width, cfg.Height, canvas.pageWidth, canvas.pageHeight, canvas.sparsePageSize)

	// Create image view
	imageView, err := cfg.Device.CreateImageView(&vk.ImageViewCreateInfo{
		Image:    image,
		ViewType: vk.IMAGE_VIEW_TYPE_2D,
		Format:   cfg.Format,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	})
	if err != nil {
		canvas.Destroy()
		return nil, fmt.Errorf("failed to create image view: %w", err)
	}
	canvas.imageView = imageView

	return canvas, nil
}

// AllocatePage binds physical memory to a virtual page
func (c *SparseCanvas) AllocatePage(pageX, pageY int) error {
	// Check if already allocated
	key := pageKey(pageX, pageY)
	if _, exists := c.boundPages[key]; exists {
		return nil // Already allocated
	}

	// Allocate memory for one page
	memReqs := c.device.GetImageMemoryRequirements(c.image)
	memory, err := c.device.AllocateMemory(&vk.MemoryAllocateInfo{
		AllocationSize:  c.sparsePageSize,
		MemoryTypeIndex: memReqs.MemoryTypeBits, // Use first compatible memory type
	})
	if err != nil {
		return fmt.Errorf("failed to allocate page memory: %w", err)
	}

	// Bind the memory to the sparse image
	bind := vk.SparseImageMemoryBind{
		Subresource: vk.ImageSubresource{
			AspectMask: vk.IMAGE_ASPECT_COLOR_BIT,
			MipLevel:   0,
			ArrayLayer: 0,
		},
		Offset: vk.Offset3D{
			X: int32(pageX) * int32(c.pageWidth),
			Y: int32(pageY) * int32(c.pageHeight),
			Z: 0,
		},
		Extent: vk.Extent3D{
			Width:  c.pageWidth,
			Height: c.pageHeight,
			Depth:  1,
		},
		Memory:       memory,
		MemoryOffset: 0,
	}

	err = c.queue.QueueBindSparse([]vk.BindSparseInfo{
		{
			ImageBinds: []vk.SparseImageMemoryBindInfo{
				{
					Image: c.image,
					Binds: []vk.SparseImageMemoryBind{bind},
				},
			},
		},
	}, vk.Fence{})
	if err != nil {
		c.device.FreeMemory(memory)
		return fmt.Errorf("failed to bind sparse memory: %w", err)
	}

	// Track the binding
	c.boundPages[key] = memory
	c.pageMemories = append(c.pageMemories, memory)

	return nil
}

// DeallocatePage unbinds physical memory from a virtual page
func (c *SparseCanvas) DeallocatePage(pageX, pageY int) error {
	// Check if allocated
	key := pageKey(pageX, pageY)
	memory, exists := c.boundPages[key]
	if !exists {
		return nil // Not allocated
	}

	// Unbind the page (bind with null memory handle)
	bind := vk.SparseImageMemoryBind{
		Subresource: vk.ImageSubresource{
			AspectMask: vk.IMAGE_ASPECT_COLOR_BIT,
			MipLevel:   0,
			ArrayLayer: 0,
		},
		Offset: vk.Offset3D{
			X: int32(pageX) * int32(c.pageWidth),
			Y: int32(pageY) * int32(c.pageHeight),
			Z: 0,
		},
		Extent: vk.Extent3D{
			Width:  c.pageWidth,
			Height: c.pageHeight,
			Depth:  1,
		},
		Memory:       vk.DeviceMemory{}, // Null handle = unbind
		MemoryOffset: 0,
	}

	err := c.queue.QueueBindSparse([]vk.BindSparseInfo{
		{
			ImageBinds: []vk.SparseImageMemoryBindInfo{
				{
					Image: c.image,
					Binds: []vk.SparseImageMemoryBind{bind},
				},
			},
		},
	}, vk.Fence{})
	if err != nil {
		return fmt.Errorf("failed to unbind sparse memory: %w", err)
	}

	// Free the memory
	c.device.FreeMemory(memory)
	delete(c.boundPages, key)

	return nil
}

// IsPageAllocated checks if a page has physical memory bound
func (c *SparseCanvas) IsPageAllocated(pageX, pageY int) bool {
	if c.boundPages == nil {
		return false
	}
	_, exists := c.boundPages[pageKey(pageX, pageY)]
	return exists
}

// GetImage returns the underlying Vulkan image
func (c *SparseCanvas) GetImage() vk.Image { return c.image }

// GetView returns the image view
func (c *SparseCanvas) GetView() vk.ImageView { return c.imageView }

// GetWidth returns canvas width
func (c *SparseCanvas) GetWidth() uint32 { return c.width }

// GetHeight returns canvas height
func (c *SparseCanvas) GetHeight() uint32 { return c.height }

// GetFormat returns pixel format
func (c *SparseCanvas) GetFormat() vk.Format { return c.format }

// MarkDirty marks a region as modified
func (c *SparseCanvas) MarkDirty(x, y, width, height uint32) {
	// TODO: Track dirty pages for optimization
	// Could batch uploads, defer page binding, etc.
}

// Upload uploads pixel data to a region
func (c *SparseCanvas) Upload(x, y, width, height uint32, data []byte) error {
	// Determine which pages this upload touches
	startPageX := int(x / c.pageWidth)
	startPageY := int(y / c.pageHeight)
	endPageX := int((x + width - 1) / c.pageWidth)
	endPageY := int((y + height - 1) / c.pageHeight)

	// Allocate all touched pages
	for py := startPageY; py <= endPageY; py++ {
		for px := startPageX; px <= endPageX; px++ {
			if !c.IsPageAllocated(px, py) {
				if err := c.AllocatePage(px, py); err != nil {
					return fmt.Errorf("failed to allocate page (%d,%d): %w", px, py, err)
				}
			}
		}
	}

	// Now upload the data (same as DenseCanvas)
	bytesPerPixel := getBytesPerPixel(c.format)
	dataSize := int(width*height) * bytesPerPixel

	if len(data) < dataSize {
		return fmt.Errorf("data too small: got %d bytes, need %d bytes", len(data), dataSize)
	}

	// Create staging buffer
	stagingBuffer, stagingMemory, err := c.device.CreateBufferWithMemory(
		uint64(dataSize),
		vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		c.physicalDevice,
	)
	if err != nil {
		return fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer c.device.DestroyBuffer(stagingBuffer)
	defer c.device.FreeMemory(stagingMemory)

	// Copy data to staging buffer
	ptr, err := c.device.MapMemory(stagingMemory, 0, uint64(dataSize))
	if err != nil {
		return fmt.Errorf("failed to map staging memory: %w", err)
	}
	copy((*[1 << 30]byte)(ptr)[:dataSize:dataSize], data[:dataSize])
	c.device.UnmapMemory(stagingMemory)

	// Record and submit transfer commands
	cmdBuf, err := c.device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        c.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	defer c.device.FreeCommandBuffers(c.commandPool, cmdBuf)

	cmd := cmdBuf[0]
	cmd.Begin(&vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	})

	// Transition to TRANSFER_DST
	cmd.PipelineBarrier(
		vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		0,
		[]vk.ImageMemoryBarrier{
			{
				SrcAccessMask:       0,
				DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
				OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
				NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				SrcQueueFamilyIndex: ^uint32(0),
				DstQueueFamilyIndex: ^uint32(0),
				Image:               c.image,
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	// Copy buffer to image
	cmd.CopyBufferToImage(stagingBuffer, c.image, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{
		{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: vk.Offset3D{X: int32(x), Y: int32(y), Z: 0},
			ImageExtent: vk.Extent3D{Width: width, Height: height, Depth: 1},
		},
	})

	// Transition to SHADER_READ_ONLY
	cmd.PipelineBarrier(
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{
			{
				SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
				DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
				OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				SrcQueueFamilyIndex: ^uint32(0),
				DstQueueFamilyIndex: ^uint32(0),
				Image:               c.image,
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	cmd.End()

	// Submit and wait
	c.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	c.queue.WaitIdle()

	return nil
}

// Download reads pixel data from a region
func (c *SparseCanvas) Download(x, y, width, height uint32) ([]byte, error) {
	// Determine which pages this download touches
	startPageX := int(x / c.pageWidth)
	startPageY := int(y / c.pageHeight)
	endPageX := int((x + width - 1) / c.pageWidth)
	endPageY := int((y + height - 1) / c.pageHeight)

	// Allocate any unallocated pages (they'll be black/transparent)
	for py := startPageY; py <= endPageY; py++ {
		for px := startPageX; px <= endPageX; px++ {
			if !c.IsPageAllocated(px, py) {
				if err := c.AllocatePage(px, py); err != nil {
					return nil, fmt.Errorf("failed to allocate page (%d,%d): %w", px, py, err)
				}
			}
		}
	}

	// Download the data (same as DenseCanvas)
	bytesPerPixel := getBytesPerPixel(c.format)
	dataSize := int(width*height) * bytesPerPixel

	// Create staging buffer
	stagingBuffer, stagingMemory, err := c.device.CreateBufferWithMemory(
		uint64(dataSize),
		vk.BUFFER_USAGE_TRANSFER_DST_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		c.physicalDevice,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer c.device.DestroyBuffer(stagingBuffer)
	defer c.device.FreeMemory(stagingMemory)

	// Record and submit transfer commands
	cmdBuf, err := c.device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        c.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	defer c.device.FreeCommandBuffers(c.commandPool, cmdBuf)

	cmd := cmdBuf[0]
	cmd.Begin(&vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	})

	// Transition to TRANSFER_SRC
	cmd.PipelineBarrier(
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		0,
		[]vk.ImageMemoryBarrier{
			{
				SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
				DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
				OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				SrcQueueFamilyIndex: ^uint32(0),
				DstQueueFamilyIndex: ^uint32(0),
				Image:               c.image,
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	// Copy image to buffer
	cmd.CopyImageToBuffer(c.image, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, stagingBuffer, []vk.BufferImageCopy{
		{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: vk.ImageSubresourceLayers{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: vk.Offset3D{X: int32(x), Y: int32(y), Z: 0},
			ImageExtent: vk.Extent3D{Width: width, Height: height, Depth: 1},
		},
	})

	// Transition back to SHADER_READ_ONLY
	cmd.PipelineBarrier(
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{
			{
				SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
				DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
				OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				SrcQueueFamilyIndex: ^uint32(0),
				DstQueueFamilyIndex: ^uint32(0),
				Image:               c.image,
				SubresourceRange: vk.ImageSubresourceRange{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	cmd.End()

	// Submit and wait
	c.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	c.queue.WaitIdle()

	// Read data from staging buffer
	data := make([]byte, dataSize)
	ptr, err := c.device.MapMemory(stagingMemory, 0, uint64(dataSize))
	if err != nil {
		return nil, fmt.Errorf("failed to map staging memory: %w", err)
	}
	copy(data, (*[1 << 30]byte)(ptr)[:dataSize:dataSize])
	c.device.UnmapMemory(stagingMemory)

	return data, nil
}

// Clear fills a region with a color
func (c *SparseCanvas) Clear(x, y, width, height uint32, r, g, b, a float32) error {
	return fmt.Errorf("sparse canvas not implemented")
}

// Destroy releases all resources
func (c *SparseCanvas) Destroy() {
	if c.imageView != (vk.ImageView{}) {
		c.device.DestroyImageView(c.imageView)
	}
	if c.image != (vk.Image{}) {
		c.device.DestroyImage(c.image)
	}
	// Free all bound page memories
	for _, memory := range c.boundPages {
		if memory != (vk.DeviceMemory{}) {
			c.device.FreeMemory(memory)
		}
	}
	c.boundPages = nil
}

// pageKey converts page coordinates to a map key
func pageKey(pageX, pageY int) uint32 {
	return (uint32(pageY) << 16) | uint32(pageX)
}

// Advanced features for future implementation:
//
// 1. Page Cache with LRU eviction:
//    - Track access time per page
//    - When memory is full, evict least-recently-used pages
//    - Save evicted pages to disk for later restoration
//
// 2. Feedback Buffer (like id Tech's megatexture):
//    - Render scene to low-res buffer tracking which pages are accessed
//    - Use compute shader to analyze feedback and request pages
//    - Asynchronously load pages in background thread
//
// 3. Mip Tail handling:
//    - Lower mip levels might be required to be bound together
//    - Handle mipTailFirstLod from sparse memory requirements
//
// 4. Streaming:
//    - Background thread that loads pages from disk
//    - Double-buffering for smooth page swapping
//    - Compression (BC7, ASTC, etc.) for smaller disk footprint
//
// 5. Prefetching:
//    - Predict which pages will be needed based on brush movement
//    - Pre-load pages around cursor position
//    - Touch gesture prediction for tablet painting
