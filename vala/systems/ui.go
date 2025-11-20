package systems

import (
	"unsafe"

	vk "github.com/NOT-REAL-GAMES/vulkango"
	"github.com/NOT-REAL-GAMES/vulkango/vala/canvas"
	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
	"github.com/NOT-REAL-GAMES/vulkango/vala/font"
)

// UIRenderContext holds rendering resources needed by the UI system.
type UIRenderContext struct {
	Device               vk.Device
	CommandBuffer        vk.CommandBuffer
	UIRectPipeline       vk.Pipeline
	UIRectPipelineLayout vk.PipelineLayout
	BrushVertexBuffer    vk.Buffer
	SwapExtent           vk.Extent2D
	TextRenderer         *TextRenderer
}

// UpdateUIButtons updates all UI button states based on mouse position and button state.
// This should be called once per frame BEFORE rendering.
// mouseX, mouseY: current mouse position in screen space (pixels)
// mouseButtonDown: true if left mouse button is currently pressed
// cameraX, cameraY, cameraZoom: camera transform parameters
// swapWidth, swapHeight: window dimensions for coordinate conversion
func UpdateUIButtons(world *ecs.World, mouseX, mouseY float32, mouseButtonDown bool, buttonJustPressed bool, cameraX, cameraY, cameraZoom float32, swapWidth, swapHeight uint32) {
	buttons := world.QueryUIButtons()

	for _, entity := range buttons {
		button := world.GetUIButton(entity)
		if !button.Enabled {
			continue
		}

		// Check if button is in screen space (doesn't move with camera)
		screenSpace := world.GetScreenSpace(entity)
		isScreenSpace := screenSpace != nil && screenSpace.Enabled

		var testX, testY float32
		if isScreenSpace {
			// Screen-space button: use direct mouse coordinates
			testX = mouseX
			testY = mouseY
		} else {
			// World-space button: apply inverse camera transform to mouse position
			// Convert mouse pixels to clip space (-1 to 1)
			mouseClipX := (mouseX/float32(swapWidth))*2.0 - 1.0
			mouseClipY := (mouseY/float32(swapHeight))*2.0 - 1.0

			// Apply inverse camera transform: pos = pos / zoom + offset
			// (This undoes the camera transform applied during composite rendering)
			worldClipX := mouseClipX/cameraZoom + cameraX
			worldClipY := mouseClipY/cameraZoom + cameraY

			// Convert back to pixel coordinates for comparison with button bounds
			testX = ((worldClipX + 1.0) / 2.0) * float32(swapWidth)
			testY = ((worldClipY + 1.0) / 2.0) * float32(swapHeight)
		}

		// Check if mouse is hovering over button
		isHovering := testX >= button.X && testX <= button.X+button.Width &&
			testY >= button.Y && testY <= button.Y+button.Height

		// Update button state based on hover and mouse button
		if isHovering {
			if mouseButtonDown {
				// Only set WasPressed if button was just pressed this frame while hovering
				if buttonJustPressed {
					button.WasPressed = true
				}
				button.State = ecs.UIButtonPressed
			} else {
				// Mouse released while hovering - trigger onClick!
				if button.WasPressed && button.OnClick != nil {
					button.OnClick()
				}
				button.State = ecs.UIButtonHovered
				button.WasPressed = false
			}
		} else {
			button.State = ecs.UIButtonNormal
			// Reset WasPressed if mouse moved away
			if !mouseButtonDown {
				button.WasPressed = false
			}
		}
	}
}

// RenderUIButtons renders all UI buttons to the command buffer.
// This uses the UI rectangle pipeline to draw colored rectangles for buttons.
// DEPRECATED: Use RenderUIButtonBases and RenderUIButtonLabels separately for multi-layer rendering.
func RenderUIButtons(world *ecs.World, ctx *UIRenderContext, c canvas.Canvas) {
	RenderUIButtonBases(world, ctx)
	if ctx.TextRenderer != nil {
		RenderUIButtonLabels(world, ctx)
	}
}

