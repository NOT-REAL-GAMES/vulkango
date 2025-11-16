# Sparse Binding Implementation Roadmap

This document outlines the steps needed to complete the sparse canvas implementation and unlock megatexture capabilities.

## Prerequisites

Before implementing sparse binding, ensure you understand:

1. **Vulkan Sparse Resources**: Read the spec sections on sparse binding
2. **Virtual Memory Concepts**: Page tables, resident/non-resident pages
3. **Queue Operations**: `vkQueueBindSparse` is different from regular command buffers
4. **Synchronization**: Sparse binding requires careful fence/semaphore handling

## Phase 1: Feature Detection & Validation

### Step 1.1: Query Device Features

```go
// In vulkango/device.go or wherever you query features
type PhysicalDeviceFeatures struct {
    // ... existing fields ...
    SparseBinding           bool
    SparseResidencyImage2D  bool
    SparseResidencyAliased  bool
}

// When creating device, check:
physicalFeatures := physicalDevice.GetFeatures()
if !physicalFeatures.SparseBinding {
    return fmt.Errorf("device doesn't support sparse binding")
}
if !physicalFeatures.SparseResidencyImage2D {
    return fmt.Errorf("device doesn't support sparse residency for 2D images")
}
```

### Step 1.2: Find Sparse Queue Family

```go
// Queue must support SPARSE_BINDING_BIT
func FindSparseQueueFamily(device vk.PhysicalDevice) (uint32, error) {
    queueFamilies := device.GetQueueFamilyProperties()

    for i, family := range queueFamilies {
        if family.QueueFlags & vk.QUEUE_SPARSE_BINDING_BIT != 0 {
            return uint32(i), nil
        }
    }

    return 0, fmt.Errorf("no queue family supports sparse binding")
}
```

### Step 1.3: Enable Features on Device Creation

```go
deviceCreateInfo := vk.DeviceCreateInfo{
    // ... other fields ...
    EnabledFeatures: &vk.PhysicalDeviceFeatures{
        SparseBinding:          true,
        SparseResidencyImage2D: true,
    },
}
```

## Phase 2: Sparse Image Creation

### Step 2.1: Create Sparse Image

```go
// In sparse.go NewSparseCanvas()
imageInfo := vk.ImageCreateInfo{
    ImageType: vk.IMAGE_TYPE_2D,
    Format:    cfg.Format,
    Extent: vk.Extent3D{
        Width:  cfg.Width,
        Height: cfg.Height,
        Depth:  1,
    },
    MipLevels:   1,
    ArrayLayers: 1,
    Samples:     vk.SAMPLE_COUNT_1_BIT,
    Tiling:      vk.IMAGE_TILING_OPTIMAL,

    // KEY: Sparse binding flags
    Flags: vk.IMAGE_CREATE_SPARSE_BINDING_BIT |
           vk.IMAGE_CREATE_SPARSE_RESIDENCY_BIT,

    Usage:       cfg.Usage,
    SharingMode: vk.SHARING_MODE_EXCLUSIVE,
    InitialLayout: vk.IMAGE_LAYOUT_UNDEFINED,
}

image, err := cfg.Device.CreateImage(&imageInfo)
if err != nil {
    return nil, err
}
```

### Step 2.2: Query Sparse Memory Requirements

```go
// This is the critical part - tells us page size and structure
var count uint32
device.GetImageSparseMemoryRequirements(image, &count, nil)

sparseReqs := make([]vk.SparseImageMemoryRequirements, count)
device.GetImageSparseMemoryRequirements(image, &count, sparseReqs)

// Extract key information
for _, req := range sparseReqs {
    formatProps := req.FormatProperties

    // Page size in pixels (e.g., 256x256)
    pageWidth := formatProps.ImageGranularity.Width
    pageHeight := formatProps.ImageGranularity.Height

    // Which memory types are compatible
    memoryTypeBits := req.ImageMemoryRequirements.MemoryTypeBits

    // Mip tail info (lower mips that must be bound together)
    mipTailFirstLod := req.ImageMipTailFirstLod
    mipTailSize := req.ImageMipTailSize
    mipTailOffset := req.ImageMipTailOffset
}
```

## Phase 3: Memory Allocation & Binding

### Step 3.1: Allocate Memory for a Page

```go
func (c *SparseCanvas) allocatePageMemory() (vk.DeviceMemory, error) {
    allocInfo := vk.MemoryAllocateInfo{
        AllocationSize:  c.sparsePageSize,
        MemoryTypeIndex: c.sparseMemoryTypeIndex,
    }

    return c.device.AllocateMemory(&allocInfo)
}
```

### Step 3.2: Bind Page to Image (AllocatePage implementation)

