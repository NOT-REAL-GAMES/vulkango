package vulkango

/*
#cgo CFLAGS: -I/usr/include
#cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lvulkan
#cgo windows LDFLAGS: -lvulkan-1
#cgo darwin LDFLAGS: -lvulkan

#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>

// Function pointer types for Vulkan Video extensions
typedef VkResult (*PFN_vkGetPhysicalDeviceVideoCapabilitiesKHR)(VkPhysicalDevice, const VkVideoProfileInfoKHR*, VkVideoCapabilitiesKHR*);
typedef VkResult (*PFN_vkCreateVideoSessionKHR)(VkDevice, const VkVideoSessionCreateInfoKHR*, const VkAllocationCallbacks*, VkVideoSessionKHR*);
typedef void (*PFN_vkDestroyVideoSessionKHR)(VkDevice, VkVideoSessionKHR, const VkAllocationCallbacks*);
typedef VkResult (*PFN_vkGetVideoSessionMemoryRequirementsKHR)(VkDevice, VkVideoSessionKHR, uint32_t*, VkVideoSessionMemoryRequirementsKHR*);
typedef VkResult (*PFN_vkBindVideoSessionMemoryKHR)(VkDevice, VkVideoSessionKHR, uint32_t, const VkBindVideoSessionMemoryInfoKHR*);
typedef VkResult (*PFN_vkCreateVideoSessionParametersKHR)(VkDevice, const VkVideoSessionParametersCreateInfoKHR*, const VkAllocationCallbacks*, VkVideoSessionParametersKHR*);
typedef void (*PFN_vkDestroyVideoSessionParametersKHR)(VkDevice, VkVideoSessionParametersKHR, const VkAllocationCallbacks*);
typedef void (*PFN_vkCmdBeginVideoCodingKHR)(VkCommandBuffer, const VkVideoBeginCodingInfoKHR*);
typedef void (*PFN_vkCmdEndVideoCodingKHR)(VkCommandBuffer, const VkVideoEndCodingInfoKHR*);
typedef void (*PFN_vkCmdControlVideoCodingKHR)(VkCommandBuffer, const VkVideoCodingControlInfoKHR*);
typedef void (*PFN_vkCmdDecodeVideoKHR)(VkCommandBuffer, const VkVideoDecodeInfoKHR*);
typedef void (*PFN_vkCmdEncodeVideoKHR)(VkCommandBuffer, const VkVideoEncodeInfoKHR*);

// Global function pointers
static PFN_vkGetPhysicalDeviceVideoCapabilitiesKHR pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR = NULL;
static PFN_vkCreateVideoSessionKHR pfn_vkCreateVideoSessionKHR = NULL;
static PFN_vkDestroyVideoSessionKHR pfn_vkDestroyVideoSessionKHR = NULL;
static PFN_vkGetVideoSessionMemoryRequirementsKHR pfn_vkGetVideoSessionMemoryRequirementsKHR = NULL;
static PFN_vkBindVideoSessionMemoryKHR pfn_vkBindVideoSessionMemoryKHR = NULL;
static PFN_vkCreateVideoSessionParametersKHR pfn_vkCreateVideoSessionParametersKHR = NULL;
static PFN_vkDestroyVideoSessionParametersKHR pfn_vkDestroyVideoSessionParametersKHR = NULL;
static PFN_vkCmdBeginVideoCodingKHR pfn_vkCmdBeginVideoCodingKHR = NULL;
static PFN_vkCmdEndVideoCodingKHR pfn_vkCmdEndVideoCodingKHR = NULL;
static PFN_vkCmdControlVideoCodingKHR pfn_vkCmdControlVideoCodingKHR = NULL;
static PFN_vkCmdDecodeVideoKHR pfn_vkCmdDecodeVideoKHR = NULL;
static PFN_vkCmdEncodeVideoKHR pfn_vkCmdEncodeVideoKHR = NULL;

// Load video extension functions from instance
static int loadVideoFunctionsInstance(VkInstance instance) {
	pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR = (PFN_vkGetPhysicalDeviceVideoCapabilitiesKHR)
		vkGetInstanceProcAddr(instance, "vkGetPhysicalDeviceVideoCapabilitiesKHR");
	return pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR != NULL;
}

// Load video extension functions from device
static int loadVideoFunctionsDevice(VkDevice device) {
	pfn_vkCreateVideoSessionKHR = (PFN_vkCreateVideoSessionKHR)
		vkGetDeviceProcAddr(device, "vkCreateVideoSessionKHR");
	pfn_vkDestroyVideoSessionKHR = (PFN_vkDestroyVideoSessionKHR)
		vkGetDeviceProcAddr(device, "vkDestroyVideoSessionKHR");
	pfn_vkGetVideoSessionMemoryRequirementsKHR = (PFN_vkGetVideoSessionMemoryRequirementsKHR)
		vkGetDeviceProcAddr(device, "vkGetVideoSessionMemoryRequirementsKHR");
	pfn_vkBindVideoSessionMemoryKHR = (PFN_vkBindVideoSessionMemoryKHR)
		vkGetDeviceProcAddr(device, "vkBindVideoSessionMemoryKHR");
	pfn_vkCreateVideoSessionParametersKHR = (PFN_vkCreateVideoSessionParametersKHR)
		vkGetDeviceProcAddr(device, "vkCreateVideoSessionParametersKHR");
	pfn_vkDestroyVideoSessionParametersKHR = (PFN_vkDestroyVideoSessionParametersKHR)
		vkGetDeviceProcAddr(device, "vkDestroyVideoSessionParametersKHR");
	pfn_vkCmdBeginVideoCodingKHR = (PFN_vkCmdBeginVideoCodingKHR)
		vkGetDeviceProcAddr(device, "vkCmdBeginVideoCodingKHR");
	pfn_vkCmdEndVideoCodingKHR = (PFN_vkCmdEndVideoCodingKHR)
		vkGetDeviceProcAddr(device, "vkCmdEndVideoCodingKHR");
	pfn_vkCmdControlVideoCodingKHR = (PFN_vkCmdControlVideoCodingKHR)
		vkGetDeviceProcAddr(device, "vkCmdControlVideoCodingKHR");
	pfn_vkCmdDecodeVideoKHR = (PFN_vkCmdDecodeVideoKHR)
		vkGetDeviceProcAddr(device, "vkCmdDecodeVideoKHR");
	pfn_vkCmdEncodeVideoKHR = (PFN_vkCmdEncodeVideoKHR)
		vkGetDeviceProcAddr(device, "vkCmdEncodeVideoKHR");

	return pfn_vkCreateVideoSessionKHR != NULL;
}

// Wrapper functions that use the loaded function pointers
static VkResult call_vkGetPhysicalDeviceVideoCapabilitiesKHR(VkPhysicalDevice pd, const VkVideoProfileInfoKHR* profile, VkVideoCapabilitiesKHR* caps) {
	if (pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;
	return pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR(pd, profile, caps);
}

// Global to store the queried STD header version
static VkExtensionProperties g_h264EncodeStdHeaderVersion = {0};

// H.264 encode specific capabilities query with proper pNext chaining
static VkResult call_vkGetPhysicalDeviceVideoCapabilitiesH264KHR(
	VkPhysicalDevice pd,
	VkVideoCodecOperationFlagBitsKHR codecOp,
	VkVideoChromaSubsamplingFlagsKHR chromaSubsampling,
	VkVideoComponentBitDepthFlagsKHR lumaBitDepth,
	VkVideoComponentBitDepthFlagsKHR chromaBitDepth,
	StdVideoH264ProfileIdc h264Profile,
	VkVideoCapabilitiesKHR* caps,
	VkVideoEncodeCapabilitiesKHR* encodeCaps,
	VkVideoEncodeH264CapabilitiesKHR* h264Caps
) {
	if (pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Build H.264 profile info
	VkVideoEncodeH264ProfileInfoKHR h264ProfileInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H264_PROFILE_INFO_KHR,
		.pNext = NULL,
		.stdProfileIdc = h264Profile
	};

	// Build base video profile with H.264 profile chained
	VkVideoProfileInfoKHR profile = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_PROFILE_INFO_KHR,
		.pNext = &h264ProfileInfo,
		.videoCodecOperation = codecOp,
		.chromaSubsampling = chromaSubsampling,
		.lumaBitDepth = lumaBitDepth,
		.chromaBitDepth = chromaBitDepth
	};

	// Chain capabilities: caps -> encodeCaps -> h264Caps
	h264Caps->sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H264_CAPABILITIES_KHR;
	h264Caps->pNext = NULL;

	encodeCaps->sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_CAPABILITIES_KHR;
	encodeCaps->pNext = h264Caps;

	caps->sType = VK_STRUCTURE_TYPE_VIDEO_CAPABILITIES_KHR;
	caps->pNext = encodeCaps;

	VkResult result = pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR(pd, &profile, caps);

	// Store the STD header version from capabilities for later use
	if (result == VK_SUCCESS) {
		g_h264EncodeStdHeaderVersion = caps->stdHeaderVersion;
	}

	return result;
}

static VkResult call_vkCreateVideoSessionKHR(VkDevice device, const VkVideoSessionCreateInfoKHR* info, VkVideoSessionKHR* session) {
	if (pfn_vkCreateVideoSessionKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;
	return pfn_vkCreateVideoSessionKHR(device, info, NULL, session);
}

// H.264 encode specific video session creation with proper pNext chaining
static VkResult call_vkCreateVideoSessionH264KHR(
	VkDevice device,
	uint32_t queueFamilyIndex,
	VkVideoCodecOperationFlagBitsKHR codecOp,
	VkVideoChromaSubsamplingFlagsKHR chromaSubsampling,
	VkVideoComponentBitDepthFlagsKHR lumaBitDepth,
	VkVideoComponentBitDepthFlagsKHR chromaBitDepth,
	StdVideoH264ProfileIdc h264Profile,
	VkFormat pictureFormat,
	VkExtent2D maxCodedExtent,
	VkFormat referencePictureFormat,
	uint32_t maxDpbSlots,
	uint32_t maxActiveReferencePictures,
	VkVideoSessionKHR* session
) {
	if (pfn_vkCreateVideoSessionKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Build H.264 profile info
	VkVideoEncodeH264ProfileInfoKHR h264ProfileInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H264_PROFILE_INFO_KHR,
		.pNext = NULL,
		.stdProfileIdc = h264Profile
	};

	// Build base video profile with H.264 profile chained
	VkVideoProfileInfoKHR profile = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_PROFILE_INFO_KHR,
		.pNext = &h264ProfileInfo,
		.videoCodecOperation = codecOp,
		.chromaSubsampling = chromaSubsampling,
		.lumaBitDepth = lumaBitDepth,
		.chromaBitDepth = chromaBitDepth
	};

	// Use the STD header version from the capabilities query
	// g_h264EncodeStdHeaderVersion must be populated by calling GetVideoCapabilitiesH264KHR first
	VkVideoSessionCreateInfoKHR createInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_SESSION_CREATE_INFO_KHR,
		.pNext = NULL,
		.queueFamilyIndex = queueFamilyIndex,
		.flags = 0,
		.pVideoProfile = &profile,
		.pictureFormat = pictureFormat,
		.maxCodedExtent = maxCodedExtent,
		.referencePictureFormat = referencePictureFormat,
		.maxDpbSlots = maxDpbSlots,
		.maxActiveReferencePictures = maxActiveReferencePictures,
		.pStdHeaderVersion = &g_h264EncodeStdHeaderVersion
	};

	return pfn_vkCreateVideoSessionKHR(device, &createInfo, NULL, session);
}

static void call_vkDestroyVideoSessionKHR(VkDevice device, VkVideoSessionKHR session) {
	if (pfn_vkDestroyVideoSessionKHR != NULL) {
		pfn_vkDestroyVideoSessionKHR(device, session, NULL);
	}
}

static VkResult call_vkGetVideoSessionMemoryRequirementsKHR(VkDevice device, VkVideoSessionKHR session, uint32_t* count, VkVideoSessionMemoryRequirementsKHR* reqs) {
	if (pfn_vkGetVideoSessionMemoryRequirementsKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;
	return pfn_vkGetVideoSessionMemoryRequirementsKHR(device, session, count, reqs);
}

static VkResult call_vkBindVideoSessionMemoryKHR(VkDevice device, VkVideoSessionKHR session, uint32_t count, const VkBindVideoSessionMemoryInfoKHR* bindings) {
	if (pfn_vkBindVideoSessionMemoryKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;
	return pfn_vkBindVideoSessionMemoryKHR(device, session, count, bindings);
}

static VkResult call_vkCreateVideoSessionParametersKHR(VkDevice device, const VkVideoSessionParametersCreateInfoKHR* info, VkVideoSessionParametersKHR* params) {
	if (pfn_vkCreateVideoSessionParametersKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;
	return pfn_vkCreateVideoSessionParametersKHR(device, info, NULL, params);
}

// H.264 encode session parameters with SPS/PPS
static VkResult call_vkCreateVideoSessionParametersH264KHR(
	VkDevice device,
	VkVideoSessionKHR session,
	uint32_t width,
	uint32_t height,
	uint32_t frameRateNum,
	uint32_t frameRateDen,
	StdVideoH264ProfileIdc profileIdc,
	StdVideoH264LevelIdc levelIdc,
	VkVideoSessionParametersKHR* params
) {
	if (pfn_vkCreateVideoSessionParametersKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Create SPS
	StdVideoH264SpsVuiFlags vuiFlags = {0};
	vuiFlags.timing_info_present_flag = 1;
	vuiFlags.fixed_frame_rate_flag = 1;

	StdVideoH264HrdParameters hrdParams = {0};

	StdVideoH264SequenceParameterSetVui vui = {
		.flags = vuiFlags,
		.aspect_ratio_idc = STD_VIDEO_H264_ASPECT_RATIO_IDC_SQUARE,
		.sar_width = 1,
		.sar_height = 1,
		.video_format = 5, // Unspecified
		.colour_primaries = 2, // Unspecified
		.transfer_characteristics = 2, // Unspecified
		.matrix_coefficients = 2, // Unspecified
		.num_units_in_tick = frameRateDen,
		.time_scale = frameRateNum * 2, // H.264 counts fields
		.max_num_reorder_frames = 0,
		.max_dec_frame_buffering = 1,
		.chroma_sample_loc_type_top_field = 0,
		.chroma_sample_loc_type_bottom_field = 0,
		.pHrdParameters = &hrdParams
	};

	StdVideoH264SpsFlags spsFlags = {0};
	spsFlags.direct_8x8_inference_flag = 1;
	spsFlags.frame_mbs_only_flag = 1;

	StdVideoH264SequenceParameterSet sps = {
		.flags = spsFlags,
		.profile_idc = profileIdc,
		.level_idc = levelIdc,
		.chroma_format_idc = STD_VIDEO_H264_CHROMA_FORMAT_IDC_420,
		.seq_parameter_set_id = 0,
		.bit_depth_luma_minus8 = 0,
		.bit_depth_chroma_minus8 = 0,
		.log2_max_frame_num_minus4 = 4, // max_frame_num = 256
		.pic_order_cnt_type = 2, // No B-frames, POC derived from frame_num
		.offset_for_non_ref_pic = 0,
		.offset_for_top_to_bottom_field = 0,
		.log2_max_pic_order_cnt_lsb_minus4 = 0,
		.num_ref_frames_in_pic_order_cnt_cycle = 0,
		.max_num_ref_frames = 1,
		.pic_width_in_mbs_minus1 = (width + 15) / 16 - 1,
		.pic_height_in_map_units_minus1 = (height + 15) / 16 - 1,
		.frame_crop_left_offset = 0,
		.frame_crop_right_offset = ((width + 15) / 16 * 16 - width) / 2,
		.frame_crop_top_offset = 0,
		.frame_crop_bottom_offset = ((height + 15) / 16 * 16 - height) / 2,
		.pOffsetForRefFrame = NULL,
		.pScalingLists = NULL,
		.pSequenceParameterSetVui = &vui
	};

	// Create PPS
	StdVideoH264PpsFlags ppsFlags = {0};
	ppsFlags.entropy_coding_mode_flag = 1; // CABAC
	ppsFlags.deblocking_filter_control_present_flag = 1;

	StdVideoH264PictureParameterSet pps = {
		.flags = ppsFlags,
		.seq_parameter_set_id = 0,
		.pic_parameter_set_id = 0,
		.num_ref_idx_l0_default_active_minus1 = 0,
		.num_ref_idx_l1_default_active_minus1 = 0,
		.weighted_bipred_idc = STD_VIDEO_H264_WEIGHTED_BIPRED_IDC_DEFAULT,
		.pic_init_qp_minus26 = 0,
		.pic_init_qs_minus26 = 0,
		.chroma_qp_index_offset = 0,
		.second_chroma_qp_index_offset = 0,
		.pScalingLists = NULL
	};

	// H.264 session parameters add info
	VkVideoEncodeH264SessionParametersAddInfoKHR h264AddInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H264_SESSION_PARAMETERS_ADD_INFO_KHR,
		.pNext = NULL,
		.stdSPSCount = 1,
		.pStdSPSs = &sps,
		.stdPPSCount = 1,
		.pStdPPSs = &pps
	};

	// H.264 session parameters create info
	VkVideoEncodeH264SessionParametersCreateInfoKHR h264ParamsInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H264_SESSION_PARAMETERS_CREATE_INFO_KHR,
		.pNext = NULL,
		.maxStdSPSCount = 1,
		.maxStdPPSCount = 1,
		.pParametersAddInfo = &h264AddInfo
	};

	// Main session parameters create info with H.264 chained
	VkVideoSessionParametersCreateInfoKHR createInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_SESSION_PARAMETERS_CREATE_INFO_KHR,
		.pNext = &h264ParamsInfo,
		.flags = 0,
		.videoSessionParametersTemplate = VK_NULL_HANDLE,
		.videoSession = session
	};

	return pfn_vkCreateVideoSessionParametersKHR(device, &createInfo, NULL, params);
}

static void call_vkDestroyVideoSessionParametersKHR(VkDevice device, VkVideoSessionParametersKHR params) {
	if (pfn_vkDestroyVideoSessionParametersKHR != NULL) {
		pfn_vkDestroyVideoSessionParametersKHR(device, params, NULL);
	}
}

static void call_vkCmdBeginVideoCodingKHR(VkCommandBuffer cb, const VkVideoBeginCodingInfoKHR* info) {
	if (pfn_vkCmdBeginVideoCodingKHR != NULL) {
		pfn_vkCmdBeginVideoCodingKHR(cb, info);
	}
}

static void call_vkCmdEndVideoCodingKHR(VkCommandBuffer cb, const VkVideoEndCodingInfoKHR* info) {
	if (pfn_vkCmdEndVideoCodingKHR != NULL) {
		pfn_vkCmdEndVideoCodingKHR(cb, info);
	}
}

static void call_vkCmdControlVideoCodingKHR(VkCommandBuffer cb, const VkVideoCodingControlInfoKHR* info) {
	if (pfn_vkCmdControlVideoCodingKHR != NULL) {
		pfn_vkCmdControlVideoCodingKHR(cb, info);
	}
}

static void call_vkCmdDecodeVideoKHR(VkCommandBuffer cb, const VkVideoDecodeInfoKHR* info) {
	if (pfn_vkCmdDecodeVideoKHR != NULL) {
		pfn_vkCmdDecodeVideoKHR(cb, info);
	}
}

static void call_vkCmdEncodeVideoKHR(VkCommandBuffer cb, const VkVideoEncodeInfoKHR* info) {
	if (pfn_vkCmdEncodeVideoKHR != NULL) {
		pfn_vkCmdEncodeVideoKHR(cb, info);
	}
}

// Global to store the queried H.265 STD header version
static VkExtensionProperties g_h265EncodeStdHeaderVersion = {0};

// H.265 encode specific capabilities query with proper pNext chaining
static VkResult call_vkGetPhysicalDeviceVideoCapabilitiesH265KHR(
	VkPhysicalDevice pd,
	VkVideoCodecOperationFlagBitsKHR codecOp,
	VkVideoChromaSubsamplingFlagsKHR chromaSubsampling,
	VkVideoComponentBitDepthFlagsKHR lumaBitDepth,
	VkVideoComponentBitDepthFlagsKHR chromaBitDepth,
	StdVideoH265ProfileIdc h265Profile,
	VkVideoCapabilitiesKHR* caps,
	VkVideoEncodeCapabilitiesKHR* encodeCaps,
	VkVideoEncodeH265CapabilitiesKHR* h265Caps
) {
	if (pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Build H.265 profile info
	VkVideoEncodeH265ProfileInfoKHR h265ProfileInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_PROFILE_INFO_KHR,
		.pNext = NULL,
		.stdProfileIdc = h265Profile
	};

	// Build base video profile with H.265 profile chained
	VkVideoProfileInfoKHR profile = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_PROFILE_INFO_KHR,
		.pNext = &h265ProfileInfo,
		.videoCodecOperation = codecOp,
		.chromaSubsampling = chromaSubsampling,
		.lumaBitDepth = lumaBitDepth,
		.chromaBitDepth = chromaBitDepth
	};

	// Chain capabilities: caps -> encodeCaps -> h265Caps
	h265Caps->sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_CAPABILITIES_KHR;
	h265Caps->pNext = NULL;

	encodeCaps->sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_CAPABILITIES_KHR;
	encodeCaps->pNext = h265Caps;

	caps->sType = VK_STRUCTURE_TYPE_VIDEO_CAPABILITIES_KHR;
	caps->pNext = encodeCaps;

	VkResult result = pfn_vkGetPhysicalDeviceVideoCapabilitiesKHR(pd, &profile, caps);

	// Store the STD header version from capabilities for later use
	if (result == VK_SUCCESS) {
		g_h265EncodeStdHeaderVersion = caps->stdHeaderVersion;
	}

	return result;
}

// H.265 encode specific video session creation with proper pNext chaining
static VkResult call_vkCreateVideoSessionH265KHR(
	VkDevice device,
	uint32_t queueFamilyIndex,
	VkVideoCodecOperationFlagBitsKHR codecOp,
	VkVideoChromaSubsamplingFlagsKHR chromaSubsampling,
	VkVideoComponentBitDepthFlagsKHR lumaBitDepth,
	VkVideoComponentBitDepthFlagsKHR chromaBitDepth,
	StdVideoH265ProfileIdc h265Profile,
	VkFormat pictureFormat,
	VkExtent2D maxCodedExtent,
	VkFormat referencePictureFormat,
	uint32_t maxDpbSlots,
	uint32_t maxActiveReferencePictures,
	VkVideoSessionKHR* session
) {
	if (pfn_vkCreateVideoSessionKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Build H.265 profile info
	VkVideoEncodeH265ProfileInfoKHR h265ProfileInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_PROFILE_INFO_KHR,
		.pNext = NULL,
		.stdProfileIdc = h265Profile
	};

	// Build base video profile with H.265 profile chained
	VkVideoProfileInfoKHR profile = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_PROFILE_INFO_KHR,
		.pNext = &h265ProfileInfo,
		.videoCodecOperation = codecOp,
		.chromaSubsampling = chromaSubsampling,
		.lumaBitDepth = lumaBitDepth,
		.chromaBitDepth = chromaBitDepth
	};

	// Use the STD header version from the capabilities query
	VkVideoSessionCreateInfoKHR createInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_SESSION_CREATE_INFO_KHR,
		.pNext = NULL,
		.queueFamilyIndex = queueFamilyIndex,
		.flags = 0,
		.pVideoProfile = &profile,
		.pictureFormat = pictureFormat,
		.maxCodedExtent = maxCodedExtent,
		.referencePictureFormat = referencePictureFormat,
		.maxDpbSlots = maxDpbSlots,
		.maxActiveReferencePictures = maxActiveReferencePictures,
		.pStdHeaderVersion = &g_h265EncodeStdHeaderVersion
	};

	return pfn_vkCreateVideoSessionKHR(device, &createInfo, NULL, session);
}

// H.265 encode session parameters with VPS/SPS/PPS
static VkResult call_vkCreateVideoSessionParametersH265KHR(
	VkDevice device,
	VkVideoSessionKHR session,
	uint32_t width,
	uint32_t height,
	uint32_t frameRateNum,
	uint32_t frameRateDen,
	StdVideoH265ProfileIdc profileIdc,
	StdVideoH265LevelIdc levelIdc,
	VkVideoSessionParametersKHR* params
) {
	if (pfn_vkCreateVideoSessionParametersKHR == NULL) return VK_ERROR_EXTENSION_NOT_PRESENT;

	// Create VPS (minimal)
	StdVideoH265VideoParameterSet vps = {
		.flags = {
			.vps_temporal_id_nesting_flag = 1,
			.vps_sub_layer_ordering_info_present_flag = 1
		},
		.vps_video_parameter_set_id = 0,
		.vps_max_sub_layers_minus1 = 0,
		.vps_num_units_in_tick = frameRateDen,
		.vps_time_scale = frameRateNum,
		.vps_num_ticks_poc_diff_one_minus1 = 0,
		.pDecPicBufMgr = NULL,
		.pHrdParameters = NULL,
		.pProfileTierLevel = NULL
	};

	// Create SPS
	StdVideoH265SpsFlags spsFlags = {0};
	spsFlags.sps_temporal_id_nesting_flag = 1;
	spsFlags.sps_sub_layer_ordering_info_present_flag = 1;
	spsFlags.sample_adaptive_offset_enabled_flag = 0;
	spsFlags.pcm_enabled_flag = 1;
	spsFlags.pcm_loop_filter_disabled_flag = 1;

	StdVideoH265SequenceParameterSet sps = {
		.flags = spsFlags,
		.chroma_format_idc = STD_VIDEO_H265_CHROMA_FORMAT_IDC_420,
		.pic_width_in_luma_samples = width,
		.pic_height_in_luma_samples = height,
		.sps_video_parameter_set_id = 0,
		.sps_max_sub_layers_minus1 = 0,
		.sps_seq_parameter_set_id = 0,
		.bit_depth_luma_minus8 = 0,
		.bit_depth_chroma_minus8 = 0,
		.log2_max_pic_order_cnt_lsb_minus4 = 4,
		.log2_min_luma_coding_block_size_minus3 = 0,
		.log2_diff_max_min_luma_coding_block_size = 3,
		.log2_min_luma_transform_block_size_minus2 = 0,
		.log2_diff_max_min_luma_transform_block_size = 3,
		.max_transform_hierarchy_depth_inter = 0,
		.max_transform_hierarchy_depth_intra = 0,
		.num_short_term_ref_pic_sets = 0,
		.num_long_term_ref_pics_sps = 0,
		.pcm_sample_bit_depth_luma_minus1 = 7,
		.pcm_sample_bit_depth_chroma_minus1 = 7,
		.log2_min_pcm_luma_coding_block_size_minus3 = 0,
		.log2_diff_max_min_pcm_luma_coding_block_size = 2,
		.pProfileTierLevel = NULL,
		.pDecPicBufMgr = NULL,
		.pScalingLists = NULL,
		.pShortTermRefPicSet = NULL,
		.pLongTermRefPicsSps = NULL,
		.pSequenceParameterSetVui = NULL,
		.pPredictorPaletteEntries = NULL
	};

	// Create PPS
	StdVideoH265PpsFlags ppsFlags = {0};
	ppsFlags.cabac_init_present_flag = 0;
	ppsFlags.deblocking_filter_control_present_flag = 1;
	ppsFlags.pps_deblocking_filter_disabled_flag = 1;

	StdVideoH265PictureParameterSet pps = {
		.flags = ppsFlags,
		.pps_pic_parameter_set_id = 0,
		.pps_seq_parameter_set_id = 0,
		.sps_video_parameter_set_id = 0,
		.num_extra_slice_header_bits = 0,
		.num_ref_idx_l0_default_active_minus1 = 0,
		.num_ref_idx_l1_default_active_minus1 = 0,
		.init_qp_minus26 = 0,
		.diff_cu_qp_delta_depth = 0,
		.pps_cb_qp_offset = 0,
		.pps_cr_qp_offset = 0,
		.pps_beta_offset_div2 = 0,
		.pps_tc_offset_div2 = 0,
		.log2_parallel_merge_level_minus2 = 0,
		.pScalingLists = NULL,
		.pPredictorPaletteEntries = NULL
	};

	// H.265 session parameters add info
	VkVideoEncodeH265SessionParametersAddInfoKHR h265AddInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_SESSION_PARAMETERS_ADD_INFO_KHR,
		.pNext = NULL,
		.stdVPSCount = 1,
		.pStdVPSs = &vps,
		.stdSPSCount = 1,
		.pStdSPSs = &sps,
		.stdPPSCount = 1,
		.pStdPPSs = &pps
	};

	// H.265 session parameters create info
	VkVideoEncodeH265SessionParametersCreateInfoKHR h265ParamsInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_SESSION_PARAMETERS_CREATE_INFO_KHR,
		.pNext = NULL,
		.maxStdVPSCount = 1,
		.maxStdSPSCount = 1,
		.maxStdPPSCount = 1,
		.pParametersAddInfo = &h265AddInfo
	};

	// Main session parameters create info with H.265 chained
	VkVideoSessionParametersCreateInfoKHR createInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_SESSION_PARAMETERS_CREATE_INFO_KHR,
		.pNext = &h265ParamsInfo,
		.flags = 0,
		.videoSessionParametersTemplate = VK_NULL_HANDLE,
		.videoSession = session
	};

	return pfn_vkCreateVideoSessionParametersKHR(device, &createInfo, NULL, params);
}

// Check if H.265 encode is supported
static int checkH265EncodeSupport(VkPhysicalDevice pd) {
	uint32_t count = 0;
	vkGetPhysicalDeviceQueueFamilyProperties2(pd, &count, NULL);
	if (count == 0) return 0;

	VkQueueFamilyProperties2* props = (VkQueueFamilyProperties2*)malloc(count * sizeof(VkQueueFamilyProperties2));
	VkQueueFamilyVideoPropertiesKHR* videoProps = (VkQueueFamilyVideoPropertiesKHR*)malloc(count * sizeof(VkQueueFamilyVideoPropertiesKHR));

	for (uint32_t i = 0; i < count; i++) {
		props[i].sType = VK_STRUCTURE_TYPE_QUEUE_FAMILY_PROPERTIES_2;
		props[i].pNext = &videoProps[i];
		videoProps[i].sType = VK_STRUCTURE_TYPE_QUEUE_FAMILY_VIDEO_PROPERTIES_KHR;
		videoProps[i].pNext = NULL;
	}

	vkGetPhysicalDeviceQueueFamilyProperties2(pd, &count, props);

	int supported = 0;
	for (uint32_t i = 0; i < count; i++) {
		if (videoProps[i].videoCodecOperations & VK_VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR) {
			supported = 1;
			break;
		}
	}

	free(props);
	free(videoProps);
	return supported;
}

// H.265 encode a single frame with proper pNext chaining
static void call_vkCmdEncodeVideoH265KHR(
	VkCommandBuffer cb,
	VkBuffer dstBuffer,
	VkDeviceSize dstBufferOffset,
	VkDeviceSize dstBufferRange,
	VkImageView srcImageView,
	uint32_t width,
	uint32_t height,
	uint32_t frameNum,
	int isIDR
) {
	if (pfn_vkCmdEncodeVideoKHR == NULL) return;

	// Slice segment header
	StdVideoEncodeH265SliceSegmentHeaderFlags sliceFlags = {0};
	sliceFlags.first_slice_segment_in_pic_flag = 1;
	sliceFlags.slice_sao_luma_flag = 0;
	sliceFlags.slice_sao_chroma_flag = 0;

	StdVideoEncodeH265SliceSegmentHeader sliceHeader = {
		.flags = sliceFlags,
		.slice_type = isIDR ? STD_VIDEO_H265_SLICE_TYPE_I : STD_VIDEO_H265_SLICE_TYPE_I,
		.slice_segment_address = 0,
		.collocated_ref_idx = 0,
		.MaxNumMergeCand = 5,
		.slice_cb_qp_offset = 0,
		.slice_cr_qp_offset = 0,
		.slice_beta_offset_div2 = 0,
		.slice_tc_offset_div2 = 0,
		.slice_act_y_qp_offset = 0,
		.slice_act_cb_qp_offset = 0,
		.slice_act_cr_qp_offset = 0,
		.slice_qp_delta = 0,
		.pWeightTable = NULL
	};

	// NALU slice segment info
	VkVideoEncodeH265NaluSliceSegmentInfoKHR sliceInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_NALU_SLICE_SEGMENT_INFO_KHR,
		.pNext = NULL,
		.constantQp = 26, // Medium quality
		.pStdSliceSegmentHeader = &sliceHeader
	};

	// Picture info flags
	StdVideoEncodeH265PictureInfoFlags picFlags = {0};
	picFlags.is_reference = 0; // I-frames only, no references
	picFlags.IrapPicFlag = isIDR ? 1 : 0;
	picFlags.used_for_long_term_reference = 0;
	picFlags.discardable_flag = 0;
	picFlags.cross_layer_bla_flag = 0;

	// Picture info
	StdVideoEncodeH265PictureInfo picInfo = {
		.flags = picFlags,
		.pic_type = isIDR ? STD_VIDEO_H265_PICTURE_TYPE_IDR : STD_VIDEO_H265_PICTURE_TYPE_I,
		.sps_video_parameter_set_id = 0,
		.pps_seq_parameter_set_id = 0,
		.pps_pic_parameter_set_id = 0,
		.short_term_ref_pic_set_idx = 0,
		.PicOrderCntVal = frameNum,
		.TemporalId = 0,
		.pRefLists = NULL
	};

	// H.265 picture info
	VkVideoEncodeH265PictureInfoKHR h265PicInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_H265_PICTURE_INFO_KHR,
		.pNext = NULL,
		.naluSliceSegmentEntryCount = 1,
		.pNaluSliceSegmentEntries = &sliceInfo,
		.pStdPictureInfo = &picInfo
	};

	// Source picture resource
	VkVideoPictureResourceInfoKHR srcResource = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_PICTURE_RESOURCE_INFO_KHR,
		.pNext = NULL,
		.codedOffset = {0, 0},
		.codedExtent = {width, height},
		.baseArrayLayer = 0,
		.imageViewBinding = srcImageView
	};

	// Encode info
	VkVideoEncodeInfoKHR encodeInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_ENCODE_INFO_KHR,
		.pNext = &h265PicInfo,
		.flags = 0,
		.dstBuffer = dstBuffer,
		.dstBufferOffset = dstBufferOffset,
		.dstBufferRange = dstBufferRange,
		.srcPictureResource = srcResource,
		.pSetupReferenceSlot = NULL,
		.referenceSlotCount = 0,
		.pReferenceSlots = NULL,
		.precedingExternallyEncodedBytes = 0
	};

	pfn_vkCmdEncodeVideoKHR(cb, &encodeInfo);
}

// Begin video coding for H.265 encode
static void call_vkCmdBeginVideoCodingH265KHR(
	VkCommandBuffer cb,
	VkVideoSessionKHR session,
	VkVideoSessionParametersKHR params,
	VkImageView dpbImageView,
	uint32_t width,
	uint32_t height
) {
	if (pfn_vkCmdBeginVideoCodingKHR == NULL) return;

	VkVideoBeginCodingInfoKHR beginInfo = {
		.sType = VK_STRUCTURE_TYPE_VIDEO_BEGIN_CODING_INFO_KHR,
		.pNext = NULL,
		.flags = 0,
		.videoSession = session,
		.videoSessionParameters = params,
		.referenceSlotCount = 0,
		.pReferenceSlots = NULL
	};

	pfn_vkCmdBeginVideoCodingKHR(cb, &beginInfo);
}

*/
import "C"
import (
	"fmt"
	"unsafe"
)

