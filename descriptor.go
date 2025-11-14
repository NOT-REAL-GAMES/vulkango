// descriptor.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type DescriptorPool struct {
	handle C.VkDescriptorPool
}

type DescriptorSet struct {
	handle C.VkDescriptorSet
}

// Descriptor Set Layout
type DescriptorSetLayoutCreateInfo struct {
	Bindings []DescriptorSetLayoutBinding
}

type DescriptorSetLayoutBinding struct {
	Binding         uint32
	DescriptorType  DescriptorType
	DescriptorCount uint32
	StageFlags      ShaderStageFlags
}

type DescriptorType int32

const (
	DESCRIPTOR_TYPE_SAMPLER                DescriptorType = C.VK_DESCRIPTOR_TYPE_SAMPLER
	DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER DescriptorType = C.VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER
	DESCRIPTOR_TYPE_SAMPLED_IMAGE          DescriptorType = C.VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE
	DESCRIPTOR_TYPE_STORAGE_IMAGE          DescriptorType = C.VK_DESCRIPTOR_TYPE_STORAGE_IMAGE
	DESCRIPTOR_TYPE_UNIFORM_BUFFER         DescriptorType = C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
	DESCRIPTOR_TYPE_STORAGE_BUFFER         DescriptorType = C.VK_DESCRIPTOR_TYPE_STORAGE_BUFFER
)

