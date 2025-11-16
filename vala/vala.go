package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	"math"
	"os"
	"runtime"
	"time"
	"unsafe"

	sdl "github.com/NOT-REAL-GAMES/sdl3go"
	vk "github.com/NOT-REAL-GAMES/vulkango"
	shaderc "github.com/NOT-REAL-GAMES/vulkango/shaderc"

	"github.com/NOT-REAL-GAMES/vulkango/vala/canvas"
	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
	fontpkg "github.com/NOT-REAL-GAMES/vulkango/vala/font"
	"github.com/NOT-REAL-GAMES/vulkango/vala/systems"
)

const vertexShader = `
#version 450

layout(location = 0) in vec2 inPosition;
layout(location = 1) in vec3 inColor;
layout(location = 2) in vec2 inTexCoord;

layout(push_constant) uniform PushConstants {
    vec2 offset;
    float scale;
    float opacity;
    float depth;
} pushConstants;

layout(location = 0) out vec3 fragColor;
layout(location = 1) out vec2 fragTexCoord;
layout(location = 2) out float fragOpacity;

void main() {
    vec2 pos = inPosition * pushConstants.scale + pushConstants.offset;
    gl_Position = vec4(pos, pushConstants.depth, 1.0);
    fragColor = inColor;
    fragTexCoord = inTexCoord;
    fragOpacity = pushConstants.opacity;
}
`

const fragmentShader = `
#version 450

layout(location = 0) in vec3 fragColor;
layout(location = 1) in vec2 fragTexCoord;
layout(location = 2) in float fragOpacity;

layout(binding = 0) uniform sampler2D texSampler;

layout(location = 0) out vec4 outColor;

void main() {
    vec4 texColor = texture(texSampler, fragTexCoord);
    outColor = texColor * vec4(fragColor, fragOpacity);
}
`

const compositeVertexShader = `
#version 450

// Fullscreen quad vertices (no input needed, generated in shader)
vec2 positions[6] = vec2[](
    vec2(-1.0, -1.0),  // Bottom-left
    vec2( 1.0, -1.0),  // Bottom-right
    vec2( 1.0,  1.0),  // Top-right
    vec2(-1.0, -1.0),  // Bottom-left
    vec2( 1.0,  1.0),  // Top-right
    vec2(-1.0,  1.0)   // Top-left
);

vec2 texCoords[6] = vec2[](
    vec2(0.0, 0.0),
    vec2(1.0, 0.0),
    vec2(1.0, 1.0),
    vec2(0.0, 0.0),
    vec2(1.0, 1.0),
    vec2(0.0, 1.0)
);

layout(location = 0) out vec2 fragTexCoord;

void main() {
    gl_Position = vec4(positions[gl_VertexIndex], 0.0, 1.0);
    fragTexCoord = texCoords[gl_VertexIndex];
}
`

const compositeFragmentShader = `
#version 450

layout(location = 0) in vec2 fragTexCoord;

layout(binding = 0) uniform sampler2D layerTexture;

// Push constant for layer opacity
layout(push_constant) uniform LayerPushConstants {
    float opacity; // Layer opacity (0.0 to 1.0)
} layer;

layout(location = 0) out vec4 outColor;

void main() {
    vec4 texColor = texture(layerTexture, fragTexCoord);
    // Apply layer opacity to the alpha channel
    outColor = vec4(texColor.rgb, texColor.a * layer.opacity);
}
`

const brushVertexShader = `
#version 450

// Input: quad vertices (0,0 to 1,1)
layout(location = 0) in vec2 inPosition;

// Push constants for brush
layout(push_constant) uniform BrushPushConstants {
    vec2 canvasSize;     // Canvas dimensions in pixels
    vec2 brushPos;       // Brush center in canvas pixels
    float brushSize;     // Brush radius in pixels
    float brushOpacity;  // 0.0 to 1.0
    vec4 brushColor;     // RGBA color
} brush;

layout(location = 0) out vec2 fragLocalPos; // Position within brush quad (0-1)

void main() {
    // Calculate brush quad corners in canvas pixels
    vec2 quadPos = inPosition; // 0,0 to 1,1
    vec2 canvasPos = brush.brushPos + (quadPos - 0.5) * brush.brushSize * 2.0;

    // Convert to NDC (-1 to 1)
    vec2 ndc = (canvasPos / brush.canvasSize) * 2.0 - 1.0;

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragLocalPos = quadPos;
}
`

