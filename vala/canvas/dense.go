package canvas

import (
	"fmt"

	vk "github.com/NOT-REAL-GAMES/vulkango"
)

// DenseCanvas is a canvas backed by a regular Vulkan image with all
// memory allocated upfront. This is the simple, traditional approach.
//
// All pages are "allocated" from the start, so AllocatePage/DeallocatePage
// are no-ops.
type DenseCanvas struct {
	device       vk.Device
	pdevice      vk.PhysicalDevice
	image        vk.Image
	imageView    vk.ImageView
	memory       vk.DeviceMemory
	width        uint32
	height       uint32
	format       vk.Format
	commandPool  vk.CommandPool
	queue        vk.Queue
	dirtyRegions []Rect
}

// Rect represents a rectangular region in pixels
type Rect struct {
	X, Y, Width, Height uint32
}

// NewDenseCanvas creates a new dense canvas with regular texture allocation
func NewDenseCanvas(cfg Config, commandPool vk.CommandPool, queue vk.Queue) (*DenseCanvas, error) {
	canvas := &DenseCanvas{
		device:      cfg.Device,
		pdevice:     cfg.PhysicalDevice,
		width:       cfg.Width,
		height:      cfg.Height,
		format:      cfg.Format,
		commandPool: commandPool,
		queue:       queue,
	}

	// Create image with memory (using helper function)
	image, memory, err := cfg.Device.CreateImageWithMemory(
		cfg.Width,
		cfg.Height,
		cfg.Format,
		vk.IMAGE_TILING_OPTIMAL,
		cfg.Usage,
		vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
		cfg.PhysicalDevice,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create image with memory: %w", err)
	}
	canvas.image = image
	canvas.memory = memory

	// Create image view
	viewInfo := vk.ImageViewCreateInfo{
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
	}

	imageView, err := cfg.Device.CreateImageView(&viewInfo)
	if err != nil {
		canvas.Destroy()
		return nil, fmt.Errorf("failed to create image view: %w", err)
	}
	canvas.imageView = imageView

	return canvas, nil
}

func (c *DenseCanvas) GetImage() vk.Image    { return c.image }
func (c *DenseCanvas) GetView() vk.ImageView { return c.imageView }
func (c *DenseCanvas) GetWidth() uint32      { return c.width }
func (c *DenseCanvas) GetHeight() uint32     { return c.height }
func (c *DenseCanvas) GetFormat() vk.Format  { return c.format }

// AllocatePage is a no-op for dense canvases (all pages already allocated)
func (c *DenseCanvas) AllocatePage(pageX, pageY int) error {
	return nil
}

// DeallocatePage is a no-op for dense canvases
func (c *DenseCanvas) DeallocatePage(pageX, pageY int) error {
	return nil
}

// IsPageAllocated always returns true for dense canvases
func (c *DenseCanvas) IsPageAllocated(pageX, pageY int) bool {
	return true
}

// MarkDirty marks a region as modified
func (c *DenseCanvas) MarkDirty(x, y, width, height uint32) {
	c.dirtyRegions = append(c.dirtyRegions, Rect{
		X: x, Y: y, Width: width, Height: height,
	})
}

// AllocateAll is a no-op for dense canvases (already fully allocated)
func (c *DenseCanvas) AllocateAll() error {
	return nil
}

