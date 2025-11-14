package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"os"
	"runtime"
	"time"
	"unsafe"

	sdl "github.com/NOT-REAL-GAMES/sdl3go"
	vk "github.com/NOT-REAL-GAMES/vulkango"
	shaderc "github.com/NOT-REAL-GAMES/vulkango/shaderc"
)

const vertexShader = `
#version 450

layout(location = 0) in vec2 inPosition;
layout(location = 1) in vec3 inColor;
layout(location = 2) in vec2 inTexCoord;

layout(location = 0) out vec3 fragColor;
layout(location = 1) out vec2 fragTexCoord;

void main() {
    gl_Position = vec4(inPosition, 0.0, 1.0);
    fragColor = inColor;
    fragTexCoord = inTexCoord;
}
`

const fragmentShader = `
#version 450

layout(location = 0) in vec3 fragColor;
layout(location = 1) in vec2 fragTexCoord;

layout(binding = 0) uniform sampler2D texSampler;

layout(location = 0) out vec4 outColor;

void main() {
    vec4 texColor = texture(texSampler, fragTexCoord);
    outColor = texColor * vec4(fragColor, 1.0);
}
`

type Vertex struct {
	Pos      [2]float32
	Color    [3]float32
	TexCoord [2]float32
}

type ImageData struct {
	Width  uint32
	Height uint32
	Pixels []byte
}

func LoadImage(path string) (*ImageData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %v", err)
	}
	defer file.Close()

	// Decode image (format auto-detected)
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	fmt.Printf("Loaded %s image: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())

	// Convert to RGBA
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	return &ImageData{
		Width:  uint32(bounds.Dx()),
		Height: uint32(bounds.Dy()),
		Pixels: rgba.Pix, // Raw RGBA bytes
	}, nil
}

