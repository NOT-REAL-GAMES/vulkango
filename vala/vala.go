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
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sync/semaphore"

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

    // Apply layer opacity to both RGB and alpha (for proper transparency)
    outColor = texColor * layer.opacity;
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

// Timeline holds animation state for the project
type Timeline struct {
	CurrentFrame int     // The frame currently being displayed/edited (0-based)
	TotalFrames  int     // Total number of frames in the animation
	FPS          float32 // Frames per second for playback
	IsPlaying    bool    // Whether the animation is currently playing
}

// GPUGarbage holds Vulkan resources marked for deferred destruction
// Resources cannot be destroyed immediately when replaced because they may
// still be referenced by in-flight GPU command buffers. This struct tracks
// when they were marked for deletion so they can be safely destroyed later.
type GPUGarbage struct {
	Image      vk.Image
	ImageView  vk.ImageView
	Memory     vk.DeviceMemory
	DeathFrame uint64 // frameCounter when this was marked for deletion
}

// Global garbage collection queue for deferred Vulkan resource destruction
var garbageQueue []GPUGarbage
var garbageMutex sync.Mutex
var frameCounter uint64 // Frame counter for garbage collection timing

// GPU memory operation mutex - prevents concurrent allocation/deallocation
// Windows drivers are strict about simultaneous GPU memory operations
var gpuMemoryMutex sync.Mutex

// PendingUpgrade holds resources ready to be swapped into a frame
// Upgrades are prepared in background goroutines, then applied on main thread
// to avoid descriptor set contention (Windows WDDM requirement)
type PendingUpgrade struct {
	FrameIndex   int
	NewImage     vk.Image
	NewView      vk.ImageView
	NewMemory    vk.DeviceMemory
	NewMipLevels uint32
	NewSize      uint32
}

// Upgrade queue - background thread sends completed upgrades here
// Main thread processes them safely (no descriptor set contention)
var upgradeQueue = make(chan PendingUpgrade, 10)

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

