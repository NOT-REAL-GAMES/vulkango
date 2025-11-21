// swapchain_helper.go
package vulkango

import "fmt"

type SwapchainSupportDetails struct {
	Capabilities SurfaceCapabilitiesKHR
	Formats      []SurfaceFormatKHR
	PresentModes []PresentModeKHR
}

func (device PhysicalDevice) QuerySwapchainSupport(surface SurfaceKHR) (SwapchainSupportDetails, error) {
	var details SwapchainSupportDetails
	var err error

	details.Capabilities, err = device.GetSurfaceCapabilitiesKHR(surface)
	if err != nil {
		return details, err
	}

	details.Formats, err = device.GetSurfaceFormatsKHR(surface)
	if err != nil {
		return details, err
	}

	details.PresentModes, err = device.GetSurfacePresentModesKHR(surface)
	if err != nil {
		return details, err
	}

	return details, nil
}

func ChooseSurfaceFormat(availableFormats []SurfaceFormatKHR) SurfaceFormatKHR {
	// Prefer SRGB format
	for _, format := range availableFormats {
		if format.Format == FORMAT_B8G8R8A8_SRGB &&
			format.ColorSpace == COLOR_SPACE_SRGB_NONLINEAR_KHR {
			return format
		}
	}

	// Fallback to first available
	return availableFormats[0]
}

func ChoosePresentMode(availableModes []PresentModeKHR) PresentModeKHR {
	// Prefer mailbox (triple buffering) for lowest latency without tearing
	for _, mode := range availableModes {
		fmt.Printf("%v", mode)
		if mode == PRESENT_MODE_MAILBOX_KHR {
			return mode
		}
	}

	// FIFO is always available and is vsync
	return PRESENT_MODE_IMMEDIATE_KHR
}

func ChooseSwapExtent(capabilities SurfaceCapabilitiesKHR, windowWidth, windowHeight uint32) Extent2D {
	// If width is max uint32, we can choose our own extent
	if capabilities.CurrentExtent.Width != 0xFFFFFFFF {
		return capabilities.CurrentExtent
	}

	// Otherwise clamp to min/max
	extent := Extent2D{
		Width:  windowWidth,
		Height: windowHeight,
	}

	if extent.Width < capabilities.MinImageExtent.Width {
		extent.Width = capabilities.MinImageExtent.Width
	}
	if extent.Width > capabilities.MaxImageExtent.Width {
		extent.Width = capabilities.MaxImageExtent.Width
	}

	if extent.Height < capabilities.MinImageExtent.Height {
		extent.Height = capabilities.MinImageExtent.Height
	}
	if extent.Height > capabilities.MaxImageExtent.Height {
		extent.Height = capabilities.MaxImageExtent.Height
	}

	return extent
}

func ChooseImageCount(capabilities SurfaceCapabilitiesKHR) uint32 {
	// Request one more than minimum for better performance
	imageCount := capabilities.MinImageCount + 1

	// Don't exceed maximum (0 means no limit)
	if capabilities.MaxImageCount > 0 && imageCount > capabilities.MaxImageCount {
		imageCount = capabilities.MaxImageCount
	}

	return imageCount
}

// Convenience function to create swapchain with good defaults
func CreateSwapchain(
	device Device,
	physicalDevice PhysicalDevice,
	surface SurfaceKHR,
	windowWidth, windowHeight uint32,
	graphicsFamily uint32,
) (SwapchainKHR, Format, Extent2D, error) {

	// Query support
	support, err := physicalDevice.QuerySwapchainSupport(surface)
	if err != nil {
		return SwapchainKHR{}, 0, Extent2D{}, err
	}

	if len(support.Formats) == 0 {
		return SwapchainKHR{}, 0, Extent2D{}, fmt.Errorf("no surface formats available")
	}

	if len(support.PresentModes) == 0 {
		return SwapchainKHR{}, 0, Extent2D{}, fmt.Errorf("no present modes available")
	}

	// Choose settings
	surfaceFormat := ChooseSurfaceFormat(support.Formats)
	presentMode := ChoosePresentMode(support.PresentModes)
	extent := ChooseSwapExtent(support.Capabilities, windowWidth, windowHeight)
	imageCount := ChooseImageCount(support.Capabilities)

	// Create swapchain
	swapchain, err := device.CreateSwapchainKHR(&SwapchainCreateInfoKHR{
		Surface:          surface,
		MinImageCount:    imageCount,
		ImageFormat:      surfaceFormat.Format,
		ImageColorSpace:  surfaceFormat.ColorSpace,
		ImageExtent:      extent,
		ImageArrayLayers: 1,
		ImageUsage:       IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
		ImageSharingMode: SHARING_MODE_EXCLUSIVE,
		PreTransform:     support.Capabilities.CurrentTransform,
		CompositeAlpha:   COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
		PresentMode:      presentMode,
		Clipped:          true,
		OldSwapchain:     SwapchainKHR{},
	})

	if err != nil {
		return SwapchainKHR{}, 0, Extent2D{}, err
	}

	return swapchain, surfaceFormat.Format, extent, nil
}

// Create image views for swapchain images
func CreateSwapchainImageViews(device Device, images []Image, format Format) ([]ImageView, error) {
	imageViews := make([]ImageView, len(images))

	for i, image := range images {
		view, err := device.CreateImageView(&ImageViewCreateInfo{
			Image:    image,
			ViewType: IMAGE_VIEW_TYPE_2D,
			Format:   format,
			Components: ComponentMapping{
				R: COMPONENT_SWIZZLE_IDENTITY,
				G: COMPONENT_SWIZZLE_IDENTITY,
				B: COMPONENT_SWIZZLE_IDENTITY,
				A: COMPONENT_SWIZZLE_IDENTITY,
			},
			SubresourceRange: ImageSubresourceRange{
				AspectMask:     IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		})

		if err != nil {
			// Clean up already created views
			for j := 0; j < i; j++ {
				device.DestroyImageView(imageViews[j])
			}
			return nil, err
		}

		imageViews[i] = view
	}

	return imageViews, nil
}
