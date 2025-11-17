package ecs

import (
	"fmt"
)

// Entity is a unique identifier for an entity in the ECS world.
// It's just a number - all the data lives in component maps.
type Entity uint64

// World manages all entities and their components.
// It's the central registry for the ECS system.
type World struct {
	nextEntity Entity

	// Component storage - one map per component type
	transforms      map[Entity]*Transform
	renderTargets   map[Entity]*RenderTarget
	vulkanPipelines map[Entity]*VulkanPipeline
	blendModes      map[Entity]*BlendMode
	textureData     map[Entity]*TextureData
	texts           map[Entity]*Text
	uiButtons       map[Entity]*UIButton

	// Track all living entities for iteration
	entities map[Entity]bool
}

// NewWorld creates a new ECS world.
func NewWorld() *World {
	return &World{
		nextEntity:      1, // Start at 1, 0 is invalid entity
		transforms:      make(map[Entity]*Transform),
		renderTargets:   make(map[Entity]*RenderTarget),
		vulkanPipelines: make(map[Entity]*VulkanPipeline),
		blendModes:      make(map[Entity]*BlendMode),
		textureData:     make(map[Entity]*TextureData),
		texts:           make(map[Entity]*Text),
		uiButtons:       make(map[Entity]*UIButton),
		entities:        make(map[Entity]bool),
	}
}

// CreateEntity creates a new entity and returns its ID.
// The entity starts with no components - add them separately.
func (w *World) CreateEntity() Entity {
	entity := w.nextEntity
	w.nextEntity++
	w.entities[entity] = true
	return entity
}

// DeleteEntity removes an entity and all its components.
// Note: This doesn't call Vulkan cleanup functions - caller must handle that.
func (w *World) DeleteEntity(entity Entity) {
	delete(w.entities, entity)
	delete(w.transforms, entity)
	delete(w.renderTargets, entity)
	delete(w.vulkanPipelines, entity)
	delete(w.blendModes, entity)
	delete(w.textureData, entity)
}

// EntityExists checks if an entity ID is valid and alive.
func (w *World) EntityExists(entity Entity) bool {
	return w.entities[entity]
}

// Entities returns a slice of all living entity IDs.
// Useful for iteration.
func (w *World) Entities() []Entity {
	result := make([]Entity, 0, len(w.entities))
	for e := range w.entities {
		result = append(result, e)
	}
	return result
}

// EntityCount returns the number of living entities.
func (w *World) EntityCount() int {
	return len(w.entities)
}

// --- Component Add/Remove/Get Methods ---

// AddTransform adds a Transform component to an entity.
func (w *World) AddTransform(entity Entity, transform *Transform) {
	if !w.EntityExists(entity) {
		panic(fmt.Sprintf("entity %d does not exist", entity))
	}
	w.transforms[entity] = transform
}

// GetTransform retrieves the Transform component for an entity.
// Returns nil if the entity doesn't have this component.
func (w *World) GetTransform(entity Entity) *Transform {
	return w.transforms[entity]
}

// RemoveTransform removes the Transform component from an entity.
func (w *World) RemoveTransform(entity Entity) {
	delete(w.transforms, entity)
}

// HasTransform checks if an entity has a Transform component.
func (w *World) HasTransform(entity Entity) bool {
	_, exists := w.transforms[entity]
	return exists
}

// AddRenderTarget adds a RenderTarget component to an entity.
func (w *World) AddRenderTarget(entity Entity, target *RenderTarget) {
	if !w.EntityExists(entity) {
		panic(fmt.Sprintf("entity %d does not exist", entity))
	}
	w.renderTargets[entity] = target
}

// GetRenderTarget retrieves the RenderTarget component for an entity.
func (w *World) GetRenderTarget(entity Entity) *RenderTarget {
	return w.renderTargets[entity]
}

// RemoveRenderTarget removes the RenderTarget component from an entity.
func (w *World) RemoveRenderTarget(entity Entity) {
	delete(w.renderTargets, entity)
}

// HasRenderTarget checks if an entity has a RenderTarget component.
func (w *World) HasRenderTarget(entity Entity) bool {
	_, exists := w.renderTargets[entity]
	return exists
}

// AddVulkanPipeline adds a VulkanPipeline component to an entity.
func (w *World) AddVulkanPipeline(entity Entity, pipeline *VulkanPipeline) {
	if !w.EntityExists(entity) {
		panic(fmt.Sprintf("entity %d does not exist", entity))
	}
	w.vulkanPipelines[entity] = pipeline
}

// GetVulkanPipeline retrieves the VulkanPipeline component for an entity.
func (w *World) GetVulkanPipeline(entity Entity) *VulkanPipeline {
	return w.vulkanPipelines[entity]
}

// RemoveVulkanPipeline removes the VulkanPipeline component from an entity.
func (w *World) RemoveVulkanPipeline(entity Entity) {
	delete(w.vulkanPipelines, entity)
}

// HasVulkanPipeline checks if an entity has a VulkanPipeline component.
func (w *World) HasVulkanPipeline(entity Entity) bool {
	_, exists := w.vulkanPipelines[entity]
	return exists
}

// AddBlendMode adds a BlendMode component to an entity.
func (w *World) AddBlendMode(entity Entity, blend *BlendMode) {
	if !w.EntityExists(entity) {
		panic(fmt.Sprintf("entity %d does not exist", entity))
	}
	w.blendModes[entity] = blend
}

// GetBlendMode retrieves the BlendMode component for an entity.
func (w *World) GetBlendMode(entity Entity) *BlendMode {
	return w.blendModes[entity]
}

// RemoveBlendMode removes the BlendMode component from an entity.
func (w *World) RemoveBlendMode(entity Entity) {
	delete(w.blendModes, entity)
}

// HasBlendMode checks if an entity has a BlendMode component.
func (w *World) HasBlendMode(entity Entity) bool {
	_, exists := w.blendModes[entity]
	return exists
}

// AddTextureData adds a TextureData component to an entity.
func (w *World) AddTextureData(entity Entity, texture *TextureData) {
	if !w.EntityExists(entity) {
		panic(fmt.Sprintf("entity %d does not exist", entity))
	}
	w.textureData[entity] = texture
}

// GetTextureData retrieves the TextureData component for an entity.
func (w *World) GetTextureData(entity Entity) *TextureData {
	return w.textureData[entity]
}

// RemoveTextureData removes the TextureData component from an entity.
func (w *World) RemoveTextureData(entity Entity) {
	delete(w.textureData, entity)
}

// HasTextureData checks if an entity has a TextureData component.
func (w *World) HasTextureData(entity Entity) bool {
	_, exists := w.textureData[entity]
	return exists
}

// ===== Text Component =====

func (w *World) AddText(e Entity, t *Text) {
	w.texts[e] = t
}

func (w *World) GetText(e Entity) *Text {
	return w.texts[e]
}

func (w *World) HasText(e Entity) bool {
	_, exists := w.texts[e]
	return exists
}

func (w *World) RemoveText(e Entity) {
	delete(w.texts, e)
}

// ===== UIButton Component =====

func (w *World) AddUIButton(e Entity, b *UIButton) {
	w.uiButtons[e] = b
}

func (w *World) GetUIButton(e Entity) *UIButton {
	return w.uiButtons[e]
}

func (w *World) HasUIButton(e Entity) bool {
	_, exists := w.uiButtons[e]
	return exists
}

func (w *World) RemoveUIButton(e Entity) {
	delete(w.uiButtons, e)
}