// RenderUIButtonBases renders only the button base rectangles (no text).
// This should be called on the button base layer.
func RenderUIButtonBases(world *ecs.World, ctx *UIRenderContext) {
	buttons := world.QueryUIButtons()

	// Bind UI rectangle pipeline for drawing UI buttons
	ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, ctx.UIRectPipeline)

	// Set viewport and scissor
	ctx.CommandBuffer.SetViewport(0, []vk.Viewport{
		{
			X:        0,
			Y:        0,
			Width:    float32(ctx.SwapExtent.Width),
			Height:   float32(ctx.SwapExtent.Height),
			MinDepth: 0.0,
			MaxDepth: 1.0,
		},
	})

	ctx.CommandBuffer.SetScissor(0, []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: ctx.SwapExtent,
		},
	})

	// Bind vertex buffer
	ctx.CommandBuffer.BindVertexBuffers(0, []vk.Buffer{ctx.BrushVertexBuffer}, []uint64{0})

	renderedCount := 0
	for _, entity := range buttons {
		button := world.GetUIButton(entity)

		// Skip disabled buttons
		if !button.Enabled {
			//fmt.Printf("[UI DEBUG] Skipping disabled button: Entity=%d, Label='%s'\n", entity, button.Label)
			continue
		}

		renderedCount++

		// Select color based on button state
		var color [4]float32
		switch button.State {
		case ecs.UIButtonPressed:
			color = button.ColorPressed
		case ecs.UIButtonHovered:
			color = button.ColorHovered
		default:
			color = button.ColorNormal
		}

		// Draw button rectangle using UI rectangle pipeline
		// UI rect shader expects: posX, posY, width, height, colorR, colorG, colorB, colorA
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

		pushData := UIRectPushConstants{
			PosX:   button.X,
			PosY:   button.Y,
			Width:  button.Width,
			Height: button.Height,
			ColorR: color[0],
			ColorG: color[1],
			ColorB: color[2],
			ColorA: color[3],
		}

		// Push constants for button
		ctx.CommandBuffer.CmdPushConstants(
			ctx.UIRectPipelineLayout,
			vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
			0,
			32, // Size of UIRectPushConstants struct (8 floats = 32 bytes)
			unsafe.Pointer(&pushData),
		)

		// Draw 6 vertices (2 triangles = 1 quad)
		ctx.CommandBuffer.Draw(6, 1, 0, 0)
	}

	// Debug: Show how many buttons were actually rendered vs queried
	if renderedCount != len(buttons) {
		//fmt.Printf("[UI DEBUG] Rendered %d/%d buttons (some were disabled)\n", renderedCount, len(buttons))
	}
}

// RenderUIButtonLabels renders only the button text labels (no rectangles).
// This should be called on the button text layer (above the base layer).
func RenderUIButtonLabels(world *ecs.World, ctx *UIRenderContext) {
	renderButtonLabels(world, ctx)
}

