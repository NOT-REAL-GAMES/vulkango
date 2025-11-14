// imageview.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type imageViewCreateData struct {
	cInfo *C.VkImageViewCreateInfo
}

func (info *ImageViewCreateInfo) vulkanize() *imageViewCreateData {
	data := &imageViewCreateData{}

	data.cInfo = (*C.VkImageViewCreateInfo)(C.calloc(1, C.sizeof_VkImageViewCreateInfo))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO
	data.cInfo.pNext = nil
	data.cInfo.flags = 0
	data.cInfo.image = info.Image.handle
	data.cInfo.viewType = C.VkImageViewType(info.ViewType)
	data.cInfo.format = C.VkFormat(info.Format)

	// Component mapping
	data.cInfo.components.r = C.VkComponentSwizzle(info.Components.R)
	data.cInfo.components.g = C.VkComponentSwizzle(info.Components.G)
	data.cInfo.components.b = C.VkComponentSwizzle(info.Components.B)
	data.cInfo.components.a = C.VkComponentSwizzle(info.Components.A)

	// Subresource range
	data.cInfo.subresourceRange.aspectMask = C.VkImageAspectFlags(info.SubresourceRange.AspectMask)
	data.cInfo.subresourceRange.baseMipLevel = C.uint32_t(info.SubresourceRange.BaseMipLevel)
	data.cInfo.subresourceRange.levelCount = C.uint32_t(info.SubresourceRange.LevelCount)
	data.cInfo.subresourceRange.baseArrayLayer = C.uint32_t(info.SubresourceRange.BaseArrayLayer)
	data.cInfo.subresourceRange.layerCount = C.uint32_t(info.SubresourceRange.LayerCount)

	return data
}

func (data *imageViewCreateData) free() {
	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}

func (device Device) CreateImageView(createInfo *ImageViewCreateInfo) (ImageView, error) {
	data := createInfo.vulkanize()
	defer data.free()

	var imageView C.VkImageView
	result := C.vkCreateImageView(device.handle, data.cInfo, nil, &imageView)

	if result != C.VK_SUCCESS {
		return ImageView{}, Result(result)
	}

	return ImageView{handle: imageView}, nil
}

func (device Device) DestroyImageView(imageView ImageView) {
	C.vkDestroyImageView(device.handle, imageView.handle, nil)
}