```go
func (c *SparseCanvas) AllocatePage(pageX, pageY int) error {
    // Check if already allocated
    pageKey := pageKey(pageX, pageY)
    if _, exists := c.boundPages[pageKey]; exists {
        return nil // Already allocated
    }

    // Allocate memory
    memory, err := c.allocatePageMemory()
    if err != nil {
        return err
    }

    // Build sparse bind info
    bind := vk.SparseImageMemoryBind{
        Subresource: vk.ImageSubresource{
            AspectMask: vk.IMAGE_ASPECT_COLOR_BIT,
            MipLevel:   0,
            ArrayLayer: 0,
        },
        Offset: vk.Offset3D{
            X: int32(pageX * PageSize),
            Y: int32(pageY * PageSize),
            Z: 0,
        },
        Extent: vk.Extent3D{
            Width:  PageSize,
            Height: PageSize,
            Depth:  1,
        },
        Memory:       memory,
        MemoryOffset: 0,
        Flags:        0,
    }

    bindInfo := vk.BindSparseInfo{
        ImageBinds: []vk.SparseImageMemoryBindInfo{
            {
                Image: c.image,
                Binds: []vk.SparseImageMemoryBind{bind},
            },
        },
    }

    // Submit to queue (NOT a command buffer!)
    // This is a queue operation, not a command
    fence, _ := c.device.CreateFence(nil)
    err = c.queue.BindSparse([]vk.BindSparseInfo{bindInfo}, fence)
    if err != nil {
        c.device.FreeMemory(memory)
        return err
    }

    // Wait for binding to complete
    c.device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))
    c.device.DestroyFence(fence)

    // Track the binding
    c.boundPages[pageKey] = memory

    return nil
}
```

### Step 3.3: Unbind Page (DeallocatePage implementation)

```go
func (c *SparseCanvas) DeallocatePage(pageX, pageY int) error {
    pageKey := pageKey(pageX, pageY)
    memory, exists := c.boundPages[pageKey]
    if !exists {
        return nil // Not allocated
    }

    // Unbind by binding VK_NULL_HANDLE
    bind := vk.SparseImageMemoryBind{
        Subresource: vk.ImageSubresource{
            AspectMask: vk.IMAGE_ASPECT_COLOR_BIT,
            MipLevel:   0,
            ArrayLayer: 0,
        },
        Offset: vk.Offset3D{
            X: int32(pageX * PageSize),
            Y: int32(pageY * PageSize),
            Z: 0,
        },
        Extent: vk.Extent3D{
            Width:  PageSize,
            Height: PageSize,
            Depth:  1,
        },
        Memory:       0, // VK_NULL_HANDLE unbinds the page
        MemoryOffset: 0,
    }

    bindInfo := vk.BindSparseInfo{
        ImageBinds: []vk.SparseImageMemoryBindInfo{
            {
                Image: c.image,
                Binds: []vk.SparseImageMemoryBind{bind},
            },
        },
    }

    fence, _ := c.device.CreateFence(nil)
    c.queue.BindSparse([]vk.BindSparseInfo{bindInfo}, fence)
    c.device.WaitForFences([]vk.Fence{fence}, true, ^uint64(0))
    c.device.DestroyFence(fence)

    // Free the memory
    c.device.FreeMemory(memory)
    delete(c.boundPages, pageKey)

    return nil
}
```

## Phase 4: Upload to Sparse Canvas

### Step 4.1: Ensure Pages Are Allocated Before Upload

```go
func (c *SparseCanvas) Upload(x, y, width, height uint32, data []byte) error {
    // Find all pages touched by this upload
    pages := GetPagesInRect(x, y, width, height)

    // Allocate any non-resident pages
    for _, page := range pages {
        if !c.IsPageAllocated(page.X, page.Y) {
            err := c.AllocatePage(page.X, page.Y)
            if err != nil {
                return fmt.Errorf("failed to allocate page (%d,%d): %w",
                    page.X, page.Y, err)
            }
        }
    }

    // Now do the upload (same as dense canvas)
    return c.uploadInternal(x, y, width, height, data)
}
```

## Phase 5: Vulkan Bindings Updates

You'll need to add these to `vulkango`:

### types.go
```go
// Image creation flags
const IMAGE_CREATE_SPARSE_BINDING_BIT ImageCreateFlags = 0x00000001
const IMAGE_CREATE_SPARSE_RESIDENCY_BIT ImageCreateFlags = 0x00000002
const IMAGE_CREATE_SPARSE_ALIASED_BIT ImageCreateFlags = 0x00000004

// Queue flags
const QUEUE_SPARSE_BINDING_BIT QueueFlags = 0x00000008
```

### sparse.go (new file in vulkango)
```go
package vulkango

// #include <vulkan/vulkan.h>
import "C"

type SparseImageMemoryRequirements struct {
    FormatProperties         SparseImageFormatProperties
    ImageMipTailFirstLod     uint32
    ImageMipTailSize         DeviceSize
    ImageMipTailOffset       DeviceSize
    ImageMipTailStride       DeviceSize
}

type SparseImageFormatProperties struct {
    AspectMask       ImageAspectFlags
    ImageGranularity Extent3D
    Flags            SparseImageFormatFlags
}

type SparseImageMemoryBind struct {
    Subresource  ImageSubresource
    Offset       Offset3D
    Extent       Extent3D
    Memory       DeviceMemory
    MemoryOffset DeviceSize
    Flags        SparseMemoryBindFlags
}

type SparseImageMemoryBindInfo struct {
    Image Image
    Binds []SparseImageMemoryBind
}

type BindSparseInfo struct {
    WaitSemaphores   []Semaphore
    BufferBinds      []SparseBufferMemoryBindInfo
    ImageOpaqueBinds []SparseImageOpaqueMemoryBindInfo
    ImageBinds       []SparseImageMemoryBindInfo
    SignalSemaphores []Semaphore
}

func (d Device) GetImageSparseMemoryRequirements(image Image,
    count *uint32, reqs []SparseImageMemoryRequirements) {
    // CGO implementation
}

func (q Queue) BindSparse(bindInfo []BindSparseInfo, fence Fence) error {
    // CGO implementation
}
```

