package ecs

import (
	vk "github.com/NOT-REAL-GAMES/vulkango"
)

// Transform component holds 2D transformation data for a layer.
// Position, scale, and rotation.
type Transform struct {
	X, Y             float32 // Position in pixels
	ScaleX, ScaleY   float32 // Scale factors (1.0 = normal)
	Rotation         float32 // Rotation in radians
	ZIndex           int     // Layer depth order (higher = closer to camera)
	Width, Height    uint32  // Size of the layer content
}

// NewTransform creates a default Transform component.
func NewTransform() *Transform {
	return &Transform{
		X:        0,
		Y:        0,
		ScaleX:   1.0,
		ScaleY:   1.0,
		Rotation: 0.0,
		ZIndex:   0,
		Width:    0,
		Height:   0,
	}
}

// RenderTarget component holds Vulkan rendering resources for a layer.
// Each layer can render to its own framebuffer/image.
type RenderTarget struct {
	Image       vk.Image       // The image to render to
	ImageView   vk.ImageView   // View of the image for rendering
	ImageMemory vk.DeviceMemory // Memory backing the image
	Format      vk.Format      // Image format (RGBA8, etc.)
	Width       uint32         // Image width
	Height      uint32         // Image height
}

// VulkanPipeline component holds the graphics pipeline for a layer.
// Each layer can have its own shaders and pipeline state.
type VulkanPipeline struct {
	Pipeline            vk.Pipeline            // The graphics pipeline
	PipelineLayout      vk.PipelineLayout      // Pipeline layout
	DescriptorPool      vk.DescriptorPool      // Descriptor pool for this layer
	DescriptorSet       vk.DescriptorSet       // Descriptor set (textures, uniforms)
	DescriptorSetLayout vk.DescriptorSetLayout // Descriptor set layout
	CompositeDescriptorSet vk.DescriptorSet    // For sampling layer framebuffer in composite pass
}

// BlendMode component controls how a layer is composited with others.
type BlendMode struct {
	Mode    BlendModeType // The blending mode (normal, multiply, add, etc.)
	Opacity float32       // Layer opacity (0.0 = transparent, 1.0 = opaque)
	Visible bool          // Whether the layer is visible
}

// BlendModeType defines different compositing modes.
type BlendModeType int

const (
	BlendNormal BlendModeType = iota
	BlendMultiply
	BlendScreen
	BlendOverlay
	BlendAdd
	BlendSubtract
)

// NewBlendMode creates a default BlendMode component.
func NewBlendMode() *BlendMode {
	return &BlendMode{
		Mode:    BlendNormal,
		Opacity: 1.0,
		Visible: true,
	}
}

// TextureData component holds texture information for a layer.
// This is the actual image data that gets rendered.
type TextureData struct {
	Image       vk.Image          // Texture image
	ImageView   vk.ImageView      // Texture image view
	ImageMemory vk.DeviceMemory   // Texture memory
	Sampler     vk.Sampler        // Texture sampler
	Width       uint32            // Texture width
	Height      uint32            // Texture height
}

// BufferData component holds vertex and index buffers for a layer.
// Used for rendering geometry (quads, paths, etc.)
type BufferData struct {
	VertexBuffer       vk.Buffer       // Vertex buffer
	VertexBufferMemory vk.DeviceMemory // Vertex buffer memory
	IndexBuffer        vk.Buffer       // Index buffer
	IndexBufferMemory  vk.DeviceMemory // Index buffer memory
	IndexCount         uint32          // Number of indices
	VertexCount        uint32          // Number of vertices
}
