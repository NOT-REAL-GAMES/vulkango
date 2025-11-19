// sparse.go - Vulkan sparse binding operations
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// SparseImageMemoryRequirements describes sparse memory requirements for an image
type SparseImageMemoryRequirements struct {
	FormatProperties    SparseImageFormatProperties
	ImageMipTailFirstLod uint32
	ImageMipTailSize     uint64
	ImageMipTailOffset   uint64
	ImageMipTailStride   uint64
}

// SparseImageFormatProperties describes sparse image format properties
type SparseImageFormatProperties struct {
	AspectMask       ImageAspectFlags
	ImageGranularity Extent3D
	Flags            SparseImageFormatFlags
}

type SparseImageFormatFlags uint32

// GetImageSparseMemoryRequirements queries sparse memory requirements for an image
func (device Device) GetImageSparseMemoryRequirements(image Image) []SparseImageMemoryRequirements {
	var count C.uint32_t
	C.vkGetImageSparseMemoryRequirements(device.handle, image.handle, &count, nil)

	if count == 0 {
		return nil
	}

	cReqs := make([]C.VkSparseImageMemoryRequirements, count)
	C.vkGetImageSparseMemoryRequirements(device.handle, image.handle, &count, &cReqs[0])

	reqs := make([]SparseImageMemoryRequirements, count)
	for i := range reqs {
		reqs[i] = SparseImageMemoryRequirements{
			FormatProperties: SparseImageFormatProperties{
				AspectMask: ImageAspectFlags(cReqs[i].formatProperties.aspectMask),
				ImageGranularity: Extent3D{
					Width:  uint32(cReqs[i].formatProperties.imageGranularity.width),
					Height: uint32(cReqs[i].formatProperties.imageGranularity.height),
					Depth:  uint32(cReqs[i].formatProperties.imageGranularity.depth),
				},
				Flags: SparseImageFormatFlags(cReqs[i].formatProperties.flags),
			},
			ImageMipTailFirstLod: uint32(cReqs[i].imageMipTailFirstLod),
			ImageMipTailSize:     uint64(cReqs[i].imageMipTailSize),
			ImageMipTailOffset:   uint64(cReqs[i].imageMipTailOffset),
			ImageMipTailStride:   uint64(cReqs[i].imageMipTailStride),
		}
	}

	return reqs
}

// SparseImageMemoryBind describes a sparse image memory binding operation
type SparseImageMemoryBind struct {
	Subresource  ImageSubresource
	Offset       Offset3D
	Extent       Extent3D
	Memory       DeviceMemory
	MemoryOffset uint64
}

// ImageSubresource specifies an image subresource
type ImageSubresource struct {
	AspectMask ImageAspectFlags
	MipLevel   uint32
	ArrayLayer uint32
}

// SparseImageMemoryBindInfo contains sparse image memory bindings
type SparseImageMemoryBindInfo struct {
	Image Image
	Binds []SparseImageMemoryBind
}

// BindSparseInfo describes a sparse binding operation
type BindSparseInfo struct {
	ImageBinds []SparseImageMemoryBindInfo
}

// QueueBindSparse binds device memory to sparse resources
func (queue Queue) QueueBindSparse(bindInfos []BindSparseInfo, fence Fence) error {
	if len(bindInfos) == 0 {
		return nil
	}

	// Use C.malloc for all allocations to avoid CGo pointer restrictions
	cBindInfos := (*C.VkBindSparseInfo)(C.calloc(C.size_t(len(bindInfos)), C.sizeof_VkBindSparseInfo))
	defer C.free(unsafe.Pointer(cBindInfos))

	// Track C allocations for cleanup
	var cImageBindArrays []*C.VkSparseImageMemoryBindInfo
	var cMemBindArrays []*C.VkSparseImageMemoryBind

	// Convert to slice for easier indexing
	bindInfoSlice := (*[1 << 30]C.VkBindSparseInfo)(unsafe.Pointer(cBindInfos))[:len(bindInfos):len(bindInfos)]

	for i, info := range bindInfos {
		bindInfoSlice[i].sType = C.VK_STRUCTURE_TYPE_BIND_SPARSE_INFO
		bindInfoSlice[i].pNext = nil
		bindInfoSlice[i].waitSemaphoreCount = 0
		bindInfoSlice[i].pWaitSemaphores = nil
		bindInfoSlice[i].bufferBindCount = 0
		bindInfoSlice[i].pBufferBinds = nil
		bindInfoSlice[i].signalSemaphoreCount = 0
		bindInfoSlice[i].pSignalSemaphores = nil

		// Handle image binds
		if len(info.ImageBinds) > 0 {
			imageBinds := (*C.VkSparseImageMemoryBindInfo)(C.calloc(C.size_t(len(info.ImageBinds)), C.sizeof_VkSparseImageMemoryBindInfo))
			cImageBindArrays = append(cImageBindArrays, imageBinds)
			imageBindSlice := (*[1 << 30]C.VkSparseImageMemoryBindInfo)(unsafe.Pointer(imageBinds))[:len(info.ImageBinds):len(info.ImageBinds)]

			for j, imageBind := range info.ImageBinds {
				imageBindSlice[j].image = imageBind.Image.handle
				imageBindSlice[j].bindCount = C.uint32_t(len(imageBind.Binds))

				if len(imageBind.Binds) > 0 {
					binds := (*C.VkSparseImageMemoryBind)(C.calloc(C.size_t(len(imageBind.Binds)), C.sizeof_VkSparseImageMemoryBind))
					cMemBindArrays = append(cMemBindArrays, binds)
					bindSlice := (*[1 << 30]C.VkSparseImageMemoryBind)(unsafe.Pointer(binds))[:len(imageBind.Binds):len(imageBind.Binds)]

					for k, bind := range imageBind.Binds {
						bindSlice[k].subresource.aspectMask = C.VkImageAspectFlags(bind.Subresource.AspectMask)
						bindSlice[k].subresource.mipLevel = C.uint32_t(bind.Subresource.MipLevel)
						bindSlice[k].subresource.arrayLayer = C.uint32_t(bind.Subresource.ArrayLayer)
						bindSlice[k].offset.x = C.int32_t(bind.Offset.X)
						bindSlice[k].offset.y = C.int32_t(bind.Offset.Y)
						bindSlice[k].offset.z = C.int32_t(bind.Offset.Z)
						bindSlice[k].extent.width = C.uint32_t(bind.Extent.Width)
						bindSlice[k].extent.height = C.uint32_t(bind.Extent.Height)
						bindSlice[k].extent.depth = C.uint32_t(bind.Extent.Depth)
						bindSlice[k].memory = bind.Memory.handle
						bindSlice[k].memoryOffset = C.VkDeviceSize(bind.MemoryOffset)
						bindSlice[k].flags = 0
					}
					imageBindSlice[j].pBinds = binds
				}
			}

			bindInfoSlice[i].imageBindCount = C.uint32_t(len(info.ImageBinds))
			bindInfoSlice[i].pImageBinds = imageBinds
		}
	}

	// Call vkQueueBindSparse
	var cFence C.VkFence
	if fence.handle != nil {
		cFence = fence.handle
	}

	result := C.vkQueueBindSparse(queue.handle, C.uint32_t(len(bindInfos)), cBindInfos, cFence)

	// Free all C allocations
	for _, ptr := range cImageBindArrays {
		C.free(unsafe.Pointer(ptr))
	}
	for _, ptr := range cMemBindArrays {
		C.free(unsafe.Pointer(ptr))
	}

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	return nil
}
