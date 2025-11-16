# Canvas System Usage Guide

## Overview

The canvas system provides a megatexture-inspired abstraction for managing large textures with optional sparse binding (virtual texturing). It's designed to work seamlessly whether you use dense allocation (regular textures) or sparse binding (on-demand page loading).

## Quick Start

### Creating a Canvas

```go
import (
    "github.com/NOT-REAL-GAMES/vulkango/vala/canvas"
    vk "github.com/NOT-REAL-GAMES/vulkango"
)

// Create a 8K x 8K canvas for painting
myCanvas, err := canvas.New(canvas.Config{
    Device: device,
    Width:  8192,
    Height: 8192,
    Format: vk.FORMAT_R8G8B8A8_UNORM,
    Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
           vk.IMAGE_USAGE_SAMPLED_BIT |
           vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
    UseSparseBinding: false, // Start with dense allocation
}, commandPool, queue)
if err != nil {
    panic(err)
}
defer myCanvas.Destroy()
```

### Uploading Pixel Data

```go
// Create some test data (red square)
width, height := uint32(512), uint32(512)
data := make([]byte, width*height*4) // RGBA format
for i := 0; i < len(data); i += 4 {
    data[i] = 255   // R
    data[i+1] = 0   // G
    data[i+2] = 0   // B
    data[i+3] = 255 // A
}

// Upload to canvas at position (100, 100)
err = myCanvas.Upload(100, 100, width, height, data)
if err != nil {
    panic(err)
}
```

### Using in Rendering

```go
// Get the image and view for binding to descriptors
image := myCanvas.GetImage()
imageView := myCanvas.GetView()

// Update descriptor set to sample from canvas
writes := []vk.WriteDescriptorSet{
    {
        DstSet:          descriptorSet,
        DstBinding:      0,
        DstArrayElement: 0,
        DescriptorType:  vk.DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
        ImageInfo: []vk.DescriptorImageInfo{
            {
                ImageLayout: vk.IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
                ImageView:   imageView,
                Sampler:     sampler,
            },
        },
    },
}
device.UpdateDescriptorSets(writes, nil)

// Now render using the canvas texture in your shaders
```

## Page-Based Operations

The canvas system uses a page-based abstraction even for dense textures. This makes it easy to upgrade to sparse binding later without changing your code.

### Working with Pages

```go
// Page size is 256x256 pixels by default
const pageSize = canvas.PageSize

// Convert pixel coordinates to page coordinates
page := canvas.PixelToPage(1000, 1000)
// page.X = 3, page.Y = 3 (since 1000/256 = 3)

// Convert page coordinates back to pixels
x, y := canvas.PageToPixel(page.X, page.Y)
// x = 768, y = 768 (top-left corner of the page)

// Find all pages that intersect a rectangle
pages := canvas.GetPagesInRect(500, 500, 1024, 1024)
// Returns all pages that overlap this region
for _, page := range pages {
    fmt.Printf("Page (%d, %d)\n", page.X, page.Y)
}
```

### Page Allocation (Sparse Canvases Only)

For dense canvases, all pages are allocated automatically. For sparse canvases (future), you can manage pages on-demand:

```go
// Check if a page is allocated
if !myCanvas.IsPageAllocated(pageX, pageY) {
    // Allocate physical memory for this page
    err := myCanvas.AllocatePage(pageX, pageY)
    if err != nil {
        panic(err)
    }
}

// Later, deallocate pages you don't need
err = myCanvas.DeallocatePage(pageX, pageY)
```

## Migrating to Sparse Binding

When you're ready to enable megatextures, just change the config:

```go
myCanvas, err := canvas.New(canvas.Config{
    Device: device,
    Width:  16384, // Now you can go even bigger!
    Height: 16384,
    Format: vk.FORMAT_R8G8B8A8_UNORM,
    Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
           vk.IMAGE_USAGE_SAMPLED_BIT |
           vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
    UseSparseBinding: true, // Enable sparse binding
    InitiallyAllocateAll: false, // Only allocate pages as needed
}, commandPool, queue)
```

The rest of your code stays the same! The canvas implementation will handle sparse page binding automatically.

## Advanced Patterns

### Dirty Region Tracking

```go
// Mark regions as dirty for potential optimizations
myCanvas.MarkDirty(100, 100, 512, 512)

// Future: Could batch uploads, defer GPU transfers, etc.
```

### On-Demand Page Loading

