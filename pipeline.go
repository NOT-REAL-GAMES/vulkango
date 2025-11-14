// pipeline.go
package vulkango

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import "unsafe"

// Pipeline Layout
func (device Device) CreatePipelineLayout(createInfo *PipelineLayoutCreateInfo) (PipelineLayout, error) {
	cInfo := (*C.VkPipelineLayoutCreateInfo)(C.calloc(1, C.sizeof_VkPipelineLayoutCreateInfo))
	defer C.free(unsafe.Pointer(cInfo))

	cInfo.sType = C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO
	cInfo.pNext = nil
	cInfo.flags = 0
	cInfo.setLayoutCount = 0
	cInfo.pSetLayouts = nil
	cInfo.pushConstantRangeCount = 0
	cInfo.pPushConstantRanges = nil

	var layout C.VkPipelineLayout
	result := C.vkCreatePipelineLayout(device.handle, cInfo, nil, &layout)

	if result != C.VK_SUCCESS {
		return PipelineLayout{}, Result(result)
	}

	return PipelineLayout{handle: layout}, nil
}

func (device Device) DestroyPipelineLayout(layout PipelineLayout) {
	C.vkDestroyPipelineLayout(device.handle, layout.handle, nil)
}

func (device Device) DestroyPipeline(pipeline Pipeline) {
	C.vkDestroyPipeline(device.handle, pipeline.handle, nil)
}

// Graphics Pipeline
type graphicsPipelineData struct {
	cInfo                 *C.VkGraphicsPipelineCreateInfo
	shaderStages          []C.VkPipelineShaderStageCreateInfo
	shaderEntryNames      []*C.char
	vertexInputState      *C.VkPipelineVertexInputStateCreateInfo
	inputAssemblyState    *C.VkPipelineInputAssemblyStateCreateInfo
	viewportState         *C.VkPipelineViewportStateCreateInfo
	viewports             []C.VkViewport
	scissors              []C.VkRect2D
	rasterizationState    *C.VkPipelineRasterizationStateCreateInfo
	multisampleState      *C.VkPipelineMultisampleStateCreateInfo
	colorBlendState       *C.VkPipelineColorBlendStateCreateInfo
	colorBlendAttachments []C.VkPipelineColorBlendAttachmentState
	dynamicState          *C.VkPipelineDynamicStateCreateInfo
	dynamicStates         []C.VkDynamicState
	renderingInfo         *C.VkPipelineRenderingCreateInfo
	colorFormats          []C.VkFormat
}

