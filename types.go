package vulkango

// #include <vulkan/vulkan.h>
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

type StructureType int32

const (
	APPLICATION_INFO                                                      StructureType = 0
	INSTANCE_CREATE_INFO                                                  StructureType = 1
	DEVICE_QUEUE_CREATE_INFO                                              StructureType = 2
	DEVICE_CREATE_INFO                                                    StructureType = 3
	SUBMIT_INFO                                                           StructureType = 4
	MEMORY_ALLOCATE_INFO                                                  StructureType = 5
	MAPPED_MEMORY_RANGE                                                   StructureType = 6
	BIND_SPARSE_INFO                                                      StructureType = 7
	FENCE_CREATE_INFO                                                     StructureType = 8
	SEMAPHORE_CREATE_INFO                                                 StructureType = 9
	EVENT_CREATE_INFO                                                     StructureType = 10
	QUERY_POOL_CREATE_INFO                                                StructureType = 11
	BUFFER_CREATE_INFO                                                    StructureType = 12
	BUFFER_VIEW_CREATE_INFO                                               StructureType = 13
	IMAGE_CREATE_INFO                                                     StructureType = 14
	IMAGE_VIEW_CREATE_INFO                                                StructureType = 15
	SHADER_MODULE_CREATE_INFO                                             StructureType = 16
	PIPELINE_CACHE_CREATE_INFO                                            StructureType = 17
	PIPELINE_SHADER_STAGE_CREATE_INFO                                     StructureType = 18
	PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO                               StructureType = 19
	PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO                             StructureType = 20
	PIPELINE_TESSELLATION_STATE_CREATE_INFO                               StructureType = 21
	PIPELINE_VIEWPORT_STATE_CREATE_INFO                                   StructureType = 22
	PIPELINE_RASTERIZATION_STATE_CREATE_INFO                              StructureType = 23
	PIPELINE_MULTISAMPLE_STATE_CREATE_INFO                                StructureType = 24
	PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO                              StructureType = 25
	PIPELINE_COLOR_BLEND_STATE_CREATE_INFO                                StructureType = 26
	PIPELINE_DYNAMIC_STATE_CREATE_INFO                                    StructureType = 27
	GRAPHICS_PIPELINE_CREATE_INFO                                         StructureType = 28
	COMPUTE_PIPELINE_CREATE_INFO                                          StructureType = 29
	PIPELINE_LAYOUT_CREATE_INFO                                           StructureType = 30
	SAMPLER_CREATE_INFO                                                   StructureType = 31
	DESCRIPTOR_SET_LAYOUT_CREATE_INFO                                     StructureType = 32
	DESCRIPTOR_POOL_CREATE_INFO                                           StructureType = 33
	DESCRIPTOR_SET_ALLOCATE_INFO                                          StructureType = 34
	WRITE_DESCRIPTOR_SET                                                  StructureType = 35
	COPY_DESCRIPTOR_SET                                                   StructureType = 36
	FRAMEBUFFER_CREATE_INFO                                               StructureType = 37
	RENDER_PASS_CREATE_INFO                                               StructureType = 38
	COMMAND_POOL_CREATE_INFO                                              StructureType = 39
	COMMAND_BUFFER_ALLOCATE_INFO                                          StructureType = 40
	COMMAND_BUFFER_INHERITANCE_INFO                                       StructureType = 41
	COMMAND_BUFFER_BEGIN_INFO                                             StructureType = 42
	RENDER_PASS_BEGIN_INFO                                                StructureType = 43
	BUFFER_MEMORY_BARRIER                                                 StructureType = 44
	IMAGE_MEMORY_BARRIER                                                  StructureType = 45
	MEMORY_BARRIER                                                        StructureType = 46
	LOADER_INSTANCE_CREATE_INFO                                           StructureType = 47
	LOADER_DEVICE_CREATE_INFO                                             StructureType = 48
	PHYSICAL_DEVICE_SUBGROUP_PROPERTIES                                   StructureType = 1000094000
	BIND_BUFFER_MEMORY_INFO                                               StructureType = 1000157000
	BIND_IMAGE_MEMORY_INFO                                                StructureType = 1000157001
	PHYSICAL_DEVICE_16BIT_STORAGE_FEATURES                                StructureType = 1000083000
	MEMORY_DEDICATED_REQUIREMENTS                                         StructureType = 1000127000
	MEMORY_DEDICATED_ALLOCATE_INFO                                        StructureType = 1000127001
	MEMORY_ALLOCATE_FLAGS_INFO                                            StructureType = 1000060000
	DEVICE_GROUP_RENDER_PASS_BEGIN_INFO                                   StructureType = 1000060003
	DEVICE_GROUP_COMMAND_BUFFER_BEGIN_INFO                                StructureType = 1000060004
	DEVICE_GROUP_SUBMIT_INFO                                              StructureType = 1000060005
	DEVICE_GROUP_BIND_SPARSE_INFO                                         StructureType = 1000060006
	BIND_BUFFER_MEMORY_DEVICE_GROUP_INFO                                  StructureType = 1000060013
	BIND_IMAGE_MEMORY_DEVICE_GROUP_INFO                                   StructureType = 1000060014
	PHYSICAL_DEVICE_GROUP_PROPERTIES                                      StructureType = 1000070000
	DEVICE_GROUP_DEVICE_CREATE_INFO                                       StructureType = 1000070001
	BUFFER_MEMORY_REQUIREMENTS_INFO_2                                     StructureType = 1000146000
	IMAGE_MEMORY_REQUIREMENTS_INFO_2                                      StructureType = 1000146001
	IMAGE_SPARSE_MEMORY_REQUIREMENTS_INFO_2                               StructureType = 1000146002
	MEMORY_REQUIREMENTS_2                                                 StructureType = 1000146003
	SPARSE_IMAGE_MEMORY_REQUIREMENTS_2                                    StructureType = 1000146004
	PHYSICAL_DEVICE_FEATURES_2                                            StructureType = 1000059000
	PHYSICAL_DEVICE_PROPERTIES_2                                          StructureType = 1000059001
	FORMAT_PROPERTIES_2                                                   StructureType = 1000059002
	IMAGE_FORMAT_PROPERTIES_2                                             StructureType = 1000059003
	PHYSICAL_DEVICE_IMAGE_FORMAT_INFO_2                                   StructureType = 1000059004
	QUEUE_FAMILY_PROPERTIES_2                                             StructureType = 1000059005
	PHYSICAL_DEVICE_MEMORY_PROPERTIES_2                                   StructureType = 1000059006
	SPARSE_IMAGE_FORMAT_PROPERTIES_2                                      StructureType = 1000059007
	PHYSICAL_DEVICE_SPARSE_IMAGE_FORMAT_INFO_2                            StructureType = 1000059008
	PHYSICAL_DEVICE_POINT_CLIPPING_PROPERTIES                             StructureType = 1000117000
	RENDER_PASS_INPUT_ATTACHMENT_ASPECT_CREATE_INFO                       StructureType = 1000117001
	IMAGE_VIEW_USAGE_CREATE_INFO                                          StructureType = 1000117002
	PIPELINE_TESSELLATION_DOMAIN_ORIGIN_STATE_CREATE_INFO                 StructureType = 1000117003
	RENDER_PASS_MULTIVIEW_CREATE_INFO                                     StructureType = 1000053000
	PHYSICAL_DEVICE_MULTIVIEW_FEATURES                                    StructureType = 1000053001
	PHYSICAL_DEVICE_MULTIVIEW_PROPERTIES                                  StructureType = 1000053002
	PHYSICAL_DEVICE_VARIABLE_POINTERS_FEATURES                            StructureType = 1000120000
	PROTECTED_SUBMIT_INFO                                                 StructureType = 1000145000
	PHYSICAL_DEVICE_PROTECTED_MEMORY_FEATURES                             StructureType = 1000145001
	PHYSICAL_DEVICE_PROTECTED_MEMORY_PROPERTIES                           StructureType = 1000145002
	DEVICE_QUEUE_INFO_2                                                   StructureType = 1000145003
	SAMPLER_YCBCR_CONVERSION_CREATE_INFO                                  StructureType = 1000156000
	SAMPLER_YCBCR_CONVERSION_INFO                                         StructureType = 1000156001
	BIND_IMAGE_PLANE_MEMORY_INFO                                          StructureType = 1000156002
	IMAGE_PLANE_MEMORY_REQUIREMENTS_INFO                                  StructureType = 1000156003
	PHYSICAL_DEVICE_SAMPLER_YCBCR_CONVERSION_FEATURES                     StructureType = 1000156004
	SAMPLER_YCBCR_CONVERSION_IMAGE_FORMAT_PROPERTIES                      StructureType = 1000156005
	DESCRIPTOR_UPDATE_TEMPLATE_CREATE_INFO                                StructureType = 1000085000
	PHYSICAL_DEVICE_EXTERNAL_IMAGE_FORMAT_INFO                            StructureType = 1000071000
	EXTERNAL_IMAGE_FORMAT_PROPERTIES                                      StructureType = 1000071001
	PHYSICAL_DEVICE_EXTERNAL_BUFFER_INFO                                  StructureType = 1000071002
	EXTERNAL_BUFFER_PROPERTIES                                            StructureType = 1000071003
	PHYSICAL_DEVICE_ID_PROPERTIES                                         StructureType = 1000071004
	EXTERNAL_MEMORY_BUFFER_CREATE_INFO                                    StructureType = 1000072000
	EXTERNAL_MEMORY_IMAGE_CREATE_INFO                                     StructureType = 1000072001
	EXPORT_MEMORY_ALLOCATE_INFO                                           StructureType = 1000072002
	PHYSICAL_DEVICE_EXTERNAL_FENCE_INFO                                   StructureType = 1000112000
	EXTERNAL_FENCE_PROPERTIES                                             StructureType = 1000112001
	EXPORT_FENCE_CREATE_INFO                                              StructureType = 1000113000
	EXPORT_SEMAPHORE_CREATE_INFO                                          StructureType = 1000077000
	PHYSICAL_DEVICE_EXTERNAL_SEMAPHORE_INFO                               StructureType = 1000076000
	EXTERNAL_SEMAPHORE_PROPERTIES                                         StructureType = 1000076001
	PHYSICAL_DEVICE_MAINTENANCE_3_PROPERTIES                              StructureType = 1000168000
	DESCRIPTOR_SET_LAYOUT_SUPPORT                                         StructureType = 1000168001
	PHYSICAL_DEVICE_SHADER_DRAW_PARAMETERS_FEATURES                       StructureType = 1000063000
	PHYSICAL_DEVICE_VULKAN_1_1_FEATURES                                   StructureType = 49
	PHYSICAL_DEVICE_VULKAN_1_1_PROPERTIES                                 StructureType = 50
	PHYSICAL_DEVICE_VULKAN_1_2_FEATURES                                   StructureType = 51
	PHYSICAL_DEVICE_VULKAN_1_2_PROPERTIES                                 StructureType = 52
	IMAGE_FORMAT_LIST_CREATE_INFO                                         StructureType = 1000147000
	ATTACHMENT_DESCRIPTION_2                                              StructureType = 1000109000
	ATTACHMENT_REFERENCE_2                                                StructureType = 1000109001
	SUBPASS_DESCRIPTION_2                                                 StructureType = 1000109002
	SUBPASS_DEPENDENCY_2                                                  StructureType = 1000109003
	RENDER_PASS_CREATE_INFO_2                                             StructureType = 1000109004
	SUBPASS_BEGIN_INFO                                                    StructureType = 1000109005
	SUBPASS_END_INFO                                                      StructureType = 1000109006
	PHYSICAL_DEVICE_8BIT_STORAGE_FEATURES                                 StructureType = 1000177000
	PHYSICAL_DEVICE_DRIVER_PROPERTIES                                     StructureType = 1000196000
	PHYSICAL_DEVICE_SHADER_ATOMIC_INT64_FEATURES                          StructureType = 1000180000
	PHYSICAL_DEVICE_SHADER_FLOAT16_INT8_FEATURES                          StructureType = 1000082000
	PHYSICAL_DEVICE_FLOAT_CONTROLS_PROPERTIES                             StructureType = 1000197000
	DESCRIPTOR_SET_LAYOUT_BINDING_FLAGS_CREATE_INFO                       StructureType = 1000161000
	PHYSICAL_DEVICE_DESCRIPTOR_INDEXING_FEATURES                          StructureType = 1000161001
	PHYSICAL_DEVICE_DESCRIPTOR_INDEXING_PROPERTIES                        StructureType = 1000161002
	DESCRIPTOR_SET_VARIABLE_DESCRIPTOR_COUNT_ALLOCATE_INFO                StructureType = 1000161003
	DESCRIPTOR_SET_VARIABLE_DESCRIPTOR_COUNT_LAYOUT_SUPPORT               StructureType = 1000161004
	PHYSICAL_DEVICE_DEPTH_STENCIL_RESOLVE_PROPERTIES                      StructureType = 1000199000
	SUBPASS_DESCRIPTION_DEPTH_STENCIL_RESOLVE                             StructureType = 1000199001
	PHYSICAL_DEVICE_SCALAR_BLOCK_LAYOUT_FEATURES                          StructureType = 1000221000
	IMAGE_STENCIL_USAGE_CREATE_INFO                                       StructureType = 1000246000
	PHYSICAL_DEVICE_SAMPLER_FILTER_MINMAX_PROPERTIES                      StructureType = 1000130000
	SAMPLER_REDUCTION_MODE_CREATE_INFO                                    StructureType = 1000130001
	PHYSICAL_DEVICE_VULKAN_MEMORY_MODEL_FEATURES                          StructureType = 1000211000
	PHYSICAL_DEVICE_IMAGELESS_FRAMEBUFFER_FEATURES                        StructureType = 1000108000
	FRAMEBUFFER_ATTACHMENTS_CREATE_INFO                                   StructureType = 1000108001
	FRAMEBUFFER_ATTACHMENT_IMAGE_INFO                                     StructureType = 1000108002
	RENDER_PASS_ATTACHMENT_BEGIN_INFO                                     StructureType = 1000108003
	PHYSICAL_DEVICE_UNIFORM_BUFFER_STANDARD_LAYOUT_FEATURES               StructureType = 1000253000
	PHYSICAL_DEVICE_SHADER_SUBGROUP_EXTENDED_TYPES_FEATURES               StructureType = 1000175000
	PHYSICAL_DEVICE_SEPARATE_DEPTH_STENCIL_LAYOUTS_FEATURES               StructureType = 1000241000
	ATTACHMENT_REFERENCE_STENCIL_LAYOUT                                   StructureType = 1000241001
	ATTACHMENT_DESCRIPTION_STENCIL_LAYOUT                                 StructureType = 1000241002
	PHYSICAL_DEVICE_HOST_QUERY_RESET_FEATURES                             StructureType = 1000261000
	PHYSICAL_DEVICE_TIMELINE_SEMAPHORE_FEATURES                           StructureType = 1000207000
	PHYSICAL_DEVICE_TIMELINE_SEMAPHORE_PROPERTIES                         StructureType = 1000207001
	SEMAPHORE_TYPE_CREATE_INFO                                            StructureType = 1000207002
	TIMELINE_SEMAPHORE_SUBMIT_INFO                                        StructureType = 1000207003
	SEMAPHORE_WAIT_INFO                                                   StructureType = 1000207004
	SEMAPHORE_SIGNAL_INFO                                                 StructureType = 1000207005
	PHYSICAL_DEVICE_BUFFER_DEVICE_ADDRESS_FEATURES                        StructureType = 1000257000
	BUFFER_DEVICE_ADDRESS_INFO                                            StructureType = 1000244001
	BUFFER_OPAQUE_CAPTURE_ADDRESS_CREATE_INFO                             StructureType = 1000257002
	MEMORY_OPAQUE_CAPTURE_ADDRESS_ALLOCATE_INFO                           StructureType = 1000257003
	DEVICE_MEMORY_OPAQUE_CAPTURE_ADDRESS_INFO                             StructureType = 1000257004
	PHYSICAL_DEVICE_VULKAN_1_3_FEATURES                                   StructureType = 53
	PHYSICAL_DEVICE_VULKAN_1_3_PROPERTIES                                 StructureType = 54
	PIPELINE_CREATION_FEEDBACK_CREATE_INFO                                StructureType = 1000192000
	PHYSICAL_DEVICE_SHADER_TERMINATE_INVOCATION_FEATURES                  StructureType = 1000215000
	PHYSICAL_DEVICE_TOOL_PROPERTIES                                       StructureType = 1000245000
	PHYSICAL_DEVICE_SHADER_DEMOTE_TO_HELPER_INVOCATION_FEATURES           StructureType = 1000276000
	PHYSICAL_DEVICE_PRIVATE_DATA_FEATURES                                 StructureType = 1000295000
	DEVICE_PRIVATE_DATA_CREATE_INFO                                       StructureType = 1000295001
	PRIVATE_DATA_SLOT_CREATE_INFO                                         StructureType = 1000295002
	PHYSICAL_DEVICE_PIPELINE_CREATION_CACHE_CONTROL_FEATURES              StructureType = 1000297000
	MEMORY_BARRIER_2                                                      StructureType = 1000314000
	BUFFER_MEMORY_BARRIER_2                                               StructureType = 1000314001
	IMAGE_MEMORY_BARRIER_2                                                StructureType = 1000314002
	DEPENDENCY_INFO                                                       StructureType = 1000314003
	SUBMIT_INFO_2                                                         StructureType = 1000314004
	SEMAPHORE_SUBMIT_INFO                                                 StructureType = 1000314005
	COMMAND_BUFFER_SUBMIT_INFO                                            StructureType = 1000314006
	PHYSICAL_DEVICE_SYNCHRONIZATION_2_FEATURES                            StructureType = 1000314007
	PHYSICAL_DEVICE_ZERO_INITIALIZE_WORKGROUP_MEMORY_FEATURES             StructureType = 1000325000
	PHYSICAL_DEVICE_IMAGE_ROBUSTNESS_FEATURES                             StructureType = 1000335000
	COPY_BUFFER_INFO_2                                                    StructureType = 1000337000
	COPY_IMAGE_INFO_2                                                     StructureType = 1000337001
	COPY_BUFFER_TO_IMAGE_INFO_2                                           StructureType = 1000337002
	COPY_IMAGE_TO_BUFFER_INFO_2                                           StructureType = 1000337003
	BLIT_IMAGE_INFO_2                                                     StructureType = 1000337004
	RESOLVE_IMAGE_INFO_2                                                  StructureType = 1000337005
	BUFFER_COPY_2                                                         StructureType = 1000337006
	IMAGE_COPY_2                                                          StructureType = 1000337007
	IMAGE_BLIT_2                                                          StructureType = 1000337008
	BUFFER_IMAGE_COPY_2                                                   StructureType = 1000337009
	IMAGE_RESOLVE_2                                                       StructureType = 1000337010
	PHYSICAL_DEVICE_SUBGROUP_SIZE_CONTROL_PROPERTIES                      StructureType = 1000225000
	PIPELINE_SHADER_STAGE_REQUIRED_SUBGROUP_SIZE_CREATE_INFO              StructureType = 1000225001
	PHYSICAL_DEVICE_SUBGROUP_SIZE_CONTROL_FEATURES                        StructureType = 1000225002
	PHYSICAL_DEVICE_INLINE_UNIFORM_BLOCK_FEATURES                         StructureType = 1000138000
	PHYSICAL_DEVICE_INLINE_UNIFORM_BLOCK_PROPERTIES                       StructureType = 1000138001
	WRITE_DESCRIPTOR_SET_INLINE_UNIFORM_BLOCK                             StructureType = 1000138002
	DESCRIPTOR_POOL_INLINE_UNIFORM_BLOCK_CREATE_INFO                      StructureType = 1000138003
	PHYSICAL_DEVICE_TEXTURE_COMPRESSION_ASTC_HDR_FEATURES                 StructureType = 1000066000
	RENDERING_INFO                                                        StructureType = 1000044000
	RENDERING_ATTACHMENT_INFO                                             StructureType = 1000044001
	PIPELINE_RENDERING_CREATE_INFO                                        StructureType = 1000044002
	PHYSICAL_DEVICE_DYNAMIC_RENDERING_FEATURES                            StructureType = 1000044003
	COMMAND_BUFFER_INHERITANCE_RENDERING_INFO                             StructureType = 1000044004
	PHYSICAL_DEVICE_SHADER_INTEGER_DOT_PRODUCT_FEATURES                   StructureType = 1000280000
	PHYSICAL_DEVICE_SHADER_INTEGER_DOT_PRODUCT_PROPERTIES                 StructureType = 1000280001
	PHYSICAL_DEVICE_TEXEL_BUFFER_ALIGNMENT_PROPERTIES                     StructureType = 1000281001
	FORMAT_PROPERTIES_3                                                   StructureType = 1000360000
	PHYSICAL_DEVICE_MAINTENANCE_4_FEATURES                                StructureType = 1000413000
	PHYSICAL_DEVICE_MAINTENANCE_4_PROPERTIES                              StructureType = 1000413001
	DEVICE_BUFFER_MEMORY_REQUIREMENTS                                     StructureType = 1000413002
	DEVICE_IMAGE_MEMORY_REQUIREMENTS                                      StructureType = 1000413003
	PHYSICAL_DEVICE_VULKAN_1_4_FEATURES                                   StructureType = 55
	PHYSICAL_DEVICE_VULKAN_1_4_PROPERTIES                                 StructureType = 56
	DEVICE_QUEUE_GLOBAL_PRIORITY_CREATE_INFO                              StructureType = 1000174000
	PHYSICAL_DEVICE_GLOBAL_PRIORITY_QUERY_FEATURES                        StructureType = 1000388000
	QUEUE_FAMILY_GLOBAL_PRIORITY_PROPERTIES                               StructureType = 1000388001
	PHYSICAL_DEVICE_SHADER_SUBGROUP_ROTATE_FEATURES                       StructureType = 1000416000
	PHYSICAL_DEVICE_SHADER_FLOAT_CONTROLS_2_FEATURES                      StructureType = 1000528000
	PHYSICAL_DEVICE_SHADER_EXPECT_ASSUME_FEATURES                         StructureType = 1000544000
	PHYSICAL_DEVICE_LINE_RASTERIZATION_FEATURES                           StructureType = 1000259000
	PIPELINE_RASTERIZATION_LINE_STATE_CREATE_INFO                         StructureType = 1000259001
	PHYSICAL_DEVICE_LINE_RASTERIZATION_PROPERTIES                         StructureType = 1000259002
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_DIVISOR_PROPERTIES                   StructureType = 1000525000
	PIPELINE_VERTEX_INPUT_DIVISOR_STATE_CREATE_INFO                       StructureType = 1000190001
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_DIVISOR_FEATURES                     StructureType = 1000190002
	PHYSICAL_DEVICE_INDEX_TYPE_UINT8_FEATURES                             StructureType = 1000265000
	MEMORY_MAP_INFO                                                       StructureType = 1000271000
	MEMORY_UNMAP_INFO                                                     StructureType = 1000271001
	PHYSICAL_DEVICE_MAINTENANCE_5_FEATURES                                StructureType = 1000470000
	PHYSICAL_DEVICE_MAINTENANCE_5_PROPERTIES                              StructureType = 1000470001
	RENDERING_AREA_INFO                                                   StructureType = 1000470003
	DEVICE_IMAGE_SUBRESOURCE_INFO                                         StructureType = 1000470004
	SUBRESOURCE_LAYOUT_2                                                  StructureType = 1000338002
	IMAGE_SUBRESOURCE_2                                                   StructureType = 1000338003
	PIPELINE_CREATE_FLAGS_2_CREATE_INFO                                   StructureType = 1000470005
	BUFFER_USAGE_FLAGS_2_CREATE_INFO                                      StructureType = 1000470006
	PHYSICAL_DEVICE_PUSH_DESCRIPTOR_PROPERTIES                            StructureType = 1000080000
	PHYSICAL_DEVICE_DYNAMIC_RENDERING_LOCAL_READ_FEATURES                 StructureType = 1000232000
	RENDERING_ATTACHMENT_LOCATION_INFO                                    StructureType = 1000232001
	RENDERING_INPUT_ATTACHMENT_INDEX_INFO                                 StructureType = 1000232002
	PHYSICAL_DEVICE_MAINTENANCE_6_FEATURES                                StructureType = 1000545000
	PHYSICAL_DEVICE_MAINTENANCE_6_PROPERTIES                              StructureType = 1000545001
	BIND_MEMORY_STATUS                                                    StructureType = 1000545002
	BIND_DESCRIPTOR_SETS_INFO                                             StructureType = 1000545003
	PUSH_CONSTANTS_INFO                                                   StructureType = 1000545004
	PUSH_DESCRIPTOR_SET_INFO                                              StructureType = 1000545005
	PUSH_DESCRIPTOR_SET_WITH_TEMPLATE_INFO                                StructureType = 1000545006
	PHYSICAL_DEVICE_PIPELINE_PROTECTED_ACCESS_FEATURES                    StructureType = 1000466000
	PIPELINE_ROBUSTNESS_CREATE_INFO                                       StructureType = 1000068000
	PHYSICAL_DEVICE_PIPELINE_ROBUSTNESS_FEATURES                          StructureType = 1000068001
	PHYSICAL_DEVICE_PIPELINE_ROBUSTNESS_PROPERTIES                        StructureType = 1000068002
	PHYSICAL_DEVICE_HOST_IMAGE_COPY_FEATURES                              StructureType = 1000270000
	PHYSICAL_DEVICE_HOST_IMAGE_COPY_PROPERTIES                            StructureType = 1000270001
	MEMORY_TO_IMAGE_COPY                                                  StructureType = 1000270002
	IMAGE_TO_MEMORY_COPY                                                  StructureType = 1000270003
	COPY_IMAGE_TO_MEMORY_INFO                                             StructureType = 1000270004
	COPY_MEMORY_TO_IMAGE_INFO                                             StructureType = 1000270005
	HOST_IMAGE_LAYOUT_TRANSITION_INFO                                     StructureType = 1000270006
	COPY_IMAGE_TO_IMAGE_INFO                                              StructureType = 1000270007
	SUBRESOURCE_HOST_MEMCPY_SIZE                                          StructureType = 1000270008
	HOST_IMAGE_COPY_DEVICE_PERFORMANCE_QUERY                              StructureType = 1000270009
	SWAPCHAIN_CREATE_INFO_KHR                                             StructureType = 1000001000
	PRESENT_INFO_KHR                                                      StructureType = 1000001001
	DEVICE_GROUP_PRESENT_CAPABILITIES_KHR                                 StructureType = 1000060007
	IMAGE_SWAPCHAIN_CREATE_INFO_KHR                                       StructureType = 1000060008
	BIND_IMAGE_MEMORY_SWAPCHAIN_INFO_KHR                                  StructureType = 1000060009
	ACQUIRE_NEXT_IMAGE_INFO_KHR                                           StructureType = 1000060010
	DEVICE_GROUP_PRESENT_INFO_KHR                                         StructureType = 1000060011
	DEVICE_GROUP_SWAPCHAIN_CREATE_INFO_KHR                                StructureType = 1000060012
	DISPLAY_MODE_CREATE_INFO_KHR                                          StructureType = 1000002000
	DISPLAY_SURFACE_CREATE_INFO_KHR                                       StructureType = 1000002001
	DISPLAY_PRESENT_INFO_KHR                                              StructureType = 1000003000
	XLIB_SURFACE_CREATE_INFO_KHR                                          StructureType = 1000004000
	XCB_SURFACE_CREATE_INFO_KHR                                           StructureType = 1000005000
	WAYLAND_SURFACE_CREATE_INFO_KHR                                       StructureType = 1000006000
	ANDROID_SURFACE_CREATE_INFO_KHR                                       StructureType = 1000008000
	WIN32_SURFACE_CREATE_INFO_KHR                                         StructureType = 1000009000
	DEBUG_REPORT_CALLBACK_CREATE_INFO_EXT                                 StructureType = 1000011000
	PIPELINE_RASTERIZATION_STATE_RASTERIZATION_ORDER_AMD                  StructureType = 1000018000
	DEBUG_MARKER_OBJECT_NAME_INFO_EXT                                     StructureType = 1000022000
	DEBUG_MARKER_OBJECT_TAG_INFO_EXT                                      StructureType = 1000022001
	DEBUG_MARKER_MARKER_INFO_EXT                                          StructureType = 1000022002
	VIDEO_PROFILE_INFO_KHR                                                StructureType = 1000023000
	VIDEO_CAPABILITIES_KHR                                                StructureType = 1000023001
	VIDEO_PICTURE_RESOURCE_INFO_KHR                                       StructureType = 1000023002
	VIDEO_SESSION_MEMORY_REQUIREMENTS_KHR                                 StructureType = 1000023003
	BIND_VIDEO_SESSION_MEMORY_INFO_KHR                                    StructureType = 1000023004
	VIDEO_SESSION_CREATE_INFO_KHR                                         StructureType = 1000023005
	VIDEO_SESSION_PARAMETERS_CREATE_INFO_KHR                              StructureType = 1000023006
	VIDEO_SESSION_PARAMETERS_UPDATE_INFO_KHR                              StructureType = 1000023007
	VIDEO_BEGIN_CODING_INFO_KHR                                           StructureType = 1000023008
	VIDEO_END_CODING_INFO_KHR                                             StructureType = 1000023009
	VIDEO_CODING_CONTROL_INFO_KHR                                         StructureType = 1000023010
	VIDEO_REFERENCE_SLOT_INFO_KHR                                         StructureType = 1000023011
	QUEUE_FAMILY_VIDEO_PROPERTIES_KHR                                     StructureType = 1000023012
	VIDEO_PROFILE_LIST_INFO_KHR                                           StructureType = 1000023013
	PHYSICAL_DEVICE_VIDEO_FORMAT_INFO_KHR                                 StructureType = 1000023014
	VIDEO_FORMAT_PROPERTIES_KHR                                           StructureType = 1000023015
	QUEUE_FAMILY_QUERY_RESULT_STATUS_PROPERTIES_KHR                       StructureType = 1000023016
	VIDEO_DECODE_INFO_KHR                                                 StructureType = 1000024000
	VIDEO_DECODE_CAPABILITIES_KHR                                         StructureType = 1000024001
	VIDEO_DECODE_USAGE_INFO_KHR                                           StructureType = 1000024002
	DEDICATED_ALLOCATION_IMAGE_CREATE_INFO_NV                             StructureType = 1000026000
	DEDICATED_ALLOCATION_BUFFER_CREATE_INFO_NV                            StructureType = 1000026001
	DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV                          StructureType = 1000026002
	PHYSICAL_DEVICE_TRANSFORM_FEEDBACK_FEATURES_EXT                       StructureType = 1000028000
	PHYSICAL_DEVICE_TRANSFORM_FEEDBACK_PROPERTIES_EXT                     StructureType = 1000028001
	PIPELINE_RASTERIZATION_STATE_STREAM_CREATE_INFO_EXT                   StructureType = 1000028002
	CU_MODULE_CREATE_INFO_NVX                                             StructureType = 1000029000
	CU_FUNCTION_CREATE_INFO_NVX                                           StructureType = 1000029001
	CU_LAUNCH_INFO_NVX                                                    StructureType = 1000029002
	CU_MODULE_TEXTURING_MODE_CREATE_INFO_NVX                              StructureType = 1000029004
	IMAGE_VIEW_HANDLE_INFO_NVX                                            StructureType = 1000030000
	IMAGE_VIEW_ADDRESS_PROPERTIES_NVX                                     StructureType = 1000030001
	VIDEO_ENCODE_H264_CAPABILITIES_KHR                                    StructureType = 1000038000
	VIDEO_ENCODE_H264_SESSION_PARAMETERS_CREATE_INFO_KHR                  StructureType = 1000038001
	VIDEO_ENCODE_H264_SESSION_PARAMETERS_ADD_INFO_KHR                     StructureType = 1000038002
	VIDEO_ENCODE_H264_PICTURE_INFO_KHR                                    StructureType = 1000038003
	VIDEO_ENCODE_H264_DPB_SLOT_INFO_KHR                                   StructureType = 1000038004
	VIDEO_ENCODE_H264_NALU_SLICE_INFO_KHR                                 StructureType = 1000038005
	VIDEO_ENCODE_H264_GOP_REMAINING_FRAME_INFO_KHR                        StructureType = 1000038006
	VIDEO_ENCODE_H264_PROFILE_INFO_KHR                                    StructureType = 1000038007
	VIDEO_ENCODE_H264_RATE_CONTROL_INFO_KHR                               StructureType = 1000038008
	VIDEO_ENCODE_H264_RATE_CONTROL_LAYER_INFO_KHR                         StructureType = 1000038009
	VIDEO_ENCODE_H264_SESSION_CREATE_INFO_KHR                             StructureType = 1000038010
	VIDEO_ENCODE_H264_QUALITY_LEVEL_PROPERTIES_KHR                        StructureType = 1000038011
	VIDEO_ENCODE_H264_SESSION_PARAMETERS_GET_INFO_KHR                     StructureType = 1000038012
	VIDEO_ENCODE_H264_SESSION_PARAMETERS_FEEDBACK_INFO_KHR                StructureType = 1000038013
	VIDEO_ENCODE_H265_CAPABILITIES_KHR                                    StructureType = 1000039000
	VIDEO_ENCODE_H265_SESSION_PARAMETERS_CREATE_INFO_KHR                  StructureType = 1000039001
	VIDEO_ENCODE_H265_SESSION_PARAMETERS_ADD_INFO_KHR                     StructureType = 1000039002
	VIDEO_ENCODE_H265_PICTURE_INFO_KHR                                    StructureType = 1000039003
	VIDEO_ENCODE_H265_DPB_SLOT_INFO_KHR                                   StructureType = 1000039004
	VIDEO_ENCODE_H265_NALU_SLICE_SEGMENT_INFO_KHR                         StructureType = 1000039005
	VIDEO_ENCODE_H265_GOP_REMAINING_FRAME_INFO_KHR                        StructureType = 1000039006
	VIDEO_ENCODE_H265_PROFILE_INFO_KHR                                    StructureType = 1000039007
	VIDEO_ENCODE_H265_RATE_CONTROL_INFO_KHR                               StructureType = 1000039009
	VIDEO_ENCODE_H265_RATE_CONTROL_LAYER_INFO_KHR                         StructureType = 1000039010
	VIDEO_ENCODE_H265_SESSION_CREATE_INFO_KHR                             StructureType = 1000039011
	VIDEO_ENCODE_H265_QUALITY_LEVEL_PROPERTIES_KHR                        StructureType = 1000039012
	VIDEO_ENCODE_H265_SESSION_PARAMETERS_GET_INFO_KHR                     StructureType = 1000039013
	VIDEO_ENCODE_H265_SESSION_PARAMETERS_FEEDBACK_INFO_KHR                StructureType = 1000039014
	VIDEO_DECODE_H264_CAPABILITIES_KHR                                    StructureType = 1000040000
	VIDEO_DECODE_H264_PICTURE_INFO_KHR                                    StructureType = 1000040001
	VIDEO_DECODE_H264_PROFILE_INFO_KHR                                    StructureType = 1000040003
	VIDEO_DECODE_H264_SESSION_PARAMETERS_CREATE_INFO_KHR                  StructureType = 1000040004
	VIDEO_DECODE_H264_SESSION_PARAMETERS_ADD_INFO_KHR                     StructureType = 1000040005
	VIDEO_DECODE_H264_DPB_SLOT_INFO_KHR                                   StructureType = 1000040006
	TEXTURE_LOD_GATHER_FORMAT_PROPERTIES_AMD                              StructureType = 1000041000
	STREAM_DESCRIPTOR_SURFACE_CREATE_INFO_GGP                             StructureType = 1000049000
	PHYSICAL_DEVICE_CORNER_SAMPLED_IMAGE_FEATURES_NV                      StructureType = 1000050000
	EXTERNAL_MEMORY_IMAGE_CREATE_INFO_NV                                  StructureType = 1000056000
	EXPORT_MEMORY_ALLOCATE_INFO_NV                                        StructureType = 1000056001
	IMPORT_MEMORY_WIN32_HANDLE_INFO_NV                                    StructureType = 1000057000
	EXPORT_MEMORY_WIN32_HANDLE_INFO_NV                                    StructureType = 1000057001
	WIN32_KEYED_MUTEX_ACQUIRE_RELEASE_INFO_NV                             StructureType = 1000058000
	VALIDATION_FLAGS_EXT                                                  StructureType = 1000061000
	VI_SURFACE_CREATE_INFO_NN                                             StructureType = 1000062000
	IMAGE_VIEW_ASTC_DECODE_MODE_EXT                                       StructureType = 1000067000
	PHYSICAL_DEVICE_ASTC_DECODE_FEATURES_EXT                              StructureType = 1000067001
	IMPORT_MEMORY_WIN32_HANDLE_INFO_KHR                                   StructureType = 1000073000
	EXPORT_MEMORY_WIN32_HANDLE_INFO_KHR                                   StructureType = 1000073001
	MEMORY_WIN32_HANDLE_PROPERTIES_KHR                                    StructureType = 1000073002
	MEMORY_GET_WIN32_HANDLE_INFO_KHR                                      StructureType = 1000073003
	IMPORT_MEMORY_FD_INFO_KHR                                             StructureType = 1000074000
	MEMORY_FD_PROPERTIES_KHR                                              StructureType = 1000074001
	MEMORY_GET_FD_INFO_KHR                                                StructureType = 1000074002
	WIN32_KEYED_MUTEX_ACQUIRE_RELEASE_INFO_KHR                            StructureType = 1000075000
	IMPORT_SEMAPHORE_WIN32_HANDLE_INFO_KHR                                StructureType = 1000078000
	EXPORT_SEMAPHORE_WIN32_HANDLE_INFO_KHR                                StructureType = 1000078001
	D3D12_FENCE_SUBMIT_INFO_KHR                                           StructureType = 1000078002
	SEMAPHORE_GET_WIN32_HANDLE_INFO_KHR                                   StructureType = 1000078003
	IMPORT_SEMAPHORE_FD_INFO_KHR                                          StructureType = 1000079000
	SEMAPHORE_GET_FD_INFO_KHR                                             StructureType = 1000079001
	COMMAND_BUFFER_INHERITANCE_CONDITIONAL_RENDERING_INFO_EXT             StructureType = 1000081000
	PHYSICAL_DEVICE_CONDITIONAL_RENDERING_FEATURES_EXT                    StructureType = 1000081001
	CONDITIONAL_RENDERING_BEGIN_INFO_EXT                                  StructureType = 1000081002
	PRESENT_REGIONS_KHR                                                   StructureType = 1000084000
	PIPELINE_VIEWPORT_W_SCALING_STATE_CREATE_INFO_NV                      StructureType = 1000087000
	SURFACE_CAPABILITIES_2_EXT                                            StructureType = 1000090000
	DISPLAY_POWER_INFO_EXT                                                StructureType = 1000091000
	DEVICE_EVENT_INFO_EXT                                                 StructureType = 1000091001
	DISPLAY_EVENT_INFO_EXT                                                StructureType = 1000091002
	SWAPCHAIN_COUNTER_CREATE_INFO_EXT                                     StructureType = 1000091003
	PRESENT_TIMES_INFO_GOOGLE                                             StructureType = 1000092000
	PHYSICAL_DEVICE_MULTIVIEW_PER_VIEW_ATTRIBUTES_PROPERTIES_NVX          StructureType = 1000097000
	MULTIVIEW_PER_VIEW_ATTRIBUTES_INFO_NVX                                StructureType = 1000044009
	PIPELINE_VIEWPORT_SWIZZLE_STATE_CREATE_INFO_NV                        StructureType = 1000098000
	PHYSICAL_DEVICE_DISCARD_RECTANGLE_PROPERTIES_EXT                      StructureType = 1000099000
	PIPELINE_DISCARD_RECTANGLE_STATE_CREATE_INFO_EXT                      StructureType = 1000099001
	PHYSICAL_DEVICE_CONSERVATIVE_RASTERIZATION_PROPERTIES_EXT             StructureType = 1000101000
	PIPELINE_RASTERIZATION_CONSERVATIVE_STATE_CREATE_INFO_EXT             StructureType = 1000101001
	PHYSICAL_DEVICE_DEPTH_CLIP_ENABLE_FEATURES_EXT                        StructureType = 1000102000
	PIPELINE_RASTERIZATION_DEPTH_CLIP_STATE_CREATE_INFO_EXT               StructureType = 1000102001
	HDR_METADATA_EXT                                                      StructureType = 1000105000
	PHYSICAL_DEVICE_RELAXED_LINE_RASTERIZATION_FEATURES_IMG               StructureType = 1000110000
	SHARED_PRESENT_SURFACE_CAPABILITIES_KHR                               StructureType = 1000111000
	IMPORT_FENCE_WIN32_HANDLE_INFO_KHR                                    StructureType = 1000114000
	EXPORT_FENCE_WIN32_HANDLE_INFO_KHR                                    StructureType = 1000114001
	FENCE_GET_WIN32_HANDLE_INFO_KHR                                       StructureType = 1000114002
	IMPORT_FENCE_FD_INFO_KHR                                              StructureType = 1000115000
	FENCE_GET_FD_INFO_KHR                                                 StructureType = 1000115001
	PHYSICAL_DEVICE_PERFORMANCE_QUERY_FEATURES_KHR                        StructureType = 1000116000
	PHYSICAL_DEVICE_PERFORMANCE_QUERY_PROPERTIES_KHR                      StructureType = 1000116001
	QUERY_POOL_PERFORMANCE_CREATE_INFO_KHR                                StructureType = 1000116002
	PERFORMANCE_QUERY_SUBMIT_INFO_KHR                                     StructureType = 1000116003
	ACQUIRE_PROFILING_LOCK_INFO_KHR                                       StructureType = 1000116004
	PERFORMANCE_COUNTER_KHR                                               StructureType = 1000116005
	PERFORMANCE_COUNTER_DESCRIPTION_KHR                                   StructureType = 1000116006
	PHYSICAL_DEVICE_SURFACE_INFO_2_KHR                                    StructureType = 1000119000
	SURFACE_CAPABILITIES_2_KHR                                            StructureType = 1000119001
	SURFACE_FORMAT_2_KHR                                                  StructureType = 1000119002
	DISPLAY_PROPERTIES_2_KHR                                              StructureType = 1000121000
	DISPLAY_PLANE_PROPERTIES_2_KHR                                        StructureType = 1000121001
	DISPLAY_MODE_PROPERTIES_2_KHR                                         StructureType = 1000121002
	DISPLAY_PLANE_INFO_2_KHR                                              StructureType = 1000121003
	DISPLAY_PLANE_CAPABILITIES_2_KHR                                      StructureType = 1000121004
	IOS_SURFACE_CREATE_INFO_MVK                                           StructureType = 1000122000
	MACOS_SURFACE_CREATE_INFO_MVK                                         StructureType = 1000123000
	DEBUG_UTILS_OBJECT_NAME_INFO_EXT                                      StructureType = 1000128000
	DEBUG_UTILS_OBJECT_TAG_INFO_EXT                                       StructureType = 1000128001
	DEBUG_UTILS_LABEL_EXT                                                 StructureType = 1000128002
	DEBUG_UTILS_MESSENGER_CALLBACK_DATA_EXT                               StructureType = 1000128003
	DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT                                 StructureType = 1000128004
	ANDROID_HARDWARE_BUFFER_USAGE_ANDROID                                 StructureType = 1000129000
	ANDROID_HARDWARE_BUFFER_PROPERTIES_ANDROID                            StructureType = 1000129001
	ANDROID_HARDWARE_BUFFER_FORMAT_PROPERTIES_ANDROID                     StructureType = 1000129002
	IMPORT_ANDROID_HARDWARE_BUFFER_INFO_ANDROID                           StructureType = 1000129003
	MEMORY_GET_ANDROID_HARDWARE_BUFFER_INFO_ANDROID                       StructureType = 1000129004
	EXTERNAL_FORMAT_ANDROID                                               StructureType = 1000129005
	ANDROID_HARDWARE_BUFFER_FORMAT_PROPERTIES_2_ANDROID                   StructureType = 1000129006
	PHYSICAL_DEVICE_SHADER_ENQUEUE_FEATURES_AMDX                          StructureType = 1000134000
	PHYSICAL_DEVICE_SHADER_ENQUEUE_PROPERTIES_AMDX                        StructureType = 1000134001
	EXECUTION_GRAPH_PIPELINE_SCRATCH_SIZE_AMDX                            StructureType = 1000134002
	EXECUTION_GRAPH_PIPELINE_CREATE_INFO_AMDX                             StructureType = 1000134003
	PIPELINE_SHADER_STAGE_NODE_CREATE_INFO_AMDX                           StructureType = 1000134004
	ATTACHMENT_SAMPLE_COUNT_INFO_AMD                                      StructureType = 1000044008
	PHYSICAL_DEVICE_SHADER_BFLOAT16_FEATURES_KHR                          StructureType = 1000141000
	SAMPLE_LOCATIONS_INFO_EXT                                             StructureType = 1000143000
	RENDER_PASS_SAMPLE_LOCATIONS_BEGIN_INFO_EXT                           StructureType = 1000143001
	PIPELINE_SAMPLE_LOCATIONS_STATE_CREATE_INFO_EXT                       StructureType = 1000143002
	PHYSICAL_DEVICE_SAMPLE_LOCATIONS_PROPERTIES_EXT                       StructureType = 1000143003
	MULTISAMPLE_PROPERTIES_EXT                                            StructureType = 1000143004
	PHYSICAL_DEVICE_BLEND_OPERATION_ADVANCED_FEATURES_EXT                 StructureType = 1000148000
	PHYSICAL_DEVICE_BLEND_OPERATION_ADVANCED_PROPERTIES_EXT               StructureType = 1000148001
	PIPELINE_COLOR_BLEND_ADVANCED_STATE_CREATE_INFO_EXT                   StructureType = 1000148002
	PIPELINE_COVERAGE_TO_COLOR_STATE_CREATE_INFO_NV                       StructureType = 1000149000
	WRITE_DESCRIPTOR_SET_ACCELERATION_STRUCTURE_KHR                       StructureType = 1000150007
	ACCELERATION_STRUCTURE_BUILD_GEOMETRY_INFO_KHR                        StructureType = 1000150000
	ACCELERATION_STRUCTURE_DEVICE_ADDRESS_INFO_KHR                        StructureType = 1000150002
	ACCELERATION_STRUCTURE_GEOMETRY_AABBS_DATA_KHR                        StructureType = 1000150003
	ACCELERATION_STRUCTURE_GEOMETRY_INSTANCES_DATA_KHR                    StructureType = 1000150004
	ACCELERATION_STRUCTURE_GEOMETRY_TRIANGLES_DATA_KHR                    StructureType = 1000150005
	ACCELERATION_STRUCTURE_GEOMETRY_KHR                                   StructureType = 1000150006
	ACCELERATION_STRUCTURE_VERSION_INFO_KHR                               StructureType = 1000150009
	COPY_ACCELERATION_STRUCTURE_INFO_KHR                                  StructureType = 1000150010
	COPY_ACCELERATION_STRUCTURE_TO_MEMORY_INFO_KHR                        StructureType = 1000150011
	COPY_MEMORY_TO_ACCELERATION_STRUCTURE_INFO_KHR                        StructureType = 1000150012
	PHYSICAL_DEVICE_ACCELERATION_STRUCTURE_FEATURES_KHR                   StructureType = 1000150013
	PHYSICAL_DEVICE_ACCELERATION_STRUCTURE_PROPERTIES_KHR                 StructureType = 1000150014
	ACCELERATION_STRUCTURE_CREATE_INFO_KHR                                StructureType = 1000150017
	ACCELERATION_STRUCTURE_BUILD_SIZES_INFO_KHR                           StructureType = 1000150020
	PHYSICAL_DEVICE_RAY_TRACING_PIPELINE_FEATURES_KHR                     StructureType = 1000347000
	PHYSICAL_DEVICE_RAY_TRACING_PIPELINE_PROPERTIES_KHR                   StructureType = 1000347001
	RAY_TRACING_PIPELINE_CREATE_INFO_KHR                                  StructureType = 1000150015
	RAY_TRACING_SHADER_GROUP_CREATE_INFO_KHR                              StructureType = 1000150016
	RAY_TRACING_PIPELINE_INTERFACE_CREATE_INFO_KHR                        StructureType = 1000150018
	PHYSICAL_DEVICE_RAY_QUERY_FEATURES_KHR                                StructureType = 1000348013
	PIPELINE_COVERAGE_MODULATION_STATE_CREATE_INFO_NV                     StructureType = 1000152000
	PHYSICAL_DEVICE_SHADER_SM_BUILTINS_FEATURES_NV                        StructureType = 1000154000
	PHYSICAL_DEVICE_SHADER_SM_BUILTINS_PROPERTIES_NV                      StructureType = 1000154001
	DRM_FORMAT_MODIFIER_PROPERTIES_LIST_EXT                               StructureType = 1000158000
	PHYSICAL_DEVICE_IMAGE_DRM_FORMAT_MODIFIER_INFO_EXT                    StructureType = 1000158002
	IMAGE_DRM_FORMAT_MODIFIER_LIST_CREATE_INFO_EXT                        StructureType = 1000158003
	IMAGE_DRM_FORMAT_MODIFIER_EXPLICIT_CREATE_INFO_EXT                    StructureType = 1000158004
	IMAGE_DRM_FORMAT_MODIFIER_PROPERTIES_EXT                              StructureType = 1000158005
	DRM_FORMAT_MODIFIER_PROPERTIES_LIST_2_EXT                             StructureType = 1000158006
	VALIDATION_CACHE_CREATE_INFO_EXT                                      StructureType = 1000160000
	SHADER_MODULE_VALIDATION_CACHE_CREATE_INFO_EXT                        StructureType = 1000160001
	PHYSICAL_DEVICE_PORTABILITY_SUBSET_FEATURES_KHR                       StructureType = 1000163000
	PHYSICAL_DEVICE_PORTABILITY_SUBSET_PROPERTIES_KHR                     StructureType = 1000163001
	PIPELINE_VIEWPORT_SHADING_RATE_IMAGE_STATE_CREATE_INFO_NV             StructureType = 1000164000
	PHYSICAL_DEVICE_SHADING_RATE_IMAGE_FEATURES_NV                        StructureType = 1000164001
	PHYSICAL_DEVICE_SHADING_RATE_IMAGE_PROPERTIES_NV                      StructureType = 1000164002
	PIPELINE_VIEWPORT_COARSE_SAMPLE_ORDER_STATE_CREATE_INFO_NV            StructureType = 1000164005
	RAY_TRACING_PIPELINE_CREATE_INFO_NV                                   StructureType = 1000165000
	ACCELERATION_STRUCTURE_CREATE_INFO_NV                                 StructureType = 1000165001
	GEOMETRY_NV                                                           StructureType = 1000165003
	GEOMETRY_TRIANGLES_NV                                                 StructureType = 1000165004
	GEOMETRY_AABB_NV                                                      StructureType = 1000165005
	BIND_ACCELERATION_STRUCTURE_MEMORY_INFO_NV                            StructureType = 1000165006
	WRITE_DESCRIPTOR_SET_ACCELERATION_STRUCTURE_NV                        StructureType = 1000165007
	ACCELERATION_STRUCTURE_MEMORY_REQUIREMENTS_INFO_NV                    StructureType = 1000165008
	PHYSICAL_DEVICE_RAY_TRACING_PROPERTIES_NV                             StructureType = 1000165009
	RAY_TRACING_SHADER_GROUP_CREATE_INFO_NV                               StructureType = 1000165011
	ACCELERATION_STRUCTURE_INFO_NV                                        StructureType = 1000165012
	PHYSICAL_DEVICE_REPRESENTATIVE_FRAGMENT_TEST_FEATURES_NV              StructureType = 1000166000
	PIPELINE_REPRESENTATIVE_FRAGMENT_TEST_STATE_CREATE_INFO_NV            StructureType = 1000166001
	PHYSICAL_DEVICE_IMAGE_VIEW_IMAGE_FORMAT_INFO_EXT                      StructureType = 1000170000
	FILTER_CUBIC_IMAGE_VIEW_IMAGE_FORMAT_PROPERTIES_EXT                   StructureType = 1000170001
	IMPORT_MEMORY_HOST_POINTER_INFO_EXT                                   StructureType = 1000178000
	MEMORY_HOST_POINTER_PROPERTIES_EXT                                    StructureType = 1000178001
	PHYSICAL_DEVICE_EXTERNAL_MEMORY_HOST_PROPERTIES_EXT                   StructureType = 1000178002
	PHYSICAL_DEVICE_SHADER_CLOCK_FEATURES_KHR                             StructureType = 1000181000
	PIPELINE_COMPILER_CONTROL_CREATE_INFO_AMD                             StructureType = 1000183000
	PHYSICAL_DEVICE_SHADER_CORE_PROPERTIES_AMD                            StructureType = 1000185000
	VIDEO_DECODE_H265_CAPABILITIES_KHR                                    StructureType = 1000187000
	VIDEO_DECODE_H265_SESSION_PARAMETERS_CREATE_INFO_KHR                  StructureType = 1000187001
	VIDEO_DECODE_H265_SESSION_PARAMETERS_ADD_INFO_KHR                     StructureType = 1000187002
	VIDEO_DECODE_H265_PROFILE_INFO_KHR                                    StructureType = 1000187003
	VIDEO_DECODE_H265_PICTURE_INFO_KHR                                    StructureType = 1000187004
	VIDEO_DECODE_H265_DPB_SLOT_INFO_KHR                                   StructureType = 1000187005
	DEVICE_MEMORY_OVERALLOCATION_CREATE_INFO_AMD                          StructureType = 1000189000
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_DIVISOR_PROPERTIES_EXT               StructureType = 1000190000
	PRESENT_FRAME_TOKEN_GGP                                               StructureType = 1000191000
	PHYSICAL_DEVICE_MESH_SHADER_FEATURES_NV                               StructureType = 1000202000
	PHYSICAL_DEVICE_MESH_SHADER_PROPERTIES_NV                             StructureType = 1000202001
	PHYSICAL_DEVICE_SHADER_IMAGE_FOOTPRINT_FEATURES_NV                    StructureType = 1000204000
	PIPELINE_VIEWPORT_EXCLUSIVE_SCISSOR_STATE_CREATE_INFO_NV              StructureType = 1000205000
	PHYSICAL_DEVICE_EXCLUSIVE_SCISSOR_FEATURES_NV                         StructureType = 1000205002
	CHECKPOINT_DATA_NV                                                    StructureType = 1000206000
	QUEUE_FAMILY_CHECKPOINT_PROPERTIES_NV                                 StructureType = 1000206001
	QUEUE_FAMILY_CHECKPOINT_PROPERTIES_2_NV                               StructureType = 1000314008
	CHECKPOINT_DATA_2_NV                                                  StructureType = 1000314009
	PHYSICAL_DEVICE_SHADER_INTEGER_FUNCTIONS_2_FEATURES_INTEL             StructureType = 1000209000
	QUERY_POOL_PERFORMANCE_QUERY_CREATE_INFO_INTEL                        StructureType = 1000210000
	INITIALIZE_PERFORMANCE_API_INFO_INTEL                                 StructureType = 1000210001
	PERFORMANCE_MARKER_INFO_INTEL                                         StructureType = 1000210002
	PERFORMANCE_STREAM_MARKER_INFO_INTEL                                  StructureType = 1000210003
	PERFORMANCE_OVERRIDE_INFO_INTEL                                       StructureType = 1000210004
	PERFORMANCE_CONFIGURATION_ACQUIRE_INFO_INTEL                          StructureType = 1000210005
	PHYSICAL_DEVICE_PCI_BUS_INFO_PROPERTIES_EXT                           StructureType = 1000212000
	DISPLAY_NATIVE_HDR_SURFACE_CAPABILITIES_AMD                           StructureType = 1000213000
	SWAPCHAIN_DISPLAY_NATIVE_HDR_CREATE_INFO_AMD                          StructureType = 1000213001
	IMAGEPIPE_SURFACE_CREATE_INFO_FUCHSIA                                 StructureType = 1000214000
	METAL_SURFACE_CREATE_INFO_EXT                                         StructureType = 1000217000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_FEATURES_EXT                     StructureType = 1000218000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_PROPERTIES_EXT                   StructureType = 1000218001
	RENDER_PASS_FRAGMENT_DENSITY_MAP_CREATE_INFO_EXT                      StructureType = 1000218002
	RENDERING_FRAGMENT_DENSITY_MAP_ATTACHMENT_INFO_EXT                    StructureType = 1000044007
	FRAGMENT_SHADING_RATE_ATTACHMENT_INFO_KHR                             StructureType = 1000226000
	PIPELINE_FRAGMENT_SHADING_RATE_STATE_CREATE_INFO_KHR                  StructureType = 1000226001
	PHYSICAL_DEVICE_FRAGMENT_SHADING_RATE_PROPERTIES_KHR                  StructureType = 1000226002
	PHYSICAL_DEVICE_FRAGMENT_SHADING_RATE_FEATURES_KHR                    StructureType = 1000226003
	PHYSICAL_DEVICE_FRAGMENT_SHADING_RATE_KHR                             StructureType = 1000226004
	RENDERING_FRAGMENT_SHADING_RATE_ATTACHMENT_INFO_KHR                   StructureType = 1000044006
	PHYSICAL_DEVICE_SHADER_CORE_PROPERTIES_2_AMD                          StructureType = 1000227000
	PHYSICAL_DEVICE_COHERENT_MEMORY_FEATURES_AMD                          StructureType = 1000229000
	PHYSICAL_DEVICE_SHADER_IMAGE_ATOMIC_INT64_FEATURES_EXT                StructureType = 1000234000
	PHYSICAL_DEVICE_SHADER_QUAD_CONTROL_FEATURES_KHR                      StructureType = 1000235000
	PHYSICAL_DEVICE_MEMORY_BUDGET_PROPERTIES_EXT                          StructureType = 1000237000
	PHYSICAL_DEVICE_MEMORY_PRIORITY_FEATURES_EXT                          StructureType = 1000238000
	MEMORY_PRIORITY_ALLOCATE_INFO_EXT                                     StructureType = 1000238001
	SURFACE_PROTECTED_CAPABILITIES_KHR                                    StructureType = 1000239000
	PHYSICAL_DEVICE_DEDICATED_ALLOCATION_IMAGE_ALIASING_FEATURES_NV       StructureType = 1000240000
	PHYSICAL_DEVICE_BUFFER_DEVICE_ADDRESS_FEATURES_EXT                    StructureType = 1000244000
	BUFFER_DEVICE_ADDRESS_CREATE_INFO_EXT                                 StructureType = 1000244002
	VALIDATION_FEATURES_EXT                                               StructureType = 1000247000
	PHYSICAL_DEVICE_PRESENT_WAIT_FEATURES_KHR                             StructureType = 1000248000
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_FEATURES_NV                        StructureType = 1000249000
	COOPERATIVE_MATRIX_PROPERTIES_NV                                      StructureType = 1000249001
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_PROPERTIES_NV                      StructureType = 1000249002
	PHYSICAL_DEVICE_COVERAGE_REDUCTION_MODE_FEATURES_NV                   StructureType = 1000250000
	PIPELINE_COVERAGE_REDUCTION_STATE_CREATE_INFO_NV                      StructureType = 1000250001
	FRAMEBUFFER_MIXED_SAMPLES_COMBINATION_NV                              StructureType = 1000250002
	PHYSICAL_DEVICE_FRAGMENT_SHADER_INTERLOCK_FEATURES_EXT                StructureType = 1000251000
	PHYSICAL_DEVICE_YCBCR_IMAGE_ARRAYS_FEATURES_EXT                       StructureType = 1000252000
	PHYSICAL_DEVICE_PROVOKING_VERTEX_FEATURES_EXT                         StructureType = 1000254000
	PIPELINE_RASTERIZATION_PROVOKING_VERTEX_STATE_CREATE_INFO_EXT         StructureType = 1000254001
	PHYSICAL_DEVICE_PROVOKING_VERTEX_PROPERTIES_EXT                       StructureType = 1000254002
	SURFACE_FULL_SCREEN_EXCLUSIVE_INFO_EXT                                StructureType = 1000255000
	SURFACE_CAPABILITIES_FULL_SCREEN_EXCLUSIVE_EXT                        StructureType = 1000255002
	SURFACE_FULL_SCREEN_EXCLUSIVE_WIN32_INFO_EXT                          StructureType = 1000255001
	HEADLESS_SURFACE_CREATE_INFO_EXT                                      StructureType = 1000256000
	PHYSICAL_DEVICE_SHADER_ATOMIC_FLOAT_FEATURES_EXT                      StructureType = 1000260000
	PHYSICAL_DEVICE_EXTENDED_DYNAMIC_STATE_FEATURES_EXT                   StructureType = 1000267000
	PHYSICAL_DEVICE_PIPELINE_EXECUTABLE_PROPERTIES_FEATURES_KHR           StructureType = 1000269000
	PIPELINE_INFO_KHR                                                     StructureType = 1000269001
	PIPELINE_EXECUTABLE_PROPERTIES_KHR                                    StructureType = 1000269002
	PIPELINE_EXECUTABLE_INFO_KHR                                          StructureType = 1000269003
	PIPELINE_EXECUTABLE_STATISTIC_KHR                                     StructureType = 1000269004
	PIPELINE_EXECUTABLE_INTERNAL_REPRESENTATION_KHR                       StructureType = 1000269005
	PHYSICAL_DEVICE_MAP_MEMORY_PLACED_FEATURES_EXT                        StructureType = 1000272000
	PHYSICAL_DEVICE_MAP_MEMORY_PLACED_PROPERTIES_EXT                      StructureType = 1000272001
	MEMORY_MAP_PLACED_INFO_EXT                                            StructureType = 1000272002
	PHYSICAL_DEVICE_SHADER_ATOMIC_FLOAT_2_FEATURES_EXT                    StructureType = 1000273000
	PHYSICAL_DEVICE_DEVICE_GENERATED_COMMANDS_PROPERTIES_NV               StructureType = 1000277000
	GRAPHICS_SHADER_GROUP_CREATE_INFO_NV                                  StructureType = 1000277001
	GRAPHICS_PIPELINE_SHADER_GROUPS_CREATE_INFO_NV                        StructureType = 1000277002
	INDIRECT_COMMANDS_LAYOUT_TOKEN_NV                                     StructureType = 1000277003
	INDIRECT_COMMANDS_LAYOUT_CREATE_INFO_NV                               StructureType = 1000277004
	GENERATED_COMMANDS_INFO_NV                                            StructureType = 1000277005
	GENERATED_COMMANDS_MEMORY_REQUIREMENTS_INFO_NV                        StructureType = 1000277006
	PHYSICAL_DEVICE_DEVICE_GENERATED_COMMANDS_FEATURES_NV                 StructureType = 1000277007
	PHYSICAL_DEVICE_INHERITED_VIEWPORT_SCISSOR_FEATURES_NV                StructureType = 1000278000
	COMMAND_BUFFER_INHERITANCE_VIEWPORT_SCISSOR_INFO_NV                   StructureType = 1000278001
	PHYSICAL_DEVICE_TEXEL_BUFFER_ALIGNMENT_FEATURES_EXT                   StructureType = 1000281000
	COMMAND_BUFFER_INHERITANCE_RENDER_PASS_TRANSFORM_INFO_QCOM            StructureType = 1000282000
	RENDER_PASS_TRANSFORM_BEGIN_INFO_QCOM                                 StructureType = 1000282001
	PHYSICAL_DEVICE_DEPTH_BIAS_CONTROL_FEATURES_EXT                       StructureType = 1000283000
	DEPTH_BIAS_INFO_EXT                                                   StructureType = 1000283001
	DEPTH_BIAS_REPRESENTATION_INFO_EXT                                    StructureType = 1000283002
	PHYSICAL_DEVICE_DEVICE_MEMORY_REPORT_FEATURES_EXT                     StructureType = 1000284000
	DEVICE_DEVICE_MEMORY_REPORT_CREATE_INFO_EXT                           StructureType = 1000284001
	DEVICE_MEMORY_REPORT_CALLBACK_DATA_EXT                                StructureType = 1000284002
	SAMPLER_CUSTOM_BORDER_COLOR_CREATE_INFO_EXT                           StructureType = 1000287000
	PHYSICAL_DEVICE_CUSTOM_BORDER_COLOR_PROPERTIES_EXT                    StructureType = 1000287001
	PHYSICAL_DEVICE_CUSTOM_BORDER_COLOR_FEATURES_EXT                      StructureType = 1000287002
	PIPELINE_LIBRARY_CREATE_INFO_KHR                                      StructureType = 1000290000
	PHYSICAL_DEVICE_PRESENT_BARRIER_FEATURES_NV                           StructureType = 1000292000
	SURFACE_CAPABILITIES_PRESENT_BARRIER_NV                               StructureType = 1000292001
	SWAPCHAIN_PRESENT_BARRIER_CREATE_INFO_NV                              StructureType = 1000292002
	PRESENT_ID_KHR                                                        StructureType = 1000294000
	PHYSICAL_DEVICE_PRESENT_ID_FEATURES_KHR                               StructureType = 1000294001
	VIDEO_ENCODE_INFO_KHR                                                 StructureType = 1000299000
	VIDEO_ENCODE_RATE_CONTROL_INFO_KHR                                    StructureType = 1000299001
	VIDEO_ENCODE_RATE_CONTROL_LAYER_INFO_KHR                              StructureType = 1000299002
	VIDEO_ENCODE_CAPABILITIES_KHR                                         StructureType = 1000299003
	VIDEO_ENCODE_USAGE_INFO_KHR                                           StructureType = 1000299004
	QUERY_POOL_VIDEO_ENCODE_FEEDBACK_CREATE_INFO_KHR                      StructureType = 1000299005
	PHYSICAL_DEVICE_VIDEO_ENCODE_QUALITY_LEVEL_INFO_KHR                   StructureType = 1000299006
	VIDEO_ENCODE_QUALITY_LEVEL_PROPERTIES_KHR                             StructureType = 1000299007
	VIDEO_ENCODE_QUALITY_LEVEL_INFO_KHR                                   StructureType = 1000299008
	VIDEO_ENCODE_SESSION_PARAMETERS_GET_INFO_KHR                          StructureType = 1000299009
	VIDEO_ENCODE_SESSION_PARAMETERS_FEEDBACK_INFO_KHR                     StructureType = 1000299010
	PHYSICAL_DEVICE_DIAGNOSTICS_CONFIG_FEATURES_NV                        StructureType = 1000300000
	DEVICE_DIAGNOSTICS_CONFIG_CREATE_INFO_NV                              StructureType = 1000300001
	CUDA_MODULE_CREATE_INFO_NV                                            StructureType = 1000307000
	CUDA_FUNCTION_CREATE_INFO_NV                                          StructureType = 1000307001
	CUDA_LAUNCH_INFO_NV                                                   StructureType = 1000307002
	PHYSICAL_DEVICE_CUDA_KERNEL_LAUNCH_FEATURES_NV                        StructureType = 1000307003
	PHYSICAL_DEVICE_CUDA_KERNEL_LAUNCH_PROPERTIES_NV                      StructureType = 1000307004
	PHYSICAL_DEVICE_TILE_SHADING_FEATURES_QCOM                            StructureType = 1000309000
	PHYSICAL_DEVICE_TILE_SHADING_PROPERTIES_QCOM                          StructureType = 1000309001
	RENDER_PASS_TILE_SHADING_CREATE_INFO_QCOM                             StructureType = 1000309002
	PER_TILE_BEGIN_INFO_QCOM                                              StructureType = 1000309003
	PER_TILE_END_INFO_QCOM                                                StructureType = 1000309004
	DISPATCH_TILE_INFO_QCOM                                               StructureType = 1000309005
	QUERY_LOW_LATENCY_SUPPORT_NV                                          StructureType = 1000310000
	EXPORT_METAL_OBJECT_CREATE_INFO_EXT                                   StructureType = 1000311000
	EXPORT_METAL_OBJECTS_INFO_EXT                                         StructureType = 1000311001
	EXPORT_METAL_DEVICE_INFO_EXT                                          StructureType = 1000311002
	EXPORT_METAL_COMMAND_QUEUE_INFO_EXT                                   StructureType = 1000311003
	EXPORT_METAL_BUFFER_INFO_EXT                                          StructureType = 1000311004
	IMPORT_METAL_BUFFER_INFO_EXT                                          StructureType = 1000311005
	EXPORT_METAL_TEXTURE_INFO_EXT                                         StructureType = 1000311006
	IMPORT_METAL_TEXTURE_INFO_EXT                                         StructureType = 1000311007
	EXPORT_METAL_IO_SURFACE_INFO_EXT                                      StructureType = 1000311008
	IMPORT_METAL_IO_SURFACE_INFO_EXT                                      StructureType = 1000311009
	EXPORT_METAL_SHARED_EVENT_INFO_EXT                                    StructureType = 1000311010
	IMPORT_METAL_SHARED_EVENT_INFO_EXT                                    StructureType = 1000311011
	PHYSICAL_DEVICE_DESCRIPTOR_BUFFER_PROPERTIES_EXT                      StructureType = 1000316000
	PHYSICAL_DEVICE_DESCRIPTOR_BUFFER_DENSITY_MAP_PROPERTIES_EXT          StructureType = 1000316001
	PHYSICAL_DEVICE_DESCRIPTOR_BUFFER_FEATURES_EXT                        StructureType = 1000316002
	DESCRIPTOR_ADDRESS_INFO_EXT                                           StructureType = 1000316003
	DESCRIPTOR_GET_INFO_EXT                                               StructureType = 1000316004
	BUFFER_CAPTURE_DESCRIPTOR_DATA_INFO_EXT                               StructureType = 1000316005
	IMAGE_CAPTURE_DESCRIPTOR_DATA_INFO_EXT                                StructureType = 1000316006
	IMAGE_VIEW_CAPTURE_DESCRIPTOR_DATA_INFO_EXT                           StructureType = 1000316007
	SAMPLER_CAPTURE_DESCRIPTOR_DATA_INFO_EXT                              StructureType = 1000316008
	OPAQUE_CAPTURE_DESCRIPTOR_DATA_CREATE_INFO_EXT                        StructureType = 1000316010
	DESCRIPTOR_BUFFER_BINDING_INFO_EXT                                    StructureType = 1000316011
	DESCRIPTOR_BUFFER_BINDING_PUSH_DESCRIPTOR_BUFFER_HANDLE_EXT           StructureType = 1000316012
	ACCELERATION_STRUCTURE_CAPTURE_DESCRIPTOR_DATA_INFO_EXT               StructureType = 1000316009
	PHYSICAL_DEVICE_GRAPHICS_PIPELINE_LIBRARY_FEATURES_EXT                StructureType = 1000320000
	PHYSICAL_DEVICE_GRAPHICS_PIPELINE_LIBRARY_PROPERTIES_EXT              StructureType = 1000320001
	GRAPHICS_PIPELINE_LIBRARY_CREATE_INFO_EXT                             StructureType = 1000320002
	PHYSICAL_DEVICE_SHADER_EARLY_AND_LATE_FRAGMENT_TESTS_FEATURES_AMD     StructureType = 1000321000
	PHYSICAL_DEVICE_FRAGMENT_SHADER_BARYCENTRIC_FEATURES_KHR              StructureType = 1000203000
	PHYSICAL_DEVICE_FRAGMENT_SHADER_BARYCENTRIC_PROPERTIES_KHR            StructureType = 1000322000
	PHYSICAL_DEVICE_SHADER_SUBGROUP_UNIFORM_CONTROL_FLOW_FEATURES_KHR     StructureType = 1000323000
	PHYSICAL_DEVICE_FRAGMENT_SHADING_RATE_ENUMS_PROPERTIES_NV             StructureType = 1000326000
	PHYSICAL_DEVICE_FRAGMENT_SHADING_RATE_ENUMS_FEATURES_NV               StructureType = 1000326001
	PIPELINE_FRAGMENT_SHADING_RATE_ENUM_STATE_CREATE_INFO_NV              StructureType = 1000326002
	ACCELERATION_STRUCTURE_GEOMETRY_MOTION_TRIANGLES_DATA_NV              StructureType = 1000327000
	PHYSICAL_DEVICE_RAY_TRACING_MOTION_BLUR_FEATURES_NV                   StructureType = 1000327001
	ACCELERATION_STRUCTURE_MOTION_INFO_NV                                 StructureType = 1000327002
	PHYSICAL_DEVICE_MESH_SHADER_FEATURES_EXT                              StructureType = 1000328000
	PHYSICAL_DEVICE_MESH_SHADER_PROPERTIES_EXT                            StructureType = 1000328001
	PHYSICAL_DEVICE_YCBCR_2_PLANE_444_FORMATS_FEATURES_EXT                StructureType = 1000330000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_2_FEATURES_EXT                   StructureType = 1000332000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_2_PROPERTIES_EXT                 StructureType = 1000332001
	COPY_COMMAND_TRANSFORM_INFO_QCOM                                      StructureType = 1000333000
	PHYSICAL_DEVICE_WORKGROUP_MEMORY_EXPLICIT_LAYOUT_FEATURES_KHR         StructureType = 1000336000
	PHYSICAL_DEVICE_IMAGE_COMPRESSION_CONTROL_FEATURES_EXT                StructureType = 1000338000
	IMAGE_COMPRESSION_CONTROL_EXT                                         StructureType = 1000338001
	IMAGE_COMPRESSION_PROPERTIES_EXT                                      StructureType = 1000338004
	PHYSICAL_DEVICE_ATTACHMENT_FEEDBACK_LOOP_LAYOUT_FEATURES_EXT          StructureType = 1000339000
	PHYSICAL_DEVICE_4444_FORMATS_FEATURES_EXT                             StructureType = 1000340000
	PHYSICAL_DEVICE_FAULT_FEATURES_EXT                                    StructureType = 1000341000
	DEVICE_FAULT_COUNTS_EXT                                               StructureType = 1000341001
	DEVICE_FAULT_INFO_EXT                                                 StructureType = 1000341002
	PHYSICAL_DEVICE_RGBA10X6_FORMATS_FEATURES_EXT                         StructureType = 1000344000
	DIRECTFB_SURFACE_CREATE_INFO_EXT                                      StructureType = 1000346000
	PHYSICAL_DEVICE_VERTEX_INPUT_DYNAMIC_STATE_FEATURES_EXT               StructureType = 1000352000
	VERTEX_INPUT_BINDING_DESCRIPTION_2_EXT                                StructureType = 1000352001
	VERTEX_INPUT_ATTRIBUTE_DESCRIPTION_2_EXT                              StructureType = 1000352002
	PHYSICAL_DEVICE_DRM_PROPERTIES_EXT                                    StructureType = 1000353000
	PHYSICAL_DEVICE_ADDRESS_BINDING_REPORT_FEATURES_EXT                   StructureType = 1000354000
	DEVICE_ADDRESS_BINDING_CALLBACK_DATA_EXT                              StructureType = 1000354001
	PHYSICAL_DEVICE_DEPTH_CLIP_CONTROL_FEATURES_EXT                       StructureType = 1000355000
	PIPELINE_VIEWPORT_DEPTH_CLIP_CONTROL_CREATE_INFO_EXT                  StructureType = 1000355001
	PHYSICAL_DEVICE_PRIMITIVE_TOPOLOGY_LIST_RESTART_FEATURES_EXT          StructureType = 1000356000
	IMPORT_MEMORY_ZIRCON_HANDLE_INFO_FUCHSIA                              StructureType = 1000364000
	MEMORY_ZIRCON_HANDLE_PROPERTIES_FUCHSIA                               StructureType = 1000364001
	MEMORY_GET_ZIRCON_HANDLE_INFO_FUCHSIA                                 StructureType = 1000364002
	IMPORT_SEMAPHORE_ZIRCON_HANDLE_INFO_FUCHSIA                           StructureType = 1000365000
	SEMAPHORE_GET_ZIRCON_HANDLE_INFO_FUCHSIA                              StructureType = 1000365001
	BUFFER_COLLECTION_CREATE_INFO_FUCHSIA                                 StructureType = 1000366000
	IMPORT_MEMORY_BUFFER_COLLECTION_FUCHSIA                               StructureType = 1000366001
	BUFFER_COLLECTION_IMAGE_CREATE_INFO_FUCHSIA                           StructureType = 1000366002
	BUFFER_COLLECTION_PROPERTIES_FUCHSIA                                  StructureType = 1000366003
	BUFFER_CONSTRAINTS_INFO_FUCHSIA                                       StructureType = 1000366004
	BUFFER_COLLECTION_BUFFER_CREATE_INFO_FUCHSIA                          StructureType = 1000366005
	IMAGE_CONSTRAINTS_INFO_FUCHSIA                                        StructureType = 1000366006
	IMAGE_FORMAT_CONSTRAINTS_INFO_FUCHSIA                                 StructureType = 1000366007
	SYSMEM_COLOR_SPACE_FUCHSIA                                            StructureType = 1000366008
	BUFFER_COLLECTION_CONSTRAINTS_INFO_FUCHSIA                            StructureType = 1000366009
	SUBPASS_SHADING_PIPELINE_CREATE_INFO_HUAWEI                           StructureType = 1000369000
	PHYSICAL_DEVICE_SUBPASS_SHADING_FEATURES_HUAWEI                       StructureType = 1000369001
	PHYSICAL_DEVICE_SUBPASS_SHADING_PROPERTIES_HUAWEI                     StructureType = 1000369002
	PHYSICAL_DEVICE_INVOCATION_MASK_FEATURES_HUAWEI                       StructureType = 1000370000
	MEMORY_GET_REMOTE_ADDRESS_INFO_NV                                     StructureType = 1000371000
	PHYSICAL_DEVICE_EXTERNAL_MEMORY_RDMA_FEATURES_NV                      StructureType = 1000371001
	PIPELINE_PROPERTIES_IDENTIFIER_EXT                                    StructureType = 1000372000
	PHYSICAL_DEVICE_PIPELINE_PROPERTIES_FEATURES_EXT                      StructureType = 1000372001
	PHYSICAL_DEVICE_FRAME_BOUNDARY_FEATURES_EXT                           StructureType = 1000375000
	FRAME_BOUNDARY_EXT                                                    StructureType = 1000375001
	PHYSICAL_DEVICE_MULTISAMPLED_RENDER_TO_SINGLE_SAMPLED_FEATURES_EXT    StructureType = 1000376000
	SUBPASS_RESOLVE_PERFORMANCE_QUERY_EXT                                 StructureType = 1000376001
	MULTISAMPLED_RENDER_TO_SINGLE_SAMPLED_INFO_EXT                        StructureType = 1000376002
	PHYSICAL_DEVICE_EXTENDED_DYNAMIC_STATE_2_FEATURES_EXT                 StructureType = 1000377000
	SCREEN_SURFACE_CREATE_INFO_QNX                                        StructureType = 1000378000
	PHYSICAL_DEVICE_COLOR_WRITE_ENABLE_FEATURES_EXT                       StructureType = 1000381000
	PIPELINE_COLOR_WRITE_CREATE_INFO_EXT                                  StructureType = 1000381001
	PHYSICAL_DEVICE_PRIMITIVES_GENERATED_QUERY_FEATURES_EXT               StructureType = 1000382000
	PHYSICAL_DEVICE_RAY_TRACING_MAINTENANCE_1_FEATURES_KHR                StructureType = 1000386000
	PHYSICAL_DEVICE_IMAGE_VIEW_MIN_LOD_FEATURES_EXT                       StructureType = 1000391000
	IMAGE_VIEW_MIN_LOD_CREATE_INFO_EXT                                    StructureType = 1000391001
	PHYSICAL_DEVICE_MULTI_DRAW_FEATURES_EXT                               StructureType = 1000392000
	PHYSICAL_DEVICE_MULTI_DRAW_PROPERTIES_EXT                             StructureType = 1000392001
	PHYSICAL_DEVICE_IMAGE_2D_VIEW_OF_3D_FEATURES_EXT                      StructureType = 1000393000
	PHYSICAL_DEVICE_SHADER_TILE_IMAGE_FEATURES_EXT                        StructureType = 1000395000
	PHYSICAL_DEVICE_SHADER_TILE_IMAGE_PROPERTIES_EXT                      StructureType = 1000395001
	MICROMAP_BUILD_INFO_EXT                                               StructureType = 1000396000
	MICROMAP_VERSION_INFO_EXT                                             StructureType = 1000396001
	COPY_MICROMAP_INFO_EXT                                                StructureType = 1000396002
	COPY_MICROMAP_TO_MEMORY_INFO_EXT                                      StructureType = 1000396003
	COPY_MEMORY_TO_MICROMAP_INFO_EXT                                      StructureType = 1000396004
	PHYSICAL_DEVICE_OPACITY_MICROMAP_FEATURES_EXT                         StructureType = 1000396005
	PHYSICAL_DEVICE_OPACITY_MICROMAP_PROPERTIES_EXT                       StructureType = 1000396006
	MICROMAP_CREATE_INFO_EXT                                              StructureType = 1000396007
	MICROMAP_BUILD_SIZES_INFO_EXT                                         StructureType = 1000396008
	ACCELERATION_STRUCTURE_TRIANGLES_OPACITY_MICROMAP_EXT                 StructureType = 1000396009
	PHYSICAL_DEVICE_DISPLACEMENT_MICROMAP_FEATURES_NV                     StructureType = 1000397000
	PHYSICAL_DEVICE_DISPLACEMENT_MICROMAP_PROPERTIES_NV                   StructureType = 1000397001
	ACCELERATION_STRUCTURE_TRIANGLES_DISPLACEMENT_MICROMAP_NV             StructureType = 1000397002
	PHYSICAL_DEVICE_CLUSTER_CULLING_SHADER_FEATURES_HUAWEI                StructureType = 1000404000
	PHYSICAL_DEVICE_CLUSTER_CULLING_SHADER_PROPERTIES_HUAWEI              StructureType = 1000404001
	PHYSICAL_DEVICE_CLUSTER_CULLING_SHADER_VRS_FEATURES_HUAWEI            StructureType = 1000404002
	PHYSICAL_DEVICE_BORDER_COLOR_SWIZZLE_FEATURES_EXT                     StructureType = 1000411000
	SAMPLER_BORDER_COLOR_COMPONENT_MAPPING_CREATE_INFO_EXT                StructureType = 1000411001
	PHYSICAL_DEVICE_PAGEABLE_DEVICE_LOCAL_MEMORY_FEATURES_EXT             StructureType = 1000412000
	PHYSICAL_DEVICE_SHADER_CORE_PROPERTIES_ARM                            StructureType = 1000415000
	DEVICE_QUEUE_SHADER_CORE_CONTROL_CREATE_INFO_ARM                      StructureType = 1000417000
	PHYSICAL_DEVICE_SCHEDULING_CONTROLS_FEATURES_ARM                      StructureType = 1000417001
	PHYSICAL_DEVICE_SCHEDULING_CONTROLS_PROPERTIES_ARM                    StructureType = 1000417002
	PHYSICAL_DEVICE_IMAGE_SLICED_VIEW_OF_3D_FEATURES_EXT                  StructureType = 1000418000
	IMAGE_VIEW_SLICED_CREATE_INFO_EXT                                     StructureType = 1000418001
	PHYSICAL_DEVICE_DESCRIPTOR_SET_HOST_MAPPING_FEATURES_VALVE            StructureType = 1000420000
	DESCRIPTOR_SET_BINDING_REFERENCE_VALVE                                StructureType = 1000420001
	DESCRIPTOR_SET_LAYOUT_HOST_MAPPING_INFO_VALVE                         StructureType = 1000420002
	PHYSICAL_DEVICE_NON_SEAMLESS_CUBE_MAP_FEATURES_EXT                    StructureType = 1000422000
	PHYSICAL_DEVICE_RENDER_PASS_STRIPED_FEATURES_ARM                      StructureType = 1000424000
	PHYSICAL_DEVICE_RENDER_PASS_STRIPED_PROPERTIES_ARM                    StructureType = 1000424001
	RENDER_PASS_STRIPE_BEGIN_INFO_ARM                                     StructureType = 1000424002
	RENDER_PASS_STRIPE_INFO_ARM                                           StructureType = 1000424003
	RENDER_PASS_STRIPE_SUBMIT_INFO_ARM                                    StructureType = 1000424004
	PHYSICAL_DEVICE_COPY_MEMORY_INDIRECT_FEATURES_NV                      StructureType = 1000426000
	PHYSICAL_DEVICE_COPY_MEMORY_INDIRECT_PROPERTIES_NV                    StructureType = 1000426001
	PHYSICAL_DEVICE_MEMORY_DECOMPRESSION_FEATURES_NV                      StructureType = 1000427000
	PHYSICAL_DEVICE_MEMORY_DECOMPRESSION_PROPERTIES_NV                    StructureType = 1000427001
	PHYSICAL_DEVICE_DEVICE_GENERATED_COMMANDS_COMPUTE_FEATURES_NV         StructureType = 1000428000
	COMPUTE_PIPELINE_INDIRECT_BUFFER_INFO_NV                              StructureType = 1000428001
	PIPELINE_INDIRECT_DEVICE_ADDRESS_INFO_NV                              StructureType = 1000428002
	PHYSICAL_DEVICE_RAY_TRACING_LINEAR_SWEPT_SPHERES_FEATURES_NV          StructureType = 1000429008
	ACCELERATION_STRUCTURE_GEOMETRY_LINEAR_SWEPT_SPHERES_DATA_NV          StructureType = 1000429009
	ACCELERATION_STRUCTURE_GEOMETRY_SPHERES_DATA_NV                       StructureType = 1000429010
	PHYSICAL_DEVICE_LINEAR_COLOR_ATTACHMENT_FEATURES_NV                   StructureType = 1000430000
	PHYSICAL_DEVICE_SHADER_MAXIMAL_RECONVERGENCE_FEATURES_KHR             StructureType = 1000434000
	PHYSICAL_DEVICE_IMAGE_COMPRESSION_CONTROL_SWAPCHAIN_FEATURES_EXT      StructureType = 1000437000
	PHYSICAL_DEVICE_IMAGE_PROCESSING_FEATURES_QCOM                        StructureType = 1000440000
	PHYSICAL_DEVICE_IMAGE_PROCESSING_PROPERTIES_QCOM                      StructureType = 1000440001
	IMAGE_VIEW_SAMPLE_WEIGHT_CREATE_INFO_QCOM                             StructureType = 1000440002
	PHYSICAL_DEVICE_NESTED_COMMAND_BUFFER_FEATURES_EXT                    StructureType = 1000451000
	PHYSICAL_DEVICE_NESTED_COMMAND_BUFFER_PROPERTIES_EXT                  StructureType = 1000451001
	EXTERNAL_MEMORY_ACQUIRE_UNMODIFIED_EXT                                StructureType = 1000453000
	PHYSICAL_DEVICE_EXTENDED_DYNAMIC_STATE_3_FEATURES_EXT                 StructureType = 1000455000
	PHYSICAL_DEVICE_EXTENDED_DYNAMIC_STATE_3_PROPERTIES_EXT               StructureType = 1000455001
	PHYSICAL_DEVICE_SUBPASS_MERGE_FEEDBACK_FEATURES_EXT                   StructureType = 1000458000
	RENDER_PASS_CREATION_CONTROL_EXT                                      StructureType = 1000458001
	RENDER_PASS_CREATION_FEEDBACK_CREATE_INFO_EXT                         StructureType = 1000458002
	RENDER_PASS_SUBPASS_FEEDBACK_CREATE_INFO_EXT                          StructureType = 1000458003
	DIRECT_DRIVER_LOADING_INFO_LUNARG                                     StructureType = 1000459000
	DIRECT_DRIVER_LOADING_LIST_LUNARG                                     StructureType = 1000459001
	TENSOR_CREATE_INFO_ARM                                                StructureType = 1000460000
	TENSOR_VIEW_CREATE_INFO_ARM                                           StructureType = 1000460001
	BIND_TENSOR_MEMORY_INFO_ARM                                           StructureType = 1000460002
	WRITE_DESCRIPTOR_SET_TENSOR_ARM                                       StructureType = 1000460003
	PHYSICAL_DEVICE_TENSOR_PROPERTIES_ARM                                 StructureType = 1000460004
	TENSOR_FORMAT_PROPERTIES_ARM                                          StructureType = 1000460005
	TENSOR_DESCRIPTION_ARM                                                StructureType = 1000460006
	TENSOR_MEMORY_REQUIREMENTS_INFO_ARM                                   StructureType = 1000460007
	TENSOR_MEMORY_BARRIER_ARM                                             StructureType = 1000460008
	PHYSICAL_DEVICE_TENSOR_FEATURES_ARM                                   StructureType = 1000460009
	DEVICE_TENSOR_MEMORY_REQUIREMENTS_ARM                                 StructureType = 1000460010
	COPY_TENSOR_INFO_ARM                                                  StructureType = 1000460011
	TENSOR_COPY_ARM                                                       StructureType = 1000460012
	TENSOR_DEPENDENCY_INFO_ARM                                            StructureType = 1000460013
	MEMORY_DEDICATED_ALLOCATE_INFO_TENSOR_ARM                             StructureType = 1000460014
	PHYSICAL_DEVICE_EXTERNAL_TENSOR_INFO_ARM                              StructureType = 1000460015
	EXTERNAL_TENSOR_PROPERTIES_ARM                                        StructureType = 1000460016
	EXTERNAL_MEMORY_TENSOR_CREATE_INFO_ARM                                StructureType = 1000460017
	PHYSICAL_DEVICE_DESCRIPTOR_BUFFER_TENSOR_FEATURES_ARM                 StructureType = 1000460018
	PHYSICAL_DEVICE_DESCRIPTOR_BUFFER_TENSOR_PROPERTIES_ARM               StructureType = 1000460019
	DESCRIPTOR_GET_TENSOR_INFO_ARM                                        StructureType = 1000460020
	TENSOR_CAPTURE_DESCRIPTOR_DATA_INFO_ARM                               StructureType = 1000460021
	TENSOR_VIEW_CAPTURE_DESCRIPTOR_DATA_INFO_ARM                          StructureType = 1000460022
	FRAME_BOUNDARY_TENSORS_ARM                                            StructureType = 1000460023
	PHYSICAL_DEVICE_SHADER_MODULE_IDENTIFIER_FEATURES_EXT                 StructureType = 1000462000
	PHYSICAL_DEVICE_SHADER_MODULE_IDENTIFIER_PROPERTIES_EXT               StructureType = 1000462001
	PIPELINE_SHADER_STAGE_MODULE_IDENTIFIER_CREATE_INFO_EXT               StructureType = 1000462002
	SHADER_MODULE_IDENTIFIER_EXT                                          StructureType = 1000462003
	PHYSICAL_DEVICE_RASTERIZATION_ORDER_ATTACHMENT_ACCESS_FEATURES_EXT    StructureType = 1000342000
	PHYSICAL_DEVICE_OPTICAL_FLOW_FEATURES_NV                              StructureType = 1000464000
	PHYSICAL_DEVICE_OPTICAL_FLOW_PROPERTIES_NV                            StructureType = 1000464001
	OPTICAL_FLOW_IMAGE_FORMAT_INFO_NV                                     StructureType = 1000464002
	OPTICAL_FLOW_IMAGE_FORMAT_PROPERTIES_NV                               StructureType = 1000464003
	OPTICAL_FLOW_SESSION_CREATE_INFO_NV                                   StructureType = 1000464004
	OPTICAL_FLOW_EXECUTE_INFO_NV                                          StructureType = 1000464005
	OPTICAL_FLOW_SESSION_CREATE_PRIVATE_DATA_INFO_NV                      StructureType = 1000464010
	PHYSICAL_DEVICE_LEGACY_DITHERING_FEATURES_EXT                         StructureType = 1000465000
	PHYSICAL_DEVICE_EXTERNAL_FORMAT_RESOLVE_FEATURES_ANDROID              StructureType = 1000468000
	PHYSICAL_DEVICE_EXTERNAL_FORMAT_RESOLVE_PROPERTIES_ANDROID            StructureType = 1000468001
	ANDROID_HARDWARE_BUFFER_FORMAT_RESOLVE_PROPERTIES_ANDROID             StructureType = 1000468002
	PHYSICAL_DEVICE_ANTI_LAG_FEATURES_AMD                                 StructureType = 1000476000
	ANTI_LAG_DATA_AMD                                                     StructureType = 1000476001
	ANTI_LAG_PRESENTATION_INFO_AMD                                        StructureType = 1000476002
	SURFACE_CAPABILITIES_PRESENT_ID_2_KHR                                 StructureType = 1000479000
	PRESENT_ID_2_KHR                                                      StructureType = 1000479001
	PHYSICAL_DEVICE_PRESENT_ID_2_FEATURES_KHR                             StructureType = 1000479002
	SURFACE_CAPABILITIES_PRESENT_WAIT_2_KHR                               StructureType = 1000480000
	PHYSICAL_DEVICE_PRESENT_WAIT_2_FEATURES_KHR                           StructureType = 1000480001
	PRESENT_WAIT_2_INFO_KHR                                               StructureType = 1000480002
	PHYSICAL_DEVICE_RAY_TRACING_POSITION_FETCH_FEATURES_KHR               StructureType = 1000481000
	PHYSICAL_DEVICE_SHADER_OBJECT_FEATURES_EXT                            StructureType = 1000482000
	PHYSICAL_DEVICE_SHADER_OBJECT_PROPERTIES_EXT                          StructureType = 1000482001
	SHADER_CREATE_INFO_EXT                                                StructureType = 1000482002
	PHYSICAL_DEVICE_PIPELINE_BINARY_FEATURES_KHR                          StructureType = 1000483000
	PIPELINE_BINARY_CREATE_INFO_KHR                                       StructureType = 1000483001
	PIPELINE_BINARY_INFO_KHR                                              StructureType = 1000483002
	PIPELINE_BINARY_KEY_KHR                                               StructureType = 1000483003
	PHYSICAL_DEVICE_PIPELINE_BINARY_PROPERTIES_KHR                        StructureType = 1000483004
	RELEASE_CAPTURED_PIPELINE_DATA_INFO_KHR                               StructureType = 1000483005
	PIPELINE_BINARY_DATA_INFO_KHR                                         StructureType = 1000483006
	PIPELINE_CREATE_INFO_KHR                                              StructureType = 1000483007
	DEVICE_PIPELINE_BINARY_INTERNAL_CACHE_CONTROL_KHR                     StructureType = 1000483008
	PIPELINE_BINARY_HANDLES_INFO_KHR                                      StructureType = 1000483009
	PHYSICAL_DEVICE_TILE_PROPERTIES_FEATURES_QCOM                         StructureType = 1000484000
	TILE_PROPERTIES_QCOM                                                  StructureType = 1000484001
	PHYSICAL_DEVICE_AMIGO_PROFILING_FEATURES_SEC                          StructureType = 1000485000
	AMIGO_PROFILING_SUBMIT_INFO_SEC                                       StructureType = 1000485001
	SURFACE_PRESENT_MODE_KHR                                              StructureType = 1000274000
	SURFACE_PRESENT_SCALING_CAPABILITIES_KHR                              StructureType = 1000274001
	SURFACE_PRESENT_MODE_COMPATIBILITY_KHR                                StructureType = 1000274002
	PHYSICAL_DEVICE_SWAPCHAIN_MAINTENANCE_1_FEATURES_KHR                  StructureType = 1000275000
	SWAPCHAIN_PRESENT_FENCE_INFO_KHR                                      StructureType = 1000275001
	SWAPCHAIN_PRESENT_MODES_CREATE_INFO_KHR                               StructureType = 1000275002
	SWAPCHAIN_PRESENT_MODE_INFO_KHR                                       StructureType = 1000275003
	SWAPCHAIN_PRESENT_SCALING_CREATE_INFO_KHR                             StructureType = 1000275004
	RELEASE_SWAPCHAIN_IMAGES_INFO_KHR                                     StructureType = 1000275005
	PHYSICAL_DEVICE_MULTIVIEW_PER_VIEW_VIEWPORTS_FEATURES_QCOM            StructureType = 1000488000
	PHYSICAL_DEVICE_RAY_TRACING_INVOCATION_REORDER_FEATURES_NV            StructureType = 1000490000
	PHYSICAL_DEVICE_RAY_TRACING_INVOCATION_REORDER_PROPERTIES_NV          StructureType = 1000490001
	PHYSICAL_DEVICE_COOPERATIVE_VECTOR_FEATURES_NV                        StructureType = 1000491000
	PHYSICAL_DEVICE_COOPERATIVE_VECTOR_PROPERTIES_NV                      StructureType = 1000491001
	COOPERATIVE_VECTOR_PROPERTIES_NV                                      StructureType = 1000491002
	CONVERT_COOPERATIVE_VECTOR_MATRIX_INFO_NV                             StructureType = 1000491004
	PHYSICAL_DEVICE_EXTENDED_SPARSE_ADDRESS_SPACE_FEATURES_NV             StructureType = 1000492000
	PHYSICAL_DEVICE_EXTENDED_SPARSE_ADDRESS_SPACE_PROPERTIES_NV           StructureType = 1000492001
	PHYSICAL_DEVICE_MUTABLE_DESCRIPTOR_TYPE_FEATURES_EXT                  StructureType = 1000351000
	MUTABLE_DESCRIPTOR_TYPE_CREATE_INFO_EXT                               StructureType = 1000351002
	PHYSICAL_DEVICE_LEGACY_VERTEX_ATTRIBUTES_FEATURES_EXT                 StructureType = 1000495000
	PHYSICAL_DEVICE_LEGACY_VERTEX_ATTRIBUTES_PROPERTIES_EXT               StructureType = 1000495001
	LAYER_SETTINGS_CREATE_INFO_EXT                                        StructureType = 1000496000
	PHYSICAL_DEVICE_SHADER_CORE_BUILTINS_FEATURES_ARM                     StructureType = 1000497000
	PHYSICAL_DEVICE_SHADER_CORE_BUILTINS_PROPERTIES_ARM                   StructureType = 1000497001
	PHYSICAL_DEVICE_PIPELINE_LIBRARY_GROUP_HANDLES_FEATURES_EXT           StructureType = 1000498000
	PHYSICAL_DEVICE_DYNAMIC_RENDERING_UNUSED_ATTACHMENTS_FEATURES_EXT     StructureType = 1000499000
	LATENCY_SLEEP_MODE_INFO_NV                                            StructureType = 1000505000
	LATENCY_SLEEP_INFO_NV                                                 StructureType = 1000505001
	SET_LATENCY_MARKER_INFO_NV                                            StructureType = 1000505002
	GET_LATENCY_MARKER_INFO_NV                                            StructureType = 1000505003
	LATENCY_TIMINGS_FRAME_REPORT_NV                                       StructureType = 1000505004
	LATENCY_SUBMISSION_PRESENT_ID_NV                                      StructureType = 1000505005
	OUT_OF_BAND_QUEUE_TYPE_INFO_NV                                        StructureType = 1000505006
	SWAPCHAIN_LATENCY_CREATE_INFO_NV                                      StructureType = 1000505007
	LATENCY_SURFACE_CAPABILITIES_NV                                       StructureType = 1000505008
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_FEATURES_KHR                       StructureType = 1000506000
	COOPERATIVE_MATRIX_PROPERTIES_KHR                                     StructureType = 1000506001
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_PROPERTIES_KHR                     StructureType = 1000506002
	DATA_GRAPH_PIPELINE_CREATE_INFO_ARM                                   StructureType = 1000507000
	DATA_GRAPH_PIPELINE_SESSION_CREATE_INFO_ARM                           StructureType = 1000507001
	DATA_GRAPH_PIPELINE_RESOURCE_INFO_ARM                                 StructureType = 1000507002
	DATA_GRAPH_PIPELINE_CONSTANT_ARM                                      StructureType = 1000507003
	DATA_GRAPH_PIPELINE_SESSION_MEMORY_REQUIREMENTS_INFO_ARM              StructureType = 1000507004
	BIND_DATA_GRAPH_PIPELINE_SESSION_MEMORY_INFO_ARM                      StructureType = 1000507005
	PHYSICAL_DEVICE_DATA_GRAPH_FEATURES_ARM                               StructureType = 1000507006
	DATA_GRAPH_PIPELINE_SHADER_MODULE_CREATE_INFO_ARM                     StructureType = 1000507007
	DATA_GRAPH_PIPELINE_PROPERTY_QUERY_RESULT_ARM                         StructureType = 1000507008
	DATA_GRAPH_PIPELINE_INFO_ARM                                          StructureType = 1000507009
	DATA_GRAPH_PIPELINE_COMPILER_CONTROL_CREATE_INFO_ARM                  StructureType = 1000507010
	DATA_GRAPH_PIPELINE_SESSION_BIND_POINT_REQUIREMENTS_INFO_ARM          StructureType = 1000507011
	DATA_GRAPH_PIPELINE_SESSION_BIND_POINT_REQUIREMENT_ARM                StructureType = 1000507012
	DATA_GRAPH_PIPELINE_IDENTIFIER_CREATE_INFO_ARM                        StructureType = 1000507013
	DATA_GRAPH_PIPELINE_DISPATCH_INFO_ARM                                 StructureType = 1000507014
	DATA_GRAPH_PROCESSING_ENGINE_CREATE_INFO_ARM                          StructureType = 1000507016
	QUEUE_FAMILY_DATA_GRAPH_PROCESSING_ENGINE_PROPERTIES_ARM              StructureType = 1000507017
	QUEUE_FAMILY_DATA_GRAPH_PROPERTIES_ARM                                StructureType = 1000507018
	PHYSICAL_DEVICE_QUEUE_FAMILY_DATA_GRAPH_PROCESSING_ENGINE_INFO_ARM    StructureType = 1000507019
	DATA_GRAPH_PIPELINE_CONSTANT_TENSOR_SEMI_STRUCTURED_SPARSITY_INFO_ARM StructureType = 1000507015
	PHYSICAL_DEVICE_MULTIVIEW_PER_VIEW_RENDER_AREAS_FEATURES_QCOM         StructureType = 1000510000
	MULTIVIEW_PER_VIEW_RENDER_AREAS_RENDER_PASS_BEGIN_INFO_QCOM           StructureType = 1000510001
	PHYSICAL_DEVICE_COMPUTE_SHADER_DERIVATIVES_FEATURES_KHR               StructureType = 1000201000
	PHYSICAL_DEVICE_COMPUTE_SHADER_DERIVATIVES_PROPERTIES_KHR             StructureType = 1000511000
	VIDEO_DECODE_AV1_CAPABILITIES_KHR                                     StructureType = 1000512000
	VIDEO_DECODE_AV1_PICTURE_INFO_KHR                                     StructureType = 1000512001
	VIDEO_DECODE_AV1_PROFILE_INFO_KHR                                     StructureType = 1000512003
	VIDEO_DECODE_AV1_SESSION_PARAMETERS_CREATE_INFO_KHR                   StructureType = 1000512004
	VIDEO_DECODE_AV1_DPB_SLOT_INFO_KHR                                    StructureType = 1000512005
	VIDEO_ENCODE_AV1_CAPABILITIES_KHR                                     StructureType = 1000513000
	VIDEO_ENCODE_AV1_SESSION_PARAMETERS_CREATE_INFO_KHR                   StructureType = 1000513001
	VIDEO_ENCODE_AV1_PICTURE_INFO_KHR                                     StructureType = 1000513002
	VIDEO_ENCODE_AV1_DPB_SLOT_INFO_KHR                                    StructureType = 1000513003
	PHYSICAL_DEVICE_VIDEO_ENCODE_AV1_FEATURES_KHR                         StructureType = 1000513004
	VIDEO_ENCODE_AV1_PROFILE_INFO_KHR                                     StructureType = 1000513005
	VIDEO_ENCODE_AV1_RATE_CONTROL_INFO_KHR                                StructureType = 1000513006
	VIDEO_ENCODE_AV1_RATE_CONTROL_LAYER_INFO_KHR                          StructureType = 1000513007
	VIDEO_ENCODE_AV1_QUALITY_LEVEL_PROPERTIES_KHR                         StructureType = 1000513008
	VIDEO_ENCODE_AV1_SESSION_CREATE_INFO_KHR                              StructureType = 1000513009
	VIDEO_ENCODE_AV1_GOP_REMAINING_FRAME_INFO_KHR                         StructureType = 1000513010
	PHYSICAL_DEVICE_VIDEO_DECODE_VP9_FEATURES_KHR                         StructureType = 1000514000
	VIDEO_DECODE_VP9_CAPABILITIES_KHR                                     StructureType = 1000514001
	VIDEO_DECODE_VP9_PICTURE_INFO_KHR                                     StructureType = 1000514002
	VIDEO_DECODE_VP9_PROFILE_INFO_KHR                                     StructureType = 1000514003
	PHYSICAL_DEVICE_VIDEO_MAINTENANCE_1_FEATURES_KHR                      StructureType = 1000515000
	VIDEO_INLINE_QUERY_INFO_KHR                                           StructureType = 1000515001
	PHYSICAL_DEVICE_PER_STAGE_DESCRIPTOR_SET_FEATURES_NV                  StructureType = 1000516000
	PHYSICAL_DEVICE_IMAGE_PROCESSING_2_FEATURES_QCOM                      StructureType = 1000518000
	PHYSICAL_DEVICE_IMAGE_PROCESSING_2_PROPERTIES_QCOM                    StructureType = 1000518001
	SAMPLER_BLOCK_MATCH_WINDOW_CREATE_INFO_QCOM                           StructureType = 1000518002
	SAMPLER_CUBIC_WEIGHTS_CREATE_INFO_QCOM                                StructureType = 1000519000
	PHYSICAL_DEVICE_CUBIC_WEIGHTS_FEATURES_QCOM                           StructureType = 1000519001
	BLIT_IMAGE_CUBIC_WEIGHTS_INFO_QCOM                                    StructureType = 1000519002
	PHYSICAL_DEVICE_YCBCR_DEGAMMA_FEATURES_QCOM                           StructureType = 1000520000
	SAMPLER_YCBCR_CONVERSION_YCBCR_DEGAMMA_CREATE_INFO_QCOM               StructureType = 1000520001
	PHYSICAL_DEVICE_CUBIC_CLAMP_FEATURES_QCOM                             StructureType = 1000521000
	PHYSICAL_DEVICE_ATTACHMENT_FEEDBACK_LOOP_DYNAMIC_STATE_FEATURES_EXT   StructureType = 1000524000
	PHYSICAL_DEVICE_UNIFIED_IMAGE_LAYOUTS_FEATURES_KHR                    StructureType = 1000527000
	ATTACHMENT_FEEDBACK_LOOP_INFO_EXT                                     StructureType = 1000527001
	SCREEN_BUFFER_PROPERTIES_QNX                                          StructureType = 1000529000
	SCREEN_BUFFER_FORMAT_PROPERTIES_QNX                                   StructureType = 1000529001
	IMPORT_SCREEN_BUFFER_INFO_QNX                                         StructureType = 1000529002
	EXTERNAL_FORMAT_QNX                                                   StructureType = 1000529003
	PHYSICAL_DEVICE_EXTERNAL_MEMORY_SCREEN_BUFFER_FEATURES_QNX            StructureType = 1000529004
	PHYSICAL_DEVICE_LAYERED_DRIVER_PROPERTIES_MSFT                        StructureType = 1000530000
	CALIBRATED_TIMESTAMP_INFO_KHR                                         StructureType = 1000184000
	SET_DESCRIPTOR_BUFFER_OFFSETS_INFO_EXT                                StructureType = 1000545007
	BIND_DESCRIPTOR_BUFFER_EMBEDDED_SAMPLERS_INFO_EXT                     StructureType = 1000545008
	PHYSICAL_DEVICE_DESCRIPTOR_POOL_OVERALLOCATION_FEATURES_NV            StructureType = 1000546000
	PHYSICAL_DEVICE_TILE_MEMORY_HEAP_FEATURES_QCOM                        StructureType = 1000547000
	PHYSICAL_DEVICE_TILE_MEMORY_HEAP_PROPERTIES_QCOM                      StructureType = 1000547001
	TILE_MEMORY_REQUIREMENTS_QCOM                                         StructureType = 1000547002
	TILE_MEMORY_BIND_INFO_QCOM                                            StructureType = 1000547003
	TILE_MEMORY_SIZE_INFO_QCOM                                            StructureType = 1000547004
	DISPLAY_SURFACE_STEREO_CREATE_INFO_NV                                 StructureType = 1000551000
	DISPLAY_MODE_STEREO_PROPERTIES_NV                                     StructureType = 1000551001
	VIDEO_ENCODE_INTRA_REFRESH_CAPABILITIES_KHR                           StructureType = 1000552000
	VIDEO_ENCODE_SESSION_INTRA_REFRESH_CREATE_INFO_KHR                    StructureType = 1000552001
	VIDEO_ENCODE_INTRA_REFRESH_INFO_KHR                                   StructureType = 1000552002
	VIDEO_REFERENCE_INTRA_REFRESH_INFO_KHR                                StructureType = 1000552003
	PHYSICAL_DEVICE_VIDEO_ENCODE_INTRA_REFRESH_FEATURES_KHR               StructureType = 1000552004
	VIDEO_ENCODE_QUANTIZATION_MAP_CAPABILITIES_KHR                        StructureType = 1000553000
	VIDEO_FORMAT_QUANTIZATION_MAP_PROPERTIES_KHR                          StructureType = 1000553001
	VIDEO_ENCODE_QUANTIZATION_MAP_INFO_KHR                                StructureType = 1000553002
	VIDEO_ENCODE_QUANTIZATION_MAP_SESSION_PARAMETERS_CREATE_INFO_KHR      StructureType = 1000553005
	PHYSICAL_DEVICE_VIDEO_ENCODE_QUANTIZATION_MAP_FEATURES_KHR            StructureType = 1000553009
	VIDEO_ENCODE_H264_QUANTIZATION_MAP_CAPABILITIES_KHR                   StructureType = 1000553003
	VIDEO_ENCODE_H265_QUANTIZATION_MAP_CAPABILITIES_KHR                   StructureType = 1000553004
	VIDEO_FORMAT_H265_QUANTIZATION_MAP_PROPERTIES_KHR                     StructureType = 1000553006
	VIDEO_ENCODE_AV1_QUANTIZATION_MAP_CAPABILITIES_KHR                    StructureType = 1000553007
	VIDEO_FORMAT_AV1_QUANTIZATION_MAP_PROPERTIES_KHR                      StructureType = 1000553008
	PHYSICAL_DEVICE_RAW_ACCESS_CHAINS_FEATURES_NV                         StructureType = 1000555000
	EXTERNAL_COMPUTE_QUEUE_DEVICE_CREATE_INFO_NV                          StructureType = 1000556000
	EXTERNAL_COMPUTE_QUEUE_CREATE_INFO_NV                                 StructureType = 1000556001
	EXTERNAL_COMPUTE_QUEUE_DATA_PARAMS_NV                                 StructureType = 1000556002
	PHYSICAL_DEVICE_EXTERNAL_COMPUTE_QUEUE_PROPERTIES_NV                  StructureType = 1000556003
	PHYSICAL_DEVICE_SHADER_RELAXED_EXTENDED_INSTRUCTION_FEATURES_KHR      StructureType = 1000558000
	PHYSICAL_DEVICE_COMMAND_BUFFER_INHERITANCE_FEATURES_NV                StructureType = 1000559000
	PHYSICAL_DEVICE_MAINTENANCE_7_FEATURES_KHR                            StructureType = 1000562000
	PHYSICAL_DEVICE_MAINTENANCE_7_PROPERTIES_KHR                          StructureType = 1000562001
	PHYSICAL_DEVICE_LAYERED_API_PROPERTIES_LIST_KHR                       StructureType = 1000562002
	PHYSICAL_DEVICE_LAYERED_API_PROPERTIES_KHR                            StructureType = 1000562003
	PHYSICAL_DEVICE_LAYERED_API_VULKAN_PROPERTIES_KHR                     StructureType = 1000562004
	PHYSICAL_DEVICE_SHADER_ATOMIC_FLOAT16_VECTOR_FEATURES_NV              StructureType = 1000563000
	PHYSICAL_DEVICE_SHADER_REPLICATED_COMPOSITES_FEATURES_EXT             StructureType = 1000564000
	PHYSICAL_DEVICE_SHADER_FLOAT8_FEATURES_EXT                            StructureType = 1000567000
	PHYSICAL_DEVICE_RAY_TRACING_VALIDATION_FEATURES_NV                    StructureType = 1000568000
	PHYSICAL_DEVICE_CLUSTER_ACCELERATION_STRUCTURE_FEATURES_NV            StructureType = 1000569000
	PHYSICAL_DEVICE_CLUSTER_ACCELERATION_STRUCTURE_PROPERTIES_NV          StructureType = 1000569001
	CLUSTER_ACCELERATION_STRUCTURE_CLUSTERS_BOTTOM_LEVEL_INPUT_NV         StructureType = 1000569002
	CLUSTER_ACCELERATION_STRUCTURE_TRIANGLE_CLUSTER_INPUT_NV              StructureType = 1000569003
	CLUSTER_ACCELERATION_STRUCTURE_MOVE_OBJECTS_INPUT_NV                  StructureType = 1000569004
	CLUSTER_ACCELERATION_STRUCTURE_INPUT_INFO_NV                          StructureType = 1000569005
	CLUSTER_ACCELERATION_STRUCTURE_COMMANDS_INFO_NV                       StructureType = 1000569006
	RAY_TRACING_PIPELINE_CLUSTER_ACCELERATION_STRUCTURE_CREATE_INFO_NV    StructureType = 1000569007
	PHYSICAL_DEVICE_PARTITIONED_ACCELERATION_STRUCTURE_FEATURES_NV        StructureType = 1000570000
	PHYSICAL_DEVICE_PARTITIONED_ACCELERATION_STRUCTURE_PROPERTIES_NV      StructureType = 1000570001
	WRITE_DESCRIPTOR_SET_PARTITIONED_ACCELERATION_STRUCTURE_NV            StructureType = 1000570002
	PARTITIONED_ACCELERATION_STRUCTURE_INSTANCES_INPUT_NV                 StructureType = 1000570003
	BUILD_PARTITIONED_ACCELERATION_STRUCTURE_INFO_NV                      StructureType = 1000570004
	PARTITIONED_ACCELERATION_STRUCTURE_FLAGS_NV                           StructureType = 1000570005
	PHYSICAL_DEVICE_DEVICE_GENERATED_COMMANDS_FEATURES_EXT                StructureType = 1000572000
	PHYSICAL_DEVICE_DEVICE_GENERATED_COMMANDS_PROPERTIES_EXT              StructureType = 1000572001
	GENERATED_COMMANDS_MEMORY_REQUIREMENTS_INFO_EXT                       StructureType = 1000572002
	INDIRECT_EXECUTION_SET_CREATE_INFO_EXT                                StructureType = 1000572003
	GENERATED_COMMANDS_INFO_EXT                                           StructureType = 1000572004
	INDIRECT_COMMANDS_LAYOUT_CREATE_INFO_EXT                              StructureType = 1000572006
	INDIRECT_COMMANDS_LAYOUT_TOKEN_EXT                                    StructureType = 1000572007
	WRITE_INDIRECT_EXECUTION_SET_PIPELINE_EXT                             StructureType = 1000572008
	WRITE_INDIRECT_EXECUTION_SET_SHADER_EXT                               StructureType = 1000572009
	INDIRECT_EXECUTION_SET_PIPELINE_INFO_EXT                              StructureType = 1000572010
	INDIRECT_EXECUTION_SET_SHADER_INFO_EXT                                StructureType = 1000572011
	INDIRECT_EXECUTION_SET_SHADER_LAYOUT_INFO_EXT                         StructureType = 1000572012
	GENERATED_COMMANDS_PIPELINE_INFO_EXT                                  StructureType = 1000572013
	GENERATED_COMMANDS_SHADER_INFO_EXT                                    StructureType = 1000572014
	PHYSICAL_DEVICE_MAINTENANCE_8_FEATURES_KHR                            StructureType = 1000574000
	MEMORY_BARRIER_ACCESS_FLAGS_3_KHR                                     StructureType = 1000574002
	PHYSICAL_DEVICE_IMAGE_ALIGNMENT_CONTROL_FEATURES_MESA                 StructureType = 1000575000
	PHYSICAL_DEVICE_IMAGE_ALIGNMENT_CONTROL_PROPERTIES_MESA               StructureType = 1000575001
	IMAGE_ALIGNMENT_CONTROL_CREATE_INFO_MESA                              StructureType = 1000575002
	PHYSICAL_DEVICE_DEPTH_CLAMP_CONTROL_FEATURES_EXT                      StructureType = 1000582000
	PIPELINE_VIEWPORT_DEPTH_CLAMP_CONTROL_CREATE_INFO_EXT                 StructureType = 1000582001
	PHYSICAL_DEVICE_MAINTENANCE_9_FEATURES_KHR                            StructureType = 1000584000
	PHYSICAL_DEVICE_MAINTENANCE_9_PROPERTIES_KHR                          StructureType = 1000584001
	QUEUE_FAMILY_OWNERSHIP_TRANSFER_PROPERTIES_KHR                        StructureType = 1000584002
	PHYSICAL_DEVICE_VIDEO_MAINTENANCE_2_FEATURES_KHR                      StructureType = 1000586000
	VIDEO_DECODE_H264_INLINE_SESSION_PARAMETERS_INFO_KHR                  StructureType = 1000586001
	VIDEO_DECODE_H265_INLINE_SESSION_PARAMETERS_INFO_KHR                  StructureType = 1000586002
	VIDEO_DECODE_AV1_INLINE_SESSION_PARAMETERS_INFO_KHR                   StructureType = 1000586003
	OH_SURFACE_CREATE_INFO_OHOS                                           StructureType = 1000587000
	PHYSICAL_DEVICE_HDR_VIVID_FEATURES_HUAWEI                             StructureType = 1000590000
	HDR_VIVID_DYNAMIC_METADATA_HUAWEI                                     StructureType = 1000590001
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_2_FEATURES_NV                      StructureType = 1000593000
	COOPERATIVE_MATRIX_FLEXIBLE_DIMENSIONS_PROPERTIES_NV                  StructureType = 1000593001
	PHYSICAL_DEVICE_COOPERATIVE_MATRIX_2_PROPERTIES_NV                    StructureType = 1000593002
	PHYSICAL_DEVICE_PIPELINE_OPACITY_MICROMAP_FEATURES_ARM                StructureType = 1000596000
	IMPORT_MEMORY_METAL_HANDLE_INFO_EXT                                   StructureType = 1000602000
	MEMORY_METAL_HANDLE_PROPERTIES_EXT                                    StructureType = 1000602001
	MEMORY_GET_METAL_HANDLE_INFO_EXT                                      StructureType = 1000602002
	PHYSICAL_DEVICE_DEPTH_CLAMP_ZERO_ONE_FEATURES_KHR                     StructureType = 1000421000
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_ROBUSTNESS_FEATURES_EXT              StructureType = 1000608000
	PHYSICAL_DEVICE_FORMAT_PACK_FEATURES_ARM                              StructureType = 1000609000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_LAYERED_FEATURES_VALVE           StructureType = 1000611000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_LAYERED_PROPERTIES_VALVE         StructureType = 1000611001
	PIPELINE_FRAGMENT_DENSITY_MAP_LAYERED_CREATE_INFO_VALVE               StructureType = 1000611002
	PHYSICAL_DEVICE_ROBUSTNESS_2_FEATURES_KHR                             StructureType = 1000286000
	PHYSICAL_DEVICE_ROBUSTNESS_2_PROPERTIES_KHR                           StructureType = 1000286001
	SET_PRESENT_CONFIG_NV                                                 StructureType = 1000613000
	PHYSICAL_DEVICE_PRESENT_METERING_FEATURES_NV                          StructureType = 1000613001
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_OFFSET_FEATURES_EXT              StructureType = 1000425000
	PHYSICAL_DEVICE_FRAGMENT_DENSITY_MAP_OFFSET_PROPERTIES_EXT            StructureType = 1000425001
	RENDER_PASS_FRAGMENT_DENSITY_MAP_OFFSET_END_INFO_EXT                  StructureType = 1000425002
	RENDERING_END_INFO_EXT                                                StructureType = 1000619003
	PHYSICAL_DEVICE_ZERO_INITIALIZE_DEVICE_MEMORY_FEATURES_EXT            StructureType = 1000620000
	PHYSICAL_DEVICE_PRESENT_MODE_FIFO_LATEST_READY_FEATURES_KHR           StructureType = 1000361000
	PHYSICAL_DEVICE_PIPELINE_CACHE_INCREMENTAL_MODE_FEATURES_SEC          StructureType = 1000637000
)