```go
// When uploading data, automatically allocate pages if needed
err := myCanvas.Upload(x, y, width, height, brushStrokeData)
// This will:
// 1. Calculate which pages are touched by this upload
// 2. Allocate any pages that aren't yet resident (sparse only)
// 3. Upload the data
```

### Multiple Canvases (Layer System)

```go
// Create multiple canvases for different layers
backgroundCanvas, _ := canvas.New(config, cmdPool, queue)
paintCanvas, _ := canvas.New(config, cmdPool, queue)
effectsCanvas, _ := canvas.New(config, cmdPool, queue)

// Each canvas is independent and can be sampled/rendered separately
// Composite them in your final render pass
```

## Performance Tips

### For Dense Canvases

1. **Batch uploads**: Instead of many small uploads, combine them when possible
2. **Reuse staging buffers**: The current implementation creates a new staging buffer per upload (could be optimized)
3. **Use appropriate formats**: R8G8B8A8 is fine for painting, consider R16G16B16A16_SFLOAT for HDR

### For Sparse Canvases (Future)

1. **Allocate on first paint**: Only allocate pages when the user paints on them
2. **Prefetch around cursor**: Allocate pages near the brush before painting
3. **LRU eviction**: When memory is full, evict least-recently-used pages
4. **Save to disk**: For very large canvases, save evicted pages to disk

## Architecture Benefits

### Why This Design?

1. **Start Simple**: Use dense allocation today, works immediately
2. **Easy Upgrade**: Flip one flag to enable sparse binding later
3. **Consistent API**: Same interface whether sparse or dense
4. **Future-Proof**: Ready for megatexture features when needed
5. **Testable**: Can test sparse pipeline with `InitiallyAllocateAll: true`

### What's the Point?

- **Dense Canvas**: "I have 2GB VRAM, give me a 4K canvas" → Works great!
- **Sparse Canvas**: "I have 2GB VRAM, give me a 32K canvas" → Only allocates painted regions!

This is the same technique id Software used for Rage and Doom Eternal's virtual texturing system.

## Current Limitations

### Not Yet Implemented

- `Clear()` function (fill regions with solid color)
- Mipmap support
- Multi-sampled canvases (MSAA)
- Sparse binding itself (the sparse.go file is a stub)
- Page compression/streaming
- Automatic page eviction

### Planned Features

When sparse binding is implemented:

1. **Feedback Buffer**: GPU reports which pages were accessed during rendering
2. **Async Loading**: Background thread loads pages from disk
3. **Prefetching**: Predict and pre-load pages based on brush movement
4. **Page Cache**: LRU cache with configurable memory limit
5. **Compression**: BC7/ASTC compression for smaller memory footprint

## Example: Painting Application

```go
package main

import (
    "github.com/NOT-REAL-GAMES/vulkango/vala/canvas"
    vk "github.com/NOT-REAL-GAMES/vulkango"
)

func main() {
    // ... Vulkan initialization ...

    // Create main painting canvas (8K for high-res artwork)
    paintCanvas, err := canvas.New(canvas.Config{
        Device: device,
        Width:  8192,
        Height: 8192,
        Format: vk.FORMAT_R8G8B8A8_UNORM,
        Usage: vk.IMAGE_USAGE_TRANSFER_DST_BIT |
               vk.IMAGE_USAGE_SAMPLED_BIT |
               vk.IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
        UseSparseBinding: false,
    }, commandPool, queue)
    if err != nil {
        panic(err)
    }
    defer paintCanvas.Destroy()

    // Main loop
    for running {
        // Handle input...
        if brushStroke {
            // Render brush stroke to CPU buffer
            brushData := renderBrushStroke(x, y, pressure, color)

            // Upload to canvas
            paintCanvas.Upload(x, y, brushSize, brushSize, brushData)

            // Mark dirty for potential optimizations
            paintCanvas.MarkDirty(x, y, brushSize, brushSize)
        }

        // Render canvas to screen
        renderCanvasToScreen(paintCanvas)
    }
}
```

## Summary

The canvas system gives you:

- ✅ Simple API for texture management
- ✅ Page-based abstraction for future sparse binding
- ✅ Working implementation today (dense)
- ✅ Clear upgrade path to megatextures
- ✅ Designed for VALA's painting/compositing workflow

Start with `UseSparseBinding: false` and get to work. When you need massive canvases, flip it to `true` and watch the magic happen! 🎨
