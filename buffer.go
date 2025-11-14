// buffer.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

type Buffer struct {
	handle C.VkBuffer
}

type DeviceMemory struct {
	handle C.VkDeviceMemory
}

type BufferCreateInfo struct {
	Size        uint64
	Usage       BufferUsageFlags
	SharingMode SharingMode
}

type BufferUsageFlags uint32

const (
	BUFFER_USAGE_TRANSFER_SRC_BIT  BufferUsageFlags = C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT
	BUFFER_USAGE_TRANSFER_DST_BIT  BufferUsageFlags = C.VK_BUFFER_USAGE_TRANSFER_DST_BIT
	BUFFER_USAGE_VERTEX_BUFFER_BIT BufferUsageFlags = C.VK_BUFFER_USAGE_VERTEX_BUFFER_BIT
	BUFFER_USAGE_INDEX_BUFFER_BIT  BufferUsageFlags = C.VK_BUFFER_USAGE_INDEX_BUFFER_BIT
)

type MemoryRequirements struct {
	Size           uint64
	Alignment      uint64
	MemoryTypeBits uint32
}

type MemoryPropertyFlags uint32

const (
	MEMORY_PROPERTY_DEVICE_LOCAL_BIT  MemoryPropertyFlags = C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
	MEMORY_PROPERTY_HOST_VISIBLE_BIT  MemoryPropertyFlags = C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT
	MEMORY_PROPERTY_HOST_COHERENT_BIT MemoryPropertyFlags = C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
)

type MemoryAllocateInfo struct {
	AllocationSize  uint64
	MemoryTypeIndex uint32
}

