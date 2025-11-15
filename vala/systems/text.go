package systems

import (
	"github.com/NOT-REAL-GAMES/vala/ecs"
	"github.com/NOT-REAL-GAMES/vala/font"
	vk "github.com/NOT-REAL-GAMES/vulkango"
)

// TextRenderer holds resources for rendering SDF text
type TextRenderer struct {
	Pipeline       vk.Pipeline
	PipelineLayout vk.PipelineLayout
	DescriptorSet  vk.DescriptorSet
	Atlas          *font.SDFAtlas
	VertexBuffer   vk.Buffer
	VertexMemory   vk.DeviceMemory
	IndexBuffer    vk.Buffer
	IndexMemory    vk.DeviceMemory
	MaxChars       int // Maximum characters that can be rendered
}

// TextVertex represents a vertex for text rendering
type TextVertex struct {
	PosX, PosY float32 // Screen-space position
	U, V       float32 // UV coordinates in atlas
}

// RenderText renders all text entities to the screen
// This should be called during the UI overlay pass
func RenderText(world *ecs.World, renderer *TextRenderer, cmd vk.CommandBuffer, screenWidth, screenHeight uint32) {
	// Bind text pipeline
	cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, renderer.Pipeline)

	// Bind descriptor set (SDF atlas texture)
	cmd.BindDescriptorSets(
		vk.PIPELINE_BIND_POINT_GRAPHICS,
		renderer.PipelineLayout,
		0,
		[]vk.DescriptorSet{renderer.DescriptorSet},
		nil,
	)

	// Set viewport
	cmd.SetViewport(0, []vk.Viewport{
		{
			X:        0,
			Y:        0,
			Width:    float32(screenWidth),
			Height:   float32(screenHeight),
			MinDepth: 0.0,
			MaxDepth: 1.0,
		},
	})

	// Set scissor
	cmd.SetScissor(0, []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: vk.Extent2D{Width: screenWidth, Height: screenHeight},
		},
	})

	// For MVP: We'll implement actual text rendering in the next iteration
	// This is just the framework for now

	// TODO:
	// 1. Query all entities with Text components
	// 2. For each text entity, generate quads for each character
	// 3. Update vertex buffer with quad data
	// 4. Push constants (screen size, color)
	// 5. Draw quads
}

// GenerateTextQuads generates vertex data for rendering a text string
// Returns vertices and indices for the text
func GenerateTextQuads(text *ecs.Text, atlas *font.SDFAtlas) ([]TextVertex, []uint16) {
	vertices := make([]TextVertex, 0, len(text.Content)*4) // 4 vertices per char
	indices := make([]uint16, 0, len(text.Content)*6)      // 6 indices per char (2 triangles)

	cursorX := text.X
	cursorY := text.Y
	vertexIndex := uint16(0)

	// Calculate scale factor (text font size / atlas font size)
	scale := text.FontSize / atlas.FontSize

	for _, char := range text.Content {
		// Get character metrics from atlas
		charData, exists := atlas.Chars[char]
		if !exists {
			// Skip unknown characters
			continue
		}

		// Calculate quad corners (scaled to match requested font size)
		x0 := cursorX + float32(charData.XOffset)*scale
		y0 := cursorY + float32(charData.YOffset)*scale
		x1 := x0 + float32(charData.Width)*scale
		y1 := y0 + float32(charData.Height)*scale

		// Add 4 vertices for this character
		vertices = append(vertices,
			TextVertex{PosX: x0, PosY: y0, U: charData.U0, V: charData.V0}, // Top-left
			TextVertex{PosX: x1, PosY: y0, U: charData.U1, V: charData.V0}, // Top-right
			TextVertex{PosX: x1, PosY: y1, U: charData.U1, V: charData.V1}, // Bottom-right
			TextVertex{PosX: x0, PosY: y1, U: charData.U0, V: charData.V1}, // Bottom-left
		)

		// Add 6 indices for 2 triangles
		indices = append(indices,
			vertexIndex+0, vertexIndex+1, vertexIndex+2, // First triangle
			vertexIndex+0, vertexIndex+2, vertexIndex+3, // Second triangle
		)

		// Advance cursor (also scaled)
		// Apply 0.6x spacing factor to reduce letter spacing
		cursorX += float32(charData.XAdvance) * scale
		vertexIndex += 4
	}

	return vertices, indices
}