// Upload uploads pixel data to a region using a staging buffer
func (c *DenseCanvas) Upload(x, y, width, height uint32, data []byte) error {
	// Calculate expected data size based on format
	bytesPerPixel := getBytesPerPixel(c.format)
	expectedSize := int(width*height) * bytesPerPixel
	if len(data) < expectedSize {
		return fmt.Errorf("data too small: got %d bytes, need %d", len(data), expectedSize)
	}

	// Create staging buffer
	bufferInfo := vk.BufferCreateInfo{
		Size:        uint64(expectedSize),
		Usage:       vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
		SharingMode: vk.SHARING_MODE_EXCLUSIVE,
	}

	stagingBuffer, err := c.device.CreateBuffer(&bufferInfo)
	if err != nil {
		return fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer c.device.DestroyBuffer(stagingBuffer)

	// Allocate staging memory
	memReqs := c.device.GetBufferMemoryRequirements(stagingBuffer)
	memTypeIndex, err := findMemoryType(c.pdevice, memReqs.MemoryTypeBits,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT)
	if err != nil {
		return err
	}

	allocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	}

	stagingMemory, err := c.device.AllocateMemory(&allocInfo)
	if err != nil {
		return fmt.Errorf("failed to allocate staging memory: %w", err)
	}
	defer c.device.FreeMemory(stagingMemory)

	err = c.device.BindBufferMemory(stagingBuffer, stagingMemory, 0)
	if err != nil {
		return fmt.Errorf("failed to bind staging buffer memory: %w", err)
	}

	// Copy data to staging buffer
	err = c.device.UploadToBuffer(stagingMemory, data[:expectedSize])
	if err != nil {
		return fmt.Errorf("failed to upload to staging buffer: %w", err)
	}

	// Create command buffer for copy operation
	allocInfo2 := vk.CommandBufferAllocateInfo{
		CommandPool:        c.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}

	cmdBuffers, err := c.device.AllocateCommandBuffers(&allocInfo2)
	if err != nil {
		return fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	cmdBuffer := cmdBuffers[0]
	defer c.device.FreeCommandBuffers(c.commandPool, cmdBuffers)

	// Record commands
	beginInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}
	cmdBuffer.Begin(&beginInfo)

	// Transition to TRANSFER_DST_OPTIMAL
	barrier := vk.ImageMemoryBarrier{
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
		SrcAccessMask: 0,
		DstAccessMask: vk.ACCESS_TRANSFER_WRITE_BIT,
	}
	cmdBuffer.PipelineBarrier(
		vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		0,
		[]vk.ImageMemoryBarrier{barrier},
	)

	// Copy buffer to image
	region := vk.BufferImageCopy{
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
	}
	cmdBuffer.CopyBufferToImage(stagingBuffer, c.image, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{region})

	// Transition to SHADER_READ_ONLY_OPTIMAL
	barrier.OldLayout = vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	barrier.NewLayout = vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
	barrier.SrcAccessMask = vk.ACCESS_TRANSFER_WRITE_BIT
	barrier.DstAccessMask = vk.ACCESS_SHADER_READ_BIT
	cmdBuffer.PipelineBarrier(
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{barrier},
	)

	cmdBuffer.End()

	// Submit and wait
	submitInfo := vk.SubmitInfo{
		CommandBuffers: []vk.CommandBuffer{cmdBuffer},
	}
	err = c.queue.Submit([]vk.SubmitInfo{submitInfo}, vk.Fence{})
	if err != nil {
		return fmt.Errorf("failed to submit command buffer: %w", err)
	}

	c.queue.WaitIdle()
	return nil
}

