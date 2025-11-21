package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// Push constants for layer transform
layout(push_constant) uniform LayerPushConstants {
    float opacity;       // Layer opacity (0.0 to 1.0)
    uint textureIndex;   // Index into bindless texture array
    vec2 offset;         // Layer position offset
    float scale;         // Layer scale
    vec2 cameraOffset;   // Camera pan offset
    float cameraZoom;    // Camera zoom level
    uint screenSpace;    // 1 = screen space (ignore camera), 0 = world space
} layer;

layout(location = 0) out vec2 fragTexCoord;

void main() {
    // Apply layer transform first
    vec2 pos = positions[gl_VertexIndex] * layer.scale + layer.offset;

    // Apply camera transform only if NOT in screen space
    if (layer.screenSpace == 0u) {
        pos = (pos - layer.cameraOffset) * layer.cameraZoom;
    }

    gl_Position = vec4(pos, 0.0, 1.0);
    fragTexCoord = texCoords[gl_VertexIndex];
}
`

const compositeFragmentShader = `
#version 450
#extension GL_EXT_nonuniform_qualifier : require

layout(location = 0) in vec2 fragTexCoord;

// MULTI-BINDING BINDLESS: 8 texture arrays for massive layer counts!
// Each binding holds 16K textures = 131K total textures
layout(set = 0, binding = 0) uniform sampler2D textures0[16384];  // Textures 0-16383
layout(set = 0, binding = 1) uniform sampler2D textures1[16384];  // Textures 16384-32767
layout(set = 0, binding = 2) uniform sampler2D textures2[16384];  // Textures 32768-49151
layout(set = 0, binding = 3) uniform sampler2D textures3[16384];  // Textures 49152-65535
layout(set = 0, binding = 4) uniform sampler2D textures4[16384];  // Textures 65536-81919
layout(set = 0, binding = 5) uniform sampler2D textures5[16384];  // Textures 81920-98303
layout(set = 0, binding = 6) uniform sampler2D textures6[16384];  // Textures 98304-114687
layout(set = 0, binding = 7) uniform sampler2D textures7[16384];  // Textures 114688-131071

// Push constant for layer opacity and texture index
layout(push_constant) uniform LayerPushConstants {
    float opacity;       // Layer opacity (0.0 to 1.0)
    uint textureIndex;   // Global index into bindless texture arrays
} layer;

layout(location = 0) out vec4 outColor;