// renderButtonLabels renders text labels on buttons
func renderButtonLabels(world *ecs.World, ctx *UIRenderContext) {
	buttons := world.QueryUIButtons()
	fontSize := float32(24.0)

	// First pass: Collect all text quads for all buttons
	type buttonTextData struct {
		vertices []TextVertex
		indices  []uint16
		color    [4]float32
	}
	textData := make([]buttonTextData, 0, len(buttons))
	totalVertices := 0
	totalIndices := 0

	for _, entity := range buttons {
		button := world.GetUIButton(entity)

		if button.Label == "" || !button.Enabled {
			continue
		}

		// Calculate text metrics to center it on the button
		textWidth := measureText(button.Label, ctx.TextRenderer.Atlas, fontSize)
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

		vertices, indices := GenerateTextQuads(textComponent, ctx.TextRenderer.Atlas)

		if len(vertices) == 0 {
			continue
		}

		// Adjust indices for accumulated vertex offset
		adjustedIndices := make([]uint16, len(indices))
		for i, idx := range indices {
			adjustedIndices[i] = idx + uint16(totalVertices)
		}

		textData = append(textData, buttonTextData{
			vertices: vertices,
			indices:  adjustedIndices,
			color:    textComponent.Color,
		})

		totalVertices += len(vertices)
		totalIndices += len(adjustedIndices)
	}

	if len(textData) == 0 {
		return // No text to render
	}

	// Calculate buffer sizes
	vertexBufferSize := uint64(totalVertices * int(unsafe.Sizeof(TextVertex{})))
	indexBufferSize := uint64(totalIndices * 2) // uint16 = 2 bytes

	// Map staging buffers (CPU-writable)
	vertexPtr, err := ctx.Device.MapMemory(ctx.TextRenderer.StagingVertexMemory, 0, vertexBufferSize)
	if err != nil {
		panic(err)
	}
	indexPtr, err := ctx.Device.MapMemory(ctx.TextRenderer.StagingIndexMemory, 0, indexBufferSize)
	if err != nil {
		panic(err)
	}

	// Copy all vertex and index data to staging buffers
	vertexOffset := 0
	indexOffset := 0
	for _, data := range textData {
		// Copy vertices
		vertexSize := len(data.vertices) * int(unsafe.Sizeof(TextVertex{}))
		copy(unsafe.Slice((*TextVertex)(vertexPtr), totalVertices)[vertexOffset/int(unsafe.Sizeof(TextVertex{})):], data.vertices)
		vertexOffset += vertexSize

		// Copy indices
		indexSize := len(data.indices) * 2 // uint16 = 2 bytes
		copy(unsafe.Slice((*uint16)(indexPtr), totalIndices)[indexOffset/2:], data.indices)
		indexOffset += indexSize
	}

	// Unmap staging buffers
	ctx.Device.UnmapMemory(ctx.TextRenderer.StagingVertexMemory)
	ctx.Device.UnmapMemory(ctx.TextRenderer.StagingIndexMemory)

	// Copy from staging buffers to main buffers (this IS allowed during rendering!)
	vertexDataSize := uint64(totalVertices * int(unsafe.Sizeof(TextVertex{})))
	indexDataSize := uint64(totalIndices * 2)

	ctx.CommandBuffer.CmdCopyBuffer(
		ctx.TextRenderer.StagingVertexBuffer,
		ctx.TextRenderer.VertexBuffer,
		[]vk.BufferCopy{{
			SrcOffset: 0,
			DstOffset: 0,
			Size:      vertexDataSize,
		}},
	)

	ctx.CommandBuffer.CmdCopyBuffer(
		ctx.TextRenderer.StagingIndexBuffer,
		ctx.TextRenderer.IndexBuffer,
		[]vk.BufferCopy{{
			SrcOffset: 0,
			DstOffset: 0,
			Size:      indexDataSize,
		}},
	)

	// Bind text pipeline
	ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, ctx.TextRenderer.Pipeline)

	// Bind descriptor set (SDF atlas texture)
	ctx.CommandBuffer.BindDescriptorSets(
		vk.PIPELINE_BIND_POINT_GRAPHICS,
		ctx.TextRenderer.PipelineLayout,
		0,
		[]vk.DescriptorSet{ctx.TextRenderer.DescriptorSet},
		nil,
	)

	// Bind text vertex and index buffers
	ctx.CommandBuffer.BindVertexBuffers(0, []vk.Buffer{ctx.TextRenderer.VertexBuffer}, []uint64{0})
	ctx.CommandBuffer.BindIndexBuffer(ctx.TextRenderer.IndexBuffer, 0, vk.INDEX_TYPE_UINT16)

	// Render all text with individual push constants for each button
	indexStart := 0
	for _, data := range textData {
		// Push constants for text rendering
		type TextPushConstants struct {
			ScreenWidth  float32
			ScreenHeight float32
			_            [2]float32 // Padding to align vec4 to 16-byte boundary
			ColorR       float32
			ColorG       float32
			ColorB       float32
			ColorA       float32
		}

		pushData := TextPushConstants{
			ScreenWidth:  float32(ctx.SwapExtent.Width),
			ScreenHeight: float32(ctx.SwapExtent.Height),
			ColorR:       data.color[0],
			ColorG:       data.color[1],
			ColorB:       data.color[2],
			ColorA:       data.color[3],
		}

		ctx.CommandBuffer.CmdPushConstants(
			ctx.TextRenderer.PipelineLayout,
			vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
			0,
			32, // 8 floats * 4 bytes = 32 bytes (was 24, now includes padding)
			unsafe.Pointer(&pushData),
		)

		ctx.CommandBuffer.DrawIndexed(uint32(len(data.indices)), 1, uint32(indexStart), 0, 0)
		indexStart += len(data.indices)
	}
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
