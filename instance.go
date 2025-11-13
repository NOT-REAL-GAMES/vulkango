package vulkango

// #cgo windows LDFLAGS: -LC:/VulkanSDK/1.4.328.1/Lib -lvulkan-1
// #cgo windows CFLAGS: -IC:/VulkanSDK/1.4.328.1/Include
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lvulkan
// #cgo darwin LDFLAGS: -lvulkan
// #include <vulkan/vulkan.h>
import "C"

func EnumerateInstanceVersion() (uint32, error) {
	var version C.uint32_t
	result := C.vkEnumerateInstanceVersion(&version)

	if result != C.VK_SUCCESS {
		return 0, Result(result)
	}

	return uint32(version), nil
}
