# Per-Layer Framebuffer Implementation Plan

## Overview

Transform VALA's rendering from direct-to-swapchain to a multi-pass system:
1. **Layer Pass**: Each layer renders to its own framebuffer
2. **Composite Pass**: Blend all layer framebuffers to swapchain in Z-order

This enables proper layer ordering, transparency, and sets the foundation for inter-layer operations.

---

## Current State Analysis

### What We Have ✅
- Per-layer framebuffer **images created** (layer1Image, layer2Image)
- RenderTarget **components populated** with framebuffer data
- Composite **shaders written** (fullscreen quad)
- ECS architecture with Transform (ZIndex), BlendMode (Opacity), VulkanPipeline, TextureData

### What's Missing ❌
- Composite **pipeline creation**
- **Descriptor sets** for sampling layer framebuffers
- **Sampler** for layer textures
- **Render loop refactor** (two-pass rendering)
- **Image layout transitions** (COLOR_ATTACHMENT → SHADER_READ_ONLY)
- **Synchronization** between passes

---

## Architecture Design

### Rendering Flow

```
┌─────────────────────────────────────────────────────┐
│                 Frame Start                         │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│  PASS 1: Layer Rendering (for each layer)          │
│  ─────────────────────────────────────────────      │
│  1. Bind layer's framebuffer as render target      │
│  2. Clear to transparent (0,0,0,0)                  │
│  3. Render layer content (textured quad)            │
│  4. Transition image: COLOR_ATTACHMENT →            │
│                       SHADER_READ_ONLY              │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│  PASS 2: Composite Pass                             │
│  ─────────────────────────────────────────────      │
│  1. Sort layers by ZIndex (low to high)             │
│  2. Bind swapchain as render target                 │
│  3. Clear to background color                       │
│  4. For each layer in order:                        │
│     a. Bind composite pipeline                      │
│     b. Bind layer's framebuffer texture             │
│     c. Draw fullscreen quad                         │
│     d. Alpha blend based on layer opacity           │
│  5. Transition swapchain: COLOR_ATTACHMENT →        │
│                            PRESENT                  │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│                 Present Frame                       │
└─────────────────────────────────────────────────────┘
```

### Data Structures

#### Per-Layer Resources (already have)
```go
// In ECS components:
type RenderTarget struct {
    Image       vk.Image       // Layer's framebuffer
    ImageView   vk.ImageView   // For rendering TO layer
    ImageMemory vk.DeviceMemory
    Format      vk.Format      // RGBA8
    Width, Height uint32
}
```

#### New Global Resources Needed
```go
// Composite pipeline (reused for all layers)
compositePipeline vk.Pipeline
compositePipelineLayout vk.PipelineLayout

// Descriptor set per layer (for sampling layer texture)
layer1DescriptorSet vk.DescriptorSet
layer2DescriptorSet vk.DescriptorSet

// Shared descriptor pool & layout
compositeDescriptorPool vk.DescriptorPool
compositeDescriptorSetLayout vk.DescriptorSetLayout

// Sampler for layer textures
layerSampler vk.Sampler
```

---

## Implementation Steps

### Step 1: Create Composite Pipeline Resources

**Location**: After main graphics pipeline creation (~line 410)

**Tasks**:
1. Compile composite shaders (vertex + fragment)
2. Create descriptor set layout (binding 0: sampler2D)
3. Create composite pipeline layout
4. Create composite graphics pipeline:
   - No vertex input (quad generated in shader)
   - Blend mode: SRC_ALPHA / ONE_MINUS_SRC_ALPHA
   - Dynamic rendering with swapchain format
5. Create descriptor pool (2 sets for 2 layers)
6. Create sampler (LINEAR filtering)