const brushFragmentShader = `
#version 450

layout(location = 0) in vec2 fragLocalPos; // 0,0 to 1,1

layout(push_constant) uniform BrushPushConstants {
    vec2 canvasSize;
    vec2 brushPos;
    float brushSize;
    float brushOpacity;
    vec4 brushColor;
} brush;

layout(location = 0) out vec4 outColor;

void main() {
    // Calculate distance from center (0.5, 0.5)
    vec2 center = vec2(0.5, 0.5);
    float dist = length(fragLocalPos - center);

    // Circular brush with soft edges
    float radius = 0.5;
    float softness = 0.1;
    float alpha = 1.0 - smoothstep(radius - softness, radius, dist);

    // Apply brush opacity
    alpha *= brush.brushOpacity;

    // Output color with alpha
    outColor = vec4(brush.brushColor.rgb, brush.brushColor.a);
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
			EnabledExtensionNames: []string{
				"VK_KHR_swapchain",
				"VK_EXT_blend_operation_advanced",
			},
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
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT,
					Offset:     0,
					Size:       20, // vec2 (8 bytes) + float (4) + float (4) + float (4) = 20 bytes
				},
			},
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
						BlendEnable:         true,
						SrcColorBlendFactor: vk.BLEND_FACTOR_SRC_ALPHA,
						DstColorBlendFactor: vk.BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
						ColorBlendOp:        vk.BLEND_OP_ADD,
						SrcAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
						DstAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
						AlphaBlendOp:        vk.BLEND_OP_ADD,
						ColorWriteMask:      vk.COLOR_COMPONENT_ALL,
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

		// === Composite Pipeline Setup ===
		fmt.Println("Creating composite pipeline...")

		// Compile composite shaders
		compositeVertResult, err := compiler.CompileIntoSPV(compositeVertexShader, "composite.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(fmt.Sprintf("Composite vertex shader compilation failed: %v", err))
		}
		compositeVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: compositeVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(compositeVertShader)

		compositeFragResult, err := compiler.CompileIntoSPV(compositeFragmentShader, "composite.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(fmt.Sprintf("Composite fragment shader compilation failed: %v", err))
		}
		compositeFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: compositeFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(compositeFragShader)

		// Create composite descriptor set layout
		compositeDescriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
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
		defer device.DestroyDescriptorSetLayout(compositeDescriptorSetLayout)

		// Create composite pipeline layout
		compositePipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			SetLayouts: []vk.DescriptorSetLayout{compositeDescriptorSetLayout},
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       4, // sizeof(float) for opacity
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(compositePipelineLayout)

		// Create composite graphics pipeline (no vertex input, fullscreen quad)
		compositePipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: compositeVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: compositeFragShader, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				// No vertex input - fullscreen quad generated in shader
			},
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
						BlendEnable:         true,
						SrcColorBlendFactor: vk.BLEND_FACTOR_SRC_ALPHA,
						DstColorBlendFactor: vk.BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
						ColorBlendOp:        vk.BLEND_OP_ADD,
						SrcAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
						DstAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
						AlphaBlendOp:        vk.BLEND_OP_ADD,
						ColorWriteMask:      vk.COLOR_COMPONENT_ALL,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{
					vk.DYNAMIC_STATE_VIEWPORT,
					vk.DYNAMIC_STATE_SCISSOR,
				},
			},
			Layout: compositePipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{swapFormat},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(compositePipeline)

		// === Brush Pipeline Setup ===
		fmt.Println("Creating brush pipeline...")

		// Compile brush shaders
		brushVertResult, err := compiler.CompileIntoSPV(brushVertexShader, "brush.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(fmt.Sprintf("Brush vertex shader compilation failed: %v", err))
		}
		brushVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: brushVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(brushVertShader)

		brushFragResult, err := compiler.CompileIntoSPV(brushFragmentShader, "brush.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(fmt.Sprintf("Brush fragment shader compilation failed: %v", err))
		}
		brushFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: brushFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(brushFragShader)

		// Brush push constants: vec2 canvasSize, vec2 brushPos, float size, float opacity, vec4 color
		// Total: 8 + 8 + 4 + 4 + 16 = 40 bytes
		brushPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       48, // sizeof(BrushPushConstants)
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(brushPipelineLayout)

		// Create brush graphics pipeline
		brushPipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: brushVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: brushFragShader, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				VertexBindingDescriptions: []vk.VertexInputBindingDescription{
					{
						Binding:   0,
						Stride:    8, // 2 floats (vec2)
						InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
					},
				},
				VertexAttributeDescriptions: []vk.VertexInputAttributeDescription{
					{
						Location: 0,
						Binding:  0,
						Format:   vk.FORMAT_R32G32_SFLOAT, // vec2
						Offset:   0,
					},
				},
			},
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
						BlendEnable:         true,
						SrcColorBlendFactor: vk.BLEND_FACTOR_SRC_ALPHA,
						DstColorBlendFactor: vk.BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
						ColorBlendOp:        vk.BLEND_OP_ADD,
						SrcAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
						DstAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
						AlphaBlendOp:        vk.BLEND_OP_ADD,
						ColorWriteMask:      vk.COLOR_COMPONENT_ALL,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{
					vk.DYNAMIC_STATE_VIEWPORT,
					vk.DYNAMIC_STATE_SCISSOR,
				},
			},
			Layout: brushPipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{vk.FORMAT_R8G8B8A8_UNORM}, // Canvas format
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(brushPipeline)

		fmt.Println("Brush pipeline created!")

		// Create descriptor pool for composite pass (2 sets for 2 layers)
		compositeDescriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 2,
			PoolSizes: []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, DescriptorCount: 2},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorPool(compositeDescriptorPool)

		// Create sampler for layer textures
		layerSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
			MagFilter:    vk.FILTER_LINEAR,
			MinFilter:    vk.FILTER_LINEAR,
			AddressModeU: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeV: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeW: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroySampler(layerSampler)

		fmt.Println("Composite pipeline created!")

		// === Text Rendering Setup ===
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

		// Create brush vertex buffer (simple quad 0,0 to 1,1)
		brushVertices := [][2]float32{
			{0.0, 0.0}, // Bottom-left
			{1.0, 0.0}, // Bottom-right
			{1.0, 1.0}, // Top-right
			{0.0, 0.0}, // Bottom-left
			{1.0, 1.0}, // Top-right
			{0.0, 1.0}, // Top-left
		}
		brushVertexBufferSize := uint64(len(brushVertices) * 8) // 6 vertices * 8 bytes each

		brushVertexBuffer, brushVertexMemory, err := device.CreateBufferWithMemory(
			brushVertexBufferSize,
			vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(brushVertexBuffer)
		defer device.FreeMemory(brushVertexMemory)

		// Upload brush vertex data
		brushVertexData := (*[1 << 30]byte)(unsafe.Pointer(&brushVertices[0]))[:brushVertexBufferSize:brushVertexBufferSize]
		err = device.UploadToBuffer(brushVertexMemory, brushVertexData)
		if err != nil {
			panic(err)
		}

		fmt.Println("Brush vertex buffer created!")

		// Create command pool
		commandPool, err := device.CreateCommandPool(&vk.CommandPoolCreateInfo{
			Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
			QueueFamilyIndex: uint32(graphicsFamily),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyCommandPool(commandPool)

		// Create paint canvas
		fmt.Println("\nCreating paint canvas...")
		paintCanvas, err := canvas.New(canvas.Config{
			Device:         device,
			PhysicalDevice: physicalDevice,
			Width:          2048,
			Height:         2048,
			Format:         vk.FORMAT_R8G8B8A8_UNORM,
			Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
				vk.IMAGE_USAGE_SAMPLED_BIT |
				vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
			UseSparseBinding: false, // Start with dense allocation
		}, commandPool, queue)
		if err != nil {
			panic(fmt.Sprintf("Failed to create paint canvas: %v", err))
		}
		defer paintCanvas.Destroy()
		fmt.Printf("Paint canvas created: %dx%d\n", paintCanvas.GetWidth(), paintCanvas.GetHeight())

		// Clear canvas to white
		whitePixels := make([]byte, 2048*2048*4)
		for i := 0; i < len(whitePixels); i += 4 {
			whitePixels[i] = 255   // R
			whitePixels[i+1] = 255 // G
			whitePixels[i+2] = 255 // B
			whitePixels[i+3] = 255 // A
		}
		err = paintCanvas.Upload(0, 0, 2048, 2048, whitePixels)
		if err != nil {
			panic(fmt.Sprintf("Failed to clear canvas: %v", err))
		}
		fmt.Println("Canvas cleared to white")

		// Create sampler for paint canvas
		canvasSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
			MagFilter:    vk.FILTER_LINEAR,
			MinFilter:    vk.FILTER_LINEAR,
			MipmapMode:   vk.SAMPLER_MIPMAP_MODE_LINEAR,
			AddressModeU: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeV: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeW: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			MaxLod:       1.0,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create canvas sampler: %v", err))
		}
		defer device.DestroySampler(canvasSampler)

		fmt.Println("\nCreating texture...")

		imageData, err := LoadImage("djungelskog.jpg") // Put a PNG in your project folder
		if err != nil {
			panic(err)
		}

		textureWidth := imageData.Width
		atlasHeight := imageData.Height
		textureData := imageData.Pixels
		atlasSize := uint64(len(textureData))

		fmt.Printf("Loaded texture: %dx%d (%d bytes)\n", textureWidth, atlasHeight, atlasSize)

		// Create staging buffer
		stagingBuffer, stagingMemory, err := device.CreateBufferWithMemory(
			atlasSize,
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
			textureWidth, atlasHeight,
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
				ImageExtent: vk.Extent3D{Width: textureWidth, Height: atlasHeight, Depth: 1},
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
		fmt.Println("Setting up SDF text rendering...")

		// Load embedded font
		fontData, err := fontpkg.LoadEmbeddedFont()
		if err != nil {
			panic(fmt.Sprintf("Failed to load font: %v", err))
		}

		// Generate SDF atlas (32 pixel font, 8 pixel padding, edge at 128, 32 pixel distance scale for more contrast)
		sdfAtlas, err := fontpkg.GenerateSDFAtlas(fontData, 32.0, 8, 128, 32.0)
		if err != nil {
			panic(fmt.Sprintf("Failed to generate SDF atlas: %v", err))
		}
		fmt.Printf("Generated SDF atlas: %dx%d with %d characters\n",
			sdfAtlas.Width, sdfAtlas.Height, len(sdfAtlas.Chars))

		// Upload SDF atlas to GPU
		textAtlasW := uint32(sdfAtlas.Width)
		textAtlasH := uint32(sdfAtlas.Height)
		textAtlasBytes := uint64(len(sdfAtlas.Pixels))

		// Create staging buffer for atlas
		textAtlasStaging, textAtlasStagingMemory, err := device.CreateBufferWithMemory(
			textAtlasBytes,
			vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textAtlasStaging)
		defer device.FreeMemory(textAtlasStagingMemory)

		// Upload atlas data to staging buffer
		err = device.UploadToBuffer(textAtlasStagingMemory, sdfAtlas.Pixels)
		if err != nil {
			panic(err)
		}

		// Create atlas texture image (R8 format for single-channel SDF)
		textAtlasImage, textAtlasMemory, err := device.CreateImageWithMemory(
			textAtlasW,
			textAtlasH,
			vk.FORMAT_R8_UNORM,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyImage(textAtlasImage)
		defer device.FreeMemory(textAtlasMemory)

		// Create image view for atlas
		textAtlasImageView, err := device.CreateImageViewForTexture(textAtlasImage, vk.FORMAT_R8_UNORM)
		if err != nil {
			panic(err)
		}
		defer device.DestroyImageView(textAtlasImageView)

		// Create sampler for text atlas
		textAtlasSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
			MagFilter:    vk.FILTER_LINEAR,
			MinFilter:    vk.FILTER_LINEAR,
			AddressModeU: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeV: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeW: vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroySampler(textAtlasSampler)

		// Create command buffer for atlas upload
		atlasUploadCmdBuffer, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
			CommandPool:        commandPool,
			Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
			CommandBufferCount: 1,
		})
		if err != nil {
			panic(err)
		}
		atlasUploadCmd := atlasUploadCmdBuffer[0]

		// Record upload commands
		atlasUploadCmd.Begin(&vk.CommandBufferBeginInfo{
			Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
		})

		// Transition image to transfer dst
		atlasUploadCmd.PipelineBarrier(
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
					Image:               textAtlasImage,
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
		atlasUploadCmd.CopyBufferToImage(textAtlasStaging, textAtlasImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{
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
				ImageExtent: vk.Extent3D{Width: textAtlasW, Height: textAtlasH, Depth: 1},
			},
		})

		// Transition image to shader read
		atlasUploadCmd.PipelineBarrier(
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
					Image:               textAtlasImage,
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

		atlasUploadCmd.End()

		// Submit and wait
		err = queue.Submit([]vk.SubmitInfo{{CommandBuffers: []vk.CommandBuffer{atlasUploadCmd}}}, vk.Fence{})
		if err != nil {
			panic(fmt.Sprintf("Atlas upload submit failed: %v", err))
		}
		queue.WaitIdle()

		fmt.Println("SDF atlas uploaded to GPU!")

		// === Text Rendering Pipeline ===
		fmt.Println("Creating text rendering pipeline...")

		// Compile SDF shaders
		textVertResult, err := compiler.CompileIntoSPV(fontpkg.SDFVertexShader, "text.vert", shaderc.VertexShader, options)
		defer textVertResult.Release()
		if err != nil {
			panic(fmt.Sprintf("Text vertex shader compilation failed: %v", err))
		}
		textVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: textVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(textVertShader)

		textFragResult, err := compiler.CompileIntoSPV(fontpkg.SDFFragmentShader, "text.frag", shaderc.FragmentShader, options)
		defer textFragResult.Release()
		if err != nil {
			panic(fmt.Sprintf("Text fragment shader compilation failed: %v", err))
		}
		textFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: textFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(textFragShader)

		// Create descriptor set layout for text (binding 0: sampler2D)
		textDescriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
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
		defer device.DestroyDescriptorSetLayout(textDescriptorSetLayout)

		// Create text pipeline layout (with push constants for screen size and color)
		textPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			SetLayouts: []vk.DescriptorSetLayout{textDescriptorSetLayout},
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT,
					Offset:     0,
					Size:       32, // vec2 screenSize (8) + padding (8) + vec4 textColor (16) = 32 bytes (std140)
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(textPipelineLayout)

		// Create text rendering pipeline
		textPipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: textVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: textFragShader, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				VertexBindingDescriptions: []vk.VertexInputBindingDescription{
					{
						Binding:   0,
						Stride:    16, // 2 floats (pos) + 2 floats (uv) = 16 bytes
						InputRate: vk.VERTEX_INPUT_RATE_VERTEX,
					},
				},
				VertexAttributeDescriptions: []vk.VertexInputAttributeDescription{
					{Location: 0, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 0}, // position
					{Location: 1, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 8}, // texCoord
				},
			},
			InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
				Topology: vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
			},
			ViewportState: &vk.PipelineViewportStateCreateInfo{
				Viewports: []vk.Viewport{{}}, // Dynamic, but must specify count
				Scissors:  []vk.Rect2D{{}},   // Dynamic, but must specify count
			},
			RasterizationState: &vk.PipelineRasterizationStateCreateInfo{
				PolygonMode: vk.POLYGON_MODE_FILL,
				CullMode:    vk.CULL_MODE_NONE,
				FrontFace:   vk.FRONT_FACE_CLOCKWISE,
				LineWidth:   1.0,
			},
			MultisampleState: &vk.PipelineMultisampleStateCreateInfo{
				RasterizationSamples: vk.SAMPLE_COUNT_1_BIT,
			},
			ColorBlendState: &vk.PipelineColorBlendStateCreateInfo{
				Attachments: []vk.PipelineColorBlendAttachmentState{
					{
						BlendEnable:         true,
						SrcColorBlendFactor: vk.BLEND_FACTOR_SRC_ALPHA,
						DstColorBlendFactor: vk.BLEND_FACTOR_ONE_MINUS_SRC_ALPHA,
						ColorBlendOp:        vk.BLEND_OP_ADD,
						SrcAlphaBlendFactor: vk.BLEND_FACTOR_ONE,
						DstAlphaBlendFactor: vk.BLEND_FACTOR_ZERO,
						AlphaBlendOp:        vk.BLEND_OP_ADD,
						ColorWriteMask:      vk.COLOR_COMPONENT_R_BIT | vk.COLOR_COMPONENT_G_BIT | vk.COLOR_COMPONENT_B_BIT | vk.COLOR_COMPONENT_A_BIT,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{
					vk.DYNAMIC_STATE_VIEWPORT,
					vk.DYNAMIC_STATE_SCISSOR,
				},
			},
			Layout: textPipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{swapFormat},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create text pipeline: %v", err))
		}
		defer device.DestroyPipeline(textPipeline)

		// Create descriptor pool for text
		textDescriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 1,
			PoolSizes: []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, DescriptorCount: 1},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorPool(textDescriptorPool)

		// Allocate descriptor set for text atlas
		textDescriptorSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: textDescriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{textDescriptorSetLayout},
		})
		if err != nil {
			panic(err)
		}
		textDescriptorSet := textDescriptorSets[0]

		// Update descriptor set with SDF atlas texture
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          textDescriptorSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     textAtlasSampler,
						ImageView:   textAtlasImageView,
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					},
				},
			},
		})

		fmt.Println("Text rendering pipeline created!")

		// Create vertex/index buffers for text (sized for ~100 characters = 400 vertices, 600 indices)
		maxTextChars := 100
		textVertexBufferSize := uint64(maxTextChars * 4 * 16) // 4 verts * 16 bytes per vertex
		textIndexBufferSize := uint64(maxTextChars * 6 * 2)   // 6 indices * 2 bytes per index

		textVertexBuffer, textVertexMemory, err := device.CreateBufferWithMemory(
			textVertexBufferSize,
			vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textVertexBuffer)
		defer device.FreeMemory(textVertexMemory)

		textIndexBuffer, textIndexMemory, err := device.CreateBufferWithMemory(
			textIndexBufferSize,
			vk.BUFFER_USAGE_INDEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textIndexBuffer)
		defer device.FreeMemory(textIndexMemory)

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

		// Update descriptor set with paint canvas
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          descriptorSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     canvasSampler,
						ImageView:   paintCanvas.GetView(),
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

		// === ECS Setup ===
		// Create the ECS World
		world := ecs.NewWorld()
		fmt.Println("\n=== ECS World Created ===")

		// Create a layer entity for the Djungelskog texture
		layer1 := world.CreateEntity()
		fmt.Printf("Created layer entity: %d\n", layer1)

		// Add Transform component
		transform1 := ecs.NewTransform()
		transform1.ZIndex = -10 // Layer 1 is in the back
		world.AddTransform(layer1, transform1)

		// Add VulkanPipeline component
		world.AddVulkanPipeline(layer1, &ecs.VulkanPipeline{
			Pipeline:            pipeline,
			PipelineLayout:      pipelineLayout,
			DescriptorPool:      descriptorPool,
			DescriptorSet:       descriptorSet,
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add TextureData component (paint canvas)
		world.AddTextureData(layer1, &ecs.TextureData{
			Image:       paintCanvas.GetImage(),
			ImageView:   paintCanvas.GetView(),
			ImageMemory: vk.DeviceMemory{}, // Canvas manages its own memory
			Sampler:     canvasSampler,
			Width:       paintCanvas.GetWidth(),
			Height:      paintCanvas.GetHeight(),
		})

		// Add BlendMode component (visible, opaque)
		world.AddBlendMode(layer1, ecs.NewBlendMode())

		fmt.Printf("Layer 1 configured with %d components\n", 4)

		// Create framebuffer for layer 1
		layer1Image, layer1ImageMemory, err := device.CreateImageWithMemory(
			swapExtent.Width,
			swapExtent.Height,
			vk.FORMAT_R8G8B8A8_UNORM,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(fmt.Sprintf("Failed to create layer 1 framebuffer image: %v", err))
		}
		defer device.DestroyImage(layer1Image)
		defer device.FreeMemory(layer1ImageMemory)

		layer1ImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
			Image:    layer1Image,
			ViewType: vk.IMAGE_VIEW_TYPE_2D,
			Format:   vk.FORMAT_R8G8B8A8_UNORM,
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create layer 1 image view: %v", err))
		}
		defer device.DestroyImageView(layer1ImageView)

		// Add RenderTarget component for layer 1
		world.AddRenderTarget(layer1, &ecs.RenderTarget{
			Image:       layer1Image,
			ImageView:   layer1ImageView,
			ImageMemory: layer1ImageMemory,
			Format:      vk.FORMAT_R8G8B8A8_UNORM,
			Width:       swapExtent.Width,
			Height:      swapExtent.Height,
		})

		// Create composite descriptor set for layer 1
		layer1CompositeSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: compositeDescriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{compositeDescriptorSetLayout},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to allocate layer 1 composite descriptor set: %v", err))
		}
		layer1CompositeDescSet := layer1CompositeSets[0]

		// Update descriptor set to bind layer1's framebuffer texture
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          layer1CompositeDescSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     layerSampler,
						ImageView:   layer1ImageView,
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					},
				},
			},
		})

		// Update layer1's VulkanPipeline with composite descriptor set
		layer1Pipeline := world.GetVulkanPipeline(layer1)
		layer1Pipeline.CompositeDescriptorSet = layer1CompositeDescSet

		// Create a second layer entity (demonstrating multi-layer support)
		layer2 := world.CreateEntity()
		fmt.Printf("Created layer entity: %d\n", layer2)

		// Add Transform component (with higher ZIndex to be in front)
		transform2 := ecs.NewTransform()
		transform2.ZIndex = 10 // Layer 2 is in the front
		world.AddTransform(layer2, transform2)

		// Add VulkanPipeline component (reusing the same pipeline and descriptors)
		world.AddVulkanPipeline(layer2, &ecs.VulkanPipeline{
			Pipeline:            pipeline,
			PipelineLayout:      pipelineLayout,
			DescriptorPool:      descriptorPool,
			DescriptorSet:       descriptorSet,
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add TextureData component (same texture)
		world.AddTextureData(layer2, &ecs.TextureData{
			Image:       textureImage,
			ImageView:   textureImageView,
			ImageMemory: textureMemory,
			Sampler:     textureSampler,
			Width:       imageData.Width,
			Height:      imageData.Height,
		})

		// Add BlendMode component with different opacity
		blendMode2 := ecs.NewBlendMode()
		blendMode2.Opacity = 0.5 // 50% transparent
		world.AddBlendMode(layer2, blendMode2)

		// Create framebuffer for layer 2
		layer2Image, layer2ImageMemory, err := device.CreateImageWithMemory(
			swapExtent.Width,
			swapExtent.Height,
			vk.FORMAT_R8G8B8A8_UNORM,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(fmt.Sprintf("Failed to create layer 2 framebuffer image: %v", err))
		}
		defer device.DestroyImage(layer2Image)
		defer device.FreeMemory(layer2ImageMemory)

		layer2ImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
			Image:    layer2Image,
			ViewType: vk.IMAGE_VIEW_TYPE_2D,
			Format:   vk.FORMAT_R8G8B8A8_UNORM,
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create layer 2 image view: %v", err))
		}
		defer device.DestroyImageView(layer2ImageView)

		// Add RenderTarget component for layer 2
		world.AddRenderTarget(layer2, &ecs.RenderTarget{
			Image:       layer2Image,
			ImageView:   layer2ImageView,
			ImageMemory: layer2ImageMemory,
			Format:      vk.FORMAT_R8G8B8A8_UNORM,
			Width:       swapExtent.Width,
			Height:      swapExtent.Height,
		})
		// Create composite descriptor set for layer 2
		layer2CompositeSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: compositeDescriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{compositeDescriptorSetLayout},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to allocate layer 2 composite descriptor set: %v", err))
		}
		layer2CompositeDescSet := layer2CompositeSets[0]

		// Update descriptor set to bind layer2's framebuffer texture
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          layer2CompositeDescSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     layerSampler,
						ImageView:   layer2ImageView,
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					},
				},
			},
		})

		// Update layer2's VulkanPipeline with composite descriptor set
		layer2Pipeline := world.GetVulkanPipeline(layer2)
		layer2Pipeline.CompositeDescriptorSet = layer2CompositeDescSet

		fmt.Printf("Layer 2 configured with %d components (50%% opacity)\n", 4)
		fmt.Printf("Total entities in world: %d\n", world.EntityCount())

		// Create "Hello World!" text entity
		helloText := world.CreateEntity()
		textComponent := ecs.NewText("Hello World!", 100.0, 100.0, 32.0)
		textComponent.Color = [4]float32{1.0, 1.0, 1.0, 1.0} // White
		world.AddText(helloText, textComponent)
		fmt.Println("Created Hello World text entity")

		// === Initial Image Layout Transitions ===
		// Transition layer framebuffers from UNDEFINED to COLOR_ATTACHMENT_OPTIMAL
		fmt.Println("Transitioning layer images to COLOR_ATTACHMENT_OPTIMAL...")

		transitionCmd := commandBuffers[0] // Reuse first command buffer for one-time setup
		transitionCmd.Reset(0)
		transitionCmd.Begin(&vk.CommandBufferBeginInfo{})

		transitionCmd.PipelineBarrier(
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
					Image:               layer1Image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				},
				{
					SrcAccessMask:       vk.ACCESS_NONE,
					DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
					NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               layer2Image,
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

		transitionCmd.End()

		// Submit and wait for completion
		err = queue.Submit([]vk.SubmitInfo{
			{
				CommandBuffers: []vk.CommandBuffer{transitionCmd},
			},
		}, vk.Fence{})
		if err != nil {
			panic(fmt.Sprintf("Failed to submit transition commands: %v", err))
		}
		queue.WaitIdle()

		fmt.Println("Layer images transitioned!")

		// Render loop
		fmt.Println("\n=== Rendering via ECS - close window to exit ===")
		startTime := time.Now()
		running := true

		// Pen state tracking
		penX := float32(0.0)
		penY := float32(0.0)
		prevPenX := float32(0.0)
		prevPenY := float32(0.0)
		penPressure := float32(0.0)
		penDown := false

		for running {

			// Handle events
			for event, ok := sdl.PollEvent(); ok; event, ok = sdl.PollEvent() {
				switch event.Type {

				case sdl.EVENT_QUIT:
					running = false

				case sdl.EVENT_PEN_AXIS:
					axis := event.PenAxis
					if axis.Axis == sdl.PEN_AXIS_PRESSURE {
						penPressure = axis.Value
					}
				case sdl.EVENT_PEN_MOTION:
					motion := event.PenMotion
					penX = motion.X
					penY = motion.Y
					penDown = motion.IsDown()
				}
			}

			// Update pen data text
			textComp := world.GetText(helloText)
			textComp.Content = fmt.Sprintf("Pen: (%.0f, %.0f)  Pressure: %.3f  Down: %v",
				penX, penY, penPressure, penDown)
			world.AddText(helloText, textComp)

			// Animate layer2 in a circle
			elapsed := float32(time.Since(startTime).Seconds())
			layer2Transform := world.GetTransform(layer2)
			radius := float32(0.5)
			layer2Transform.X = radius * float32(math.Sin(float64(elapsed)))
			layer2Transform.Y = radius * float32(math.Cos(float64(elapsed)))

			// Oscillate layer2 opacity between 0.0 and 1.0
			layer2Blend := world.GetBlendMode(layer2)
			layer2Blend.Opacity = (float32(math.Sin(float64(elapsed*8.0))) + 1.0) / 2.0

			// Wait for previous frame
			device.WaitForFences([]vk.Fence{inFlightFence}, true, ^uint64(0))
			device.ResetFences([]vk.Fence{inFlightFence})

			// Acquire next image
			imageIndex, err := device.AcquireNextImageKHR(swapchain, ^uint64(0), imageAvailableSem, vk.Fence{})
			if err != nil {
				panic(fmt.Sprintf("Acquire failed: %v", err))
			}

			// Get sorted layers
			sortedLayers := world.QueryRenderablesSorted()

			// Record command buffer
			cmd := commandBuffers[imageIndex]
			cmd.Reset(0)
			cmd.Begin(&vk.CommandBufferBeginInfo{})

			// === PASS 0: Render brush strokes to canvas (if pen is down) ===
			if penDown && penPressure > 0.01 {
				// Map pen coordinates from screen space to canvas space
				canvasX := penX * (float32(paintCanvas.GetWidth()) / float32(swapExtent.Width))
				canvasY := penY * (float32(paintCanvas.GetHeight()) / float32(swapExtent.Height))
			// Initialize previous position on first stroke to avoid interpolating from (0,0)
			if prevPenX == 0.0 && prevPenY == 0.0 {
				prevPenX = penX
				prevPenY = penY
			}

				// Transition canvas: SHADER_READ → COLOR_ATTACHMENT
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						{
							SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvas.GetImage(),
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

				// Begin rendering to canvas
				cmd.BeginRendering(&vk.RenderingInfo{
					RenderArea: vk.Rect2D{
						Offset: vk.Offset2D{X: 0, Y: 0},
						Extent: vk.Extent2D{Width: paintCanvas.GetWidth(), Height: paintCanvas.GetHeight()},
					},
					LayerCount: 1,
					ColorAttachments: []vk.RenderingAttachmentInfo{
						{
							ImageView:   paintCanvas.GetView(),
							ImageLayout: vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							LoadOp:      vk.ATTACHMENT_LOAD_OP_LOAD, // Load existing canvas content
							StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
						},
					},
				})

				// Bind brush pipeline
				cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, brushPipeline)

				// Set viewport and scissor
				cmd.SetViewport(0, []vk.Viewport{
					{
						X:        0,
						Y:        0,
						Width:    float32(paintCanvas.GetWidth()),
						Height:   float32(paintCanvas.GetHeight()),
						MinDepth: 0.0,
						MaxDepth: 1.0,
					},
				})
				cmd.SetScissor(0, []vk.Rect2D{
					{
						Offset: vk.Offset2D{X: 0, Y: 0},
						Extent: vk.Extent2D{Width: paintCanvas.GetWidth(), Height: paintCanvas.GetHeight()},
					},
				})

				// Set brush push constants
				type BrushPushConstants struct {
					CanvasWidth  float32
					CanvasHeight float32
					BrushX       float32
					BrushY       float32
					BrushSize    float32
					BrushOpacity float32
					_            float32 // Padding for vec4 alignment
					_            float32 // Padding for vec4 alignment
					ColorR       float32
					ColorG       float32
					ColorB       float32
					ColorA       float32
				}

				// Bind brush vertex buffer (once, before loop)
				cmd.BindVertexBuffers(0, []vk.Buffer{brushVertexBuffer}, []uint64{0})

				// Interpolate brush stamps between previous and current positions
				prevCanvasX := prevPenX * (float32(paintCanvas.GetWidth()) / float32(swapExtent.Width))
				prevCanvasY := prevPenY * (float32(paintCanvas.GetHeight()) / float32(swapExtent.Height))

				// Calculate distance between previous and current positions
				dx := canvasX - prevCanvasX
				dy := canvasY - prevCanvasY
				distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))

				// Brush spacing (smaller = more stamps, smoother stroke)
				brushRadius := float32(10.0)
				brushSpacing := brushRadius * 0.1

				// Calculate number of steps
				steps := int(distance/brushSpacing) + 1
				if steps < 1 {
					steps = 1
				}

				// Render brush stamps along interpolated path
				for i := 0; i <= steps; i++ {
					t := float32(i) / float32(steps)
					interpX := prevCanvasX + dx*t
					interpY := prevCanvasY + dy*t

					pushConstants := BrushPushConstants{
						CanvasWidth:  float32(paintCanvas.GetWidth()),
						CanvasHeight: float32(paintCanvas.GetHeight()),
						BrushX:       interpX,
						BrushY:       interpY,
						BrushSize:    brushRadius * penPressure,
						BrushOpacity: 1.0,
						ColorR:       1.0, // Red
						ColorG:       0.0,
						ColorB:       0.0,
						ColorA:       1.0,
					}

					cmd.CmdPushConstants(brushPipelineLayout, vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT, 0, 48, unsafe.Pointer(&pushConstants))

					// Draw brush quad
					cmd.Draw(6, 1, 0, 0)
				}

				cmd.EndRendering()

				// Transition canvas: COLOR_ATTACHMENT → SHADER_READ
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						{
							SrcAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
							DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvas.GetImage(),
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
			// Update previous position for next frame
			prevPenX = penX
			prevPenY = penY
			} else {
			// Reset previous position when pen is up to prevent interpolation between strokes
			prevPenX = 0.0
			prevPenY = 0.0
		}

			// ===  PASS 1: Render each layer to its framebuffer ===
			for _, entity := range sortedLayers {
				renderTarget := world.GetRenderTarget(entity)
				if renderTarget == nil {
					continue
				}

				// Transition layer image: SHADER_READ → COLOR_ATTACHMENT
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						{
							SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               renderTarget.Image,
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

				// Begin rendering to layer framebuffer
				cmd.BeginRendering(&vk.RenderingInfo{
					RenderArea: vk.Rect2D{
						Offset: vk.Offset2D{X: 0, Y: 0},
						Extent: swapExtent,
					},
					LayerCount: 1,
					ColorAttachments: []vk.RenderingAttachmentInfo{
						{
							ImageView:   renderTarget.ImageView,
							ImageLayout: vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							LoadOp:      vk.ATTACHMENT_LOAD_OP_CLEAR,
							StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
							ClearValue: vk.ClearValue{
								Color: vk.ClearColorValue{
									Float32: [4]float32{0.0, 0.0, 0.0, 0.0}, // Transparent
								},
							},
						},
					},
				})

				// Render layer content
				renderCtx := &systems.RenderContext{
					CommandBuffer: cmd,
					SwapExtent:    swapExtent,
					VertexBuffer:  vertexBuffer,
					IndexBuffer:   indexBuffer,
					IndexCount:    uint32(len(indices)),
				}
				systems.RenderLayerContent(world, renderCtx, entity)

				cmd.EndRendering()

				// Transition layer image: COLOR_ATTACHMENT → SHADER_READ
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						{
							SrcAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
							DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               renderTarget.Image,
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
			}

			// === PASS 2: Composite layers to swapchain ===

			// Transition swapchain: UNDEFINED → COLOR_ATTACHMENT
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

			// Begin rendering to swapchain
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

			// Composite each layer
			compositeCtx := &systems.CompositeContext{
				CommandBuffer:           cmd,
				CompositePipeline:       compositePipeline,
				CompositePipelineLayout: compositePipelineLayout,
				SwapExtent:              swapExtent,
			}

			for _, entity := range sortedLayers {
				pipeline := world.GetVulkanPipeline(entity)
				blendMode := world.GetBlendMode(entity)
				if pipeline != nil && blendMode != nil {
					systems.CompositeLayer(compositeCtx, pipeline.CompositeDescriptorSet, blendMode.Opacity)
				}
			}

			// === Render Text Overlay ===
			// Query all text entities
			textEntities := world.QueryAll(func(e ecs.Entity) bool {
				return world.HasText(e) && world.GetText(e).Visible
			})

			for _, entity := range textEntities {
				textComp := world.GetText(entity)

				// Generate text quads
				vertices, indices := systems.GenerateTextQuads(textComp, sdfAtlas)
				if len(vertices) == 0 {
					continue
				}

				// Upload vertices
				vertexData := unsafe.Slice((*byte)(unsafe.Pointer(&vertices[0])), len(vertices)*16)
				err = device.UploadToBuffer(textVertexMemory, vertexData)
				if err != nil {
					panic(fmt.Sprintf("Text vertex upload failed: %v", err))
				}

				// Upload indices
				indexData := unsafe.Slice((*byte)(unsafe.Pointer(&indices[0])), len(indices)*2)
				err = device.UploadToBuffer(textIndexMemory, indexData)
				if err != nil {
					panic(fmt.Sprintf("Text index upload failed: %v", err))
				}

				// Bind text pipeline
				cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, textPipeline)

				// Bind descriptor set (SDF atlas)
				cmd.BindDescriptorSets(
					vk.PIPELINE_BIND_POINT_GRAPHICS,
					textPipelineLayout,
					0,
					[]vk.DescriptorSet{textDescriptorSet},
					nil,
				)

				// Set viewport and scissor
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

				// Push constants (screen size and text color)
				// NOTE: Must match std140 layout - vec4 aligned to 16 bytes
				type TextPushConstants struct {
					ScreenWidth  float32
					ScreenHeight float32
					_padding1    float32 // Padding for vec4 alignment
					_padding2    float32 // Padding for vec4 alignment
					ColorR       float32
					ColorG       float32
					ColorB       float32
					ColorA       float32
				}
				pushData := TextPushConstants{
					ScreenWidth:  float32(swapExtent.Width),
					ScreenHeight: float32(swapExtent.Height),
					ColorR:       textComp.Color[0],
					ColorG:       textComp.Color[1],
					ColorB:       textComp.Color[2],
					ColorA:       textComp.Color[3],
				}
				cmd.CmdPushConstants(
					textPipelineLayout,
					vk.SHADER_STAGE_VERTEX_BIT,
					0,
					32, // 32 bytes: vec2 (8) + padding (8) + vec4 (16)
					unsafe.Pointer(&pushData),
				)

				// Bind vertex and index buffers
				cmd.BindVertexBuffers(0, []vk.Buffer{textVertexBuffer}, []uint64{0})
				cmd.BindIndexBuffer(textIndexBuffer, 0, vk.INDEX_TYPE_UINT16)

				// Draw text
				cmd.DrawIndexed(uint32(len(indices)), 1, 0, 0, 0)
			}

			cmd.EndRendering()

			// Transition swapchain: COLOR_ATTACHMENT → PRESENT
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
				vk.PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
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

		sdl.Delay(5)
	}
}
