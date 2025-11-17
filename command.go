// command.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type CommandPool struct {
	handle C.VkCommandPool
}

type CommandBuffer struct {
	handle C.VkCommandBuffer
}

type CommandPoolCreateInfo struct {
	Flags            CommandPoolCreateFlags
	QueueFamilyIndex uint32
}

type CommandPoolCreateFlags uint32

const (
	COMMAND_POOL_CREATE_TRANSIENT_BIT            CommandPoolCreateFlags = C.VK_COMMAND_POOL_CREATE_TRANSIENT_BIT
	COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT CommandPoolCreateFlags = C.VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT
)

type CommandBufferAllocateInfo struct {
	CommandPool        CommandPool
	Level              CommandBufferLevel
	CommandBufferCount uint32
}

type CommandBufferLevel int32

const (
	COMMAND_BUFFER_LEVEL_PRIMARY   CommandBufferLevel = C.VK_COMMAND_BUFFER_LEVEL_PRIMARY
	COMMAND_BUFFER_LEVEL_SECONDARY CommandBufferLevel = C.VK_COMMAND_BUFFER_LEVEL_SECONDARY
)

type CommandBufferBeginInfo struct {
	Flags CommandBufferUsageFlags
}

type CommandBufferUsageFlags uint32

const (
	COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT CommandBufferUsageFlags = C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT
)

// Rendering structures for dynamic rendering
type RenderingInfo struct {
	RenderArea        Rect2D
	LayerCount        uint32
	ColorAttachments  []RenderingAttachmentInfo
	DepthAttachment   *RenderingAttachmentInfo
	StencilAttachment *RenderingAttachmentInfo
}

type RenderingAttachmentInfo struct {
	ImageView   ImageView
	ImageLayout ImageLayout
	LoadOp      AttachmentLoadOp
	StoreOp     AttachmentStoreOp
	ClearValue  ClearValue
}

type ImageLayout int32

const (
	IMAGE_LAYOUT_UNDEFINED                ImageLayout = C.VK_IMAGE_LAYOUT_UNDEFINED
	IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL ImageLayout = C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	IMAGE_LAYOUT_PRESENT_SRC_KHR          ImageLayout = C.VK_IMAGE_LAYOUT_PRESENT_SRC_KHR
)

type AttachmentLoadOp int32
type AttachmentStoreOp int32

const (
	ATTACHMENT_LOAD_OP_LOAD      AttachmentLoadOp = C.VK_ATTACHMENT_LOAD_OP_LOAD
	ATTACHMENT_LOAD_OP_CLEAR     AttachmentLoadOp = C.VK_ATTACHMENT_LOAD_OP_CLEAR
	ATTACHMENT_LOAD_OP_DONT_CARE AttachmentLoadOp = C.VK_ATTACHMENT_LOAD_OP_DONT_CARE

	ATTACHMENT_STORE_OP_STORE     AttachmentStoreOp = C.VK_ATTACHMENT_STORE_OP_STORE
	ATTACHMENT_STORE_OP_DONT_CARE AttachmentStoreOp = C.VK_ATTACHMENT_STORE_OP_DONT_CARE
)

type ClearValue struct {
	Color ClearColorValue
}

type ClearColorValue struct {
	Float32 [4]float32
}

type PipelineBindPoint int32

const (
	PIPELINE_BIND_POINT_GRAPHICS PipelineBindPoint = C.VK_PIPELINE_BIND_POINT_GRAPHICS
	PIPELINE_BIND_POINT_COMPUTE  PipelineBindPoint = C.VK_PIPELINE_BIND_POINT_COMPUTE
)