## Phase 6: Testing Strategy

### Test 1: Basic Sparse Image Creation
```go
func TestSparseImageCreation() {
    // Create sparse image
    // Query sparse requirements
    // Verify granularity is reasonable (e.g., 128x128 or 256x256)
}
```

### Test 2: Single Page Bind
```go
func TestSinglePageBind() {
    // Create sparse canvas
    // Allocate one page
    // Verify IsPageAllocated returns true
    // Deallocate page
    // Verify IsPageAllocated returns false
}
```

### Test 3: Upload to Sparse Canvas
```go
func TestSparseUpload() {
    // Create sparse canvas
    // Upload data to unallocated region
    // Verify pages are auto-allocated
    // Render the canvas
    // Verify data appears correctly
}
```

### Test 4: Large Sparse Canvas
```go
func TestLargeCanvas() {
    // Create 32K x 32K sparse canvas
    // Allocate only a few pages
    // Verify memory usage is small (not 32K*32K*4 bytes)
    // Upload to scattered locations
    // Verify all regions render correctly
}
```

## Phase 7: Advanced Features

Once basic sparse binding works:

### 7.1: Page Cache with LRU Eviction
```go
type PageCache struct {
    maxPages int
    pages    map[uint32]*CachedPage
    lru      *list.List // Doubly-linked list for LRU tracking
}

func (pc *PageCache) Touch(pageX, pageY int) {
    // Move page to front of LRU list
}

func (pc *PageCache) Evict() (pageX, pageY int, err error) {
    // Remove least-recently-used page
    // Save to disk if dirty
    // Unbind from GPU
}
```

### 7.2: Feedback Buffer
```go
// Render to tiny buffer tracking page access
feedbackBuffer := createStorageBuffer(...)

// In shader:
// layout(set = 0, binding = 1) buffer FeedbackBuffer {
//     uint pageAccess[];
// };
//
// When sampling canvas:
//     vec2 pageCoord = floor(texCoord * canvasSize / pageSize);
//     uint pageIndex = uint(pageCoord.y) * pagesX + uint(pageCoord.x);
//     atomicOr(pageAccess[pageIndex], 1);

// Read back and allocate touched pages
```

### 7.3: Async Page Streaming
```go
type PageStreamer struct {
    loadQueue  chan PageCoord
    saveQueue  chan PageCoord
    diskCache  *DiskCache
}

func (ps *PageStreamer) Start() {
    go ps.loadWorker()  // Background thread loading pages
    go ps.saveWorker()  // Background thread saving evicted pages
}
```

## Common Pitfalls

1. **Forgetting to wait on fences**: `QueueBindSparse` is asynchronous!
2. **Wrong queue family**: Must have `QUEUE_SPARSE_BINDING_BIT`
3. **Page alignment**: Offsets must align to granularity
4. **Mip tail**: Lower mip levels might need special handling
5. **Undefined reads**: Reading unbound pages is undefined behavior (not a crash, but garbage data)

## Performance Considerations

- **Binding is expensive**: Don't bind individual pages; batch them
- **Fence overhead**: Reuse fences when possible
- **Page size matters**: Larger pages = less overhead, smaller = more granular control
- **Memory fragmentation**: Consider using a pool allocator for page memory

## Resources

- **Vulkan Spec**: Chapter 31 - Sparse Resources
- **NVIDIA Guide**: "Sparse Virtual Textures in Vulkan"
- **id Tech Papers**: GDC talks on megatextures
- **Sascha Willems**: Vulkan sparse binding examples

## Estimated Implementation Time

- Phase 1-2: Feature detection & image creation → 2-4 hours
- Phase 3: Memory binding → 4-8 hours
- Phase 4: Upload integration → 2-4 hours
- Phase 5: Vulkan bindings → 4-6 hours
- Phase 6: Testing → 4-8 hours
- **Total**: ~20-30 hours for working sparse binding

Advanced features (Phase 7) are optional and each could take 10-20 hours depending on complexity.

## When to Implement?

**Implement sparse binding when:**
- ✅ Basic dense canvas is working and tested
- ✅ You need canvases larger than VRAM allows
- ✅ You're comfortable with Vulkan synchronization
- ✅ You've profiled and dense allocation is a bottleneck

**Don't implement it if:**
- ❌ Dense canvases work fine for your use case
- ❌ Other features are higher priority
- ❌ You're still learning Vulkan basics

Remember: **Working > Perfect**. Ship with dense canvases, add sparse binding later when you need it!