// VideoSessionKHR is a handle to a Vulkan video session
type VideoSessionKHR struct {
	handle C.VkVideoSessionKHR
}

// VideoSessionParametersKHR is a handle to video session parameters
type VideoSessionParametersKHR struct {
	handle C.VkVideoSessionParametersKHR
}

// VideoCodecOperationFlagsKHR specifies the video codec operation
type VideoCodecOperationFlagsKHR uint32

const (
	VIDEO_CODEC_OPERATION_NONE_KHR            VideoCodecOperationFlagsKHR = 0
	VIDEO_CODEC_OPERATION_ENCODE_H264_BIT_KHR VideoCodecOperationFlagsKHR = 0x00010000
	VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR VideoCodecOperationFlagsKHR = 0x00020000
	VIDEO_CODEC_OPERATION_DECODE_H264_BIT_KHR VideoCodecOperationFlagsKHR = 0x00000001
	VIDEO_CODEC_OPERATION_DECODE_H265_BIT_KHR VideoCodecOperationFlagsKHR = 0x00000002
	VIDEO_CODEC_OPERATION_DECODE_AV1_BIT_KHR  VideoCodecOperationFlagsKHR = 0x00000004
)

// VideoChromaSubsamplingFlagsKHR specifies chroma subsampling
type VideoChromaSubsamplingFlagsKHR uint32