**Code Sketch**:
```go
// Compile composite shaders
compositeVertModule := compileShader(compositeVertexShader, shaderc.VERTEX_SHADER)
compositeFragModule := compileShader(compositeFragmentShader, shaderc.FRAGMENT_SHADER)
defer device.DestroyShaderModule(compositeVertModule)
defer device.DestroyShaderModule(compositeFragModule)

// Descriptor set layout (sampler2D at binding 0)
compositeDescSetLayout := device.CreateDescriptorSetLayout(...)

// Pipeline layout
compositePipelineLayout := device.CreatePipelineLayout(...)

// Composite pipeline (no vertex input, fullscreen quad)
compositePipeline := device.CreateGraphicsPipeline(...)

// Descriptor pool
compositeDescPool := device.CreateDescriptorPool(...)

// Sampler
layerSampler := device.CreateSampler(...)
```

### Step 2: Create Descriptor Sets for Layer Textures

**Location**: After layer framebuffer creation (after RenderTarget is added)

**Tasks**:
1. Allocate descriptor set for layer1
2. Update descriptor set to bind layer1's image view + sampler
3. Repeat for layer2

**Code Sketch**:
```go
// After layer1 RenderTarget is created:
layer1DescriptorSets := device.AllocateDescriptorSets(&vk.DescriptorSetAllocateInfo{
    DescriptorPool: compositeDescPool,
    SetLayouts:     []vk.DescriptorSetLayout{compositeDescSetLayout},
})
layer1DescriptorSet := layer1DescriptorSets[0]

device.UpdateDescriptorSets([]vk.WriteDescriptorSet{
    {
        DstSet:         layer1DescriptorSet,
        DstBinding:     0,
        DescriptorType: vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
        ImageInfo: []vk.DescriptorImageInfo{
            {
                Sampler:     layerSampler,
                ImageView:   layer1ImageView,
                ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
            },
        },
    },
})

// Store in component or global variable
// Repeat for layer2
```

### Step 3: Transition Layer Images to Correct Layout

**Location**: After framebuffer creation, before render loop

**Tasks**:
1. Transition layer images from UNDEFINED to COLOR_ATTACHMENT_OPTIMAL
2. Use pipeline barrier with image memory barrier

**Code Sketch**:
```go
// Use a one-time command buffer to transition images
transitionCmd := allocateOneTimeCommandBuffer()
transitionCmd.Begin(...)

transitionCmd.PipelineBarrier(
    vk.PIPELINE_STAGE_TOP_OF_PIPE_BIT,
    vk.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT,
    0,
    []vk.ImageMemoryBarrier{
        {
            OldLayout: vk.IMAGE_LAYOUT_UNDEFINED,
            NewLayout: vk.IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
            Image:     layer1Image,
            // ... subresource range
        },
        // Same for layer2Image
    },
)

transitionCmd.End()
submitAndWait(transitionCmd)
```

### Step 4: Refactor Render Loop (MAJOR CHANGE)

**Location**: Main render loop (~line 870+)

**Current Flow** (Direct rendering):
```go
for running {
    // Handle events
    // Acquire swapchain image
    // Record commands:
    //   - Transition swapchain to COLOR_ATTACHMENT
    //   - BeginRendering on swapchain
    //   - Render layers (RenderLayers system)
    //   - EndRendering
    //   - Transition swapchain to PRESENT
    // Submit
    // Present
}
```

