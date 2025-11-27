package canvas

import (
	"fmt"
	"sync"

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
	queueMutex     *sync.Mutex // Serializes queue submissions across all threads

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
func NewSparseCanvas(cfg Config, commandPool vk.CommandPool, queue vk.Queue, queueMutex *sync.Mutex) (*SparseCanvas, error) {
	canvas := &SparseCanvas{
		device:         cfg.Device,
		physicalDevice: cfg.PhysicalDevice,
		width:          cfg.Width,
		height:         cfg.Height,
		format:         cfg.Format,
		commandPool:    commandPool,
		queue:          queue,
		queueMutex:     queueMutex,
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

	//fmt.Printf("Sparse canvas created: %dx%d, page size: %dx%d (%d bytes)\n",
	//	cfg.Width, cfg.Height, canvas.pageWidth, canvas.pageHeight, canvas.sparsePageSize)

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

// AllocateAll pre-allocates all sparse pages for the entire canvas
// This is optimized to batch all bindings in a single queue operation
func (c *SparseCanvas) AllocateAll() error {
	// Calculate number of pages
	pagesX := int((c.width + c.pageWidth - 1) / c.pageWidth)
	pagesY := int((c.height + c.pageHeight - 1) / c.pageHeight)
	//totalPages := pagesX * pagesY

	//fmt.Printf("%d: Allocating all sparse pages: %dx%d = %d pages (BATCHED)\n", time.Now().UnixMilli(), pagesX, pagesY, totalPages)

	// Count pages that need allocation
	neededPages := 0
	for py := 0; py < pagesY; py++ {
		for px := 0; px < pagesX; px++ {
			key := pageKey(px, py)
			if _, exists := c.boundPages[key]; !exists {
				neededPages++
			}
		}
	}

	if neededPages == 0 {
		//fmt.Printf("All pages already allocated\n")
		return nil
	}

	// Allocate ONE large memory block for all pages (MUCH faster!)
	memReqs := c.device.GetImageMemoryRequirements(c.image)
	totalSize := uint64(neededPages) * c.sparsePageSize
	//fmt.Printf("%d: Allocating single memory block: %d bytes for %d pages\n", time.Now().UnixMilli(), totalSize, neededPages)

	bigMemory, err := c.device.AllocateMemory(&vk.MemoryAllocateInfo{
		AllocationSize:  totalSize,
		MemoryTypeIndex: memReqs.MemoryTypeBits,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate big memory block: %w", err)
	}

	//fmt.Printf("%d: Memory allocated, now binding pages\n", time.Now().UnixMilli())

	// Collect all binds using offsets into the big memory block
	var binds []vk.SparseImageMemoryBind
	pageIndex := 0

	for py := 0; py < pagesY; py++ {
		for px := 0; px < pagesX; px++ {
			key := pageKey(px, py)
			if _, exists := c.boundPages[key]; exists {
				continue // Skip already allocated pages
			}

			// Track the memory (use big block for all pages)
			c.boundPages[key] = bigMemory

			// Create bind info for this page using offset into big memory
			bind := vk.SparseImageMemoryBind{
				Subresource: vk.ImageSubresource{
					AspectMask: vk.IMAGE_ASPECT_COLOR_BIT,
					MipLevel:   0,
					ArrayLayer: 0,
				},
				Offset: vk.Offset3D{
					X: int32(px) * int32(c.pageWidth),
					Y: int32(py) * int32(c.pageHeight),
					Z: 0,
				},
				Extent: vk.Extent3D{
					Width:  c.pageWidth,
					Height: c.pageHeight,
					Depth:  1,
				},
				Memory:       bigMemory,
				MemoryOffset: uint64(pageIndex) * c.sparsePageSize, // Offset into big block!
			}
			binds = append(binds, bind)
			pageIndex++
		}
	}

	// Submit ALL bindings in a single queue operation (MUCH faster!)
	err = c.queue.QueueBindSparse(
		[]vk.BindSparseInfo{
			{
				ImageBinds: []vk.SparseImageMemoryBindInfo{
					{
						Image: c.image,
						Binds: binds,
					},
				},
			},
		},
		vk.Fence{},
	)
	if err != nil {
		return fmt.Errorf("failed to bind sparse memory: %w", err)
	}

	//fmt.Printf("%d: All %d sparse pages allocated successfully (BATCHED)\n", time.Now().UnixMilli(), len(binds))
	return nil
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
	c.queueMutex.Lock()
	c.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	c.queueMutex.Unlock()
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
	c.queueMutex.Lock()
	c.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	c.queueMutex.Unlock()
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

// TransitionLayoutBatch transitions multiple sparse canvases to SHADER_READ_ONLY layout
func TransitionLayoutBatch(canvases []*SparseCanvas) error {
	if len(canvases) == 0 {
		return nil
	}

	// Use the first canvas's device and queue (all should be the same)
	firstCanvas := canvases[0]

	// Create one-time command buffer
	cmdBuf, err := firstCanvas.device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        firstCanvas.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	defer firstCanvas.device.FreeCommandBuffers(firstCanvas.commandPool, cmdBuf)

	cmd := cmdBuf[0]
	cmd.Begin(&vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	})

	// Transition all canvases to SHADER_READ_ONLY layout
	for _, canvas := range canvases {
		cmd.PipelineBarrier(
			vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
			vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
			0,
			[]vk.ImageMemoryBarrier{
				{
					SrcAccessMask:       0,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
					NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               canvas.image,
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
	}

	cmd.End()

	// Submit and wait
	firstCanvas.queueMutex.Lock()
	firstCanvas.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	firstCanvas.queueMutex.Unlock()
	firstCanvas.queue.WaitIdle()

	return nil
}

// ClearBatch clears multiple sparse canvases in a single GPU operation (much faster!)
func ClearBatch(canvases []*SparseCanvas, x, y, width, height uint32, r, g, b, a float32) error {
	if len(canvases) == 0 {
		return nil
	}

	// Use the first canvas's device and queue (all should be the same)
	firstCanvas := canvases[0]

	// Create one-time command buffer
	cmdBuf, err := firstCanvas.device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        firstCanvas.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	defer firstCanvas.device.FreeCommandBuffers(firstCanvas.commandPool, cmdBuf)

	cmd := cmdBuf[0]
	cmd.Begin(&vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	})

	clearColor := vk.ClearColorValue{Float32: [4]float32{r, g, b, a}}

	// Clear all canvases in a single command buffer
	for _, canvas := range canvases {
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
					Image:               canvas.image,
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

		// Clear the image
		cmd.CmdClearColorImage(
			canvas.image,
			vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
			&clearColor,
			[]vk.ImageSubresourceRange{
				{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		)

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
					Image:               canvas.image,
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
	}

	cmd.End()

	// Submit once and wait once (much faster than separate submits!)
	firstCanvas.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	firstCanvas.queue.WaitIdle()

	return nil
}

// Clear fills a region with a color using GPU clear command
func (c *SparseCanvas) Clear(x, y, width, height uint32, r, g, b, a float32) error {
	// Allocate pages that will be cleared
	startPageX := int(x / c.pageWidth)
	startPageY := int(y / c.pageHeight)
	endPageX := int((x + width - 1) / c.pageWidth)
	endPageY := int((y + height - 1) / c.pageHeight)

	for py := startPageY; py <= endPageY; py++ {
		for px := startPageX; px <= endPageX; px++ {
			if !c.IsPageAllocated(px, py) {
				if err := c.AllocatePage(px, py); err != nil {
					return fmt.Errorf("failed to allocate page (%d,%d): %w", px, py, err)
				}
			}
		}
	}

	// Create one-time command buffer
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

	// Clear the image
	clearColor := vk.ClearColorValue{Float32: [4]float32{r, g, b, a}}
	cmd.CmdClearColorImage(
		c.image,
		vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		&clearColor,
		[]vk.ImageSubresourceRange{
			{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		},
	)

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
	c.queueMutex.Lock()
	c.queue.Submit([]vk.SubmitInfo{
		{CommandBuffers: []vk.CommandBuffer{cmd}},
	}, vk.Fence{})
	c.queueMutex.Unlock()
	c.queue.WaitIdle()

	return nil
}

// Destroy releases all resources
func (c *SparseCanvas) Destroy() {
	if c.imageView != (vk.ImageView{}) {
		c.device.DestroyImageView(c.imageView)
	}
	if c.image != (vk.Image{}) {
		c.device.DestroyImage(c.image)
	}
	// Free unique memory blocks only (multiple pages may share the same memory)
	uniqueMemories := make(map[vk.DeviceMemory]bool)
	for _, memory := range c.boundPages {
		if memory != (vk.DeviceMemory{}) {
			uniqueMemories[memory] = true
		}
	}
	for memory := range uniqueMemories {
		c.device.FreeMemory(memory)
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
