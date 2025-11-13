package vulkango

// #cgo windows LDFLAGS: -lvulkan-1
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
