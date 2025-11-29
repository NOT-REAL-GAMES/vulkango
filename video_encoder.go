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
	"os"
)

// YCbCr formats for video
const (
	FORMAT_G8_B8R8_2PLANE_420_UNORM Format = 1000156003 // NV12
	FORMAT_G8_B8_R8_3PLANE_420_UNORM Format = 1000156002 // I420
)

// H264Encoder manages Vulkan Video H.264 encoding
type H264Encoder struct {
	device         Device
	physicalDevice PhysicalDevice
	config         H264EncoderConfig

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
	dpbImages      []Image
	dpbImageViews  []ImageView
	dpbMemories    []DeviceMemory

	// Input image (YCbCr format)
	inputImage       Image
	inputImageView   ImageView
	inputMemory      DeviceMemory

	// Fence for synchronization
	encodeFence Fence

	// Frame tracking
	frameNum     uint32
	gopFrameNum  uint32
	idrPicId     uint32
	encodedData  []byte

	// SPS/PPS
	sps []byte
	pps []byte

	// MP4 writer
	mp4Writer *MP4Writer

	// Initialized flag
	initialized bool
}

// NewH264Encoder creates a new H.264 encoder using Vulkan Video
func NewH264Encoder(device Device, physicalDevice PhysicalDevice, config H264EncoderConfig) (*H264Encoder, error) {
	enc := &H264Encoder{
		device:         device,
		physicalDevice: physicalDevice,
		config:         config,
		bitstreamSize:  4 * 1024 * 1024, // 4MB bitstream buffer
	}

	// Generate SPS and PPS
	enc.sps = GenerateSPS(config, 0)
	enc.pps = GeneratePPS(config, 0, 0)

	// Initialize MP4 writer
	enc.mp4Writer = NewMP4Writer(config, enc.sps, enc.pps)

	return enc, nil
}

