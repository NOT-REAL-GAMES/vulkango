package vulkango

/*
#cgo CFLAGS: -I/usr/include
#cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lvulkan
#cgo windows LDFLAGS: -lvulkan-1
#cgo darwin LDFLAGS: -lvulkan

#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"fmt"
)

// H265HardwareEncoder manages Vulkan Video H.265 hardware encoding
type H265HardwareEncoder struct {
	device         Device
	physicalDevice PhysicalDevice
	config         HEVCEncoderConfig

	// Video session
	videoSession    VideoSessionKHR
	sessionParams   VideoSessionParametersKHR
	sessionMemories []DeviceMemory

	// Video queue
	videoQueueFamily uint32
	videoQueue       Queue

	// Command pool and buffers
	commandPool   CommandPool
	commandBuffer CommandBuffer

	// Bitstream buffer (output)
	bitstreamBuffer Buffer
	bitstreamMemory DeviceMemory
	bitstreamSize   uint64

	// DPB (Decoded Picture Buffer) for reference frames
	dpbImages     []Image
	dpbImageViews []ImageView
	dpbMemories   []DeviceMemory

	// Input image (YCbCr format)
	inputImage     Image
	inputImageView ImageView
	inputMemory    DeviceMemory

	// Fence for synchronization
	encodeFence Fence

	// Frame tracking
	frameNum    uint32
	gopFrameNum uint32
	idrPicId    uint32
	encodedData []byte

	// VPS/SPS/PPS (generated from capabilities query)
	vps []byte
	sps []byte
	pps []byte

	// MP4 writer
	mp4Writer *MP4WriterHEVC

	// Initialized flag
	initialized bool

	// Hardware support flag
	hardwareSupported bool
}

// NewH265HardwareEncoder creates a new H.265 hardware encoder using Vulkan Video
func NewH265HardwareEncoder(device Device, physicalDevice PhysicalDevice, config HEVCEncoderConfig) (*H265HardwareEncoder, error) {
	enc := &H265HardwareEncoder{
		device:         device,
		physicalDevice: physicalDevice,
		config:         config,
		bitstreamSize:  4 * 1024 * 1024, // 4MB bitstream buffer
	}

	// Check hardware support
	enc.hardwareSupported = physicalDevice.CheckH265EncodeSupport()
	if !enc.hardwareSupported {
		return nil, fmt.Errorf("H.265 hardware encoding not supported on this GPU")
	}

	// Generate VPS, SPS, PPS
	enc.vps = GenerateHEVCVPS(config)
	enc.sps = GenerateHEVCSPS(config)
	enc.pps = GenerateHEVCPPS(config)

	// Initialize MP4 writer
	enc.mp4Writer = NewMP4WriterHEVC(config, enc.vps, enc.sps, enc.pps)

	return enc, nil
}

// IsHardwareSupported returns true if hardware H.265 encoding is available
func (enc *H265HardwareEncoder) IsHardwareSupported() bool {
	return enc.hardwareSupported
}

// Initialize sets up the Vulkan Video encoding session
func (enc *H265HardwareEncoder) Initialize() error {
	if enc.initialized {
		return nil
	}

	if !enc.hardwareSupported {
		return fmt.Errorf("H.265 hardware encoding not supported")
	}

	// Load video extension functions
	if !LoadVideoExtensionsDevice(enc.device) {
		return fmt.Errorf("failed to load video extension functions - ensure VK_KHR_video_queue and VK_KHR_video_encode_queue extensions are enabled")
	}

	// Find video encode queue family
	var err error
	enc.videoQueueFamily, err = enc.physicalDevice.GetVideoQueueFamilyIndex(VIDEO_CODEC_OPERATION_ENCODE_H265_BIT_KHR)
	if err != nil {
		return fmt.Errorf("failed to find H.265 encode queue family: %v", err)
	}

	// Get video queue
	enc.videoQueue = enc.device.GetQueue(enc.videoQueueFamily, 0)

	// Create command pool for video operations
	enc.commandPool, err = enc.device.CreateCommandPool(&CommandPoolCreateInfo{
		Flags:            COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
		QueueFamilyIndex: enc.videoQueueFamily,
	})
	if err != nil {
		return fmt.Errorf("failed to create command pool: %v", err)
	}

	// Allocate command buffer
	cmdBufs, err := enc.device.AllocateCommandBuffers(&CommandBufferAllocateInfo{
		CommandPool:        enc.commandPool,
		Level:              COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate command buffer: %v", err)
	}
	enc.commandBuffer = cmdBufs[0]

	// Create fence
	enc.encodeFence, err = enc.device.CreateFence(&FenceCreateInfo{})
	if err != nil {
		return fmt.Errorf("failed to create fence: %v", err)
	}

	// Create bitstream buffer
	enc.bitstreamBuffer, enc.bitstreamMemory, err = enc.device.CreateBufferWithMemory(
		enc.bitstreamSize,
		BUFFER_USAGE_VIDEO_ENCODE_DST_BIT_KHR,
		MEMORY_PROPERTY_HOST_VISIBLE_BIT|MEMORY_PROPERTY_HOST_COHERENT_BIT,
		enc.physicalDevice,
	)
	if err != nil {
		return fmt.Errorf("failed to create bitstream buffer: %v", err)
	}

	// Create video session
	err = enc.createVideoSession()
	if err != nil {
		return fmt.Errorf("failed to create video session: %v", err)
	}

	enc.initialized = true
	return nil
}

func (enc *H265HardwareEncoder) createVideoSession() error {
	// Map our HEVC profile to the Vulkan standard profile IDC
	h265ProfileIdc := STD_VIDEO_H265_PROFILE_IDC_MAIN
	switch enc.config.Profile {
	case HEVC_PROFILE_MAIN:
		h265ProfileIdc = STD_VIDEO_H265_PROFILE_IDC_MAIN
	case HEVC_PROFILE_MAIN_10:
		h265ProfileIdc = STD_VIDEO_H265_PROFILE_IDC_MAIN_10
	case HEVC_PROFILE_REXT:
		h265ProfileIdc = STD_VIDEO_H265_PROFILE_IDC_FORMAT_RANGE_EXT
	}

	// Query H.265 capabilities with proper pNext chaining
	var caps VideoCapabilitiesKHR
	var encodeCaps VideoEncodeCapabilitiesKHR
	var h265Caps VideoEncodeH265CapabilitiesKHR
	err := enc.physicalDevice.GetVideoCapabilitiesH265KHR(h265ProfileIdc, &caps, &encodeCaps, &h265Caps)
	if err != nil {
		return fmt.Errorf("failed to get video capabilities: %v", err)
	}

	fmt.Printf("H.265 Encode Capabilities:\n")
	fmt.Printf("  Max coded extent: %dx%d\n", caps.MaxCodedExtent.Width, caps.MaxCodedExtent.Height)
	fmt.Printf("  Min coded extent: %dx%d\n", caps.MinCodedExtent.Width, caps.MinCodedExtent.Height)
	fmt.Printf("  Max DPB slots: %d\n", caps.MaxDpbSlots)
	fmt.Printf("  Max active refs: %d\n", caps.MaxActiveReferencePictures)
	fmt.Printf("  Max bitrate: %d\n", encodeCaps.MaxBitrate)
	fmt.Printf("  Max level IDC: %d\n", h265Caps.MaxLevelIdc)
	fmt.Printf("  QP range: %d - %d\n", h265Caps.MinQp, h265Caps.MaxQp)

	// Validate dimensions
	if enc.config.Width < caps.MinCodedExtent.Width || enc.config.Height < caps.MinCodedExtent.Height {
		return fmt.Errorf("dimensions %dx%d below minimum %dx%d",
			enc.config.Width, enc.config.Height,
			caps.MinCodedExtent.Width, caps.MinCodedExtent.Height)
	}
	if enc.config.Width > caps.MaxCodedExtent.Width || enc.config.Height > caps.MaxCodedExtent.Height {
		return fmt.Errorf("dimensions %dx%d exceed maximum %dx%d",
			enc.config.Width, enc.config.Height,
			caps.MaxCodedExtent.Width, caps.MaxCodedExtent.Height)
	}

	// Determine DPB slots and reference pictures (use I-frames only for simplicity)
	maxDpbSlots := uint32(1)
	maxActiveRefs := uint32(0) // No reference pictures for I-frame only encoding

	// Create video session
	enc.videoSession, err = enc.device.CreateVideoSessionH265KHR(
		enc.videoQueueFamily,
		h265ProfileIdc,
		FORMAT_G8_B8R8_2PLANE_420_UNORM, // NV12 format
		enc.config.Width,
		enc.config.Height,
		maxDpbSlots,
		maxActiveRefs,
	)
	if err != nil {
		return fmt.Errorf("failed to create video session: %v", err)
	}

	// Bind memory to video session
	err = enc.bindVideoSessionMemory()
	if err != nil {
		return fmt.Errorf("failed to bind video session memory: %v", err)
	}

	// Map level to StdVideoH265LevelIdc
	levelIdc := STD_VIDEO_H265_LEVEL_IDC_4_1
	switch enc.config.Level {
	case HEVC_LEVEL_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_1_0
	case HEVC_LEVEL_2:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_2_0
	case HEVC_LEVEL_2_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_2_1
	case HEVC_LEVEL_3:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_3_0
	case HEVC_LEVEL_3_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_3_1
	case HEVC_LEVEL_4:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_4_0
	case HEVC_LEVEL_4_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_4_1
	case HEVC_LEVEL_5:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_5_0
	case HEVC_LEVEL_5_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_5_1
	case HEVC_LEVEL_5_2:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_5_2
	case HEVC_LEVEL_6:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_6_0
	case HEVC_LEVEL_6_1:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_6_1
	case HEVC_LEVEL_6_2:
		levelIdc = STD_VIDEO_H265_LEVEL_IDC_6_2
	}

	// Create session parameters with VPS/SPS/PPS
	enc.sessionParams, err = enc.device.CreateVideoSessionParametersH265KHR(
		enc.videoSession,
		enc.config.Width,
		enc.config.Height,
		enc.config.FrameRateNum,
		enc.config.FrameRateDen,
		h265ProfileIdc,
		levelIdc,
	)
	if err != nil {
		return fmt.Errorf("failed to create session parameters: %v", err)
	}

	return nil
}

func (enc *H265HardwareEncoder) bindVideoSessionMemory() error {
	// Get memory requirements
	memReqs, err := enc.device.GetVideoSessionMemoryRequirementsKHR(enc.videoSession)
	if err != nil {
		return fmt.Errorf("failed to get video session memory requirements: %v", err)
	}

	if len(memReqs) == 0 {
		return nil // No memory bindings required
	}

	// Allocate and bind memory for each requirement
	bindings := make([]BindVideoSessionMemoryInfoKHR, len(memReqs))
	enc.sessionMemories = make([]DeviceMemory, len(memReqs))

	for i, req := range memReqs {
		// Find suitable memory type
		memoryTypeIndex := uint32(0)
		memProps := enc.physicalDevice.GetMemoryProperties()
		for j := uint32(0); j < memProps.MemoryTypeCount; j++ {
			if (req.MemoryRequirements.MemoryTypeBits & (1 << j)) != 0 {
				memoryTypeIndex = j
				break
			}
		}

		// Allocate memory
		memory, err := enc.device.AllocateMemory(&MemoryAllocateInfo{
			AllocationSize:  req.MemoryRequirements.Size,
			MemoryTypeIndex: memoryTypeIndex,
		})
		if err != nil {
			return fmt.Errorf("failed to allocate video session memory %d: %v", i, err)
		}
		enc.sessionMemories[i] = memory

		bindings[i] = BindVideoSessionMemoryInfoKHR{
			MemoryBindIndex: req.MemoryBindIndex,
			Memory:          memory,
			MemoryOffset:    0,
			MemorySize:      req.MemoryRequirements.Size,
		}
	}

	// Bind all memories
	err = enc.device.BindVideoSessionMemoryKHR(enc.videoSession, bindings)
	if err != nil {
		return fmt.Errorf("failed to bind video session memory: %v", err)
	}

	return nil
}

// createInputImage creates the input image for video encoding
func (enc *H265HardwareEncoder) createInputImage() error {
	// Create NV12 input image with video encode source usage
	// NV12 has Y plane at full resolution and interleaved UV at half resolution
	imageInfo := ImageCreateInfo{
		ImageType: IMAGE_TYPE_2D,
		Format:    FORMAT_G8_B8R8_2PLANE_420_UNORM, // NV12
		Extent: Extent3D{
			Width:  enc.config.Width,
			Height: enc.config.Height,
			Depth:  1,
		},
		MipLevels:     1,
		ArrayLayers:   1,
		Samples:       SAMPLE_COUNT_1_BIT,
		Tiling:        IMAGE_TILING_OPTIMAL,
		Usage:         IMAGE_USAGE_TRANSFER_DST_BIT | IMAGE_USAGE_VIDEO_ENCODE_SRC_BIT_KHR,
		SharingMode:   SHARING_MODE_EXCLUSIVE,
		InitialLayout: IMAGE_LAYOUT_UNDEFINED,
	}

	var err error
	enc.inputImage, err = enc.device.CreateImage(&imageInfo)
	if err != nil {
		return fmt.Errorf("failed to create input image: %v", err)
	}

	// Get memory requirements
	memReqs := enc.device.GetImageMemoryRequirements(enc.inputImage)

	// Find device local memory
	memProps := enc.physicalDevice.GetMemoryProperties()
	memTypeIndex, found := FindMemoryType(memProps, memReqs.MemoryTypeBits, MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
	if !found {
		return fmt.Errorf("failed to find suitable memory type for input image")
	}

	// Allocate memory
	enc.inputMemory, err = enc.device.AllocateMemory(&MemoryAllocateInfo{
		AllocationSize:  memReqs.Size,
		MemoryTypeIndex: memTypeIndex,
	})
	if err != nil {
		return fmt.Errorf("failed to allocate input image memory: %v", err)
	}

	// Bind memory
	err = enc.device.BindImageMemory(enc.inputImage, enc.inputMemory, 0)
	if err != nil {
		return fmt.Errorf("failed to bind input image memory: %v", err)
	}

	// Create image view - for multi-planar formats, use both plane aspects
	viewInfo := ImageViewCreateInfo{
		Image:    enc.inputImage,
		ViewType: IMAGE_VIEW_TYPE_2D,
		Format:   FORMAT_G8_B8R8_2PLANE_420_UNORM,
		SubresourceRange: ImageSubresourceRange{
			AspectMask:     IMAGE_ASPECT_PLANE_0_BIT | IMAGE_ASPECT_PLANE_1_BIT,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	}

	enc.inputImageView, err = enc.device.CreateImageView(&viewInfo)
	if err != nil {
		return fmt.Errorf("failed to create input image view: %v", err)
	}

	return nil
}

// rgbaToNV12 converts RGBA data to NV12 (YCbCr 4:2:0) format
func rgbaToNV12(rgba []byte, width, height uint32) []byte {
	// NV12: Y plane (width*height) + interleaved UV plane (width*height/2)
	ySize := width * height
	uvSize := width * height / 2
	nv12 := make([]byte, ySize+uvSize)

	// Convert to Y plane
	for y := uint32(0); y < height; y++ {
		for x := uint32(0); x < width; x++ {
			idx := (y*width + x) * 4
			r := float32(rgba[idx])
			g := float32(rgba[idx+1])
			b := float32(rgba[idx+2])

			// BT.601 luma
			yVal := 16.0 + 65.481*r/255.0 + 128.553*g/255.0 + 24.966*b/255.0
			if yVal < 16 {
				yVal = 16
			} else if yVal > 235 {
				yVal = 235
			}
			nv12[y*width+x] = uint8(yVal)
		}
	}

	// Convert to interleaved UV plane (subsampled 2x2)
	uvOffset := ySize
	for y := uint32(0); y < height/2; y++ {
		for x := uint32(0); x < width/2; x++ {
			// Sample from top-left of 2x2 block
			px := x * 2
			py := y * 2
			idx := (py*width + px) * 4

			r := float32(rgba[idx])
			g := float32(rgba[idx+1])
			b := float32(rgba[idx+2])

			// BT.601 Cb and Cr
			cb := 128.0 - 37.797*r/255.0 - 74.203*g/255.0 + 112.0*b/255.0
			cr := 128.0 + 112.0*r/255.0 - 93.786*g/255.0 - 18.214*b/255.0

			if cb < 16 {
				cb = 16
			} else if cb > 240 {
				cb = 240
			}
			if cr < 16 {
				cr = 16
			} else if cr > 240 {
				cr = 240
			}

			// NV12 has interleaved U,V
			uvIdx := uvOffset + (y*width + x*2)
			nv12[uvIdx] = uint8(cb)
			nv12[uvIdx+1] = uint8(cr)
		}
	}

	return nv12
}

// EncodeFrame encodes an RGBA frame using hardware H.265 encoding
func (enc *H265HardwareEncoder) EncodeFrame(rgbaData []byte) ([]byte, error) {
	if !enc.initialized {
		if err := enc.Initialize(); err != nil {
			return nil, err
		}
	}

	// Create input image on first frame
	if enc.inputImage.handle == nil {
		if err := enc.createInputImage(); err != nil {
			return nil, fmt.Errorf("failed to create input image: %v", err)
		}
	}

	// Determine frame type
	isIDR := enc.frameNum == 0 || enc.gopFrameNum >= enc.config.GOPSize
	if isIDR {
		enc.gopFrameNum = 0
	}

	// Convert RGBA to NV12
	nv12Data := rgbaToNV12(rgbaData, enc.config.Width, enc.config.Height)

	// Upload NV12 data to input image using staging buffer
	stagingSize := uint64(len(nv12Data))
	stagingBuffer, stagingMemory, err := enc.device.CreateBufferWithMemory(
		stagingSize,
		BUFFER_USAGE_TRANSFER_SRC_BIT,
		MEMORY_PROPERTY_HOST_VISIBLE_BIT|MEMORY_PROPERTY_HOST_COHERENT_BIT,
		enc.physicalDevice,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %v", err)
	}
	defer enc.device.DestroyBuffer(stagingBuffer)
	defer enc.device.FreeMemory(stagingMemory)

	// Map and copy data
	mappedPtr, err := enc.device.MapMemory(stagingMemory, 0, stagingSize)
	if err != nil {
		return nil, fmt.Errorf("failed to map staging memory: %v", err)
	}
	copy((*[1 << 30]byte)(mappedPtr)[:len(nv12Data)], nv12Data)
	enc.device.UnmapMemory(stagingMemory)

	// Reset command buffer
	enc.commandBuffer.Reset(0)

	// Begin command buffer
	err = enc.commandBuffer.Begin(&CommandBufferBeginInfo{
		Flags: COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin command buffer: %v", err)
	}

	// NV12 has two planes: Y (plane 0) and UV (plane 1)
	// Calculate plane sizes
	yPlaneSize := uint64(enc.config.Width * enc.config.Height)
	// uvPlaneSize := uint64(enc.config.Width * enc.config.Height / 2) // Interleaved UV at half vertical res

	// Transition both planes to transfer destination
	enc.commandBuffer.PipelineBarrier(
		PIPELINE_STAGE_TOP_OF_PIPE_BIT,
		PIPELINE_STAGE_TRANSFER_BIT,
		0,
		[]ImageMemoryBarrier{
			{
				SrcAccessMask:       0,
				DstAccessMask:       ACCESS_TRANSFER_WRITE_BIT,
				OldLayout:           IMAGE_LAYOUT_UNDEFINED,
				NewLayout:           IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				SrcQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				DstQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				Image:               enc.inputImage,
				SubresourceRange: ImageSubresourceRange{
					AspectMask:     IMAGE_ASPECT_PLANE_0_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
			{
				SrcAccessMask:       0,
				DstAccessMask:       ACCESS_TRANSFER_WRITE_BIT,
				OldLayout:           IMAGE_LAYOUT_UNDEFINED,
				NewLayout:           IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				SrcQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				DstQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				Image:               enc.inputImage,
				SubresourceRange: ImageSubresourceRange{
					AspectMask:     IMAGE_ASPECT_PLANE_1_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	// Copy Y plane (plane 0) - full resolution
	enc.commandBuffer.CopyBufferToImage(
		stagingBuffer,
		enc.inputImage,
		IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		[]BufferImageCopy{{
			BufferOffset:      0,
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: ImageSubresourceLayers{
				AspectMask:     IMAGE_ASPECT_PLANE_0_BIT,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: Offset3D{X: 0, Y: 0, Z: 0},
			ImageExtent: Extent3D{
				Width:  enc.config.Width,
				Height: enc.config.Height,
				Depth:  1,
			},
		}},
	)

	// Copy UV plane (plane 1) - half vertical resolution, interleaved
	enc.commandBuffer.CopyBufferToImage(
		stagingBuffer,
		enc.inputImage,
		IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
		[]BufferImageCopy{{
			BufferOffset:      yPlaneSize, // UV data starts after Y plane
			BufferRowLength:   0,
			BufferImageHeight: 0,
			ImageSubresource: ImageSubresourceLayers{
				AspectMask:     IMAGE_ASPECT_PLANE_1_BIT,
				MipLevel:       0,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
			ImageOffset: Offset3D{X: 0, Y: 0, Z: 0},
			ImageExtent: Extent3D{
				Width:  enc.config.Width / 2, // UV plane has half width (2 bytes per texel = U + V)
				Height: enc.config.Height / 2,
				Depth:  1,
			},
		}},
	)

	// Transition both planes to video encode source
	enc.commandBuffer.PipelineBarrier(
		PIPELINE_STAGE_TRANSFER_BIT,
		PIPELINE_STAGE_VIDEO_ENCODE_BIT_KHR,
		0,
		[]ImageMemoryBarrier{
			{
				SrcAccessMask:       ACCESS_TRANSFER_WRITE_BIT,
				DstAccessMask:       ACCESS_VIDEO_ENCODE_READ_BIT_KHR,
				OldLayout:           IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				NewLayout:           IMAGE_LAYOUT_VIDEO_ENCODE_SRC_KHR,
				SrcQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				DstQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				Image:               enc.inputImage,
				SubresourceRange: ImageSubresourceRange{
					AspectMask:     IMAGE_ASPECT_PLANE_0_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
			{
				SrcAccessMask:       ACCESS_TRANSFER_WRITE_BIT,
				DstAccessMask:       ACCESS_VIDEO_ENCODE_READ_BIT_KHR,
				OldLayout:           IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				NewLayout:           IMAGE_LAYOUT_VIDEO_ENCODE_SRC_KHR,
				SrcQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				DstQueueFamilyIndex: QUEUE_FAMILY_IGNORED,
				Image:               enc.inputImage,
				SubresourceRange: ImageSubresourceRange{
					AspectMask:     IMAGE_ASPECT_PLANE_1_BIT,
					BaseMipLevel:   0,
					LevelCount:     1,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
			},
		},
	)

	// Begin video coding
	enc.commandBuffer.CmdBeginVideoCodingH265KHR(
		enc.videoSession,
		enc.sessionParams,
		enc.config.Width,
		enc.config.Height,
	)

	// Reset video coding state if first frame
	if enc.frameNum == 0 {
		enc.commandBuffer.CmdControlVideoCodingKHR(&VideoCodingControlInfoKHR{
			Flags: VIDEO_CODING_CONTROL_RESET_BIT_KHR,
		})
	}

	// Encode the frame
	enc.commandBuffer.CmdEncodeVideoH265KHR(
		enc.bitstreamBuffer,
		0,
		enc.bitstreamSize,
		enc.inputImageView,
		enc.config.Width,
		enc.config.Height,
		enc.frameNum,
		isIDR,
	)

	// End video coding
	enc.commandBuffer.CmdEndVideoCodingKHR()

	// End command buffer
	err = enc.commandBuffer.End()
	if err != nil {
		return nil, fmt.Errorf("failed to end command buffer: %v", err)
	}

	// Reset fence
	enc.device.ResetFences([]Fence{enc.encodeFence})

	// Submit command buffer
	err = enc.videoQueue.Submit([]SubmitInfo{{
		CommandBuffers: []CommandBuffer{enc.commandBuffer},
	}}, enc.encodeFence)
	if err != nil {
		return nil, fmt.Errorf("failed to submit encode command: %v", err)
	}

	// Wait for encoding to complete
	enc.device.WaitForFences([]Fence{enc.encodeFence}, true, ^uint64(0))

	// Read bitstream from output buffer
	bitstreamPtr, err := enc.device.MapMemory(enc.bitstreamMemory, 0, enc.bitstreamSize)
	if err != nil {
		return nil, fmt.Errorf("failed to map bitstream memory: %v", err)
	}

	// Find actual data size (look for NAL end or use fixed size for now)
	// The encoder should provide the actual size through query pool, but for simplicity:
	bitstreamData := make([]byte, enc.bitstreamSize)
	copy(bitstreamData, (*[1 << 30]byte)(bitstreamPtr)[:enc.bitstreamSize])
	enc.device.UnmapMemory(enc.bitstreamMemory)

	// Find actual encoded data length (scan for trailing zeros)
	actualLen := len(bitstreamData)
	for actualLen > 0 && bitstreamData[actualLen-1] == 0 {
		actualLen--
	}
	if actualLen == 0 {
		return nil, fmt.Errorf("encoder produced no output")
	}

	// Build final NAL units with VPS/SPS/PPS for IDR frames
	var result []byte
	if isIDR {
		// Add VPS
		result = append(result, WriteHEVCNALUnitAVCC(HEVC_NAL_VPS, 0, enc.vps)...)
		// Add SPS
		result = append(result, WriteHEVCNALUnitAVCC(HEVC_NAL_SPS, 0, enc.sps)...)
		// Add PPS
		result = append(result, WriteHEVCNALUnitAVCC(HEVC_NAL_PPS, 0, enc.pps)...)
	}

	// Add encoded slice data
	result = append(result, bitstreamData[:actualLen]...)

	enc.frameNum++
	enc.gopFrameNum++

	return result, nil
}

// AddFrame encodes a frame and adds it to the MP4 file
func (enc *H265HardwareEncoder) AddFrame(rgbaData []byte) error {
	data, err := enc.EncodeFrame(rgbaData)
	if err != nil {
		return err
	}

	isIDR := enc.gopFrameNum == 0
	enc.mp4Writer.AddFrame(data, isIDR)

	enc.frameNum++
	enc.gopFrameNum++
	if enc.gopFrameNum >= enc.config.GOPSize {
		enc.gopFrameNum = 0
		enc.idrPicId++
	}

	return nil
}

// Finish finalizes the encoding and writes the MP4 file
func (enc *H265HardwareEncoder) Finish(filename string) error {
	return enc.mp4Writer.WriteToFile(filename)
}

// Destroy releases all Vulkan resources
func (enc *H265HardwareEncoder) Destroy() {
	if enc.encodeFence.handle != nil {
		enc.device.DestroyFence(enc.encodeFence)
	}

	if enc.bitstreamBuffer.handle != nil {
		enc.device.DestroyBuffer(enc.bitstreamBuffer)
	}
	if enc.bitstreamMemory.handle != nil {
		enc.device.FreeMemory(enc.bitstreamMemory)
	}

	if enc.sessionParams.handle != nil {
		enc.device.DestroyVideoSessionParametersKHR(enc.sessionParams)
	}

	if enc.videoSession.handle != nil {
		enc.device.DestroyVideoSessionKHR(enc.videoSession)
	}

	for _, mem := range enc.sessionMemories {
		if mem.handle != nil {
			enc.device.FreeMemory(mem)
		}
	}

	if enc.commandPool.handle != nil {
		enc.device.DestroyCommandPool(enc.commandPool)
	}

	enc.initialized = false
}
