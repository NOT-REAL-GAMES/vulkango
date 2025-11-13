# CLAUDE.md - AI Assistant Guide for vulkango

## Project Overview

**vulkango** is a Go library providing low-level bindings to the Vulkan graphics API using CGo. This project enables Go developers to interact with Vulkan for graphics programming, compute tasks, and GPU-accelerated operations.

- **Language**: Go 1.25.1
- **Module Path**: `github.com/NOT-REAL-GAMES/vulkango`
- **License**: MIT License (Copyright 2025 Arbiter Waldorff)
- **Purpose**: Vulkan API bindings for Go

## Repository Structure

```
vulkango/
├── go.mod              # Go module definition
├── types.go            # Vulkan types and result codes
├── instance.go         # Vulkan instance functions
├── examples/           # Example code demonstrating usage
│   └── test.go        # Basic example showing version enumeration
├── LICENSE             # MIT License
├── README.md           # Project readme (minimal)
└── .gitignore         # Git ignore rules
```

## Core Files

### types.go (5978 bytes)
- Defines the `Result` type (int32) for Vulkan result codes
- Comprehensive list of Vulkan result constants (SUCCESS, ERROR codes, etc.)
- Implements `Error()` method on `Result` type for readable error messages
- All result codes match Vulkan specification naming
- Uses CGo import ("C")

**Key patterns**:
- Result codes follow pattern: `RESULT_NAME Result = value`
- Positive values indicate success/informational states
- Negative values indicate errors
- Some error codes use large negative values (extension-specific)

### instance.go (378 bytes)
- Platform-specific CGo linker directives for Vulkan library
  - Windows: `-lvulkan-1`
  - Linux: `-lvulkan`
  - macOS: `-lvulkan`
- Includes `<vulkan/vulkan.h>` header
- Currently implements one function: `EnumerateInstanceVersion()`
  - Returns Vulkan version as uint32
  - Returns error if Vulkan call fails

### examples/test.go (216 bytes)
- Demonstrates basic usage of the library
- Shows how to query Vulkan version
- Imports package with alias: `vk "github.com/NOT-REAL-GAMES/vulkango"`
- Simple error handling pattern with panic

## Development Conventions

### Code Style

1. **Go Formatting**: Use `gofmt` for all Go code
2. **Naming Conventions**:
   - Vulkan constants use SCREAMING_SNAKE_CASE (matching Vulkan spec)
   - Go functions use PascalCase (exported) or camelCase (unexported)
   - Package alias `vk` is used in examples

3. **Error Handling**:
   - Functions return `(value, error)` tuple pattern
   - Vulkan result codes are converted to Go errors via `Result` type
   - Result type implements `error` interface via `Error()` method

4. **CGo Integration**:
   - Use `import "C"` for CGo access
   - Platform-specific linker flags via `// #cgo` directives
   - Direct C function calls prefixed with `C.` (e.g., `C.vkEnumerateInstanceVersion`)
   - Type conversions between C and Go types (e.g., `C.uint32_t` to `uint32`)

### Project Structure Patterns

1. **Flat Structure**: Library maintains a simple, flat file structure
2. **Separation of Concerns**:
   - Types and constants in `types.go`
   - Vulkan functions organized by category (instance.go for instance functions)
3. **Examples**: Separate `examples/` directory for demonstration code

### Git Workflow

- **Current Branch**: `claude/claude-md-mhxujj9c2251bojj-01CW3MH1JM1Rowi7QQKkT24d`
- **Branch Naming**: Feature branches use descriptive names
- **Commit Style**: Recent commits use casual, brief messages
- **Clean State**: Repository currently has no uncommitted changes

### Build and Dependencies

1. **No External Go Dependencies**: Pure Vulkan bindings with no Go package dependencies
2. **System Requirements**:
   - Vulkan SDK must be installed on the system
   - Platform-specific Vulkan libraries (vulkan-1.dll, libvulkan.so, etc.)

3. **Building**:
   ```bash
   # Build examples
   go build ./examples/test.go

   # Run tests (when available)
   go test ./...

   # Format code
   go fmt ./...
   ```

4. **CGo Requirements**:
   - C compiler required (gcc, clang, or MSVC)
   - Vulkan headers must be in include path
   - Vulkan libraries must be in library path

## AI Assistant Guidelines

### When Working with This Codebase

1. **Preserve Vulkan Naming**: Keep Vulkan constant names exactly as specified in Vulkan spec
2. **Type Safety**: Always maintain proper type conversions between C and Go
3. **Error Handling**: Every Vulkan function should check result codes and return Go errors
4. **Documentation**: Add comments explaining Vulkan concepts for Go developers unfamiliar with the API
5. **Platform Compatibility**: Ensure CGo directives support Windows, Linux, and macOS

### Adding New Vulkan Functions

When implementing new Vulkan API functions:

1. Add appropriate `#cgo` directives if needed
2. Include necessary Vulkan headers
3. Create Go wrapper function with idiomatic Go signature
4. Convert C types to Go types appropriately
5. Check and convert Vulkan result codes to Go errors
6. Add example usage to `examples/` directory
7. Consider organizing related functions into logical files (e.g., device.go, command.go)

### Common Patterns to Follow

```go
// Pattern for Vulkan function wrapper
func VulkanFunctionName(param Type) (ReturnType, error) {
    var cResult C.TypeName
    result := C.vkFunctionName(C.TypeName(param), &cResult)

    if result != C.VK_SUCCESS {
        return ZeroValue, Result(result)
    }

    return ConvertToGoType(cResult), nil
}
```

### Testing Considerations

1. **System Requirements**: Tests require Vulkan-capable hardware/drivers
2. **Mock Considerations**: Some tests may need mock Vulkan implementations
3. **Example Code**: Examples serve as integration tests
4. **CI/CD**: May need special runners with Vulkan support

### Security Considerations

1. **CGo Safety**: Be cautious with pointer conversions and memory management
2. **Input Validation**: Validate inputs before passing to C functions
3. **Resource Cleanup**: Implement proper cleanup/destroy functions for Vulkan resources
4. **Error Propagation**: Never silently ignore Vulkan errors

## Current State and Roadmap

### Current Implementation (v0.x - Early Development)

- ✅ Basic Result type with all Vulkan result codes
- ✅ EnumerateInstanceVersion function
- ✅ CGo integration with platform-specific linking
- ✅ Basic example demonstrating usage

### Potential Future Additions

- Instance creation and management
- Physical device enumeration and selection
- Logical device creation
- Command buffers and queues
- Render passes and pipelines
- Memory management
- Synchronization primitives
- Swapchain support
- Extension support
- Validation layers integration

## Quick Reference

### Building Examples
```bash
cd examples
go run test.go
```

### Importing the Library
```go
import vk "github.com/NOT-REAL-GAMES/vulkango"
```

### Basic Usage Pattern
```go
version, err := vk.EnumerateInstanceVersion()
if err != nil {
    // Handle error - err will be a vk.Result type
    log.Fatal(err)
}
fmt.Printf("Vulkan version: %d\n", version)
```

## Notes for AI Assistants

- This is a low-level library requiring knowledge of Vulkan API
- Vulkan is complex; small changes can have significant implications
- Always test with examples when adding new functionality
- Maintain compatibility with the Vulkan specification
- Consider Go idioms while staying true to Vulkan's C API design
- The project is in early stages; architectural decisions should be discussed
- Performance is critical for graphics applications
- Memory safety is paramount when working with CGo

---

*Last Updated: 2025-11-13*
*Generated for AI assistants working with the vulkango codebase*
