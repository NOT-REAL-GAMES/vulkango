package systems

import (
	"fmt"
	"unsafe"

	vk "github.com/NOT-REAL-GAMES/vulkango"
	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
	"github.com/NOT-REAL-GAMES/vulkango/vala/font"
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
	// Per-frame, per-usage staging buffers for dynamic text updates (CPU-visible)
	// 2D array: [bufferSetIndex][frameIndex]
	// bufferSetIndex 0 = button labels, 1 = text entities
	// This prevents buffer conflicts when multiple render passes use text in same frame
	StagingVertexBuffers  [][]vk.Buffer
	StagingVertexMemories [][]vk.DeviceMemory
	StagingIndexBuffers   [][]vk.Buffer
	StagingIndexMemories  [][]vk.DeviceMemory
	MaxChars              int // Maximum characters that can be rendered
}

// TextVertex represents a vertex for text rendering
type TextVertex struct {
	PosX, PosY float32 // Screen-space position
	U, V       float32 // UV coordinates in atlas
}

// RenderText renders text to the screen
// This should be called during rendering passes
// bufferSetIndex: which buffer set to use (0=button labels, 1=text entities) to avoid conflicts
// frameIndex: which frame in flight (0 to len(swapImages)-1)
// includeTextEntities: if true, renders Text component entities
// includeButtonLabels: if true, renders UIButton labels
func RenderText(world *ecs.World, renderer *TextRenderer, cmd vk.CommandBuffer, screenWidth, screenHeight uint32, device vk.Device, bufferSetIndex int, frameIndex int, includeTextEntities bool, includeButtonLabels bool) {
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

	// Collect all vertices and indices from requested sources
	allVertices := make([]TextVertex, 0)
	allIndices := make([]uint16, 0)
	indexOffset := uint16(0)
	textEntityCount := 0
	buttonLabelCount := 0

	// Add text entities if requested
	if includeTextEntities {
		textEntities := world.QueryTexts()
		for _, entity := range textEntities {
			text := world.GetText(entity)
			if text == nil || !text.Visible {
				continue
			}

			// Generate quads for this text
			vertices, indices := GenerateTextQuads(text, renderer.Atlas)

			// Debug: Print what we're rendering (only first time to avoid spam)
			//fmt.Printf("[TEXT] Entity %d: '%s' -> %d vertices, %d indices\n", entity, text.Content, len(vertices), len(indices))

			// Offset indices for concatenation
			for _, idx := range indices {
				allIndices = append(allIndices, idx+indexOffset)
			}
			allVertices = append(allVertices, vertices...)
			indexOffset += uint16(len(vertices))
			textEntityCount++
		}
	}

	// Then, add button labels (if requested)
	if includeButtonLabels {
		buttonEntities := world.QueryUIButtons()
		fontSize := float32(24.0)
		for _, entity := range buttonEntities {
			button := world.GetUIButton(entity)
			if button == nil || button.Label == "" || !button.Enabled {
				continue
			}

			// Calculate text metrics to center it on the button
			textWidth := measureText(button.Label, renderer.Atlas, fontSize)
			textX := button.X + (button.Width-textWidth)/2
			textY := button.Y + (button.Height-fontSize)/2 + fontSize*0.7

			// Select label color based on button state
			var labelColor [4]float32
			switch button.State {
			case ecs.UIButtonPressed:
				labelColor = button.LabelPressed
			case ecs.UIButtonHovered:
				labelColor = button.LabelHovered
			default:
				labelColor = button.LabelColor
			}

			textComponent := &ecs.Text{
				Content:  button.Label,
				X:        textX,
				Y:        textY,
				FontSize: fontSize,
				Color:    labelColor,
			}

			vertices, indices := GenerateTextQuads(textComponent, renderer.Atlas)

			//fmt.Printf("[TEXT] Button %d label: '%s' -> %d vertices, %d indices\n", entity, button.Label, len(vertices), len(indices))

			// Offset indices for concatenation
			for _, idx := range indices {
				allIndices = append(allIndices, idx+indexOffset)
			}
			allVertices = append(allVertices, vertices...)
			indexOffset += uint16(len(vertices))
			buttonLabelCount++
		}
	}

	if len(allVertices) == 0 {
		return // No text to render
	}

	// Debug: Show total
	//fmt.Printf("[TEXT] Total: %d text entities, %d button labels, %d vertices, %d indices\n", textEntityCount, buttonLabelCount, len(allVertices), len(allIndices))

	// Debug: Print first few vertices to check coordinates
	if len(allVertices) > 0 {
		//fmt.Printf("[TEXT] First vertex: Pos=(%.1f, %.1f), UV=(%.3f, %.3f)\n",
		//	allVertices[0].PosX, allVertices[0].PosY, allVertices[0].U, allVertices[0].V)
		if len(allVertices) > 16 {
			//	fmt.Printf("[TEXT] Vertex 16 (char 5): Pos=(%.1f, %.1f)\n",
			//		allVertices[16].PosX, allVertices[16].PosY)
		}
	}

	// Use per-frame, per-buffer-set staging buffers to avoid conflicts
	stagingVertexBuffer := renderer.StagingVertexBuffers[bufferSetIndex][frameIndex]
	stagingVertexMemory := renderer.StagingVertexMemories[bufferSetIndex][frameIndex]

	// Update staging vertex buffer (HOST_VISIBLE, render directly from it)
	vertexData, err := device.MapMemory(stagingVertexMemory, 0, uint64(len(allVertices)*16))
	if err != nil {
		//fmt.Printf("[TEXT] ERROR: Failed to map vertex memory: %v\n", err)
		return // Failed to map memory
	}
	// Copy vertex data as bytes
	vertexSlice := (*[1 << 30]TextVertex)(vertexData)[:len(allVertices)]
	copy(vertexSlice, allVertices)

	// Debug: Verify the copy worked
	if len(vertexSlice) > 16 {
		//fmt.Printf("[TEXT] Frame %d - After copy - Vertex 0: Pos=(%.1f, %.1f), Vertex 16: Pos=(%.1f, %.1f)\n",
		//	frameIndex, vertexSlice[0].PosX, vertexSlice[0].PosY,
		//	vertexSlice[16].PosX, vertexSlice[16].PosY)
	}

	device.UnmapMemory(stagingVertexMemory)

	// Use per-frame, per-buffer-set index buffer
	stagingIndexBuffer := renderer.StagingIndexBuffers[bufferSetIndex][frameIndex]
	stagingIndexMemory := renderer.StagingIndexMemories[bufferSetIndex][frameIndex]

	// Update staging index buffer
	indexData, err := device.MapMemory(stagingIndexMemory, 0, uint64(len(allIndices)*2))
	if err != nil {
		fmt.Printf("[TEXT] ERROR: Failed to map index memory: %v\n", err)
		return // Failed to map memory
	}
	indexSlice := (*[1 << 30]uint16)(indexData)[:len(allIndices)]
	copy(indexSlice, allIndices)

	// Debug: Check indices
	//fmt.Printf("[TEXT] Frame %d - First 10 indices: %v\n", frameIndex, indexSlice[:10])

	device.UnmapMemory(stagingIndexMemory)

	// Bind per-frame staging buffers for rendering (no more flickering!)
	// Each frame has its own staging buffers, so no synchronization issues
	cmd.BindVertexBuffers(0, []vk.Buffer{stagingVertexBuffer}, []uint64{0})
	cmd.BindIndexBuffer(stagingIndexBuffer, 0, vk.INDEX_TYPE_UINT16)

	//fmt.Printf("[TEXT] BufferSet %d, Frame %d - Using separate staging buffers\n", bufferSetIndex, frameIndex)

	// Push constants: screen size + padding + color (std140 alignment)
	// Format: [screenWidth, screenHeight, padding, padding, colorR, colorG, colorB, colorA]
	// std140 requires vec4 to be 16-byte aligned, so we need 8 bytes padding after vec2
	pushConstants := []float32{
		float32(screenWidth),
		float32(screenHeight),
		0.0, // Padding for std140 alignment
		0.0, // Padding for std140 alignment
		1.0, // ColorR (white)
		1.0, // ColorG
		1.0, // ColorB
		1.0, // ColorA
	}
	//fmt.Printf("[TEXT] Push constants: screen=(%d, %d), padding, color=(1,1,1,1)\n", screenWidth, screenHeight)

	// Convert to bytes
	pushConstantsBytes := make([]byte, 32) // 8 floats * 4 bytes (with std140 padding)
	for i, val := range pushConstants {
		bits := *(*uint32)(unsafe.Pointer(&val))
		pushConstantsBytes[i*4+0] = byte(bits)
		pushConstantsBytes[i*4+1] = byte(bits >> 8)
		pushConstantsBytes[i*4+2] = byte(bits >> 16)
		pushConstantsBytes[i*4+3] = byte(bits >> 24)
	}

	cmd.PushConstants(renderer.PipelineLayout, vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT, 0, pushConstantsBytes)

	// Draw all text
	indexCount := uint32(len(allIndices))
	//fmt.Printf("[TEXT] DrawIndexed: indexCount=%d, instanceCount=1\n", indexCount)
	cmd.DrawIndexed(indexCount, 1, 0, 0, 0)
}

// measureText calculates the width of a text string in pixels
func measureText(text string, atlas *font.SDFAtlas, fontSize float32) float32 {
	scale := fontSize / atlas.FontSize
	width := float32(0)

	for _, char := range text {
		charData, exists := atlas.Chars[char]
		if !exists {
			continue
		}
		width += float32(charData.XAdvance) * scale
	}

	return width
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