const (
	VIDEO_CHROMA_SUBSAMPLING_MONOCHROME_BIT_KHR VideoChromaSubsamplingFlagsKHR = 0x00000001
	VIDEO_CHROMA_SUBSAMPLING_420_BIT_KHR        VideoChromaSubsamplingFlagsKHR = 0x00000002
	VIDEO_CHROMA_SUBSAMPLING_422_BIT_KHR        VideoChromaSubsamplingFlagsKHR = 0x00000004
	VIDEO_CHROMA_SUBSAMPLING_444_BIT_KHR        VideoChromaSubsamplingFlagsKHR = 0x00000008
)

// VideoComponentBitDepthFlagsKHR specifies bit depth
type VideoComponentBitDepthFlagsKHR uint32

const (
	VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR  VideoComponentBitDepthFlagsKHR = 0x00000001
	VIDEO_COMPONENT_BIT_DEPTH_10_BIT_KHR VideoComponentBitDepthFlagsKHR = 0x00000004
	VIDEO_COMPONENT_BIT_DEPTH_12_BIT_KHR VideoComponentBitDepthFlagsKHR = 0x00000010
)

// VideoCapabilityFlagsKHR specifies video capabilities
type VideoCapabilityFlagsKHR uint32

const (
	VIDEO_CAPABILITY_PROTECTED_CONTENT_BIT_KHR         VideoCapabilityFlagsKHR = 0x00000001
	VIDEO_CAPABILITY_SEPARATE_REFERENCE_IMAGES_BIT_KHR VideoCapabilityFlagsKHR = 0x00000002
)

