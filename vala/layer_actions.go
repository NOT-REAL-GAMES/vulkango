package main

import (
	"fmt"

	"github.com/NOT-REAL-GAMES/vulkango/vala/ecs"
)

// ActionType defines the type of undoable action
type ActionType int

const (
	ActionTypeStroke ActionType = iota
	ActionTypeLayerCreate
	ActionTypeLayerDelete
	ActionTypeLayerTransform
	ActionTypeLayerVisibility
	ActionTypeLayerOpacity
)

// String returns the name of the action type
func (a ActionType) String() string {
	switch a {
	case ActionTypeStroke:
		return "Stroke"
	case ActionTypeLayerCreate:
		return "LayerCreate"
	case ActionTypeLayerDelete:
		return "LayerDelete"
	case ActionTypeLayerTransform:
		return "LayerTransform"
	case ActionTypeLayerVisibility:
		return "LayerVisibility"
	case ActionTypeLayerOpacity:
		return "LayerOpacity"
	default:
		return "Unknown"
	}
}

// PenState records a single pen position and pressure reading
type PenState struct {
	X        float32
	Y        float32
	Pressure float32
}

// Stroke represents a complete brush stroke with all pen states
type Stroke struct {
	States []PenState
	Color  [4]float32 // RGBA
	Radius float32
}

// LayerCreateAction stores data for layer creation
type LayerCreateAction struct {
	EntityID ecs.Entity
	ZIndex   int
	// Store all layer data needed to recreate it
	Transform *ecs.Transform
	BlendMode *ecs.BlendMode
	// Add other components as needed
}

// LayerDeleteAction stores data for layer deletion
type LayerDeleteAction struct {
	EntityID  ecs.Entity
	ZIndex    int
	Transform *ecs.Transform
	BlendMode *ecs.BlendMode
	// Store full layer state for restoration
}

// LayerTransformAction stores transform changes
type LayerTransformAction struct {
	EntityID     ecs.Entity
	OldTransform ecs.Transform
	NewTransform ecs.Transform
}

// LayerVisibilityAction stores visibility changes
type LayerVisibilityAction struct {
	EntityID     ecs.Entity
	OldVisible   bool
	NewVisible   bool
	SavedOpacity float32 // The opacity to restore when showing the layer
}

// LayerOpacityAction stores opacity changes
type LayerOpacityAction struct {
	EntityID   ecs.Entity
	OldOpacity float32
	NewOpacity float32
}

// Action represents any undoable action in the application
// It's a union type - only one field will be non-nil based on Type
type Action struct {
	Type ActionType

	// Stroke action data
	Stroke *Stroke

	// Layer action data (only one will be non-nil based on Type)
	LayerCreate     *LayerCreateAction
	LayerDelete     *LayerDeleteAction
	LayerTransform  *LayerTransformAction
	LayerVisibility *LayerVisibilityAction
	LayerOpacity    *LayerOpacityAction
}

// ActionRecorder manages the undo/redo stack
type ActionRecorder struct {
	history []Action
	index   int
}

// NewActionRecorder creates a new action recorder
func NewActionRecorder() *ActionRecorder {
	return &ActionRecorder{
		history: make([]Action, 0),
		index:   0,
	}
}

// RecordStroke adds a stroke action to the history
func (r *ActionRecorder) RecordStroke(stroke Stroke) {
	// Truncate history if we're in the middle of the stack
	if r.index < len(r.history) {
		r.history = r.history[:r.index]
	}

	// Create a proper heap-allocated copy of the stroke
	// Deep copy the States slice to avoid sharing the underlying array
	strokeCopy := &Stroke{
		States: append([]PenState(nil), stroke.States...),
		Color:  stroke.Color,
		Radius: stroke.Radius,
	}

	action := Action{
		Type:   ActionTypeStroke,
		Stroke: strokeCopy,
	}
	r.history = append(r.history, action)
	r.index = len(r.history)
	fmt.Printf("Recorded stroke action: %d pen states, total history: %d actions\n",
		len(strokeCopy.States), len(r.history))
}

