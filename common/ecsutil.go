// Package common provides core ECS utilities and shared components for the roguelike game.
// It includes the EntityManager wrapper, common components like Position and Attributes,
// and utility functions for type-safe component access and entity management.
package common

import (
	"game_main/coords"
	"game_main/systems"

	"github.com/bytearena/ecs"
)

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component
	UserMsgComponent   *ecs.Component //I can probably remove this later

	// GlobalPositionSystem provides O(1) position-based entity lookup.
	// Initialized during game setup. Replaces O(n) linear search from trackers.
	GlobalPositionSystem *systems.PositionSystem
)

// EntityManager wraps the ECS library's manager and provides centralized entity and tag management.
type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}

// GetComponentType retrieves a component of type T from an entity.
// It provides type-safe component access with panic recovery for missing components.
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {

	defer func() {
		if r := recover(); r != nil {

			// ERROR HANDLING IN FUTURE

		}
	}()

	if c, ok := entity.GetComponentData(component); ok {
		return c.(T)

	} else {
		var nilValue T
		return nilValue
	}

}

// DistanceBetween calculates the Chebyshev distance between two entities.
// Both entities must have position components for this function to work correctly.
func DistanceBetween(e1 *ecs.Entity, e2 *ecs.Entity) int {

	pos1 := GetPosition(e1)
	pos2 := GetPosition(e2)

	return pos1.ChebyshevDistance(pos2)

}

// GetAttributes returns the Attributes component from an entity.
// This is a convenience function for frequently accessed components.
func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

// GetPosition returns the Position component from an entity.
// This is a convenience function for frequently accessed components.
func GetPosition(e *ecs.Entity) *coords.LogicalPosition {
	return GetComponentType[*coords.LogicalPosition](e, PositionComponent)
}

// GetCreatureAtPosition finds and returns the first monster entity at the specified position.
// Returns nil if no creature is found at that position.
// Now uses O(1) PositionSystem lookup instead of O(n) linear search.
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *coords.LogicalPosition) *ecs.Entity {
	// Use new O(1) PositionSystem if available
	if GlobalPositionSystem != nil {
		entityID := GlobalPositionSystem.GetEntityIDAt(*pos)
		if entityID == 0 {
			return nil
		}

		// Find entity and verify it's a monster
		for _, result := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
			if result.Entity.GetID() == entityID {
				return result.Entity
			}
		}
		return nil
	}

	// Fallback to old O(n) search if PositionSystem not initialized
	// This ensures backward compatibility during migration
	var e *ecs.Entity = nil
	for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
		curPos := GetPosition(c.Entity)
		if pos.IsEqual(curPos) {
			e = c.Entity
			break
		}
	}
	return e
}
