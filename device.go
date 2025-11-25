// device.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

func (physicalDevice PhysicalDevice) GetQueueFamilyProperties() []QueueFamilyProperties {
	var count C.uint32_t
	C.vkGetPhysicalDeviceQueueFamilyProperties(physicalDevice.handle, &count, nil)

	if count == 0 {
		return nil
	}

	props := make([]C.VkQueueFamilyProperties, count)
	C.vkGetPhysicalDeviceQueueFamilyProperties(physicalDevice.handle, &count, &props[0])

	goProps := make([]QueueFamilyProperties, count)
	for i := range goProps {
		goProps[i] = QueueFamilyProperties{
			QueueFlags:         QueueFlags(props[i].queueFlags),
			QueueCount:         uint32(props[i].queueCount),
			TimestampValidBits: uint32(props[i].timestampValidBits),
			MinImageTransferGranularity: Extent3D{
				Width:  uint32(props[i].minImageTransferGranularity.width),
				Height: uint32(props[i].minImageTransferGranularity.height),
				Depth:  uint32(props[i].minImageTransferGranularity.depth),
			},
		}
	}

	return goProps
}

func (physicalDevice PhysicalDevice) GetFeatures() PhysicalDeviceFeatures {
	var cFeatures C.VkPhysicalDeviceFeatures
	C.vkGetPhysicalDeviceFeatures(physicalDevice.handle, &cFeatures)

	return PhysicalDeviceFeatures{
		SparseBinding:          cFeatures.sparseBinding == C.VK_TRUE,
		SparseResidencyImage2D: cFeatures.sparseResidencyImage2D == C.VK_TRUE,
	}
}

func (physicalDevice PhysicalDevice) GetSurfaceSupportKHR(queueFamilyIndex uint32, surface SurfaceKHR) (bool, error) {
	var supported C.VkBool32
	result := C.vkGetPhysicalDeviceSurfaceSupportKHR(
		physicalDevice.handle,
		C.uint32_t(queueFamilyIndex),
		surface.handle,
		&supported,
	)

	if result != C.VK_SUCCESS {
		return false, Result(result)
	}

	return supported == C.VK_TRUE, nil
}

type deviceCreateData struct {
	cInfo            *C.VkDeviceCreateInfo
	queueCreateInfos []C.VkDeviceQueueCreateInfo
	queuePriorities  [][]C.float
	layers           []*C.char
	extensions       []*C.char
	features         *C.VkPhysicalDeviceFeatures
	features12       *C.VkPhysicalDeviceVulkan12Features
	features13       *C.VkPhysicalDeviceVulkan13Features
}