**New Flow** (Multi-pass):
```go
for running {
    // Handle events
    // Acquire swapchain image

    // === PASS 1: Render layers to framebuffers ===
    for each layer in world.QueryRenderables():
        // Transition layer image: SHADER_READ → COLOR_ATTACHMENT
        cmd.PipelineBarrier(...)

        // Begin rendering to layer framebuffer
        cmd.BeginRendering(&vk.RenderingInfo{
            ColorAttachments: []vk.RenderingAttachmentInfo{{
                ImageView:   layer.RenderTarget.ImageView,
                LoadOp:      vk.ATTACHMENT_LOAD_OP_CLEAR,
                ClearValue:  vk.ClearValue{Color: {0,0,0,0}}, // Transparent
            }},
        })

        // Render layer content
        renderLayerContent(cmd, layer)

        cmd.EndRendering()

        // Transition layer image: COLOR_ATTACHMENT → SHADER_READ
        cmd.PipelineBarrier(...)

    // === PASS 2: Composite layers to swapchain ===
    // Transition swapchain: UNDEFINED → COLOR_ATTACHMENT
    cmd.PipelineBarrier(...)

    // Begin rendering to swapchain
    cmd.BeginRendering(&vk.RenderingInfo{
        ColorAttachments: []vk.RenderingAttachmentInfo{{
            ImageView:  swapImageViews[imageIndex],
            LoadOp:     vk.ATTACHMENT_LOAD_OP_CLEAR,
            ClearValue: vk.ClearValue{Color: {0,0,0,1}}, // Black background
        }},
    })

    // Sort layers by ZIndex
    sortedLayers := sortLayersByZIndex(world)

    // Composite each layer
    for each layer in sortedLayers:
        cmd.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, compositePipeline)
        cmd.BindDescriptorSets(..., layer.DescriptorSet)
        cmd.Draw(6, 1, 0, 0) // 6 vertices for fullscreen quad

    cmd.EndRendering()

    // Transition swapchain: COLOR_ATTACHMENT → PRESENT
    cmd.PipelineBarrier(...)

    // Submit
    // Present
}
```

### Step 5: Update RenderLayers System

**Location**: `systems/render.go`

**Change**: Split into two functions:
1. `RenderLayerContent(world, ctx, entity)` - Renders ONE layer's content to its framebuffer
2. `CompositeLayer(world, ctx, entity)` - Draws ONE layer's texture to swapchain

**Sketch**:
```go
// New function: render single layer to its framebuffer
func RenderLayerContent(world *ecs.World, ctx *RenderContext, entity ecs.Entity) {
    transform := world.GetTransform(entity)
    pipeline := world.GetVulkanPipeline(entity)
    blend := world.GetBlendMode(entity)

    if !blend.Visible { return }

    // Bind layer's content pipeline
    ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, pipeline.Pipeline)

    // Push constants (transform, opacity)
    // Bind descriptors (layer's source texture)
    // Draw quad
}

// New function: composite single layer to swapchain
func CompositeLayer(ctx *CompositeContext, layerDescriptorSet vk.DescriptorSet) {
    // Bind composite pipeline
    ctx.CommandBuffer.BindPipeline(vk.PIPELINE_BIND_POINT_GRAPHICS, ctx.CompositePipeline)

    // Bind layer's framebuffer texture
    ctx.CommandBuffer.BindDescriptorSets(
        vk.PIPELINE_BIND_POINT_GRAPHICS,
        ctx.CompositePipelineLayout,
        0,
        []vk.DescriptorSet{layerDescriptorSet},
        nil,
    )

    // Draw fullscreen quad (no vertex buffer needed)
    ctx.CommandBuffer.Draw(6, 1, 0, 0)
}
```

### Step 6: Layer Sorting

**Location**: New function in `ecs/query.go`

**Task**: Sort entities by ZIndex (ascending = back to front)

```go
func (w *World) QueryRenderablesSorted() []Entity {
    renderables := w.QueryRenderables()

    // Sort by ZIndex (ascending)
    sort.Slice(renderables, func(i, j int) bool {
        ti := w.GetTransform(renderables[i])
        tj := w.GetTransform(renderables[j])
        return ti.ZIndex < tj.ZIndex
    })

    return renderables
}
```

### Step 7: Store Descriptor Sets

**Challenge**: Descriptor sets are per-layer but created during initialization

**Solution**: Add to VulkanPipeline component or create new component

**Option A**: Extend VulkanPipeline
```go
type VulkanPipeline struct {
    Pipeline            vk.Pipeline
    PipelineLayout      vk.PipelineLayout
    DescriptorPool      vk.DescriptorPool
    DescriptorSet       vk.DescriptorSet       // For layer content rendering
    DescriptorSetLayout vk.DescriptorSetLayout

    // NEW: For composite pass
    CompositeDescriptorSet vk.DescriptorSet // For sampling layer framebuffer
}
```

