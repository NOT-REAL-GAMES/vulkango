# VALA - Hybrid Animation & Compositing Software

## Project Overview

VALA (name TBD) is a hybrid animation and compositing application that combines the best aspects of Krita (painting), Blender (3D/animation), and After Effects (compositing) into a unified, high-performance tool built from the ground up in Go.

**Core Philosophy:** Every layer is a complete, independent rendering pipeline with full control over its graphics state, allowing unprecedented flexibility in non-destructive editing and compositing.

## Technical Stack

### Language & Core
- **Go** - Primary language for the entire application
- **Vulkan 1.3+** - Graphics API (via custom bindings: `vulkango`)
- **SDL3** - Windowing and input (via custom bindings: `sdl3go`)
- **shaderc** - Runtime GLSL shader compilation

### Custom Modules
- **vulkango** - Complete Vulkan 1.3 bindings with modern features:
  - Dynamic rendering (no render passes)
  - Descriptor sets and pipeline management
  - Buffer and image operations
  - Synchronization primitives
  
- **sdl3go** - SDL3 bindings for:
  - Cross-platform windowing
  - Input handling (mouse, keyboard, touch)
  - Event processing

### Build Philosophy
- "Ship fast" mentality - prioritize getting working features over perfect architecture
- Cross-platform from day one (Windows, Linux, eventually macOS)
- Small executable size (<10MB goal)
- No external engine dependencies - built from scratch

## Architecture Concepts

### The Layer System

**Core Innovation:** Each layer in VALA is a complete, independent renderer with its own:
- Vulkan pipeline state
- Vertex and index buffers
- Shader modules (vertex, fragment, compute)
- Descriptor sets (textures, uniforms)
- Render targets and framebuffers
- Projection matrices and transforms
- Depth/stencil buffers

**Key Properties:**
- **Non-Destructive Editing** - All layer properties can be modified without affecting source data
- **Desynchronization** - Layers can be desynced to create variations without duplication
- **Inter-Layer Operations** - Layers can reference each other's buffers:
  - Use depth buffer of Layer A as mask for Layer B
  - Sample color output of Layer C in Layer D's shader
  - Chain effects across multiple layers

### Rendering Modes

Each layer can operate in different modes:

1. **Raster Mode** - Pixel-based painting (Krita-style)
   - Brush strokes rendered to texture
   - Pressure sensitivity support
   - Blending modes and opacity
   
2. **Vector Mode** - Path-based drawing
   - Bezier curves and shapes
   - Resolution-independent
   - GPU-accelerated rendering
   
3. **Shader Mode** - Custom GLSL shaders
   - Full control over rendering pipeline
   - Access to all layer buffers
   - Real-time preview
   
4. **Composite Mode** - Combine other layers
   - Blend modes, masks, adjustments
   - Reference other layers' outputs
   - Non-destructive operations

### Timeline & Animation

- **Frame-based timeline** - Traditional animation workflow
- **Keyframe interpolation** - Smooth transitions between states
- **Layer properties animatable** - Transform, shader uniforms, blend modes, etc.
- **Onion skinning** - See previous/next frames while animating
- **Real-time playback** - Hardware-accelerated preview

## Development Philosophy

### "Ship Fast" Approach

RB (the developer) previously struggled with abandoning 50+ projects over a decade. The breakthrough came from adopting a "ship fast" mentality:

