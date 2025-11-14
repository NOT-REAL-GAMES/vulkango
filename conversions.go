// conversions.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

type instanceCreateInfoData struct {
	cInfo       *C.VkInstanceCreateInfo
	cAppInfo    *C.VkApplicationInfo
	cLayers     []*C.char
	cExtensions []*C.char
}

func (info *InstanceCreateInfo) vulkanize() *instanceCreateInfoData {
	data := &instanceCreateInfoData{}

	// Allocate and ZERO the main struct (calloc instead of malloc)
	data.cInfo = (*C.VkInstanceCreateInfo)(C.calloc(1, C.sizeof_VkInstanceCreateInfo))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO
	data.cInfo.pNext = nil
	data.cInfo.flags = C.VkInstanceCreateFlags(info.Flags)

	// Application info
	if info.ApplicationInfo != nil {
		data.cAppInfo = (*C.VkApplicationInfo)(C.calloc(1, C.sizeof_VkApplicationInfo))
		data.cAppInfo.sType = C.VK_STRUCTURE_TYPE_APPLICATION_INFO
		data.cAppInfo.pNext = nil
		data.cAppInfo.pApplicationName = nil
		data.cAppInfo.applicationVersion = C.uint32_t(info.ApplicationInfo.ApplicationVersion)
		data.cAppInfo.pEngineName = nil
		data.cAppInfo.engineVersion = C.uint32_t(info.ApplicationInfo.EngineVersion)
		data.cAppInfo.apiVersion = C.uint32_t(info.ApplicationInfo.ApiVersion)

		if info.ApplicationInfo.ApplicationName != "" {
			data.cAppInfo.pApplicationName = C.CString(info.ApplicationInfo.ApplicationName)
		}

		if info.ApplicationInfo.EngineName != "" {
			data.cAppInfo.pEngineName = C.CString(info.ApplicationInfo.EngineName)
		}

		data.cInfo.pApplicationInfo = data.cAppInfo
	} else {
		data.cInfo.pApplicationInfo = nil
	}

	// Layers
	data.cInfo.enabledLayerCount = 0
	data.cInfo.ppEnabledLayerNames = nil
	if len(info.EnabledLayerNames) > 0 {
		data.cLayers = make([]*C.char, len(info.EnabledLayerNames))
		for i, layer := range info.EnabledLayerNames {
			data.cLayers[i] = C.CString(layer)
		}
		data.cInfo.enabledLayerCount = C.uint32_t(len(data.cLayers))
		data.cInfo.ppEnabledLayerNames = (**C.char)(unsafe.Pointer(&data.cLayers[0]))
	}

	// Extensions
	data.cInfo.enabledExtensionCount = 0
	data.cInfo.ppEnabledExtensionNames = nil
	if len(info.EnabledExtensionNames) > 0 {
		data.cExtensions = make([]*C.char, len(info.EnabledExtensionNames))
		for i, ext := range info.EnabledExtensionNames {
			data.cExtensions[i] = C.CString(ext)
		}
		data.cInfo.enabledExtensionCount = C.uint32_t(len(data.cExtensions))
		data.cInfo.ppEnabledExtensionNames = (**C.char)(unsafe.Pointer(&data.cExtensions[0]))
	}

	return data
}

func (data *instanceCreateInfoData) free() {
	if data.cAppInfo != nil {
		if data.cAppInfo.pApplicationName != nil {
			C.free(unsafe.Pointer(data.cAppInfo.pApplicationName))
		}
		if data.cAppInfo.pEngineName != nil {
			C.free(unsafe.Pointer(data.cAppInfo.pEngineName))
		}
		C.free(unsafe.Pointer(data.cAppInfo))
	}

	for _, layer := range data.cLayers {
		C.free(unsafe.Pointer(layer))
	}

	for _, ext := range data.cExtensions {
		C.free(unsafe.Pointer(ext))
	}

	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}
