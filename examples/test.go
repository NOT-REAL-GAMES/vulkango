package main

import (
	"fmt"
	"time"

	sdl "github.com/NOT-REAL-GAMES/sdl3go"
	vk "github.com/NOT-REAL-GAMES/vulkango"
	shaderc "github.com/NOT-REAL-GAMES/vulkango/shaderc"
)

const vertexShader = `
#version 450

layout(location = 0) out vec3 fragColor;

vec2 positions[3] = vec2[](
    vec2(0.0, -0.5),
    vec2(0.5, 0.5),
    vec2(-0.5, 0.5)
);

vec3 colors[3] = vec3[](
    vec3(1.0, 0.0, 0.0),
    vec3(0.0, 1.0, 0.0),
    vec3(0.0, 0.0, 1.0)
);

void main() {
    gl_Position = vec4(positions[gl_VertexIndex], 0.0, 1.0);
    fragColor = colors[gl_VertexIndex];
}
`

const fragmentShader = `
#version 450

layout(location = 0) in vec3 fragColor;
layout(location = 0) out vec4 outColor;

void main() {
    outColor = vec4(fragColor, 1.0);
}
`

func main() {
	// Initialize SDL first
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Example", 960, 540, sdl.WINDOW_VULKAN)
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
			960, 540, // window dimensions
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

		// After creating shader modules
		pipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(pipelineLayout)

		pipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{
					Stage:  vk.SHADER_STAGE_VERTEX_BIT,
					Module: vertModule,
					Name:   "main",
				},
				{
					Stage:  vk.SHADER_STAGE_FRAGMENT_BIT,
					Module: fragModule,
					Name:   "main",
				},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{},
			InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
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

		// Create command pool
		commandPool, err := device.CreateCommandPool(&vk.CommandPoolCreateInfo{
			Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
			QueueFamilyIndex: uint32(graphicsFamily),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyCommandPool(commandPool)

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
		fmt.Println("\nRendering triangle - close window to exit")
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
			fmt.Printf("Acquired image %d\n", imageIndex)

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
			cmd.Draw(3, 1, 0, 0)

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