type PhysicalDevice struct {
	handle C.VkPhysicalDevice
}

type InstanceCreateFlags uint32

type ApplicationInfo struct {
	ApplicationName    string
	ApplicationVersion uint32
	EngineName         string
	EngineVersion      uint32
	ApiVersion         uint32
}

type InstanceCreateInfo struct {
	Flags                 InstanceCreateFlags
	ApplicationInfo       *ApplicationInfo
	EnabledLayerNames     []string
	EnabledExtensionNames []string
}

const (
	ApiVersion_1_0 uint32 = C.VK_API_VERSION_1_0
	ApiVersion_1_1 uint32 = C.VK_API_VERSION_1_1
	ApiVersion_1_2 uint32 = C.VK_API_VERSION_1_2
	ApiVersion_1_3 uint32 = C.VK_API_VERSION_1_3
	ApiVersion_1_4 uint32 = C.VK_API_VERSION_1_4
)

func MakeApiVersion(variant, major, minor, patch uint32) uint32 {
	return (variant << 29) | (major << 22) | (minor << 12) | patch
}

func ApiVersionVariant(version uint32) uint32 {
	return version >> 29
}

func ApiVersionMajor(version uint32) uint32 {
	return (version >> 22) & 0x7F
}

func ApiVersionMinor(version uint32) uint32 {
	return (version >> 12) & 0x3FF
}

