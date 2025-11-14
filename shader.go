// shader.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type ShaderModule struct {
	handle C.VkShaderModule
}

type ShaderModuleCreateInfo struct {
	Code []byte
}

func (device Device) CreateShaderModule(createInfo *ShaderModuleCreateInfo) (ShaderModule, error) {
	cInfo := (*C.VkShaderModuleCreateInfo)(C.calloc(1, C.sizeof_VkShaderModuleCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0
	cInfo.codeSize = C.size_t(len(createInfo.Code))
	cInfo.pCode = (*C.uint32_t)(unsafe.Pointer(&createInfo.Code[0]))

	var shaderModule C.VkShaderModule
	result := C.vkCreateShaderModule(device.handle, cInfo, nil, &shaderModule)

	if result != C.VK_SUCCESS {
		return ShaderModule{}, Result(result)
	}

	return ShaderModule{handle: shaderModule}, nil
}

func (device Device) DestroyShaderModule(shaderModule ShaderModule) {
	C.vkDestroyShaderModule(device.handle, shaderModule.handle, nil)
}