func (device Device) CreateBuffer(createInfo *BufferCreateInfo) (Buffer, error) {
	cInfo := (*C.VkBufferCreateInfo)(C.calloc(1, C.sizeof_VkBufferCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0
	cInfo.size = C.VkDeviceSize(createInfo.Size)
	cInfo.usage = C.VkBufferUsageFlags(createInfo.Usage)
	cInfo.sharingMode = C.VkSharingMode(createInfo.SharingMode)

	var buffer C.VkBuffer
	result := C.vkCreateBuffer(device.handle, cInfo, nil, &buffer)

	if result != C.VK_SUCCESS {
		return Buffer{}, Result(result)
	}

	return Buffer{handle: buffer}, nil
}

func (device Device) DestroyBuffer(buffer Buffer) {
	C.vkDestroyBuffer(device.handle, buffer.handle, nil)
}

func (device Device) GetBufferMemoryRequirements(buffer Buffer) MemoryRequirements {
	var memReqs C.VkMemoryRequirements
	C.vkGetBufferMemoryRequirements(device.handle, buffer.handle, &memReqs)

	return MemoryRequirements{
		Size:           uint64(memReqs.size),
		Alignment:      uint64(memReqs.alignment),
		MemoryTypeBits: uint32(memReqs.memoryTypeBits),
	}
}

func (device Device) AllocateMemory(allocInfo *MemoryAllocateInfo) (DeviceMemory, error) {
	cInfo := (*C.VkMemoryAllocateInfo)(C.calloc(1, C.sizeof_VkMemoryAllocateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO
	cInfo.pNext = nil
	cInfo.allocationSize = C.VkDeviceSize(allocInfo.AllocationSize)
	cInfo.memoryTypeIndex = C.uint32_t(allocInfo.MemoryTypeIndex)

	var memory C.VkDeviceMemory
	result := C.vkAllocateMemory(device.handle, cInfo, nil, &memory)

	if result != C.VK_SUCCESS {
		return DeviceMemory{}, Result(result)
	}

	return DeviceMemory{handle: memory}, nil
}

func (device Device) FreeMemory(memory DeviceMemory) {
	C.vkFreeMemory(device.handle, memory.handle, nil)
}

func (device Device) BindBufferMemory(buffer Buffer, memory DeviceMemory, offset uint64) error {
	result := C.vkBindBufferMemory(device.handle, buffer.handle, memory.handle, C.VkDeviceSize(offset))
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

func (device Device) MapMemory(memory DeviceMemory, offset, size uint64) (unsafe.Pointer, error) {
	var pData unsafe.Pointer
	result := C.vkMapMemory(device.handle, memory.handle, C.VkDeviceSize(offset), C.VkDeviceSize(size), 0, &pData)

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	return pData, nil
}

func (device Device) UnmapMemory(memory DeviceMemory) {
	C.vkUnmapMemory(device.handle, memory.handle)
}

// Memory type finding helper
type PhysicalDeviceMemoryProperties struct {
	MemoryTypeCount uint32
	MemoryTypes     [32]MemoryType
	MemoryHeapCount uint32
	MemoryHeaps     [16]MemoryHeap
}

type MemoryType struct {
	PropertyFlags MemoryPropertyFlags
	HeapIndex     uint32
}

type MemoryHeap struct {
	Size  uint64
	Flags uint32
}

func (physicalDevice PhysicalDevice) GetMemoryProperties() PhysicalDeviceMemoryProperties {
	var props C.VkPhysicalDeviceMemoryProperties
	C.vkGetPhysicalDeviceMemoryProperties(physicalDevice.handle, &props)

	result := PhysicalDeviceMemoryProperties{
		MemoryTypeCount: uint32(props.memoryTypeCount),
		MemoryHeapCount: uint32(props.memoryHeapCount),
	}

	for i := uint32(0); i < result.MemoryTypeCount; i++ {
		result.MemoryTypes[i] = MemoryType{
			PropertyFlags: MemoryPropertyFlags(props.memoryTypes[i].propertyFlags),
			HeapIndex:     uint32(props.memoryTypes[i].heapIndex),
		}
	}

	for i := uint32(0); i < result.MemoryHeapCount; i++ {
		result.MemoryHeaps[i] = MemoryHeap{
			Size:  uint64(props.memoryHeaps[i].size),
			Flags: uint32(props.memoryHeaps[i].flags),
		}
	}

	return result
}

// Helper to find suitable memory type
func FindMemoryType(memProperties PhysicalDeviceMemoryProperties, typeFilter uint32, properties MemoryPropertyFlags) (uint32, bool) {
	for i := uint32(0); i < memProperties.MemoryTypeCount; i++ {
		if (typeFilter&(1<<i)) != 0 && (memProperties.MemoryTypes[i].PropertyFlags&properties) == properties {
			return i, true
		}
	}
	return 0, false
}

// Helper to create buffer with memory
func (device Device) CreateBufferWithMemory(
	size uint64,
	usage BufferUsageFlags,
	properties MemoryPropertyFlags,
	physicalDevice PhysicalDevice,
) (Buffer, DeviceMemory, error) {

	// Create buffer
	buffer, err := device.CreateBuffer(&BufferCreateInfo{
		Size:        size,
		Usage:       usage,
		SharingMode: SHARING_MODE_EXCLUSIVE,
	})
	if err != nil {
		return Buffer{}, DeviceMemory{}, err
	}

	// Get memory requirements
	memReqs := device.GetBufferMemoryRequirements(buffer)

	// Find memory type
	memProps := physicalDevice.GetMemoryProperties()
	memTypeIndex, found := FindMemoryType(memProps, memReqs.MemoryTypeBits, properties)
	if !found {
		device.DestroyBuffer(buffer)
		return Buffer{}, DeviceMemory{}, Result(C.VK_ERROR_FORMAT_NOT_SUPPORTED)
	}

	// Allocate memory
	memory, err := device.AllocateMemory(&MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	})
	if err != nil {
		device.DestroyBuffer(buffer)
		return Buffer{}, DeviceMemory{}, err
	}

	// Bind buffer to memory
	err = device.BindBufferMemory(buffer, memory, 0)
	if err != nil {
		device.FreeMemory(memory)
		device.DestroyBuffer(buffer)
		return Buffer{}, DeviceMemory{}, err
	}

	return buffer, memory, nil
}

// Upload data to buffer
func (device Device) UploadToBuffer(memory DeviceMemory, data []byte) error {
	ptr, err := device.MapMemory(memory, 0, uint64(len(data)))
	if err != nil {
		return err
	}

	// Copy data
	C.memcpy(ptr, unsafe.Pointer(&data[0]), C.size_t(len(data)))

	device.UnmapMemory(memory)
	return nil
}
