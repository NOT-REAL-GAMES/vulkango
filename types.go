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

type StructureType int32

const (
	APPLICATION_INFO                                            StructureType = 0
	INSTANCE_CREATE_INFO                                        StructureType = 1
	DEVICE_QUEUE_CREATE_INFO                                    StructureType = 2
	DEVICE_CREATE_INFO                                          StructureType = 3
	SUBMIT_INFO                                                 StructureType = 4
	MEMORY_ALLOCATE_INFO                                        StructureType = 5
	MAPPED_MEMORY_RANGE                                         StructureType = 6
	BIND_SPARSE_INFO                                            StructureType = 7
	FENCE_CREATE_INFO                                           StructureType = 8
	SEMAPHORE_CREATE_INFO                                       StructureType = 9
	EVENT_CREATE_INFO                                           StructureType = 10
	QUERY_POOL_CREATE_INFO                                      StructureType = 11
	BUFFER_CREATE_INFO                                          StructureType = 12
	BUFFER_VIEW_CREATE_INFO                                     StructureType = 13
	IMAGE_CREATE_INFO                                           StructureType = 14
	IMAGE_VIEW_CREATE_INFO                                      StructureType = 15
	SHADER_MODULE_CREATE_INFO                                   StructureType = 16
	PIPELINE_CACHE_CREATE_INFO                                  StructureType = 17
	PIPELINE_SHADER_STAGE_CREATE_INFO                           StructureType = 18
	PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO                     StructureType = 19
	PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO                   StructureType = 20
	PIPELINE_TESSELLATION_STATE_CREATE_INFO                     StructureType = 21
	PIPELINE_VIEWPORT_STATE_CREATE_INFO                         StructureType = 22
	PIPELINE_RASTERIZATION_STATE_CREATE_INFO                    StructureType = 23
	PIPELINE_MULTISAMPLE_STATE_CREATE_INFO                      StructureType = 24
	PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO                    StructureType = 25
	PIPELINE_COLOR_BLEND_STATE_CREATE_INFO                      StructureType = 26
	PIPELINE_DYNAMIC_STATE_CREATE_INFO                          StructureType = 27
	GRAPHICS_PIPELINE_CREATE_INFO                               StructureType = 28
	COMPUTE_PIPELINE_CREATE_INFO                                StructureType = 29
	PIPELINE_LAYOUT_CREATE_INFO                                 StructureType = 30
	SAMPLER_CREATE_INFO                                         StructureType = 31
	DESCRIPTOR_SET_LAYOUT_CREATE_INFO                           StructureType = 32
	DESCRIPTOR_POOL_CREATE_INFO                                 StructureType = 33
	DESCRIPTOR_SET_ALLOCATE_INFO                                StructureType = 34
	WRITE_DESCRIPTOR_SET                                        StructureType = 35
	COPY_DESCRIPTOR_SET                                         StructureType = 36
	FRAMEBUFFER_CREATE_INFO                                     StructureType = 37
	RENDER_PASS_CREATE_INFO                                     StructureType = 38
	COMMAND_POOL_CREATE_INFO                                    StructureType = 39
	COMMAND_BUFFER_ALLOCATE_INFO                                StructureType = 40
	COMMAND_BUFFER_INHERITANCE_INFO                             StructureType = 41
	COMMAND_BUFFER_BEGIN_INFO                                   StructureType = 42
	RENDER_PASS_BEGIN_INFO                                      StructureType = 43
	BUFFER_MEMORY_BARRIER                                       StructureType = 44
	IMAGE_MEMORY_BARRIER                                        StructureType = 45
	MEMORY_BARRIER                                              StructureType = 46
	LOADER_INSTANCE_CREATE_INFO                                 StructureType = 47
	LOADER_DEVICE_CREATE_INFO                                   StructureType = 48
	PHYSICAL_DEVICE_SUBGROUP_PROPERTIES                         StructureType = 1000094000
	BIND_BUFFER_MEMORY_INFO                                     StructureType = 1000157000
	BIND_IMAGE_MEMORY_INFO                                      StructureType = 1000157001
	PHYSICAL_DEVICE_16BIT_STORAGE_FEATURES                      StructureType = 1000083000
	MEMORY_DEDICATED_REQUIREMENTS                               StructureType = 1000127000
	MEMORY_DEDICATED_ALLOCATE_INFO                              StructureType = 1000127001
	MEMORY_ALLOCATE_FLAGS_INFO                                  StructureType = 1000060000
	DEVICE_GROUP_RENDER_PASS_BEGIN_INFO                         StructureType = 1000060003
	DEVICE_GROUP_COMMAND_BUFFER_BEGIN_INFO                      StructureType = 1000060004
	DEVICE_GROUP_SUBMIT_INFO                                    StructureType = 1000060005
	DEVICE_GROUP_BIND_SPARSE_INFO                               StructureType = 1000060006
	BIND_BUFFER_MEMORY_DEVICE_GROUP_INFO                        StructureType = 1000060013
	BIND_IMAGE_MEMORY_DEVICE_GROUP_INFO                         StructureType = 1000060014
	PHYSICAL_DEVICE_GROUP_PROPERTIES                            StructureType = 1000070000
	DEVICE_GROUP_DEVICE_CREATE_INFO                             StructureType = 1000070001
	BUFFER_MEMORY_REQUIREMENTS_INFO_2                           StructureType = 1000146000
	IMAGE_MEMORY_REQUIREMENTS_INFO_2                            StructureType = 1000146001
	IMAGE_SPARSE_MEMORY_REQUIREMENTS_INFO_2                     StructureType = 1000146002
	MEMORY_REQUIREMENTS_2                                       StructureType = 1000146003
	SPARSE_IMAGE_MEMORY_REQUIREMENTS_2                          StructureType = 1000146004
	PHYSICAL_DEVICE_FEATURES_2                                  StructureType = 1000059000
	PHYSICAL_DEVICE_PROPERTIES_2                                StructureType = 1000059001
	FORMAT_PROPERTIES_2                                         StructureType = 1000059002
	IMAGE_FORMAT_PROPERTIES_2                                   StructureType = 1000059003
	PHYSICAL_DEVICE_IMAGE_FORMAT_INFO_2                         StructureType = 1000059004
	QUEUE_FAMILY_PROPERTIES_2                                   StructureType = 1000059005
	PHYSICAL_DEVICE_MEMORY_PROPERTIES_2                         StructureType = 1000059006
	SPARSE_IMAGE_FORMAT_PROPERTIES_2                            StructureType = 1000059007
	PHYSICAL_DEVICE_SPARSE_IMAGE_FORMAT_INFO_2                  StructureType = 1000059008
	PHYSICAL_DEVICE_POINT_CLIPPING_PROPERTIES                   StructureType = 1000117000
	RENDER_PASS_INPUT_ATTACHMENT_ASPECT_CREATE_INFO             StructureType = 1000117001
	IMAGE_VIEW_USAGE_CREATE_INFO                                StructureType = 1000117002
	PIPELINE_TESSELLATION_DOMAIN_ORIGIN_STATE_CREATE_INFO       StructureType = 1000117003
	RENDER_PASS_MULTIVIEW_CREATE_INFO                           StructureType = 1000053000
	PHYSICAL_DEVICE_MULTIVIEW_FEATURES                          StructureType = 1000053001
	PHYSICAL_DEVICE_MULTIVIEW_PROPERTIES                        StructureType = 1000053002
	PHYSICAL_DEVICE_VARIABLE_POINTERS_FEATURES                  StructureType = 1000120000
	PROTECTED_SUBMIT_INFO                                       StructureType = 1000145000
	PHYSICAL_DEVICE_PROTECTED_MEMORY_FEATURES                   StructureType = 1000145001
	PHYSICAL_DEVICE_PROTECTED_MEMORY_PROPERTIES                 StructureType = 1000145002
	DEVICE_QUEUE_INFO_2                                         StructureType = 1000145003
	SAMPLER_YCBCR_CONVERSION_CREATE_INFO                        StructureType = 1000156000
	SAMPLER_YCBCR_CONVERSION_INFO                               StructureType = 1000156001
	BIND_IMAGE_PLANE_MEMORY_INFO                                StructureType = 1000156002
	IMAGE_PLANE_MEMORY_REQUIREMENTS_INFO                        StructureType = 1000156003
	PHYSICAL_DEVICE_SAMPLER_YCBCR_CONVERSION_FEATURES           StructureType = 1000156004
	SAMPLER_YCBCR_CONVERSION_IMAGE_FORMAT_PROPERTIES            StructureType = 1000156005
	DESCRIPTOR_UPDATE_TEMPLATE_CREATE_INFO                      StructureType = 1000085000
	PHYSICAL_DEVICE_EXTERNAL_IMAGE_FORMAT_INFO                  StructureType = 1000071000
	EXTERNAL_IMAGE_FORMAT_PROPERTIES                            StructureType = 1000071001
	PHYSICAL_DEVICE_EXTERNAL_BUFFER_INFO                        StructureType = 1000071002
	EXTERNAL_BUFFER_PROPERTIES                                  StructureType = 1000071003
	PHYSICAL_DEVICE_ID_PROPERTIES                               StructureType = 1000071004
	EXTERNAL_MEMORY_BUFFER_CREATE_INFO                          StructureType = 1000072000
	EXTERNAL_MEMORY_IMAGE_CREATE_INFO                           StructureType = 1000072001
	EXPORT_MEMORY_ALLOCATE_INFO                                 StructureType = 1000072002
	PHYSICAL_DEVICE_EXTERNAL_FENCE_INFO                         StructureType = 1000112000
	EXTERNAL_FENCE_PROPERTIES                                   StructureType = 1000112001
	EXPORT_FENCE_CREATE_INFO                                    StructureType = 1000113000
	EXPORT_SEMAPHORE_CREATE_INFO                                StructureType = 1000077000
	PHYSICAL_DEVICE_EXTERNAL_SEMAPHORE_INFO                     StructureType = 1000076000
	EXTERNAL_SEMAPHORE_PROPERTIES                               StructureType = 1000076001
	PHYSICAL_DEVICE_MAINTENANCE_3_PROPERTIES                    StructureType = 1000168000
	DESCRIPTOR_SET_LAYOUT_SUPPORT                               StructureType = 1000168001
	PHYSICAL_DEVICE_SHADER_DRAW_PARAMETERS_FEATURES             StructureType = 1000063000
	PHYSICAL_DEVICE_VULKAN_1_1_FEATURES                         StructureType = 49
	PHYSICAL_DEVICE_VULKAN_1_1_PROPERTIES                       StructureType = 50
	PHYSICAL_DEVICE_VULKAN_1_2_FEATURES                         StructureType = 51
	PHYSICAL_DEVICE_VULKAN_1_2_PROPERTIES                       StructureType = 52
	IMAGE_FORMAT_LIST_CREATE_INFO                               StructureType = 1000147000
	ATTACHMENT_DESCRIPTION_2                                    StructureType = 1000109000
	ATTACHMENT_REFERENCE_2                                      StructureType = 1000109001
	SUBPASS_DESCRIPTION_2                                       StructureType = 1000109002
	SUBPASS_DEPENDENCY_2                                        StructureType = 1000109003
	RENDER_PASS_CREATE_INFO_2                                   StructureType = 1000109004
	SUBPASS_BEGIN_INFO                                          StructureType = 1000109005
	SUBPASS_END_INFO                                            StructureType = 1000109006
	PHYSICAL_DEVICE_8BIT_STORAGE_FEATURES                       StructureType = 1000177000
	PHYSICAL_DEVICE_DRIVER_PROPERTIES                           StructureType = 1000196000
	PHYSICAL_DEVICE_SHADER_ATOMIC_INT64_FEATURES                StructureType = 1000180000
	PHYSICAL_DEVICE_SHADER_FLOAT16_INT8_FEATURES                StructureType = 1000082000
	PHYSICAL_DEVICE_FLOAT_CONTROLS_PROPERTIES                   StructureType = 1000197000
	DESCRIPTOR_SET_LAYOUT_BINDING_FLAGS_CREATE_INFO             StructureType = 1000161000
	PHYSICAL_DEVICE_DESCRIPTOR_INDEXING_FEATURES                StructureType = 1000161001
	PHYSICAL_DEVICE_DESCRIPTOR_INDEXING_PROPERTIES              StructureType = 1000161002
	DESCRIPTOR_SET_VARIABLE_DESCRIPTOR_COUNT_ALLOCATE_INFO      StructureType = 1000161003
	DESCRIPTOR_SET_VARIABLE_DESCRIPTOR_COUNT_LAYOUT_SUPPORT     StructureType = 1000161004
	PHYSICAL_DEVICE_DEPTH_STENCIL_RESOLVE_PROPERTIES            StructureType = 1000199000
	SUBPASS_DESCRIPTION_DEPTH_STENCIL_RESOLVE                   StructureType = 1000199001
	PHYSICAL_DEVICE_SCALAR_BLOCK_LAYOUT_FEATURES                StructureType = 1000221000
	IMAGE_STENCIL_USAGE_CREATE_INFO                             StructureType = 1000246000
	PHYSICAL_DEVICE_SAMPLER_FILTER_MINMAX_PROPERTIES            StructureType = 1000130000
	SAMPLER_REDUCTION_MODE_CREATE_INFO                          StructureType = 1000130001
	PHYSICAL_DEVICE_VULKAN_MEMORY_MODEL_FEATURES                StructureType = 1000211000
	PHYSICAL_DEVICE_IMAGELESS_FRAMEBUFFER_FEATURES              StructureType = 1000108000
	FRAMEBUFFER_ATTACHMENTS_CREATE_INFO                         StructureType = 1000108001
	FRAMEBUFFER_ATTACHMENT_IMAGE_INFO                           StructureType = 1000108002
	RENDER_PASS_ATTACHMENT_BEGIN_INFO                           StructureType = 1000108003
	PHYSICAL_DEVICE_UNIFORM_BUFFER_STANDARD_LAYOUT_FEATURES     StructureType = 1000253000
	PHYSICAL_DEVICE_SHADER_SUBGROUP_EXTENDED_TYPES_FEATURES     StructureType = 1000175000
	PHYSICAL_DEVICE_SEPARATE_DEPTH_STENCIL_LAYOUTS_FEATURES     StructureType = 1000241000
	ATTACHMENT_REFERENCE_STENCIL_LAYOUT                         StructureType = 1000241001
	ATTACHMENT_DESCRIPTION_STENCIL_LAYOUT                       StructureType = 1000241002
	PHYSICAL_DEVICE_HOST_QUERY_RESET_FEATURES                   StructureType = 1000261000
	PHYSICAL_DEVICE_TIMELINE_SEMAPHORE_FEATURES                 StructureType = 1000207000
	PHYSICAL_DEVICE_TIMELINE_SEMAPHORE_PROPERTIES               StructureType = 1000207001
	SEMAPHORE_TYPE_CREATE_INFO                                  StructureType = 1000207002
	TIMELINE_SEMAPHORE_SUBMIT_INFO                              StructureType = 1000207003
	SEMAPHORE_WAIT_INFO                                         StructureType = 1000207004
	SEMAPHORE_SIGNAL_INFO                                       StructureType = 1000207005
	PHYSICAL_DEVICE_BUFFER_DEVICE_ADDRESS_FEATURES              StructureType = 1000257000
	BUFFER_DEVICE_ADDRESS_INFO                                  StructureType = 1000244001
	BUFFER_OPAQUE_CAPTURE_ADDRESS_CREATE_INFO                   StructureType = 1000257002
	MEMORY_OPAQUE_CAPTURE_ADDRESS_ALLOCATE_INFO                 StructureType = 1000257003
	DEVICE_MEMORY_OPAQUE_CAPTURE_ADDRESS_INFO                   StructureType = 1000257004
	PHYSICAL_DEVICE_VULKAN_1_3_FEATURES                         StructureType = 53
	PHYSICAL_DEVICE_VULKAN_1_3_PROPERTIES                       StructureType = 54
	PIPELINE_CREATION_FEEDBACK_CREATE_INFO                      StructureType = 1000192000
	PHYSICAL_DEVICE_SHADER_TERMINATE_INVOCATION_FEATURES        StructureType = 1000215000
	PHYSICAL_DEVICE_TOOL_PROPERTIES                             StructureType = 1000245000
	PHYSICAL_DEVICE_SHADER_DEMOTE_TO_HELPER_INVOCATION_FEATURES StructureType = 1000276000
	PHYSICAL_DEVICE_PRIVATE_DATA_FEATURES                       StructureType = 1000295000
	DEVICE_PRIVATE_DATA_CREATE_INFO                             StructureType = 1000295001
	PRIVATE_DATA_SLOT_CREATE_INFO                               StructureType = 1000295002
	PHYSICAL_DEVICE_PIPELINE_CREATION_CACHE_CONTROL_FEATURES    StructureType = 1000297000
	MEMORY_BARRIER_2                                            StructureType = 1000314000
	BUFFER_MEMORY_BARRIER_2                                     StructureType = 1000314001
	IMAGE_MEMORY_BARRIER_2                                      StructureType = 1000314002
	DEPENDENCY_INFO                                             StructureType = 1000314003
	SUBMIT_INFO_2                                               StructureType = 1000314004
	SEMAPHORE_SUBMIT_INFO                                       StructureType = 1000314005
	COMMAND_BUFFER_SUBMIT_INFO                                  StructureType = 1000314006
	PHYSICAL_DEVICE_SYNCHRONIZATION_2_FEATURES                  StructureType = 1000314007
	PHYSICAL_DEVICE_ZERO_INITIALIZE_WORKGROUP_MEMORY_FEATURES   StructureType = 1000325000
	PHYSICAL_DEVICE_IMAGE_ROBUSTNESS_FEATURES                   StructureType = 1000335000
	COPY_BUFFER_INFO_2                                          StructureType = 1000337000
	COPY_IMAGE_INFO_2                                           StructureType = 1000337001
	COPY_BUFFER_TO_IMAGE_INFO_2                                 StructureType = 1000337002
	COPY_IMAGE_TO_BUFFER_INFO_2                                 StructureType = 1000337003
	BLIT_IMAGE_INFO_2                                           StructureType = 1000337004
	RESOLVE_IMAGE_INFO_2                                        StructureType = 1000337005
	BUFFER_COPY_2                                               StructureType = 1000337006
	IMAGE_COPY_2                                                StructureType = 1000337007
	IMAGE_BLIT_2                                                StructureType = 1000337008
	BUFFER_IMAGE_COPY_2                                         StructureType = 1000337009
	IMAGE_RESOLVE_2                                             StructureType = 1000337010
	PHYSICAL_DEVICE_SUBGROUP_SIZE_CONTROL_PROPERTIES            StructureType = 1000225000
	PIPELINE_SHADER_STAGE_REQUIRED_SUBGROUP_SIZE_CREATE_INFO    StructureType = 1000225001
	PHYSICAL_DEVICE_SUBGROUP_SIZE_CONTROL_FEATURES              StructureType = 1000225002
	PHYSICAL_DEVICE_INLINE_UNIFORM_BLOCK_FEATURES               StructureType = 1000138000
	PHYSICAL_DEVICE_INLINE_UNIFORM_BLOCK_PROPERTIES             StructureType = 1000138001
	WRITE_DESCRIPTOR_SET_INLINE_UNIFORM_BLOCK                   StructureType = 1000138002
	DESCRIPTOR_POOL_INLINE_UNIFORM_BLOCK_CREATE_INFO            StructureType = 1000138003
	PHYSICAL_DEVICE_TEXTURE_COMPRESSION_ASTC_HDR_FEATURES       StructureType = 1000066000
	RENDERING_INFO                                              StructureType = 1000044000
	RENDERING_ATTACHMENT_INFO                                   StructureType = 1000044001
	PIPELINE_RENDERING_CREATE_INFO                              StructureType = 1000044002
	PHYSICAL_DEVICE_DYNAMIC_RENDERING_FEATURES                  StructureType = 1000044003
	COMMAND_BUFFER_INHERITANCE_RENDERING_INFO                   StructureType = 1000044004
	PHYSICAL_DEVICE_SHADER_INTEGER_DOT_PRODUCT_FEATURES         StructureType = 1000280000
	PHYSICAL_DEVICE_SHADER_INTEGER_DOT_PRODUCT_PROPERTIES       StructureType = 1000280001
	PHYSICAL_DEVICE_TEXEL_BUFFER_ALIGNMENT_PROPERTIES           StructureType = 1000281001
	FORMAT_PROPERTIES_3                                         StructureType = 1000360000
	PHYSICAL_DEVICE_MAINTENANCE_4_FEATURES                      StructureType = 1000413000
	PHYSICAL_DEVICE_MAINTENANCE_4_PROPERTIES                    StructureType = 1000413001
	DEVICE_BUFFER_MEMORY_REQUIREMENTS                           StructureType = 1000413002
	DEVICE_IMAGE_MEMORY_REQUIREMENTS                            StructureType = 1000413003
	PHYSICAL_DEVICE_VULKAN_1_4_FEATURES                         StructureType = 55
	PHYSICAL_DEVICE_VULKAN_1_4_PROPERTIES                       StructureType = 56
	DEVICE_QUEUE_GLOBAL_PRIORITY_CREATE_INFO                    StructureType = 1000174000
	PHYSICAL_DEVICE_GLOBAL_PRIORITY_QUERY_FEATURES              StructureType = 1000388000
	QUEUE_FAMILY_GLOBAL_PRIORITY_PROPERTIES                     StructureType = 1000388001
	PHYSICAL_DEVICE_SHADER_SUBGROUP_ROTATE_FEATURES             StructureType = 1000416000
	PHYSICAL_DEVICE_SHADER_FLOAT_CONTROLS_2_FEATURES            StructureType = 1000528000
	PHYSICAL_DEVICE_SHADER_EXPECT_ASSUME_FEATURES               StructureType = 1000544000
	PHYSICAL_DEVICE_LINE_RASTERIZATION_FEATURES                 StructureType = 1000259000
	PIPELINE_RASTERIZATION_LINE_STATE_CREATE_INFO               StructureType = 1000259001
	PHYSICAL_DEVICE_LINE_RASTERIZATION_PROPERTIES               StructureType = 1000259002
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_DIVISOR_PROPERTIES         StructureType = 1000525000
	PIPELINE_VERTEX_INPUT_DIVISOR_STATE_CREATE_INFO             StructureType = 1000190001
	PHYSICAL_DEVICE_VERTEX_ATTRIBUTE_DIVISOR_FEATURES           StructureType = 1000190002
	PHYSICAL_DEVICE_INDEX_TYPE_UINT8_FEATURES                   StructureType = 1000265000
	MEMORY_MAP_INFO                                             StructureType = 1000271000
	MEMORY_UNMAP_INFO                                           StructureType = 1000271001
	PHYSICAL_DEVICE_MAINTENANCE_5_FEATURES                      StructureType = 1000470000
	PHYSICAL_DEVICE_MAINTENANCE_5_PROPERTIES                    StructureType = 1000470001
	RENDERING_AREA_INFO                                         StructureType = 1000470003
	DEVICE_IMAGE_SUBRESOURCE_INFO                               StructureType = 1000470004
	SUBRESOURCE_LAYOUT_2                                        StructureType = 1000338002
	IMAGE_SUBRESOURCE_2                                         StructureType = 1000338003
	PIPELINE_CREATE_FLAGS_2_CREATE_INFO                         StructureType = 1000470005
	BUFFER_USAGE_FLAGS_2_CREATE_INFO                            StructureType = 1000470006
	PHYSICAL_DEVICE_PUSH_DESCRIPTOR_PROPERTIES                  StructureType = 1000080000
	PHYSICAL_DEVICE_DYNAMIC_RENDERING_LOCAL_READ_FEATURES       StructureType = 1000232000
	RENDERING_ATTACHMENT_LOCATION_INFO                          StructureType = 1000232001
	RENDERING_INPUT_ATTACHMENT_INDEX_INFO                       StructureType = 1000232002
	PHYSICAL_DEVICE_MAINTENANCE_6_FEATURES                      StructureType = 1000545000
	PHYSICAL_DEVICE_MAINTENANCE_6_PROPERTIES                    StructureType = 1000545001
	BIND_MEMORY_STATUS                                          StructureType = 1000545002
	BIND_DESCRIPTOR_SETS_INFO                                   StructureType = 1000545003
	PUSH_CONSTANTS_INFO                                         StructureType = 1000545004
	PUSH_DESCRIPTOR_SET_INFO                                    StructureType = 1000545005
	PUSH_DESCRIPTOR_SET_WITH_TEMPLATE_INFO                      StructureType = 1000545006
	PHYSICAL_DEVICE_PIPELINE_PROTECTED_ACCESS_FEATURES          StructureType = 1000466000
	PIPELINE_ROBUSTNESS_CREATE_INFO                             StructureType = 1000068000
	PHYSICAL_DEVICE_PIPELINE_ROBUSTNESS_FEATURES                StructureType = 1000068001
	PHYSICAL_DEVICE_PIPELINE_ROBUSTNESS_PROPERTIES              StructureType = 1000068002
	PHYSICAL_DEVICE_HOST_IMAGE_COPY_FEATURES                    StructureType = 1000270000
	PHYSICAL_DEVICE_HOST_IMAGE_COPY_PROPERTIES                  StructureType = 1000270001
	MEMORY_TO_IMAGE_COPY                                        StructureType = 1000270002
	IMAGE_TO_MEMORY_COPY                                        StructureType = 1000270003
	COPY_IMAGE_TO_MEMORY_INFO                                   StructureType = 1000270004
	COPY_MEMORY_TO_IMAGE_INFO                                   StructureType = 1000270005
	HOST_IMAGE_LAYOUT_TRANSITION_INFO                           StructureType = 1000270006
	COPY_IMAGE_TO_IMAGE_INFO                                    StructureType = 1000270007
	SUBRESOURCE_HOST_MEMCPY_SIZE                                StructureType = 1000270008
	HOST_IMAGE_COPY_DEVICE_PERFORMANCE_QUERY                    StructureType = 1000270009
	SWAPCHAIN_CREATE_INFO_KHR                                   StructureType = 1000001000
	PRESENT_INFO_KHR                                            StructureType = 1000001001
	DEVICE_GROUP_PRESENT_CAPABILITIES_KHR                       StructureType = 1000060007
	IMAGE_SWAPCHAIN_CREATE_INFO_KHR                             StructureType = 1000060008
	BIND_IMAGE_MEMORY_SWAPCHAIN_INFO_KHR                        StructureType = 1000060009
	ACQUIRE_NEXT_IMAGE_INFO_KHR                                 StructureType = 1000060010
	DEVICE_GROUP_PRESENT_INFO_KHR                               StructureType = 1000060011
	DEVICE_GROUP_SWAPCHAIN_CREATE_INFO_KHR                      StructureType = 1000060012
)