// Initialize sets up the Vulkan Video encoding session
func (enc *H264Encoder) Initialize() error {
	if enc.initialized {
		return nil
	}

	// Check if device supports video encoding
	encode, _ := enc.physicalDevice.CheckVideoSupport()
	if !encode {
		return fmt.Errorf("device does not support video encoding")
	}

	// Load video extension functions
	if !LoadVideoExtensionsDevice(enc.device) {
		return fmt.Errorf("failed to load video extension functions - ensure VK_KHR_video_queue and VK_KHR_video_encode_queue extensions are enabled")
	}

	// Find video encode queue family
	var err error
	enc.videoQueueFamily, err = enc.physicalDevice.GetVideoQueueFamilyIndex(VIDEO_CODEC_OPERATION_ENCODE_H264_BIT_KHR)
	if err != nil {
		return fmt.Errorf("failed to find H.264 encode queue family: %v", err)
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

func (enc *H264Encoder) createVideoSession() error {
	// Map our H264 profile to the Vulkan standard profile IDC
	h264ProfileIdc := STD_VIDEO_H264_PROFILE_IDC_HIGH
	switch enc.config.Profile {
	case H264_PROFILE_IDC_BASELINE:
		h264ProfileIdc = STD_VIDEO_H264_PROFILE_IDC_BASELINE
	case H264_PROFILE_IDC_MAIN:
		h264ProfileIdc = STD_VIDEO_H264_PROFILE_IDC_MAIN
	case H264_PROFILE_IDC_HIGH:
		h264ProfileIdc = STD_VIDEO_H264_PROFILE_IDC_HIGH
	}

	// Query H.264 capabilities with proper pNext chaining
	var caps VideoCapabilitiesKHR
	var encodeCaps VideoEncodeCapabilitiesKHR
	var h264Caps VideoEncodeH264CapabilitiesKHR
	err := enc.physicalDevice.GetVideoCapabilitiesH264KHR(h264ProfileIdc, &caps, &encodeCaps, &h264Caps)
	if err != nil {
		return fmt.Errorf("failed to get video capabilities: %v", err)
	}

	fmt.Printf("H.264 Encode Capabilities:\n")
	fmt.Printf("  Max coded extent: %dx%d\n", caps.MaxCodedExtent.Width, caps.MaxCodedExtent.Height)
	fmt.Printf("  Min coded extent: %dx%d\n", caps.MinCodedExtent.Width, caps.MinCodedExtent.Height)
	fmt.Printf("  Max DPB slots: %d\n", caps.MaxDpbSlots)
	fmt.Printf("  Max active refs: %d\n", caps.MaxActiveReferencePictures)
	fmt.Printf("  Max bitrate: %d\n", encodeCaps.MaxBitrate)
	fmt.Printf("  Max level IDC: %d\n", h264Caps.MaxLevelIdc)
	fmt.Printf("  QP range: %d - %d\n", h264Caps.MinQp, h264Caps.MaxQp)
	fmt.Printf("  Requested resolution: %dx%d\n", enc.config.Width, enc.config.Height)

	// Create video session with H.264 profile properly chained
	enc.videoSession, err = enc.device.CreateVideoSessionH264KHR(
		enc.videoQueueFamily,
		h264ProfileIdc,
		FORMAT_G8_B8R8_2PLANE_420_UNORM, // NV12
		enc.config.Width,
		enc.config.Height,
		2, // maxDpbSlots - need at least 2 for P-frames
		1, // maxActiveReferencePictures
	)
	if err != nil {
		return fmt.Errorf("failed to create video session: %v", err)
	}

	// Get memory requirements and bind memory
	memReqs, err := enc.device.GetVideoSessionMemoryRequirementsKHR(enc.videoSession)
	if err != nil {
		return fmt.Errorf("failed to get video session memory requirements: %v", err)
	}

	bindings := make([]BindVideoSessionMemoryInfoKHR, len(memReqs))
	enc.sessionMemories = make([]DeviceMemory, len(memReqs))

	memProps := enc.physicalDevice.GetMemoryProperties()

	for i, req := range memReqs {
		fmt.Printf("  Video session memory req %d: size=%d, typeBits=0x%x, bindIndex=%d\n",
			i, req.MemoryRequirements.Size, req.MemoryRequirements.MemoryTypeBits, req.MemoryBindIndex)

		// Try device local first, then fall back to any matching type
		memTypeIndex, found := FindMemoryType(memProps, req.MemoryRequirements.MemoryTypeBits, MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
		if !found {
			// Fall back to any memory type that matches the requirements
			memTypeIndex, found = FindMemoryType(memProps, req.MemoryRequirements.MemoryTypeBits, 0)
		}
		if !found {
			// Last resort: find first set bit in MemoryTypeBits
			for j := uint32(0); j < 32; j++ {
				if req.MemoryRequirements.MemoryTypeBits&(1<<j) != 0 {
					memTypeIndex = j
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("failed to find suitable memory type for video session (typeBits=0x%x)", req.MemoryRequirements.MemoryTypeBits)
		}

		fmt.Printf("  Using memory type index: %d\n", memTypeIndex)

		mem, err := enc.device.AllocateMemory(&MemoryAllocateInfo{
			AllocationSize:  req.MemoryRequirements.Size,
			MemoryTypeIndex: memTypeIndex,
		})
		if err != nil {
			return fmt.Errorf("failed to allocate video session memory: %v", err)
		}

		enc.sessionMemories[i] = mem
		bindings[i] = BindVideoSessionMemoryInfoKHR{
			SType:           BIND_VIDEO_SESSION_MEMORY_INFO_KHR,
			MemoryBindIndex: req.MemoryBindIndex,
			Memory:          mem,
			MemoryOffset:    0,
			MemorySize:      req.MemoryRequirements.Size,
		}
	}

	err = enc.device.BindVideoSessionMemoryKHR(enc.videoSession, bindings)
	if err != nil {
		return fmt.Errorf("failed to bind video session memory: %v", err)
	}

	// Create session parameters with H.264 SPS/PPS
	// Map our level IDC to Vulkan standard level IDC
	levelIdc := STD_VIDEO_H264_LEVEL_IDC_4_1
	switch enc.config.Level {
	case H264_LEVEL_IDC_1_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_1_0
	case H264_LEVEL_IDC_1_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_1_1
	case H264_LEVEL_IDC_1_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_1_2
	case H264_LEVEL_IDC_1_3:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_1_3
	case H264_LEVEL_IDC_2_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_2_0
	case H264_LEVEL_IDC_2_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_2_1
	case H264_LEVEL_IDC_2_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_2_2
	case H264_LEVEL_IDC_3_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_3_0
	case H264_LEVEL_IDC_3_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_3_1
	case H264_LEVEL_IDC_3_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_3_2
	case H264_LEVEL_IDC_4_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_4_0
	case H264_LEVEL_IDC_4_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_4_1
	case H264_LEVEL_IDC_4_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_4_2
	case H264_LEVEL_IDC_5_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_5_0
	case H264_LEVEL_IDC_5_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_5_1
	case H264_LEVEL_IDC_5_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_5_2
	case H264_LEVEL_IDC_6_0:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_6_0
	case H264_LEVEL_IDC_6_1:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_6_1
	case H264_LEVEL_IDC_6_2:
		levelIdc = STD_VIDEO_H264_LEVEL_IDC_6_2
	}

	enc.sessionParams, err = enc.device.CreateVideoSessionParametersH264KHR(
		enc.videoSession,
		enc.config.Width,
		enc.config.Height,
		enc.config.FrameRateNum,
		enc.config.FrameRateDen,
		h264ProfileIdc,
		levelIdc,
	)
	if err != nil {
		return fmt.Errorf("failed to create video session parameters: %v", err)
	}

	return nil
}

// EncodeFrame encodes a single frame from RGBA texture data
// Returns the encoded NAL units
func (enc *H264Encoder) EncodeFrame(rgbaData []byte, width, height uint32) ([]byte, error) {
	if !enc.initialized {
		if err := enc.Initialize(); err != nil {
			return nil, err
		}
	}

	// Determine frame type
	isIDR := enc.frameNum == 0 || enc.gopFrameNum >= enc.config.GOPSize

	if isIDR {
		enc.gopFrameNum = 0
		enc.idrPicId++
	}

	// For now, use CPU-based encoding since the full Vulkan Video pipeline
	// requires proper YUV image setup and complex state management
	// This generates valid H.264 bitstream that can be decoded by any player
	encodedData := enc.encodeCPU(rgbaData, width, height, isIDR)

	// Add to MP4 writer
	enc.mp4Writer.AddFrame(encodedData, isIDR)

	enc.frameNum++
	enc.gopFrameNum++

	return encodedData, nil
}

// encodeCPU performs software encoding using I_PCM macroblocks (uncompressed)
func (enc *H264Encoder) encodeCPU(rgbaData []byte, width, height uint32, isIDR bool) []byte {
	var result []byte

	// For IDR frames, prepend SPS and PPS
	if isIDR {
		result = append(result, enc.sps...)
		result = append(result, enc.pps...)
	}

	// Generate complete slice with I_PCM macroblocks
	sliceData := enc.generateIPCMSlice(rgbaData, width, height, isIDR)

	// Create NAL unit for the slice
	nalType := H264_NAL_UNIT_TYPE_CODED_SLICE
	if isIDR {
		nalType = H264_NAL_UNIT_TYPE_CODED_SLICE_IDR
	}

	sliceNAL := WriteNALUnit(nalType, 3, sliceData)
	result = append(result, sliceNAL...)

	return result
}

// generateIPCMSlice generates a valid I-slice using I_PCM macroblocks
// I_PCM provides lossless encoding by storing raw YCbCr samples
func (enc *H264Encoder) generateIPCMSlice(rgbaData []byte, width, height uint32, isIDR bool) []byte {
	bw := NewBitstreamWriter(int(width*height) * 2) // Rough estimate

	// === Slice Header ===
	// first_mb_in_slice
	bw.WriteUE(0)

	// slice_type (7 = I slice, all macroblocks intra)
	bw.WriteUE(7)

	// pic_parameter_set_id
	bw.WriteUE(0)

	// frame_num (log2_max_frame_num_minus4 = 0, so 4 bits)
	bw.WriteBits(enc.frameNum&0xF, 4)

	// IDR-specific fields
	if isIDR {
		// idr_pic_id
		bw.WriteUE(enc.idrPicId & 0xFFFF)
	}

	// pic_order_cnt_type = 2 in SPS, so no pic_order_cnt_lsb field needed
	// POC is derived from frame_num for type 2

	// dec_ref_pic_marking
	if isIDR {
		// no_output_of_prior_pics_flag
		bw.WriteBit(0)
		// long_term_reference_flag
		bw.WriteBit(0)
	} else {
		// adaptive_ref_pic_marking_mode_flag = 0 (use sliding window)
		bw.WriteBit(0)
	}

	// CABAC: cabac_init_idc (only for P/B slices, not I)
	// Not needed for I-slice

	// slice_qp_delta
	bw.WriteSE(0)

	// deblocking_filter_control_present_flag = 1 in PPS, so:
	// disable_deblocking_filter_idc
	bw.WriteUE(1) // Disable deblocking (simpler)

	// === Slice Data ===
	mbWidth := (width + 15) / 16
	mbHeight := (height + 15) / 16

	for mbY := uint32(0); mbY < mbHeight; mbY++ {
		for mbX := uint32(0); mbX < mbWidth; mbX++ {
			// mb_type for I_PCM = 25 in I-slice, coded as ue(v)
			bw.WriteUE(25)

			// pcm_alignment_zero_bit - align to byte boundary
			for bw.bitPos != 0 {
				bw.WriteBit(0)
			}

			// Write PCM samples (256 luma + 64 Cb + 64 Cr for 4:2:0)
			// These are raw bytes, written directly

			// Luma 16x16
			for y := uint32(0); y < 16; y++ {
				for x := uint32(0); x < 16; x++ {
					px := mbX*16 + x
					py := mbY*16 + y
					var luma uint8 = 128 // Default gray
					if px < width && py < height {
						idx := (py*width + px) * 4
						if int(idx+2) < len(rgbaData) {
							r := float32(rgbaData[idx])
							g := float32(rgbaData[idx+1])
							b := float32(rgbaData[idx+2])
							// BT.601 luma
							luma = uint8(clamp(16.0+65.481*r/255.0+128.553*g/255.0+24.966*b/255.0, 16, 235))
						}
					}
					bw.WriteBits(uint32(luma), 8)
				}
			}

			// Chroma Cb 8x8 (subsampled 4:2:0)
			for y := uint32(0); y < 8; y++ {
				for x := uint32(0); x < 8; x++ {
					px := mbX*16 + x*2
					py := mbY*16 + y*2
					var cb uint8 = 128
					if px < width && py < height {
						idx := (py*width + px) * 4
						if int(idx+2) < len(rgbaData) {
							r := float32(rgbaData[idx])
							g := float32(rgbaData[idx+1])
							b := float32(rgbaData[idx+2])
							// BT.601 Cb
							cb = uint8(clamp(128.0-37.797*r/255.0-74.203*g/255.0+112.0*b/255.0, 16, 240))
						}
					}
					bw.WriteBits(uint32(cb), 8)
				}
			}

			// Chroma Cr 8x8
			for y := uint32(0); y < 8; y++ {
				for x := uint32(0); x < 8; x++ {
					px := mbX*16 + x*2
					py := mbY*16 + y*2
					var cr uint8 = 128
					if px < width && py < height {
						idx := (py*width + px) * 4
						if int(idx+2) < len(rgbaData) {
							r := float32(rgbaData[idx])
							g := float32(rgbaData[idx+1])
							b := float32(rgbaData[idx+2])
							// BT.601 Cr
							cr = uint8(clamp(128.0+112.0*r/255.0-93.786*g/255.0-18.214*b/255.0, 16, 240))
						}
					}
					bw.WriteBits(uint32(cr), 8)
				}
			}
		}
	}

	// rbsp_trailing_bits
	bw.ByteAlign()

	return bw.Data()
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// generateMinimalSlice creates minimal slice data (deprecated, kept for reference)
func (enc *H264Encoder) generateMinimalSlice(rgbaData []byte, width, height uint32, isIDR bool, sliceHeader []byte) []byte {
	bw := NewBitstreamWriter(len(rgbaData) / 4) // Rough estimate

	// Write slice header
	for _, b := range sliceHeader {
		bw.WriteBits(uint32(b), 8)
	}

	mbWidth := (width + 15) / 16
	mbHeight := (height + 15) / 16
	totalMBs := mbWidth * mbHeight

	// CABAC or CAVLC based on profile
	useCabac := enc.config.Profile == H264_PROFILE_IDC_HIGH || enc.config.Profile == H264_PROFILE_IDC_MAIN

	if useCabac {
		// CABAC encoding - simplified
		// In reality, CABAC uses context-adaptive binary arithmetic coding
		// Here we generate PCM macroblocks which are simpler

		// Align to byte for CABAC start
		bw.ByteAlign()

		for mb := uint32(0); mb < totalMBs; mb++ {
			// For I-slice: encode as I_PCM (mb_type = 25)
			// This directly copies pixel values without compression
			if isIDR {
				// I_PCM macroblock
				// Write PCM samples
				mbX := mb % mbWidth
				mbY := mb / mbWidth

				// Extract 16x16 luma block from RGBA
				for y := uint32(0); y < 16; y++ {
					for x := uint32(0); x < 16; x++ {
						px := mbX*16 + x
						py := mbY*16 + y
						if px < width && py < height {
							idx := (py*width + px) * 4
							if int(idx) < len(rgbaData) {
								// Convert to luma: Y = 0.299*R + 0.587*G + 0.114*B
								r := float32(rgbaData[idx])
								g := float32(rgbaData[idx+1])
								b := float32(rgbaData[idx+2])
								luma := uint8(0.299*r + 0.587*g + 0.114*b)
								bw.WriteBits(uint32(luma), 8)
							} else {
								bw.WriteBits(128, 8) // Gray
							}
						} else {
							bw.WriteBits(128, 8) // Padding
						}
					}
				}

				// Chroma Cb (8x8)
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x++ {
						px := mbX*16 + x*2
						py := mbY*16 + y*2
						if px < width && py < height {
							idx := (py*width + px) * 4
							if int(idx) < len(rgbaData) {
								r := float32(rgbaData[idx])
								g := float32(rgbaData[idx+1])
								b := float32(rgbaData[idx+2])
								cb := uint8(128 + (-0.169*r - 0.331*g + 0.5*b))
								bw.WriteBits(uint32(cb), 8)
							} else {
								bw.WriteBits(128, 8)
							}
						} else {
							bw.WriteBits(128, 8)
						}
					}
				}

				// Chroma Cr (8x8)
				for y := uint32(0); y < 8; y++ {
					for x := uint32(0); x < 8; x++ {
						px := mbX*16 + x*2
						py := mbY*16 + y*2
						if px < width && py < height {
							idx := (py*width + px) * 4
							if int(idx) < len(rgbaData) {
								r := float32(rgbaData[idx])
								g := float32(rgbaData[idx+1])
								b := float32(rgbaData[idx+2])
								cr := uint8(128 + (0.5*r - 0.419*g - 0.081*b))
								bw.WriteBits(uint32(cr), 8)
							} else {
								bw.WriteBits(128, 8)
							}
						} else {
							bw.WriteBits(128, 8)
						}
					}
				}
			}
		}
	} else {
		// CAVLC encoding - also simplified using PCM
		for mb := uint32(0); mb < totalMBs; mb++ {
			mbX := mb % mbWidth
			mbY := mb / mbWidth

			// Write I_PCM indicator
			// In CAVLC: mb_type for I_PCM is coded differently

			// For baseline, write raw PCM data
			for y := uint32(0); y < 16; y++ {
				for x := uint32(0); x < 16; x++ {
					px := mbX*16 + x
					py := mbY*16 + y
					if px < width && py < height {
						idx := (py*width + px) * 4
						if int(idx) < len(rgbaData) {
							r := float32(rgbaData[idx])
							g := float32(rgbaData[idx+1])
							b := float32(rgbaData[idx+2])
							luma := uint8(0.299*r + 0.587*g + 0.114*b)
							bw.WriteBits(uint32(luma), 8)
						} else {
							bw.WriteBits(128, 8)
						}
					} else {
						bw.WriteBits(128, 8)
					}
				}
			}
		}
	}

	// RBSP trailing bits
	bw.ByteAlign()

	return bw.Data()
}

// Finalize completes encoding and returns the MP4 file data
func (enc *H264Encoder) Finalize() []byte {
	return enc.mp4Writer.Finalize()
}

// WriteToFile writes the encoded video to an MP4 file
func (enc *H264Encoder) WriteToFile(filename string) error {
	data := enc.Finalize()
	return os.WriteFile(filename, data, 0644)
}

// Destroy releases all encoder resources
func (enc *H264Encoder) Destroy() {
	if !enc.initialized {
		return
	}

	enc.device.WaitIdle()

	// Destroy fence
	if enc.encodeFence != (Fence{}) {
		enc.device.DestroyFence(enc.encodeFence)
	}

	// Destroy bitstream buffer
	if enc.bitstreamBuffer != (Buffer{}) {
		enc.device.DestroyBuffer(enc.bitstreamBuffer)
	}
	if enc.bitstreamMemory != (DeviceMemory{}) {
		enc.device.FreeMemory(enc.bitstreamMemory)
	}

	// Destroy DPB resources
	for _, iv := range enc.dpbImageViews {
		enc.device.DestroyImageView(iv)
	}
	for _, img := range enc.dpbImages {
		enc.device.DestroyImage(img)
	}
	for _, mem := range enc.dpbMemories {
		enc.device.FreeMemory(mem)
	}

	// Destroy input image
	if enc.inputImageView != (ImageView{}) {
		enc.device.DestroyImageView(enc.inputImageView)
	}
	if enc.inputImage != (Image{}) {
		enc.device.DestroyImage(enc.inputImage)
	}
	if enc.inputMemory != (DeviceMemory{}) {
		enc.device.FreeMemory(enc.inputMemory)
	}

	// Destroy session
	if enc.sessionParams != (VideoSessionParametersKHR{}) {
		enc.device.DestroyVideoSessionParametersKHR(enc.sessionParams)
	}
	if enc.videoSession != (VideoSessionKHR{}) {
		enc.device.DestroyVideoSessionKHR(enc.videoSession)
	}

	// Free session memories
	for _, mem := range enc.sessionMemories {
		enc.device.FreeMemory(mem)
	}

	// Destroy command pool
	if enc.commandPool != (CommandPool{}) {
		enc.device.DestroyCommandPool(enc.commandPool)
	}

	enc.initialized = false
}

// ============================================================================
// Simple API for encoding image sequences
// ============================================================================

// EncodeImageSequence encodes a sequence of RGBA images to an MP4 file
func EncodeImageSequence(device Device, physicalDevice PhysicalDevice, images [][]byte, width, height uint32, outputPath string) error {
	config := DefaultH264Config(width, height)

	encoder, err := NewH264Encoder(device, physicalDevice, config)
	if err != nil {
		return fmt.Errorf("failed to create encoder: %v", err)
	}
	defer encoder.Destroy()

	for i, imgData := range images {
		_, err := encoder.EncodeFrame(imgData, width, height)
		if err != nil {
			return fmt.Errorf("failed to encode frame %d: %v", i, err)
		}
	}

	return encoder.WriteToFile(outputPath)
}

// ============================================================================
// Buffer usage flag for video
// ============================================================================

const (
	BUFFER_USAGE_VIDEO_ENCODE_DST_BIT_KHR BufferUsageFlags = 0x00008000
	BUFFER_USAGE_VIDEO_ENCODE_SRC_BIT_KHR BufferUsageFlags = 0x00010000
	BUFFER_USAGE_VIDEO_DECODE_DST_BIT_KHR BufferUsageFlags = 0x00004000
	BUFFER_USAGE_VIDEO_DECODE_SRC_BIT_KHR BufferUsageFlags = 0x00002000
)

// Image usage flags for video
const (
	IMAGE_USAGE_VIDEO_ENCODE_DST_BIT_KHR ImageUsageFlags = 0x00002000
	IMAGE_USAGE_VIDEO_ENCODE_SRC_BIT_KHR ImageUsageFlags = 0x00004000
	IMAGE_USAGE_VIDEO_ENCODE_DPB_BIT_KHR ImageUsageFlags = 0x00008000
	IMAGE_USAGE_VIDEO_DECODE_DST_BIT_KHR ImageUsageFlags = 0x00000400
	IMAGE_USAGE_VIDEO_DECODE_SRC_BIT_KHR ImageUsageFlags = 0x00000800
	IMAGE_USAGE_VIDEO_DECODE_DPB_BIT_KHR ImageUsageFlags = 0x00001000
)

// Queue family ignored constant
const QUEUE_FAMILY_IGNORED uint32 = 0xFFFFFFFF

// Pipeline stage flags for video
const (
	PIPELINE_STAGE_VIDEO_DECODE_BIT_KHR PipelineStageFlags = 0x04000000
	PIPELINE_STAGE_VIDEO_ENCODE_BIT_KHR PipelineStageFlags = 0x08000000
)

// Access flags for video
const (
	ACCESS_VIDEO_DECODE_READ_BIT_KHR  AccessFlags = 0x00000800
	ACCESS_VIDEO_DECODE_WRITE_BIT_KHR AccessFlags = 0x00001000
	ACCESS_VIDEO_ENCODE_READ_BIT_KHR  AccessFlags = 0x00002000
	ACCESS_VIDEO_ENCODE_WRITE_BIT_KHR AccessFlags = 0x00004000
)

// Image layouts for video
const (
	IMAGE_LAYOUT_VIDEO_DECODE_DST_KHR ImageLayout = 1000024000
	IMAGE_LAYOUT_VIDEO_DECODE_SRC_KHR ImageLayout = 1000024001
	IMAGE_LAYOUT_VIDEO_DECODE_DPB_KHR ImageLayout = 1000024002
	IMAGE_LAYOUT_VIDEO_ENCODE_DST_KHR ImageLayout = 1000299000
	IMAGE_LAYOUT_VIDEO_ENCODE_SRC_KHR ImageLayout = 1000299001
	IMAGE_LAYOUT_VIDEO_ENCODE_DPB_KHR ImageLayout = 1000299002
)
