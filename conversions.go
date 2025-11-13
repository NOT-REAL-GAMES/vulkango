package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

// Vulkanizable types can convert themselves to C structs
type Vulkanizable interface {
	vulkanize() unsafe.Pointer
	free()
}

// ApplicationInfo
func (info *ApplicationInfo) vulkanize() unsafe.Pointer {
	cInfo := (*C.VkApplicationInfo)(C.malloc(C.sizeof_VkApplicationInfo))
	cInfo.sType = C.VK_STRUCTURE_TYPE_APPLICATION_INFO
	cInfo.pNext = nil

	if info.ApplicationName != "" {
		cInfo.pApplicationName = C.CString(info.ApplicationName)
	}
	cInfo.applicationVersion = C.uint32_t(info.ApplicationVersion)

	if info.EngineName != "" {
		cInfo.pEngineName = C.CString(info.EngineName)
	}
	cInfo.engineVersion = C.uint32_t(info.EngineVersion)
	cInfo.apiVersion = C.uint32_t(info.ApiVersion)

	return unsafe.Pointer(cInfo)
}

func (info *ApplicationInfo) free() {
	cInfo := (*C.VkApplicationInfo)(info.vulkanize())
	if cInfo.pApplicationName != nil {
		C.free(unsafe.Pointer(cInfo.pApplicationName))
	}
	if cInfo.pEngineName != nil {
		C.free(unsafe.Pointer(cInfo.pEngineName))
	}
	C.free(unsafe.Pointer(cInfo))
}

// InstanceCreateInfo

func (info *InstanceCreateInfo) vulkanize() unsafe.Pointer {
	cInfo := (*C.VkInstanceCreateInfo)(C.malloc(C.sizeof_VkInstanceCreateInfo))
	cInfo.sType = C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkInstanceCreateFlags(info.Flags)

	// Application info
	if info.ApplicationInfo != nil {
		cInfo.pApplicationInfo = (*C.VkApplicationInfo)(info.ApplicationInfo.vulkanize())
	}

	// Layers
	if len(info.EnabledLayerNames) > 0 {
		cLayers := make([]*C.char, len(info.EnabledLayerNames))
		for i, layer := range info.EnabledLayerNames {
			cLayers[i] = C.CString(layer)
		}
		cInfo.enabledLayerCount = C.uint32_t(len(cLayers))
		cInfo.ppEnabledLayerNames = &cLayers[0]
	}

	// Extensions
	if len(info.EnabledExtensionNames) > 0 {
		cExts := make([]*C.char, len(info.EnabledExtensionNames))
		for i, ext := range info.EnabledExtensionNames {
			cExts[i] = C.CString(ext)
		}
		cInfo.enabledExtensionCount = C.uint32_t(len(cExts))
		cInfo.ppEnabledExtensionNames = &cExts[0]
	}

	return unsafe.Pointer(cInfo)
}

func (info *InstanceCreateInfo) free() {
	cInfo := (*C.VkInstanceCreateInfo)(info.vulkanize())

	if cInfo.pApplicationInfo != nil {
		C.free(unsafe.Pointer(cInfo.pApplicationInfo))
	}

	// Free layer strings
	if cInfo.enabledLayerCount > 0 {
		layers := (*[1 << 30]*C.char)(unsafe.Pointer(cInfo.ppEnabledLayerNames))[:cInfo.enabledLayerCount:cInfo.enabledLayerCount]
		for _, layer := range layers {
			C.free(unsafe.Pointer(layer))
		}
	}

	// Free extension strings
	if cInfo.enabledExtensionCount > 0 {
		exts := (*[1 << 30]*C.char)(unsafe.Pointer(cInfo.ppEnabledExtensionNames))[:cInfo.enabledExtensionCount:cInfo.enabledExtensionCount]
		for _, ext := range exts {
			C.free(unsafe.Pointer(ext))
		}
	}

	C.free(unsafe.Pointer(cInfo))
}