func ApiVersionPatch(version uint32) uint32 {
	return version & 0xFFF
}

type SurfaceKHR struct {
	handle C.VkSurfaceKHR
}

type SwapchainKHR struct {
	handle C.VkSwapchainKHR
}

type Image struct {
	handle C.VkImage
}

type ImageView struct {
	handle C.VkImageView
}

// Swapchain support structures
type SurfaceFormatKHR struct {
	Format     Format
	ColorSpace ColorSpaceKHR
}

type SurfaceCapabilitiesKHR struct {
	MinImageCount           uint32
	MaxImageCount           uint32
	CurrentExtent           Extent2D
	MinImageExtent          Extent2D
	MaxImageExtent          Extent2D
	MaxImageArrayLayers     uint32
	SupportedTransforms     SurfaceTransformFlagsKHR
	CurrentTransform        SurfaceTransformFlagsKHR
	SupportedCompositeAlpha CompositeAlphaFlagsKHR
	SupportedUsageFlags     ImageUsageFlags
}

type Extent2D struct {
	Width  uint32
	Height uint32
}

// Enums and flags
type Format int32
type ColorSpaceKHR int32
type PresentModeKHR int32
type SurfaceTransformFlagsKHR uint32
type CompositeAlphaFlagsKHR uint32
type ImageUsageFlags uint32