**Option B**: New component
```go
type CompositeData struct {
    DescriptorSet vk.DescriptorSet // Samples this layer's framebuffer
}
```

Recommend **Option A** - simpler, keeps rendering data together.

---

## Testing Strategy

### Phase 1: Composite Pipeline Only
1. Disable layer rendering pass
2. Manually render to layer1 framebuffer (outside loop)
3. Test composite pass alone (blit layer1 to swapchain)
4. Verify: Should see static Djungelskog

### Phase 2: Single Layer Round-Trip
1. Enable layer rendering for layer1 only
2. Render layer1 content → layer1 framebuffer
3. Composite layer1 framebuffer → swapchain
4. Verify: Should see animated layer1

### Phase 3: Multi-Layer Compositing
1. Enable both layers
2. Verify Z-ordering works (layer2 on top)
3. Verify transparency blending (50% opacity on layer2)
4. Verify no flickering

### Phase 4: Stress Test
1. Add more layers
2. Verify sorting works
3. Check performance

---

## Potential Issues & Solutions

### Issue 1: Image Layout Confusion
**Symptom**: Validation errors about incorrect layouts
**Solution**: Carefully track layout transitions:
- Layer images: UNDEFINED → COLOR_ATTACHMENT → SHADER_READ (loop)
- Swapchain: UNDEFINED → COLOR_ATTACHMENT → PRESENT (each frame)

### Issue 2: Synchronization
**Symptom**: Flickering or black screen
**Solution**: Ensure pipeline barriers between:
- Layer rendering → Layer sampling
- Last composite → Present

### Issue 3: Descriptor Set Lifetime
**Symptom**: Crash when sampling layer texture
**Solution**: Ensure descriptor sets reference valid image views (created before descriptor set update)

### Issue 4: Alpha Blending
**Symptom**: Layers don't blend correctly
**Solution**:
- Composite pipeline must have blending enabled
- Clear layer framebuffers to transparent (0,0,0,0)
- Swapchain clear to opaque background

### Issue 5: Performance
**Symptom**: FPS drop
**Solution**:
- Profile to find bottleneck
- Consider: Don't re-render static layers every frame (future optimization)

---

## Code Organization

Suggest creating helper functions to keep `main()` readable:

```go
// In vala.go
func createCompositeResources(device, swapFormat) (pipeline, layout, descriptorPool, sampler) {
    // Step 1 code
}

func createLayerDescriptorSet(device, pool, layout, imageView, sampler) descriptorSet {
    // Step 2 code
}

func transitionLayerImages(cmd, layers) {
    // Step 3 code
}

// Main render loop stays in main(), but uses these helpers
```

Or create a new file `composite.go` for all composite-related code.

---

## Rollback Plan

If implementation gets stuck:

1. **Keep** framebuffer creation code (useful for future)
2. **Disable** composite pass
3. **Temporarily** add simple depth testing to fix flickering:
   - Create depth buffer
   - Enable depth test in pipeline
   - This gives immediate fix while we finish framebuffers properly

---

## Estimated Complexity

- **Lines of code**: ~300-400 new lines
- **New resources**: 6 Vulkan objects (pipeline, layout, pool, sampler, 2 descriptor sets)
- **Modified systems**: 1 major (render loop), 1 minor (RenderLayers split)
- **Risk level**: Medium-High (touches core rendering)
- **Debugging time**: Expect 1-2 hours for layout/sync issues

---

## Next Steps

1. Review this plan - any questions or changes?
2. Implement Step 1 (Composite pipeline resources)
3. Implement Step 2 (Descriptor sets)
4. Test Steps 1-2 in isolation
5. Implement Step 3 (Image transitions)
6. Implement Step 4 (Render loop refactor) - BIGGEST STEP
7. Test Phase 1, 2, 3 from testing strategy
8. Celebrate when Djungelskog renders correctly! 🎉

---

## References

- Vulkan Tutorial - Offscreen Rendering: https://vulkan-tutorial.com/
- Multi-pass rendering patterns
- Order-independent transparency (for future A-buffer work)
