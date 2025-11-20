// image.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type Sampler struct {
	handle C.VkSampler
}

type ImageCreateInfo struct {
	Flags         ImageCreateFlags
	ImageType     ImageType
	Format        Format
	Extent        Extent3D
	MipLevels     uint32
	ArrayLayers   uint32
	Samples       SampleCountFlags
	Tiling        ImageTiling
	Usage         ImageUsageFlags
	SharingMode   SharingMode
	InitialLayout ImageLayout
}

type ImageType int32

const (
	IMAGE_TYPE_1D ImageType = C.VK_IMAGE_TYPE_1D
	IMAGE_TYPE_2D ImageType = C.VK_IMAGE_TYPE_2D
	IMAGE_TYPE_3D ImageType = C.VK_IMAGE_TYPE_3D
)

type ImageTiling int32

const (
	IMAGE_TILING_OPTIMAL ImageTiling = C.VK_IMAGE_TILING_OPTIMAL
	IMAGE_TILING_LINEAR  ImageTiling = C.VK_IMAGE_TILING_LINEAR
)

// Additional layouts
const (
	IMAGE_LAYOUT_GENERAL                  ImageLayout = C.VK_IMAGE_LAYOUT_GENERAL
	IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL ImageLayout = C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL
	IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL     ImageLayout = C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL
	IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL     ImageLayout = C.VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
)

// Additional access flags
const (
	ACCESS_TRANSFER_READ_BIT  AccessFlags = C.VK_ACCESS_TRANSFER_READ_BIT
	ACCESS_TRANSFER_WRITE_BIT AccessFlags = C.VK_ACCESS_TRANSFER_WRITE_BIT
	ACCESS_SHADER_READ_BIT    AccessFlags = C.VK_ACCESS_SHADER_READ_BIT
	ACCESS_SHADER_WRITE_BIT   AccessFlags = C.VK_ACCESS_SHADER_WRITE_BIT
)

// Additional pipeline stages
const (
	PIPELINE_STAGE_TRANSFER_BIT        PipelineStageFlags = C.VK_PIPELINE_STAGE_TRANSFER_BIT
	PIPELINE_STAGE_FRAGMENT_SHADER_BIT PipelineStageFlags = C.VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT
	PIPELINE_STAGE_COMPUTE_SHADER_BIT  PipelineStageFlags = C.VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT
)