const (
	// Common formats
	FORMAT_B8G8R8A8_SRGB  Format = C.VK_FORMAT_B8G8R8A8_SRGB
	FORMAT_B8G8R8A8_UNORM Format = C.VK_FORMAT_B8G8R8A8_UNORM

	// Color spaces
	COLOR_SPACE_SRGB_NONLINEAR_KHR ColorSpaceKHR = C.VK_COLOR_SPACE_SRGB_NONLINEAR_KHR

	// Present modes
	PRESENT_MODE_IMMEDIATE_KHR    PresentModeKHR = C.VK_PRESENT_MODE_IMMEDIATE_KHR
	PRESENT_MODE_MAILBOX_KHR      PresentModeKHR = C.VK_PRESENT_MODE_MAILBOX_KHR
	PRESENT_MODE_FIFO_KHR         PresentModeKHR = C.VK_PRESENT_MODE_FIFO_KHR
	PRESENT_MODE_FIFO_RELAXED_KHR PresentModeKHR = C.VK_PRESENT_MODE_FIFO_RELAXED_KHR

	// Image usage
	IMAGE_USAGE_COLOR_ATTACHMENT_BIT ImageUsageFlags = C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT
	IMAGE_USAGE_TRANSFER_DST_BIT     ImageUsageFlags = C.VK_IMAGE_USAGE_TRANSFER_DST_BIT

	// Composite alpha
	COMPOSITE_ALPHA_OPAQUE_BIT_KHR CompositeAlphaFlagsKHR = C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR

	// Surface transform
	SURFACE_TRANSFORM_IDENTITY_BIT_KHR SurfaceTransformFlagsKHR = C.VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR
)