// === UI Layer Z-Index Constants ===
// Reserve top 256 Z-indices for UI layers (0x7fff00 to 0x7fffff)
const (
	UILayerButtonBase  = 0x7ffffc // Button base rectangles (below text)
	UILayerButtonOver  = 0x7ffffd // Button base rectangles (below text)
	UILayerText        = 0x7ffffe // Button text labels
	UILayerColorPicker = 0x7fffff // Color picker (on top of all UI)
)

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
				DescriptorBindingUpdateAfterBind:          true, // CRITICAL: Allow updating descriptors after binding (Windows requirement)
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
			Flags: vk.DESCRIPTOR_SET_LAYOUT_CREATE_UPDATE_AFTER_BIND_POOL_BIT,
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
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 0
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 1
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 2
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 3
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 4
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 5
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 6
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 7
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
			Flags: vk.DESCRIPTOR_SET_LAYOUT_CREATE_UPDATE_AFTER_BIND_POOL_BIT,
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
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 0
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 1
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 2
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 3
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 4
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 5
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 6
				vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT, // Binding 7
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
			Flags:   vk.DESCRIPTOR_POOL_CREATE_UPDATE_AFTER_BIND_BIT,
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

		// Create command pool for main render loop
		commandPool, err := device.CreateCommandPool(&vk.CommandPoolCreateInfo{
			Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
			QueueFamilyIndex: uint32(graphicsFamily),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyCommandPool(commandPool)

		// Create separate command pool for background operations (frame upgrades, streaming)
		// Command pools are NOT thread-safe, so we need separate pools for goroutines
		transferCommandPool, err := device.CreateCommandPool(&vk.CommandPoolCreateInfo{
			Flags:            vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
			QueueFamilyIndex: uint32(graphicsFamily),
		})
		if err != nil {
			panic(err)
		}
		defer device.DestroyCommandPool(transferCommandPool)

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
			//cmd.Dispatch(128, 128, 1)

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

		// Ping-pong state: tracks which canvas is source (read) and which is destination (write)
		paintCanvas := paintCanvasA       // Start with A as current (destination)
		paintCanvasSource := paintCanvasB // B is source for first frame

		// === Frame-by-Frame Animation Storage ===
		// Each frame gets its own texture to enable traditional animation workflow
		type FrameTexture struct {
			Image        vk.Image
			ImageView    vk.ImageView
			Memory       vk.DeviceMemory
			TextureIndex uint32
			MipLevels    uint32 // Number of mip levels in this texture

			// Low-res proxy system for fast scrubbing (Windows optimization)
			IsLowRes     bool   // True if this is a 32×32 proxy texture
			ActualWidth  uint32 // Actual texture width (32 or 2048)
			ActualHeight uint32 // Actual texture height (32 or 2048)

			// Progressive streaming state (RAGE-style)
			CurrentMip   int       // Currently loaded mip (4=lowest/fastest, 0=full quality)
			StreamCancel chan bool // Send signal here to cancel ongoing streaming

			// Per-frame synchronization for fast frame switching
			LastFence vk.Fence   // Fence signaled when this frame's last GPU work completes
			Mutex     sync.Mutex // Protects frame resources from concurrent access during upgrades

			// Action replay tracking
			NeedsReplay bool // True if this frame reached full resolution and needs action replay
		}
		frameTextures := make(map[int]*FrameTexture)
		var frameTexturesMutex sync.RWMutex                 // Protects frameTextures map (maps are not thread-safe!)
		upgradeCommandPoolMutex := semaphore.NewWeighted(1) // Only one upgrade at a time (command pool not thread-safe)

		_ = 0 // streamingFrameNum will be used for progressive streaming (TODO)
		fmt.Println("Initializing frame storage system...")

		// === GPU Snapshot System for Fast Undo ===
		// Store canvas snapshots every N strokes to avoid replaying from the beginning
		type CanvasSnapshot struct {
			Image       vk.Image
			ImageView   vk.ImageView
			Memory      vk.DeviceMemory
			ActionIndex int  // Which action index this snapshot corresponds to
			IsValid     bool // Whether this snapshot contains valid data
		}

		const maxSnapshots = 10    // Keep up to 10 snapshots
		const snapshotInterval = 5 // Take a snapshot every 5 strokes
		snapshots := make([]CanvasSnapshot, maxSnapshots)
		nextSnapshotSlot := 0 // Round-robin slot allocation

		// Pre-allocate snapshot images
		fmt.Println("Allocating snapshot images for fast undo...")
		for i := 0; i < maxSnapshots; i++ {
			snapImage, snapMem, err := device.CreateImageWithMemory(
				paintCanvasA.GetWidth(),
				paintCanvasA.GetHeight(),
				vk.FORMAT_R8G8B8A8_UNORM,
				vk.IMAGE_TILING_OPTIMAL,
				vk.IMAGE_USAGE_TRANSFER_SRC_BIT|vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
				vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
				physicalDevice,
			)
			if err != nil {
				panic(fmt.Sprintf("Failed to create snapshot image %d: %v", i, err))
			}
			defer device.DestroyImage(snapImage)
			defer device.FreeMemory(snapMem)

			snapView, err := device.CreateImageViewForTexture(snapImage, vk.FORMAT_R8G8B8A8_UNORM)
			if err != nil {
				panic(fmt.Sprintf("Failed to create snapshot image view %d: %v", i, err))
			}
			defer device.DestroyImageView(snapView)

			snapshots[i] = CanvasSnapshot{
				Image:       snapImage,
				ImageView:   snapView,
				Memory:      snapMem,
				ActionIndex: -1,
				IsValid:     false,
			}
		}
		fmt.Printf("Allocated %d snapshot images (%dx%d each)\n", maxSnapshots, paintCanvasA.GetWidth(), paintCanvasA.GetHeight())

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

		// Frame 0 will be initialized by loadFrame(0) below with proper RAGE/mipmap support
		// No need for old pre-initialization

		fmt.Println("\nCreating texture...")

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
		maxTextChars := 1048576
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

		// Create per-frame, per-usage staging buffers (CPU-writable, for uploading data)
		// 2D array: [bufferSetIndex][frameIndex]
		// Buffer set 0 = button labels, Buffer set 1 = text entities
		// This prevents conflicts when multiple render passes use text in same frame
		numFrames := len(swapImages)
		numBufferSets := 2 // 0=button labels, 1=text entities
		textStagingVertexBuffers := make([][]vk.Buffer, numBufferSets)
		textStagingVertexMemories := make([][]vk.DeviceMemory, numBufferSets)
		textStagingIndexBuffers := make([][]vk.Buffer, numBufferSets)
		textStagingIndexMemories := make([][]vk.DeviceMemory, numBufferSets)

		for set := 0; set < numBufferSets; set++ {
			textStagingVertexBuffers[set] = make([]vk.Buffer, numFrames)
			textStagingVertexMemories[set] = make([]vk.DeviceMemory, numFrames)
			textStagingIndexBuffers[set] = make([]vk.Buffer, numFrames)
			textStagingIndexMemories[set] = make([]vk.DeviceMemory, numFrames)

			for i := 0; i < numFrames; i++ {
				vertexBuf, vertexMem, err := device.CreateBufferWithMemory(
					textVertexBufferSize,
					vk.BUFFER_USAGE_TRANSFER_SRC_BIT|vk.BUFFER_USAGE_VERTEX_BUFFER_BIT,
					vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
					physicalDevice,
				)
				if err != nil {
					panic(fmt.Sprintf("Failed to create vertex staging buffer set %d, frame %d: %v", set, i, err))
				}
				textStagingVertexBuffers[set][i] = vertexBuf
				textStagingVertexMemories[set][i] = vertexMem
				defer device.DestroyBuffer(vertexBuf)
				defer device.FreeMemory(vertexMem)

				indexBuf, indexMem, err := device.CreateBufferWithMemory(
					textIndexBufferSize,
					vk.BUFFER_USAGE_TRANSFER_SRC_BIT|vk.BUFFER_USAGE_INDEX_BUFFER_BIT,
					vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT|vk.MEMORY_PROPERTY_HOST_COHERENT_BIT,
					physicalDevice,
				)
				if err != nil {
					panic(fmt.Sprintf("Failed to create index staging buffer set %d, frame %d: %v", set, i, err))
				}
				textStagingIndexBuffers[set][i] = indexBuf
				textStagingIndexMemories[set][i] = indexMem
				defer device.DestroyBuffer(indexBuf)
				defer device.FreeMemory(indexMem)
			}
		}
		fmt.Printf("Created %d buffer sets x %d frames = %d total text staging buffer sets\n", numBufferSets, numFrames, numBufferSets*numFrames)

		// Create TextRenderer for UI button labels
		textRenderer := &systems.TextRenderer{
			Pipeline:              textPipeline,
			PipelineLayout:        textPipelineLayout,
			DescriptorSet:         textDescriptorSet,
			Atlas:                 sdfAtlas,
			VertexBuffer:          textVertexBuffer,
			VertexMemory:          textVertexMemory,
			IndexBuffer:           textIndexBuffer,
			IndexMemory:           textIndexMemory,
			StagingVertexBuffers:  textStagingVertexBuffers,
			StagingVertexMemories: textStagingVertexMemories,
			StagingIndexBuffers:   textStagingIndexBuffers,
			StagingIndexMemories:  textStagingIndexMemories,
			MaxChars:              maxTextChars,
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

		// === Animation Timeline ===
		timeline := &Timeline{
			CurrentFrame: 0,     // Start at frame 0
			TotalFrames:  120,   // Default to 120 frames (5 seconds at 24fps)
			FPS:          24.0,  // Traditional animation framerate
			IsPlaying:    false, // Not playing by default
		}
		fmt.Printf("Timeline: %d frames at %.0f FPS (current frame: %d)\n",
			timeline.TotalFrames, timeline.FPS, timeline.CurrentFrame)

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

		// ===== Create Background Layer =====
		fmt.Println("\n=== Creating Background Layer ===")
		layerBg := world.CreateEntity()
		fmt.Printf("Created background layer entity: %d\n", layerBg)

		// Add Transform component (ZIndex 0 - between bear and paint layer)
		transformBg := ecs.NewTransform()
		transformBg.ZIndex = 0
		world.AddTransform(layerBg, transformBg)

		// Create framebuffer for background layer
		layerBgImage, layerBgImageMemory, err := device.CreateImageWithMemory(
			swapExtent.Width,
			swapExtent.Height,
			vk.FORMAT_R8G8B8A8_UNORM,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT|vk.IMAGE_USAGE_TRANSFER_DST_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(fmt.Sprintf("Failed to create background layer framebuffer image: %v", err))
		}
		defer device.DestroyImage(layerBgImage)
		defer device.FreeMemory(layerBgImageMemory)

		layerBgImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
			Image:    layerBgImage,
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
			panic(fmt.Sprintf("Failed to create background layer image view: %v", err))
		}
		defer device.DestroyImageView(layerBgImageView)

		// Add RenderTarget component for background layer
		world.AddRenderTarget(layerBg, &ecs.RenderTarget{
			Image:       layerBgImage,
			ImageView:   layerBgImageView,
			ImageMemory: layerBgImageMemory,
			Format:      vk.FORMAT_R8G8B8A8_UNORM,
			Width:       swapExtent.Width,
			Height:      swapExtent.Height,
		})

		// Assign bindless texture index for background layer
		layerBgTextureIndex := assignNextTextureIndex()
		fmt.Printf("Assigned texture index %d to background layer\n", layerBgTextureIndex)

		// Upload background layer framebuffer to global bindless descriptor set
		bindingBg := layerBgTextureIndex / texturesPerBinding
		arrayElementBg := layerBgTextureIndex % texturesPerBinding
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          globalBindlessDescriptorSet,
				DstBinding:      bindingBg,
				DstArrayElement: arrayElementBg,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{{
					Sampler:     layerSampler,
					ImageView:   layerBgImageView,
					ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				}},
			},
		})

		world.AddTextureData(layerBg, &ecs.TextureData{
			Image:        layerBgImage,
			ImageView:    layerBgImageView,
			ImageMemory:  layerBgImageMemory,
			Sampler:      layerSampler,
			Width:        swapExtent.Width,
			Height:       swapExtent.Height,
			TextureIndex: layerBgTextureIndex,
		})

		// Add VulkanPipeline component (required for QueryRenderables)
		world.AddVulkanPipeline(layerBg, &ecs.VulkanPipeline{
			Pipeline:            pipeline,
			PipelineLayout:      pipelineLayout,
			DescriptorPool:      descriptorPool,
			DescriptorSet:       vk.DescriptorSet{},
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add BlendMode component (visible, opaque)
		world.AddBlendMode(layerBg, ecs.NewBlendMode())

		fmt.Printf("Background layer configured\n")

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
		blendMode2.Opacity = 0.5      // 50% transparent
		blendMode2.SavedOpacity = 0.5 // IMPORTANT: Set SavedOpacity too, or it defaults to 1.0!
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

		// ===== TEST: Create a layer group and put bear inside it =====
		fmt.Println("\n=== Creating Test Layer Group ===")
		testGroup := world.CreateEntity()
		fmt.Printf("Created test group entity: %d\n", testGroup)

		// Add LayerGroup component
		world.AddLayerGroup(testGroup, ecs.NewLayerGroup())

		// Add Transform (ZIndex between background and bear)
		groupTransform := ecs.NewTransform()
		groupTransform.ZIndex = -4000000 // Between background (0) and bear (-5000000)
		world.AddTransform(testGroup, groupTransform)

		// Create framebuffer for the group (where children will render)
		groupImage, groupImageMemory, err := device.CreateImageWithMemory(
			swapExtent.Width,
			swapExtent.Height,
			vk.FORMAT_R8G8B8A8_UNORM,
			vk.IMAGE_TILING_OPTIMAL,
			vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT|vk.IMAGE_USAGE_SAMPLED_BIT|vk.IMAGE_USAGE_TRANSFER_DST_BIT,
			vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			physicalDevice,
		)
		if err != nil {
			panic(fmt.Sprintf("Failed to create group framebuffer image: %v", err))
		}
		defer device.DestroyImage(groupImage)
		defer device.FreeMemory(groupImageMemory)

		groupImageView, err := device.CreateImageView(&vk.ImageViewCreateInfo{
			Image:    groupImage,
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
			panic(fmt.Sprintf("Failed to create group image view: %v", err))
		}
		defer device.DestroyImageView(groupImageView)

		// Add RenderTarget for the group
		world.AddRenderTarget(testGroup, &ecs.RenderTarget{
			Image:       groupImage,
			ImageView:   groupImageView,
			ImageMemory: groupImageMemory,
			Format:      vk.FORMAT_R8G8B8A8_UNORM,
			Width:       swapExtent.Width,
			Height:      swapExtent.Height,
		})

		// Assign bindless texture index for the group
		groupTextureIndex := assignNextTextureIndex()
		fmt.Printf("Assigned texture index %d to test group\n", groupTextureIndex)

		// Upload group framebuffer to global bindless descriptor set
		bindingGroup := groupTextureIndex / texturesPerBinding
		arrayElementGroup := groupTextureIndex % texturesPerBinding
		device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
			{
				DstSet:          globalBindlessDescriptorSet,
				DstBinding:      bindingGroup,
				DstArrayElement: arrayElementGroup,
				DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				ImageInfo: []vk.DescriptorImageInfo{{
					Sampler:     layerSampler,
					ImageView:   groupImageView,
					ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				}},
			},
		})

		world.AddTextureData(testGroup, &ecs.TextureData{
			Image:        groupImage,
			ImageView:    groupImageView,
			ImageMemory:  groupImageMemory,
			Sampler:      layerSampler,
			Width:        swapExtent.Width,
			Height:       swapExtent.Height,
			TextureIndex: groupTextureIndex,
		})

		// Add VulkanPipeline (required for QueryRenderables)
		world.AddVulkanPipeline(testGroup, &ecs.VulkanPipeline{
			Pipeline:            pipeline,
			PipelineLayout:      pipelineLayout,
			DescriptorPool:      descriptorPool,
			DescriptorSet:       vk.DescriptorSet{},
			DescriptorSetLayout: descriptorSetLayout,
		})

		// Add BlendMode (visible, opaque)
		groupBlend := ecs.NewBlendMode()
		groupBlend.Opacity = 0.5 // 50% transparent
		world.AddBlendMode(testGroup, groupBlend)

		// Add the bear (layer1) as a child of this group
		world.AddChildToGroup(testGroup, layer1)
		fmt.Printf("Added bear (layer1=%d) as child of test group\n", layer1)

		// Verify the relationship
		children := world.GetChildren(testGroup)
		parent := world.GetParent(layer1)
		fmt.Printf("Test group children: %v\n", children.ChildEntities)
		fmt.Printf("Bear's parent: %d (should be %d)\n", parent.ParentEntity, testGroup)

		// Create "Hello World!" text entity (pen pressure indicator)
		helloText := world.CreateEntity()
		textComponent := ecs.NewText("Hello World!", 100.0, 100.0, 32.0)
		textComponent.Color = [4]float32{1.0, 1.0, 1.0, 1.0} // White
		world.AddText(helloText, textComponent)
		fmt.Println("Created Hello World text entity (screen-space)")

		textTransform := ecs.NewTransform()
		textTransform.ZIndex = UILayerText // lmao
		world.AddTransform(helloText, textTransform)
		world.SetScreenSpace(helloText, false) // Make screen-space so it stays fixed

		// Create frame counter text entity (top-right corner)
		frameCounterText := world.CreateEntity()
		frameText := fmt.Sprintf("Frame: %d/%d", timeline.CurrentFrame, timeline.TotalFrames-1)
		frameComponent := ecs.NewText(frameText, float32(swapExtent.Width)-200.0, 20.0, 24.0)
		frameComponent.Color = [4]float32{0.2, 1.0, 0.2, 1.0} // Green
		world.AddText(frameCounterText, frameComponent)
		world.SetScreenSpace(frameCounterText, true) // Make screen-space so it stays fixed
		fmt.Println("Created frame counter text entity (screen-space)")

		//world.AddTransform(frameCounterText, textTransform)

		// === Helper Functions for Mipmap Generation ===
		// Calculate number of mip levels for a texture (will be used for upgradeToFullResolution)
		calculateMipLevels := func(width, height uint32) uint32 {
			levels := uint32(1)
			for width > 1 || height > 1 {
				levels++
				if width > 1 {
					width /= 2
				}
				if height > 1 {
					height /= 2
				}
			}
			return levels
		}
		_ = calculateMipLevels // Will be used in upgradeToFullResolution

		// generateMipmaps - Generate all mip levels for an image using vkCmdBlitImage
		// This creates the progressive quality levels for RAGE-style streaming
		generateMipmaps := func(image vk.Image, width, height, mipLevels uint32, cmd vk.CommandBuffer) {
			// Transition mip 0 to TRANSFER_SRC (we'll blit FROM it)
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Generate each mip level by blitting from previous level
			mipWidth := width
			mipHeight := height

			for mip := uint32(1); mip < mipLevels; mip++ {
				// Calculate dimensions for this mip
				nextWidth := mipWidth / 2
				nextHeight := mipHeight / 2
				if nextWidth == 0 {
					nextWidth = 1
				}
				if nextHeight == 0 {
					nextHeight = 1
				}

				// Transition this mip to TRANSFER_DST for writing
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					0,
					[]vk.ImageMemoryBarrier{{
						SrcAccessMask:       0,
						DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
						NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               image,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   mip,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				// Blit from previous mip (mip-1) to current mip
				cmd.CmdBlitImage(
					image, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					image, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					[]vk.ImageBlit{{
						SrcSubresource: vk.ImageSubresourceLayers{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							MipLevel:       mip - 1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
						SrcOffsets: [2]vk.Offset3D{
							{X: 0, Y: 0, Z: 0},
							{X: int32(mipWidth), Y: int32(mipHeight), Z: 1},
						},
						DstSubresource: vk.ImageSubresourceLayers{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							MipLevel:       mip,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
						DstOffsets: [2]vk.Offset3D{
							{X: 0, Y: 0, Z: 0},
							{X: int32(nextWidth), Y: int32(nextHeight), Z: 1},
						},
					}},
					vk.FILTER_LINEAR, // Use linear filtering for smooth downsampling
				)

				// Transition this mip to TRANSFER_SRC for next iteration
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					0,
					[]vk.ImageMemoryBarrier{{
						SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
						DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               image,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   mip,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				mipWidth = nextWidth
				mipHeight = nextHeight
			}

			// Transition ALL mips to SHADER_READ_ONLY_OPTIMAL for sampling
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     mipLevels, // All mips!
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			fmt.Printf("[MIPMAPS] Generated %d mip levels (%dx%d → 1x1)\n", mipLevels, width, height)
		}

		// Declare replayRequested early so frame functions can access it
		var replayRequested bool // Trigger full replay

		// === Frame Switching Functions ===
		// saveCurrentFrame copies paintCanvas content to current frame's storage
		saveCurrentFrame := func(frameNum int) {
			currentFrame := frameNum
			frameTexture := frameTextures[currentFrame]
			if frameTexture == nil {
				fmt.Printf("Warning: No texture for frame %d to save to\n", currentFrame)
				return
			}

			// Debug: Log frame save state for diagnostics
			fmt.Printf("[SAVE] Frame %d: Starting save operation\n", currentFrame)
			fmt.Printf("[SAVE]   - replayRequested=%v\n", replayRequested)
			fmt.Printf("[SAVE]   - paintCanvas: %v (size: %dx%d)\n", paintCanvas.GetImage(), paintCanvas.GetWidth(), paintCanvas.GetHeight())

			// Lock to safely read frame resource handles
			frameTexture.Mutex.Lock()
			frameImage := frameTexture.Image
			frameActualWidth := frameTexture.ActualWidth
			frameActualHeight := frameTexture.ActualHeight
			frameIsLowRes := frameTexture.IsLowRes
			frameNeedsReplay := frameTexture.NeedsReplay
			frameTexture.Mutex.Unlock()

			fmt.Printf("[SAVE]   - frameTexture: %v (size: %dx%d, IsLowRes=%v, NeedsReplay=%v)\n",
				frameImage, frameActualWidth, frameActualHeight, frameIsLowRes, frameNeedsReplay)

			cmdBufs, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
				CommandPool:        commandPool,
				Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
				CommandBufferCount: 1,
			})
			if err != nil {
				panic(err)
			}
			cmd := cmdBufs[0]
			cmd.Begin(&vk.CommandBufferBeginInfo{
				Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
			})

			// Transition frameTexture to TRANSFER_DST
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               frameImage,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Transition paintCanvas to TRANSFER_SRC
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
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
				}},
			)

			// Copy paintCanvas to frameTexture using blit
			cmd.CmdBlitImage(
				paintCanvas.GetImage(), vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				frameImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				[]vk.ImageBlit{{
					SrcSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       0,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					SrcOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(paintCanvas.GetWidth()), Y: int32(paintCanvas.GetHeight()), Z: 1},
					},
					DstSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       0,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					DstOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(frameActualWidth), Y: int32(frameActualHeight), Z: 1},
					},
				}},
				vk.FILTER_LINEAR,
			)

			// Transition paintCanvas back to SHADER_READ_ONLY (do this BEFORE generating mipmaps)

			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
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
				}},
			)

			// Only generate mipmaps if this is a full-resolution frame (not a proxy)
			if !frameTexture.IsLowRes && frameTexture.MipLevels > 1 {
				// Generate all mip levels for frame texture (RAGE-style progressive streaming)
				// This will transition frameTexture to SHADER_READ_ONLY_OPTIMAL
				generateMipmaps(frameTexture.Image, frameTexture.ActualWidth, frameTexture.ActualHeight, frameTexture.MipLevels, cmd)
			} else {
				// Low-res proxy or single-mip image - just transition to SHADER_READ_ONLY
				cmd.PipelineBarrier(
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
						Image:               frameTexture.Image,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     frameTexture.MipLevels,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)
			}

			cmd.End()

			// Create fence for per-frame synchronization (for fast frame switching)
			fmt.Printf("[SAVE] Creating fence for frame %d\n", currentFrame)
			fence, err := device.CreateFence(&vk.FenceCreateInfo{})
			if err != nil {
				panic(fmt.Sprintf("Failed to create fence: %v", err))
			}

			fmt.Printf("[SAVE] Submitting command buffer for frame %d\n", currentFrame)
			err = queue.Submit([]vk.SubmitInfo{
				{CommandBuffers: []vk.CommandBuffer{cmd}},
			}, fence)
			if err != nil {
				panic(fmt.Sprintf("Failed to submit: %v", err))
			}

			// CRITICAL: Must wait for save to complete before continuing
			// Otherwise main render loop will use paintCanvas while GPU is still reading it
			fmt.Printf("[SAVE] Waiting for fence for frame %d...\n", currentFrame)
			time.Sleep(5 * time.Millisecond)
			err = device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))
			if err != nil {
				panic(fmt.Sprintf("Failed to wait for fence: %v", err))
			}
			fmt.Printf("[SAVE] Fence signaled for frame %d\n", currentFrame)

			// Destroy old fence if it exists, then store new (signaled) fence
			// Frame switching will wait for this fence (returns immediately since already signaled)
			// Lock to update LastFence
			frameTexture.Mutex.Lock()
			if frameTexture.LastFence != (vk.Fence{}) {
				device.DestroyFence(frameTexture.LastFence)
			}
			frameTexture.LastFence = fence

			frameTexture.Mutex.Unlock()
			device.FreeCommandBuffers(commandPool, cmdBufs)
			fmt.Printf("[SAVE] Frame %d saved (fence stored and signaled)\n", currentFrame)

			fmt.Printf("Saved frame %d\n", currentFrame)
		}

		// streamFrameProgressive - RAGE-style progressive mipmap streaming
		// Loads lowest mip first (instant), then streams higher quality over time
		// Can be cancelled if user switches to different frame
		streamFrameProgressive := func(frameNum int) {
			frameTexture := frameTextures[frameNum]
			if frameTexture == nil || frameTexture.CurrentMip == 0 {
				return // Nothing to stream (already at full quality)
			}

			fmt.Printf("[STREAM] Starting progressive streaming for frame %d (mip %d → 0)\n", frameNum, frameTexture.CurrentMip)

			// Stream from current mip down to 0 (full quality)
			// Each mip takes ~16ms (60fps pacing) to "load"
			for mip := frameTexture.CurrentMip - 1; mip >= 0; mip-- {
				select {
				case <-frameTexture.StreamCancel:
					fmt.Printf("[STREAM] Cancelled streaming for frame %d at mip %d\n", frameNum, mip)
					return // User switched away, abort streaming

				case <-time.After(16 * time.Millisecond): // ~60fps pacing
					// Update to next higher quality mip
					frameTexture.CurrentMip = mip
					fmt.Printf("[STREAM] Frame %d → mip %d loaded (quality improving...)\n", frameNum, mip)
				}
			}

			fmt.Printf("[STREAM] Frame %d fully loaded at mip 0 (full quality)\n", frameNum)
		}

		// loadFrame copies a frame's storage to paintCanvas (creates frame if it doesn't exist)
		loadFrame := func(frameNum int) {
			frameTexture := frameTextures[frameNum]
			isNewFrame := frameTexture == nil // Track if we're creating a new frame

			// Lazy frame creation - create frame if it doesn't exist yet
			if isNewFrame {
				fmt.Printf("Creating new frame %d (low-res proxy 32×32)...\n", frameNum)

				// Create TINY 32×32 proxy texture for instant frame switching (Windows optimization!)
				// Will be upgraded to 2048×2048 when scrubbing stops
				const proxySize = 32
				width := uint32(proxySize)
				height := uint32(proxySize)

				// No mipmaps needed for tiny proxy
				newImage, newMemory, err := device.CreateImageWithMemory(
					width,
					height,
					vk.FORMAT_R8G8B8A8_UNORM,
					vk.IMAGE_TILING_OPTIMAL,
					vk.IMAGE_USAGE_TRANSFER_SRC_BIT|vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
					vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
					physicalDevice,
				)
				if err != nil {
					panic(err)
				}

				fmt.Printf("Created low-res frame %d (%dx%d) - will upgrade when scrubbing stops\n", frameNum, width, height)

				newView, err := device.CreateImageViewForTexture(newImage, vk.FORMAT_R8G8B8A8_UNORM)
				if err != nil {
					panic(err)
				}

				newTextureIndex := assignNextTextureIndex()

				// Clear new frame to transparent
				cmdBufs, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
					CommandPool:        commandPool,
					Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
					CommandBufferCount: 1,
				})
				if err != nil {
					panic(err)
				}
				cmd := cmdBufs[0]
				cmd.Begin(&vk.CommandBufferBeginInfo{
					Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
				})

				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					0,
					[]vk.ImageMemoryBarrier{{
						SrcAccessMask:       0,
						DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
						NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               newImage,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				cmd.CmdClearColorImage(
					newImage,
					vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					&vk.ClearColorValue{Float32: [4]float32{0, 0, 0, 0}},
					[]vk.ImageSubresourceRange{{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					}},
				)

				// Skip mipmap generation for blank frames - huge speed win on Windows!
				// Mipmaps will be generated when frame is saved (has content)
				// Just transition to SHADER_READ_ONLY for now
				cmd.PipelineBarrier(
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
						Image:               newImage,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1, // Only mip 0 is initialized
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				cmd.End()

				// Create fence for synchronization
				fence, err := device.CreateFence(&vk.FenceCreateInfo{})
				if err != nil {
					panic(fmt.Sprintf("Failed to create fence: %v", err))
				}

				queue.Submit([]vk.SubmitInfo{
					{CommandBuffers: []vk.CommandBuffer{cmd}},
				}, fence)

				// Wait for fence, then clean up
				device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))
				device.DestroyFence(fence)
				device.FreeCommandBuffers(commandPool, cmdBufs)

				frameTextures[frameNum] = &FrameTexture{
					Image:        newImage,
					ImageView:    newView,
					Memory:       newMemory,
					TextureIndex: newTextureIndex,
					MipLevels:    1,    // No mipmaps for 32×32 proxy
					IsLowRes:     true, // Mark as low-res proxy
					ActualWidth:  32,   // Track actual size for blits
					ActualHeight: 32,
					CurrentMip:   0, // No mips, so mip 0 is all we have
					StreamCancel: make(chan bool, 1),
				}

				// Update bindless descriptor set to point to this new frame texture
				const texturesPerBinding = 16384
				binding := newTextureIndex / texturesPerBinding
				arrayElement := newTextureIndex % texturesPerBinding
				device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
					{
						DstSet:          globalBindlessDescriptorSet,
						DstBinding:      binding,
						DstArrayElement: arrayElement,
						DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
						ImageInfo: []vk.DescriptorImageInfo{{
							Sampler:     textureSampler,
							ImageView:   newView,
							ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
						}},
					},
				})
				frameTexture = frameTextures[frameNum]
				fmt.Printf("[PROXY] Frame %d created as 32×32 proxy - will upgrade when scrubbing stops\n", frameNum)

				// Don't stream proxy textures - they're already instant!
				// They'll be upgraded to full resolution when scrubbing stops
			} else {
				// For existing full-res frames, reset to mip 4 for fast RAGE loading
				// For proxy frames, CurrentMip is already correct (0)
				if !frameTexture.IsLowRes && frameTexture.MipLevels > 4 {
					frameTexture.CurrentMip = 4
				}
			}

			// Lock frame resources to prevent concurrent upgrade
			// Lock to safely read frame resource handles
			frameTexture.Mutex.Lock()
			frameImage := frameTexture.Image
			frameIsLowRes := frameTexture.IsLowRes
			frameActualWidth := frameTexture.ActualWidth
			frameActualHeight := frameTexture.ActualHeight
			frameMipLevels := frameTexture.MipLevels
			frameCurrentMip := frameTexture.CurrentMip
			frameTexture.Mutex.Unlock()
			// Copy frameTexture to paintCanvas
			cmdBufs, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
				CommandPool:        commandPool,
				Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
				CommandBufferCount: 1,
			})
			if err != nil {
				panic(err)
			}
			cmd := cmdBufs[0]
			cmd.Begin(&vk.CommandBufferBeginInfo{
				Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
			})

			// Transition frameTexture to TRANSFER_SRC
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               frameTexture.Image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Transition paintCanvas to TRANSFER_DST
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       0,
					DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
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
				}},
			)

			// Copy frameTexture to paintCanvas
			// RAGE: Use current mip level for faster loading (starts at mip 4 = 128×128, then streams to mip 0 = 2048×2048)
			currentMip := frameCurrentMip
			if currentMip > int(frameMipLevels)-1 {
				currentMip = int(frameMipLevels) - 1 // Clamp to available mips
			}

			// Calculate source dimensions based on mip level
			var srcWidth, srcHeight uint32
			if frameIsLowRes {
				// Low-res proxy: no mipmaps, just use actual dimensions
				srcWidth = frameActualWidth
				srcHeight = frameActualHeight
			} else {
				// Full-res with mipmaps: calculate mip dimensions
				srcWidth = frameActualWidth >> uint32(currentMip)
				srcHeight = frameActualHeight >> uint32(currentMip)
				if srcWidth == 0 {
					srcWidth = 1
				}
				if srcHeight == 0 {
					srcHeight = 1
				}
			}

			fmt.Printf("[BLIT] Copying frame %d from mip %d (%dx%d) to paintCanvas (2048×2048)\n", frameNum, currentMip, srcWidth, srcHeight)

			cmd.CmdBlitImage(
				frameTexture.Image, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				paintCanvas.GetImage(), vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				[]vk.ImageBlit{{
					SrcSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       uint32(currentMip), // RAGE: Use current mip for progressive loading
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					SrcOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(srcWidth), Y: int32(srcHeight), Z: 1}, // Source size based on mip
					},
					DstSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       0,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					DstOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(paintCanvas.GetWidth()), Y: int32(paintCanvas.GetHeight()), Z: 1},
					},
				}},
				vk.FILTER_LINEAR,
			)

			// Transition frameTexture back to SHADER_READ_ONLY
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               frameImage,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Transition paintCanvas back to SHADER_READ_ONLY
			cmd.PipelineBarrier(
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
					Image:               paintCanvas.GetImage(),
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

			// Create fence for per-frame synchronization
			fence, err := device.CreateFence(&vk.FenceCreateInfo{})
			if err != nil {
				panic(fmt.Sprintf("Failed to create fence: %v", err))
			}

			queue.Submit([]vk.SubmitInfo{
				{CommandBuffers: []vk.CommandBuffer{cmd}},
			}, fence)

			// Wait for fence (loading must complete before proceeding)
			device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))

			// Store fence for this frame's load operation
			// Even though we waited, future frame switches will check this fence
			// Destroy old fence if it exists
			if frameTexture.LastFence != (vk.Fence{}) {
				device.DestroyFence(frameTexture.LastFence)
			}
			frameTexture.LastFence = fence

			device.FreeCommandBuffers(commandPool, cmdBufs)

			// RAGE-style progressive streaming: Only for existing frames (new frames already started streaming)
			if !isNewFrame {
				// CurrentMip already set to 4 before the blit (for fast loading)
				fmt.Printf("[RAGE] Loaded existing frame %d at mip 4 (128×128) - streaming to full quality...\n", frameNum)

				// Kick off RAGE-style progressive streaming in background
				go streamFrameProgressive(frameNum)
			}

			// Check if this frame needs action replay (upgraded to full resolution while user was viewing another frame)
			if frameTexture.NeedsReplay {
				fmt.Printf("[REPLAY] Frame %d needs replay after upgrade - triggering action replay\n", frameNum)
				frameTexture.NeedsReplay = false // Clear flag
				replayRequested = true           // Trigger replay in next render loop
			}
		}

		// Frame switching synchronization
		var frameSwitchInProgress bool
		var pendingFrameSwitch int = -1            // -1 means no pending switch
		frameSwitchSem := semaphore.NewWeighted(1) // Weight of 1 = only one switch at a time

		// Scrubbing stop detection - upgrade frame from 32×32 to full resolution when scrubbing stops
		var scrubbingStopTimer *time.Timer
		const scrubbingStopDelay = 300 * time.Millisecond // Wait 300ms after last frame switch

		frameActions := make(map[int]*ActionRecorder)
		frameActions[0] = NewActionRecorder() // Initialize frame 0
		actionRecorder := frameActions[0]     // Current frame's recorder
		var currentStroke []PenState          // Currently drawing stroke

		// === Frame Upgrade System (32×32 → 128 → 512 → 2048) ===
		var upgradeToFullResolution func(int)
		upgradeToFullResolution = func(frameNum int) {
			// CRITICAL: Only one upgrade at a time (transferCommandPool is not thread-safe!)
			if !upgradeCommandPoolMutex.TryAcquire(1) {
				fmt.Printf("[UPGRADE] Frame %d: Another upgrade in progress, aborting\n", frameNum)
				return
			}
			defer upgradeCommandPoolMutex.Release(1)

			// CRITICAL: Protect map access (maps are not thread-safe in Go!)
			frameTexturesMutex.RLock()
			frameTexture := frameTextures[frameNum]
			frameTexturesMutex.RUnlock()

			if frameTexture == nil {
				fmt.Printf("[UPGRADE] Frame %d: Frame doesn't exist, aborting\n", frameNum)
				return
			}

			if !frameTexture.IsLowRes {
				fmt.Printf("[UPGRADE] Frame %d: Already full resolution, aborting\n", frameNum)
				return
			}

			// Determine next size
			currentSize := frameTexture.ActualWidth
			var nextSize uint32
			switch currentSize {
			case 32:
				nextSize = 128
			case 128:
				nextSize = 512
			case 512:
				nextSize = 2048
			default:
				return // Unknown size
			}

			fmt.Printf("[UPGRADE] Frame %d: %dx%d → %dx%d\n", frameNum, currentSize, currentSize, nextSize, nextSize)

			// Calculate mip levels
			mipLevels := calculateMipLevels(nextSize, nextSize)

			// CRITICAL: Lock GPU memory operations to prevent race with garbage collector
			// Windows drivers reject concurrent memory allocation/deallocation
			gpuMemoryMutex.Lock()

			// Create new larger texture
			newImage, newMemory, err := device.CreateImageWithMemory(
				nextSize, nextSize,
				vk.FORMAT_R8G8B8A8_UNORM,
				vk.IMAGE_TILING_OPTIMAL,
				vk.IMAGE_USAGE_TRANSFER_SRC_BIT|vk.IMAGE_USAGE_TRANSFER_DST_BIT|vk.IMAGE_USAGE_SAMPLED_BIT,
				vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
				physicalDevice,
			)
			if err != nil {
				gpuMemoryMutex.Unlock()
				fmt.Printf("[UPGRADE] Frame %d: Failed to create image: %v\n", frameNum, err)
				return
			}

			newView, err := device.CreateImageViewForTexture(newImage, vk.FORMAT_R8G8B8A8_UNORM)
			gpuMemoryMutex.Unlock() // Unlock after image view creation
			if err != nil {
				fmt.Printf("[UPGRADE] Frame %d: Failed to create image view: %v\n", frameNum, err)
				device.DestroyImage(newImage)
				device.FreeMemory(newMemory)
				return
			}

			// CRITICAL: Wait for frame's LastFence BEFORE blitting
			// This ensures any ongoing save/load operation completes first
			frameTexture.Mutex.Lock()
			if frameTexture.LastFence != (vk.Fence{}) {
				device.WaitForFences([]vk.Fence{frameTexture.LastFence}, true, ^uint64(0))
			}
			frameTexture.Mutex.Unlock()

			// Use transferCommandPool for background blit
			cmdBufs, err := device.AllocateCommandBuffers(&vk.CommandBufferAllocateInfo{
				CommandPool:        transferCommandPool,
				Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
				CommandBufferCount: 1,
			})
			if err != nil {
				fmt.Printf("[UPGRADE] Frame %d: Failed to allocate command buffers: %v\n", frameNum, err)
				device.DestroyImageView(newView)
				device.DestroyImage(newImage)
				device.FreeMemory(newMemory)
				return
			}
			cmd := cmdBufs[0]

			cmd.Begin(&vk.CommandBufferBeginInfo{
				Flags: vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
			})

			// Transition old image to TRANSFER_SRC
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               frameTexture.Image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Transition new image to TRANSFER_DST
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       0,
					DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
					NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               newImage,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Blit old → new (upscaling)
			cmd.CmdBlitImage(
				frameTexture.Image, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
				newImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
				[]vk.ImageBlit{{
					SrcSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       0,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					SrcOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(currentSize), Y: int32(currentSize), Z: 1},
					},
					DstSubresource: vk.ImageSubresourceLayers{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						MipLevel:       0,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
					DstOffsets: [2]vk.Offset3D{
						{X: 0, Y: 0, Z: 0},
						{X: int32(nextSize), Y: int32(nextSize), Z: 1},
					},
				}},
				vk.FILTER_LINEAR,
			)

			// Transition old image back to SHADER_READ_ONLY
			cmd.PipelineBarrier(
				vk.PIPELINE_STAGE_TRANSFER_BIT,
				vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
				0,
				[]vk.ImageMemoryBarrier{{
					SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
					DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
					OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
					NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
					SrcQueueFamilyIndex: ^uint32(0),
					DstQueueFamilyIndex: ^uint32(0),
					Image:               frameTexture.Image,
					SubresourceRange: vk.ImageSubresourceRange{
						AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
						BaseMipLevel:   0,
						LevelCount:     1,
						BaseArrayLayer: 0,
						LayerCount:     1,
					},
				}},
			)

			// Generate mipmaps if reaching 2048
			if nextSize == 2048 {
				// Transition to TRANSFER_SRC for mipmap generation
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					0,
					[]vk.ImageMemoryBarrier{{
						SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
						DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               newImage,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				// Generate mipmaps
				for i := uint32(1); i < mipLevels; i++ {
					prevWidth := nextSize >> (i - 1)
					prevHeight := nextSize >> (i - 1)
					currWidth := nextSize >> i
					currHeight := nextSize >> i
					if currWidth == 0 {
						currWidth = 1
					}
					if currHeight == 0 {
						currHeight = 1
					}

					// Transition current mip to DST
					cmd.PipelineBarrier(
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						0,
						[]vk.ImageMemoryBarrier{{
							SrcAccessMask:       0,
							DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_UNDEFINED,
							NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               newImage,
							SubresourceRange: vk.ImageSubresourceRange{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								BaseMipLevel:   i,
								LevelCount:     1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
						}},
					)

					// Blit previous mip → current mip
					cmd.CmdBlitImage(
						newImage, vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						newImage, vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						[]vk.ImageBlit{{
							SrcSubresource: vk.ImageSubresourceLayers{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								MipLevel:       i - 1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
							SrcOffsets: [2]vk.Offset3D{
								{X: 0, Y: 0, Z: 0},
								{X: int32(prevWidth), Y: int32(prevHeight), Z: 1},
							},
							DstSubresource: vk.ImageSubresourceLayers{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								MipLevel:       i,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
							DstOffsets: [2]vk.Offset3D{
								{X: 0, Y: 0, Z: 0},
								{X: int32(currWidth), Y: int32(currHeight), Z: 1},
							},
						}},
						vk.FILTER_LINEAR,
					)

					// Transition current mip to SRC for next iteration
					cmd.PipelineBarrier(
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						0,
						[]vk.ImageMemoryBarrier{{
							SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
							DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
							OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
							NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
							SrcQueueFamilyIndex: ^uint32(0),
							DstQueueFamilyIndex: ^uint32(0),
							Image:               newImage,
							SubresourceRange: vk.ImageSubresourceRange{
								AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
								BaseMipLevel:   i,
								LevelCount:     1,
								BaseArrayLayer: 0,
								LayerCount:     1,
							},
						}},
					)
				}

				// Transition all mips to SHADER_READ_ONLY
				cmd.PipelineBarrier(
					vk.PIPELINE_STAGE_TRANSFER_BIT,
					vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
					0,
					[]vk.ImageMemoryBarrier{{
						SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
						DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
						OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
						SrcQueueFamilyIndex: ^uint32(0),
						DstQueueFamilyIndex: ^uint32(0),
						Image:               newImage,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     mipLevels,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)

				fmt.Printf("[MIPMAPS] Frame %d: Generated %d mip levels\n", frameNum, mipLevels)
			} else {
				// Just transition to SHADER_READ_ONLY
				cmd.PipelineBarrier(
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
						Image:               newImage,
						SubresourceRange: vk.ImageSubresourceRange{
							AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
							BaseMipLevel:   0,
							LevelCount:     1,
							BaseArrayLayer: 0,
							LayerCount:     1,
						},
					}},
				)
			}

			cmd.End()

			// Create fence for this upgrade operation
			fence, err := device.CreateFence(&vk.FenceCreateInfo{})
			if err != nil {
				fmt.Printf("[UPGRADE] Frame %d: Failed to create fence: %v\n", frameNum, err)
				device.DestroyImageView(newView)
				device.DestroyImage(newImage)
				device.FreeMemory(newMemory)
				return
			}

			// Submit and wait for THIS upgrade only
			queue.Submit([]vk.SubmitInfo{
				{CommandBuffers: []vk.CommandBuffer{cmd}},
			}, fence)
			device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))

			// Clean up fence (don't free command buffers - let pool manage them)
			device.DestroyFence(fence)

			// Verify frame still exists
			frameTexturesMutex.RLock()
			stillExists := frameTextures[frameNum] != nil
			frameTexturesMutex.RUnlock()

			if !stillExists {
				fmt.Printf("[UPGRADE] Frame %d: Deleted during upgrade, cleaning up\n", frameNum)
				device.DestroyImageView(newView)
				device.DestroyImage(newImage)
				device.FreeMemory(newMemory)
				return
			}

			// CRITICAL: Send upgrade to main thread for safe descriptor update
			// Updating descriptor sets from goroutine causes contention on Windows (WDDM)
			// Main thread will handle resource swap + descriptor update + garbage queueing
			upgradeQueue <- PendingUpgrade{
				FrameIndex:   frameNum,
				NewImage:     newImage,
				NewView:      newView,
				NewMemory:    newMemory,
				NewMipLevels: mipLevels,
				NewSize:      nextSize,
			}
			fmt.Printf("[UPGRADE] Frame %d: Heavy lifting done (%dx%d), queued for main thread swap\n", frameNum, nextSize, nextSize)

			// Main thread will handle the rest:
			// - Resource swap
			// - Descriptor set update (safe on main thread!)
			// - Garbage queueing
			// - Scheduling next upgrade or starting RAGE streaming
			return
		}

		// === Action-based Undo System (Per-Frame) ===
		// Each frame has its own independent action history
		// Declare switchToFrame first so it can call itself for pending switches
		var switchToFrame func(int)
		switchToFrame = func(newFrame int) {
			// If a switch is already in progress, queue this request instead of skipping
			// This ensures we always reach the destination frame
			if !frameSwitchSem.TryAcquire(1) {
				fmt.Printf("[SWITCH] Frame switch in progress, queueing frame %d\n", newFrame)
				pendingFrameSwitch = newFrame // Remember the most recent request
				return
			}
			defer frameSwitchSem.Release(1)

			// Set flag to pause render loop
			frameSwitchInProgress = true
			defer func() { frameSwitchInProgress = false }()

			if newFrame < 0 || newFrame >= timeline.TotalFrames {
				fmt.Printf("Frame %d out of range (0-%d)\n", newFrame, timeline.TotalFrames-1)
				return
			}

			if newFrame == timeline.CurrentFrame {
				return // Already on this frame
			}

			oldFrame := timeline.CurrentFrame
			fmt.Printf("Switching from frame %d to frame %d\n", oldFrame, newFrame)

			// Per-frame fence synchronization: Only wait for the OLD frame's GPU work
			// This is MUCH faster than queue.WaitIdle() which waits for everything
			if oldFrameTexture := frameTextures[oldFrame]; oldFrameTexture != nil && oldFrameTexture.LastFence != (vk.Fence{}) {
				fmt.Printf("[SWITCH] Waiting for frame %d's fence...\n", oldFrame)
				time.Sleep(1 * time.Millisecond) // Still needed to prevent hangs
				err := device.WaitForFences([]vk.Fence{oldFrameTexture.LastFence}, true, ^uint64(0))
				if err != nil {
					panic(fmt.Sprintf("Failed to wait for frame %d fence: %v", oldFrame, err))
				}
				fmt.Printf("[SWITCH] Frame %d fence signaled\n", oldFrame)
			} else {
				// No fence to wait for (first frame or frame never saved)
				fmt.Printf("[SWITCH] No fence for frame %d, skipping wait\n", oldFrame)
			}
			// Update timeline FIRST to prevent upgrade from thinking old frame is still current
			timeline.CurrentFrame = newFrame
			// Switch to new frame's action recorder (create if doesn't exist)
			if frameActions[newFrame] == nil {
				frameActions[newFrame] = NewActionRecorder()
				fmt.Printf("[ACTIONS] Created new action recorder for frame %d\n", newFrame)
			}
			actionRecorder = frameActions[newFrame]
			fmt.Printf("[ACTIONS] Switched to frame %d action history (%d actions)\n", newFrame, len(actionRecorder.GetFullHistory()))

			// Save OLD frame (before it gets upgraded)
			saveCurrentFrame(oldFrame)

			// RAGE: Cancel streaming for old frame before switching
			if oldFrameTexture := frameTextures[oldFrame]; oldFrameTexture != nil {
				// Non-blocking send to cancel channel
				select {
				case oldFrameTexture.StreamCancel <- true:
					fmt.Printf("[RAGE] Cancelled streaming for old frame %d\n", oldFrame)
				default:
					// Channel already has a value or streaming finished
				}
			}

			// Load new frame
			loadFrame(newFrame)

			// CRITICAL: Update brush descriptor set to point to the newly-loaded canvas
			// Without this, the GPU shader reads from stale descriptor bindings and displays the old frame
			device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
				{
					DstSet:          brushDescriptorSet,
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

			// Check if this frame needs replay (after upgrade or if actions were added)
			// CRITICAL: Always replay frames with strokes, even if NeedsReplay=false
			// This ensures strokes appear even when loading from low-res mips (RAGE streaming)
			frameTexturesMutex.RLock()
			needsReplay := frameTextures[newFrame] != nil && frameTextures[newFrame].NeedsReplay
			frameTexturesMutex.RUnlock()

			hasStrokes := frameActions[newFrame] != nil && len(frameActions[newFrame].GetHistory()) > 0

			if needsReplay || hasStrokes {
				fmt.Printf("[REPLAY] Frame %d needs replay (NeedsReplay=%v, hasStrokes=%v)\n", newFrame, needsReplay, hasStrokes)
				// NOTE: NeedsReplay flag will be cleared AFTER replay completes (not here)
				// This prevents race conditions where frame switches before replay executes
				replayRequested = true // Trigger replay in next render loop
			}

			// Update frame counter text
			frameComp := world.GetText(frameCounterText)
			if frameComp != nil {
				frameComp.Content = fmt.Sprintf("Frame: %d/%d", timeline.CurrentFrame, timeline.TotalFrames-1)
			}

			// Check if there's a pending frame switch request (from rapid scrubbing)
			// This ensures we always reach the final destination frame
			if pendingFrameSwitch >= 0 && pendingFrameSwitch != timeline.CurrentFrame {
				nextFrame := pendingFrameSwitch
				pendingFrameSwitch = -1 // Clear pending request
				fmt.Printf("[SWITCH] Processing queued switch to frame %d\n", nextFrame)
				go switchToFrame(nextFrame) // Trigger next switch asynchronously
			}

			// Reset scrubbing stop timer - upgrade to full resolution after 300ms of no frame switches
			if scrubbingStopTimer != nil {
				scrubbingStopTimer.Stop() // Cancel previous timer
			}
			scrubbingStopTimer = time.AfterFunc(scrubbingStopDelay, func() {
				fmt.Printf("[SCRUB] Scrubbing stopped on frame %d, upgrading to full resolution...\n", timeline.CurrentFrame)
				go upgradeToFullResolution(timeline.CurrentFrame)
			})
		}

		// Initialize paintCanvas with frame 0's content
		loadFrame(0)
		fmt.Println("Paint canvas initialized with frame 0")
		// Start upgrade timer for initial frame
		scrubbingStopTimer = time.AfterFunc(scrubbingStopDelay, func() {
			fmt.Printf("[SCRUB] Initial load complete, upgrading frame 0 to full resolution...\n")
			go upgradeToFullResolution(0)
		})

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
		uiButtonTextLayer, uiButtonTextImage, uiButtonTextView, uiButtonTextMem := createUILayer("ButtonText", UILayerButtonOver)
		defer device.DestroyImage(uiButtonTextImage)
		defer device.DestroyImageView(uiButtonTextView)
		defer device.FreeMemory(uiButtonTextMem)

		uiTextLayer, uiTextImage, uiTextView, uiTextMem := createUILayer("Text", UILayerText)
		defer device.DestroyImage(uiTextImage)
		defer device.DestroyImageView(uiTextView)
		defer device.FreeMemory(uiTextMem)

		// Create color picker layer (renders color picker UI)
		colorPickerLayer, colorPickerImage, colorPickerView, colorPickerMem := createUILayer("ColorPicker", UILayerColorPicker)
		defer device.DestroyImage(colorPickerImage)
		defer device.DestroyImageView(colorPickerView)
		defer device.FreeMemory(colorPickerMem)

		world.MakeScreenSpace(uiTextLayer, true)
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
		// frameCounter is now a global variable (used for garbage collection timing)

		//timer := time.Now().UnixMilli()

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

							// Detect pen-up when pressure goes to 0 (fixes stroke not ending when lifting pen)
							if penPressure <= 0.01 && penDown {
								penDown = false
								fmt.Println("[PEN] Pen up detected via pressure = 0")
							} else if penPressure > 0.01 && !penDown {
								penDown = true
								fmt.Println("[PEN] Pen down detected via pressure > 0")
							}
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

						// Comma (,) - Previous frame
						if !ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_COMMA {
							if timeline.CurrentFrame > 0 {
								switchToFrame(timeline.CurrentFrame - 1)
								sdl.Delay(10)

							} else {
								fmt.Println("Already at first frame")
							}
						}

						// Period (.) - Next frame
						if !ctrl && !shift && keyEvent.Scancode == sdl.SCANCODE_PERIOD {
							if timeline.CurrentFrame < timeline.TotalFrames-1 {
								switchToFrame(timeline.CurrentFrame + 1)
								sdl.Delay(10)
							} else {
								fmt.Println("Already at last frame")
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
			// CRITICAL: Skip rendering if frame switch is in progress
			// This prevents segfaults when frame switching modifies resources while rendering
			if frameSwitchInProgress {
				time.Sleep(1 * time.Millisecond)
				continue
			}

			go func() {
				if !running {
					return
				}
			}()
			//currentFrameLast = currentFrame
			frameCounter++

			// === GPU Garbage Collection ===
			// Safely destroy Vulkan resources that are no longer in use by the GPU
			// Resources are only destroyed after FRAMES_IN_FLIGHT + 2 frames have passed
			// to guarantee all command buffers referencing them have completed execution
			garbageMutex.Lock()
			n := 0
			for _, trash := range garbageQueue {
				// If the trash is older than FRAMES_IN_FLIGHT + 2, it's safe to delete
				if frameCounter > trash.DeathFrame+uint64(len(inFlightFences))+2 {
					// Lock GPU memory operations to prevent race with upgrade goroutine
					gpuMemoryMutex.Lock()
					device.DestroyImageView(trash.ImageView)
					device.DestroyImage(trash.Image)
					device.FreeMemory(trash.Memory)
					gpuMemoryMutex.Unlock()
					fmt.Printf("[GARBAGE] Collected resources from death frame %d (current: %d, in-flight: %d)\n", trash.DeathFrame, frameCounter, len(inFlightFences))
				} else {
					// Keep it - still potentially in use
					garbageQueue[n] = trash
					n++
				}
			}
			if n > 0 && frameCounter%60 == 0 {
				fmt.Printf("[GARBAGE] %d items still queued (oldest death frame: %d, current frame: %d)\n", n, garbageQueue[0].DeathFrame, frameCounter)
			}
			garbageQueue = garbageQueue[:n]
			garbageMutex.Unlock()

			// === PROCESS PENDING UPGRADES (Main Thread Safe) ===
			// Upgrades are prepared in background goroutines, then applied here
			// This prevents descriptor set contention (Windows WDDM requirement)
			select {
			case upgrade := <-upgradeQueue:
				frameTexturesMutex.Lock()
				ft := frameTextures[upgrade.FrameIndex]

				// 1. Capture old resources for garbage collection
				oldImage := ft.Image
				oldView := ft.ImageView
				oldMem := ft.Memory

				// 2. Atomic swap in Go memory
				ft.Image = upgrade.NewImage
				ft.ImageView = upgrade.NewView
				ft.Memory = upgrade.NewMemory
				ft.MipLevels = upgrade.NewMipLevels
				ft.ActualWidth = upgrade.NewSize
				ft.ActualHeight = upgrade.NewSize

				// 4. Update descriptor set (SAFE on main thread!)
				// This is safe because:
				// - We're on the main thread (no concurrent access)
				// - We do this BEFORE recording new command buffers
				// - Old image still valid (queued for garbage collection)
				// - New image ready (background thread waited for blit)
				textureIndex := ft.TextureIndex
				binding := textureIndex / 16384
				arrayElement := textureIndex % 16384

				device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
					{
						DstSet:          globalBindlessDescriptorSet,
						DstBinding:      binding,
						DstArrayElement: arrayElement,
						DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
						ImageInfo: []vk.DescriptorImageInfo{{
							Sampler:     textureSampler,
							ImageView:   upgrade.NewView,
							ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
						}},
					},
				})

				// 5. Queue old resources for garbage collection
				garbageMutex.Lock()
				garbageQueue = append(garbageQueue, GPUGarbage{
					Image:      oldImage,
					ImageView:  oldView,
					Memory:     oldMem,
					DeathFrame: frameCounter,
				})
				garbageMutex.Unlock()

				// 6. Update frame status and schedule next step
				if upgrade.NewSize == 2048 {
					ft.IsLowRes = false
					ft.CurrentMip = 4 // Start RAGE streaming from mip 4
					ft.NeedsReplay = true
					frameTexturesMutex.Unlock()
					fmt.Printf("[MAIN] Swapped frame %d to 2048×2048 safely\n", upgrade.FrameIndex)
					go streamFrameProgressive(upgrade.FrameIndex)
				} else {
					ft.IsLowRes = true
					frameTexturesMutex.Unlock()
					fmt.Printf("[MAIN] Swapped frame %d to %dx%d safely\n", upgrade.FrameIndex, upgrade.NewSize, upgrade.NewSize)
					// Schedule next upgrade step
					time.AfterFunc(100*time.Millisecond, func() {
						upgradeToFullResolution(upgrade.FrameIndex)
					})
				}

			default:
				// No upgrades pending, continue
			}

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

			// DEBUG: Check replay state every frame
			if frameCounter%60 == 0 && replayRequested {
				fmt.Printf("[DEBUG] Render loop: replayRequested=%v, actionRecorder.GetIndex()=%d\n", replayRequested, actionRecorder.GetIndex())
			}

			// === ACTION-BASED REPLAY: Clear and redraw all strokes ===
			if replayRequested {
				fmt.Printf("\n=== REPLAY TRIGGERED ===\n")
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

				// Clear both canvases to transparent (not white, or it will obscure background layer!)
				transparentColor := vk.ClearColorValue{Float32: [4]float32{0.0, 0.0, 0.0, 0.0}}
				cmd.CmdClearColorImage(
					paintCanvasA.GetImage(),
					vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
					&transparentColor,
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
					&transparentColor,
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
				targetIndex := len(history)

				// === Find nearest snapshot before target index ===
				fmt.Printf("[SNAPSHOT DEBUG] Looking for snapshots before target index %d:\n", targetIndex)
				for i := range snapshots {
					fmt.Printf("  Snapshot %d: Valid=%v, ActionIndex=%d\n", i, snapshots[i].IsValid, snapshots[i].ActionIndex)
				}
				var bestSnapshot *CanvasSnapshot = nil
				bestSnapshotSlot := -1
				for i := range snapshots {
					if snapshots[i].IsValid && snapshots[i].ActionIndex < targetIndex {
						// Pick snapshot with highest ActionIndex, or if tied, highest slot (most recent)
						if bestSnapshot == nil ||
							snapshots[i].ActionIndex > bestSnapshot.ActionIndex ||
							(snapshots[i].ActionIndex == bestSnapshot.ActionIndex && i > bestSnapshotSlot) {
							bestSnapshot = &snapshots[i]
							bestSnapshotSlot = i
						}
					}
				}

				startIdx := 0 // Start replaying from beginning
				if bestSnapshot != nil {
					fmt.Printf("[SNAPSHOT] Found snapshot at action %d (slot %d), skipping %d actions!\n",
						bestSnapshot.ActionIndex, bestSnapshotSlot, bestSnapshot.ActionIndex)
					startIdx = bestSnapshot.ActionIndex

					// Restore snapshot image to paintCanvasB (the SOURCE, not destination!)
					// The descriptor set reads from paintCanvasSource which is B
					cmd.PipelineBarrier(
						vk.PIPELINE_STAGE_TRANSFER_BIT,
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

					cmd.CmdCopyImage(
						bestSnapshot.Image,
						vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						paintCanvasB.GetImage(),
						vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						[]vk.ImageCopy{
							{
								SrcSubresource: vk.ImageSubresourceLayers{
									AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
									MipLevel:       0,
									BaseArrayLayer: 0,
									LayerCount:     1,
								},
								SrcOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
								DstSubresource: vk.ImageSubresourceLayers{
									AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
									MipLevel:       0,
									BaseArrayLayer: 0,
									LayerCount:     1,
								},
								DstOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
								Extent:    vk.Extent3D{Width: paintCanvasB.GetWidth(), Height: paintCanvasB.GetHeight(), Depth: 1},
							},
						},
					)

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

					// IMPORTANT: Also copy snapshot to paintCanvasA (destination)!
					// The brush shader only writes pixels covered by stamps, so uncovered pixels
					// in the destination need to have the snapshot content as background.
					fmt.Printf("[SNAPSHOT] Also copying snapshot to paintCanvasA (destination) for proper background\n")
					cmd.PipelineBarrier(
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						0,
						[]vk.ImageMemoryBarrier{
							{
								SrcAccessMask:       0,
								DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
								OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
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
						},
					)

					cmd.CmdCopyImage(
						bestSnapshot.Image,
						vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
						paintCanvasA.GetImage(),
						vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
						[]vk.ImageCopy{
							{
								SrcSubresource: vk.ImageSubresourceLayers{
									AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
									MipLevel:       0,
									BaseArrayLayer: 0,
									LayerCount:     1,
								},
								SrcOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
								DstSubresource: vk.ImageSubresourceLayers{
									AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
									MipLevel:       0,
									BaseArrayLayer: 0,
									LayerCount:     1,
								},
								DstOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
								Extent:    vk.Extent3D{Width: paintCanvasA.GetWidth(), Height: paintCanvasA.GetHeight(), Depth: 1},
							},
						},
					)

					cmd.PipelineBarrier(
						vk.PIPELINE_STAGE_TRANSFER_BIT,
						vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
						0,
						[]vk.ImageMemoryBarrier{
							{
								SrcAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
								DstAccessMask:       vk.ACCESS_COLOR_ATTACHMENT_WRITE_BIT,
								OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
								NewLayout:           vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
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
						},
					)

					fmt.Printf("[SNAPSHOT] Canvas restored from snapshot to BOTH A and B\n")
				}

				fmt.Printf("[SNAPSHOT DEBUG] Starting replay of %d actions (from index %d to %d)\n", len(history), startIdx, targetIndex)
				if bestSnapshot != nil {
					fmt.Printf("[SNAPSHOT DEBUG] Using snapshot from slot with ActionIndex=%d (skipping stroke rendering for actions 0-%d)\n", bestSnapshot.ActionIndex, startIdx-1)
				} else {
					fmt.Printf("[SNAPSHOT DEBUG] No snapshot found, replaying from scratch\n")
				}
				// Always process ALL actions to ensure layer state changes are applied!
				// But only render strokes starting from startIdx (snapshot covers earlier strokes)
				for actionIdx := 0; actionIdx < len(history); actionIdx++ {
					action := history[actionIdx]
					isLastStroke := (actionIdx == len(history)-1)

					switch action.Type {
					case ActionTypeStroke:
						// Skip strokes that are already in the snapshot
						if actionIdx < startIdx {
							fmt.Printf("  Skipping stroke %d/%d (covered by snapshot)\n", actionIdx+1, len(history))
							continue
						}

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
						}

						// === Save snapshot at intervals during replay ===
						// After swap (if swapped), result is in paintCanvasSource
						// If no swap (last stroke), result is in paintCanvas
						if (actionIdx+1)%snapshotInterval == 0 {
							fmt.Printf("[SNAPSHOT] Saving canvas state at ActionIndex=%d (slot %d), isLastStroke=%v, targetIdx=%d\n",
								actionIdx+1, nextSnapshotSlot, isLastStroke, targetIndex)

							// Determine which canvas has the result
							var resultCanvas canvas.Canvas
							var canvasName string
							if !isLastStroke {
								// We swapped, so result is in paintCanvasSource
								resultCanvas = paintCanvasSource
								canvasName = "paintCanvasSource"
							} else {
								// Didn't swap, result is in paintCanvas
								resultCanvas = paintCanvas
								canvasName = "paintCanvas"
							}
							fmt.Printf("[SNAPSHOT] Copying from %s to snapshot slot %d\n", canvasName, nextSnapshotSlot)

							// Copy result canvas to snapshot
							// IMPORTANT: Snapshot might be in TRANSFER_SRC if previously used!
							snapshotOldLayout := vk.IMAGE_LAYOUT_UNDEFINED
							snapshotSrcAccess := vk.AccessFlags(0)
							if snapshots[nextSnapshotSlot].IsValid {
								snapshotOldLayout = vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL
								snapshotSrcAccess = vk.ACCESS_TRANSFER_READ_BIT
							}
							cmd.PipelineBarrier(
								vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
								vk.PIPELINE_STAGE_TRANSFER_BIT,
								0,
								[]vk.ImageMemoryBarrier{
									{
										SrcAccessMask:       vk.ACCESS_SHADER_READ_BIT,
										DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
										OldLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
										NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
										SrcQueueFamilyIndex: ^uint32(0),
										DstQueueFamilyIndex: ^uint32(0),
										Image:               resultCanvas.GetImage(),
										SubresourceRange: vk.ImageSubresourceRange{
											AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
											BaseMipLevel:   0,
											LevelCount:     1,
											BaseArrayLayer: 0,
											LayerCount:     1,
										},
									},
									{
										SrcAccessMask:       snapshotSrcAccess,
										DstAccessMask:       vk.ACCESS_TRANSFER_WRITE_BIT,
										OldLayout:           snapshotOldLayout,
										NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
										SrcQueueFamilyIndex: ^uint32(0),
										DstQueueFamilyIndex: ^uint32(0),
										Image:               snapshots[nextSnapshotSlot].Image,
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

							// Copy image from result canvas to snapshot
							cmd.CmdCopyImage(
								resultCanvas.GetImage(),
								vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
								snapshots[nextSnapshotSlot].Image,
								vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
								[]vk.ImageCopy{
									{
										SrcSubresource: vk.ImageSubresourceLayers{
											AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
											MipLevel:       0,
											BaseArrayLayer: 0,
											LayerCount:     1,
										},
										SrcOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
										DstSubresource: vk.ImageSubresourceLayers{
											AspectMask:     vk.IMAGE_ASPECT_COLOR_BIT,
											MipLevel:       0,
											BaseArrayLayer: 0,
											LayerCount:     1,
										},
										DstOffset: vk.Offset3D{X: 0, Y: 0, Z: 0},
										Extent:    vk.Extent3D{Width: resultCanvas.GetWidth(), Height: resultCanvas.GetHeight(), Depth: 1},
									},
								},
							)

							// Transition back
							cmd.PipelineBarrier(
								vk.PIPELINE_STAGE_TRANSFER_BIT,
								vk.PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
								0,
								[]vk.ImageMemoryBarrier{
									{
										SrcAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
										DstAccessMask:       vk.ACCESS_SHADER_READ_BIT,
										OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
										NewLayout:           vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
										SrcQueueFamilyIndex: ^uint32(0),
										DstQueueFamilyIndex: ^uint32(0),
										Image:               resultCanvas.GetImage(),
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
										DstAccessMask:       vk.ACCESS_TRANSFER_READ_BIT,
										OldLayout:           vk.IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
										NewLayout:           vk.IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
										SrcQueueFamilyIndex: ^uint32(0),
										DstQueueFamilyIndex: ^uint32(0),
										Image:               snapshots[nextSnapshotSlot].Image,
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

							// Mark snapshot as valid
							snapshots[nextSnapshotSlot].ActionIndex = actionIdx + 1
							snapshots[nextSnapshotSlot].IsValid = true
							nextSnapshotSlot = (nextSnapshotSlot + 1) % maxSnapshots
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

				// Clear NeedsReplay flag now that replay has actually completed
				frameTexturesMutex.Lock()
				if frameTextures[timeline.CurrentFrame] != nil {
					frameTextures[timeline.CurrentFrame].NeedsReplay = false
				}
				frameTexturesMutex.Unlock()

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
			if penDown && !prevPenDown && penPressure > 0.001 {
				// Check if current frame is low-res - deny strokes on low-res frames
				frameTexturesMutex.RLock()
				currentFrameTexture := frameTextures[timeline.CurrentFrame]
				frameTexturesMutex.RUnlock()

				if currentFrameTexture != nil && currentFrameTexture.IsLowRes {
					fmt.Printf("[BRUSH DENIED] Cannot paint on low-res frame %d (size: %dx%d) - waiting for upgrade to full resolution\n",
						timeline.CurrentFrame, currentFrameTexture.ActualWidth, currentFrameTexture.ActualHeight)
				} else {
					// Stroke just started
					strokeActive = true
					isFirstFrameOfStroke = true // Skip interpolation on first frame
					strokeFrameCount = 0        // Reset frame counter

					fmt.Printf("=== Stroke started (action-based undo) ===\n")
				}
			}

			cXLast, cYLast := float32(0.0), float32(0.0)

			// === PASS 0: Render brush strokes to canvas (if pen is down) ===
			if penDown && penPressure > 0.001 {
				// Defensive check: Skip rendering if current frame is low-res
				frameTexturesMutex.RLock()
				currentFrameTexture := frameTextures[timeline.CurrentFrame]
				isCurrentFrameLowRes := currentFrameTexture != nil && currentFrameTexture.IsLowRes
				frameTexturesMutex.RUnlock()

				if isCurrentFrameLowRes {
					// Skip rendering - frame is still upgrading
					goto skipBrushRender
				}

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
				if ((prevPenDown && !penDown) || penPressure < 0.001) && strokeActive {
					// Save completed stroke to action history
					if len(currentStroke) > 10 {
						// Add new stroke to action history
						// Get current brush color from color picker
						strokeR, strokeG, strokeB := hsv2rgb(colorPicker.Hue, colorPicker.Saturation, colorPicker.Value)
						newStroke := Stroke{
							States: currentStroke,
							Color:  [4]float32{strokeR, strokeG, strokeB, 1.0},
							Radius: brushRadius,
						}
						actionRecorder.RecordStroke(newStroke)

						// Mark this frame as needing replay (for when it's upgraded to full resolution)
						frameTexturesMutex.Lock()
						if frameTextures[timeline.CurrentFrame] != nil {
							frameTextures[timeline.CurrentFrame].NeedsReplay = true
						}
						frameTexturesMutex.Unlock()

						// Note: Snapshots are created during replay, not during normal drawing
						// (we can't call vkCmdCopyImage outside of command buffer recording)

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

		skipBrushRender:
			// ===  PASS 1: Render each layer to its framebuffer ===
			// Only render ROOT layers (layers with no parent)
			// Groups will composite their children to their framebuffer
			for _, entity := range sortedLayers {
				// Skip layers that have a parent (they'll be rendered by their group)
				if !world.IsRootLayer(entity) {
					continue
				}

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

				if world.IsGroup(entity) {
					// Render layer group - composite all children to group's framebuffer
					if frameCounter%60 == 0 {
						//fmt.Printf("[GROUP] Rendering layer group (entity %d)\n", entity)
					}

					children := world.GetChildren(entity)
					if children != nil && len(children.ChildEntities) > 0 {
						// Bind composite pipeline to render children
						cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, compositePipeline)
						cmd.BindDescriptorSets(
							vk.PIPELINE_BIND_POINT_GRAPHICS,
							compositePipelineLayout,
							0,
							[]vk.DescriptorSet{globalBindlessDescriptorSet},
							nil,
						)

						cmd.SetViewport(0, []vk.Viewport{{
							X:        0,
							Y:        0,
							Width:    float32(swapExtent.Width),
							Height:   float32(swapExtent.Height),
							MinDepth: 0.0,
							MaxDepth: 1.0,
						}})
						cmd.SetScissor(0, []vk.Rect2D{{
							Offset: vk.Offset2D{X: 0, Y: 0},
							Extent: swapExtent,
						}})
						cmd.BindVertexBuffers(0, []vk.Buffer{vertexBuffer}, []uint64{0})
						cmd.BindIndexBuffer(indexBuffer, 0, vk.INDEX_TYPE_UINT16)

						// Composite each child to the group's framebuffer
						for _, childEntity := range children.ChildEntities {
							childTexture := world.GetTextureData(childEntity)
							childBlend := world.GetBlendMode(childEntity)
							childTransform := world.GetTransform(childEntity)

							if childTexture != nil && childBlend != nil && childTransform != nil {
								if frameCounter%60 == 0 {
									//fmt.Printf("[GROUP] Compositing child %d to group framebuffer\n", childEntity)
								}

								// Composite the child using its texture index
								systems.CompositeLayer(
									&systems.CompositeContext{
										CommandBuffer:           cmd,
										CompositePipeline:       compositePipeline,
										CompositePipelineLayout: compositePipelineLayout,
										SwapExtent:              swapExtent,
										BindlessDescriptorSet:   globalBindlessDescriptorSet,
									},
									childTexture.TextureIndex,
									childBlend.Opacity,
									childTransform.X,
									childTransform.Y,
									childTransform.ScaleX,
									0, 0, 1, // No camera transform for now (TODO: handle properly)
									false, // Not screen space
								)
							}
						}
					}
				} else if entity == layerBg {
					// Render background layer - solid color full-screen quad
					if frameCounter%60 == 0 {
						//fmt.Printf("[BACKGROUND] Rendering background layer (entity %d)\n", entity)
					}
					cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, uiRectPipeline)
					cmd.SetViewport(0, []vk.Viewport{{
						X:        0,
						Y:        0,
						Width:    float32(swapExtent.Width),
						Height:   float32(swapExtent.Height),
						MinDepth: 0.0,
						MaxDepth: 1.0,
					}})
					cmd.SetScissor(0, []vk.Rect2D{{
						Offset: vk.Offset2D{X: 0, Y: 0},
						Extent: swapExtent,
					}})
					cmd.BindVertexBuffers(0, []vk.Buffer{brushVertexBuffer}, []uint64{0})

					// Render full-screen background quad
					type UIRectPushConstants struct {
						PosX   float32
						PosY   float32
						Width  float32
						Height float32
						ColorR float32
						ColorG float32
						ColorB float32
						ColorA float32
					}
					bgColor := UIRectPushConstants{
						PosX:   0,
						PosY:   0,
						Width:  float32(swapExtent.Width),
						Height: float32(swapExtent.Height),
						ColorR: 0.18, // Nice gray background (like Photoshop)
						ColorG: 0.18,
						ColorB: 0.18,
						ColorA: 1.0,
					}
					cmd.CmdPushConstants(
						uiRectPipelineLayout,
						vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
						0,
						32,
						unsafe.Pointer(&bgColor),
					)
					cmd.Draw(6, 1, 0, 0)
				} else if entity == uiButtonBaseLayer {
					// Render button base rectangles only (no text)
					//buttonCount := len(world.QueryUIButtons())
					if frameCounter%60 == 0 { // Log every 60 frames to avoid spam
						//fmt.Printf("[UI] Rendering %d button bases (undoValid=%v)\n", buttonCount, undoValid)
					}
					systems.RenderUIButtonBases(world, uiCtx)
				} else if entity == uiButtonTextLayer {
					// Render button text labels only (no rectangles)
					// Pass imageIndex to use the correct per-frame staging buffers
					systems.RenderUIButtonLabels(world, uiCtx, int(imageIndex))
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
			// Only composite ROOT layers - children are already composited into their parent groups
			for _, entity := range sortedLayers {
				// Skip layers that have a parent (they're inside a group)
				if !world.IsRootLayer(entity) {
					continue
				}

				textureData := world.GetTextureData(entity)
				blendMode := world.GetBlendMode(entity)
				transform := world.GetTransform(entity)
				if textureData != nil && blendMode != nil && transform != nil {
					// Check if entity is in screen space (ignores camera transforms)
					screenSpace := world.GetScreenSpace(entity)
					isScreenSpace := screenSpace != nil && screenSpace.Enabled

					if entity == layerBg {
						blendMode.Opacity = float32(math.Sin(float64(time.Now().UnixMilli())/1000.0))/2.0 + 0.5

						//fmt.Printf("[BACKGROUND] Compositing background layer (textureIndex=%d, opacity=%.2f)\n",
						//	textureData.TextureIndex, blendMode.Opacity)
					}

					if frameCounter%60 == 0 && world.IsGroup(entity) {
						//fmt.Printf("[GROUP COMPOSITE] Entity=%d, TextureIndex=%d, Opacity=%.2f\n",
						//	entity, textureData.TextureIndex, blendMode.Opacity)
					}

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

			// Render all text entities (frame counter, debug text, etc.)
			// Use buffer set 1 for text entities (buffer set 0 is for button labels)
			// Render ONLY text entities (not button labels, those are on button text layer)
			const bufferSetTextEntities = 1
			systems.RenderText(world, textRenderer, cmd, swapExtent.Width, swapExtent.Height, device, bufferSetTextEntities, int(imageIndex), true, false)

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
				//fmt.Printf("FPS: %f | milliseconds per 60 frames: %d\n", 60.0/float32(time.Now().UnixMilli()-timer)*1000, time.Now().UnixMilli()-timer)
				//timer = time.Now().UnixMilli()
			}

			time.Sleep(500 * time.Microsecond) // suptid pogrommer >:(

		}
	}
}