// Image Creation
func (device Device) CreateImage(createInfo *ImageCreateInfo) (Image, error) {
	cInfo := (*C.VkImageCreateInfo)(C.calloc(1, C.sizeof_VkImageCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkImageCreateFlags(createInfo.Flags)
	cInfo.imageType = C.VkImageType(createInfo.ImageType)
	cInfo.format = C.VkFormat(createInfo.Format)
	cInfo.extent.width = C.uint32_t(createInfo.Extent.Width)
	cInfo.extent.height = C.uint32_t(createInfo.Extent.Height)
	cInfo.extent.depth = C.uint32_t(createInfo.Extent.Depth)
	cInfo.mipLevels = C.uint32_t(createInfo.MipLevels)
	cInfo.arrayLayers = C.uint32_t(createInfo.ArrayLayers)
	cInfo.samples = C.VkSampleCountFlagBits(createInfo.Samples)
	cInfo.tiling = C.VkImageTiling(createInfo.Tiling)
	cInfo.usage = C.VkImageUsageFlags(createInfo.Usage)
	cInfo.sharingMode = C.VkSharingMode(createInfo.SharingMode)
	cInfo.queueFamilyIndexCount = 0
	cInfo.pQueueFamilyIndices = nil
	cInfo.initialLayout = C.VkImageLayout(createInfo.InitialLayout)

	var image C.VkImage
	result := C.vkCreateImage(device.handle, cInfo, nil, &image)

	if result != C.VK_SUCCESS {
		return Image{}, Result(result)
	}

	return Image{handle: image}, nil
}

func (device Device) DestroyImage(image Image) {
	C.vkDestroyImage(device.handle, image.handle, nil)
}

func (device Device) GetImageMemoryRequirements(image Image) MemoryRequirements {
	var memReqs C.VkMemoryRequirements
	C.vkGetImageMemoryRequirements(device.handle, image.handle, &memReqs)

	return MemoryRequirements{
		Size:           uint64(memReqs.size),
		Alignment:      uint64(memReqs.alignment),
		MemoryTypeBits: uint32(memReqs.memoryTypeBits),
	}
}

func (device Device) BindImageMemory(image Image, memory DeviceMemory, offset uint64) error {
	result := C.vkBindImageMemory(device.handle, image.handle, memory.handle, C.VkDeviceSize(offset))
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

// Helper to create image with memory
func (device Device) CreateImageWithMemory(
	width, height uint32,
	format Format,
	tiling ImageTiling,
	usage ImageUsageFlags,
	properties MemoryPropertyFlags,
	physicalDevice PhysicalDevice,
) (Image, DeviceMemory, error) {

	image, err := device.CreateImage(&ImageCreateInfo{
		ImageType: IMAGE_TYPE_2D,
		Format:    format,
		Extent: Extent3D{
			Width:  width,
			Height: height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       SAMPLE_COUNT_1_BIT,
		Tiling:        tiling,
		Usage:         usage,
		SharingMode:   SHARING_MODE_EXCLUSIVE,
		InitialLayout: IMAGE_LAYOUT_UNDEFINED,
	})
	if err != nil {
		return Image{}, DeviceMemory{}, err
	}

	memReqs := device.GetImageMemoryRequirements(image)
	memProps := physicalDevice.GetMemoryProperties()
	memTypeIndex, found := FindMemoryType(memProps, memReqs.MemoryTypeBits, properties)
	if !found {
		device.DestroyImage(image)
		return Image{}, DeviceMemory{}, Result(C.VK_ERROR_FORMAT_NOT_SUPPORTED)
	}

	memory, err := device.AllocateMemory(&MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	})
	if err != nil {
		device.DestroyImage(image)
		return Image{}, DeviceMemory{}, err
	}

	err = device.BindImageMemory(image, memory, 0)
	if err != nil {
		device.FreeMemory(memory)
		device.DestroyImage(image)
		return Image{}, DeviceMemory{}, err
	}

	return image, memory, nil
}

// Image View (already defined earlier, but ensuring consistency)
func (device Device) CreateImageViewForTexture(image Image, format Format) (ImageView, error) {
	return device.CreateImageView(&ImageViewCreateInfo{
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
}

// Sampler
type SamplerCreateInfo struct {
	MagFilter        Filter
	MinFilter        Filter
	MipmapMode       SamplerMipmapMode
	AddressModeU     SamplerAddressMode
	AddressModeV     SamplerAddressMode
	AddressModeW     SamplerAddressMode
	MipLodBias       float32
	AnisotropyEnable bool
	MaxAnisotropy    float32
	MinLod           float32
	MaxLod           float32
	BorderColor      BorderColor
}

type Filter int32
type SamplerMipmapMode int32
type SamplerAddressMode int32
type BorderColor int32

const (
	FILTER_NEAREST Filter = C.VK_FILTER_NEAREST
	FILTER_LINEAR  Filter = C.VK_FILTER_LINEAR

	SAMPLER_MIPMAP_MODE_NEAREST SamplerMipmapMode = C.VK_SAMPLER_MIPMAP_MODE_NEAREST
	SAMPLER_MIPMAP_MODE_LINEAR  SamplerMipmapMode = C.VK_SAMPLER_MIPMAP_MODE_LINEAR

	SAMPLER_ADDRESS_MODE_REPEAT          SamplerAddressMode = C.VK_SAMPLER_ADDRESS_MODE_REPEAT
	SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT SamplerAddressMode = C.VK_SAMPLER_ADDRESS_MODE_MIRRORED_REPEAT
	SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE   SamplerAddressMode = C.VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE
	SAMPLER_ADDRESS_MODE_CLAMP_TO_BORDER SamplerAddressMode = C.VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_BORDER

	BORDER_COLOR_FLOAT_TRANSPARENT_BLACK BorderColor = C.VK_BORDER_COLOR_FLOAT_TRANSPARENT_BLACK
	BORDER_COLOR_INT_TRANSPARENT_BLACK   BorderColor = C.VK_BORDER_COLOR_INT_TRANSPARENT_BLACK
	BORDER_COLOR_FLOAT_OPAQUE_BLACK      BorderColor = C.VK_BORDER_COLOR_FLOAT_OPAQUE_BLACK
	BORDER_COLOR_INT_OPAQUE_BLACK        BorderColor = C.VK_BORDER_COLOR_INT_OPAQUE_BLACK
	BORDER_COLOR_FLOAT_OPAQUE_WHITE      BorderColor = C.VK_BORDER_COLOR_FLOAT_OPAQUE_WHITE
	BORDER_COLOR_INT_OPAQUE_WHITE        BorderColor = C.VK_BORDER_COLOR_INT_OPAQUE_WHITE
)

func (device Device) CreateSampler(createInfo *SamplerCreateInfo) (Sampler, error) {
	cInfo := (*C.VkSamplerCreateInfo)(C.calloc(1, C.sizeof_VkSamplerCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0
	cInfo.magFilter = C.VkFilter(createInfo.MagFilter)
	cInfo.minFilter = C.VkFilter(createInfo.MinFilter)
	cInfo.mipmapMode = C.VkSamplerMipmapMode(createInfo.MipmapMode)
	cInfo.addressModeU = C.VkSamplerAddressMode(createInfo.AddressModeU)
	cInfo.addressModeV = C.VkSamplerAddressMode(createInfo.AddressModeV)
	cInfo.addressModeW = C.VkSamplerAddressMode(createInfo.AddressModeW)
	cInfo.mipLodBias = C.float(createInfo.MipLodBias)

	if createInfo.AnisotropyEnable {
		cInfo.anisotropyEnable = C.VK_TRUE
	} else {
		cInfo.anisotropyEnable = C.VK_FALSE
	}
	cInfo.maxAnisotropy = C.float(createInfo.MaxAnisotropy)

	cInfo.compareEnable = C.VK_FALSE
	cInfo.compareOp = C.VK_COMPARE_OP_ALWAYS
	cInfo.minLod = C.float(createInfo.MinLod)
	cInfo.maxLod = C.float(createInfo.MaxLod)
	cInfo.borderColor = C.VkBorderColor(createInfo.BorderColor)
	cInfo.unnormalizedCoordinates = C.VK_FALSE

	var sampler C.VkSampler
	result := C.vkCreateSampler(device.handle, cInfo, nil, &sampler)

	if result != C.VK_SUCCESS {
		return Sampler{}, Result(result)
	}

	return Sampler{handle: sampler}, nil
}

func (device Device) DestroySampler(sampler Sampler) {
	C.vkDestroySampler(device.handle, sampler.handle, nil)
}