func (device Device) CreateDescriptorSetLayout(createInfo *DescriptorSetLayoutCreateInfo) (DescriptorSetLayout, error) {
	cInfo := (*C.VkDescriptorSetLayoutCreateInfo)(C.calloc(1, C.sizeof_VkDescriptorSetLayoutCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0

	var bindings []C.VkDescriptorSetLayoutBinding
	if len(createInfo.Bindings) > 0 {
		bindings = make([]C.VkDescriptorSetLayoutBinding, len(createInfo.Bindings))
		for i, binding := range createInfo.Bindings {
			bindings[i].binding = C.uint32_t(binding.Binding)
			bindings[i].descriptorType = C.VkDescriptorType(binding.DescriptorType)
			bindings[i].descriptorCount = C.uint32_t(binding.DescriptorCount)
			bindings[i].stageFlags = C.VkShaderStageFlags(binding.StageFlags)
			bindings[i].pImmutableSamplers = nil
		}
		cInfo.bindingCount = C.uint32_t(len(bindings))
		cInfo.pBindings = &bindings[0]
	}

	var layout C.VkDescriptorSetLayout
	result := C.vkCreateDescriptorSetLayout(device.handle, cInfo, nil, &layout)

	if result != C.VK_SUCCESS {
		return DescriptorSetLayout{}, Result(result)
	}

	return DescriptorSetLayout{handle: layout}, nil
}

func (device Device) DestroyDescriptorSetLayout(layout DescriptorSetLayout) {
	C.vkDestroyDescriptorSetLayout(device.handle, layout.handle, nil)
}

// Descriptor Pool
type DescriptorPoolCreateInfo struct {
	MaxSets   uint32
	PoolSizes []DescriptorPoolSize
}

type DescriptorPoolSize struct {
	Type            DescriptorType
	DescriptorCount uint32
}

func (device Device) CreateDescriptorPool(createInfo *DescriptorPoolCreateInfo) (DescriptorPool, error) {
	cInfo := (*C.VkDescriptorPoolCreateInfo)(C.calloc(1, C.sizeof_VkDescriptorPoolCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0
	cInfo.maxSets = C.uint32_t(createInfo.MaxSets)

	var poolSizes []C.VkDescriptorPoolSize
	if len(createInfo.PoolSizes) > 0 {
		poolSizes = make([]C.VkDescriptorPoolSize, len(createInfo.PoolSizes))
		for i, size := range createInfo.PoolSizes {
			poolSizes[i]._type = C.VkDescriptorType(size.Type)
			poolSizes[i].descriptorCount = C.uint32_t(size.DescriptorCount)
		}
		cInfo.poolSizeCount = C.uint32_t(len(poolSizes))
		cInfo.pPoolSizes = &poolSizes[0]
	}

	var pool C.VkDescriptorPool
	result := C.vkCreateDescriptorPool(device.handle, cInfo, nil, &pool)

	if result != C.VK_SUCCESS {
		return DescriptorPool{}, Result(result)
	}

	return DescriptorPool{handle: pool}, nil
}

func (device Device) DestroyDescriptorPool(pool DescriptorPool) {
	C.vkDestroyDescriptorPool(device.handle, pool.handle, nil)
}

// Descriptor Set Allocation
type DescriptorSetAllocateInfo struct {
	DescriptorPool DescriptorPool
	SetLayouts     []DescriptorSetLayout
}

func (device Device) AllocateDescriptorSets(allocInfo *DescriptorSetAllocateInfo) ([]DescriptorSet, error) {
	cInfo := (*C.VkDescriptorSetAllocateInfo)(C.calloc(1, C.sizeof_VkDescriptorSetAllocateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO
	cInfo.pNext = nil
	cInfo.descriptorPool = allocInfo.DescriptorPool.handle

	var layouts []C.VkDescriptorSetLayout
	if len(allocInfo.SetLayouts) > 0 {
		layouts = make([]C.VkDescriptorSetLayout, len(allocInfo.SetLayouts))
		for i, layout := range allocInfo.SetLayouts {
			layouts[i] = layout.handle
		}
		cInfo.descriptorSetCount = C.uint32_t(len(layouts))
		cInfo.pSetLayouts = &layouts[0]
	}

	sets := make([]C.VkDescriptorSet, len(allocInfo.SetLayouts))
	result := C.vkAllocateDescriptorSets(device.handle, cInfo, &sets[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	descriptorSets := make([]DescriptorSet, len(sets))
	for i, set := range sets {
		descriptorSets[i] = DescriptorSet{handle: set}
	}

	return descriptorSets, nil
}

// Descriptor Set Updates
type WriteDescriptorSet struct {
	DstSet          DescriptorSet
	DstBinding      uint32
	DstArrayElement uint32
	DescriptorType  DescriptorType
	ImageInfo       []DescriptorImageInfo
	BufferInfo      []DescriptorBufferInfo
}

type DescriptorImageInfo struct {
	Sampler     Sampler
	ImageView   ImageView
	ImageLayout ImageLayout
}

type DescriptorBufferInfo struct {
	Buffer Buffer
	Offset uint64
	Range  uint64
}

func (device Device) UpdateDescriptorSets(writes []WriteDescriptorSet) {
	if len(writes) == 0 {
		return
	}

	// Allocate C memory for writes
	cWrites := (*[1 << 30]C.VkWriteDescriptorSet)(C.calloc(C.size_t(len(writes)), C.sizeof_VkWriteDescriptorSet))[:len(writes):len(writes)]
	defer C.free(unsafe.Pointer(&cWrites[0]))

	// Track allocations for cleanup
	var imageInfos [][]C.VkDescriptorImageInfo
	var bufferInfos [][]C.VkDescriptorBufferInfo

	for i, write := range writes {
		cWrites[i].sType = C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET
		cWrites[i].pNext = nil
		cWrites[i].dstSet = write.DstSet.handle
		cWrites[i].dstBinding = C.uint32_t(write.DstBinding)
		cWrites[i].dstArrayElement = C.uint32_t(write.DstArrayElement)
		cWrites[i].descriptorType = C.VkDescriptorType(write.DescriptorType)

		// Image info
		if len(write.ImageInfo) > 0 {
			imgInfo := (*[1 << 30]C.VkDescriptorImageInfo)(C.calloc(C.size_t(len(write.ImageInfo)), C.sizeof_VkDescriptorImageInfo))[:len(write.ImageInfo):len(write.ImageInfo)]
			for j, info := range write.ImageInfo {
				imgInfo[j].sampler = info.Sampler.handle
				imgInfo[j].imageView = info.ImageView.handle
				imgInfo[j].imageLayout = C.VkImageLayout(info.ImageLayout)
			}
			imageInfos = append(imageInfos, imgInfo)

			cWrites[i].descriptorCount = C.uint32_t(len(imgInfo))
			cWrites[i].pImageInfo = &imgInfo[0]
			cWrites[i].pBufferInfo = nil
			cWrites[i].pTexelBufferView = nil
		}

		// Buffer info
		if len(write.BufferInfo) > 0 {
			bufInfo := (*[1 << 30]C.VkDescriptorBufferInfo)(C.calloc(C.size_t(len(write.BufferInfo)), C.sizeof_VkDescriptorBufferInfo))[:len(write.BufferInfo):len(write.BufferInfo)]
			for j, info := range write.BufferInfo {
				bufInfo[j].buffer = info.Buffer.handle
				bufInfo[j].offset = C.VkDeviceSize(info.Offset)
				bufInfo[j]._range = C.VkDeviceSize(info.Range)
			}
			bufferInfos = append(bufferInfos, bufInfo)

			cWrites[i].descriptorCount = C.uint32_t(len(bufInfo))
			cWrites[i].pImageInfo = nil
			cWrites[i].pBufferInfo = &bufInfo[0]
			cWrites[i].pTexelBufferView = nil
		}
	}

	C.vkUpdateDescriptorSets(device.handle, C.uint32_t(len(cWrites)), &cWrites[0], 0, nil)

	// Cleanup allocated memory
	for _, imgInfo := range imageInfos {
		C.free(unsafe.Pointer(&imgInfo[0]))
	}
	for _, bufInfo := range bufferInfos {
		C.free(unsafe.Pointer(&bufInfo[0]))
	}
}