// VideoSessionCreateFlagsKHR specifies video session creation flags
type VideoSessionCreateFlagsKHR uint32

const (
	VIDEO_SESSION_CREATE_PROTECTED_CONTENT_BIT_KHR            VideoSessionCreateFlagsKHR = 0x00000001
	VIDEO_SESSION_CREATE_ALLOW_ENCODE_PARAMETER_OPTIMIZATIONS VideoSessionCreateFlagsKHR = 0x00000002
	VIDEO_SESSION_CREATE_INLINE_QUERIES_BIT_KHR               VideoSessionCreateFlagsKHR = 0x00000004
)

// VideoCodingControlFlagsKHR specifies coding control flags
type VideoCodingControlFlagsKHR uint32

const (
	VIDEO_CODING_CONTROL_RESET_BIT_KHR                VideoCodingControlFlagsKHR = 0x00000001
	VIDEO_CODING_CONTROL_ENCODE_RATE_CONTROL_BIT_KHR  VideoCodingControlFlagsKHR = 0x00000002
	VIDEO_CODING_CONTROL_ENCODE_QUALITY_LEVEL_BIT_KHR VideoCodingControlFlagsKHR = 0x00000004
)

// VideoEncodeRateControlModeFlagsKHR specifies rate control mode
type VideoEncodeRateControlModeFlagsKHR uint32

const (
	VIDEO_ENCODE_RATE_CONTROL_MODE_DEFAULT_KHR      VideoEncodeRateControlModeFlagsKHR = 0
	VIDEO_ENCODE_RATE_CONTROL_MODE_DISABLED_BIT_KHR VideoEncodeRateControlModeFlagsKHR = 0x00000001
	VIDEO_ENCODE_RATE_CONTROL_MODE_CBR_BIT_KHR      VideoEncodeRateControlModeFlagsKHR = 0x00000002
	VIDEO_ENCODE_RATE_CONTROL_MODE_VBR_BIT_KHR      VideoEncodeRateControlModeFlagsKHR = 0x00000004
)

// VideoProfileInfoKHR describes a video profile
type VideoProfileInfoKHR struct {
	SType               StructureType
	PNext               unsafe.Pointer
	VideoCodecOperation VideoCodecOperationFlagsKHR
	ChromaSubsampling   VideoChromaSubsamplingFlagsKHR
	LumaBitDepth        VideoComponentBitDepthFlagsKHR
	ChromaBitDepth      VideoComponentBitDepthFlagsKHR
}

// VideoCapabilitiesKHR describes video capabilities
type VideoCapabilitiesKHR struct {
	SType                             StructureType
	PNext                             unsafe.Pointer
	Flags                             VideoCapabilityFlagsKHR
	MinBitstreamBufferOffsetAlignment uint64
	MinBitstreamBufferSizeAlignment   uint64
	PictureAccessGranularity          Extent2D
	MinCodedExtent                    Extent2D
	MaxCodedExtent                    Extent2D
	MaxDpbSlots                       uint32
	MaxActiveReferencePictures        uint32
}

// VideoSessionCreateInfoKHR specifies parameters for video session creation
type VideoSessionCreateInfoKHR struct {
	SType                      StructureType
	PNext                      unsafe.Pointer
	QueueFamilyIndex           uint32
	Flags                      VideoSessionCreateFlagsKHR
	PVideoProfile              *VideoProfileInfoKHR
	PictureFormat              Format
	MaxCodedExtent             Extent2D
	ReferencePictureFormat     Format
	MaxDpbSlots                uint32
	MaxActiveReferencePictures uint32
}