- **Weekly shipping cadence** - Release something working every week
- **Minimum viable features** - Get core functionality working, polish later
- **Iterative improvements** - Build on working foundation rather than perfect architecture
- **Avoid over-engineering** - YAGNI (You Aren't Gonna Need It) principle

### Code Organization

**Preferred Structure:**
- Keep related functionality together
- Favor clarity over cleverness
- Document non-obvious decisions
- Use Go's simplicity - avoid over-abstraction

**Vulkan Patterns:**
- Create dedicated files for major Vulkan concepts (buffers, images, pipelines, etc.)
- Use helper functions to reduce boilerplate
- Always clean up resources properly (defer pattern)
- Minimize CGO complexity - keep C interop localized

### Error Handling

- Panic on initialization/setup errors (fail fast)
- Return errors for runtime operations
- Provide clear error messages
- Log useful debugging information

## Current State

### Completed ✅
- Vulkan instance and device creation
- Swapchain with image views
- Graphics pipeline with dynamic rendering
- Vertex and index buffer system
- Texture loading and sampling
- Descriptor sets for shader bindings
- Command buffer recording and submission
- Synchronization (fences, semaphores)
- Basic render loop
- Cross-platform window creation (SDL3)
- Shader compilation via shaderc
- Image loading (PNG, JPEG via Go's image package)

### In Progress 🚧
- Uniform buffer support for transforms
- Multiple render targets
- Framebuffer system for layer rendering

### Planned 📋
- Brush engine (raster painting)
- Vector path renderer
- Timeline and keyframe system
- Layer panel UI
- Tools system (brush, pen, selection, transform)
- File format (save/load projects)
- Export pipeline (video, image sequences)
- Compute shader integration
- Mesh shaders (future consideration)

## Technical Considerations

### Memory Management

**Vulkan Resources:**
- Always pair Create with Destroy
- Use defer for cleanup in initialization code
- Consider memory pooling for frequently allocated resources
- Track memory usage - GPU memory is limited

**Go Patterns:**
- Minimize allocations in hot paths (render loop)
- Use object pooling for command buffers
- Be mindful of CGO overhead - batch operations

### Performance Targets

- **60 FPS minimum** for UI and preview rendering
- **Sub-frame latency** for brush input
- **Real-time playback** at target framerate
- **Efficient memory usage** - handle 4K+ textures
- **Fast shader compilation** - compile on-demand with caching

### Vulkan-Specific Notes

**Dynamic Rendering:**
- No render passes needed (Vulkan 1.3 core feature)
- Enable `dynamicRendering` device feature
- Use `VkPipelineRenderingCreateInfo` in pipeline creation

**Synchronization:**
- Use timeline semaphores for complex dependencies (future)
- Fence per frame in flight (current pattern)
- Pipeline barriers for image layout transitions

**Descriptor Management:**
- Pool per layer or global pool with careful tracking
- Update descriptor sets when layer resources change
- Consider bindless descriptors (future optimization)

### Cross-Platform Considerations

**Windows:**
- MSYS2 MinGW64 toolchain for building
- pkg-config for dependency discovery
- Vulkan SDK from LunarG

**Linux:**
- Standard apt packages for dependencies
- Native Vulkan support via Mesa/proprietary drivers
- Wayland/X11 support via SDL3

**macOS (Future):**
- MoltenVK for Vulkan on Metal
- Requires special consideration for shader compilation
- Different window management paradigms

## Code Style

### Go Conventions
- Follow standard Go formatting (gofmt)
- Use meaningful variable names
- Keep functions focused and small
- Prefer composition over inheritance

### Vulkan Bindings Style
- Type-safe wrappers around C types
- Go-friendly naming (PascalCase for exports)
- Hide CGO complexity behind clean Go APIs
- Provide helper functions for common patterns

### Naming Patterns
- Vulkan types: `Device`, `Buffer`, `Pipeline` (no `Vk` prefix)
- Create functions: `CreateX(info)` returns `(X, error)`
- Destroy functions: `DestroyX(x)` - no return value
- Info structs: `XCreateInfo` - match Vulkan naming

## Interaction Guidelines

### When Helping with VALA

**Do:**
- Provide complete, working code examples
- Explain Vulkan concepts when introducing new features
- Consider performance implications
- Think about the "ship fast" philosophy
- Suggest iterative improvements
- Reference the custom vulkango/sdl3go APIs

**Don't:**
- Over-engineer solutions
- Add dependencies without strong justification
- Suggest rewrites without clear benefit
- Introduce complex abstractions prematurely
- Forget to handle Vulkan synchronization
- Assume standard library Vulkan bindings exist

### Common Tasks

**Adding a new Vulkan feature:**
1. Check if types exist in vulkango
2. Add necessary constants/types to types.go
3. Create dedicated file (e.g., `compute.go`)
4. Write CGO wrappers for Vulkan functions
5. Add helper functions for common patterns
6. Test with minimal example

**Debugging Vulkan issues:**
1. Enable validation layers
2. Check VkResult return codes
3. Verify synchronization (fences/semaphores)
4. Confirm resource creation order
5. Check pipeline state matches usage
6. Validate descriptor set bindings

**Performance optimization:**
1. Profile before optimizing
2. Reduce draw calls (batch geometry)
3. Minimize pipeline switches
4. Use staging buffers efficiently
5. Consider compute shaders for parallel work
6. Cache compiled shaders

## Project Goals

### Short-term (Current Phase)
- ✅ Establish solid Vulkan rendering foundation
- ✅ Implement texture-mapped quad rendering
- 🚧 Add transform system (pan, zoom, rotate)
- 🚧 Basic UI framework for tools
- 📋 Simple brush implementation
- 📋 Layer panel mockup

### Medium-term (Next Months)
- Complete layer system with independent pipelines
- Full painting engine with pressure sensitivity
- Timeline with keyframe support
- Multiple blend modes and compositing operations
- File save/load (custom format)
- Export to common formats (PNG sequence, MP4)

### Long-term (Vision)
- Professional-grade painting tools
- Advanced vector graphics engine
- 3D layer support (imported models)
- Node-based shader editor
- Python/Lua scripting API
- Plugin system for extensibility
- Cloud collaboration features

## Success Metrics

**Technical:**
- Consistent 60+ FPS during painting
- <100ms brush latency
- Stable memory usage under heavy use
- Cross-platform feature parity

**User Experience:**
- Intuitive layer management
- Responsive UI
- Non-destructive workflow
- Fast iteration times

**Project Health:**
- Weekly progress/releases
- Growing feature set
- Maintainable codebase
- Active development momentum

## Resources & References

**Official Documentation:**
- Vulkan Specification: https://registry.khronos.org/vulkan/specs/1.3/html/
- SDL3 Wiki: https://wiki.libsdl.org/SDL3/
- Go Documentation: https://go.dev/doc/

**Learning Resources:**
- Vulkan Tutorial: https://vulkan-tutorial.com/
- Sascha Willems Examples: https://github.com/SaschaWillems/Vulkan

**Similar Projects (Inspiration):**
- Krita (https://krita.org/) - Painting workflow
- Blender (https://www.blender.org/) - Timeline and animation
- After Effects - Compositing and effects

---

## Notes for Claude

When working on VALA:
1. **Always consider the layer-as-renderer architecture** - This is the core innovation
2. **Respect the "ship fast" philosophy** - Working > Perfect
3. **Use the custom bindings** - Don't reference standard library Vulkan packages
4. **Think cross-platform** - Test assumptions on both Windows and Linux
5. **Validate Vulkan usage** - Synchronization bugs are subtle
6. **Keep RB's momentum** - This project represents breaking a decade-long pattern of abandonment

The developer (RB) is a solo indie game developer who has successfully shipped games (TIMEHEIST, GOBI) and is now building foundational graphics tooling. They value directness, practical solutions, and maintaining shipping velocity.