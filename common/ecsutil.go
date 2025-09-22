// Package common provides core ECS utilities and shared components for the roguelike game.
// It includes the EntityManager wrapper, common components like Position and Attributes,
// and utility functions for type-safe component access and entity management.
package common

import (
	"github.com/bytearena/ecs"
)

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component
	UserMsgComponent   *ecs.Component //I can probably remove this later
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
func GetPosition(e *ecs.Entity) *Position {
	return GetComponentType[*Position](e, PositionComponent)
}

// GetCreatureAtPosition finds and returns the first monster entity at the specified position.
// Returns nil if no creature is found at that position.
// TODO: Optimize this to avoid searching all monsters every time.
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *Position) *ecs.Entity {

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
