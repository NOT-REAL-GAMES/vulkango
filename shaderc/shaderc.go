package shaderc

/*
#cgo pkg-config: shaderc
#include <shaderc/shaderc.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Compiler struct {
	handle C.shaderc_compiler_t
}

type CompileOptions struct {
	handle C.shaderc_compile_options_t
}

type ShaderKind int

const (
	VertexShader   ShaderKind = C.shaderc_vertex_shader
	FragmentShader ShaderKind = C.shaderc_fragment_shader
	ComputeShader  ShaderKind = C.shaderc_compute_shader
)

type CompilationResult struct {
	handle C.shaderc_compilation_result_t
}

func NewCompiler() Compiler {
	return Compiler{handle: C.shaderc_compiler_initialize()}
}

func (c Compiler) Release() {
	C.shaderc_compiler_release(c.handle)
}

func NewCompileOptions() CompileOptions {
	return CompileOptions{handle: C.shaderc_compile_options_initialize()}
}

func (o CompileOptions) Release() {
	C.shaderc_compile_options_release(o.handle)
}

func (o CompileOptions) SetTargetEnv(env int, version uint32) {
	C.shaderc_compile_options_set_target_env(
		o.handle,
		C.shaderc_target_env(env),
		C.uint32_t(version),
	)
}

func (o CompileOptions) SetOptimizationLevel(level int) {
	C.shaderc_compile_options_set_optimization_level(
		o.handle,
		C.shaderc_optimization_level(level),
	)
}

const (
	TargetEnvVulkan              = C.shaderc_target_env_vulkan
	EnvVersionVulkan_1_3         = C.shaderc_env_version_vulkan_1_3
	OptimizationLevelPerformance = C.shaderc_optimization_level_performance
)

func (c Compiler) CompileIntoSPV(source, filename string, kind ShaderKind, options CompileOptions) (CompilationResult, error) {
	cSource := C.CString(source)
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cSource))
	defer C.free(unsafe.Pointer(cFilename))

	result := C.shaderc_compile_into_spv(
		c.handle,
		cSource,
		C.size_t(len(source)),
		C.shaderc_shader_kind(kind),
		cFilename,
		(*C.char)(unsafe.Pointer(C.CString("main"))),
		options.handle,
	)

	status := C.shaderc_result_get_compilation_status(result)
	if status != C.shaderc_compilation_status_success {
		errorMsg := C.GoString(C.shaderc_result_get_error_message(result))
		C.shaderc_result_release(result)
		return CompilationResult{}, fmt.Errorf("shader compilation failed: %s", errorMsg)
	}

	return CompilationResult{handle: result}, nil
}

func (r CompilationResult) GetBytes() []byte {
	ptr := C.shaderc_result_get_bytes(r.handle)
	length := C.shaderc_result_get_length(r.handle)

	// Convert to Go slice
	return C.GoBytes(unsafe.Pointer(ptr), C.int(length))
}

func (r CompilationResult) Release() {
	C.shaderc_result_release(r.handle)
}