// Device type
type Device struct {
	handle C.VkDevice
}

type Queue struct {
	handle C.VkQueue
}

// Queue family and device types
type QueueFamilyProperties struct {
	QueueFlags                  QueueFlags
	QueueCount                  uint32
	TimestampValidBits          uint32
	MinImageTransferGranularity Extent3D
}

type Extent3D struct {
	Width  uint32
	Height uint32
	Depth  uint32
}

type QueueFlags uint32

const (
	QUEUE_GRAPHICS_BIT       QueueFlags = C.VK_QUEUE_GRAPHICS_BIT
	QUEUE_COMPUTE_BIT        QueueFlags = C.VK_QUEUE_COMPUTE_BIT
	QUEUE_TRANSFER_BIT       QueueFlags = C.VK_QUEUE_TRANSFER_BIT
	QUEUE_SPARSE_BINDING_BIT QueueFlags = C.VK_QUEUE_SPARSE_BINDING_BIT
)

type DeviceQueueCreateInfo struct {
	QueueFamilyIndex uint32
	QueuePriorities  []float32
}

type DeviceCreateInfo struct {
	QueueCreateInfos      []DeviceQueueCreateInfo
	EnabledLayerNames     []string
	EnabledExtensionNames []string
	Vulkan13Features      *PhysicalDeviceVulkan13Features
}

