package vulkango

// #cgo LDFLAGS: -lvulkan

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