// VideoSessionMemoryRequirementsKHR describes video session memory requirements
type VideoSessionMemoryRequirementsKHR struct {
	SType              StructureType
	PNext              unsafe.Pointer
	MemoryBindIndex    uint32
	MemoryRequirements MemoryRequirements
}

// BindVideoSessionMemoryInfoKHR specifies memory binding for video session
type BindVideoSessionMemoryInfoKHR struct {
	SType           StructureType
	PNext           unsafe.Pointer
	MemoryBindIndex uint32
	Memory          DeviceMemory
	MemoryOffset    uint64
	MemorySize      uint64
}

// VideoPictureResourceInfoKHR describes a video picture resource
type VideoPictureResourceInfoKHR struct {
	SType            StructureType
	PNext            unsafe.Pointer
	CodedOffset      Offset2D
	CodedExtent      Extent2D
	BaseArrayLayer   uint32
	ImageViewBinding ImageView
}

// VideoReferenceSlotInfoKHR describes a reference picture slot
type VideoReferenceSlotInfoKHR struct {
	SType            StructureType
	PNext            unsafe.Pointer
	SlotIndex        int32
	PPictureResource *VideoPictureResourceInfoKHR
}

// VideoBeginCodingInfoKHR specifies parameters for beginning video coding
type VideoBeginCodingInfoKHR struct {
	SType                  StructureType
	PNext                  unsafe.Pointer
	Flags                  uint32
	VideoSession           VideoSessionKHR
	VideoSessionParameters VideoSessionParametersKHR
	ReferenceSlotCount     uint32
	PReferenceSlots        []VideoReferenceSlotInfoKHR
}

// VideoEndCodingInfoKHR specifies parameters for ending video coding
type VideoEndCodingInfoKHR struct {
	SType StructureType
	PNext unsafe.Pointer
	Flags uint32
}

// VideoCodingControlInfoKHR specifies coding control parameters
type VideoCodingControlInfoKHR struct {
	SType StructureType
	PNext unsafe.Pointer
	Flags VideoCodingControlFlagsKHR
}

// VideoEncodeInfoKHR specifies parameters for encoding
type VideoEncodeInfoKHR struct {
	SType                           StructureType
	PNext                           unsafe.Pointer
	Flags                           uint32
	DstBuffer                       Buffer
	DstBufferOffset                 uint64
	DstBufferRange                  uint64
	SrcPictureResource              VideoPictureResourceInfoKHR
	PSetupReferenceSlot             *VideoReferenceSlotInfoKHR
	ReferenceSlotCount              uint32
	PReferenceSlots                 []VideoReferenceSlotInfoKHR
	PrecedingExternallyEncodedBytes uint32
}

// VideoDecodeInfoKHR specifies parameters for decoding
type VideoDecodeInfoKHR struct {
	SType               StructureType
	PNext               unsafe.Pointer
	Flags               uint32
	SrcBuffer           Buffer
	SrcBufferOffset     uint64
	SrcBufferRange      uint64
	DstPictureResource  VideoPictureResourceInfoKHR
	PSetupReferenceSlot *VideoReferenceSlotInfoKHR
	ReferenceSlotCount  uint32
	PReferenceSlots     []VideoReferenceSlotInfoKHR
}

// VideoSessionParametersCreateInfoKHR specifies video session parameters creation
type VideoSessionParametersCreateInfoKHR struct {
	SType                          StructureType
	PNext                          unsafe.Pointer
	Flags                          uint32
	VideoSessionParametersTemplate VideoSessionParametersKHR
	VideoSession                   VideoSessionKHR
}

// QueueFamilyVideoPropertiesKHR describes video properties of a queue family
type QueueFamilyVideoPropertiesKHR struct {
	SType                StructureType
	PNext                unsafe.Pointer
	VideoCodecOperations VideoCodecOperationFlagsKHR
}

// H.264 Profile IDC values
const (
	STD_VIDEO_H264_PROFILE_IDC_BASELINE            uint32 = 66
	STD_VIDEO_H264_PROFILE_IDC_MAIN                uint32 = 77
	STD_VIDEO_H264_PROFILE_IDC_HIGH                uint32 = 100
	STD_VIDEO_H264_PROFILE_IDC_HIGH_444_PREDICTIVE uint32 = 244
)

// H.264 Level IDC values (StdVideoH264LevelIdc enum)
const (
	STD_VIDEO_H264_LEVEL_IDC_1_0 uint32 = 0
	STD_VIDEO_H264_LEVEL_IDC_1_1 uint32 = 1
	STD_VIDEO_H264_LEVEL_IDC_1_2 uint32 = 2
	STD_VIDEO_H264_LEVEL_IDC_1_3 uint32 = 3
	STD_VIDEO_H264_LEVEL_IDC_2_0 uint32 = 4
	STD_VIDEO_H264_LEVEL_IDC_2_1 uint32 = 5
	STD_VIDEO_H264_LEVEL_IDC_2_2 uint32 = 6
	STD_VIDEO_H264_LEVEL_IDC_3_0 uint32 = 7
	STD_VIDEO_H264_LEVEL_IDC_3_1 uint32 = 8
	STD_VIDEO_H264_LEVEL_IDC_3_2 uint32 = 9
	STD_VIDEO_H264_LEVEL_IDC_4_0 uint32 = 10
	STD_VIDEO_H264_LEVEL_IDC_4_1 uint32 = 11
	STD_VIDEO_H264_LEVEL_IDC_4_2 uint32 = 12
	STD_VIDEO_H264_LEVEL_IDC_5_0 uint32 = 13
	STD_VIDEO_H264_LEVEL_IDC_5_1 uint32 = 14
	STD_VIDEO_H264_LEVEL_IDC_5_2 uint32 = 15
	STD_VIDEO_H264_LEVEL_IDC_6_0 uint32 = 16
	STD_VIDEO_H264_LEVEL_IDC_6_1 uint32 = 17
	STD_VIDEO_H264_LEVEL_IDC_6_2 uint32 = 18
)

// ---- Extension Loading ----

// LoadVideoExtensionsInstance loads instance-level video extension functions
func LoadVideoExtensionsInstance(instance Instance) bool {
	return C.loadVideoFunctionsInstance(instance.handle) != 0
}

// LoadVideoExtensionsDevice loads device-level video extension functions
func LoadVideoExtensionsDevice(device Device) bool {
	return C.loadVideoFunctionsDevice(device.handle) != 0
}

// ---- Device Functions ----

// GetVideoCapabilitiesKHR queries video decode or encode capabilities
func (pd PhysicalDevice) GetVideoCapabilitiesKHR(profile *VideoProfileInfoKHR, capabilities *VideoCapabilitiesKHR) error {
	cProfile := C.VkVideoProfileInfoKHR{
		sType:               C.VkStructureType(VIDEO_PROFILE_INFO_KHR),
		videoCodecOperation: C.VkVideoCodecOperationFlagBitsKHR(profile.VideoCodecOperation),
		chromaSubsampling:   C.VkVideoChromaSubsamplingFlagsKHR(profile.ChromaSubsampling),
		lumaBitDepth:        C.VkVideoComponentBitDepthFlagsKHR(profile.LumaBitDepth),
		chromaBitDepth:      C.VkVideoComponentBitDepthFlagsKHR(profile.ChromaBitDepth),
	}

	var cCaps C.VkVideoCapabilitiesKHR
	cCaps.sType = C.VkStructureType(VIDEO_CAPABILITIES_KHR)

	result := C.call_vkGetPhysicalDeviceVideoCapabilitiesKHR(
		pd.handle,
		&cProfile,
		&cCaps,
	)

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	capabilities.Flags = VideoCapabilityFlagsKHR(cCaps.flags)
	capabilities.MinBitstreamBufferOffsetAlignment = uint64(cCaps.minBitstreamBufferOffsetAlignment)
	capabilities.MinBitstreamBufferSizeAlignment = uint64(cCaps.minBitstreamBufferSizeAlignment)
	capabilities.PictureAccessGranularity = Extent2D{
		Width:  uint32(cCaps.pictureAccessGranularity.width),
		Height: uint32(cCaps.pictureAccessGranularity.height),
	}
	capabilities.MinCodedExtent = Extent2D{
		Width:  uint32(cCaps.minCodedExtent.width),
		Height: uint32(cCaps.minCodedExtent.height),
	}
	capabilities.MaxCodedExtent = Extent2D{
		Width:  uint32(cCaps.maxCodedExtent.width),
		Height: uint32(cCaps.maxCodedExtent.height),
	}
	capabilities.MaxDpbSlots = uint32(cCaps.maxDpbSlots)
	capabilities.MaxActiveReferencePictures = uint32(cCaps.maxActiveReferencePictures)

	return nil
}

// VideoEncodeCapabilitiesKHR describes video encode capabilities
type VideoEncodeCapabilitiesKHR struct {
	Flags                        uint32
	RateControlModes             uint32
	MaxRateControlLayers         uint32
	MaxBitrate                   uint64
	MaxQualityLevels             uint32
	EncodeInputPictureGranularity Extent2D
	SupportedEncodeFeedbackFlags uint32
}

// VideoEncodeH264CapabilitiesKHR describes H.264 encode capabilities
type VideoEncodeH264CapabilitiesKHR struct {
	Flags                        uint32
	MaxLevelIdc                  uint32
	MaxSliceCount                uint32
	MaxPPictureL0ReferenceCount  uint32
	MaxBPictureL0ReferenceCount  uint32
	MaxL1ReferenceCount          uint32
	MaxTemporalLayerCount        uint32
	ExpectDyadicTemporalLayerPattern bool
	MinQp                        int32
	MaxQp                        int32
	PrefersGopRemainingFrames    bool
	RequiresGopRemainingFrames   bool
}