type PhysicalDeviceFeatures struct {
	// We'll leave this empty for now - can add fields as needed
}

// Image view types
type ImageViewCreateInfo struct {
	Image            Image
	ViewType         ImageViewType
	Format           Format
	Components       ComponentMapping
	SubresourceRange ImageSubresourceRange
}

type ImageViewType int32
type ComponentSwizzle int32

const (
	IMAGE_VIEW_TYPE_1D         ImageViewType = C.VK_IMAGE_VIEW_TYPE_1D
	IMAGE_VIEW_TYPE_2D         ImageViewType = C.VK_IMAGE_VIEW_TYPE_2D
	IMAGE_VIEW_TYPE_3D         ImageViewType = C.VK_IMAGE_VIEW_TYPE_3D
	IMAGE_VIEW_TYPE_CUBE       ImageViewType = C.VK_IMAGE_VIEW_TYPE_CUBE
	IMAGE_VIEW_TYPE_1D_ARRAY   ImageViewType = C.VK_IMAGE_VIEW_TYPE_1D_ARRAY
	IMAGE_VIEW_TYPE_2D_ARRAY   ImageViewType = C.VK_IMAGE_VIEW_TYPE_2D_ARRAY
	IMAGE_VIEW_TYPE_CUBE_ARRAY ImageViewType = C.VK_IMAGE_VIEW_TYPE_CUBE_ARRAY

	COMPONENT_SWIZZLE_IDENTITY ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_IDENTITY
	COMPONENT_SWIZZLE_ZERO     ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_ZERO
	COMPONENT_SWIZZLE_ONE      ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_ONE
	COMPONENT_SWIZZLE_R        ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_R
	COMPONENT_SWIZZLE_G        ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_G
	COMPONENT_SWIZZLE_B        ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_B
	COMPONENT_SWIZZLE_A        ComponentSwizzle = C.VK_COMPONENT_SWIZZLE_A
)

