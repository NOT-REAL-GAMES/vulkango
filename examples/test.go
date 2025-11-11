package main

import (
	"fmt"

	vk "github.com/NOT-REAL-GAMES/vulkango"
)

func main() {
	version, err := vk.EnumerateInstanceVersion()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Vulkan version: %d\n", version)
}
