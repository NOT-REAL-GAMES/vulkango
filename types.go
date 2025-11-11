package vulkango

import "C"

import "fmt"

type Result int32

const (
	SUCCESS                                  Result = 0
	NOT_READY                                Result = 1
	TIMEOUT                                  Result = 2
	EVENT_SET                                Result = 3
	EVENT_RESET                              Result = 4
	INCOMPLETE                               Result = 5
	OUT_OF_HOST_MEMORY                       Result = -1
	OUT_OF_DEVICE_MEMORY                     Result = -2
	INITIALIZATION_FAILED                    Result = -3
	DEVICE_LOST                              Result = -4
	MEMORY_MAP_FAILED                        Result = -5
	LAYER_NOT_PRESENT                        Result = -6
	EXTENSION_NOT_PRESENT                    Result = -7
	FEATURE_NOT_PRESENT                      Result = -8
	INCOMPATIBLE_DRIVER                      Result = -9
	TOO_MANY_OBJECTS                         Result = -10
	FORMAT_NOT_SUPPORTED                     Result = -11
	FRAGMENTED_POOL                          Result = -12
	UNKNOWN                                  Result = -13
	OUT_OF_POOL_MEMORY                       Result = -1000069000
	INVALID_EXTERNAL_HANDLE                  Result = -1000072003
	FRAGMENTATION                            Result = -1000161000
	INVALID_OPAQUE_CAPTURE_ADDRESS           Result = -1000257000
	PIPELINE_COMPILE_REQUIRED                Result = 1000297000
	NOT_PERMITTED                            Result = -1000174001
	SURFACE_LOST                             Result = -1000000000
	NATIVE_WINDOW_IN_USE                     Result = -1000000001
	SUBOPTIMAL                               Result = 1000001003
	OUT_OF_DATE                              Result = -1000001004
	INCOMPATIBLE_DISPLAY                     Result = -1000003001
	VALIDATION_FAILED                        Result = -1000011001
	INVALID_SHADER                           Result = -1000012000
	IMAGE_USAGE_NOT_SUPPORTED                Result = -1000023000
	VIDEO_PICTURE_LAYOUT_NOT_SUPPORTED       Result = -1000023001
	VIDEO_PROFILE_OPERATION_NOT_SUPPORTED    Result = -1000023002
	VIDEO_PROFILE_FORMAT_NOT_SUPPORTED       Result = -1000023003
	VIDEO_PROFILE_CODEC_NOT_SUPPORTED        Result = -1000023004
	VIDEO_STD_VERSION_NOT_SUPPORTED          Result = -1000023005
	INVALID_DRM_FORMAT_MODIFIER_PLANE_LAYOUT Result = -1000158000
	FULL_SCREEN_EXCLUSIVE_MODE_LOST          Result = -1000255000
	THREAD_IDLE                              Result = 1000268000
	THREAD_DONE                              Result = 1000268001
	OPERATION_DEFERRED                       Result = 1000268002
	OPERATION_NOT_DEFERRED                   Result = 1000268003
	INVALID_VIDEO_STD_PARAMETERS             Result = -1000299000
	COMPRESSION_EXHAUSTED                    Result = -1000338000
	INCOMPATIBLE_SHADER_BINARY               Result = 1000482000
	PIPELINE_BINARY_MISSING                  Result = 1000483000
	NOT_ENOUGH_SPACE                         Result = -1000483000
)