// Download reads pixel data from a region using a staging buffer
func (c *DenseCanvas) Download(x, y, width, height uint32) ([]byte, error) {
	// Calculate data size based on format
	bytesPerPixel := getBytesPerPixel(c.format)
	dataSize := int(width*height) * bytesPerPixel

	// Create staging buffer for download
	bufferInfo := vk.BufferCreateInfo{
		Size:        uint64(dataSize),
		Usage:       vk.BUFFER_USAGE_TRANSFER_DST_BIT,
		SharingMode: vk.SHARING_MODE_EXCLUSIVE,
	}

	stagingBuffer, err := c.device.CreateBuffer(&bufferInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer c.device.DestroyBuffer(stagingBuffer)

	// Allocate staging memory
	memReqs := c.device.GetBufferMemoryRequirements(stagingBuffer)
	memTypeIndex, err := findMemoryType(c.pdevice, memReqs.MemoryTypeBits,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT)
	if err != nil {
		return nil, err
	}

	allocInfo := vk.MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	}

	stagingMemory, err := c.device.AllocateMemory(&allocInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate staging memory: %w", err)
	}
	defer c.device.FreeMemory(stagingMemory)

	err = c.device.BindBufferMemory(stagingBuffer, stagingMemory, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to bind staging buffer memory: %w", err)
	}

	// Create command buffer for copy operation
	allocInfo2 := vk.CommandBufferAllocateInfo{
		CommandPool:        c.commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}

	cmdBuffers, err := c.device.AllocateCommandBuffers(&allocInfo2)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate command buffer: %w", err)
	}
	cmdBuffer := cmdBuffers[0]
	defer c.device.FreeCommandBuffers(c.commandPool, cmdBuffers)

	// Record commands
	beginInfo := vk.CommandBufferBeginInfo{
		Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}
	cmdBuffer.Begin(&beginInfo)

	// Transition to TRANSFER_SRC_OPTIMAL
	barrier := vk.ImageMemoryBarrier{
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
		SrcAccessMask: vk.ACCESS_SHADER_READ_BIT,
		DstAccessMask: vk.ACCESS_TRANSFER_READ_BIT,
	}
	cmdBuffer.PipelineBarrier(
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		0,
		[]vk.ImageMemoryBarrier{barrier},
	)

	// Copy image to buffer
	region := vk.BufferImageCopy{
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
	}
	cmdBuffer.CopyImageToBuffer(c.image, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, stagingBuffer, []vk.BufferImageCopy{region})

	// Transition back to SHADER_READ_ONLY_OPTIMAL
	barrier.OldLayout = vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
	barrier.NewLayout = vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
	barrier.SrcAccessMask = vk.ACCESS_TRANSFER_READ_BIT
	barrier.DstAccessMask = vk.ACCESS_SHADER_READ_BIT
	cmdBuffer.PipelineBarrier(
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{barrier},
	)

	cmdBuffer.End()

	// Submit and wait
	submitInfo := vk.SubmitInfo{
		CommandBuffers: []vk.CommandBuffer{cmdBuffer},
	}
	err = c.queue.Submit([]vk.SubmitInfo{submitInfo}, vk.Fence{})
	if err != nil {
		return nil, fmt.Errorf("failed to submit command buffer: %w", err)
	}

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

// Clear fills a region with a solid color
func (c *DenseCanvas) Clear(x, y, width, height uint32, r, g, b, a float32) error {
	// TODO: Implement using vkCmdClearColorImage
	// For now, this is a placeholder
	return fmt.Errorf("Clear not yet implemented")
}

// Destroy releases all Vulkan resources
func (c *DenseCanvas) Destroy() {
	if c.imageView != (vk.ImageView{}) {
		c.device.DestroyImageView(c.imageView)
		c.imageView = vk.ImageView{}
	}
	if c.image != (vk.Image{}) {
		c.device.DestroyImage(c.image)
		c.image = vk.Image{}
	}
	if c.memory != (vk.DeviceMemory{}) {
		c.device.FreeMemory(c.memory)
		c.memory = vk.DeviceMemory{}
	}
}

// Helper function to find suitable memory type
func findMemoryType(device vk.PhysicalDevice, typeFilter uint32, properties vk.MemoryPropertyFlags) (uint32, error) {
	memProps := device.GetMemoryProperties()

	for i := uint32(0); i < memProps.MemoryTypeCount; i++ {
		if (typeFilter&(1<<i)) != 0 && (memProps.MemoryTypes[i].PropertyFlags&properties) == properties {
			return i, nil
		}
	}

	return 0, fmt.Errorf("failed to find suitable memory type")
}

// Helper to get bytes per pixel for a format
func getBytesPerPixel(format vk.Format) int {
	switch format {
	case vk.FORMAT_R8G8B8A8_UNORM, vk.FORMAT_R8G8B8A8_SRGB,
		vk.FORMAT_B8G8R8A8_UNORM, vk.FORMAT_B8G8R8A8_SRGB:
		return 4
	case vk.FORMAT_R8G8B8_UNORM, vk.FORMAT_R8G8B8_SRGB:
		return 3
	case vk.FORMAT_R8_UNORM:
		return 1
	case vk.FORMAT_R16G16B16A16_SFLOAT:
		return 8
	case vk.FORMAT_R32G32B32A32_SFLOAT:
		return 16
	default:
		return 4 // Assume 4 bytes for unknown formats
	}
}
