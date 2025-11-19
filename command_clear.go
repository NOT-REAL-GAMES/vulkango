package vulkango

/*
#include <vulkan/vulkan.h>
*/
import "C"
import "unsafe"

// CmdClearColorImage fills an image with a constant color value
func (cmd CommandBuffer) CmdClearColorImage(
	image Image,
	imageLayout ImageLayout,
	color *ClearColorValue,
	ranges []ImageSubresourceRange,
) {
	var cRanges []C.VkImageSubresourceRange
	for _, r := range ranges {
		cRanges = append(cRanges, C.VkImageSubresourceRange{
			aspectMask:     C.VkImageAspectFlags(r.AspectMask),
			baseMipLevel:   C.uint32_t(r.BaseMipLevel),
			levelCount:     C.uint32_t(r.LevelCount),
			baseArrayLayer: C.uint32_t(r.BaseArrayLayer),
			layerCount:     C.uint32_t(r.LayerCount),
		})
	}

	var cRangesPtr *C.VkImageSubresourceRange
	if len(cRanges) > 0 {
		cRangesPtr = &cRanges[0]
	}

	C.vkCmdClearColorImage(
		cmd.handle,
		image.handle,
		C.VkImageLayout(imageLayout),
		(*C.VkClearColorValue)(unsafe.Pointer(color)),
		C.uint32_t(len(ranges)),
		cRangesPtr,
	)
}