type ComponentMapping struct {
	R ComponentSwizzle
	G ComponentSwizzle
	B ComponentSwizzle
	A ComponentSwizzle
}

type ImageSubresourceRange struct {
	AspectMask     ImageAspectFlags
	BaseMipLevel   uint32
	LevelCount     uint32
	BaseArrayLayer uint32
	LayerCount     uint32
}

type ImageAspectFlags uint32

const (
	IMAGE_ASPECT_COLOR_BIT   ImageAspectFlags = C.VK_IMAGE_ASPECT_COLOR_BIT
	IMAGE_ASPECT_DEPTH_BIT   ImageAspectFlags = C.VK_IMAGE_ASPECT_DEPTH_BIT
	IMAGE_ASPECT_STENCIL_BIT ImageAspectFlags = C.VK_IMAGE_ASPECT_STENCIL_BIT
)

type PipelineLayout struct {
	handle C.VkPipelineLayout
}

type Pipeline struct {
	handle C.VkPipeline
}

type PipelineLayoutCreateInfo struct {
	SetLayouts         []DescriptorSetLayout
	PushConstantRanges []PushConstantRange
}

type DescriptorSetLayout struct {
	handle C.VkDescriptorSetLayout
}

type PushConstantRange struct {
	StageFlags ShaderStageFlags
	Offset     uint32
	Size       uint32
}

