package systems

import (
	"math"
	"unsafe"

	vk "github.com/NOT-REAL-GAMES/vulkango"
	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
)

// RenderContext holds the rendering state needed by render systems.
// This is passed to render systems each frame.
type RenderContext struct {
	CommandBuffer       vk.CommandBuffer
	SwapExtent          vk.Extent2D
	VertexBuffer        vk.Buffer
	IndexBuffer         vk.Buffer
	IndexCount          uint32
	Device              vk.Device
	DescriptorPool      vk.DescriptorPool
	DescriptorSetLayout vk.DescriptorSetLayout
}

// ensureDescriptorSet creates a descriptor set for a layer if it doesn't have one.
// This allows layers to dynamically get descriptor sets based on their TextureData.
func ensureDescriptorSet(world *ecs.World, entity ecs.Entity, ctx *RenderContext) {
	pipeline := world.GetVulkanPipeline(entity)
	textureData := world.GetTextureData(entity)

	// If descriptor set already exists, nothing to do
	if pipeline.DescriptorSet != (vk.DescriptorSet{}) {
		return
	}

	// Allocate a new descriptor set
	descriptorSets, err := ctx.Device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
		DescriptorPool: ctx.DescriptorPool,
		SetLayouts:     []vk.DescriptorSetLayout{ctx.DescriptorSetLayout},
	})
	if err != nil {
		panic(err) // In production, handle this more gracefully
	}

	descriptorSet := descriptorSets[0]

	// Update descriptor set with the layer's texture data
	ctx.Device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
		{
			DstSet:          descriptorSet,
			DstBinding:      0,
			DstArrayElement: 0,
			DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
			ImageInfo: []vk.DescriptorImageInfo{
				{
					Sampler:     textureData.Sampler,
					ImageView:   textureData.ImageView,
					ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
				},
			},
		},
	})

	// Update the pipeline component with the new descriptor set
	pipeline.DescriptorSet = descriptorSet
}

// RenderLayers renders all visible layers to the command buffer.
// This system queries for renderable entities and draws them in order.
func RenderLayers(world *ecs.World, ctx *RenderContext) {
	// Query all renderable entities (Transform + VulkanPipeline + TextureData)
	renderables := world.QueryRenderables()

	// TODO: Sort by layer order / z-index
	// For now, just render in entity order

	for _, entity := range renderables {
		transform := world.GetTransform(entity)
		pipeline := world.GetVulkanPipeline(entity)
		blend := world.GetBlendMode(entity)

		// Skip invisible layers
		if blend != nil && !blend.Visible {
			continue
		}

		// Bind pipeline for this layer
		ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, pipeline.Pipeline)

		// Bind descriptor sets (textures, uniforms)
		ctx.CommandBuffer.BindDescriptorSets(
			vk.PIPELINE_BIND_POINT_GRAPHICS,
			pipeline.PipelineLayout,
			0,
			[]vk.DescriptorSet{pipeline.DescriptorSet},
			nil,
		)

		// Set viewport (could vary per layer in the future)
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

		// Set scissor
		ctx.CommandBuffer.SetScissor(0, []vk.Rect2D{
			{
				Offset: vk.Offset2D{X: 0, Y: 0},
				Extent: ctx.SwapExtent,
			},
		})

		// Push transform constants (offset, scale, opacity, and depth)
		type PushConstants struct {
			OffsetX float32
			OffsetY float32
			Scale   float32
			Opacity float32
			Depth   float32
		}
		opacity := float32(1.0)
		if blend != nil {
			opacity = blend.Opacity
		}
		// Convert ZIndex to depth: higher ZIndex = closer to camera (lower Z value)
		// ZIndex 0 = 0.5, positive moves closer (0.0), negative moves farther (1.0)
		depth := 0.5 - float32(transform.ZIndex)*0.01

		pushData := PushConstants{
			OffsetX: transform.X,
			OffsetY: transform.Y,
			Scale:   transform.ScaleX, // Using ScaleX for uniform scaling
			Opacity: opacity,
			Depth:   depth,
		}
		ctx.CommandBuffer.CmdPushConstants(
			pipeline.PipelineLayout,
			vk.SHADER_STAGE_VERTEX_BIT,
			0,
			20, // size in bytes
			unsafe.Pointer(&pushData),
		)
		// Bind vertex/index buffers
		ctx.CommandBuffer.BindVertexBuffers(0, []vk.Buffer{ctx.VertexBuffer}, []uint64{0})
		ctx.CommandBuffer.BindIndexBuffer(ctx.IndexBuffer, 0, vk.INDEX_TYPE_UINT16)

		// Draw!
		ctx.CommandBuffer.DrawIndexed(ctx.IndexCount, 1, 0, 0, 0)

	}
}

