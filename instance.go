package vulkango

// #cgo windows LDFLAGS: -lvulkan-1
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lvulkan
// #cgo darwin LDFLAGS: -lvulkan
// #include <vulkan/vulkan.h>
import "C"
import "unsafe"

type Instance struct {
	handle C.VkInstance
}

func (instance Instance) Handle() unsafe.Pointer {
	return unsafe.Pointer(instance.handle)
}

func EnumerateInstanceVersion() (uint32, error) {
	var version C.uint32_t
	result := C.vkEnumerateInstanceVersion(&version)

	if result != C.VK_SUCCESS {
		return 0, Result(result)
	}

	return uint32(version), nil
}

func CreateInstance(createInfo *InstanceCreateInfo) (Instance, error) {
	data := createInfo.vulkanize()
	defer data.free()

	var instance C.VkInstance
	result := C.vkCreateInstance(data.cInfo, nil, &instance)

	if result != C.VK_SUCCESS {
		return Instance{}, Result(result)
	}

	return Instance{handle: instance}, nil
}

func (instance Instance) Destroy() {
	C.vkDestroyInstance(instance.handle, nil)
}

func (instance Instance) EnumeratePhysicalDevices() ([]PhysicalDevice, error) {
	var count C.uint32_t
	result := C.vkEnumeratePhysicalDevices(instance.handle, &count, nil)

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	devices := make([]C.VkPhysicalDevice, count)
	result = C.vkEnumeratePhysicalDevices(instance.handle, &count, &devices[0])

	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	goDevices := make([]PhysicalDevice, count)
	for i := range goDevices {
		goDevices[i] = PhysicalDevice{handle: devices[i]}
	}

	return goDevices, nil
}

func (instance Instance) GetPhysicalDeviceProperties(device PhysicalDevice) PhysicalDeviceProperties {
	var props C.VkPhysicalDeviceProperties
	C.vkGetPhysicalDeviceProperties(device.handle, &props)

	return PhysicalDeviceProperties{
		Limits: PhysicalDeviceLimits{
			MaxDescriptorSetSampledImages: uint32(props.limits.maxDescriptorSetSampledImages),
		},
	}
}
