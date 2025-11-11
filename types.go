package vulkango

import "C"

import "fmt"

type Result int32

const (
	SUCCESS                                      Result = 0
	NOT_READY                                    Result = 1
	TIMEOUT                                      Result = 2
	EVENT_SET                                    Result = 3
	EVENT_RESET                                  Result = 4
	INCOMPLETE                                   Result = 5
	OUT_OF_HOST_MEMORY                           Result = -1
	OUT_OF_DEVICE_MEMORY                         Result = -2
	INITIALIZATION_FAILED                        Result = -3
	DEVICE_LOST                                  Result = -4
	MEMORY_MAP_FAILED                            Result = -5
	LAYER_NOT_PRESENT                            Result = -6
	EXTENSION_NOT_PRESENT                        Result = -7
	FEATURE_NOT_PRESENT                          Result = -8
	INCOMPATIBLE_DRIVER                          Result = -9
	TOO_MANY_OBJECTS                             Result = -10
	FORMAT_NOT_SUPPORTED                         Result = -11
	FRAGMENTED_POOL                              Result = -12
	UNKNOWN                                      Result = -13
	OUT_OF_POOL_MEMORY                                  = -1000069000
	INVALID_EXTERNAL_HANDLE                             = -1000072003
	FRAGMENTATION                                       = -1000161000
	INVALID_OPAQUE_CAPTURE_ADDRESS                      = -1000257000
	PIPELINE_COMPILE_REQUIRED                           = 1000297000
	NOT_PERMITTED                                       = -1000174001
	SURFACE_LOST_KHR                                    = -1000000000
	NATIVE_WINDOW_IN_USE_KHR                            = -1000000001
	SUBOPTIMAL_KHR                                      = 1000001003
	OUT_OF_DATE_KHR                                     = -1000001004
	INCOMPATIBLE_DISPLAY_KHR                            = -1000003001
	VALIDATION_FAILED_EXT                               = -1000011001
	INVALID_SHADER_NV                                   = -1000012000
	IMAGE_USAGE_NOT_SUPPORTED_KHR                       = -1000023000
	VIDEO_PICTURE_LAYOUT_NOT_SUPPORTED_KHR              = -1000023001
	VIDEO_PROFILE_OPERATION_NOT_SUPPORTED_KHR           = -1000023002
	VIDEO_PROFILE_FORMAT_NOT_SUPPORTED_KHR              = -1000023003
	VIDEO_PROFILE_CODEC_NOT_SUPPORTED_KHR               = -1000023004
	VIDEO_STD_VERSION_NOT_SUPPORTED_KHR                 = -1000023005
	INVALID_DRM_FORMAT_MODIFIER_PLANE_LAYOUT_EXT        = -1000158000
	FULL_SCREEN_EXCLUSIVE_MODE_LOST_EXT                 = -1000255000
	THREAD_IDLE_KHR                                     = 1000268000
	THREAD_DONE_KHR                                     = 1000268001
	OPERATION_DEFERRED_KHR                              = 1000268002
	OPERATION_NOT_DEFERRED_KHR                          = 1000268003
	INVALID_VIDEO_STD_PARAMETERS_KHR                    = -1000299000
	COMPRESSION_EXHAUSTED_EXT                           = -1000338000
	INCOMPATIBLE_SHADER_BINARY_EXT                      = 1000482000
	PIPELINE_BINARY_MISSING_KHR                         = 1000483000
	NOT_ENOUGH_SPACE_KHR                                = -1000483000
	OUT_OF_POOL_MEMORY_KHR                              = OUT_OF_POOL_MEMORY
	INVALID_EXTERNAL_HANDLE_KHR                         = INVALID_EXTERNAL_HANDLE
	FRAGMENTATION_EXT                                   = FRAGMENTATION
	NOT_PERMITTED_EXT                                   = NOT_PERMITTED
	NOT_PERMITTED_KHR                                   = NOT_PERMITTED
	INVALID_DEVICE_ADDRESS_EXT                          = INVALID_OPAQUE_CAPTURE_ADDRESS
	INVALID_OPAQUE_CAPTURE_ADDRESS_KHR                  = INVALID_OPAQUE_CAPTURE_ADDRESS
	PIPELINE_COMPILE_REQUIRED_EXT                       = PIPELINE_COMPILE_REQUIRED
	ERROR_PIPELINE_COMPILE_REQUIRED_EXT                 = PIPELINE_COMPILE_REQUIRED
	ERROR_INCOMPATIBLE_SHADER_BINARY_EXT                = INCOMPATIBLE_SHADER_BINARY_EXT
)

func (r Result) Error() string {
	// Convert result codes to strings
	switch r {
	case SUCCESS:
		return "SUCCESS"
	default:
		return fmt.Sprintf("VkResult(%d)", r)
	}
}