void main() {
    // Map global index to (binding, arrayIndex)
    uint binding = layer.textureIndex / 16384u;
    uint arrayIndex = layer.textureIndex % 16384u;

    // Sample from the appropriate texture array
    vec4 texColor;
    if (binding == 0u) {
        texColor = texture(textures0[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 1u) {
        texColor = texture(textures1[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 2u) {
        texColor = texture(textures2[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 3u) {
        texColor = texture(textures3[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 4u) {
        texColor = texture(textures4[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 5u) {
        texColor = texture(textures5[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else if (binding == 6u) {
        texColor = texture(textures6[nonuniformEXT(arrayIndex)], fragTexCoord);
    } else { // binding == 7u
        texColor = texture(textures7[nonuniformEXT(arrayIndex)], fragTexCoord);
    }

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
layout(location = 1) out vec2 fragCanvasUV; // UV coordinates for canvas texture (0-1)

void main() {
    // Calculate brush quad corners in canvas pixels
    vec2 quadPos = inPosition; // 0,0 to 1,1
    vec2 canvasPos = brush.brushPos + (quadPos - 0.5) * brush.brushSize * 2.0;

    // Convert to NDC (-1 to 1)
    vec2 ndc = (canvasPos / brush.canvasSize) * 2.0 - 1.0;

    // Calculate UV coordinates for canvas sampling
    vec2 canvasUV = canvasPos / brush.canvasSize;

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragLocalPos = quadPos;
    fragCanvasUV = canvasUV;
}
`

const brushFragmentShader = `
#version 450

layout(location = 0) in vec2 fragLocalPos; // 0,0 to 1,1
layout(location = 1) in vec2 fragCanvasUV;  // UV for canvas texture

layout(push_constant) uniform BrushPushConstants {
    vec2 canvasSize;
    vec2 brushPos;
    float brushSize;
    float brushOpacity;
    vec4 brushColor;
} brush;

// Canvas texture for reading existing content (for blending)
layout(set = 0, binding = 0) uniform sampler2D canvasTexture;

layout(location = 0) out vec4 outColor;

void main() {
    // Calculate distance from center (0.5, 0.5)
    vec2 center = vec2(0.5, 0.5);
    float dist = length(fragLocalPos - center);

    // Circular brush with soft edges
    float radius = 0.5;
    float softness = 0.1;
    float brushAlpha = 1.0 - smoothstep(radius - softness, radius, dist);

	if (brush.brushSize < 1.0) {brushAlpha *= 10.0;}

    // Apply brush opacity
    brushAlpha *= brush.brushOpacity;

    // Sample existing canvas content
    vec4 canvasColor = texture(canvasTexture, fragCanvasUV);

    // Manual alpha blending: blend brush color over existing canvas
    // Formula: outColor = src * srcAlpha + dst * (1 - srcAlpha)
    vec4 brushColor = vec4(brush.brushColor.rgb, brushAlpha * brush.brushColor.a);
    float srcAlpha = brushColor.a;

    // Blend brush over existing canvas content
    outColor = brushColor * srcAlpha + canvasColor * (1.0 - srcAlpha);
    outColor.a = max(canvasColor.a, brushColor.a); // Preserve maximum alpha
}
`

const uiRectVertexShader = `
#version 450

// Input: quad vertices (0,0 to 1,1)
layout(location = 0) in vec2 inPosition;

// Push constants for UI rectangles
layout(push_constant) uniform UIRectPushConstants {
    float posX;
    float posY;
    float width;
    float height;
    float colorR;
    float colorG;
    float colorB;
    float colorA;
} rect;

layout(location = 0) out vec4 fragColor;

void main() {
    // Convert button rect to NDC
    // Button coords are in pixels (0,0 = top-left)
    vec2 screenSize = vec2(960.0, 960.0); // TODO: Pass as uniform

    // Calculate position in pixels
    vec2 pixelPos = vec2(rect.posX, rect.posY) + inPosition * vec2(rect.width, rect.height);

    // Convert to NDC (-1 to 1, with Y flipped for Vulkan)
    vec2 ndc = (pixelPos / screenSize) * 2.0 - 1.0;

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragColor = vec4(rect.colorR, rect.colorG, rect.colorB, rect.colorA);
}
`

const uiRectFragmentShader = `
#version 450

layout(location = 0) in vec4 fragColor;
layout(location = 0) out vec4 outColor;

void main() {
    outColor = fragColor;
}
`

// HSV to RGB conversion helper
const hsvHelperFunctions = `
vec3 hsv2rgb(vec3 c) {
    vec4 K = vec4(1.0, 2.0 / 3.0, 1.0 / 3.0, 3.0);
    vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
    return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
}
`

// Hue wheel shader - displays circular gradient of all hues
const hueWheelVertexShader = `
#version 450

layout(location = 0) in vec2 inPosition; // 0-1 range

layout(push_constant) uniform PushConstants {
    vec4 rect; // x, y, width, height in pixels
} pc;

layout(location = 0) out vec2 fragUV; // 0-1, will be centered to -1 to 1 in frag shader

void main() {
    // Convert from pixel coordinates to NDC
    vec2 screenSize = vec2(960.0, 960.0); // TODO: make dynamic
    vec2 pixelPos = pc.rect.xy + inPosition * pc.rect.zw;
    vec2 ndc = (pixelPos / screenSize) * 2.0 - 1.0;

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragUV = inPosition;
}
`

const hueWheelFragmentShader = `
#version 450
` + hsvHelperFunctions + `

layout(location = 0) in vec2 fragUV;
layout(location = 0) out vec4 outColor;

void main() {
    // Center UV coordinates (-1 to 1)
    vec2 centered = fragUV * 2.0 - 1.0;

    // Calculate distance from center
    float dist = length(centered);

    // Only draw in a ring (inner radius 0.6, outer radius 1.0)
    if (dist < 0.6 || dist > 1.0) {
        discard;
    }

    // Calculate angle (hue)
    float angle = atan(centered.y, centered.x);
    float hue = (angle + 3.14159265) / (2.0 * 3.14159265); // 0 to 1

    // Convert hue to RGB (full saturation, full value)
    vec3 color = hsv2rgb(vec3(hue, 1.0, 1.0));

    outColor = vec4(color, 1.0);
}
`

// SV box shader - displays saturation/value grid for selected hue
const svBoxVertexShader = `
#version 450

layout(location = 0) in vec2 inPosition; // 0-1 range

layout(push_constant) uniform PushConstants {
    vec4 rect; // x, y, width, height in pixels
    float hue; // selected hue (0-1)
} pc;

layout(location = 0) out vec2 fragUV;

void main() {
    vec2 screenSize = vec2(960.0, 960.0);
    vec2 pixelPos = pc.rect.xy + inPosition * pc.rect.zw;
    vec2 ndc = (pixelPos / screenSize) * 2.0 - 1.0;

    gl_Position = vec4(ndc, 0.0, 1.0);
    fragUV = inPosition;
}
`

const svBoxFragmentShader = `
#version 450
` + hsvHelperFunctions + `

layout(push_constant) uniform PushConstants {
    vec4 rect;
    float hue;
} pc;

layout(location = 0) in vec2 fragUV;
layout(location = 0) out vec4 outColor;

void main() {
    // X axis = saturation (0 = left/saturated, 1 = right/desaturated)
    // Y axis = value (0 = top/bright, 1 = bottom/dark)
    float saturation = 1.0 - fragUV.x; // Flip so left is saturated
    float value = 1.0 - fragUV.y;       // Flip so top is bright

    vec3 color = hsv2rgb(vec3(pc.hue, saturation, value));
    outColor = vec4(color, 1.0);
}
`

type Vertex struct {
	Pos      [2]float32
	Color    [3]float32
	TexCoord [2]float32
}

// ColorPicker state
type ColorPicker struct {
	Hue        float32 // 0.0 to 1.0
	Saturation float32 // 0.0 to 1.0
	Value      float32 // 0.0 to 1.0
	Visible    bool
}

// HSV to RGB conversion
func hsv2rgb(h, s, v float32) (r, g, b float32) {
	if s == 0 {
		return v, v, v
	}

	h = h * 6.0
	i := int(h)
	f := h - float32(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))

	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

// renderColorPicker renders the HSV color picker to the command buffer
func renderColorPicker(
	cmd vk.CommandBuffer,
	picker *ColorPicker,
	vertexBuffer vk.Buffer,
	hueWheelPipeline vk.Pipeline,
	hueWheelPipelineLayout vk.PipelineLayout,
	svBoxPipeline vk.Pipeline,
	svBoxPipelineLayout vk.PipelineLayout,
	swapExtent vk.Extent2D,
) {
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

	// Bind vertex buffer
	cmd.BindVertexBuffers(0, []vk.Buffer{vertexBuffer}, []uint64{0})

	// === Render Hue Wheel ===
	// Position in bottom-right corner, 200x200 pixels
	hueWheelX := float32(swapExtent.Width) - 220.0
	hueWheelY := float32(swapExtent.Height) - 220.0
	hueWheelSize := float32(200.0)

	cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, hueWheelPipeline)

	// Push constants: vec4 rect (x, y, width, height)
	hueWheelRect := [4]float32{hueWheelX, hueWheelY, hueWheelSize, hueWheelSize}
	cmd.CmdPushConstants(
		hueWheelPipelineLayout,
		vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
		0,
		16, // 4 floats * 4 bytes = 16 bytes
		unsafe.Pointer(&hueWheelRect[0]),
	)

	cmd.Draw(6, 1, 0, 0)

	// === Render SV Box ===
	// Position above the hue wheel, same X position
	svBoxX := hueWheelX
	svBoxY := hueWheelY - 220.0
	svBoxSize := float32(200.0)

	cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, svBoxPipeline)

	// Push constants: vec4 rect + float hue (20 bytes)
	type SVBoxPushConstants struct {
		Rect [4]float32 // x, y, width, height
		Hue  float32    // current hue value
	}

	svBoxPush := SVBoxPushConstants{
		Rect: [4]float32{svBoxX, svBoxY, svBoxSize, svBoxSize},
		Hue:  picker.Hue,
	}

	cmd.CmdPushConstants(
		svBoxPipelineLayout,
		vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
		0,
		20, // 5 floats * 4 bytes = 20 bytes
		unsafe.Pointer(&svBoxPush),
	)

	cmd.Draw(6, 1, 0, 0)
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

// evaluateQuadraticBezier evaluates a quadratic Bezier curve at parameter t (0 to 1).
// p0 = start point, p1 = control point, p2 = end point
func evaluateQuadraticBezier(p0, p1, p2, t float32) float32 {
	oneMinusT := 1.0 - t
	return oneMinusT*oneMinusT*p0 + 2.0*oneMinusT*t*p1 + t*t*p2
}

// CreateImageLayer creates a new layer entity with an image loaded from the specified file path
func CreateImageLayer(
	world *ecs.World,
	device *vk.Device,
	physicalDevice *vk.PhysicalDevice,
	commandPool *vk.CommandPool,
	queue *vk.Queue,
	pipeline *vk.Pipeline,
	pipelineLayout *vk.PipelineLayout,
	descriptorPool *vk.DescriptorPool,
	descriptorSetLayout *vk.DescriptorSetLayout,
	globalBindlessDescriptorSet vk.DescriptorSet,
	nextTextureIndex *uint32,
	maxTextures uint32,
	layerSampler vk.Sampler,
	swapExtent vk.Extent2D,
	imagePath string,
	zindex int,
) (ecs.Entity, error) {
	// Load image from file
	imageData, err := LoadImage(imagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to load image: %v", err)
	}

	fmt.Printf("Creating layer from: %s (%dx%d)\n", imagePath, imageData.Width, imageData.Height)

	textureWidth := imageData.Width
	textureHeight := imageData.Height
	textureData := imageData.Pixels
	textureSize := uint64(len(textureData))

	// Create staging buffer
	stagingBuffer, stagingMemory, err := device.CreateBufferWithMemory(
		textureSize,
		vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
		vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
		*physicalDevice,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create staging buffer: %v", err)
	}
	defer device.DestroyBuffer(stagingBuffer)
	defer device.FreeMemory(stagingMemory)

	// Upload texture data to staging buffer
	err = device.UploadToBuffer(stagingMemory, textureData)
	if err != nil {
		return 0, fmt.Errorf("failed to upload to buffer: %v", err)
	}

	// Create texture image
	textureImage, textureMemory, err := device.CreateImageWithMemory(
		textureWidth, textureHeight,
		vk.FORMAT_R8G8B8A8_SRGB,
		vk.IMAGE_TILING_OPTIMAL,
		vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
		vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
		*physicalDevice,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create image: %v", err)
	}

	// Create command buffer for texture upload
	uploadCmdBuffer, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        *commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to allocate command buffer: %v", err)
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
		[]vk.ImageMemoryBarrier{{
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
		}},
	)

	// Copy buffer to image
	uploadCmd.CopyBufferToImage(stagingBuffer, textureImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, []vk.BufferImageCopy{{
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
	}})

	// Transition to shader read
	uploadCmd.PipelineBarrier(
		vk.PIPELINE_STAGE_TRANSFER_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{{
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
		}},
	)

	uploadCmd.End()

	// Submit and wait
	err = queue.Submit([]vk.SubmitInfo{{CommandBuffers: []vk.CommandBuffer{uploadCmd}}}, vk.Fence{})
	if err != nil {
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to submit command buffer: %v", err)
	}
	queue.WaitIdle()
	device.FreeCommandBuffers(*commandPool, []vk.CommandBuffer{uploadCmd})

	// Create image view
	textureImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
		Image:    textureImage,
		ViewType: vk.IMAGE_VIEW_TYPE_2D,
		Format:   vk.FORMAT_R8G8B8A8_SRGB,
		SubresourceRange: vk.ImageSubresourceRange{
			AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
			BaseMipLevel:   0,
			LevelCount:     1,
			BaseArrayLayer: 0,
			LayerCount:     1,
		},
	})
	if err != nil {
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to create image view: %v", err)
	}

	// Create sampler
	textureSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
		MagFilter:        vk.FILTER_LINEAR,
		MinFilter:        vk.FILTER_LINEAR,
		AddressModeU:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		AddressModeV:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		AddressModeW:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
		AnisotropyEnable: false,
		BorderColor:      vk.BORDER_COLOR_INT_OPAQUE_BLACK,
		MipmapMode:       vk.SAMPLER_MIPMAP_MODE_LINEAR,
	})
	if err != nil {
		device.DestroyImageView(textureImageView)
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to create sampler: %v", err)
	}

	// Create ECS entity
	entity := world.CreateEntity()

	// Add Transform component (centered)
	transform := ecs.NewTransform()
	transform.ZIndex = zindex // Ensure dropped layers appear on top
	world.AddTransform(entity, transform)

	// Add VulkanPipeline component
	world.AddVulkanPipeline(entity, &ecs.VulkanPipeline{
		Pipeline:            *pipeline,
		PipelineLayout:      *pipelineLayout,
		DescriptorPool:      *descriptorPool,
		DescriptorSet:       vk.DescriptorSet{}, // Created on-demand
		DescriptorSetLayout: *descriptorSetLayout,
	})

	// Add TextureData component
	world.AddTextureData(entity, &ecs.TextureData{
		Image:       textureImage,
		ImageView:   textureImageView,
		ImageMemory: textureMemory,
		Sampler:     textureSampler,
	})

	// Add BlendMode component (default 100% opacity)
	world.AddBlendMode(entity, ecs.NewBlendMode())

	// Create RenderTarget framebuffer for this layer
	layerImage, layerImageMemory, err := device.CreateImageWithMemory(
		swapExtent.Width,
		swapExtent.Height,
		vk.FORMAT_R8G8B8A8_UNORM,
		vk.IMAGE_TILING_OPTIMAL,
		vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
		vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
		*physicalDevice,
	)
	if err != nil {
		// Clean up already-created resources
		device.DestroySampler(textureSampler)
		device.DestroyImageView(textureImageView)
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to create layer framebuffer: %v", err)
	}

	layerImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
		Image:    layerImage,
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
		device.DestroyImage(layerImage)
		device.FreeMemory(layerImageMemory)
		device.DestroySampler(textureSampler)
		device.DestroyImageView(textureImageView)
		device.DestroyImage(textureImage)
		device.FreeMemory(textureMemory)
		return 0, fmt.Errorf("failed to create layer framebuffer view: %v", err)
	}

	// Add RenderTarget component
	world.AddRenderTarget(entity, &ecs.RenderTarget{
		Image:       layerImage,
		ImageView:   layerImageView,
		ImageMemory: layerImageMemory,
		Format:      vk.FORMAT_R8G8B8A8_UNORM,
		Width:       swapExtent.Width,
		Height:      swapExtent.Height,
	})

	// Transition RenderTarget image to SHADER_READ_ONLY_OPTIMAL for compositing
	transitionCmd, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
		CommandPool:        *commandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	})
	if err != nil {
		return entity, fmt.Errorf("failed to allocate transition command buffer: %v", err)
	}
	cmd := transitionCmd[0]

	cmd.Begin(&vk.CommandBufferBeginInfo{Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT})
	cmd.PipelineBarrier(
		vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
		vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
		0,
		[]vk.ImageMemoryBarrier{{
			SrcAccessMask:       vk.ACCESS_NONE,
			DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
			OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
			NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
			SrcQueueFamilyIndex: ^uint32(0),
			DstQueueFamilyIndex: ^uint32(0),
			Image:               layerImage,
			SubresourceRange: vk.ImageSubresourceRange{
				AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
				BaseMipLevel:   0,
				LevelCount:     1,
				BaseArrayLayer: 0,
				LayerCount:     1,
			},
		}},
	)
	cmd.End()

	err = queue.Submit([]vk.SubmitInfo{{CommandBuffers: []vk.CommandBuffer{cmd}}}, vk.Fence{})
	if err != nil {
		return entity, fmt.Errorf("failed to submit transition command: %v", err)
	}
	queue.WaitIdle()
	device.FreeCommandBuffers(*commandPool, []vk.CommandBuffer{cmd})

	// BINDLESS: Assign texture index and upload to global descriptor set
	textureIndex := *nextTextureIndex
	*nextTextureIndex++
	if *nextTextureIndex >= maxTextures {
		return entity, fmt.Errorf("exceeded maximum texture count: %d", maxTextures)
	}
	fmt.Printf("Assigned texture index %d to new layer\n", textureIndex)

	// Upload SOURCE TEXTURE to global bindless descriptor set
	// (For simple image layers, we composite the source directly, not a rendered framebuffer)
	// Calculate which binding and array index to use for multi-binding architecture
	const texturesPerBinding = 16384
	binding := textureIndex / texturesPerBinding
	arrayElement := textureIndex % texturesPerBinding
	device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
		{
			DstSet:          globalBindlessDescriptorSet,
			DstBinding:      binding,
			DstArrayElement: arrayElement,
			DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
			ImageInfo: []vk.DescriptorImageInfo{{
				Sampler:     textureSampler,
				ImageView:   textureImageView,
				ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
			}},
		},
	})

	// Update TextureData with texture index
	texData := world.GetTextureData(entity)
	texData.TextureIndex = textureIndex

	fmt.Printf("Created layer entity %d from %s\n", entity, imagePath)
	return entity, nil
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

		// Query physical device features to check for sparse binding support
		features := physicalDevice.GetFeatures()
		fmt.Printf("\nSparse binding support:\n")
		fmt.Printf("  sparseBinding: %v\n", features.SparseBinding)
		fmt.Printf("  sparseResidencyImage2D: %v\n", features.SparseResidencyImage2D)

		if !features.SparseBinding || !features.SparseResidencyImage2D {
			fmt.Println("WARNING: Sparse binding not supported! Falling back to dense canvas.")
		}

		// Create device with sparse binding enabled
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
			EnabledFeatures: &vk.PhysicalDeviceFeatures{
				SparseBinding:          features.SparseBinding,          // Enable sparse binding
				SparseResidencyImage2D: features.SparseResidencyImage2D, // Enable sparse residency for 2D images
			},
			Vulkan12Features: &vk.PhysicalDeviceVulkan12Features{
				DescriptorIndexing:                        true, // Enable descriptor indexing for bindless textures
				ShaderSampledImageArrayNonUniformIndexing: true, // Allow nonuniformEXT in shaders
				DescriptorBindingPartiallyBound:           true, // Allow partially bound descriptor sets
				RuntimeDescriptorArray:                    true, // Enable runtime-sized descriptor arrays
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

		// Query device limits for bindless textures
		deviceProps := instance.GetPhysicalDeviceProperties(physicalDevice)
		gpuLimit := deviceProps.Limits.MaxDescriptorSetSampledImages

		// Cap at 16K for cross-platform compatibility (Windows WDDM can be restrictive)
		// Most apps won't need more than a few thousand textures anyway
		const maxTexturesCap = 16384
		maxTextures := gpuLimit
		if maxTextures > maxTexturesCap {
			maxTextures = maxTexturesCap
		}

		fmt.Printf("\n🎨 BINDLESS TEXTURES ENABLED!\n")
		fmt.Printf("GPU reports %d sampled images, using %d (capped for compatibility)\n", gpuLimit, maxTextures)

		const texturesPerBinding = 16384
		// Create MULTI-BINDING BINDLESS descriptor set layout
		// 8 bindings × 16K textures = 131K total texture capacity
		descriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
			Bindings: []vk.DescriptorSetLayoutBinding{
				{
					Binding:         0,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         1,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         2,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         3,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         4,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         5,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         6,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         7,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
			},
			BindingFlags: []vk.DescriptorBindingFlagBits{
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 0
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 1
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 2
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 3
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 4
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 5
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 6
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 7
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
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       24, // vec2 (8) + float (4) + float (4) + float (4) + uint (4) = 24 bytes
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

		// Create composite descriptor set layout (MULTI-BINDING BINDLESS!)
		// 8 bindings × 16K textures = 131K total texture capacity
		compositeDescriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
			Bindings: []vk.DescriptorSetLayoutBinding{
				{
					Binding:         0,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         1,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         2,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         3,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         4,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         5,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         6,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
				{
					Binding:         7,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: texturesPerBinding,
					StageFlags:      vk.SHADER_STAGE_FRAGMENT_BIT,
				},
			},
			BindingFlags: []vk.DescriptorBindingFlagBits{
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 0
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 1
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 2
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 3
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 4
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 5
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 6
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT, // Binding 7
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
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       48, // opacity(4) + textureIndex(4) + offset(8) + scale(4) + padding(4) + cameraOffset(8) + cameraZoom(4) + screenSpace(4) + padding(8) = 48 bytes (16-byte aligned)
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

		// Create descriptor set layout for canvas texture sampling
		brushDescriptorSetLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
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
		defer device.DestroyDescriptorSetLayout(brushDescriptorSetLayout)

		// Brush push constants: vec2 canvasSize, vec2 brushPos, float size, float opacity, vec4 color
		// Total: 8 + 8 + 4 + 4 + 16 = 40 bytes
		brushPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			SetLayouts: []vk.DescriptorSetLayout{brushDescriptorSetLayout},
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

		// Create descriptor pool for brush canvas texture
		brushDescriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
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
		defer device.DestroyDescriptorPool(brushDescriptorPool)

		// Allocate descriptor set for brush canvas texture
		brushDescriptorSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: brushDescriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{brushDescriptorSetLayout},
		})
		if err != nil {
			panic(err)
		}
		brushDescriptorSet := brushDescriptorSets[0]
		// Will be updated after canvas creation to bind source canvas texture

		fmt.Println("Brush pipeline created!")

		// ===== UI Rectangle Pipeline =====
		// Compile UI rectangle shaders
		uiRectVertResult, err := compiler.CompileIntoSPV(uiRectVertexShader, "ui_rect.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(fmt.Sprintf("UI rect vertex shader compilation failed: %v", err))
		}
		uiRectVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: uiRectVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(uiRectVertShader)

		uiRectFragResult, err := compiler.CompileIntoSPV(uiRectFragmentShader, "ui_rect.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(fmt.Sprintf("UI rect fragment shader compilation failed: %v", err))
		}
		uiRectFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: uiRectFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(uiRectFragShader)

		// UI rect push constants: posX, posY, width, height, colorR, colorG, colorB, colorA
		// Total: 8 floats = 32 bytes
		uiRectPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       32, // 8 floats
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(uiRectPipelineLayout)

		// Create UI rectangle graphics pipeline
		uiRectPipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: uiRectVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: uiRectFragShader, Name: "main"},
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
			Layout: uiRectPipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{vk.FORMAT_R8G8B8A8_UNORM}, // UI layer format
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(uiRectPipeline)

		fmt.Println("UI rectangle pipeline created!")

		// ===== Color Picker Pipelines =====
		fmt.Println("Creating color picker pipelines...")

		// Compile hue wheel shaders
		hueWheelVertResult, err := compiler.CompileIntoSPV(hueWheelVertexShader, "hue_wheel.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(fmt.Sprintf("Hue wheel vertex shader compilation failed: %v", err))
		}
		hueWheelVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: hueWheelVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(hueWheelVertShader)

		hueWheelFragResult, err := compiler.CompileIntoSPV(hueWheelFragmentShader, "hue_wheel.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(fmt.Sprintf("Hue wheel fragment shader compilation failed: %v", err))
		}
		hueWheelFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: hueWheelFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(hueWheelFragShader)

		// Hue wheel pipeline layout (push constants: vec4 rect = 16 bytes)
		hueWheelPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       16, // vec4 rect
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(hueWheelPipelineLayout)

		// Hue wheel pipeline
		hueWheelPipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: hueWheelVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: hueWheelFragShader, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				VertexBindingDescriptions: []vk.VertexInputBindingDescription{
					{Binding: 0, Stride: 8, InputRate: vk.VERTEX_INPUT_RATE_VERTEX}, // vec2 position
				},
				VertexAttributeDescriptions: []vk.VertexInputAttributeDescription{
					{Location: 0, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 0},
				},
			},
			InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
				Topology: vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
			},
			ViewportState: &vk.PipelineViewportStateCreateInfo{
				Viewports: nil, // Dynamic viewport
				Scissors:  nil, // Dynamic scissor
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
						ColorWriteMask:      vk.COLOR_COMPONENT_R_BIT | vk.COLOR_COMPONENT_G_BIT | vk.COLOR_COMPONENT_B_BIT | vk.COLOR_COMPONENT_A_BIT,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{vk.DYNAMIC_STATE_VIEWPORT, vk.DYNAMIC_STATE_SCISSOR},
			},
			Layout: hueWheelPipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{swapFormat},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(hueWheelPipeline)

		// Compile SV box shaders
		svBoxVertResult, err := compiler.CompileIntoSPV(svBoxVertexShader, "sv_box.vert", shaderc.VertexShader, options)
		if err != nil {
			panic(fmt.Sprintf("SV box vertex shader compilation failed: %v", err))
		}
		svBoxVertShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: svBoxVertResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(svBoxVertShader)

		svBoxFragResult, err := compiler.CompileIntoSPV(svBoxFragmentShader, "sv_box.frag", shaderc.FragmentShader, options)
		if err != nil {
			panic(fmt.Sprintf("SV box fragment shader compilation failed: %v", err))
		}
		svBoxFragShader, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{Code: svBoxFragResult.GetBytes()})
		if err != nil {
			panic(err)
		}
		defer device.DestroyShaderModule(svBoxFragShader)

		// SV box pipeline layout (push constants: vec4 rect + float hue = 20 bytes)
		svBoxPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_VERTEX_BIT | vk.SHADER_STAGE_FRAGMENT_BIT,
					Offset:     0,
					Size:       20, // vec4 rect + float hue
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipelineLayout(svBoxPipelineLayout)

		// SV box pipeline (same config as hue wheel)
		svBoxPipeline, err := device.CreateGraphicsPipeline(&vk.GraphicsPipelineCreateInfo{
			Stages: []vk.PipelineShaderStageCreateInfo{
				{Stage: vk.SHADER_STAGE_VERTEX_BIT, Module: svBoxVertShader, Name: "main"},
				{Stage: vk.SHADER_STAGE_FRAGMENT_BIT, Module: svBoxFragShader, Name: "main"},
			},
			VertexInputState: &vk.PipelineVertexInputStateCreateInfo{
				VertexBindingDescriptions: []vk.VertexInputBindingDescription{
					{Binding: 0, Stride: 8, InputRate: vk.VERTEX_INPUT_RATE_VERTEX},
				},
				VertexAttributeDescriptions: []vk.VertexInputAttributeDescription{
					{Location: 0, Binding: 0, Format: vk.FORMAT_R32G32_SFLOAT, Offset: 0},
				},
			},
			InputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
				Topology: vk.PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
			},
			ViewportState: &vk.PipelineViewportStateCreateInfo{
				Viewports: nil, // Dynamic viewport
				Scissors:  nil, // Dynamic scissor
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
						ColorWriteMask:      vk.COLOR_COMPONENT_R_BIT | vk.COLOR_COMPONENT_G_BIT | vk.COLOR_COMPONENT_B_BIT | vk.COLOR_COMPONENT_A_BIT,
					},
				},
			},
			DynamicState: &vk.PipelineDynamicStateCreateInfo{
				DynamicStates: []vk.DynamicState{vk.DYNAMIC_STATE_VIEWPORT, vk.DYNAMIC_STATE_SCISSOR},
			},
			Layout: svBoxPipelineLayout,
			RenderingInfo: &vk.PipelineRenderingCreateInfo{
				ColorAttachmentFormats: []vk.Format{swapFormat},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyPipeline(svBoxPipeline)

		fmt.Println("Color picker pipelines created!")

		// Create MULTI-BINDING BINDLESS descriptor pool
		// 1 set with 8 bindings × 16K textures = 131K total capacity
		const numBindings = 8
		totalTextureCapacity := texturesPerBinding * numBindings
		fmt.Printf("Creating multi-binding bindless descriptor pool:\n")
		fmt.Printf("  - %d bindings × %d textures = %d total capacity\n", numBindings, texturesPerBinding, totalTextureCapacity)
		bindlessDescriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 1, // Only ONE descriptor set needed for bindless!
			PoolSizes: []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, DescriptorCount: uint32(totalTextureCapacity)},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorPool(bindlessDescriptorPool)

		// Allocate THE global bindless descriptor set
		bindlessDescriptorSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
			DescriptorPool: bindlessDescriptorPool,
			SetLayouts:     []vk.DescriptorSetLayout{descriptorSetLayout},
		})
		if err != nil {
			panic(err)
		}
		globalBindlessDescriptorSet := bindlessDescriptorSets[0]
		fmt.Printf("✨ Global bindless descriptor set allocated!\n\n")

		// Texture index management for bindless rendering
		var nextTextureIndex uint32 = 0
		assignNextTextureIndex := func() uint32 {
			idx := nextTextureIndex
			nextTextureIndex++
			if nextTextureIndex >= maxTextures {
				panic(fmt.Sprintf("Exceeded maximum texture count: %d", maxTextures))
			}
			return idx
		}
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
			{Pos: [2]float32{-1.0, -1.0}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{0.0, 0.0}}, // Bottom-left
			{Pos: [2]float32{1.0, -1.0}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{1.0, 0.0}},  // Bottom-right
			{Pos: [2]float32{1.0, 1.0}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{1.0, 1.0}},   // Top-right
			{Pos: [2]float32{-1.0, 1.0}, Color: [3]float32{1.0, 1.0, 1.0}, TexCoord: [2]float32{0.0, 1.0}},  // Top-left
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

		// Create vertex buffer for color picker (simple quad: 0-1 positions)
		colorPickerVertices := []float32{
			// Position (x, y)
			0.0, 0.0,
			1.0, 0.0,
			1.0, 1.0,
			0.0, 0.0,
			1.0, 1.0,
			0.0, 1.0,
		}

		colorPickerVertexBufferSize := uint64(len(colorPickerVertices) * 4)
		colorPickerVertexBuffer, colorPickerVertexMemory, err := device.CreateBufferWithMemory(
			colorPickerVertexBufferSize,
			vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(colorPickerVertexBuffer)
		defer device.FreeMemory(colorPickerVertexMemory)

		// Upload color picker vertex data
		colorPickerVertexData := (*[1 << 30]byte)(unsafe.Pointer(&colorPickerVertices[0]))[:colorPickerVertexBufferSize:colorPickerVertexBufferSize]
		err = device.UploadToBuffer(colorPickerVertexMemory, colorPickerVertexData)
		if err != nil {
			panic(err)
		}

		fmt.Println("Color picker vertex buffer created!")

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
		fmt.Println("\nCreating paint canvases (ping-pong buffers)...")

		// Canvas A - Front buffer
		paintCanvasA, err := canvas.New(canvas.Config{
			Device:         device,
			PhysicalDevice: physicalDevice,
			Width:          2048,
			Height:         2048,
			Format:         vk.FORMAT_R8G8B8A8_UNORM,
			Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
				vk.IMAGE_USAGE_SAMPLED_BIT |
				vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT |
				vk.IMAGE_USAGE_TRANSFER_SRC_BIT | // Add TRANSFER_SRC for Download
				vk.IMAGE_USAGE_STORAGE_BIT, // For compute shader access
			UseSparseBinding: true, // SPARSE BINDING ENABLED! RTX 2000+ only
		}, commandPool, queue)
		if err != nil {
			panic(fmt.Sprintf("Failed to create paint canvas A: %v", err))
		}
		defer paintCanvasA.Destroy()

		// Canvas B - Back buffer
		paintCanvasB, err := canvas.New(canvas.Config{
			Device:         device,
			PhysicalDevice: physicalDevice,
			Width:          2048,
			Height:         2048,
			Format:         vk.FORMAT_R8G8B8A8_UNORM,
			Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
				vk.IMAGE_USAGE_SAMPLED_BIT |
				vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT |
				vk.IMAGE_USAGE_TRANSFER_SRC_BIT |
				vk.IMAGE_USAGE_STORAGE_BIT, // For compute shader access
			UseSparseBinding: true,
		}, commandPool, queue)
		if err != nil {
			panic(fmt.Sprintf("Failed to create paint canvas B: %v", err))
		}
		defer paintCanvasB.Destroy()

		fmt.Printf("Paint canvases created: %dx%d (ping-pong buffers)\n", paintCanvasA.GetWidth(), paintCanvasA.GetHeight())

		// Pre-allocate all sparse pages
		fmt.Printf("%d: Starting canvas initialization...\n", time.Now().UnixMilli())
		err = paintCanvasA.AllocateAll()
		if err != nil {
			panic(fmt.Sprintf("Failed to allocate canvas A pages: %v", err))
		}
		err = paintCanvasB.AllocateAll()
		if err != nil {
			panic(fmt.Sprintf("Failed to allocate canvas B pages: %v", err))
		}

		// Use compute shader for fast canvas clear (much faster!)
		clearCompSource := `#version 450
layout(local_size_x = 16, local_size_y = 16) in;
layout(binding = 0, rgba8) uniform writeonly image2D outputImage;
layout(push_constant) uniform PushConstants {
    vec4 clearColor;
    uvec2 imageSize;
} pc;
void main() {
    uvec2 pixelCoord = gl_GlobalInvocationID.xy;
    if (pixelCoord.x >= pc.imageSize.x || pixelCoord.y >= pc.imageSize.y) return;
    imageStore(outputImage, ivec2(pixelCoord), pc.clearColor);
}`

		clearCompResult, err := compiler.CompileIntoSPV(clearCompSource, "clear.comp", shaderc.ComputeShader, options)
		if err != nil {
			panic(fmt.Sprintf("Failed to compile clear compute shader: %v", err))
		}
		defer clearCompResult.Release()

		clearCompModule, err := device.CreateShaderModule(&vk.ShaderModuleCreateInfo{
			Code: clearCompResult.GetBytes(),
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create clear compute shader module: %v", err))
		}
		defer device.DestroyShaderModule(clearCompModule)

		// Create descriptor set layout for compute shader (single storage image)
		clearDescLayout, err := device.CreateDescriptorSetLayout(&vk.DescriptorSetLayoutCreateInfo{
			Bindings: []vk.DescriptorSetLayoutBinding{
				{
					Binding:         0,
					DescriptorType:  vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
					DescriptorCount: 1,
					StageFlags:      vk.SHADER_STAGE_COMPUTE_BIT,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create clear descriptor layout: %v", err))
		}
		defer device.DestroyDescriptorSetLayout(clearDescLayout)

		// Create compute pipeline layout with push constants
		clearPipelineLayout, err := device.CreatePipelineLayout(&vk.PipelineLayoutCreateInfo{
			SetLayouts: []vk.DescriptorSetLayout{clearDescLayout},
			PushConstantRanges: []vk.PushConstantRange{
				{
					StageFlags: vk.SHADER_STAGE_COMPUTE_BIT,
					Offset:     0,
					Size:       24, // vec4(16) + uvec2(8) = 24 bytes
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create clear pipeline layout: %v", err))
		}
		defer device.DestroyPipelineLayout(clearPipelineLayout)

		// Create compute pipeline
		clearPipeline, err := device.CreateComputePipeline(&vk.ComputePipelineCreateInfo{
			Stage: vk.PipelineShaderStageCreateInfo{
				Stage:  vk.SHADER_STAGE_COMPUTE_BIT,
				Module: clearCompModule,
				Name:   "main",
			},
			Layout: clearPipelineLayout,
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create clear compute pipeline: %v", err))
		}
		defer device.DestroyComputePipeline(clearPipeline)

		// Create descriptor pool for clear operation
		clearDescPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 2, // One for each canvas
			PoolSizes: []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_STORAGE_IMAGE, DescriptorCount: 2},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create clear descriptor pool: %v", err))
		}
		defer device.DestroyDescriptorPool(clearDescPool)

		// Clear both canvases using compute shader
		fmt.Printf("%d: Clearing canvases with compute shader...\n", time.Now().UnixMilli())

		for _, cvs := range []canvas.Canvas{paintCanvasA, paintCanvasB} {
			// Allocate descriptor set
			descSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
				DescriptorPool: clearDescPool,
				SetLayouts:     []vk.DescriptorSetLayout{clearDescLayout},
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to allocate clear descriptor set: %v", err))
			}
			descSet := descSets[0]

			// Update descriptor set to bind canvas image
			device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
				{
					DstSet:         descSet,
					DstBinding:     0,
					DescriptorType: vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
					ImageInfo: []vk.DescriptorImageInfo{{
						ImageView:   cvs.GetView(),
						ImageLayout: vk.IMAGE_LAYOUT_GENERAL,
					}},
				},
			})

			// Create command buffer
			cmdBufs, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
				CommandPool:        commandPool,
				Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
				CommandBufferCount: 1,
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to allocate clear command buffer: %v", err))
			}
			cmd := cmdBufs[0]

			cmd.Begin(&vk.CommandBufferBeginInfo{
				Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
			})

			// Transition to GENERAL layout for compute
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_COMPUTE_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{
					{
						SrcAccessMask:       0,
						DstAccessMask:       vk.ACCESS_SHADER_WRITE_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
						NewLayout:           vk.IMAGE_LAYOUT_GENERAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               cvs.GetImage(),
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

			// Bind compute pipeline and descriptor set
			cmd.BindPipeline(vk.PIPELINE_BIND_POINT_COMPUTE, clearPipeline)
			cmd.BindDescriptorSets(vk.PIPELINE_BIND_POINT_COMPUTE, clearPipelineLayout, 0, []vk.DescriptorSet{descSet}, nil)

			// Push constants (white color + image size)
			pushData := []byte{
				0x00, 0x00, 0x80, 0x3F, // clearColor.r = 1.0 (float32 little-endian)
				0x00, 0x00, 0x80, 0x3F, // clearColor.g = 1.0
				0x00, 0x00, 0x80, 0x3F, // clearColor.b = 1.0
				0x00, 0x00, 0x80, 0x3F, // clearColor.a = 1.0
				0x00, 0x0f, 0x00, 0x00, // imageSize.x = 2048 (uint32 little-endian)
				0x00, 0x0f, 0x00, 0x00, // imageSize.y = 2048 (uint32 little-endian)
			}
			cmd.PushConstants(clearPipelineLayout, vk.SHADER_STAGE_COMPUTE_BIT, 0, pushData)

			// Dispatch compute shader (2048/16 = 512 work groups per dimension)
			cmd.Dispatch(128, 128, 1)

			// Transition to SHADER_READ_ONLY for fragment shader access
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_COMPUTE_SHADER_BIT,
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{
					{
						SrcAccessMask:       vk.ACCESS_SHADER_WRITE_BIT,
						DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_GENERAL,
						NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               cvs.GetImage(),
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

			// Submit and wait
			queue.Submit([]vk.SubmitInfo{
				{CommandBuffers: []vk.CommandBuffer{cmd}},
			}, vk.Fence{})
			queue.WaitIdle()

			device.FreeCommandBuffers(commandPool, cmdBufs)
		}

		fmt.Printf("%d: Both canvases cleared with compute shader!\n", time.Now().UnixMilli())

		// Note: No GPU undo buffer needed - using action-based undo with stroke replay

		// Ping-pong state: tracks which canvas is source (read) and which is destination (write)
		paintCanvas := paintCanvasA       // Start with A as current (destination)
		paintCanvasSource := paintCanvasB // B is source for first frame

		// Create sampler for paint canvas with anisotropic filtering for smoother zoom
		canvasSampler, err := device.CreateSampler(&vk.SamplerCreateInfo{
			MagFilter:        vk.FILTER_LINEAR,
			MinFilter:        vk.FILTER_LINEAR,
			MipmapMode:       vk.SAMPLER_MIPMAP_MODE_LINEAR,
			AddressModeU:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeV:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			AddressModeW:     vk.SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
			MaxLod:           1.0,
			AnisotropyEnable: true,
			MaxAnisotropy:    16.0, // High quality filtering for zoomed views
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create canvas sampler: %v", err))
		}
		defer device.DestroySampler(canvasSampler)

		// Update brush descriptor set to bind source canvas texture
		// This will be updated each frame to bind the correct source canvas
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          brushDescriptorSet,
				DstBinding:      0,
				DstArrayElement: 0,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{
					{
						Sampler:     canvasSampler,
						ImageView:   paintCanvasSource.GetView(),
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					},
				},
			},
		})
		fmt.Println("Brush descriptor set updated with source canvas texture")

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

		// Create DEVICE_LOCAL main buffers (GPU-only, fast rendering)
		textVertexBuffer, textVertexMemory, err := device.CreateBufferWithMemory(
			textVertexBufferSize,
			vk.BUFFER_USAGE_VERTEX_BUFFER_BIT|vk.BUFFER_USAGE_TRANSFER_DST_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textVertexBuffer)
		defer device.FreeMemory(textVertexMemory)

		textIndexBuffer, textIndexMemory, err := device.CreateBufferWithMemory(
			textIndexBufferSize,
			vk.BUFFER_USAGE_INDEX_BUFFER_BIT|vk.BUFFER_USAGE_TRANSFER_DST_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textIndexBuffer)
		defer device.FreeMemory(textIndexMemory)

		// Create HOST_VISIBLE staging buffers (CPU-writable, for uploading data)
		textStagingVertexBuffer, textStagingVertexMemory, err := device.CreateBufferWithMemory(
			textVertexBufferSize,
			vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textStagingVertexBuffer)
		defer device.FreeMemory(textStagingVertexMemory)

		textStagingIndexBuffer, textStagingIndexMemory, err := device.CreateBufferWithMemory(
			textIndexBufferSize,
			vk.BUFFER_USAGE_TRANSFER_SRC_BIT,
			vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(err)
		}
		defer device.DestroyBuffer(textStagingIndexBuffer)
		defer device.FreeMemory(textStagingIndexMemory)

		// Create TextRenderer for UI button labels
		textRenderer := &systems.TextRenderer{
			Pipeline:            textPipeline,
			PipelineLayout:      textPipelineLayout,
			DescriptorSet:       textDescriptorSet,
			Atlas:               sdfAtlas,
			VertexBuffer:        textVertexBuffer,
			VertexMemory:        textVertexMemory,
			IndexBuffer:         textIndexBuffer,
			IndexMemory:         textIndexMemory,
			StagingVertexBuffer: textStagingVertexBuffer,
			StagingVertexMemory: textStagingVertexMemory,
			StagingIndexBuffer:  textStagingIndexBuffer,
			StagingIndexMemory:  textStagingIndexMemory,
			MaxChars:            maxTextChars,
		}
		fmt.Println("TextRenderer initialized for UI labels!")

		// Create descriptor set layout

		// Create descriptor pool
		descriptorPool, err := device.CreateDescriptorPool(&vk.DescriptorPoolCreateInfo{
			MaxSets: 100,
			PoolSizes: []vk.DescriptorPoolSize{
				{
					Type:            vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					DescriptorCount: 100,
				},
			},
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyDescriptorPool(descriptorPool)

		// Allocate command buffers (one per swapchain image)
		commandBuffers, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
			CommandPool:        commandPool,
			Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
			CommandBufferCount: uint32(len(swapImages)),
		})
		if err != nil {
			panic(err)
		}

		const FRAMES_IN_FLIGHT = 3

		var (
			renderFinishedSems [FRAMES_IN_FLIGHT]vk.Semaphore
			imageAvailableSems [FRAMES_IN_FLIGHT]vk.Semaphore
		)

		for i := range renderFinishedSems {
			renderFinishedSems[i], err = device.CreateSemaphore(&vk.SemaphoreCreateInfo{})
			if err != nil {
				panic(err)
			}
			defer device.DestroySemaphore(renderFinishedSems[i])

		}

		for i := range imageAvailableSems {
			imageAvailableSems[i], err = device.CreateSemaphore(&vk.SemaphoreCreateInfo{})
			if err != nil {
				panic(err)
			}
			defer device.DestroySemaphore(imageAvailableSems[i])

		}

		inFlightFences := make([]vk.Fence, len(swapImages))
		for i := range inFlightFences {
			inFlightFences[i], err = device.CreateFence(&vk.FenceCreateInfo{
				Flags: vk.FENCE_CREATE_SIGNALED_BIT,
			})
			if err != nil {
				panic(err)
			}
			defer device.DestroyFence(inFlightFences[i])
		}

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
		transform1.ZIndex = -5000000 // Layer 1 is in the back
		world.AddTransform(layer1, transform1)

		// Add VulkanPipeline component
		world.AddVulkanPipeline(layer1, &ecs.VulkanPipeline{
			Pipeline:            pipeline,
			PipelineLayout:      pipelineLayout,
			DescriptorPool:      descriptorPool,
			DescriptorSet:       vk.DescriptorSet{}, // Will be created on-demand
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add TextureData component (Djungelskog texture)
		// Assign bindless texture index for layer1
		layer1TextureIndex := assignNextTextureIndex()
		fmt.Printf("Assigned texture index %d to layer1\n", layer1TextureIndex)

		// Upload texture to global bindless descriptor set
		// Calculate which binding and array index to use for multi-binding architecture
		binding1 := layer1TextureIndex / texturesPerBinding
		arrayElement1 := layer1TextureIndex % texturesPerBinding
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          globalBindlessDescriptorSet,
				DstBinding:      binding1,
				DstArrayElement: arrayElement1,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{{
					Sampler:     textureSampler,
					ImageView:   textureImageView,
					ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				}},
			},
		})

		world.AddTextureData(layer1, &ecs.TextureData{
			Image:        textureImage,
			ImageView:    textureImageView,
			ImageMemory:  textureMemory,
			Sampler:      textureSampler,
			Width:        imageData.Width,
			Height:       imageData.Height,
			TextureIndex: layer1TextureIndex, // Bindless texture array index
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
		// BINDLESS: 		layer1CompositeSets, err := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
		// BINDLESS: 			DescriptorPool: compositeDescriptorPool,
		// BINDLESS: 			SetLayouts:     []vk.DescriptorSetLayout{compositeDescriptorSetLayout},
		// BINDLESS: 		})
		// BINDLESS: 		if err != nil {
		// BINDLESS: 			panic(fmt.Sprintf("Failed to allocate layer 1 composite descriptor set: %v", err))
		// BINDLESS: 		}
		// BINDLESS: 		layer1CompositeDescSet := layer1CompositeSets[0]
		// BINDLESS:
		// BINDLESS: 		// Update descriptor set to bind layer1's framebuffer texture
		// BINDLESS: 		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
		// BINDLESS: 			{
		// BINDLESS: 				DstSet:          layer1CompositeDescSet,
		// BINDLESS: 				DstBinding:      0,
		// BINDLESS: 				DstArrayElement: 0,
		// BINDLESS: 				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
		// BINDLESS: 				ImageInfo: []vk.DescriptorImageInfo{
		// BINDLESS: 					{
		// BINDLESS: 						Sampler:     layerSampler,
		// BINDLESS: 						ImageView:   layer1ImageView,
		// BINDLESS: 						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
		// BINDLESS: 					},
		// BINDLESS: 				},
		// BINDLESS: 			},
		// BINDLESS: 		})

		// BINDLESS: 		// Update layer1's VulkanPipeline with composite descriptor set
		// BINDLESS: 		layer1Pipeline := world.GetVulkanPipeline(layer1)
		// BINDLESS: 		layer1Pipeline.CompositeDescriptorSet = layer1CompositeDescSet

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
			DescriptorSet:       vk.DescriptorSet{}, // Will be created on-demand
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add TextureData component (paint canvas)
		// Assign bindless texture index for layer2
		layer2TextureIndex := assignNextTextureIndex()
		fmt.Printf("Assigned texture index %d to layer2\n", layer2TextureIndex)

		// Upload texture to global bindless descriptor set
		// Calculate which binding and array index to use for multi-binding architecture
		binding2 := layer2TextureIndex / texturesPerBinding
		arrayElement2 := layer2TextureIndex % texturesPerBinding
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          globalBindlessDescriptorSet,
				DstBinding:      binding2,
				DstArrayElement: arrayElement2,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{{
					Sampler:     canvasSampler,
					ImageView:   paintCanvas.GetView(),
					ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				}},
			},
		})

		world.AddTextureData(layer2, &ecs.TextureData{
			Image:        paintCanvas.GetImage(),
			ImageView:    paintCanvas.GetView(),
			ImageMemory:  vk.DeviceMemory{}, // Canvas manages its own memory,
			Sampler:      canvasSampler,
			Width:        paintCanvas.GetWidth(),
			Height:       paintCanvas.GetHeight(),
			TextureIndex: layer2TextureIndex, // Bindless texture array index
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

		fmt.Printf("Layer 2 configured with %d components (50%% opacity)\n", 4)
		fmt.Printf("Total entities in world: %d\n", world.EntityCount())

		// Create "Hello World!" text entity
		helloText := world.CreateEntity()
		textComponent := ecs.NewText("Hello World!", 100.0, 100.0, 32.0)
		textComponent.Color = [4]float32{1.0, 1.0, 1.0, 1.0} // White
		world.AddText(helloText, textComponent)
		fmt.Println("Created Hello World text entity")

		// === Action-based Undo System ===
		// Unified undo/redo for strokes and layer operations
		actionRecorder := NewActionRecorder()
		var currentStroke []PenState // Currently drawing stroke
		var replayRequested bool     // Trigger full replay
		// Create test UI button (toggles layer2 visibility with undo support)
		testButton := world.CreateEntity()
		buttonComponent := ecs.NewUIButton(300.0, 200.0, 150.0, 50.0, func() {
			// Toggle layer2 visibility and record it in undo history
			blendMode := world.GetBlendMode(layer2)
			if blendMode != nil {
				oldVisible := blendMode.Visible
				newVisible := !oldVisible
				blendMode.Visible = newVisible
				// Save/restore opacity to preserve user's opacity setting
				if newVisible {
					// Showing: restore saved opacity
					blendMode.Opacity = blendMode.SavedOpacity
				} else {
					// Hiding: save current opacity, then set to 0
					blendMode.SavedOpacity = blendMode.Opacity
					blendMode.Opacity = 0.0
				}
				// Record the action with the SavedOpacity value for undo/redo
				actionRecorder.RecordLayerVisibility(layer2, oldVisible, newVisible, blendMode.SavedOpacity)
			}
		})
		buttonComponent.Label = "Toggle Layer"
		buttonComponent.LabelColor = [4]float32{0.0, 0.0, 0.0, 1.0}
		buttonComponent.LabelHovered = [4]float32{0.0, 0.0, 0.0, 1.0}
		buttonComponent.LabelPressed = [4]float32{1.0, 0.0, 0.0, 1.0}
		world.AddUIButton(testButton, buttonComponent)
		fmt.Println("Created test button entity")

		// Create Undo button (action-based)
		undoButton := world.CreateEntity()
		undoButtonComponent := ecs.NewUIButton(10.0, 10.0, 80.0, 40.0, func() {
			if actionRecorder.Undo() {
				replayRequested = true
			}
		})
		undoButtonComponent.Label = "Undo"
		undoButtonComponent.LabelColor = [4]float32{1.0, 1.0, 1.0, 1.0}   // White
		undoButtonComponent.LabelHovered = [4]float32{1.0, 1.0, 1.0, 1.0} // White
		undoButtonComponent.LabelPressed = [4]float32{0.8, 0.8, 1.0, 1.0} // Light blue
		undoButtonComponent.ColorNormal = [4]float32{0.2, 0.2, 0.2, 0.9}  // Dark gray
		undoButtonComponent.ColorHovered = [4]float32{0.3, 0.3, 0.3, 0.9} // Lighter gray
		undoButtonComponent.ColorPressed = [4]float32{0.1, 0.2, 0.5, 0.9} // Blue
		world.AddUIButton(undoButton, undoButtonComponent)
		world.AddScreenSpace(undoButton, ecs.NewScreenSpace()) // Make it screen-space
		fmt.Printf("=== UNDO BUTTON CREATED: Entity=%d, Position=(%.1f,%.1f), Size=(%.1f×%.1f) ===\n",
			undoButton, undoButtonComponent.X, undoButtonComponent.Y, undoButtonComponent.Width, undoButtonComponent.Height)

		// Create Redo button (action-based)
		redoButton := world.CreateEntity()
		redoButtonComponent := ecs.NewUIButton(100.0, 10.0, 80.0, 40.0, func() {
			if actionRecorder.Redo() {
				replayRequested = true
			}
		})
		redoButtonComponent.Label = "Redo"
		redoButtonComponent.LabelColor = [4]float32{1.0, 1.0, 1.0, 1.0}   // White
		redoButtonComponent.LabelHovered = [4]float32{1.0, 1.0, 1.0, 1.0} // White
		redoButtonComponent.LabelPressed = [4]float32{0.8, 0.8, 1.0, 1.0} // Light blue
		redoButtonComponent.ColorNormal = [4]float32{0.2, 0.2, 0.2, 0.9}  // Dark gray
		redoButtonComponent.ColorHovered = [4]float32{0.3, 0.3, 0.3, 0.9} // Lighter gray
		redoButtonComponent.ColorPressed = [4]float32{0.1, 0.2, 0.5, 0.9} // Blue
		world.AddUIButton(redoButton, redoButtonComponent)
		world.AddScreenSpace(redoButton, ecs.NewScreenSpace()) // Make it screen-space
		fmt.Println("Created redo button entity")

		// Debug: Print all created buttons
		fmt.Println("\n=== ALL BUTTONS CREATED ===")
		for _, entity := range world.QueryUIButtons() {
			btn := world.GetUIButton(entity)
			screenSp := world.GetScreenSpace(entity)
			isScreenSpace := screenSp != nil && screenSp.Enabled
			fmt.Printf("  Entity %d: '%s' at (%.0f,%.0f) size %.0f×%.0f, Enabled=%v, ScreenSpace=%v\n",
				entity, btn.Label, btn.X, btn.Y, btn.Width, btn.Height, btn.Enabled, isScreenSpace)
		}
		fmt.Println("===========================\n")

		screenSpace := ecs.NewScreenSpace()
		screenSpace.Enabled = true

		world.AddScreenSpace(testButton, screenSpace)

		// === UI Layer Z-Index Constants ===
		// Reserve top 256 Z-indices for UI layers (0x7fff00 to 0x7fffff)
		const (
			UILayerButtonBase  = 0x7ffffd // Button base rectangles (below text)
			UILayerButtonText  = 0x7ffffe // Button text labels
			UILayerColorPicker = 0x7fffff // Color picker (on top of all UI)
		)

		// Helper function to create a UI layer
		createUILayer := func(name string, zindex int) (ecs.Entity, vk.Image, vk.ImageView, vk.DeviceMemory) {
			layer := world.CreateEntity()
			fmt.Printf("Created UI layer '%s' (entity %d, zindex 0x%x)\n", name, layer, zindex)

			// Add Transform component
			transform := ecs.NewTransform()
			transform.ZIndex = zindex
			world.AddTransform(layer, transform)

			// Add VulkanPipeline component
			world.AddVulkanPipeline(layer, &ecs.VulkanPipeline{
				Pipeline:            pipeline,
				PipelineLayout:      pipelineLayout,
				DescriptorPool:      descriptorPool,
				DescriptorSet:       vk.DescriptorSet{},
				DescriptorSetLayout: descriptorSetLayout,
			})

			// Create framebuffer for this UI layer
			layerImage, layerImageMemory, err := device.CreateImageWithMemory(
				swapExtent.Width,
				swapExtent.Height,
				vk.FORMAT_R8G8B8A8_UNORM,
				vk.IMAGE_TILING_OPTIMAL,
				vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
				vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
				physicalDevice,
			)
			if err != nil {
				panic(fmt.Sprintf("Failed to create UI layer '%s' image: %v", name, err))
			}

			layerImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
				Image:    layerImage,
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
				panic(fmt.Sprintf("Failed to create UI layer '%s' image view: %v", name, err))
			}

			// Add RenderTarget component
			world.AddRenderTarget(layer, &ecs.RenderTarget{
				Image:       layerImage,
				ImageView:   layerImageView,
				ImageMemory: layerImageMemory,
				Format:      vk.FORMAT_R8G8B8A8_UNORM,
				Width:       swapExtent.Width,
				Height:      swapExtent.Height,
			})

			// Assign bindless texture index
			textureIndex := assignNextTextureIndex()
			binding := textureIndex / texturesPerBinding
			arrayElement := textureIndex % texturesPerBinding

			// Upload to global bindless descriptor set
			device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
				{
					DstSet:          globalBindlessDescriptorSet,
					DstBinding:      binding,
					DstArrayElement: arrayElement,
					DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
					ImageInfo: []vk.DescriptorImageInfo{{
						Sampler:     layerSampler,
						ImageView:   layerImageView,
						ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					}},
				},
			})

			// Add TextureData component
			world.AddTextureData(layer, &ecs.TextureData{
				Image:        layerImage,
				ImageView:    layerImageView,
				ImageMemory:  layerImageMemory,
				Sampler:      layerSampler,
				Width:        swapExtent.Width,
				Height:       swapExtent.Height,
				TextureIndex: textureIndex,
			})

			// Add BlendMode component (fully opaque and visible)
			world.AddBlendMode(layer, ecs.NewBlendMode())

			fmt.Printf("UI layer '%s' configured (texture index %d)\n", name, textureIndex)
			return layer, layerImage, layerImageView, layerImageMemory
		}

		// === Create UI Layers (Multi-Layer Architecture) ===
		// Create button base layer (renders button rectangles)
		uiButtonBaseLayer, uiButtonBaseImage, uiButtonBaseView, uiButtonBaseMem := createUILayer("ButtonBase", UILayerButtonBase)
		defer device.DestroyImage(uiButtonBaseImage)
		defer device.DestroyImageView(uiButtonBaseView)
		defer device.FreeMemory(uiButtonBaseMem)

		// Create button text layer (renders button labels)
		uiButtonTextLayer, uiButtonTextImage, uiButtonTextView, uiButtonTextMem := createUILayer("ButtonText", UILayerButtonText)
		defer device.DestroyImage(uiButtonTextImage)
		defer device.DestroyImageView(uiButtonTextView)
		defer device.FreeMemory(uiButtonTextMem)

		// Create color picker layer (renders color picker UI)
		colorPickerLayer, colorPickerImage, colorPickerView, colorPickerMem := createUILayer("ColorPicker", UILayerColorPicker)
		defer device.DestroyImage(colorPickerImage)
		defer device.DestroyImageView(colorPickerView)
		defer device.FreeMemory(colorPickerMem)

		world.MakeScreenSpace(uiButtonBaseLayer, true)
		world.MakeScreenSpace(uiButtonTextLayer, true)
		world.MakeScreenSpace(colorPickerLayer, true)

		// Note: Old uiLayer variable removed - now using uiButtonBaseLayer and uiButtonTextLayer
		fmt.Printf("Created %d UI layers for multi-layer rendering\n", 3)

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
				{
					SrcAccessMask:       vk.ACCESS_NONE,
					DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
					NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               uiButtonBaseImage,
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
					Image:               uiButtonTextImage,
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
		// startTime := time.Now() // Not currently used (circular motion removed)
		running := true

		// Pen state tracking
		penX := float32(0.0)
		penY := float32(0.0)
		truePenX := float32(0.0)
		truePenY := float32(0.0)
		prevPenX := float32(0.0)
		prevPenY := float32(0.0)
		prevPrevPenX := float32(0.0) // For Bezier control point calculation
		prevPrevPenY := float32(0.0)
		prevPrevPenPressure := float32(0.0)

		// Brush settings
		useBezierInterpolation := true // Toggle between linear and Bezier
		penPressure := float32(0.0)
		prevPenPressure := float32(0.0)
		penDown := false
		prevPenDown := false // Track pen state transitions for undo/redo

		// GPU undo state and stroke bounds already declared above (before buttons)

		// Stroke tracking
		strokeActive := false
		isFirstFrameOfStroke := false // Skip interpolation on first frame to avoid (0,0) jolt
		strokeFrameCount := 0         // Count frames since stroke start for Bezier dampening
		brushRadius := float32(20.0)  // Current brush radius (scaled for 8K canvas)

		// Color picker state
		colorPicker := ColorPicker{
			Hue:        0.0,   // Start with red
			Saturation: 1.0,   // Full saturation
			Value:      1.0,   // Full brightness
			Visible:    false, // Hidden by default
		}

		// Mouse state tracking for UI
		mouseX := float32(0.0)
		mouseY := float32(0.0)
		mouseButtonDown := false
		mouseButtonDownPrev := false

		// Camera state for canvas navigation
		cameraX := float32(0.0)
		cameraY := float32(0.0)
		cameraZoom := float32(1.0)
		middleButtonDown := false
		lastMouseX := float32(0.0)
		lastMouseY := float32(0.0)

		currentLayer := -4999999
		currentFrame := 0
		//currentFrameLast := 0

		//imageIndexLast := uint32(0)
		frameCounter := uint64(0) // For periodic debug logging

		timer := time.Now().UnixMilli()

		go func() {
			//for i := 0; i < 3; i++ {
			for running {
				for event, ok := sdl.PollEvent(); ok; event, ok = sdl.PollEvent() {
					switch event.Type {

					case sdl.EVENT_QUIT:
						running = false
						sdl.Quit()

					case sdl.EVENT_PEN_AXIS:
						axis := event.PenAxis
						if axis.Axis == sdl.PEN_AXIS_PRESSURE {
							penPressure = axis.Value
							// Ensure minimum pressure for visibility (pen tablets sometimes report low values)
							/*if penPressure < 0.1 {
								penPressure = 1.0
							}*/
						}
					case sdl.EVENT_PEN_MOTION:
						motion := event.PenMotion

						truePenX = motion.X
						truePenY = motion.Y

						penX = penX + ((motion.X - penX) * max(0.02, min((cameraZoom/10), 1.0)))
						penY = penY + ((motion.Y - penY) * max(0.02, min((cameraZoom/10), 1.0)))

						penDown = motion.IsDown()

					case sdl.EVENT_MOUSE_MOTION:
						mouseMotion := event.MouseMotion
						mouseX = mouseMotion.X
						mouseY = mouseMotion.Y

						// Handle canvas panning with middle mouse button
						if middleButtonDown {
							deltaX := mouseX - lastMouseX
							deltaY := mouseY - lastMouseY
							// Convert screen delta to NDC delta (account for zoom)
							ndcDeltaX := (deltaX / float32(swapExtent.Width)) * 2.0 / cameraZoom
							ndcDeltaY := (deltaY / float32(swapExtent.Height)) * 2.0 / cameraZoom
							cameraX -= ndcDeltaX
							cameraY -= ndcDeltaY
							lastMouseX = mouseX
							lastMouseY = mouseY
						}

					case sdl.EVENT_MOUSE_BUTTON_DOWN:
						if event.MouseButton.Button == sdl.BUTTON_LEFT {
							mouseButtonDown = true

							// Check if clicking on color picker (highest priority)
							if colorPicker.Visible {
								// Calculate color picker positions
								hueWheelX := float32(swapExtent.Width) - 220.0
								hueWheelY := float32(swapExtent.Height) - 220.0
								hueWheelSize := float32(200.0)
								svBoxX := hueWheelX
								svBoxY := hueWheelY - 220.0
								svBoxSize := float32(200.0)

								// Check if clicking on SV box
								if mouseX >= svBoxX && mouseX <= svBoxX+svBoxSize &&
									mouseY >= svBoxY && mouseY <= svBoxY+svBoxSize {
									// Update saturation and value based on click position
									colorPicker.Saturation = 1.0 - (mouseX-svBoxX)/svBoxSize
									colorPicker.Value = 1.0 - (mouseY-svBoxY)/svBoxSize
									r, g, b := hsv2rgb(colorPicker.Hue, colorPicker.Saturation, colorPicker.Value)
									fmt.Printf("Color selected: RGB(%.2f, %.2f, %.2f)\n", r, g, b)
								} else if mouseX >= hueWheelX && mouseX <= hueWheelX+hueWheelSize &&
									mouseY >= hueWheelY && mouseY <= hueWheelY+hueWheelSize {
									// Check if clicking on hue wheel ring
									centerX := hueWheelX + hueWheelSize/2.0
									centerY := hueWheelY + hueWheelSize/2.0
									dx := mouseX - centerX
									dy := mouseY - centerY
									dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
									outerRadius := hueWheelSize / 2.0
									innerRadius := outerRadius * 0.6

									if dist >= innerRadius && dist <= outerRadius {
										// Calculate hue from angle
										angle := math.Atan2(float64(dy), float64(dx))
										hue := float32(angle/(2*math.Pi)) + 0.5 // Normalize to 0-1
										if hue < 0 {
											hue += 1.0
										}
										colorPicker.Hue = hue
										r, g, b := hsv2rgb(colorPicker.Hue, colorPicker.Saturation, colorPicker.Value)
										fmt.Printf("Hue selected: %.2f (RGB: %.2f, %.2f, %.2f)\n", hue, r, g, b)
									}
								}
							}
						} else if event.MouseButton.Button == sdl.BUTTON_MIDDLE {
							middleButtonDown = true
							lastMouseX = mouseX
							lastMouseY = mouseY
						}

					case sdl.EVENT_MOUSE_BUTTON_UP:
						if event.MouseButton.Button == sdl.BUTTON_LEFT {
							mouseButtonDown = false
						} else if event.MouseButton.Button == sdl.BUTTON_MIDDLE {
							middleButtonDown = false
						}

					case sdl.EVENT_MOUSE_WHEEL:
						wheel := event.MouseWheel
						// Calculate world position of mouse before zoom
						screenCenterX := float32(swapExtent.Width) / 2.0
						screenCenterY := float32(swapExtent.Height) / 2.0
						// Mouse position in NDC relative to screen center
						mouseNDCX := (mouseX - screenCenterX) / screenCenterX
						mouseNDCY := (mouseY - screenCenterY) / screenCenterY
						// World position under mouse cursor
						worldX := mouseNDCX/cameraZoom + cameraX
						worldY := mouseNDCY/cameraZoom + cameraY

						// Apply zoom
						zoomFactor := float32(1.0)
						if wheel.Y > 0 {
							zoomFactor = 1.1 // Zoom in
						} else if wheel.Y < 0 {
							zoomFactor = 0.9 // Zoom out
						}
						cameraZoom *= zoomFactor

						// Clamp zoom to reasonable limits
						if cameraZoom < 0.1 {
							cameraZoom = 0.1
						} else if cameraZoom > 10.0 {
							cameraZoom = 10.0
						}

						// Adjust camera position to keep world position under mouse
						cameraX = worldX - mouseNDCX/cameraZoom
						cameraY = worldY - mouseNDCY/cameraZoom

					case sdl.EVENT_KEY_DOWN:
						keyEvent := event.Keyboard
						ctrl := (keyEvent.Mod & sdl.KMOD_CTRL) != 0
						shift := (keyEvent.Mod & sdl.KMOD_SHIFT) != 0

						// Ctrl+Z - Undo
						if ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_Z {
							if actionRecorder.Undo() {
								replayRequested = true
								fmt.Println("Undo successful")
							}
						}

						// Ctrl+Shift+Z or Ctrl+Y - Redo
						if (ctrl && shift && keyEvent.Scancode == sdl.SCANCODE_Z) ||
							(ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_Y) {
							if actionRecorder.Redo() {
								replayRequested = true
								fmt.Println("Redo successful")
							}
						}

						// Up arrow - Increase layer2 opacity (example of OpacityAction)
						if !ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_UP {
							blendMode := world.GetBlendMode(layer2)
							if blendMode != nil {
								oldOpacity := blendMode.Opacity
								newOpacity := oldOpacity + 0.1
								if newOpacity >= 1.1 {
									newOpacity = 1.0
									break
								}
								blendMode.Opacity = newOpacity
								actionRecorder.RecordLayerOpacity(layer2, oldOpacity, newOpacity)
								fmt.Printf("Layer opacity increased to %.2f\n", newOpacity)
							}
						}

						// Down arrow - Decrease layer2 opacity
						if !ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_DOWN {
							blendMode := world.GetBlendMode(layer2)
							if blendMode != nil {
								oldOpacity := blendMode.Opacity
								newOpacity := oldOpacity - 0.1
								if newOpacity < 0.0 {
									newOpacity = 0.0
									break
								}
								blendMode.Opacity = newOpacity
								actionRecorder.RecordLayerOpacity(layer2, oldOpacity, newOpacity)
								fmt.Printf("Layer opacity decreased to %.2f\n", newOpacity)
							}
						}

						// 'C' key - Toggle color picker
						if !ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_C {
							colorPicker.Visible = !colorPicker.Visible
							if colorPicker.Visible {
								fmt.Println("Color picker shown (press C to hide)")
							} else {
								fmt.Println("Color picker hidden")
							}
						}

					case sdl.EVENT_DROP_COMPLETE:
						fmt.Printf("File loaded!")

					case sdl.EVENT_DROP_TEXT, sdl.EVENT_DROP_FILE:
						drop := event.Drop
						filePath := drop.Data
						fmt.Printf("File dropped: %s\n", filePath)

						// Check if it's an image file
						ext := strings.ToLower(filepath.Ext(filePath))
						if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".bmp" {
							fmt.Println("Loading dropped image...")
							_, err := CreateImageLayer(
								world,
								&device,
								&physicalDevice,
								&commandPool,
								&queue,
								&pipeline,
								&pipelineLayout,
								&descriptorPool,
								&descriptorSetLayout,
								globalBindlessDescriptorSet,
								&nextTextureIndex,
								maxTextures,
								layerSampler,
								swapExtent,
								filePath,
								currentLayer,
							)
							if err != nil {
								fmt.Printf("Failed to create layer: %v\n", err)
							} else {
								currentLayer++
							}
						} else {
							fmt.Printf("Unsupported file type: %s\n", ext)
						}
					}
				}
			}
			//}

			sdl.Delay(5)

		}()

		for running {
			//currentFrameLast = currentFrame
			frameCounter++

			l2tx := world.GetTransform(layer2).X
			l2ty := world.GetTransform(layer2).Y

			// Handle events

			// Update pen data text
			textComp := world.GetText(helloText)
			textComp.Content = fmt.Sprintf("Pen: (%.0f, %.0f)  Pressure: %.3f  Down: %v",
				truePenX, truePenY, penPressure, penDown)
			world.AddText(helloText, textComp)

			// Update UI button states
			// Animate layer2 in a circle

			// Wait for previous frame
			//device.WaitForFences([]vk.Fence{inFlightFences[imageIndexLast]}, true, ^uint64(1000))

			device.WaitForFences([]vk.Fence{inFlightFences[currentFrame]}, false, ^uint64(0))

			// Acquire next image
			imageIndex, err := device.AcquireNextImageKHR(swapchain, ^uint64(0), imageAvailableSems[currentFrame], vk.Fence{})
			if err != nil {
				panic(fmt.Sprintf("Acquire failed: %v", err))
			}

			device.ResetFences([]vk.Fence{inFlightFences[imageIndex]})

			// Get sorted layers
			sortedLayers := world.QueryRenderablesSorted()

			// Record command buffer
			cmd := commandBuffers[imageIndex]
			cmd.Reset(0)
			cmd.Begin(&vk.CommandBufferBeginInfo{})

			// === ACTION-BASED REPLAY: Clear and redraw all strokes ===
			if replayRequested {
				fmt.Printf("Replaying canvas: %d actions\n", actionRecorder.GetIndex())

				// Reset canvas pointers to initial state before replay
				paintCanvas = paintCanvasA
				paintCanvasSource = paintCanvasB
				fmt.Printf("Reset: paintCanvas = A (%v), paintCanvasSource = B (%v)\n",
					paintCanvas.GetImage(), paintCanvasSource.GetImage())

				// Note: Don't reset layer2 opacity/visibility here!
				// Only actions in the history should affect these properties.
				// This preserves user-made changes that aren't being undone.

				// Clear both canvases to white using vkCmdClearColorImage
				// Transition both canvases to TRANSFER_DST for clearing
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						{
							SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvasA.GetImage(),
							SubresourceRange: vk.ImageSubresourceRange{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								BaseMipLevel:   0,
								LevelCount:     1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
						},
						{
							SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvasB.GetImage(),
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

				// Clear both canvases to white
				whiteColor := vk.ClearColorValue{Float32: [4]float32{1.0, 1.0, 1.0, 1.0}}
				cmd.CmdClearColorImage(
					paintCanvasA.GetImage(),
					vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					&whiteColor,
					[]vk.ImageSubresourceRange{
						{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					},
				)
				cmd.CmdClearColorImage(
					paintCanvasB.GetImage(),
					vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					&whiteColor,
					[]vk.ImageSubresourceRange{
						{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					},
				)

				// Transition both canvases back to SHADER_READ_ONLY
				cmd.PipelineBarrier(
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
							Image:               paintCanvasA.GetImage(),
							SubresourceRange: vk.ImageSubresourceRange{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								BaseMipLevel:   0,
								LevelCount:     1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
						},
						{
							SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
							DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvasB.GetImage(),
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

				// Update brushDescriptorSet to point to paintCanvasSource at start of replay
				device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
					{
						DstSet:          brushDescriptorSet,
						DstBinding:      0,
						DstArrayElement: 0,
						DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
						ImageInfo: []vk.DescriptorImageInfo{
							{
								Sampler:     canvasSampler,
								ImageView:   paintCanvasSource.GetView(),
								ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							},
						},
					},
				})

				// Replay all actions up to undoIndex
				history := actionRecorder.GetHistory()
				fmt.Printf("Starting replay of %d actions\n", len(history))
				for actionIdx, action := range history {
					isLastStroke := (actionIdx == len(history)-1)

					switch action.Type {
					case ActionTypeStroke:
						// Replay stroke action
						if action.Stroke == nil || len(action.Stroke.States) == 0 {
							continue
						}
						stroke := action.Stroke
						fmt.Printf("  Replaying stroke %d/%d: %d stamps\n", actionIdx+1, len(history), len(stroke.States))

						// Step 1: Collect ALL stamps for the entire stroke
						var stamps []struct {
							x, y, pressure float32
						}

						for stateIdx := 0; stateIdx < len(stroke.States); stateIdx++ {
							state := stroke.States[stateIdx]

							if stateIdx == 0 {
								// First state: just render a single stamp at this position
								stamps = append(stamps, struct{ x, y, pressure float32 }{
									x:        state.X,
									y:        state.Y,
									pressure: state.Pressure,
								})
							} else {
								// Interpolate between previous and current state
								prevState := stroke.States[stateIdx-1]

								dx := state.X - prevState.X
								dy := state.Y - prevState.Y
								distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))

								brushSpacing := stroke.Radius * prevState.Pressure * 0.1
								steps := int(distance/brushSpacing) + 1
								if steps < 1 {
									steps = 1
								}

								for i := 0; i <= steps+2; i++ {
									t := float32(i) / float32(steps)
									stamps = append(stamps, struct{ x, y, pressure float32 }{
										x:        prevState.X + dx*t,
										y:        prevState.Y + dy*t,
										pressure: prevState.Pressure + (state.Pressure-prevState.Pressure)*t,
									})
								}
							}
						}

						// Step 2: Setup rendering ONCE for the entire stroke
						// Transition canvases for rendering
						cmd.PipelineBarrier(
							vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
							vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
							0,
							[]vk.ImageMemoryBarrier{
								// Source canvas: ensure SHADER_READ_ONLY
								{
									SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
									DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
									OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
									NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
									SrcQueueFamilyIndex: ^uint32(0),
									DstQueueFamilyIndex: ^uint32(0),
									Image:               paintCanvasSource.GetImage(),
									SubresourceRange: vk.ImageSubresourceRange{
										AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
										BaseMipLevel:   0,
										LevelCount:     1,
										BaseArrayLayer: 0,
										LayerCount:     1,
									},
								},
								// Destination canvas: transition to COLOR_ATTACHMENT
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

						// Begin rendering
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
									LoadOp:      vk.ATTACHMENT_LOAD_OP_LOAD,
									StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
								},
							},
						})

						// Bind pipeline and resources (ONCE)
						cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, brushPipeline)
						cmd.BindDescriptorSets(
							vk.PIPELINE_BIND_POINT_GRAPHICS,
							brushPipelineLayout,
							0,
							[]vk.DescriptorSet{brushDescriptorSet},
							nil,
						)
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
						cmd.BindVertexBuffers(0, []vk.Buffer{brushVertexBuffer}, []uint64{0})

						// Step 3: Draw all stamps (only push constants + draw in loop!)
						type BrushPushConstants struct {
							CanvasWidth  float32
							CanvasHeight float32
							BrushX       float32
							BrushY       float32
							BrushSize    float32
							BrushOpacity float32
							_            float32
							_            float32
							ColorR       float32
							ColorG       float32
							ColorB       float32
							ColorA       float32
						}

						for stampIdx, stamp := range stamps {
							// Apply minimum pressure threshold for visibility
							pressure := stamp.pressure
							/*if pressure < 0.1 {
								pressure = 1.0
							}*/

							pushConstants := BrushPushConstants{
								CanvasWidth:  float32(paintCanvas.GetWidth()),
								CanvasHeight: float32(paintCanvas.GetHeight()),
								BrushX:       stamp.x,
								BrushY:       stamp.y,
								BrushSize:    stroke.Radius * pressure,
								BrushOpacity: 1.0,
								ColorR:       stroke.Color[0],
								ColorG:       stroke.Color[1],
								ColorB:       stroke.Color[2],
								ColorA:       stroke.Color[3],
							}

							// Debug first stamp of first stroke
							if actionIdx == 0 && stampIdx == 0 {
								fmt.Printf("    First stamp: pos=(%.1f, %.1f) radius=%.1f pressure=%.3f size=%.1f color=(%.1f,%.1f,%.1f,%.1f) canvas=%dx%d\n",
									pushConstants.BrushX, pushConstants.BrushY, stroke.Radius, stamp.pressure, pushConstants.BrushSize,
									pushConstants.ColorR, pushConstants.ColorG, pushConstants.ColorB, pushConstants.ColorA,
									int(pushConstants.CanvasWidth), int(pushConstants.CanvasHeight))
							}

							cmd.CmdPushConstants(brushPipelineLayout, vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT, 0, 48, unsafe.Pointer(&pushConstants))
							cmd.Draw(6, 1, 0, 0)

							// Swap canvases after each stamp (shader needs to read accumulated result)
							if stampIdx < len(stamps)-1 {
								// End current rendering
								cmd.EndRendering()

								// Transition back
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

								// Swap
								paintCanvas, paintCanvasSource = paintCanvasSource, paintCanvas

								// Update descriptor
								device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
									{
										DstSet:          brushDescriptorSet,
										DstBinding:      0,
										DstArrayElement: 0,
										DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
										ImageInfo: []vk.DescriptorImageInfo{
											{
												Sampler:     canvasSampler,
												ImageView:   paintCanvasSource.GetView(),
												ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
											},
										},
									},
								})

								// Transition for next stamp
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

								// Begin rendering for next stamp
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
											LoadOp:      vk.ATTACHMENT_LOAD_OP_LOAD,
											StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
										},
									},
								})

								// Rebind everything
								cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, brushPipeline)
								cmd.BindDescriptorSets(vk.PIPELINE_BIND_POINT_GRAPHICS, brushPipelineLayout, 0, []vk.DescriptorSet{brushDescriptorSet}, nil)
								cmd.SetViewport(0, []vk.Viewport{{X: 0, Y: 0, Width: float32(paintCanvas.GetWidth()), Height: float32(paintCanvas.GetHeight()), MinDepth: 0.0, MaxDepth: 1.0}})
								cmd.SetScissor(0, []vk.Rect2D{{Offset: vk.Offset2D{X: 0, Y: 0}, Extent: vk.Extent2D{Width: paintCanvas.GetWidth(), Height: paintCanvas.GetHeight()}}})
								cmd.BindVertexBuffers(0, []vk.Buffer{brushVertexBuffer}, []uint64{0})
							}
						}

						// Step 4: Teardown rendering ONCE (for last stamp)
						// End rendering
						cmd.EndRendering()

						// Transition destination canvas back to SHADER_READ
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

						// Ping-pong swap (only if not the last stroke!)
						// Don't swap on the last stroke so paintCanvas points to the final result
						if !isLastStroke {
							fmt.Printf("    Swapping canvases for next stroke\n")
							paintCanvas, paintCanvasSource = paintCanvasSource, paintCanvas

							// Update descriptor set for next stroke
							device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
								{
									DstSet:          brushDescriptorSet,
									DstBinding:      0,
									DstArrayElement: 0,
									DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
									ImageInfo: []vk.DescriptorImageInfo{
										{
											Sampler:     canvasSampler,
											ImageView:   paintCanvasSource.GetView(),
											ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
										},
									},
								},
							})
						} else {
							fmt.Printf("    Skipping swap (last stroke) - paintCanvas points to result\n")
						}

					case ActionTypeLayerVisibility:
						// Replay layer visibility change
						if action.LayerVisibility != nil {
							if blend := world.GetBlendMode(action.LayerVisibility.EntityID); blend != nil {
								blend.Visible = action.LayerVisibility.NewVisible
								blend.SavedOpacity = action.LayerVisibility.SavedOpacity
								// Apply opacity based on visibility
								if blend.Visible {
									// Showing: restore saved opacity from action
									blend.Opacity = blend.SavedOpacity
								} else {
									// Hiding: set to 0
									blend.Opacity = 0.0
								}
								fmt.Printf("Replay: Set layer %d visibility to %v (opacity=%.1f, saved=%.1f)\n",
									action.LayerVisibility.EntityID, blend.Visible, blend.Opacity, blend.SavedOpacity)
							}
						}

					case ActionTypeLayerOpacity:
						// Replay layer opacity change
						if action.LayerOpacity != nil {
							if blend := world.GetBlendMode(action.LayerOpacity.EntityID); blend != nil {
								blend.Opacity = action.LayerOpacity.NewOpacity
								fmt.Printf("Replay: Set layer %d opacity to %.2f\n",
									action.LayerOpacity.EntityID, blend.Opacity)
							}
						}

					case ActionTypeLayerTransform:
						// Replay layer transform change
						if action.LayerTransform != nil {
							if transform := world.GetTransform(action.LayerTransform.EntityID); transform != nil {
								*transform = action.LayerTransform.NewTransform
								fmt.Printf("Replay: Set layer %d transform\n", action.LayerTransform.EntityID)
							}
						}

					case ActionTypeLayerCreate:
						// TODO: Implement layer creation replay
						fmt.Println("Replay: LayerCreate not yet implemented")

					case ActionTypeLayerDelete:
						// TODO: Implement layer deletion replay
						fmt.Println("Replay: LayerDelete not yet implemented")
					}
				}

				replayRequested = false
				fmt.Printf("Replay complete: rendered %d actions\n", actionRecorder.GetIndex())

				// Update layer2's texture to point to the current paintCanvas after replay
				fmt.Printf("Updating layer2 texture to point to paintCanvas (result canvas)\n")
				layer2Texture := world.GetTextureData(layer2)
				if layer2Texture != nil {
					fmt.Printf("  Old image: %v, New image: %v\n", layer2Texture.Image, paintCanvas.GetImage())
					layer2Texture.Image = paintCanvas.GetImage()
					layer2Texture.ImageView = paintCanvas.GetView()

					// Update the bindless descriptor set
					binding := layer2Texture.TextureIndex / texturesPerBinding
					arrayElement := layer2Texture.TextureIndex % texturesPerBinding
					device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
						{
							DstSet:          globalBindlessDescriptorSet,
							DstBinding:      binding,
							DstArrayElement: arrayElement,
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
				}
			}

			// === Stroke start: Detect pen down transition ===
			if penDown && !prevPenDown && penPressure > 0.0 {
				// Stroke just started
				strokeActive = true
				isFirstFrameOfStroke = true // Skip interpolation on first frame
				strokeFrameCount = 0        // Reset frame counter

				fmt.Printf("=== Stroke started (action-based undo) ===\n")
			}

			cXLast, cYLast := float32(0.0), float32(0.0)

			// === PASS 0: Render brush strokes to canvas (if pen is down) ===
			if penDown && penPressure > 0.0 {
				// Get layer2's transform to account for canvas position/scale
				layer2Transform := world.GetTransform(layer2)

				// Convert window pixel coordinates to clip space (-1 to 1)
				clipX := (penX/float32(swapExtent.Width))*2.0 - 1.0
				clipY := (penY/float32(swapExtent.Height))*2.0 - 1.0
				// Apply inverse camera transform (undo zoom and pan)
				// In shader: pos = (pos - cameraOffset) * cameraZoom
				// Inverse: pos = pos / cameraZoom + cameraOffset
				worldClipX := clipX/cameraZoom + cameraX
				worldClipY := clipY/cameraZoom + cameraY

				// Apply inverse transform to get local clip coordinates
				localClipX := (worldClipX - layer2Transform.X) / layer2Transform.ScaleX
				localClipY := (worldClipY - layer2Transform.Y) / layer2Transform.ScaleX

				// Convert from clip space (-1 to 1) to canvas pixel coordinates
				canvasX := (localClipX + 1.0) / 2.0 * float32(paintCanvas.GetWidth())
				canvasY := (localClipY + 1.0) / 2.0 * float32(paintCanvas.GetHeight())

				// === Record pen state for action-based undo ===
				if !math.IsNaN(float64(canvasX)) && !math.IsNaN(float64(canvasY)) {
					currentStroke = append(currentStroke, PenState{
						X:        canvasX,
						Y:        canvasY,
						Pressure: penPressure,
					})
					cXLast = canvasX
					cYLast = canvasY
				} else {
					currentStroke = append(currentStroke, PenState{
						X:        cXLast,
						Y:        cYLast,
						Pressure: penPressure,
					})

				}

				// PING-PONG BUFFERS: Transition source (read) and destination (write) canvases
				// Source canvas: SHADER_READ_ONLY (for sampling in shader)
				// Destination canvas: COLOR_ATTACHMENT (for writing)
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
					0,
					[]vk.ImageMemoryBarrier{
						// Source canvas: ensure it's in SHADER_READ_ONLY layout
						{
							SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               paintCanvasSource.GetImage(),
							SubresourceRange: vk.ImageSubresourceRange{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								BaseMipLevel:   0,
								LevelCount:     1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
						},
						// Destination canvas: transition to COLOR_ATTACHMENT
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

				// Bind descriptor set with source canvas texture for sampling
				cmd.BindDescriptorSets(
					vk.PIPELINE_BIND_POINT_GRAPHICS,
					brushPipelineLayout,
					0, // First set
					[]vk.DescriptorSet{brushDescriptorSet},
					nil, // No dynamic offsets
				)

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

				// Brush radius and spacing settings
				brushSpacing := brushRadius * penPressure * 0.05 // Tighter spacing for real-time strokes because me stupid.

				var steps int
				var prevCanvasX, prevCanvasY float32

				dx := float32(0.0)
				dy := float32(0.0)
				dp := float32(0.0)

				// On first frame of stroke, skip interpolation to avoid (0,0) jolt
				if isFirstFrameOfStroke {
					// Draw single stamp at current position
					steps = 0
					prevCanvasX = canvasX
					prevCanvasY = canvasY
					isFirstFrameOfStroke = false // Clear flag for next frame
				} else {
					// Interpolate brush stamps between previous and current positions
					// Initialize previous position on first stroke to avoid interpolating from (0,0)
					if prevPenX == 0.0 && prevPenY == 0.0 {
						prevPenX = penX
						prevPenY = penY
						prevPenPressure = penPressure
					}

					prevClipX := (prevPenX/float32(swapExtent.Width))*2.0 - 1.0
					prevClipY := (prevPenY/float32(swapExtent.Height))*2.0 - 1.0

					// Apply inverse camera transform to previous position
					prevWorldClipX := prevClipX/cameraZoom + cameraX
					prevWorldClipY := prevClipY/cameraZoom + cameraY
					prevLocalClipX := (prevWorldClipX - l2tx) / layer2Transform.ScaleX
					prevLocalClipY := (prevWorldClipY - l2ty) / layer2Transform.ScaleX
					prevCanvasX = (prevLocalClipX + 1.0) / 2.0 * float32(paintCanvas.GetWidth())
					prevCanvasY = (prevLocalClipY + 1.0) / 2.0 * float32(paintCanvas.GetHeight())

					// Calculate distance between previous and current positions
					dx := canvasX - prevCanvasX
					dy := canvasY - prevCanvasY
					distance := float32(math.Sqrt(float64(dx*dx + dy*dy)))

					// Calculate number of steps
					steps = int(distance/brushSpacing) + 1
					if steps < 1 {
						steps = 1
					}
				}
				// Calculate prevPrev canvas positions for Bezier control points
				prevPrevClipX := (prevPrevPenX/float32(swapExtent.Width))*2.0 - 1.0
				prevPrevClipY := (prevPrevPenY/float32(swapExtent.Height))*2.0 - 1.0
				// Apply inverse camera transform to prevPrev position
				prevPrevWorldClipX := prevPrevClipX/cameraZoom + cameraX
				prevPrevWorldClipY := prevPrevClipY/cameraZoom + cameraY

				prevPrevLocalClipX := (prevPrevWorldClipX - l2tx) / layer2Transform.ScaleX
				prevPrevLocalClipY := (prevPrevWorldClipY - l2ty) / layer2Transform.ScaleX
				prevPrevCanvasX := (prevPrevLocalClipX + 1.0) / 2.0 * float32(paintCanvas.GetWidth())
				prevPrevCanvasY := (prevPrevLocalClipY + 1.0) / 2.0 * float32(paintCanvas.GetHeight())

				// Calculate Bezier control points based on stroke direction
				controlX := prevCanvasX
				controlY := prevCanvasY
				controlP := prevPenPressure

				if prevPrevPenX != 0.0 || prevPrevPenY != 0.0 {
					// Dampen Bezier influence for first few frames to avoid bulges
					bezierFactor := float32(0.33)
					if strokeFrameCount < 4 {
						// Turn off completely for the first few frames
						bezierFactor = 0.00
					}

					controlX = prevCanvasX + (prevCanvasX-prevPrevCanvasX)*bezierFactor
					controlY = prevCanvasY + (prevCanvasY-prevPrevCanvasY)*bezierFactor
					controlP = prevPenPressure + (prevPenPressure-prevPrevPenPressure)*bezierFactor
				}

				// Increment frame counter while drawing
				if strokeActive {
					strokeFrameCount++
				}

				fuck := steps

				// Render brush stamps along interpolated path
				// IMPORTANT: Each stamp needs its own render pass with ping-pong swap
				// to properly accumulate with previous stamps in the stroke
				for i := 0; i <= fuck; i++ {
					t := float32(i) / float32(steps)

					var interpX, interpY, interpP float32 = 0.0, 0.0, 0.0
					if useBezierInterpolation && (prevPrevPenX != 0.0 || prevPrevPenY != 0.0) {
						// Quadratic Bezier interpolation
						interpX = evaluateQuadraticBezier(prevCanvasX, controlX, canvasX, t)
						interpY = evaluateQuadraticBezier(prevCanvasY, controlY, canvasY, t)
						interpP = evaluateQuadraticBezier(prevPenPressure, controlP, penPressure, t)
					} else {
						// Linear interpolation (fallback)
						if !math.IsNaN(float64(prevCanvasX + dx*t)) {
							interpX = prevCanvasX + dx*t
						}
						if !math.IsNaN(float64(prevCanvasY + dy*t)) {
							interpY = prevCanvasY + dy*t
						}
						if !math.IsNaN(float64(prevPenPressure + dp*t)) {
							interpP = prevPenPressure + dp*t
						}
					}

					// Get current brush color from color picker
					brushR, brushG, brushB := hsv2rgb(colorPicker.Hue, colorPicker.Saturation, colorPicker.Value)

					// Debug: Print color on first stamp of stroke
					if i == 0 && strokeFrameCount == 1 {
						fmt.Printf("[BRUSH] HSV(%.2f, %.2f, %.2f) -> RGB(%.2f, %.2f, %.2f)\n",
							colorPicker.Hue, colorPicker.Saturation, colorPicker.Value,
							brushR, brushG, brushB)
					}

					pushConstants := BrushPushConstants{
						CanvasWidth:  float32(paintCanvas.GetWidth()),
						CanvasHeight: float32(paintCanvas.GetHeight()),
						BrushX:       interpX,
						BrushY:       interpY,
						BrushSize:    brushRadius * interpP,
						BrushOpacity: 1.0,
						ColorR:       brushR,
						ColorG:       brushG,
						ColorB:       brushB,
						ColorA:       1.0,
					}

					cmd.CmdPushConstants(brushPipelineLayout, vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT, 0, 48, unsafe.Pointer(&pushConstants))
					cmd.Draw(6, 1, 0, 0)

					// After each stamp: End rendering, transition, swap, update descriptor
					cmd.EndRendering()

					// Transition destination canvas: COLOR_ATTACHMENT → SHADER_READ
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

					// PING-PONG SWAP: Swap after each stamp so next stamp sees accumulated result
					paintCanvas, paintCanvasSource = paintCanvasSource, paintCanvas

					// Update descriptor set to bind the new source canvas
					device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
						{
							DstSet:          brushDescriptorSet,
							DstBinding:      0,
							DstArrayElement: 0,
							DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
							ImageInfo: []vk.DescriptorImageInfo{
								{
									Sampler:     canvasSampler,
									ImageView:   paintCanvasSource.GetView(),
									ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
								},
							},
						},
					})

					// If not the last stamp, prepare for next stamp
					if i < fuck {
						// Transition canvases for next stamp
						cmd.PipelineBarrier(
							vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
							vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
							0,
							[]vk.ImageMemoryBarrier{
								// Source canvas: ensure it's in SHADER_READ_ONLY layout
								{
									SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
									DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
									OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
									NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
									SrcQueueFamilyIndex: ^uint32(0),
									DstQueueFamilyIndex: ^uint32(0),
									Image:               paintCanvasSource.GetImage(),
									SubresourceRange: vk.ImageSubresourceRange{
										AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
										BaseMipLevel:   0,
										LevelCount:     1,
										BaseArrayLayer: 0,
										LayerCount:     1,
									},
								},
								// Destination canvas: transition to COLOR_ATTACHMENT
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

						// Begin rendering for next stamp
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
									LoadOp:      vk.ATTACHMENT_LOAD_OP_LOAD,
									StoreOp:     vk.ATTACHMENT_STORE_OP_STORE,
								},
							},
						})

						// Rebind pipeline and descriptor set for next stamp
						cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, brushPipeline)
						cmd.BindDescriptorSets(
							vk.PIPELINE_BIND_POINT_GRAPHICS,
							brushPipelineLayout,
							0,
							[]vk.DescriptorSet{brushDescriptorSet},
							nil,
						)
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
						cmd.BindVertexBuffers(0, []vk.Buffer{brushVertexBuffer}, []uint64{0})
					}
				}
				// Loop handles all rendering, transitions, and swaps

				// Update previous position for next frame
				prevPrevPenX = prevPenX
				prevPrevPenY = prevPenY
				prevPrevPenPressure = prevPenPressure
				prevPenX = penX
				prevPenY = penY
				prevPenPressure = penPressure
			} else {
				// === Stroke end: Detect pen up transition ===
				if prevPenDown && !penDown && strokeActive {
					// Save completed stroke to action history
					if len(currentStroke) > 0 {
						// Add new stroke to action history
						// Get current brush color from color picker
						strokeR, strokeG, strokeB := hsv2rgb(colorPicker.Hue, colorPicker.Saturation, colorPicker.Value)
						newStroke := Stroke{
							States: currentStroke,
							Color:  [4]float32{strokeR, strokeG, strokeB, 1.0},
							Radius: brushRadius,
						}
						actionRecorder.RecordStroke(newStroke)

						// Update layer2's texture to point to the current paintCanvas
						layer2Texture := world.GetTextureData(layer2)
						if layer2Texture != nil {
							layer2Texture.Image = paintCanvas.GetImage()
							layer2Texture.ImageView = paintCanvas.GetView()

							// Update the bindless descriptor set
							binding := layer2Texture.TextureIndex / texturesPerBinding
							arrayElement := layer2Texture.TextureIndex % texturesPerBinding
							device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
								{
									DstSet:          globalBindlessDescriptorSet,
									DstBinding:      binding,
									DstArrayElement: arrayElement,
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
						}

						// Reset current stroke for next drawing
						currentStroke = nil
					}
					strokeActive = false
				}

				// Reset previous position when pen is up to prevent interpolation between strokes
				prevPenX = 0.0
				prevPenY = 0.0
				prevPenPressure = 0.0
				prevPrevPenX = 0.0
				prevPrevPenY = 0.0
				prevPrevPenPressure = 0.0
			}

			// Update prevPenDown for stroke tracking
			prevPenDown = penDown

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
				// Check if this is a UI layer - render UI elements instead of textured content
				uiCtx := &systems.UIRenderContext{
					Device:               device,
					CommandBuffer:        cmd,
					UIRectPipeline:       uiRectPipeline,
					UIRectPipelineLayout: uiRectPipelineLayout,
					BrushVertexBuffer:    brushVertexBuffer,
					SwapExtent:           swapExtent,
					TextRenderer:         textRenderer,
				}

				if entity == uiButtonBaseLayer {
					// Render button base rectangles only (no text)
					//buttonCount := len(world.QueryUIButtons())
					if frameCounter%60 == 0 { // Log every 60 frames to avoid spam
						//fmt.Printf("[UI] Rendering %d button bases (undoValid=%v)\n", buttonCount, undoValid)
					}
					systems.RenderUIButtonBases(world, uiCtx)
				} else if entity == uiButtonTextLayer {
					// Render button text labels only (no rectangles)
					systems.RenderUIButtonLabels(world, uiCtx)
				} else if entity == colorPickerLayer {
					// Render color picker if visible
					if colorPicker.Visible {
						renderColorPicker(cmd, &colorPicker, colorPickerVertexBuffer,
							hueWheelPipeline, hueWheelPipelineLayout,
							svBoxPipeline, svBoxPipelineLayout, swapExtent)
					}
				} else {
					// Normal layer - render textured quad
					renderCtx := &systems.RenderContext{
						CommandBuffer:       cmd,
						SwapExtent:          swapExtent,
						VertexBuffer:        vertexBuffer,
						IndexBuffer:         indexBuffer,
						IndexCount:          uint32(len(indices)),
						Device:              device,
						DescriptorPool:      descriptorPool,
						DescriptorSetLayout: descriptorSetLayout,
					}
					systems.RenderLayerContent(world, renderCtx, entity)
				}

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

			// Composite each layer using BINDLESS textures
			compositeCtx := &systems.CompositeContext{
				CommandBuffer:           cmd,
				CompositePipeline:       compositePipeline,
				CompositePipelineLayout: compositePipelineLayout,
				SwapExtent:              swapExtent,
				BindlessDescriptorSet:   globalBindlessDescriptorSet,
			}

			// BIND GLOBAL BINDLESS DESCRIPTOR SET ONCE!
			cmd.BindDescriptorSets(
				vk.PIPELINE_BIND_POINT_GRAPHICS,
				compositePipelineLayout,
				0,
				[]vk.DescriptorSet{globalBindlessDescriptorSet},
				nil,
			)

			// Composite each layer (push texture index per layer)
			for _, entity := range sortedLayers {
				textureData := world.GetTextureData(entity)
				blendMode := world.GetBlendMode(entity)
				transform := world.GetTransform(entity)
				if textureData != nil && blendMode != nil && transform != nil {
					// Check if entity is in screen space (ignores camera transforms)
					screenSpace := world.GetScreenSpace(entity)
					isScreenSpace := screenSpace != nil && screenSpace.Enabled

					systems.CompositeLayer(
						compositeCtx,
						textureData.TextureIndex,
						blendMode.Opacity,
						transform.X,
						transform.Y,
						transform.ScaleX,
						cameraX,
						cameraY,
						cameraZoom,
						isScreenSpace,
					)
				}
			}

			// Update undo/redo button enabled states (action-based)
			if undoBtn := world.GetUIButton(undoButton); undoBtn != nil {
				undoBtn.Enabled = actionRecorder.CanUndo() // Can undo if we have history
				// Dim the button when disabled
				if !undoBtn.Enabled {
					undoBtn.ColorNormal = [4]float32{0.15, 0.15, 0.15, 0.5}  // Very dark, semi-transparent
					undoBtn.ColorHovered = [4]float32{0.15, 0.15, 0.15, 0.5} // Same as normal when disabled
					undoBtn.LabelColor = [4]float32{0.5, 0.5, 0.5, 0.5}      // Dim text
					undoBtn.LabelHovered = [4]float32{0.5, 0.5, 0.5, 0.5}
				} else {
					undoBtn.ColorNormal = [4]float32{0.2, 0.2, 0.2, 0.9}
					undoBtn.ColorHovered = [4]float32{0.3, 0.3, 0.3, 0.9}
					undoBtn.LabelColor = [4]float32{1.0, 1.0, 1.0, 1.0}
					undoBtn.LabelHovered = [4]float32{1.0, 1.0, 1.0, 1.0}
				}
				world.AddUIButton(undoButton, undoBtn)
			}
			if redoBtn := world.GetUIButton(redoButton); redoBtn != nil {
				redoBtn.Enabled = actionRecorder.CanRedo() // Can redo if we haven't reached the end
				// Dim the button when disabled
				if !redoBtn.Enabled {
					redoBtn.ColorNormal = [4]float32{0.15, 0.15, 0.15, 0.5}  // Very dark, semi-transparent
					redoBtn.ColorHovered = [4]float32{0.15, 0.15, 0.15, 0.5} // Same as normal when disabled
					redoBtn.LabelColor = [4]float32{0.5, 0.5, 0.5, 0.5}      // Dim text
					redoBtn.LabelHovered = [4]float32{0.5, 0.5, 0.5, 0.5}
				} else {
					redoBtn.ColorNormal = [4]float32{0.2, 0.2, 0.2, 0.9}
					redoBtn.ColorHovered = [4]float32{0.3, 0.3, 0.3, 0.9}
					redoBtn.LabelColor = [4]float32{1.0, 1.0, 1.0, 1.0}
					redoBtn.LabelHovered = [4]float32{1.0, 1.0, 1.0, 1.0}
				}
				world.AddUIButton(redoButton, redoBtn)
			}

			if frameCounter%60 == 0 { // Log every 60 frames
				//fmt.Printf("[UI] Updating buttons: mouse=(%.1f,%.1f), down=%v, buttons=%d\n",
				//	mouseX, mouseY, mouseButtonDown, len(world.QueryUIButtons()))
			}

			// Use pen coordinates for UI if pen is active, otherwise use mouse
			uiX, uiY := mouseX, mouseY
			if penDown || prevPenDown {
				uiX, uiY = penX, penY
			}
			uiButtonDown := mouseButtonDown || penDown
			uiButtonDownPrev := mouseButtonDownPrev || prevPenDown
			uiButtonJustPressed := uiButtonDown && !uiButtonDownPrev

			systems.UpdateUIButtons(world, uiX, uiY, uiButtonDown, uiButtonJustPressed, cameraX, cameraY, cameraZoom, swapExtent.Width, swapExtent.Height)

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
					WaitSemaphores:   []vk.Semaphore{imageAvailableSems[currentFrame]},
					WaitDstStageMask: []vk.PipelineStageFlags{vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT},
					CommandBuffers:   []vk.CommandBuffer{commandBuffers[imageIndex]},
					SignalSemaphores: []vk.Semaphore{renderFinishedSems[currentFrame]},
				},
			}, inFlightFences[imageIndex])
			if err != nil {
				panic(fmt.Sprintf("Queue submit failed: %v", err))
			}

			// Present
			queue.PresentKHR(&vk.PresentInfoKHR{
				WaitSemaphores: []vk.Semaphore{renderFinishedSems[currentFrame]},
				Swapchains:     []vk.SwapchainKHR{swapchain},
				ImageIndices:   []uint32{imageIndex},
			})

			//imageIndexLast = imageIndex

			// Update previous button state at end of frame for next frame's "just pressed" detection
			mouseButtonDownPrev = mouseButtonDown

			currentFrame = (currentFrame + 1) % FRAMES_IN_FLIGHT

			if frameCounter%60 == 0 { // Log every 60 frames
				fmt.Printf("FPS: %f | milliseconds per 60 frames: %d\n", 60.0/float32(time.Now().UnixMilli()-timer)*1000, time.Now().UnixMilli()-timer)
				timer = time.Now().UnixMilli()
			}

			time.Sleep(1 * time.Millisecond)

		}
	}
}
