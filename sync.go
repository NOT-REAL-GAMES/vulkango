// sync.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

type Semaphore struct {
	handle C.VkSemaphore
}

type Fence struct {
	handle C.VkFence
}

type SemaphoreCreateInfo struct {
	Flags uint32
}

type FenceCreateInfo struct {
	Flags FenceCreateFlags
}

type FenceCreateFlags uint32

const (
	FENCE_CREATE_SIGNALED_BIT FenceCreateFlags = C.VK_FENCE_CREATE_SIGNALED_BIT
)

// Semaphore
func (device Device) CreateSemaphore(createInfo *SemaphoreCreateInfo) (Semaphore, error) {
	cInfo := (*C.VkSemaphoreCreateInfo)(C.calloc(1, C.sizeof_VkSemaphoreCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkSemaphoreCreateFlags(createInfo.Flags)

	var semaphore C.VkSemaphore
	result := C.vkCreateSemaphore(device.handle, cInfo, nil, &semaphore)

	if result != C.VK_SUCCESS {
		return Semaphore{}, Result(result)
	}

	return Semaphore{handle: semaphore}, nil
}

func (device Device) DestroySemaphore(semaphore Semaphore) {
	C.vkDestroySemaphore(device.handle, semaphore.handle, nil)
}

// Fence
func (device Device) CreateFence(createInfo *FenceCreateInfo) (Fence, error) {
	cInfo := (*C.VkFenceCreateInfo)(C.calloc(1, C.sizeof_VkFenceCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = C.VkFenceCreateFlags(createInfo.Flags)

	var fence C.VkFence
	result := C.vkCreateFence(device.handle, cInfo, nil, &fence)

	if result != C.VK_SUCCESS {
		return Fence{}, Result(result)
	}

	return Fence{handle: fence}, nil
}

func (device Device) DestroyFence(fence Fence) {
	C.vkDestroyFence(device.handle, fence.handle, nil)
}

func (device Device) WaitForFences(fences []Fence, waitAll bool, timeout uint64) error {
	if len(fences) == 0 {
		return nil
	}

	cFences := make([]C.VkFence, len(fences))
	for i, fence := range fences {
		cFences[i] = fence.handle
	}

	var cWaitAll C.VkBool32
	if waitAll {
		cWaitAll = C.VK_TRUE
	} else {
		cWaitAll = C.VK_FALSE
	}

	result := C.vkWaitForFences(device.handle, C.uint32_t(len(cFences)), &cFences[0], cWaitAll, C.uint64_t(timeout))

	if result != C.VK_SUCCESS && result != C.VK_TIMEOUT {
		return Result(result)
	}

	return nil
}

func (device Device) ResetFences(fences []Fence) error {
	if len(fences) == 0 {
		return nil
	}

	cFences := make([]C.VkFence, len(fences))
	for i, fence := range fences {
		cFences[i] = fence.handle
	}

	result := C.vkResetFences(device.handle, C.uint32_t(len(cFences)), &cFences[0])

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	return nil
}

// Queue Operations
type SubmitInfo struct {
	WaitSemaphores   []Semaphore
	WaitDstStageMask []PipelineStageFlags
	CommandBuffers   []CommandBuffer
	SignalSemaphores []Semaphore
}

func (queue Queue) Submit(submits []SubmitInfo, fence Fence) error {
	if len(submits) == 0 {
		return nil
	}

	// Allocate C memory for submit infos
	cSubmits := (*[1 << 30]C.VkSubmitInfo)(C.calloc(C.size_t(len(submits)), C.sizeof_VkSubmitInfo))[:len(submits):len(submits)]
	defer C.free(unsafe.Pointer(&cSubmits[0]))

	// Track all C allocations for cleanup
	var allocations []unsafe.Pointer
	defer func() {
		for _, ptr := range allocations {
			C.free(ptr)
		}
	}()

	for i, submit := range submits {
		cSubmits[i].sType = C.VK_STRUCTURE_TYPE_SUBMIT_INFO
		cSubmits[i].pNext = nil

		// Wait semaphores
		if len(submit.WaitSemaphores) > 0 {
			waitSems := (*[1 << 30]C.VkSemaphore)(C.calloc(C.size_t(len(submit.WaitSemaphores)), C.sizeof_VkSemaphore))[:len(submit.WaitSemaphores):len(submit.WaitSemaphores)]
			waitStgs := (*[1 << 30]C.VkPipelineStageFlags)(C.calloc(C.size_t(len(submit.WaitDstStageMask)), C.sizeof_VkPipelineStageFlags))[:len(submit.WaitDstStageMask):len(submit.WaitDstStageMask)]
			allocations = append(allocations, unsafe.Pointer(&waitSems[0]), unsafe.Pointer(&waitStgs[0]))

			for j, sem := range submit.WaitSemaphores {
				waitSems[j] = sem.handle
			}
			for j, stage := range submit.WaitDstStageMask {
				waitStgs[j] = C.VkPipelineStageFlags(stage)
			}

			cSubmits[i].waitSemaphoreCount = C.uint32_t(len(waitSems))
			cSubmits[i].pWaitSemaphores = &waitSems[0]
			cSubmits[i].pWaitDstStageMask = &waitStgs[0]
		}

		// Command buffers
		if len(submit.CommandBuffers) > 0 {
			cmdBufs := (*[1 << 30]C.VkCommandBuffer)(C.calloc(C.size_t(len(submit.CommandBuffers)), C.sizeof_VkCommandBuffer))[:len(submit.CommandBuffers):len(submit.CommandBuffers)]
			allocations = append(allocations, unsafe.Pointer(&cmdBufs[0]))

			for j, cmd := range submit.CommandBuffers {
				cmdBufs[j] = cmd.handle
			}

			cSubmits[i].commandBufferCount = C.uint32_t(len(cmdBufs))
			cSubmits[i].pCommandBuffers = &cmdBufs[0]
		}

		// Signal semaphores
		if len(submit.SignalSemaphores) > 0 {
			sigSems := (*[1 << 30]C.VkSemaphore)(C.calloc(C.size_t(len(submit.SignalSemaphores)), C.sizeof_VkSemaphore))[:len(submit.SignalSemaphores):len(submit.SignalSemaphores)]
			allocations = append(allocations, unsafe.Pointer(&sigSems[0]))

			for j, sem := range submit.SignalSemaphores {
				sigSems[j] = sem.handle
			}

			cSubmits[i].signalSemaphoreCount = C.uint32_t(len(sigSems))
			cSubmits[i].pSignalSemaphores = &sigSems[0]
		}
	}

	var cFence C.VkFence
	if fence.handle != nil {
		cFence = fence.handle
	}

	result := C.vkQueueSubmit(queue.handle, C.uint32_t(len(cSubmits)), &cSubmits[0], cFence)

	if result != C.VK_SUCCESS {
		return Result(result)
	}

	return nil
}
func (queue Queue) WaitIdle() error {
	result := C.vkQueueWaitIdle(queue.handle)
	if result != C.VK_SUCCESS {
		return Result(result)
	}
	return nil
}

// Swapchain Present
type PresentInfoKHR struct {
	WaitSemaphores []Semaphore
	Swapchains     []SwapchainKHR
	ImageIndices   []uint32
}

func (queue Queue) PresentKHR(presentInfo *PresentInfoKHR) error {
	cInfo := (*C.VkPresentInfoKHR)(C.calloc(1, C.sizeof_VkPresentInfoKHR))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR
	cInfo.pNext = nil

	var waitSemaphores []C.VkSemaphore
	if len(presentInfo.WaitSemaphores) > 0 {
		waitSemaphores = make([]C.VkSemaphore, len(presentInfo.WaitSemaphores))
		for i, sem := range presentInfo.WaitSemaphores {
			waitSemaphores[i] = sem.handle
		}
		cInfo.waitSemaphoreCount = C.uint32_t(len(waitSemaphores))
		cInfo.pWaitSemaphores = &waitSemaphores[0]
	}

	var swapchains []C.VkSwapchainKHR
	if len(presentInfo.Swapchains) > 0 {
		swapchains = make([]C.VkSwapchainKHR, len(presentInfo.Swapchains))
		for i, sc := range presentInfo.Swapchains {
			swapchains[i] = sc.handle
		}
		cInfo.swapchainCount = C.uint32_t(len(swapchains))
		cInfo.pSwapchains = &swapchains[0]
	}

	var imageIndices []C.uint32_t
	if len(presentInfo.ImageIndices) > 0 {
		imageIndices = make([]C.uint32_t, len(presentInfo.ImageIndices))
		for i, idx := range presentInfo.ImageIndices {
			imageIndices[i] = C.uint32_t(idx)
		}
		cInfo.pImageIndices = &imageIndices[0]
	}

	cInfo.pResults = nil

	result := C.vkQueuePresentKHR(queue.handle, cInfo)

	if result != C.VK_SUCCESS && result != C.VK_SUBOPTIMAL_KHR {
		return Result(result)
	}

	return nil
}

// Swapchain Image Acquisition
func (device Device) AcquireNextImageKHR(swapchain SwapchainKHR, timeout uint64, semaphore Semaphore, fence Fence) (uint32, error) {
	var imageIndex C.uint32_t

	var cSemaphore C.VkSemaphore
	if semaphore.handle != nil {
		cSemaphore = semaphore.handle
	}

	var cFence C.VkFence
	if fence.handle != nil {
		cFence = fence.handle
	}

	result := C.vkAcquireNextImageKHR(device.handle, swapchain.handle, C.uint64_t(timeout), cSemaphore, cFence, &imageIndex)

	if result != C.VK_SUCCESS && result != C.VK_SUBOPTIMAL_KHR {
		return 0, Result(result)
	}

	return uint32(imageIndex), nil
}
