package main

import (
	"fmt"

	sdl "github.com/NOT-REAL-GAMES/sdl3go"
	vk "github.com/NOT-REAL-GAMES/vulkango"
)

func main() {
	// Initialize SDL first
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("SDL3 + Vulkan", 960, 540, sdl.WINDOW_VULKAN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// Get required extensions from SDL BEFORE creating instance
	exts, err := sdl.VulkanGetInstanceExtensions()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Required Vulkan extensions: %v\n", exts)

	// Create Vulkan instance WITH the required extensions
	version, _ := vk.EnumerateInstanceVersion()
	fmt.Printf("Vulkan %d.%d.%d\n",
		vk.ApiVersionMajor(version),
		vk.ApiVersionMinor(version),
		vk.ApiVersionPatch(version))

	instance, err := vk.CreateInstance(&vk.InstanceCreateInfo{
		ApplicationInfo: &vk.ApplicationInfo{
			ApplicationName:    "VulkanGo Test",
			ApplicationVersion: vk.MakeApiVersion(0, 1, 0, 0),
			EngineName:         "No Engine",
			EngineVersion:      vk.MakeApiVersion(0, 1, 0, 0),
			ApiVersion:         vk.ApiVersion_1_4,
		},
		EnabledExtensionNames: exts,
	})
	if err != nil {
		panic(err)
	}
	defer instance.Destroy()

	devices, _ := instance.EnumeratePhysicalDevices()
	fmt.Printf("Found %d physical devices\n", len(devices))

	// Create the surface
	surf, err := window.VulkanCreateSurface(instance.Handle())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created Vulkan surface: %v\n", surf)

	// Event loop - keeps window open until you close it
	fmt.Println("Window open - close it to exit")
	running := true
	for running {
		for event, ok := sdl.PollEvent(); ok; event, ok = sdl.PollEvent() {
			if event.Type == sdl.EVENT_QUIT {
				running = false
			}
		}
	}
}