// GetVideoCapabilitiesH264KHR queries H.264 encode capabilities with proper pNext chaining
func (pd PhysicalDevice) GetVideoCapabilitiesH264KHR(h264ProfileIdc uint32, caps *VideoCapabilitiesKHR, encodeCaps *VideoEncodeCapabilitiesKHR, h264Caps *VideoEncodeH264CapabilitiesKHR) error {
	var cCaps C.VkVideoCapabilitiesKHR
	var cEncodeCaps C.VkVideoEncodeCapabilitiesKHR
	var cH264Caps C.VkVideoEncodeH264CapabilitiesKHR

	result := C.call_vkGetPhysicalDeviceVideoCapabilitiesH264KHR(
		pd.handle,
		C.VkVideoCodecOperationFlagBitsKHR(VIDEO_CODEC_OPERATION_ENCODE_H264_BIT_KHR),
		C.VkVideoChromaSubsamplingFlagsKHR(VIDEO_CHROMA_SUBSAMPLING_420_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.StdVideoH264ProfileIdc(h264ProfileIdc),
		&cCaps,
		&cEncodeCaps,
		&cH264Caps,
	)

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	// Fill base capabilities
	caps.Flags = VideoCapabilityFlagsKHR(cCaps.flags)
	caps.MinBitstreamBufferOffsetAlignment = uint64(cCaps.minBitstreamBufferOffsetAlignment)
	caps.MinBitstreamBufferSizeAlignment = uint64(cCaps.minBitstreamBufferSizeAlignment)
	caps.PictureAccessGranularity = Extent2D{
		Width:  uint32(cCaps.pictureAccessGranularity.width),
		Height: uint32(cCaps.pictureAccessGranularity.height),
	}
	caps.MinCodedExtent = Extent2D{
		Width:  uint32(cCaps.minCodedExtent.width),
		Height: uint32(cCaps.minCodedExtent.height),
	}
	caps.MaxCodedExtent = Extent2D{
		Width:  uint32(cCaps.maxCodedExtent.width),
		Height: uint32(cCaps.maxCodedExtent.height),
	}
	caps.MaxDpbSlots = uint32(cCaps.maxDpbSlots)
	caps.MaxActiveReferencePictures = uint32(cCaps.maxActiveReferencePictures)

	// Fill encode capabilities
	encodeCaps.Flags = uint32(cEncodeCaps.flags)
	encodeCaps.RateControlModes = uint32(cEncodeCaps.rateControlModes)
	encodeCaps.MaxRateControlLayers = uint32(cEncodeCaps.maxRateControlLayers)
	encodeCaps.MaxBitrate = uint64(cEncodeCaps.maxBitrate)
	encodeCaps.MaxQualityLevels = uint32(cEncodeCaps.maxQualityLevels)
	encodeCaps.EncodeInputPictureGranularity = Extent2D{
		Width:  uint32(cEncodeCaps.encodeInputPictureGranularity.width),
		Height: uint32(cEncodeCaps.encodeInputPictureGranularity.height),
	}
	encodeCaps.SupportedEncodeFeedbackFlags = uint32(cEncodeCaps.supportedEncodeFeedbackFlags)

	// Fill H.264 capabilities
	h264Caps.Flags = uint32(cH264Caps.flags)
	h264Caps.MaxLevelIdc = uint32(cH264Caps.maxLevelIdc)
	h264Caps.MaxSliceCount = uint32(cH264Caps.maxSliceCount)
	h264Caps.MaxPPictureL0ReferenceCount = uint32(cH264Caps.maxPPictureL0ReferenceCount)
	h264Caps.MaxBPictureL0ReferenceCount = uint32(cH264Caps.maxBPictureL0ReferenceCount)
	h264Caps.MaxL1ReferenceCount = uint32(cH264Caps.maxL1ReferenceCount)
	h264Caps.MaxTemporalLayerCount = uint32(cH264Caps.maxTemporalLayerCount)
	h264Caps.ExpectDyadicTemporalLayerPattern = cH264Caps.expectDyadicTemporalLayerPattern != 0
	h264Caps.MinQp = int32(cH264Caps.minQp)
	h264Caps.MaxQp = int32(cH264Caps.maxQp)
	h264Caps.PrefersGopRemainingFrames = cH264Caps.prefersGopRemainingFrames != 0
	h264Caps.RequiresGopRemainingFrames = cH264Caps.requiresGopRemainingFrames != 0

	return nil
}

// CreateVideoSessionH264KHR creates a video session for H.264 encoding with proper pNext chaining
func (d Device) CreateVideoSessionH264KHR(queueFamilyIndex uint32, h264ProfileIdc uint32, pictureFormat Format, width, height uint32, maxDpbSlots, maxActiveReferencePictures uint32) (VideoSessionKHR, error) {
	var session C.VkVideoSessionKHR

	result := C.call_vkCreateVideoSessionH264KHR(
		d.handle,
		C.uint32_t(queueFamilyIndex),
		C.VkVideoCodecOperationFlagBitsKHR(VIDEO_CODEC_OPERATION_ENCODE_H264_BIT_KHR),
		C.VkVideoChromaSubsamplingFlagsKHR(VIDEO_CHROMA_SUBSAMPLING_420_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.StdVideoH264ProfileIdc(h264ProfileIdc),
		C.VkFormat(pictureFormat),
		C.VkExtent2D{width: C.uint32_t(width), height: C.uint32_t(height)},
		C.VkFormat(pictureFormat),
		C.uint32_t(maxDpbSlots),
		C.uint32_t(maxActiveReferencePictures),
		&session,
	)

	if result != C.VK_SUCCESS {
		return VideoSessionKHR{}, Result(result)
	}

	return VideoSessionKHR{handle: session}, nil
}

// CreateVideoSessionKHR creates a video session
func (d Device) CreateVideoSessionKHR(info *VideoSessionCreateInfoKHR) (VideoSessionKHR, error) {
	cProfile := C.VkVideoProfileInfoKHR{
		sType:               C.VkStructureType(VIDEO_PROFILE_INFO_KHR),
		videoCodecOperation: C.VkVideoCodecOperationFlagBitsKHR(info.PVideoProfile.VideoCodecOperation),
		chromaSubsampling:   C.VkVideoChromaSubsamplingFlagsKHR(info.PVideoProfile.ChromaSubsampling),
		lumaBitDepth:        C.VkVideoComponentBitDepthFlagsKHR(info.PVideoProfile.LumaBitDepth),
		chromaBitDepth:      C.VkVideoComponentBitDepthFlagsKHR(info.PVideoProfile.ChromaBitDepth),
	}

	cInfo := C.VkVideoSessionCreateInfoKHR{
		sType:            C.VkStructureType(VIDEO_SESSION_CREATE_INFO_KHR),
		queueFamilyIndex: C.uint32_t(info.QueueFamilyIndex),
		flags:            C.VkVideoSessionCreateFlagsKHR(info.Flags),
		pVideoProfile:    &cProfile,
		pictureFormat:    C.VkFormat(info.PictureFormat),
		maxCodedExtent: C.VkExtent2D{
			width:  C.uint32_t(info.MaxCodedExtent.Width),
			height: C.uint32_t(info.MaxCodedExtent.Height),
		},
		referencePictureFormat:     C.VkFormat(info.ReferencePictureFormat),
		maxDpbSlots:               C.uint32_t(info.MaxDpbSlots),
		maxActiveReferencePictures: C.uint32_t(info.MaxActiveReferencePictures),
	}

	var session C.VkVideoSessionKHR
	result := C.call_vkCreateVideoSessionKHR(d.handle, &cInfo, &session)
	if result != C.VK_SUCCESS {
		return VideoSessionKHR{}, Result(result)
	}

	return VideoSessionKHR{handle: session}, nil
}

// DestroyVideoSessionKHR destroys a video session
func (d Device) DestroyVideoSessionKHR(session VideoSessionKHR) {
	C.call_vkDestroyVideoSessionKHR(d.handle, session.handle)
}

// GetVideoSessionMemoryRequirementsKHR gets memory requirements for a video session
func (d Device) GetVideoSessionMemoryRequirementsKHR(session VideoSessionKHR) ([]VideoSessionMemoryRequirementsKHR, error) {
	var count C.uint32_t
	result := C.call_vkGetVideoSessionMemoryRequirementsKHR(
		d.handle,
		session.handle,
		&count,
		nil,
	)
	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	if count == 0 {
		return nil, nil
	}

	cReqs := make([]C.VkVideoSessionMemoryRequirementsKHR, count)
	for i := range cReqs {
		cReqs[i].sType = C.VkStructureType(VIDEO_SESSION_MEMORY_REQUIREMENTS_KHR)
	}

	result = C.call_vkGetVideoSessionMemoryRequirementsKHR(
		d.handle,
		session.handle,
		&count,
		&cReqs[0],
	)
	if result != C.VK_SUCCESS {
		return nil, Result(result)
	}

	reqs := make([]VideoSessionMemoryRequirementsKHR, count)
	for i, cr := range cReqs {
		reqs[i] = VideoSessionMemoryRequirementsKHR{
			SType:           StructureType(cr.sType),
			MemoryBindIndex: uint32(cr.memoryBindIndex),
			MemoryRequirements: MemoryRequirements{
				Size:           uint64(cr.memoryRequirements.size),
				Alignment:      uint64(cr.memoryRequirements.alignment),
				MemoryTypeBits: uint32(cr.memoryRequirements.memoryTypeBits),
			},
		}
	}

	return reqs, nil
}

// BindVideoSessionMemoryKHR binds memory to a video session
func (d Device) BindVideoSessionMemoryKHR(session VideoSessionKHR, bindings []BindVideoSessionMemoryInfoKHR) error {
	if len(bindings) == 0 {
		return nil
	}

	cBindings := make([]C.VkBindVideoSessionMemoryInfoKHR, len(bindings))
	for i, b := range bindings {
		cBindings[i] = C.VkBindVideoSessionMemoryInfoKHR{
			sType:           C.VkStructureType(BIND_VIDEO_SESSION_MEMORY_INFO_KHR),
			memoryBindIndex: C.uint32_t(b.MemoryBindIndex),
			memory:          b.Memory.handle,
			memoryOffset:    C.VkDeviceSize(b.MemoryOffset),
			memorySize:      C.VkDeviceSize(b.MemorySize),
		}
	}

	result := C.call_vkBindVideoSessionMemoryKHR(
		d.handle,
		session.handle,
		C.uint32_t(len(cBindings)),
		&cBindings[0],
	)
	if result != C.VK_SUCCESS {
		return Result(result)
	}

	return nil
}

// CreateVideoSessionParametersKHR creates video session parameters
func (d Device) CreateVideoSessionParametersKHR(info *VideoSessionParametersCreateInfoKHR) (VideoSessionParametersKHR, error) {
	cInfo := C.VkVideoSessionParametersCreateInfoKHR{
		sType:                          C.VkStructureType(VIDEO_SESSION_PARAMETERS_CREATE_INFO_KHR),
		pNext:                          info.PNext,
		flags:                          C.VkVideoSessionParametersCreateFlagsKHR(info.Flags),
		videoSessionParametersTemplate: info.VideoSessionParametersTemplate.handle,
		videoSession:                   info.VideoSession.handle,
	}

	var params C.VkVideoSessionParametersKHR
	result := C.call_vkCreateVideoSessionParametersKHR(d.handle, &cInfo, &params)
	if result != C.VK_SUCCESS {
		return VideoSessionParametersKHR{}, Result(result)
	}

	return VideoSessionParametersKHR{handle: params}, nil
}

// DestroyVideoSessionParametersKHR destroys video session parameters
func (d Device) DestroyVideoSessionParametersKHR(params VideoSessionParametersKHR) {
	C.call_vkDestroyVideoSessionParametersKHR(d.handle, params.handle)
}

// CreateVideoSessionParametersH264KHR creates video session parameters with H.264 SPS/PPS
func (d Device) CreateVideoSessionParametersH264KHR(session VideoSessionKHR, width, height, frameRateNum, frameRateDen uint32, profileIdc, levelIdc uint32) (VideoSessionParametersKHR, error) {
	var params C.VkVideoSessionParametersKHR

	result := C.call_vkCreateVideoSessionParametersH264KHR(
		d.handle,
		session.handle,
		C.uint32_t(width),
		C.uint32_t(height),
		C.uint32_t(frameRateNum),
		C.uint32_t(frameRateDen),
		C.StdVideoH264ProfileIdc(profileIdc),
		C.StdVideoH264LevelIdc(levelIdc),
		&params,
	)

	if result != C.VK_SUCCESS {
		return VideoSessionParametersKHR{}, Result(result)
	}

	return VideoSessionParametersKHR{handle: params}, nil
}

// ---- Command Buffer Functions ----

// CmdBeginVideoCodingKHR begins a video coding scope
func (cb CommandBuffer) CmdBeginVideoCodingKHR(info *VideoBeginCodingInfoKHR) {
	cInfo := C.VkVideoBeginCodingInfoKHR{
		sType:                  C.VkStructureType(VIDEO_BEGIN_CODING_INFO_KHR),
		flags:                  C.VkVideoBeginCodingFlagsKHR(info.Flags),
		videoSession:           info.VideoSession.handle,
		videoSessionParameters: info.VideoSessionParameters.handle,
		referenceSlotCount:     C.uint32_t(info.ReferenceSlotCount),
	}

	C.call_vkCmdBeginVideoCodingKHR(cb.handle, &cInfo)
}

// CmdEndVideoCodingKHR ends a video coding scope
func (cb CommandBuffer) CmdEndVideoCodingKHR() {
	cInfo := C.VkVideoEndCodingInfoKHR{
		sType: C.VkStructureType(VIDEO_END_CODING_INFO_KHR),
	}
	C.call_vkCmdEndVideoCodingKHR(cb.handle, &cInfo)
}

// CmdControlVideoCodingKHR controls video coding
func (cb CommandBuffer) CmdControlVideoCodingKHR(info *VideoCodingControlInfoKHR) {
	cInfo := C.VkVideoCodingControlInfoKHR{
		sType: C.VkStructureType(VIDEO_CODING_CONTROL_INFO_KHR),
		flags: C.VkVideoCodingControlFlagsKHR(info.Flags),
	}
	C.call_vkCmdControlVideoCodingKHR(cb.handle, &cInfo)
}

// CmdEncodeVideoKHR encodes video content
func (cb CommandBuffer) CmdEncodeVideoKHR(info *VideoEncodeInfoKHR) {
	cSrcResource := C.VkVideoPictureResourceInfoKHR{
		sType: C.VkStructureType(VIDEO_PICTURE_RESOURCE_INFO_KHR),
		codedOffset: C.VkOffset2D{
			x: C.int32_t(info.SrcPictureResource.CodedOffset.X),
			y: C.int32_t(info.SrcPictureResource.CodedOffset.Y),
		},
		codedExtent: C.VkExtent2D{
			width:  C.uint32_t(info.SrcPictureResource.CodedExtent.Width),
			height: C.uint32_t(info.SrcPictureResource.CodedExtent.Height),
		},
		baseArrayLayer:   C.uint32_t(info.SrcPictureResource.BaseArrayLayer),
		imageViewBinding: info.SrcPictureResource.ImageViewBinding.handle,
	}

	cInfo := C.VkVideoEncodeInfoKHR{
		sType:                           C.VkStructureType(VIDEO_ENCODE_INFO_KHR),
		flags:                           C.VkVideoEncodeFlagsKHR(info.Flags),
		dstBuffer:                       info.DstBuffer.handle,
		dstBufferOffset:                 C.VkDeviceSize(info.DstBufferOffset),
		dstBufferRange:                  C.VkDeviceSize(info.DstBufferRange),
		srcPictureResource:              cSrcResource,
		referenceSlotCount:              C.uint32_t(info.ReferenceSlotCount),
		precedingExternallyEncodedBytes: C.uint32_t(info.PrecedingExternallyEncodedBytes),
	}

	C.call_vkCmdEncodeVideoKHR(cb.handle, &cInfo)
}

// CmdDecodeVideoKHR decodes video content
func (cb CommandBuffer) CmdDecodeVideoKHR(info *VideoDecodeInfoKHR) {
	cDstResource := C.VkVideoPictureResourceInfoKHR{
		sType: C.VkStructureType(VIDEO_PICTURE_RESOURCE_INFO_KHR),
		codedOffset: C.VkOffset2D{
			x: C.int32_t(info.DstPictureResource.CodedOffset.X),
			y: C.int32_t(info.DstPictureResource.CodedOffset.Y),
		},
		codedExtent: C.VkExtent2D{
			width:  C.uint32_t(info.DstPictureResource.CodedExtent.Width),
			height: C.uint32_t(info.DstPictureResource.CodedExtent.Height),
		},
		baseArrayLayer:   C.uint32_t(info.DstPictureResource.BaseArrayLayer),
		imageViewBinding: info.DstPictureResource.ImageViewBinding.handle,
	}

	cInfo := C.VkVideoDecodeInfoKHR{
		sType:              C.VkStructureType(VIDEO_DECODE_INFO_KHR),
		flags:              C.VkVideoDecodeFlagsKHR(info.Flags),
		srcBuffer:          info.SrcBuffer.handle,
		srcBufferOffset:    C.VkDeviceSize(info.SrcBufferOffset),
		srcBufferRange:     C.VkDeviceSize(info.SrcBufferRange),
		dstPictureResource: cDstResource,
		referenceSlotCount: C.uint32_t(info.ReferenceSlotCount),
	}

	C.call_vkCmdDecodeVideoKHR(cb.handle, &cInfo)
}

// ---- Helper Functions ----

// CheckVideoSupport checks if the physical device supports video encode/decode
func (pd PhysicalDevice) CheckVideoSupport() (encode, decode bool) {
	var count C.uint32_t
	C.vkGetPhysicalDeviceQueueFamilyProperties2(pd.handle, &count, nil)

	if count == 0 {
		return false, false
	}

	// Allocate C memory to avoid CGo pointer issues
	propsSize := C.size_t(count) * C.size_t(unsafe.Sizeof(C.VkQueueFamilyProperties2{}))
	videoPropsSize := C.size_t(count) * C.size_t(unsafe.Sizeof(C.VkQueueFamilyVideoPropertiesKHR{}))

	propsPtr := C.malloc(propsSize)
	videoPropsPtr := C.malloc(videoPropsSize)
	defer C.free(propsPtr)
	defer C.free(videoPropsPtr)

	props := (*[1024]C.VkQueueFamilyProperties2)(propsPtr)[:count:count]
	videoProps := (*[1024]C.VkQueueFamilyVideoPropertiesKHR)(videoPropsPtr)[:count:count]

	for i := C.uint32_t(0); i < count; i++ {
		props[i].sType = C.VkStructureType(QUEUE_FAMILY_PROPERTIES_2)
		props[i].pNext = unsafe.Pointer(&videoProps[i])
		videoProps[i].sType = C.VkStructureType(QUEUE_FAMILY_VIDEO_PROPERTIES_KHR)
		videoProps[i].pNext = nil
	}

	C.vkGetPhysicalDeviceQueueFamilyProperties2(pd.handle, &count, (*C.VkQueueFamilyProperties2)(propsPtr))

	for i := uint32(0); i < uint32(count); i++ {
		ops := VideoCodecOperationFlagsKHR(videoProps[i].videoCodecOperations)
		if ops&VIDEO_CODEC_OPERATION_ENCODE_H264_BIT_KHR != 0 ||
			ops&VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR != 0 {
			encode = true
		}
		if ops&VIDEO_CODEC_OPERATION_DECODE_H264_BIT_KHR != 0 ||
			ops&VIDEO_CODEC_OPERATION_DECODE_H265_BIT_KHR != 0 ||
			ops&VIDEO_CODEC_OPERATION_DECODE_AV1_BIT_KHR != 0 {
			decode = true
		}
	}

	return encode, decode
}

// GetVideoQueueFamilyIndex finds a queue family that supports the specified video operation
func (pd PhysicalDevice) GetVideoQueueFamilyIndex(operation VideoCodecOperationFlagsKHR) (uint32, error) {
	var count C.uint32_t
	C.vkGetPhysicalDeviceQueueFamilyProperties2(pd.handle, &count, nil)

	if count == 0 {
		return 0, fmt.Errorf("no queue families found")
	}

	// Allocate C memory to avoid CGo pointer issues
	propsSize := C.size_t(count) * C.size_t(unsafe.Sizeof(C.VkQueueFamilyProperties2{}))
	videoPropsSize := C.size_t(count) * C.size_t(unsafe.Sizeof(C.VkQueueFamilyVideoPropertiesKHR{}))

	propsPtr := C.malloc(propsSize)
	videoPropsPtr := C.malloc(videoPropsSize)
	defer C.free(propsPtr)
	defer C.free(videoPropsPtr)

	props := (*[1024]C.VkQueueFamilyProperties2)(propsPtr)[:count:count]
	videoProps := (*[1024]C.VkQueueFamilyVideoPropertiesKHR)(videoPropsPtr)[:count:count]

	for i := C.uint32_t(0); i < count; i++ {
		props[i].sType = C.VkStructureType(QUEUE_FAMILY_PROPERTIES_2)
		props[i].pNext = unsafe.Pointer(&videoProps[i])
		videoProps[i].sType = C.VkStructureType(QUEUE_FAMILY_VIDEO_PROPERTIES_KHR)
		videoProps[i].pNext = nil
	}

	C.vkGetPhysicalDeviceQueueFamilyProperties2(pd.handle, &count, (*C.VkQueueFamilyProperties2)(propsPtr))

	for i := uint32(0); i < uint32(count); i++ {
		ops := VideoCodecOperationFlagsKHR(videoProps[i].videoCodecOperations)
		if ops&operation != 0 {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no queue family supports the requested video operation")
}

// ---- H.265/HEVC Video Encoding Functions ----

// H.265 Profile IDC values
const (
	STD_VIDEO_H265_PROFILE_IDC_MAIN               uint32 = 1
	STD_VIDEO_H265_PROFILE_IDC_MAIN_10            uint32 = 2
	STD_VIDEO_H265_PROFILE_IDC_MAIN_STILL_PICTURE uint32 = 3
	STD_VIDEO_H265_PROFILE_IDC_FORMAT_RANGE_EXT   uint32 = 4
	STD_VIDEO_H265_PROFILE_IDC_SCC_EXTENSIONS     uint32 = 9
)

// H.265 Level IDC values (StdVideoH265LevelIdc enum)
const (
	STD_VIDEO_H265_LEVEL_IDC_1_0 uint32 = 0
	STD_VIDEO_H265_LEVEL_IDC_2_0 uint32 = 1
	STD_VIDEO_H265_LEVEL_IDC_2_1 uint32 = 2
	STD_VIDEO_H265_LEVEL_IDC_3_0 uint32 = 3
	STD_VIDEO_H265_LEVEL_IDC_3_1 uint32 = 4
	STD_VIDEO_H265_LEVEL_IDC_4_0 uint32 = 5
	STD_VIDEO_H265_LEVEL_IDC_4_1 uint32 = 6
	STD_VIDEO_H265_LEVEL_IDC_5_0 uint32 = 7
	STD_VIDEO_H265_LEVEL_IDC_5_1 uint32 = 8
	STD_VIDEO_H265_LEVEL_IDC_5_2 uint32 = 9
	STD_VIDEO_H265_LEVEL_IDC_6_0 uint32 = 10
	STD_VIDEO_H265_LEVEL_IDC_6_1 uint32 = 11
	STD_VIDEO_H265_LEVEL_IDC_6_2 uint32 = 12
)

// VideoEncodeH265CapabilitiesKHR describes H.265 encode capabilities
type VideoEncodeH265CapabilitiesKHR struct {
	Flags                             uint32
	MaxLevelIdc                       uint32
	MaxSliceSegmentCount              uint32
	MaxTiles                          Extent2D
	CtbSizes                          uint32
	TransformBlockSizes               uint32
	MaxPPictureL0ReferenceCount       uint32
	MaxBPictureL0ReferenceCount       uint32
	MaxL1ReferenceCount               uint32
	MaxSubLayerCount                  uint32
	ExpectDyadicTemporalSubLayerPattern bool
	MinQp                             int32
	MaxQp                             int32
	PrefersGopRemainingFrames         bool
	RequiresGopRemainingFrames        bool
}

// CheckH265EncodeSupport checks if the physical device supports H.265 hardware encoding
func (pd PhysicalDevice) CheckH265EncodeSupport() bool {
	return C.checkH265EncodeSupport(pd.handle) != 0
}

// GetVideoCapabilitiesH265KHR queries H.265 encode capabilities with proper pNext chaining
func (pd PhysicalDevice) GetVideoCapabilitiesH265KHR(h265ProfileIdc uint32, caps *VideoCapabilitiesKHR, encodeCaps *VideoEncodeCapabilitiesKHR, h265Caps *VideoEncodeH265CapabilitiesKHR) error {
	var cCaps C.VkVideoCapabilitiesKHR
	var cEncodeCaps C.VkVideoEncodeCapabilitiesKHR
	var cH265Caps C.VkVideoEncodeH265CapabilitiesKHR

	result := C.call_vkGetPhysicalDeviceVideoCapabilitiesH265KHR(
		pd.handle,
		C.VkVideoCodecOperationFlagBitsKHR(VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR),
		C.VkVideoChromaSubsamplingFlagsKHR(VIDEO_CHROMA_SUBSAMPLING_420_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.StdVideoH265ProfileIdc(h265ProfileIdc),
		&cCaps,
		&cEncodeCaps,
		&cH265Caps,
	)

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	// Fill base capabilities
	caps.Flags = VideoCapabilityFlagsKHR(cCaps.flags)
	caps.MinBitstreamBufferOffsetAlignment = uint64(cCaps.minBitstreamBufferOffsetAlignment)
	caps.MinBitstreamBufferSizeAlignment = uint64(cCaps.minBitstreamBufferSizeAlignment)
	caps.PictureAccessGranularity = Extent2D{
		Width:  uint32(cCaps.pictureAccessGranularity.width),
		Height: uint32(cCaps.pictureAccessGranularity.height),
	}
	caps.MinCodedExtent = Extent2D{
		Width:  uint32(cCaps.minCodedExtent.width),
		Height: uint32(cCaps.minCodedExtent.height),
	}
	caps.MaxCodedExtent = Extent2D{
		Width:  uint32(cCaps.maxCodedExtent.width),
		Height: uint32(cCaps.maxCodedExtent.height),
	}
	caps.MaxDpbSlots = uint32(cCaps.maxDpbSlots)
	caps.MaxActiveReferencePictures = uint32(cCaps.maxActiveReferencePictures)

	// Fill encode capabilities
	encodeCaps.Flags = uint32(cEncodeCaps.flags)
	encodeCaps.RateControlModes = uint32(cEncodeCaps.rateControlModes)
	encodeCaps.MaxRateControlLayers = uint32(cEncodeCaps.maxRateControlLayers)
	encodeCaps.MaxBitrate = uint64(cEncodeCaps.maxBitrate)
	encodeCaps.MaxQualityLevels = uint32(cEncodeCaps.maxQualityLevels)
	encodeCaps.EncodeInputPictureGranularity = Extent2D{
		Width:  uint32(cEncodeCaps.encodeInputPictureGranularity.width),
		Height: uint32(cEncodeCaps.encodeInputPictureGranularity.height),
	}
	encodeCaps.SupportedEncodeFeedbackFlags = uint32(cEncodeCaps.supportedEncodeFeedbackFlags)

	// Fill H.265 capabilities
	h265Caps.Flags = uint32(cH265Caps.flags)
	h265Caps.MaxLevelIdc = uint32(cH265Caps.maxLevelIdc)
	h265Caps.MaxSliceSegmentCount = uint32(cH265Caps.maxSliceSegmentCount)
	h265Caps.MaxTiles = Extent2D{
		Width:  uint32(cH265Caps.maxTiles.width),
		Height: uint32(cH265Caps.maxTiles.height),
	}
	h265Caps.CtbSizes = uint32(cH265Caps.ctbSizes)
	h265Caps.TransformBlockSizes = uint32(cH265Caps.transformBlockSizes)
	h265Caps.MaxPPictureL0ReferenceCount = uint32(cH265Caps.maxPPictureL0ReferenceCount)
	h265Caps.MaxBPictureL0ReferenceCount = uint32(cH265Caps.maxBPictureL0ReferenceCount)
	h265Caps.MaxL1ReferenceCount = uint32(cH265Caps.maxL1ReferenceCount)
	h265Caps.MaxSubLayerCount = uint32(cH265Caps.maxSubLayerCount)
	h265Caps.ExpectDyadicTemporalSubLayerPattern = cH265Caps.expectDyadicTemporalSubLayerPattern != 0
	h265Caps.MinQp = int32(cH265Caps.minQp)
	h265Caps.MaxQp = int32(cH265Caps.maxQp)
	h265Caps.PrefersGopRemainingFrames = cH265Caps.prefersGopRemainingFrames != 0
	h265Caps.RequiresGopRemainingFrames = cH265Caps.requiresGopRemainingFrames != 0

	return nil
}

// CreateVideoSessionH265KHR creates a video session for H.265 encoding with proper pNext chaining
func (d Device) CreateVideoSessionH265KHR(queueFamilyIndex uint32, h265ProfileIdc uint32, pictureFormat Format, width, height uint32, maxDpbSlots, maxActiveReferencePictures uint32) (VideoSessionKHR, error) {
	var session C.VkVideoSessionKHR

	result := C.call_vkCreateVideoSessionH265KHR(
		d.handle,
		C.uint32_t(queueFamilyIndex),
		C.VkVideoCodecOperationFlagBitsKHR(VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR),
		C.VkVideoChromaSubsamplingFlagsKHR(VIDEO_CHROMA_SUBSAMPLING_420_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.VkVideoComponentBitDepthFlagsKHR(VIDEO_COMPONENT_BIT_DEPTH_8_BIT_KHR),
		C.StdVideoH265ProfileIdc(h265ProfileIdc),
		C.VkFormat(pictureFormat),
		C.VkExtent2D{width: C.uint32_t(width), height: C.uint32_t(height)},
		C.VkFormat(pictureFormat),
		C.uint32_t(maxDpbSlots),
		C.uint32_t(maxActiveReferencePictures),
		&session,
	)

	if result != C.VK_SUCCESS {
		return VideoSessionKHR{}, Result(result)
	}

	return VideoSessionKHR{handle: session}, nil
}

// CreateVideoSessionParametersH265KHR creates video session parameters with H.265 VPS/SPS/PPS
func (d Device) CreateVideoSessionParametersH265KHR(session VideoSessionKHR, width, height, frameRateNum, frameRateDen uint32, profileIdc, levelIdc uint32) (VideoSessionParametersKHR, error) {
	var params C.VkVideoSessionParametersKHR

	result := C.call_vkCreateVideoSessionParametersH265KHR(
		d.handle,
		session.handle,
		C.uint32_t(width),
		C.uint32_t(height),
		C.uint32_t(frameRateNum),
		C.uint32_t(frameRateDen),
		C.StdVideoH265ProfileIdc(profileIdc),
		C.StdVideoH265LevelIdc(levelIdc),
		&params,
	)

	if result != C.VK_SUCCESS {
		return VideoSessionParametersKHR{}, Result(result)
	}

	return VideoSessionParametersKHR{handle: params}, nil
}

// CmdBeginVideoCodingH265KHR begins video coding for H.265 encode
func (cb CommandBuffer) CmdBeginVideoCodingH265KHR(session VideoSessionKHR, params VideoSessionParametersKHR, width, height uint32) {
	C.call_vkCmdBeginVideoCodingH265KHR(
		cb.handle,
		session.handle,
		params.handle,
		nil, // dpbImageView - not used for I-frame only
		C.uint32_t(width),
		C.uint32_t(height),
	)
}

// CmdEncodeVideoH265KHR encodes a single H.265 frame
func (cb CommandBuffer) CmdEncodeVideoH265KHR(dstBuffer Buffer, dstOffset, dstRange uint64, srcImageView ImageView, width, height, frameNum uint32, isIDR bool) {
	idrFlag := 0
	if isIDR {
		idrFlag = 1
	}
	C.call_vkCmdEncodeVideoH265KHR(
		cb.handle,
		dstBuffer.handle,
		C.VkDeviceSize(dstOffset),
		C.VkDeviceSize(dstRange),
		srcImageView.handle,
		C.uint32_t(width),
		C.uint32_t(height),
		C.uint32_t(frameNum),
		C.int(idrFlag),
	)
}