// Command Pool
func (device Device) CreateCommandPool(createInfo *CommandPoolCreateInfo) (CommandPool, error) {
	cInfo := (*C.VkCommandPoolCreateInfo)(C.calloc(1, C.sizeof_VkCommandPoolCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkCommandPoolCreateFlags(createInfo.Flags)
	cInfo.queueFamilyIndex = C.uint32_t(createInfo.QueueFamilyIndex)

	var pool C.VkCommandPool
	result := C.vkCreateCommandPool(device.handle, cInfo, nil, &pool)

	if result != C.VK_SUCCESS {
		return CommandPool{}, Result(result)
	}

	return CommandPool{handle: pool}, nil
}

func (device Device) DestroyCommandPool(pool CommandPool) {
	C.vkDestroyCommandPool(device.handle, pool.handle, nil)
}

func (device Device) ResetCommandPool(pool CommandPool, flags uint32) error {
	result := C.vkResetCommandPool(device.handle, pool.handle, C.VkCommandPoolResetFlags(flags))
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

// Command Buffer Allocation
func (device Device) AllocateCommandBuffers(allocInfo *CommandBufferAllocateInfo) ([]CommandBuffer, error) {
	cInfo := (*C.VkCommandBufferAllocateInfo)(C.calloc(1, C.sizeof_VkCommandBufferAllocateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO
	cInfo.pNext = nil
	cInfo.commandPool = allocInfo.CommandPool.handle
	cInfo.level = C.VkCommandBufferLevel(allocInfo.Level)
	cInfo.commandBufferCount = C.uint32_t(allocInfo.CommandBufferCount)

	cBuffers := make([]C.VkCommandBuffer, allocInfo.CommandBufferCount)
	result := C.vkAllocateCommandBuffers(device.handle, cInfo, &cBuffers[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	buffers := make([]CommandBuffer, allocInfo.CommandBufferCount)
	for i := range buffers {
		buffers[i] = CommandBuffer{handle: cBuffers[i]}
	}

	return buffers, nil
}

func (device Device) FreeCommandBuffers(pool CommandPool, buffers []CommandBuffer) {
	if len(buffers) == 0 {
		return
	}

	cBuffers := make([]C.VkCommandBuffer, len(buffers))
	for i, buf := range buffers {
		cBuffers[i] = buf.handle
	}

	C.vkFreeCommandBuffers(device.handle, pool.handle, C.uint32_t(len(cBuffers)), &cBuffers[0])
}

// Command Buffer Recording
func (cmd CommandBuffer) Begin(beginInfo *CommandBufferBeginInfo) error {
	cInfo := (*C.VkCommandBufferBeginInfo)(C.calloc(1, C.sizeof_VkCommandBufferBeginInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkCommandBufferUsageFlags(beginInfo.Flags)
	cInfo.pInheritanceInfo = nil

	result := C.vkBeginCommandBuffer(cmd.handle, cInfo)
	if result != C.VK_SUCCESS {
		return Result(result)
	}

	return nil
}

func (cmd CommandBuffer) End() error {
	result := C.vkEndCommandBuffer(cmd.handle)
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

func (cmd CommandBuffer) Reset(flags uint32) error {
	result := C.vkResetCommandBuffer(cmd.handle, C.VkCommandBufferResetFlags(flags))
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

// Dynamic Rendering Commands
type renderingData struct {
	cInfo            *C.VkRenderingInfo
	colorAttachments []C.VkRenderingAttachmentInfo
}

func (info *RenderingInfo) vulkanize() *renderingData {
	data := &renderingData{}

	data.cInfo = (*C.VkRenderingInfo)(C.calloc(1, C.sizeof_VkRenderingInfo))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_RENDERING_INFO
	data.cInfo.pNext = nil
	data.cInfo.flags = 0
	data.cInfo.renderArea.offset.x = C.int32_t(info.RenderArea.Offset.X)
	data.cInfo.renderArea.offset.y = C.int32_t(info.RenderArea.Offset.Y)
	data.cInfo.renderArea.extent.width = C.uint32_t(info.RenderArea.Extent.Width)
	data.cInfo.renderArea.extent.height = C.uint32_t(info.RenderArea.Extent.Height)
	data.cInfo.layerCount = C.uint32_t(info.LayerCount)
	data.cInfo.viewMask = 0

	// Color attachments
	if len(info.ColorAttachments) > 0 {
		data.colorAttachments = make([]C.VkRenderingAttachmentInfo, len(info.ColorAttachments))
		for i, att := range info.ColorAttachments {
			data.colorAttachments[i].sType = C.VK_STRUCTURE_TYPE_RENDERING_ATTACHMENT_INFO
			data.colorAttachments[i].pNext = nil
			data.colorAttachments[i].imageView = att.ImageView.handle
			data.colorAttachments[i].imageLayout = C.VkImageLayout(att.ImageLayout)
			data.colorAttachments[i].resolveMode = C.VK_RESOLVE_MODE_NONE
			data.colorAttachments[i].resolveImageView = nil
			data.colorAttachments[i].resolveImageLayout = C.VK_IMAGE_LAYOUT_UNDEFINED
			data.colorAttachments[i].loadOp = C.VkAttachmentLoadOp(att.LoadOp)
			data.colorAttachments[i].storeOp = C.VkAttachmentStoreOp(att.StoreOp)
			colorPtr := (*[4]C.float)(unsafe.Pointer(&data.colorAttachments[i].clearValue))
			colorPtr[0] = C.float(att.ClearValue.Color.Float32[0])
			colorPtr[1] = C.float(att.ClearValue.Color.Float32[1])
			colorPtr[2] = C.float(att.ClearValue.Color.Float32[2])
			colorPtr[3] = C.float(att.ClearValue.Color.Float32[3])
		}
		data.cInfo.colorAttachmentCount = C.uint32_t(len(data.colorAttachments))
		data.cInfo.pColorAttachments = &data.colorAttachments[0]
	}

	data.cInfo.pDepthAttachment = nil
	data.cInfo.pStencilAttachment = nil

	return data
}

func (data *renderingData) free() {
	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}

func (cmd CommandBuffer) BeginRendering(renderingInfo *RenderingInfo) {
	data := renderingInfo.vulkanize()
	defer data.free()

	C.vkCmdBeginRendering(cmd.handle, data.cInfo)
}

func (cmd CommandBuffer) EndRendering() {
	C.vkCmdEndRendering(cmd.handle)
}

// Pipeline Commands
func (cmd CommandBuffer) BindPipeline(bindPoint PipelineBindPoint, pipeline Pipeline) {
	C.vkCmdBindPipeline(cmd.handle, C.VkPipelineBindPoint(bindPoint), pipeline.handle)
}

func (cmd CommandBuffer) SetViewport(firstViewport uint32, viewports []Viewport) {
	cViewports := make([]C.VkViewport, len(viewports))
	for i, vp := range viewports {
		cViewports[i].x = C.float(vp.X)
		cViewports[i].y = C.float(vp.Y)
		cViewports[i].width = C.float(vp.Width)
		cViewports[i].height = C.float(vp.Height)
		cViewports[i].minDepth = C.float(vp.MinDepth)
		cViewports[i].maxDepth = C.float(vp.MaxDepth)
	}

	C.vkCmdSetViewport(cmd.handle, C.uint32_t(firstViewport), C.uint32_t(len(cViewports)), &cViewports[0])
}

func (cmd CommandBuffer) SetScissor(firstScissor uint32, scissors []Rect2D) {
	cScissors := make([]C.VkRect2D, len(scissors))
	for i, sc := range scissors {
		cScissors[i].offset.x = C.int32_t(sc.Offset.X)
		cScissors[i].offset.y = C.int32_t(sc.Offset.Y)
		cScissors[i].extent.width = C.uint32_t(sc.Extent.Width)
		cScissors[i].extent.height = C.uint32_t(sc.Extent.Height)
	}

	C.vkCmdSetScissor(cmd.handle, C.uint32_t(firstScissor), C.uint32_t(len(cScissors)), &cScissors[0])
}

// Draw Commands
func (cmd CommandBuffer) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	C.vkCmdDraw(cmd.handle, C.uint32_t(vertexCount), C.uint32_t(instanceCount),
		C.uint32_t(firstVertex), C.uint32_t(firstInstance))
}

// Image Layout Transition
type ImageMemoryBarrier struct {
	SrcAccessMask       AccessFlags
	DstAccessMask       AccessFlags
	OldLayout           ImageLayout
	NewLayout           ImageLayout
	SrcQueueFamilyIndex uint32
	DstQueueFamilyIndex uint32
	Image               Image
	SubresourceRange    ImageSubresourceRange
}

type AccessFlags uint32
type PipelineStageFlags uint32

const (
	ACCESS_NONE                       AccessFlags = 0
	ACCESS_COLOR_ATTACHMENT_WRITE_BIT AccessFlags = C.VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT

	PIPELINE_STAGE_TOP_OF_PIPE_BIT             PipelineStageFlags = C.VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT
	PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT PipelineStageFlags = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT
	PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT          PipelineStageFlags = C.VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT
)

func (cmd CommandBuffer) PipelineBarrier(
	srcStageMask, dstStageMask PipelineStageFlags,
	dependencyFlags uint32,
	imageMemoryBarriers []ImageMemoryBarrier,
) {
	var cBarriers []C.VkImageMemoryBarrier

	if len(imageMemoryBarriers) > 0 {
		cBarriers = make([]C.VkImageMemoryBarrier, len(imageMemoryBarriers))
		for i, barrier := range imageMemoryBarriers {
			cBarriers[i].sType = C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER
			cBarriers[i].pNext = nil
			cBarriers[i].srcAccessMask = C.VkAccessFlags(barrier.SrcAccessMask)
			cBarriers[i].dstAccessMask = C.VkAccessFlags(barrier.DstAccessMask)
			cBarriers[i].oldLayout = C.VkImageLayout(barrier.OldLayout)
			cBarriers[i].newLayout = C.VkImageLayout(barrier.NewLayout)
			cBarriers[i].srcQueueFamilyIndex = C.uint32_t(barrier.SrcQueueFamilyIndex)
			cBarriers[i].dstQueueFamilyIndex = C.uint32_t(barrier.DstQueueFamilyIndex)
			cBarriers[i].image = barrier.Image.handle
			cBarriers[i].subresourceRange.aspectMask = C.VkImageAspectFlags(barrier.SubresourceRange.AspectMask)
			cBarriers[i].subresourceRange.baseMipLevel = C.uint32_t(barrier.SubresourceRange.BaseMipLevel)
			cBarriers[i].subresourceRange.levelCount = C.uint32_t(barrier.SubresourceRange.LevelCount)
			cBarriers[i].subresourceRange.baseArrayLayer = C.uint32_t(barrier.SubresourceRange.BaseArrayLayer)
			cBarriers[i].subresourceRange.layerCount = C.uint32_t(barrier.SubresourceRange.LayerCount)
		}
	}

	var pImageBarriers *C.VkImageMemoryBarrier
	if len(cBarriers) > 0 {
		pImageBarriers = &cBarriers[0]
	}

	C.vkCmdPipelineBarrier(
		cmd.handle,
		C.VkPipelineStageFlags(srcStageMask),
		C.VkPipelineStageFlags(dstStageMask),
		C.VkDependencyFlags(dependencyFlags),
		0, nil,
		0, nil,
		C.uint32_t(len(cBarriers)), pImageBarriers,
	)
}

// Add to command.go

type IndexType int32

const (
	INDEX_TYPE_UINT16 IndexType = C.VK_INDEX_TYPE_UINT16
	INDEX_TYPE_UINT32 IndexType = C.VK_INDEX_TYPE_UINT32
)

func (cmd CommandBuffer) BindVertexBuffers(firstBinding uint32, buffers []Buffer, offsets []uint64) {
	cBuffers := make([]C.VkBuffer, len(buffers))
	cOffsets := make([]C.VkDeviceSize, len(offsets))

	for i, buf := range buffers {
		cBuffers[i] = buf.handle
	}
	for i, off := range offsets {
		cOffsets[i] = C.VkDeviceSize(off)
	}

	C.vkCmdBindVertexBuffers(cmd.handle, C.uint32_t(firstBinding), C.uint32_t(len(cBuffers)), &cBuffers[0], &cOffsets[0])
}

func (cmd CommandBuffer) BindIndexBuffer(buffer Buffer, offset uint64, indexType IndexType) {
	C.vkCmdBindIndexBuffer(cmd.handle, buffer.handle, C.VkDeviceSize(offset), C.VkIndexType(indexType))
}

func (cmd CommandBuffer) DrawIndexed(indexCount, instanceCount, firstIndex uint32, vertexOffset int32, firstInstance uint32) {
	C.vkCmdDrawIndexed(cmd.handle, C.uint32_t(indexCount), C.uint32_t(instanceCount),
		C.uint32_t(firstIndex), C.int32_t(vertexOffset), C.uint32_t(firstInstance))
}

// Add to command.go

// BufferCopy describes a buffer copy region
type BufferCopy struct {
	SrcOffset uint64 // Starting offset in source buffer
	DstOffset uint64 // Starting offset in destination buffer
	Size      uint64 // Number of bytes to copy
}

// Buffer to Image Copy
type BufferImageCopy struct {
	BufferOffset      uint64
	BufferRowLength   uint32
	BufferImageHeight uint32
	ImageSubresource  ImageSubresourceLayers
	ImageOffset       Offset3D
	ImageExtent       Extent3D
}

type ImageSubresourceLayers struct {
	AspectMask     ImageAspectFlags
	MipLevel       uint32
	BaseArrayLayer uint32
	LayerCount     uint32
}

type Offset3D struct {
	X int32
	Y int32
	Z int32
}

func (cmd CommandBuffer) CopyBufferToImage(srcBuffer Buffer, dstImage Image, dstImageLayout ImageLayout, regions []BufferImageCopy) {
	cRegions := make([]C.VkBufferImageCopy, len(regions))
	for i, region := range regions {
		cRegions[i].bufferOffset = C.VkDeviceSize(region.BufferOffset)
		cRegions[i].bufferRowLength = C.uint32_t(region.BufferRowLength)
		cRegions[i].bufferImageHeight = C.uint32_t(region.BufferImageHeight)
		cRegions[i].imageSubresource.aspectMask = C.VkImageAspectFlags(region.ImageSubresource.AspectMask)
		cRegions[i].imageSubresource.mipLevel = C.uint32_t(region.ImageSubresource.MipLevel)
		cRegions[i].imageSubresource.baseArrayLayer = C.uint32_t(region.ImageSubresource.BaseArrayLayer)
		cRegions[i].imageSubresource.layerCount = C.uint32_t(region.ImageSubresource.LayerCount)
		cRegions[i].imageOffset.x = C.int32_t(region.ImageOffset.X)
		cRegions[i].imageOffset.y = C.int32_t(region.ImageOffset.Y)
		cRegions[i].imageOffset.z = C.int32_t(region.ImageOffset.Z)
		cRegions[i].imageExtent.width = C.uint32_t(region.ImageExtent.Width)
		cRegions[i].imageExtent.height = C.uint32_t(region.ImageExtent.Height)
		cRegions[i].imageExtent.depth = C.uint32_t(region.ImageExtent.Depth)
	}

	C.vkCmdCopyBufferToImage(cmd.handle, srcBuffer.handle, dstImage.handle,
		C.VkImageLayout(dstImageLayout),
		C.uint32_t(len(cRegions)), &cRegions[0])
}

// Descriptor Set Binding
func (cmd CommandBuffer) BindDescriptorSets(
	pipelineBindPoint PipelineBindPoint,
	layout PipelineLayout,
	firstSet uint32,
	descriptorSets []DescriptorSet,
	dynamicOffsets []uint32,
) {
	var cSets []C.VkDescriptorSet
	if len(descriptorSets) > 0 {
		cSets = make([]C.VkDescriptorSet, len(descriptorSets))
		for i, set := range descriptorSets {
			cSets[i] = set.handle
		}
	}

	var cOffsets []C.uint32_t
	var pOffsets *C.uint32_t
	if len(dynamicOffsets) > 0 {
		cOffsets = make([]C.uint32_t, len(dynamicOffsets))
		for i, offset := range dynamicOffsets {
			cOffsets[i] = C.uint32_t(offset)
		}
		pOffsets = &cOffsets[0]
	}

	var pSets *C.VkDescriptorSet
	if len(cSets) > 0 {
		pSets = &cSets[0]
	}

	C.vkCmdBindDescriptorSets(
		cmd.handle,
		C.VkPipelineBindPoint(pipelineBindPoint),
		layout.handle,
		C.uint32_t(firstSet),
		C.uint32_t(len(cSets)),
		pSets,
		C.uint32_t(len(cOffsets)),
		pOffsets,
	)
}

// CmdPushConstants updates push constant values
func (cmd CommandBuffer) CmdPushConstants(
	layout PipelineLayout,
	stageFlags ShaderStageFlags,
	offset uint32,
	size uint32,
	pValues unsafe.Pointer,
) {
	C.vkCmdPushConstants(
		cmd.handle,
		layout.handle,
		C.VkShaderStageFlags(stageFlags),
		C.uint32_t(offset),
		C.uint32_t(size),
		pValues,
	)
}

// CmdUpdateBuffer updates a buffer's contents from host memory
// dstOffset and dataSize are in bytes
// This is useful for updating small amounts of buffer data (up to 65536 bytes)
func (cmd CommandBuffer) CmdUpdateBuffer(
	dstBuffer Buffer,
	dstOffset uint64,
	dataSize uint64,
	pData unsafe.Pointer,
) {
	C.vkCmdUpdateBuffer(
		cmd.handle,
		dstBuffer.handle,
		C.VkDeviceSize(dstOffset),
		C.VkDeviceSize(dataSize),
		pData,
	)
}

// CmdCopyBuffer copies data between buffers
// This can be used during a render pass (unlike CmdUpdateBuffer)
func (cmd CommandBuffer) CmdCopyBuffer(
	srcBuffer Buffer,
	dstBuffer Buffer,
	regions []BufferCopy,
) {
	if len(regions) == 0 {
		return
	}

	cRegions := make([]C.VkBufferCopy, len(regions))
	for i, region := range regions {
		cRegions[i] = C.VkBufferCopy{
			srcOffset: C.VkDeviceSize(region.SrcOffset),
			dstOffset: C.VkDeviceSize(region.DstOffset),
			size:      C.VkDeviceSize(region.Size),
		}
	}

	C.vkCmdCopyBuffer(
		cmd.handle,
		srcBuffer.handle,
		dstBuffer.handle,
		C.uint32_t(len(cRegions)),
		&cRegions[0],
	)
}