// BeginFrame prepares the command buffer and swapchain image for rendering.
// This handles image layout transitions and begins dynamic rendering.
func BeginFrame(cmd vk.CommandBuffer, swapImage vk.Image, swapImageView vk.ImageView, swapExtent vk.Extent2D) {
	// Transition swapchain image from UNDEFINED to COLOR_ATTACHMENT_OPTIMAL
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
				Image:               swapImage,
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

	// Begin dynamic rendering
	cmd.BeginRendering(&vk.RenderingInfo{
		RenderArea: vk.Rect2D{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: swapExtent,
		},
		LayerCount: 1,
		ColorAttachments: []vk.RenderingAttachmentInfo{
			{
				ImageView:   swapImageView,
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
}

// EndFrame finishes rendering and transitions the image for presentation.
func EndFrame(cmd vk.CommandBuffer, swapImage vk.Image) {
	// End rendering
	cmd.EndRendering()

	// Transition swapchain image from COLOR_ATTACHMENT_OPTIMAL to PRESENT_SRC_KHR
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
				Image:               swapImage,
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

// zIndexToDepth converts ZIndex to depth using IEEE 754 bit manipulation.
// This packs the ZIndex directly into the float32 mantissa bits for maximum precision.
// Supports range: -8,388,608 to +8,388,607 (2^23 values) with perfect separation.
func zIndexToDepth(zindex int) float32 {
	// Flip sign bit so negative values map to lower depths (back)
	// This ensures: negative ZIndex → [0, 0.5), positive ZIndex → [0.5, 1.0)
	u := uint32(zindex) ^ 0x80000000

	// Take top 23 bits for float32 mantissa
	mantissa := u >> 9

	// Construct IEEE 754 float in range [1.0, 2.0) then subtract 1.0 for [0.0, 1.0)
	// Exponent = 127 (bias for 2^0 = 1.0)
	bits := (127 << 23) | mantissa
	depth := math.Float32frombits(bits) - 1.0

	return depth
}

// RenderLayerContent renders a single layer's content to its own framebuffer.
// This is called during Pass 1 (layer rendering).
func RenderLayerContent(world *ecs.World, ctx *RenderContext, entity ecs.Entity) {
	transform := world.GetTransform(entity)
	pipeline := world.GetVulkanPipeline(entity)
	blend := world.GetBlendMode(entity)

	// Skip invisible layers
	if blend != nil && !blend.Visible {
		return
	}

	// Ensure this layer has a descriptor set based on its TextureData
	ensureDescriptorSet(world, entity, ctx)

	// Bind layer's content rendering pipeline
	ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, pipeline.Pipeline)

	// Bind descriptor sets (layer's source texture)
	ctx.CommandBuffer.BindDescriptorSets(
		vk.PIPELINE_BIND_POINT_GRAPHICS,
		pipeline.PipelineLayout,
		0,
		[]vk.DescriptorSet{pipeline.DescriptorSet},
		nil,
	)

	// Push transform constants (offset, scale, opacity, depth)
	type PushConstants struct {
		OffsetX float32
		OffsetY float32
		Scale   float32
		Opacity float32
		Depth   float32
	}
	opacity := float32(1.0)
	if blend != nil {
		opacity = blend.Opacity
	}

	// IEEE 754 bit-hack: Map ZIndex directly to depth using mantissa bits
	// Supports 2^23 = 8,388,608 distinct layers with perfect precision
	// Higher ZIndex = closer to camera (lower depth value)
	depth := zIndexToDepth(transform.ZIndex)

	pushData := PushConstants{
		OffsetX: transform.X,
		OffsetY: transform.Y,
		Scale:   transform.ScaleX,
		Opacity: opacity,
		Depth:   depth,
	}
	ctx.CommandBuffer.CmdPushConstants(
		pipeline.PipelineLayout,
		vk.SHADER_STAGE_VERTEX_BIT,
		0,
		20,
		unsafe.Pointer(&pushData),
	)

	// Set viewport
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

	// Set scissor
	ctx.CommandBuffer.SetScissor(0, []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: ctx.SwapExtent,
		},
	})

	// Bind vertex/index buffers
	ctx.CommandBuffer.BindVertexBuffers(0, []vk.Buffer{ctx.VertexBuffer}, []uint64{0})
	ctx.CommandBuffer.BindIndexBuffer(ctx.IndexBuffer, 0, vk.INDEX_TYPE_UINT16)

	// Draw!
	ctx.CommandBuffer.DrawIndexed(ctx.IndexCount, 1, 0, 0, 0)
}

// CompositeContext holds data needed for the composite pass.
type CompositeContext struct {
	CommandBuffer            vk.CommandBuffer
	CompositePipeline        vk.Pipeline
	CompositePipelineLayout  vk.PipelineLayout
	SwapExtent               vk.Extent2D
	BindlessDescriptorSet    vk.DescriptorSet // Global bindless texture array
}

// CompositeLayer draws a single layer's framebuffer to the swapchain using BINDLESS textures.
// This is called during Pass 2 (compositing).
func CompositeLayer(ctx *CompositeContext, textureIndex uint32, opacity, offsetX, offsetY, scale float32) {
	// Bind composite pipeline
	ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, ctx.CompositePipeline)

	// Push constants for layer transform, opacity, and texture index
	type PushConstants struct {
		Opacity      float32
		TextureIndex uint32
		OffsetX      float32
		OffsetY      float32
		Scale        float32
	}
	pushData := PushConstants{
		Opacity:      opacity,
		TextureIndex: textureIndex,
		OffsetX:      offsetX,
		OffsetY:      offsetY,
		Scale:        scale,
	}
	ctx.CommandBuffer.CmdPushConstants(
		ctx.CompositePipelineLayout,
		vk.SHADER_STAGE_VERTEX_BIT|vk.SHADER_STAGE_FRAGMENT_BIT,
		0,
		20, // opacity(4) + textureIndex(4) + offsetX(4) + offsetY(4) + scale(4) = 20 bytes
		unsafe.Pointer(&pushData),
	)

	// Set viewport
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

	// Set scissor
	ctx.CommandBuffer.SetScissor(0, []vk.Rect2D{
		{
			Offset: vk.Offset2D{X: 0, Y: 0},
			Extent: ctx.SwapExtent,
		},
	})

	// Draw fullscreen quad (no vertex buffer needed)
	ctx.CommandBuffer.Draw(6, 1, 0, 0)
}