// RecordLayerVisibility adds a visibility change action to the history
func (r *ActionRecorder) RecordLayerVisibility(entityID ecs.Entity, oldVisible, newVisible bool, savedOpacity float32) {
	// Truncate history if we're in the middle of the stack
	if r.index < len(r.history) {
		r.history = r.history[:r.index]
	}

	action := Action{
		Type: ActionTypeLayerVisibility,
		LayerVisibility: &LayerVisibilityAction{
			EntityID:     entityID,
			OldVisible:   oldVisible,
			NewVisible:   newVisible,
			SavedOpacity: savedOpacity,
		},
	}
	r.history = append(r.history, action)
	r.index = len(r.history)
	fmt.Printf("Recorded layer visibility change: layer=%d, %v→%v, savedOpacity=%.2f\n", entityID, oldVisible, newVisible, savedOpacity)
}

// RecordLayerOpacity adds an opacity change action to the history
func (r *ActionRecorder) RecordLayerOpacity(entityID ecs.Entity, oldOpacity, newOpacity float32) {
	// Truncate history if we're in the middle of the stack
	if r.index < len(r.history) {
		r.history = r.history[:r.index]
	}

	action := Action{
		Type: ActionTypeLayerOpacity,
		LayerOpacity: &LayerOpacityAction{
			EntityID:   entityID,
			OldOpacity: oldOpacity,
			NewOpacity: newOpacity,
		},
	}
	r.history = append(r.history, action)
	r.index = len(r.history)
	fmt.Printf("Recorded layer opacity change: layer=%d, %.2f→%.2f\n", entityID, oldOpacity, newOpacity)
}

// RecordLayerTransform adds a transform change action to the history
func (r *ActionRecorder) RecordLayerTransform(entityID ecs.Entity, oldTransform, newTransform ecs.Transform) {
	// Truncate history if we're in the middle of the stack
	if r.index < len(r.history) {
		r.history = r.history[:r.index]
	}

	action := Action{
		Type: ActionTypeLayerTransform,
		LayerTransform: &LayerTransformAction{
			EntityID:     entityID,
			OldTransform: oldTransform,
			NewTransform: newTransform,
		},
	}
	r.history = append(r.history, action)
	r.index = len(r.history)
	fmt.Printf("Recorded layer transform change: layer=%d\n", entityID)
}

// Undo moves back one action in the history
func (r *ActionRecorder) Undo() bool {
	if r.index > 0 {
		r.index--
		fmt.Printf("Undo: reverting to action %d/%d\n", r.index, len(r.history))
		return true
	}
	fmt.Println("Nothing to undo")
	return false
}

// Redo moves forward one action in the history
func (r *ActionRecorder) Redo() bool {
	if r.index < len(r.history) {
		r.index++
		fmt.Printf("Redo: advancing to action %d/%d\n", r.index, len(r.history))
		return true
	}
	fmt.Println("Nothing to redo")
	return false
}

// CanUndo returns true if there are actions to undo
func (r *ActionRecorder) CanUndo() bool {
	return r.index > 0
}

// CanRedo returns true if there are actions to redo
func (r *ActionRecorder) CanRedo() bool {
	return r.index < len(r.history)
}

// GetHistory returns a slice of all actions up to the current index
func (r *ActionRecorder) GetHistory() []Action {
	return r.history[:r.index]
}

// GetFullHistory returns all actions in the history
func (r *ActionRecorder) GetFullHistory() []Action {
	return r.history
}

// GetIndex returns the current position in the history
func (r *ActionRecorder) GetIndex() int {
	return r.index
}

// Clear clears all history
func (r *ActionRecorder) Clear() {
	r.history = make([]Action, 0)
	r.index = 0
	fmt.Println("Action history cleared")
}