func main() {

	// Comment out if GTK won't initialize.
	if runtime.GOOS == "linux" {
		os.Setenv("SDL_VIDEODRIVER", "X11")
	}

	// Initialize SDL first
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Example", 960, 960, sdl.WINDOW_VULKAN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// Get required extensions from SDL BEFORE creating instance
	exts, err := sdl.VulkanGetInstanceExtensions()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Required Vulkan extensions: %v\n", exts)

	// Create Vulkan instance WITH the required extensions
	version, _ := vk.EnumerateInstanceVersion()
	fmt.Printf("Vulkan %d.%d.%d\n",
		vk.ApiVersionMajor(version),
		vk.ApiVersionMinor(version),
		vk.ApiVersionPatch(version))

	instance, err := vk.CreateInstance(&vk.InstanceCreateInfo{
		ApplicationInfo: &vk.ApplicationInfo{
			ApplicationName:    "Example Application",
			ApplicationVersion: vk.MakeApiVersion(0, 1, 0, 0),
			EngineName:         "Example Engine",
			EngineVersion:      vk.MakeApiVersion(0, 1, 0, 0),
			ApiVersion:         vk.ApiVersion_1_4,
		},
		EnabledExtensionNames: exts,
	})
	if err != nil {
		panic(err)
	}
	defer instance.Destroy()

	devices, _ := instance.EnumeratePhysicalDevices()
	fmt.Printf("Found %d physical devices\n", len(devices))

	// Create the surface
	surfHandle, err := window.VulkanCreateSurface(instance.Handle())
	if err != nil {
		panic(err)
	}
	surface := vk.NewSurfaceKHR(surfHandle)
	fmt.Printf("Created Vulkan surface: %v\n", surface)

	// Test surface queries with first physical device
	if len(devices) > 0 {
		device := devices[0]

		// Query surface capabilities
		caps, err := device.GetSurfaceCapabilitiesKHR(surface)
		if err != nil {
			panic(err)
		}
		fmt.Printf("\nSurface Capabilities:\n")
		fmt.Printf("  Min images: %d, Max images: %d\n", caps.MinImageCount, caps.MaxImageCount)
		fmt.Printf("  Current extent: %dx%d\n", caps.CurrentExtent.Width, caps.CurrentExtent.Height)

		// Query surface formats
		formats, err := device.GetSurfaceFormatsKHR(surface)
		if err != nil {
			panic(err)
		}
		fmt.Printf("\nAvailable formats: %d\n", len(formats))
		for i, format := range formats {
			if i < 3 { // Just print first 3
				fmt.Printf("  Format: %d, ColorSpace: %d\n", format.Format, format.ColorSpace)
			}
		}

		// Query present modes
		modes, err := device.GetSurfacePresentModesKHR(surface)
		if err != nil {
			panic(err)
		}
		fmt.Printf("\nAvailable present modes: %d\n", len(modes))
		for _, mode := range modes {
			fmt.Printf("  Mode: %d\n", mode)
		}
	}

	if len(devices) > 0 {
		physicalDevice := devices[0]

		// Find a graphics queue family that supports presentation
		queueFamilies := physicalDevice.GetQueueFamilyProperties()
		fmt.Printf("\nQueue families: %d\n", len(queueFamilies))

		graphicsFamily := -1
		for i, family := range queueFamilies {
			fmt.Printf("  Family %d: queues=%d, flags=%d\n", i, family.QueueCount, family.QueueFlags)

			// Check if supports graphics
			if family.QueueFlags&vk.QUEUE_GRAPHICS_BIT != 0 {
				// Check if supports presentation
				supported, _ := physicalDevice.GetSurfaceSupportKHR(uint32(i), surface)
				if supported && graphicsFamily == -1 {
					graphicsFamily = i
				}
			}
		}

		if graphicsFamily == -1 {
			panic("No suitable queue family found!")
		}

		fmt.Printf("\nUsing queue family %d for graphics\n", graphicsFamily)

		// Create device
		device, err := physicalDevice.CreateDevice(&vk.DeviceCreateInfo{
			QueueCreateInfos: []vk.DeviceQueueCreateInfo{
				{
					QueueFamilyIndex: uint32(graphicsFamily),
					QueuePriorities:  []float32{1.0},
				},
			},
			EnabledExtensionNames: []string{"VK_KHR_swapchain"},
			Vulkan13Features: &vk.PhysicalDeviceVulkan13Features{
				DynamicRendering: true, // ENABLE IT!
			},
		})

		if err != nil {
			panic(err)
		}
		defer device.Destroy()

		fmt.Println("Logical device created!")

		queue := device.GetQueue(uint32(graphicsFamily), 0)
		fmt.Printf("Got queue: %v\n", queue)

		// Create swapchain
		swapchain, swapFormat, swapExtent, err := vk.CreateSwapchain(
			device,
			physicalDevice,
			surface,
			960, 960, // window dimensions
			uint32(graphicsFamily),
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroySwapchainKHR(swapchain)

		fmt.Printf("\nSwapchain created!\n")
		fmt.Printf("  Format: %d\n", swapFormat)
		fmt.Printf("  Extent: %dx%d\n", swapExtent.Width, swapExtent.Height)

		// Get swapchain images
		swapImages, err := device.GetSwapchainImagesKHR(swapchain)
		if err != nil {
			panic(err)
		}
		fmt.Printf("  Images: %d\n", len(swapImages))

		swapImageViews, err := vk.CreateSwapchainImageViews(device, swapImages, swapFormat)
		if err != nil {
			panic(err)
		}
		defer func() {
			for _, view := range swapImageViews {
				device.DestroyImageView(view)
			}
		}()

		fmt.Printf("  Image views: %d\n", len(swapImageViews))

		compiler := shaderc.NewCompiler()
		defer compiler.Release()

		options := shaderc.NewCompileOptions()
		defer options.Release()
		options.SetTargetEnv(shaderc.TargetEnvVulkan, shaderc.EnvVersionVulkan_1_3)
		options.SetOptimizationLevel(shaderc.OptimizationLevelPerformance)

		vertResult, err := compiler.CompileIntoSPV(vertexShader, "shader.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(err)
		}
		defer vertResult.Release()

		vertModule, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{
			Code: vertResult.GetBytes(),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(vertModule)

		// Compile fragment shader
		fragResult, err := compiler.CompileIntoSPV(fragmentShader, "shader.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(err)
		}
		defer fragResult.Release()

		fragModule, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{
			Code: fragResult.GetBytes(),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(fragModule)

		fmt.Println("\nShaders compiled and modules created!")

		descriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
			Bindings: []vk.DescriptorSetLayoutBinding{
				{
					Binding:         0,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: 1,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorSetLayout(descriptorSetLayout)

		// After creating shader modules
		pipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			SetLayouts: []vk.DescriptorSetLayout{descriptorSetLayout},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(pipelineLayout)

		pipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: vertModule, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: fragModule, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				VertexBindingDescriptions: []vk.VertexInputBindingDescription{
					{
						Binding:   0,
						Stride:    uint32(unsafe.Sizeof(Vertex{})),
						InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
					},
				},
				VertexAttributeDescriptions: []vk.VertexInputAttributeDescription{
					{Location: 0, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 0},    // Position
					{Location: 1, Binding: 0, Format: vk.FORMAT_R32G32B32_SFLOAT, Offset: 8}, // Color
					{Location: 2, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 20},   // TexCoord
				},
			}, InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
				Topology: vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
			},
			ViewportState: &vk.PipelineViewportStateCreateInfo{
				Viewports: []vk.Viewport{},
				Scissors:  []vk.Rect2D{},
			},
			RasterizationState: &vk.PipelineRasterizationStateCreateInfo{
				PolygonMode: vk.POLYGON_MODE_FILL,
				CullMode:    vk.CULL_MODE_NONE,
				FrontFace:   vk.FRONT_FACE_COUNTER_CLOCKWISE,
				LineWidth:   1.0,
			},
			MultisampleState: &vk.PipelineMultisampleStateCreateInfo{
				RasterizationSamples: vk.SAMPLE_COUNT_1_BIT,
			},
			ColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
				Attachments: []vk.PipelineColorBlendAttachmentState{
					{
						BlendEnable:    false,
						ColorWriteMask: vk.COLOR_COMPONENT_ALL,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{
					vk.DYNAMIC_STATE_VIEWPORT,
					vk.DYNAMIC_STATE_SCISSOR,
				},
			},
			Layout: pipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{swapFormat},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(pipeline)

		fmt.Println("Graphics pipeline created!")

		// Quad vertices (2 triangles)
		vertices := []Vertex{
			{Pos: [2]float32{-0.5, -0.5}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{0.0, 0.0}}, // Bottom-left
			{Pos: [2]float32{0.5, -0.5}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{1.0, 0.0}},  // Bottom-right
			{Pos: [2]float32{0.5, 0.5}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{1.0, 1.0}},   // Top-right
			{Pos: [2]float32{-0.5, 0.5}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{0.0, 1.0}},  // Top-left
		}
		indices := []uint16{0, 1, 2, 2, 3, 0}

		// Create vertex buffer
		vertexBufferSize := uint64(len(vertices) * int(unsafe.Sizeof(Vertex{})))
		vertexBuffer, vertexMemory, err := device.CreateBufferWithMemory(
			vertexBufferSize,
			vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(vertexBuffer)
		defer device.FreeMemory(vertexMemory)

		// Upload vertex data
		vertexData := (*[1 << 30]byte)(unsafe.Pointer(&vertices[0]))[:vertexBufferSize:vertexBufferSize]
		err = device.UploadToBuffer(vertexMemory, vertexData)
		if err != nil {
			panic(err)
		}

		// Create index buffer
		indexBufferSize := uint64(len(indices) * 2) // uint16 = 2 bytes
		indexBuffer, indexMemory, err := device.CreateBufferWithMemory(
			indexBufferSize,
			vk.BUFFER_USAGE_INDEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(indexBuffer)
		defer device.FreeMemory(indexMemory)

		// Upload index data
		indexData := (*[1 << 30]byte)(unsafe.Pointer(&indices[0]))[:indexBufferSize:indexBufferSize]
		err = device.UploadToBuffer(indexMemory, indexData)
		if err != nil {
			panic(err)
		}

		fmt.Println("Vertex and index buffers created!")

		// Create command pool
		commandPool, err := device.CreateCommandPool(&vk.CommandPoolCreateInfo{
			Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
			QueueFamilyIndex: uint32(graphicsFamily),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyCommandPool(commandPool)

		fmt.Println("\nCreating texture...")

		imageData, err := LoadImage("djungelskog.jpg") // Put a PNG in your project folder
		if err != nil {
			panic(err)
		}

		textureWidth := imageData.Width
		textureHeight := imageData.Height
		textureData := imageData.Pixels
		textureSize := uint64(len(textureData))

		fmt.Printf("Loaded texture: %dx%d (%d bytes)\n", textureWidth, textureHeight, textureSize)

		// Create staging buffer
		stagingBuffer, stagingMemory, err := device.CreateBufferWithMemory(
			textureSize,
			vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}

		// Upload texture data to staging buffer
		err = device.UploadToBuffer(stagingMemory, textureData)
		if err != nil {
			panic(err)
		}

		// Create texture image
		textureImage, textureMemory, err := device.CreateImageWithMemory(
			textureWidth, textureHeight,
			vk.FORMAT_R8G8B8A8_SRGB,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyImage(textureImage)
		defer device.FreeMemory(textureMemory)

		// Create command buffer for texture upload
		uploadCmdBuffer, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
			CommandPool:        commandPool,
			Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
			CommandBufferCount: 1,
		})
		if err != nil {
			panic(err)
		}
		uploadCmd := uploadCmdBuffer[0]

		// Record upload commands
		uploadCmd.Begin(&vk.CommandBufferBeginInfo{
			Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
		})

		// Transition image to transfer dst
		uploadCmd.PipelineBarrier(
			vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
			vk.PIPELINE_STAGE_TRANSFER_BIT,
			0,
			[]vk.ImageMemoryBarrier{
				{
					SrcAccessMask:       vk.ACCESS_NONE,
					DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               textureImage,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				},
			},
		)

		// Copy buffer to image
		uploadCmd.CopyBufferToImage(stagingBuffer, textureImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{
			{
				BufferOffset:      0,
				BufferRowLength:   0,
				BufferImageHeight: 0,
				ImageSubresource: vk.ImageSubresourceLayers{
					AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
					MipLevel:       0,
					BaseArrayLayer: 0,
					LayerCount:     1,
				},
				ImageOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
				ImageExtent: vk.Extent3D{Width: textureWidth, Height: textureHeight, Depth: 1},
			},
		})

		// Transition image to shader read
		uploadCmd.PipelineBarrier(
			vk.PIPELINE_STAGE_TRANSFER_BIT,
			vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
			0,
			[]vk.ImageMemoryBarrier{
				{
					SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               textureImage,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				},
			},
		)

		uploadCmd.End()

		// Submit and wait
		queue.Submit([]vk.SubmitInfo{
			{CommandBuffers: []vk.CommandBuffer{uploadCmd}},
		}, vk.Fence{})
		queue.WaitIdle()

		// Clean up staging buffer
		device.DestroyBuffer(stagingBuffer)
		device.FreeMemory(stagingMemory)

		// Create texture image view
		textureImageView, err := device.CreateImageViewForTexture(textureImage, vk.FORMAT_R8G8B8A8_SRGB)
		if err != nil {
			panic(err)
		}
		defer device.DestroyImageView(textureImageView)

		// Create sampler
		textureSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
			MagFilter:    vk.FILTER_NEAREST, // Use NEAREST to see the checkerboard clearly
			MinFilter:    vk.FILTER_NEAREST,
			MipmapMode:   vk.SAMPLER_MIPMAP_MODE_NEAREST,
			AddressModeU: vk.SAMPLER_ADDRESS_MODE_REPEAT,
			AddressModeV: vk.SAMPLER_ADDRESS_MODE_REPEAT,
			AddressModeW: vk.SAMPLER_ADDRESS_MODE_REPEAT,
			MaxLod:       1.0,
			BorderColor:  vk.BORDER_COLOR_FLOAT_OPAQUE_BLACK,
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroySampler(textureSampler)

		fmt.Println("Texture created!")

		// Create descriptor set layout

		// Create descriptor pool
		descriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 1,
			PoolSizes: []vk.DescriptorPoolSize{
				{
					Type:            vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: 1,
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorPool(descriptorPool)

		// Allocate descriptor set
		descriptorSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: descriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{descriptorSetLayout},
		})
		if err != nil {
			panic(err)
		}
		descriptorSet := descriptorSets[0]

		// Update descriptor set with texture
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          descriptorSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     textureSampler,
						ImageView:   textureImageView,
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					},
				},
			},
		})

		fmt.Println("Descriptor sets created and updated!")

		// Allocate command buffers (one per swapchain image)
		commandBuffers, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
			CommandPool:        commandPool,
			Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
			CommandBufferCount: uint32(len(swapImages)),
		})
		if err != nil {
			panic(err)
		}

		// Create synchronization objects
		imageAvailableSem, err := device.CreateSemaphore(&vk.SemaphoreCreateInfo{})
		if err != nil {
			panic(err)
		}
		defer device.DestroySemaphore(imageAvailableSem)

		renderFinishedSem, err := device.CreateSemaphore(&vk.SemaphoreCreateInfo{})
		if err != nil {
			panic(err)
		}
		defer device.DestroySemaphore(renderFinishedSem)

		inFlightFence, err := device.CreateFence(&vk.FenceCreateInfo{
			Flags: vk.FENCE_CREATE_SIGNALED_BIT,
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyFence(inFlightFence)

		fmt.Println("Command buffers and sync objects created!")

		// Render loop
		fmt.Println("\nRendering Djungelskog - close window to exit")
		running := true
		for running {

			// Handle events
			for event, ok := sdl.PollEvent(); ok; event, ok = sdl.PollEvent() {
				if event.Type == sdl.EVENT_QUIT {
					running = false
				}
			}

			time.Sleep(time.Millisecond * 5)

			// Wait for previous frame
			device.WaitForFences([]vk.Fence{inFlightFence}, true, ^uint64(0))
			device.ResetFences([]vk.Fence{inFlightFence})

			// Acquire next image
			imageIndex, err := device.AcquireNextImageKHR(swapchain, ^uint64(0), imageAvailableSem, vk.Fence{})
			if err != nil {
				panic(fmt.Sprintf("Acquire failed: %v", err))
			}

			// Record command buffer
			cmd := commandBuffers[imageIndex]
			cmd.Reset(0)
			cmd.Begin(&vk.CommandBufferBeginInfo{})

			// Transition image to color attachment
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
				0,
				[]vk.ImageMemoryBarrier{
					{
						SrcAccessMask:       vk.ACCESS_NONE,
						DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
						NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               swapImages[imageIndex],
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					},
				},
			)

			// Begin rendering
			cmd.BeginRendering(&vk.RenderingInfo{
				RenderArea: vk.Rect2D{
					Offset: vk.Offset2D{X: 0, Y: 0},
					Extent: swapExtent,
				},
				LayerCount: 1,
				ColorAttachments: []vk.RenderingAttachmentInfo{
					{
						ImageView:   swapImageViews[imageIndex],
						ImageLayout: vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
						LoadOp:      vk.ATTACHMENT_LOAD_OP_CLEAR,
						StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
						ClearValue: vk.ClearValue{
							Color: vk.ClearColorValue{
								Float32: [4]float32{0.0, 0.0, 0.0, 1.0}, // Black background
							},
						},
					},
				},
			})

			// Bind pipeline and set dynamic state
			cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, pipeline)

			cmd.BindDescriptorSets(
				vk.PIPELINE_BIND_POINT_GRAPHICS,
				pipelineLayout,
				0,
				[]vk.DescriptorSet{descriptorSet},
				nil,
			)

			cmd.SetViewport(0, []vk.Viewport{
				{
					X:        0,
					Y:        0,
					Width:    float32(swapExtent.Width),
					Height:   float32(swapExtent.Height),
					MinDepth: 0.0,
					MaxDepth: 1.0,
				},
			})
			cmd.SetScissor(0, []vk.Rect2D{
				{
					Offset: vk.Offset2D{X: 0, Y: 0},
					Extent: swapExtent,
				},
			})

			// Draw triangle!
			cmd.BindVertexBuffers(0, []vk.Buffer{vertexBuffer}, []uint64{0})
			cmd.BindIndexBuffer(indexBuffer, 0, vk.INDEX_TYPE_UINT16)
			cmd.DrawIndexed(uint32(len(indices)), 1, 0, 0, 0)

			cmd.EndRendering()

			// Transition image to present
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				0,
				[]vk.ImageMemoryBarrier{
					{
						SrcAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
						DstAccessMask:       vk.ACCESS_NONE,
						OldLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
						NewLayout:           vk.IMAGE_LAYOUT_PRESENT_SRC_KHR,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               swapImages[imageIndex],
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					},
				},
			)

			cmd.End()

			// Submit command buffer
			err = queue.Submit([]vk.SubmitInfo{
				{
					WaitSemaphores:   []vk.Semaphore{imageAvailableSem},
					WaitDstStageMask: []vk.PipelineStageFlags{vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT},
					CommandBuffers:   []vk.CommandBuffer{cmd},
					SignalSemaphores: []vk.Semaphore{renderFinishedSem},
				},
			}, inFlightFence)
			if err != nil {
				panic(fmt.Sprintf("Queue submit failed: %v", err))
			}

			// Present
			queue.PresentKHR(&vk.PresentInfoKHR{
				WaitSemaphores: []vk.Semaphore{renderFinishedSem},
				Swapchains:     []vk.SwapchainKHR{swapchain},
				ImageIndices:   []uint32{imageIndex},
			})
		}

		// Wait for device to finish
		device.WaitForFences([]vk.Fence{inFlightFence}, true, ^uint64(0))

	}
}
