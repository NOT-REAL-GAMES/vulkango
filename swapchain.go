// swapchain.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type SwapchainCreateInfoKHR struct {
	Surface            SurfaceKHR
	MinImageCount      uint32
	ImageFormat        Format
	ImageColorSpace    ColorSpaceKHR
	ImageExtent        Extent2D
	ImageArrayLayers   uint32
	ImageUsage         ImageUsageFlags
	ImageSharingMode   SharingMode
	QueueFamilyIndices []uint32
	PreTransform       SurfaceTransformFlagsKHR
	CompositeAlpha     CompositeAlphaFlagsKHR
	PresentMode        PresentModeKHR
	Clipped            bool
	OldSwapchain       SwapchainKHR
}

type SharingMode int32

const (
	SHARING_MODE_EXCLUSIVE  SharingMode = C.VK_SHARING_MODE_EXCLUSIVE
	SHARING_MODE_CONCURRENT SharingMode = C.VK_SHARING_MODE_CONCURRENT
)

type swapchainCreateData struct {
	cInfo         *C.VkSwapchainCreateInfoKHR
	queueFamilies []C.uint32_t
}

func (info *SwapchainCreateInfoKHR) vulkanize() *swapchainCreateData {
	data := &swapchainCreateData{}

	data.cInfo = (*C.VkSwapchainCreateInfoKHR)(C.calloc(1, C.sizeof_VkSwapchainCreateInfoKHR))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR
	data.cInfo.pNext = nil
	data.cInfo.flags = 0
	data.cInfo.surface = info.Surface.handle
	data.cInfo.minImageCount = C.uint32_t(info.MinImageCount)
	data.cInfo.imageFormat = C.VkFormat(info.ImageFormat)
	data.cInfo.imageColorSpace = C.VkColorSpaceKHR(info.ImageColorSpace)
	data.cInfo.imageExtent.width = C.uint32_t(info.ImageExtent.Width)
	data.cInfo.imageExtent.height = C.uint32_t(info.ImageExtent.Height)
	data.cInfo.imageArrayLayers = C.uint32_t(info.ImageArrayLayers)
	data.cInfo.imageUsage = C.VkImageUsageFlags(info.ImageUsage)
	data.cInfo.imageSharingMode = C.VkSharingMode(info.ImageSharingMode)

	// Queue family indices
	if len(info.QueueFamilyIndices) > 0 {
		data.queueFamilies = make([]C.uint32_t, len(info.QueueFamilyIndices))
		for i, idx := range info.QueueFamilyIndices {
			data.queueFamilies[i] = C.uint32_t(idx)
		}
		data.cInfo.queueFamilyIndexCount = C.uint32_t(len(data.queueFamilies))
		data.cInfo.pQueueFamilyIndices = &data.queueFamilies[0]
	} else {
		data.cInfo.queueFamilyIndexCount = 0
		data.cInfo.pQueueFamilyIndices = nil
	}

	data.cInfo.preTransform = C.VkSurfaceTransformFlagBitsKHR(info.PreTransform)
	data.cInfo.compositeAlpha = C.VkCompositeAlphaFlagBitsKHR(info.CompositeAlpha)
	data.cInfo.presentMode = C.VkPresentModeKHR(info.PresentMode)

	if info.Clipped {
		data.cInfo.clipped = C.VK_TRUE
	} else {
		data.cInfo.clipped = C.VK_FALSE
	}

	data.cInfo.oldSwapchain = info.OldSwapchain.handle

	return data
}

func (data *swapchainCreateData) free() {
	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}

func (device Device) CreateSwapchainKHR(createInfo *SwapchainCreateInfoKHR) (SwapchainKHR, error) {
	data := createInfo.vulkanize()
	defer data.free()

	var swapchain C.VkSwapchainKHR
	result := C.vkCreateSwapchainKHR(device.handle, data.cInfo, nil, &swapchain)

	if result != C.VK_SUCCESS {
		return SwapchainKHR{}, Result(result)
	}

	return SwapchainKHR{handle: swapchain}, nil
}

func (device Device) DestroySwapchainKHR(swapchain SwapchainKHR) {
	C.vkDestroySwapchainKHR(device.handle, swapchain.handle, nil)
}

func (device Device) GetSwapchainImagesKHR(swapchain SwapchainKHR) ([]Image, error) {
	var count C.uint32_t
	result := C.vkGetSwapchainImagesKHR(device.handle, swapchain.handle, &count, nil)

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	images := make([]C.VkImage, count)
	result = C.vkGetSwapchainImagesKHR(device.handle, swapchain.handle, &count, &images[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	goImages := make([]Image, count)
	for i := range goImages {
		goImages[i] = Image{handle: images[i]}
	}

	return goImages, nil
}