func (info *DeviceCreateInfo) vulkanize() *deviceCreateData {
	data := &deviceCreateData{}

	data.cInfo = (*C.VkDeviceCreateInfo)(C.calloc(1, C.sizeof_VkDeviceCreateInfo))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO
	data.cInfo.pNext = nil

	// Queue create infos
	if len(info.QueueCreateInfos) > 0 {
		data.queueCreateInfos = make([]C.VkDeviceQueueCreateInfo, len(info.QueueCreateInfos))
		data.queuePriorities = make([][]C.float, len(info.QueueCreateInfos))

		for i, queueInfo := range info.QueueCreateInfos {
			data.queueCreateInfos[i].sType = C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO
			data.queueCreateInfos[i].pNext = nil
			data.queueCreateInfos[i].flags = 0
			data.queueCreateInfos[i].queueFamilyIndex = C.uint32_t(queueInfo.QueueFamilyIndex)
			data.queueCreateInfos[i].queueCount = C.uint32_t(len(queueInfo.QueuePriorities))

			// Convert priorities
			data.queuePriorities[i] = make([]C.float, len(queueInfo.QueuePriorities))
			for j, priority := range queueInfo.QueuePriorities {
				data.queuePriorities[i][j] = C.float(priority)
			}
			data.queueCreateInfos[i].pQueuePriorities = &data.queuePriorities[i][0]
		}

		data.cInfo.queueCreateInfoCount = C.uint32_t(len(data.queueCreateInfos))
		data.cInfo.pQueueCreateInfos = &data.queueCreateInfos[0]
	}

	// Layers
	if len(info.EnabledLayerNames) > 0 {
		data.layers = make([]*C.char, len(info.EnabledLayerNames))
		for i, layer := range info.EnabledLayerNames {
			data.layers[i] = C.CString(layer)
		}
		data.cInfo.enabledLayerCount = C.uint32_t(len(data.layers))
		data.cInfo.ppEnabledLayerNames = &data.layers[0]
	}

	// Extensions
	if len(info.EnabledExtensionNames) > 0 {
		data.extensions = make([]*C.char, len(info.EnabledExtensionNames))
		for i, ext := range info.EnabledExtensionNames {
			data.extensions[i] = C.CString(ext)
		}
		data.cInfo.enabledExtensionCount = C.uint32_t(len(data.extensions))
		data.cInfo.ppEnabledExtensionNames = &data.extensions[0]
	}

	// Chain feature structures if needed
	var pNext unsafe.Pointer = nil

	// Setup Vulkan 1.3 features
	if info.Vulkan13Features != nil && info.Vulkan13Features.DynamicRendering {
		data.features13 = (*C.VkPhysicalDeviceVulkan13Features)(C.calloc(1, C.sizeof_VkPhysicalDeviceVulkan13Features))
		data.features13.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_3_FEATURES
		data.features13.pNext = pNext
		data.features13.dynamicRendering = C.VK_TRUE
		pNext = unsafe.Pointer(data.features13)
	}

	// Setup Vulkan 1.2 features (descriptor indexing)
	if info.Vulkan12Features != nil {
		data.features12 = (*C.VkPhysicalDeviceVulkan12Features)(C.calloc(1, C.sizeof_VkPhysicalDeviceVulkan12Features))
		data.features12.sType = C.VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_2_FEATURES
		data.features12.pNext = pNext

		if info.Vulkan12Features.DescriptorIndexing {
			data.features12.descriptorIndexing = C.VK_TRUE
		}
		if info.Vulkan12Features.ShaderSampledImageArrayNonUniformIndexing {
			data.features12.shaderSampledImageArrayNonUniformIndexing = C.VK_TRUE
		}
		if info.Vulkan12Features.DescriptorBindingPartiallyBound {
			data.features12.descriptorBindingPartiallyBound = C.VK_TRUE
		}
		if info.Vulkan12Features.DescriptorBindingUpdateAfterBind {
			data.features12.descriptorBindingSampledImageUpdateAfterBind = C.VK_TRUE
		}
		if info.Vulkan12Features.RuntimeDescriptorArray {
			data.features12.runtimeDescriptorArray = C.VK_TRUE
		}

		pNext = unsafe.Pointer(data.features12)
	}

	data.cInfo.pNext = pNext

	// Setup basic features
	if info.EnabledFeatures != nil {
		data.features = (*C.VkPhysicalDeviceFeatures)(C.calloc(1, C.sizeof_VkPhysicalDeviceFeatures))

		if info.EnabledFeatures.SparseBinding {
			data.features.sparseBinding = C.VK_TRUE
		}
		if info.EnabledFeatures.SparseResidencyImage2D {
			data.features.sparseResidencyImage2D = C.VK_TRUE
		}

		data.cInfo.pEnabledFeatures = data.features
	} else {
		data.cInfo.pEnabledFeatures = nil
	}

	return data
}

func (data *deviceCreateData) free() {
	for _, layer := range data.layers {
		C.free(unsafe.Pointer(layer))
	}

	for _, ext := range data.extensions {
		C.free(unsafe.Pointer(ext))
	}

	if data.features != nil {
		C.free(unsafe.Pointer(data.features))
	}

	if data.features12 != nil {
		C.free(unsafe.Pointer(data.features12))
	}

	if data.features13 != nil {
		C.free(unsafe.Pointer(data.features13))
	}

	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}

func (physicalDevice PhysicalDevice) CreateDevice(createInfo *DeviceCreateInfo) (Device, error) {
	data := createInfo.vulkanize()
	defer data.free()

	var device C.VkDevice
	result := C.vkCreateDevice(physicalDevice.handle, data.cInfo, nil, &device)

	if result != C.VK_SUCCESS {
		return Device{}, Result(result)
	}

	return Device{handle: device}, nil
}

func (device Device) Destroy() {
	C.vkDestroyDevice(device.handle, nil)
}

func (device Device) WaitIdle() error {
	result := C.vkDeviceWaitIdle(device.handle)
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

func (device Device) GetQueue(queueFamilyIndex, queueIndex uint32) Queue {
	var queue C.VkQueue
	C.vkGetDeviceQueue(device.handle, C.uint32_t(queueFamilyIndex), C.uint32_t(queueIndex), &queue)
	return Queue{handle: queue}
}