func (r Result) Error() string {
	// Convert result codes to strings
	switch r {
	case SUCCESS:
		return "SUCCESS"
	case NOT_READY:
		return "NOT READY"
	case TIMEOUT:
		return "TIMEOUT"
	case EVENT_SET:
		return "EVENT SET"
	case EVENT_RESET:
		return "EVENT RESET"
	case INCOMPLETE:
		return "INCOMPLETE"
	case OUT_OF_HOST_MEMORY:
		return "OUT OF HOST MEMORY"
	case OUT_OF_DEVICE_MEMORY:
		return "OUT OF DEVICE MEMORY"
	case INITIALIZATION_FAILED:
		return "INITIALIZATION FAILED"
	case DEVICE_LOST:
		return "DEVICE LOST"
	case MEMORY_MAP_FAILED:
		return "MEMORY MAP FAILED"
	case LAYER_NOT_PRESENT:
		return "LAYER NOT PRESENT"
	case EXTENSION_NOT_PRESENT:
		return "EXTENSION NOT PRESENT"
	case FEATURE_NOT_PRESENT:
		return "FEATURE NOT PRESENT"
	case INCOMPATIBLE_DRIVER:
		return "INCOMPATIBLE DRIVER"
	case TOO_MANY_OBJECTS:
		return "TOO MANY OBJECTS"
	case FORMAT_NOT_SUPPORTED:
		return "FORMAT NOT SUPPORTED"
	case FRAGMENTED_POOL:
		return "FRAGMENTED POOL"
	case UNKNOWN:
		return "UNKNOWN"
	case OUT_OF_POOL_MEMORY:
		return "OUT OF POOL MEMORY"
	case INVALID_EXTERNAL_HANDLE:
		return "INVALID EXTERNAL HANDLE"
	case FRAGMENTATION:
		return "FRAGMENTATION"
	case INVALID_OPAQUE_CAPTURE_ADDRESS:
		return "INVALID OPAQUE CAPTURE ADDRESS"
	case PIPELINE_COMPILE_REQUIRED:
		return "PIPELINE COMPILE REQUIRED"
	case NOT_PERMITTED:
		return "NOT PERMITTED"
	case SURFACE_LOST:
		return "SURFACE LOST"
	case NATIVE_WINDOW_IN_USE:
		return "NATIVE WINDOW IN USE"
	case SUBOPTIMAL:
		return "SUBOPTIMAL"
	case OUT_OF_DATE:
		return "OUT OF DATE"
	case INCOMPATIBLE_DISPLAY:
		return "INCOMPATIBLE DISPLAY"
	case VALIDATION_FAILED:
		return "VALIDATION FAILED"
	case INVALID_SHADER:
		return "INVALID SHADER"
	case IMAGE_USAGE_NOT_SUPPORTED:
		return "IMAGE USAGE NOT SUPPORTED"
	case VIDEO_PICTURE_LAYOUT_NOT_SUPPORTED:
		return "VIDEO PICTURE LAYOUT NOT SUPPORTED"
	case VIDEO_PROFILE_OPERATION_NOT_SUPPORTED:
		return "VIDEO PROFILE OPERATION NOT SUPPORTED"
	case VIDEO_PROFILE_FORMAT_NOT_SUPPORTED:
		return "VIDEO PROFILE FORMAT NOT SUPPORTED"
	case VIDEO_PROFILE_CODEC_NOT_SUPPORTED:
		return "VIDEO PROFILE CODEC NOT SUPPORTED"
	case VIDEO_STD_VERSION_NOT_SUPPORTED:
		return "VIDEO STD VERSION NOT SUPPORTED"
	case INVALID_DRM_FORMAT_MODIFIER_PLANE_LAYOUT:
		return "INVALID DRM FORMAT MODIFIER PLANE LAYOUT"
	case FULL_SCREEN_EXCLUSIVE_MODE_LOST:
		return "FULL SCREEN EXCLUSIVE MODE LOST"
	case THREAD_IDLE:
		return "THREAD IDLE"
	case THREAD_DONE:
		return "THREAD DONE"
	case OPERATION_DEFERRED:
		return "OPERATION DEFERRED"
	case OPERATION_NOT_DEFERRED:
		return "OPERATION NOT DEFERRED"
	case INVALID_VIDEO_STD_PARAMETERS:
		return "INVALID VIDEO STD PARAMETERS"
	case COMPRESSION_EXHAUSTED:
		return "COMPRESSION EXHAUSTED"
	case INCOMPATIBLE_SHADER_BINARY:
		return "INCOMPATIBLE SHADER BINARY"
	case PIPELINE_BINARY_MISSING:
		return "PIPELINE BINARY MISSING"
	case NOT_ENOUGH_SPACE:
		return "NOT ENOUGH SPACE"
	default:
		return fmt.Sprintf("VkResult(%d)", r)
	}
}
