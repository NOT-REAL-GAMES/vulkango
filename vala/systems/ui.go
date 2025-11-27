package systems

import (
	"unsafe"

	vk "github.com/NOT-REAL-GAMES/vulkango"
	"github.com/NOT-REAL-GAMES/vulkango/vala/canvas"
	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
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
// frameIndex specifies which staging buffer set to use (0 to len(swapImages)-1).
func RenderUIButtons(world *ecs.World, ctx *UIRenderContext, c canvas.Canvas, frameIndex int) {
	RenderUIButtonBases(world, ctx)
	if ctx.TextRenderer != nil {
		// Render button labels via RenderText
		RenderUIButtonLabels(world, ctx, frameIndex)
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
			////fmt.Printf("[UI DEBUG] Skipping disabled button: Entity=%d, Label='%s'\n", entity, button.Label)
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
		////fmt.Printf("[UI DEBUG] Rendered %d/%d buttons (some were disabled)\n", renderedCount, len(buttons))
	}
}

// RenderUIButtonLabels renders only the button text labels (no rectangles).
// This should be called on the button text layer (above the base layer).
// frameIndex specifies which staging buffer set to use (0 to len(swapImages)-1).
// NOTE: This calls RenderText with bufferSetIndex=0 for button labels
func RenderUIButtonLabels(world *ecs.World, ctx *UIRenderContext, frameIndex int) {
	// Render only button labels using buffer set 0 (dedicated to button labels)
	const bufferSetButton = 0
	RenderText(world, ctx.TextRenderer, ctx.CommandBuffer, ctx.SwapExtent.Width, ctx.SwapExtent.Height, ctx.Device, bufferSetButton, frameIndex, false, true)
}

// renderButtonLabels is DEPRECATED - removed in favor of RenderText with buffer set index
// All text rendering now goes through RenderText() to avoid buffer conflicts