func (info *GraphicsPipelineCreateInfo) vulkanize() *graphicsPipelineData {
	data := &graphicsPipelineData{}

	// Main create info
	data.cInfo = (*C.VkGraphicsPipelineCreateInfo)(C.calloc(1, C.sizeof_VkGraphicsPipelineCreateInfo))
	data.cInfo.sType = C.VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO
	data.cInfo.pNext = nil
	data.cInfo.flags = 0

	// Shader stages
	if len(info.Stages) > 0 {
		data.shaderStages = make([]C.VkPipelineShaderStageCreateInfo, len(info.Stages))
		data.shaderEntryNames = make([]*C.char, len(info.Stages))

		for i, stage := range info.Stages {
			data.shaderStages[i].sType = C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO
			data.shaderStages[i].pNext = nil
			data.shaderStages[i].flags = 0
			data.shaderStages[i].stage = C.VkShaderStageFlagBits(stage.Stage)
			data.shaderStages[i].module = stage.Module.handle
			data.shaderEntryNames[i] = C.CString(stage.Name)
			data.shaderStages[i].pName = data.shaderEntryNames[i]
			data.shaderStages[i].pSpecializationInfo = nil
		}

		data.cInfo.stageCount = C.uint32_t(len(data.shaderStages))
		data.cInfo.pStages = &data.shaderStages[0]
	}

	// Vertex input state
	if info.VertexInputState != nil {
		data.vertexInputState = (*C.VkPipelineVertexInputStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineVertexInputStateCreateInfo))
		data.vertexInputState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO
		data.vertexInputState.pNext = nil
		data.vertexInputState.flags = 0
		data.vertexInputState.vertexBindingDescriptionCount = 0
		data.vertexInputState.pVertexBindingDescriptions = nil
		data.vertexInputState.vertexAttributeDescriptionCount = 0
		data.vertexInputState.pVertexAttributeDescriptions = nil
		data.cInfo.pVertexInputState = data.vertexInputState
	}

	// Input assembly state
	if info.InputAssemblyState != nil {
		data.inputAssemblyState = (*C.VkPipelineInputAssemblyStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineInputAssemblyStateCreateInfo))
		data.inputAssemblyState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO
		data.inputAssemblyState.pNext = nil
		data.inputAssemblyState.flags = 0
		data.inputAssemblyState.topology = C.VkPrimitiveTopology(info.InputAssemblyState.Topology)
		if info.InputAssemblyState.PrimitiveRestartEnable {
			data.inputAssemblyState.primitiveRestartEnable = C.VK_TRUE
		} else {
			data.inputAssemblyState.primitiveRestartEnable = C.VK_FALSE
		}
		data.cInfo.pInputAssemblyState = data.inputAssemblyState
	}

	// Viewport state
	if info.ViewportState != nil {
		data.viewportState = (*C.VkPipelineViewportStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineViewportStateCreateInfo))
		data.viewportState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO
		data.viewportState.pNext = nil
		data.viewportState.flags = 0

		if len(info.ViewportState.Viewports) > 0 {
			data.viewports = make([]C.VkViewport, len(info.ViewportState.Viewports))
			for i, vp := range info.ViewportState.Viewports {
				data.viewports[i].x = C.float(vp.X)
				data.viewports[i].y = C.float(vp.Y)
				data.viewports[i].width = C.float(vp.Width)
				data.viewports[i].height = C.float(vp.Height)
				data.viewports[i].minDepth = C.float(vp.MinDepth)
				data.viewports[i].maxDepth = C.float(vp.MaxDepth)
			}
			data.viewportState.viewportCount = C.uint32_t(len(data.viewports))
			data.viewportState.pViewports = &data.viewports[0]
		} else {
			data.viewportState.viewportCount = 1
			data.viewportState.pViewports = nil
		}

		if len(info.ViewportState.Scissors) > 0 {
			data.scissors = make([]C.VkRect2D, len(info.ViewportState.Scissors))
			for i, sc := range info.ViewportState.Scissors {
				data.scissors[i].offset.x = C.int32_t(sc.Offset.X)
				data.scissors[i].offset.y = C.int32_t(sc.Offset.Y)
				data.scissors[i].extent.width = C.uint32_t(sc.Extent.Width)
				data.scissors[i].extent.height = C.uint32_t(sc.Extent.Height)
			}
			data.viewportState.scissorCount = C.uint32_t(len(data.scissors))
			data.viewportState.pScissors = &data.scissors[0]
		} else {
			data.viewportState.scissorCount = 1
			data.viewportState.pScissors = nil
		}

		data.cInfo.pViewportState = data.viewportState
	}

	// Rasterization state
	if info.RasterizationState != nil {
		data.rasterizationState = (*C.VkPipelineRasterizationStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineRasterizationStateCreateInfo))
		data.rasterizationState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO
		data.rasterizationState.pNext = nil
		data.rasterizationState.flags = 0
		data.rasterizationState.depthClampEnable = C.VK_FALSE
		data.rasterizationState.rasterizerDiscardEnable = C.VK_FALSE
		data.rasterizationState.polygonMode = C.VkPolygonMode(info.RasterizationState.PolygonMode)
		data.rasterizationState.cullMode = C.VkCullModeFlags(info.RasterizationState.CullMode)
		data.rasterizationState.frontFace = C.VkFrontFace(info.RasterizationState.FrontFace)
		data.rasterizationState.depthBiasEnable = C.VK_FALSE
		data.rasterizationState.lineWidth = C.float(info.RasterizationState.LineWidth)
		data.cInfo.pRasterizationState = data.rasterizationState
	}

	// Multisample state
	if info.MultisampleState != nil {
		data.multisampleState = (*C.VkPipelineMultisampleStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineMultisampleStateCreateInfo))
		data.multisampleState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO
		data.multisampleState.pNext = nil
		data.multisampleState.flags = 0
		data.multisampleState.rasterizationSamples = C.VkSampleCountFlagBits(info.MultisampleState.RasterizationSamples)
		data.multisampleState.sampleShadingEnable = C.VK_FALSE
		data.multisampleState.pSampleMask = nil
		data.multisampleState.alphaToCoverageEnable = C.VK_FALSE
		data.multisampleState.alphaToOneEnable = C.VK_FALSE
		data.cInfo.pMultisampleState = data.multisampleState
	}

	// Color blend state
	if info.ColorBlendState != nil {
		data.colorBlendState = (*C.VkPipelineColorBlendStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineColorBlendStateCreateInfo))
		data.colorBlendState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO
		data.colorBlendState.pNext = nil
		data.colorBlendState.flags = 0
		data.colorBlendState.logicOpEnable = C.VK_FALSE
		data.colorBlendState.logicOp = C.VK_LOGIC_OP_COPY

		if len(info.ColorBlendState.Attachments) > 0 {
			data.colorBlendAttachments = make([]C.VkPipelineColorBlendAttachmentState, len(info.ColorBlendState.Attachments))
			for i, att := range info.ColorBlendState.Attachments {
				if att.BlendEnable {
					data.colorBlendAttachments[i].blendEnable = C.VK_TRUE
				} else {
					data.colorBlendAttachments[i].blendEnable = C.VK_FALSE
				}
				data.colorBlendAttachments[i].colorWriteMask = C.VkColorComponentFlags(att.ColorWriteMask)
			}
			data.colorBlendState.attachmentCount = C.uint32_t(len(data.colorBlendAttachments))
			data.colorBlendState.pAttachments = &data.colorBlendAttachments[0]
		}

		data.cInfo.pColorBlendState = data.colorBlendState
	}

	// Dynamic state
	if info.DynamicState != nil && len(info.DynamicState.DynamicStates) > 0 {
		data.dynamicState = (*C.VkPipelineDynamicStateCreateInfo)(C.calloc(1, C.sizeof_VkPipelineDynamicStateCreateInfo))
		data.dynamicState.sType = C.VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO
		data.dynamicState.pNext = nil
		data.dynamicState.flags = 0

		data.dynamicStates = make([]C.VkDynamicState, len(info.DynamicState.DynamicStates))
		for i, state := range info.DynamicState.DynamicStates {
			data.dynamicStates[i] = C.VkDynamicState(state)
		}
		data.dynamicState.dynamicStateCount = C.uint32_t(len(data.dynamicStates))
		data.dynamicState.pDynamicStates = &data.dynamicStates[0]
		data.cInfo.pDynamicState = data.dynamicState
	}

	// Pipeline rendering create info (for dynamic rendering)
	if info.RenderingInfo != nil {
		data.renderingInfo = (*C.VkPipelineRenderingCreateInfo)(C.calloc(1, C.sizeof_VkPipelineRenderingCreateInfo))
		data.renderingInfo.sType = C.VK_STRUCTURE_TYPE_PIPELINE_RENDERING_CREATE_INFO
		data.renderingInfo.pNext = nil

		if len(info.RenderingInfo.ColorAttachmentFormats) > 0 {
			data.colorFormats = make([]C.VkFormat, len(info.RenderingInfo.ColorAttachmentFormats))
			for i, fmt := range info.RenderingInfo.ColorAttachmentFormats {
				data.colorFormats[i] = C.VkFormat(fmt)
			}
			data.renderingInfo.colorAttachmentCount = C.uint32_t(len(data.colorFormats))
			data.renderingInfo.pColorAttachmentFormats = &data.colorFormats[0]
		}

		data.renderingInfo.depthAttachmentFormat = C.VK_FORMAT_UNDEFINED
		data.renderingInfo.stencilAttachmentFormat = C.VK_FORMAT_UNDEFINED

		// Chain it to main create info
		data.cInfo.pNext = unsafe.Pointer(data.renderingInfo)
	}

	// Layout
	data.cInfo.layout = info.Layout.handle
	data.cInfo.renderPass = nil // Must be NULL for dynamic rendering
	data.cInfo.subpass = 0
	data.cInfo.basePipelineHandle = nil
	data.cInfo.basePipelineIndex = -1

	return data
}

