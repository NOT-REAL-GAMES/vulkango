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
	Image        vk.Image        // Texture image
	ImageView    vk.ImageView    // Texture image view
	ImageMemory  vk.DeviceMemory // Texture memory
	Sampler      vk.Sampler      // Texture sampler
	Width        uint32          // Texture width
	Height       uint32          // Texture height
	TextureIndex uint32          // Index in global bindless texture array
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

// Text component holds text rendering data.
// For rendering debug overlays, UI labels, etc.
type Text struct {
	Content  string    // The text string to display
	X, Y     float32   // Screen position in pixels
	FontSize float32   // Font size in pixels
	Color    [4]float32 // Text color (RGBA, 0-1 range)
	Visible  bool      // Whether the text is visible
}

// NewText creates a default Text component with white color
func NewText(content string, x, y, fontSize float32) *Text {
	return &Text{
		Content:  content,
		X:        x,
		Y:        y,
		FontSize: fontSize,
		Color:    [4]float32{1.0, 1.0, 1.0, 1.0}, // White
		Visible:  true,
	}
}

// UIButtonState represents the current state of a button
type UIButtonState int

const (
	UIButtonNormal UIButtonState = iota
	UIButtonHovered
	UIButtonPressed
)

// UIButton component for interactive UI buttons.
// Can be used in editor and in executable/game UIs.
type UIButton struct {
	// Bounds in screen space (pixels)
	X, Y          float32
	Width, Height float32

	// Current state
	State UIButtonState

	// Was the button pressed (for detecting release)
	WasPressed bool

	// Colors for each state (RGBA, 0-1 range)
	ColorNormal  [4]float32
	ColorHovered [4]float32
	ColorPressed [4]float32

	// Callback function to execute when button is clicked (pressed then released while hovering)
	OnClick func()

	// Whether the button is enabled
	Enabled bool

	// Optional label
	Label string

	// Label text colors for each state (RGBA, 0-1 range)
	LabelColor   [4]float32 // Normal state
	LabelHovered [4]float32 // Hovered state
	LabelPressed [4]float32 // Pressed state
}

// NewUIButton creates a default UI button with sensible defaults
func NewUIButton(x, y, width, height float32, onClick func()) *UIButton {
	return &UIButton{
		X:            x,
		Y:            y,
		Width:        width,
		Height:       height,
		State:        UIButtonNormal,
		WasPressed:   false,
		ColorNormal:  [4]float32{0.3, 0.3, 0.3, 1.0}, // Dark gray
		ColorHovered: [4]float32{0.5, 0.5, 0.5, 1.0}, // Light gray
		ColorPressed: [4]float32{0.2, 0.4, 0.8, 1.0}, // Blue
		OnClick:      onClick,
		Enabled:      true,
		Label:        "",
		LabelColor:   [4]float32{1.0, 1.0, 1.0, 1.0}, // White
		LabelHovered: [4]float32{1.0, 1.0, 1.0, 1.0}, // White
		LabelPressed: [4]float32{1.0, 1.0, 1.0, 1.0}, // White
	}
}

// ScreenSpace component marks entities that should remain in screen space,
// unaffected by camera panning and zooming. Useful for UI elements, HUD,
// and other overlays that should stay fixed on screen.
type ScreenSpace struct {
	// If true, entity ignores camera transforms completely (stays in screen space)
	// If false, entity uses world space (affected by camera pan/zoom)
	Enabled bool
}

// NewScreenSpace creates a ScreenSpace component with enabled=true by default
func NewScreenSpace() *ScreenSpace {
	return &ScreenSpace{
		Enabled: true,
	}
}
