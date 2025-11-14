// surface.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

func (device PhysicalDevice) GetSurfaceCapabilitiesKHR(surface SurfaceKHR) (SurfaceCapabilitiesKHR, error) {
	var caps C.VkSurfaceCapabilitiesKHR
	result := C.vkGetPhysicalDeviceSurfaceCapabilitiesKHR(device.handle, surface.handle, &caps)

	if result != C.VK_SUCCESS {
		return SurfaceCapabilitiesKHR{}, Result(result)
	}

	return SurfaceCapabilitiesKHR{
		MinImageCount:           uint32(caps.minImageCount),
		MaxImageCount:           uint32(caps.maxImageCount),
		CurrentExtent:           Extent2D{Width: uint32(caps.currentExtent.width), Height: uint32(caps.currentExtent.height)},
		MinImageExtent:          Extent2D{Width: uint32(caps.minImageExtent.width), Height: uint32(caps.minImageExtent.height)},
		MaxImageExtent:          Extent2D{Width: uint32(caps.maxImageExtent.width), Height: uint32(caps.maxImageExtent.height)},
		MaxImageArrayLayers:     uint32(caps.maxImageArrayLayers),
		SupportedTransforms:     SurfaceTransformFlagsKHR(caps.supportedTransforms),
		CurrentTransform:        SurfaceTransformFlagsKHR(caps.currentTransform),
		SupportedCompositeAlpha: CompositeAlphaFlagsKHR(caps.supportedCompositeAlpha),
		SupportedUsageFlags:     ImageUsageFlags(caps.supportedUsageFlags),
	}, nil
}

func (device PhysicalDevice) GetSurfaceFormatsKHR(surface SurfaceKHR) ([]SurfaceFormatKHR, error) {
	var count C.uint32_t
	result := C.vkGetPhysicalDeviceSurfaceFormatsKHR(device.handle, surface.handle, &count, nil)

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	formats := make([]C.VkSurfaceFormatKHR, count)
	result = C.vkGetPhysicalDeviceSurfaceFormatsKHR(device.handle, surface.handle, &count, &formats[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	goFormats := make([]SurfaceFormatKHR, count)
	for i := range goFormats {
		goFormats[i] = SurfaceFormatKHR{
			Format:     Format(formats[i].format),
			ColorSpace: ColorSpaceKHR(formats[i].colorSpace),
		}
	}

	return goFormats, nil
}

func (device PhysicalDevice) GetSurfacePresentModesKHR(surface SurfaceKHR) ([]PresentModeKHR, error) {
	var count C.uint32_t
	result := C.vkGetPhysicalDeviceSurfacePresentModesKHR(device.handle, surface.handle, &count, nil)

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	modes := make([]C.VkPresentModeKHR, count)
	result = C.vkGetPhysicalDeviceSurfacePresentModesKHR(device.handle, surface.handle, &count, &modes[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	goModes := make([]PresentModeKHR, count)
	for i := range goModes {
		goModes[i] = PresentModeKHR(modes[i])
	}

	return goModes, nil
}

// Wrap SDL's surface in our type
func NewSurfaceKHR(handle unsafe.Pointer) SurfaceKHR {
	return SurfaceKHR{handle: C.VkSurfaceKHR(handle)}
}