func (data *graphicsPipelineData) free() {
	for _, name := range data.shaderEntryNames {
		C.free(unsafe.Pointer(name))
	}

	if data.vertexInputState != nil {
		C.free(unsafe.Pointer(data.vertexInputState))
	}
	if data.inputAssemblyState != nil {
		C.free(unsafe.Pointer(data.inputAssemblyState))
	}
	if data.viewportState != nil {
		C.free(unsafe.Pointer(data.viewportState))
	}
	if data.rasterizationState != nil {
		C.free(unsafe.Pointer(data.rasterizationState))
	}
	if data.multisampleState != nil {
		C.free(unsafe.Pointer(data.multisampleState))
	}
	if data.colorBlendState != nil {
		C.free(unsafe.Pointer(data.colorBlendState))
	}
	if data.dynamicState != nil {
		C.free(unsafe.Pointer(data.dynamicState))
	}
	if data.renderingInfo != nil {
		C.free(unsafe.Pointer(data.renderingInfo))
	}
	if data.cInfo != nil {
		C.free(unsafe.Pointer(data.cInfo))
	}
}

func (device Device) CreateGraphicsPipeline(createInfo *GraphicsPipelineCreateInfo) (Pipeline, error) {
	data := createInfo.vulkanize()
	defer data.free()

	var pipeline C.VkPipeline
	result := C.vkCreateGraphicsPipelines(device.handle, nil, 1, data.cInfo, nil, &pipeline)

	if result != C.VK_SUCCESS {
		return Pipeline{}, Result(result)
	}

	return Pipeline{handle: pipeline}, nil
}
