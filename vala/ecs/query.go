package ecs

// Query holds the result of a component query.
// It's an iterator-like structure for accessing entities with specific components.
type Query struct {
	entities []Entity
	world    *World
}

// QueryWithTransform returns all entities that have a Transform component.
func (w *World) QueryWithTransform() *Query {
	entities := make([]Entity, 0)
	for entity := range w.transforms {
		if w.EntityExists(entity) {
			entities = append(entities, entity)
		}
	}
	return &Query{entities: entities, world: w}
}

// QueryWithRenderTarget returns all entities that have a RenderTarget component.
func (w *World) QueryWithRenderTarget() *Query {
	entities := make([]Entity, 0)
	for entity := range w.renderTargets {
		if w.EntityExists(entity) {
			entities = append(entities, entity)
		}
	}
	return &Query{entities: entities, world: w}
}

// QueryRenderables returns entities that have all components needed for rendering:
// Transform, VulkanPipeline, and TextureData.
func (w *World) QueryRenderables() []Entity {
	result := make([]Entity, 0)

	for entity := range w.entities {
		if w.HasTransform(entity) &&
		   w.HasVulkanPipeline(entity) &&
		   w.HasTextureData(entity) {
			result = append(result, entity)
		}
	}

	return result
}

// QueryVisibleLayers returns entities that are visible layers.
// They must have Transform, BlendMode (with Visible=true), and VulkanPipeline.
func (w *World) QueryVisibleLayers() []Entity {
	result := make([]Entity, 0)

	for entity := range w.entities {
		blend := w.GetBlendMode(entity)
		if blend != nil && blend.Visible &&
		   w.HasTransform(entity) &&
		   w.HasVulkanPipeline(entity) {
			result = append(result, entity)
		}
	}

	return result
}

// QueryAll returns all entities that match a custom filter function.
// This is the most flexible query method.
func (w *World) QueryAll(filter func(Entity) bool) []Entity {
	result := make([]Entity, 0)

	for entity := range w.entities {
		if filter(entity) {
			result = append(result, entity)
		}
	}

	return result
}

// Entities returns the list of entity IDs in this query.
func (q *Query) Entities() []Entity {
	return q.entities
}

// Count returns the number of entities in this query.
func (q *Query) Count() int {
	return len(q.entities)
}

// First returns the first entity in the query, or 0 if empty.
func (q *Query) First() Entity {
	if len(q.entities) > 0 {
		return q.entities[0]
	}
	return 0
}

// ForEach executes a function for each entity in the query.
func (q *Query) ForEach(fn func(Entity)) {
	for _, entity := range q.entities {
		fn(entity)
	}
}

// QueryRenderablesSorted returns renderable entities sorted by ZIndex (ascending = back to front).
func (w *World) QueryRenderablesSorted() []Entity {
	entities := w.QueryRenderables()

	// Sort by ZIndex (low to high = back to front)
	// Simple bubble sort since we have few entities
	for i := 0; i < len(entities)-1; i++ {
		for j := 0; j < len(entities)-i-1; j++ {
			t1 := w.GetTransform(entities[j])
			t2 := w.GetTransform(entities[j+1])
			if t1.ZIndex > t2.ZIndex {
				entities[j], entities[j+1] = entities[j+1], entities[j]
			}
		}
	}

	return entities
}