type ShaderStageFlags uint32

const (
	SHADER_STAGE_VERTEX_BIT   ShaderStageFlags = C.VK_SHADER_STAGE_VERTEX_BIT
	SHADER_STAGE_FRAGMENT_BIT ShaderStageFlags = C.VK_SHADER_STAGE_FRAGMENT_BIT
	SHADER_STAGE_COMPUTE_BIT  ShaderStageFlags = C.VK_SHADER_STAGE_COMPUTE_BIT
	SHADER_STAGE_ALL_GRAPHICS ShaderStageFlags = C.VK_SHADER_STAGE_ALL_GRAPHICS
)

type GraphicsPipelineCreateInfo struct {
	Stages             []PipelineShaderStageCreateInfo
	VertexInputState   *PipelineVertexInputStateCreateInfo
	InputAssemblyState *PipelineInputAssemblyStateCreateInfo
	ViewportState      *PipelineViewportStateCreateInfo
	RasterizationState *PipelineRasterizationStateCreateInfo
	MultisampleState   *PipelineMultisampleStateCreateInfo
	ColorBlendState    *PipelineColorBlendStateCreateInfo
	DynamicState       *PipelineDynamicStateCreateInfo
	Layout             PipelineLayout
	RenderingInfo      *PipelineRenderingCreateInfo
}

type PipelineShaderStageCreateInfo struct {
	Stage  ShaderStageFlags
	Module ShaderModule
	Name   string
}

type PipelineVertexInputStateCreateInfo struct {
	// Empty for now - vertices hardcoded in shader
}

type PipelineInputAssemblyStateCreateInfo struct {
	Topology               PrimitiveTopology
	PrimitiveRestartEnable bool
}

type PrimitiveTopology int32

const (
	PRIMITIVE_TOPOLOGY_POINT_LIST    PrimitiveTopology = C.VK_PRIMITIVE_TOPOLOGY_POINT_LIST
	PRIMITIVE_TOPOLOGY_LINE_LIST     PrimitiveTopology = C.VK_PRIMITIVE_TOPOLOGY_LINE_LIST
	PRIMITIVE_TOPOLOGY_TRIANGLE_LIST PrimitiveTopology = C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST
)

type PipelineViewportStateCreateInfo struct {
	Viewports []Viewport
	Scissors  []Rect2D
}

type Viewport struct {
	X        float32
	Y        float32
	Width    float32
	Height   float32
	MinDepth float32
	MaxDepth float32
}

type Rect2D struct {
	Offset Offset2D
	Extent Extent2D
}

type Offset2D struct {
	X int32
	Y int32
}

type PipelineRasterizationStateCreateInfo struct {
	DepthClampEnable        bool
	RasterizerDiscardEnable bool
	PolygonMode             PolygonMode
	CullMode                CullModeFlags
	FrontFace               FrontFace
	DepthBiasEnable         bool
	LineWidth               float32
}

type PolygonMode int32
type CullModeFlags uint32
type FrontFace int32

const (
	POLYGON_MODE_FILL            PolygonMode   = C.VK_POLYGON_MODE_FILL
	POLYGON_MODE_LINE            PolygonMode   = C.VK_POLYGON_MODE_LINE
	POLYGON_MODE_POINT           PolygonMode   = C.VK_POLYGON_MODE_POINT
	CULL_MODE_NONE               CullModeFlags = 0
	CULL_MODE_FRONT_BIT          CullModeFlags = C.VK_CULL_MODE_FRONT_BIT
	CULL_MODE_BACK_BIT           CullModeFlags = C.VK_CULL_MODE_BACK_BIT
	FRONT_FACE_COUNTER_CLOCKWISE FrontFace     = C.VK_FRONT_FACE_COUNTER_CLOCKWISE
	FRONT_FACE_CLOCKWISE         FrontFace     = C.VK_FRONT_FACE_CLOCKWISE
)

type PipelineMultisampleStateCreateInfo struct {
	RasterizationSamples SampleCountFlags
	SampleShadingEnable  bool
}

type SampleCountFlags int32

const (
	SAMPLE_COUNT_1_BIT SampleCountFlags = C.VK_SAMPLE_COUNT_1_BIT
	SAMPLE_COUNT_2_BIT SampleCountFlags = C.VK_SAMPLE_COUNT_2_BIT
	SAMPLE_COUNT_4_BIT SampleCountFlags = C.VK_SAMPLE_COUNT_4_BIT
	SAMPLE_COUNT_8_BIT SampleCountFlags = C.VK_SAMPLE_COUNT_8_BIT
)

type PipelineColorBlendStateCreateInfo struct {
	LogicOpEnable bool
	LogicOp       LogicOp
	Attachments   []PipelineColorBlendAttachmentState
}

type LogicOp int32

const (
	LOGIC_OP_COPY LogicOp = C.VK_LOGIC_OP_COPY
)

type PipelineColorBlendAttachmentState struct {
	BlendEnable    bool
	ColorWriteMask ColorComponentFlags
}

type ColorComponentFlags uint32

const (
	COLOR_COMPONENT_R_BIT ColorComponentFlags = C.VK_COLOR_COMPONENT_R_BIT
	COLOR_COMPONENT_G_BIT ColorComponentFlags = C.VK_COLOR_COMPONENT_G_BIT
	COLOR_COMPONENT_B_BIT ColorComponentFlags = C.VK_COLOR_COMPONENT_B_BIT
	COLOR_COMPONENT_A_BIT ColorComponentFlags = C.VK_COLOR_COMPONENT_A_BIT
	COLOR_COMPONENT_ALL   ColorComponentFlags = COLOR_COMPONENT_R_BIT | COLOR_COMPONENT_G_BIT | COLOR_COMPONENT_B_BIT | COLOR_COMPONENT_A_BIT
)

type PipelineDynamicStateCreateInfo struct {
	DynamicStates []DynamicState
}

type DynamicState int32

const (
	DYNAMIC_STATE_VIEWPORT DynamicState = C.VK_DYNAMIC_STATE_VIEWPORT
	DYNAMIC_STATE_SCISSOR  DynamicState = C.VK_DYNAMIC_STATE_SCISSOR
)

type PipelineRenderingCreateInfo struct {
	ViewMask                uint32
	ColorAttachmentFormats  []Format
	DepthAttachmentFormat   Format
	StencilAttachmentFormat Format
}

type PhysicalDeviceVulkan13Features struct {
	DynamicRendering bool
}
