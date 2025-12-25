// Package common provides core ECS utilities and shared components for the roguelike game.
// It includes the EntityManager wrapper, common components like Position and Attributes,
// and utility functions for type-safe component access and entity management.
package common

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component
	UserMsgComponent   *ecs.Component //I can probably remove this later

	// AllEntitiesTag queries all entities in the ECS world (empty component set).
	// Used by utility functions that need to work with any entity regardless of components.
	AllEntitiesTag ecs.Tag

	GlobalPositionSystem *PositionSystem

	MonsterComponent *ecs.Component // Marks monster/creature entities
)

// EntityManager wraps the ECS library's manager and provides centralized entity and tag management.
type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}

func NewEntityManager() *EntityManager {
	return &EntityManager{
		World:     ecs.NewManager(),
		WorldTags: make(map[string]ecs.Tag),
	}
}

// HasComponent checks if an entity has a specific component.
// Returns false if the entity ID is invalid or the component is not found.
func (em *EntityManager) HasComponent(entityID ecs.EntityID, component *ecs.Component) bool {
	_, ok := em.GetComponent(entityID, component)
	return ok
}

// GetComponent retrieves component data from an entity by its ID.
// Returns the component data and a boolean indicating if the component was found.
// Returns (nil, false) if the entity ID is invalid or the component is not found.
func (em *EntityManager) GetComponent(entityID ecs.EntityID, component *ecs.Component) (interface{}, bool) {
	// Use ECS library's O(1) entitiesByID map lookup
	result := em.World.GetEntityByID(entityID)
	if result == nil {
		return nil, false
	}
	return result.Entity.GetComponentData(component)
}

// GetComponentType retrieves a component of type T from an entity pointer.
// Preferred when entity is already available from a query result.
// Returns zero value if component not found.
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
	defer func() {
		if r := recover(); r != nil {
			// ERROR HANDLING IN FUTURE
		}
	}()

	if c, ok := entity.GetComponentData(component); ok {
		return c.(T)
	}

	var nilValue T
	return nilValue
}

// GetComponentTypeByID retrieves a component of type T from an entity by ID.
// Returns zero value if entity or component not found.
func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID, component *ecs.Component) T {
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		var nilValue T
		return nilValue
	}
	return GetComponentType[T](entity, component)
}

// GetCreatureAtPosition finds the first monster entity ID at the specified position.
// Returns 0 if no creature is found at that position.
func GetCreatureAtPosition(ecsmanager *EntityManager, pos *coords.LogicalPosition) ecs.EntityID {
	// Use new O(1) PositionSystem if available
	if GlobalPositionSystem != nil {
		entityID := GlobalPositionSystem.GetEntityIDAt(*pos)
		if entityID == 0 {
			return 0
		}

		// Verify it's a monster
		for _, result := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
			if result.Entity.GetID() == entityID {
				return entityID
			}
		}
		return 0
	}

	// Fallback to old O(n) search if PositionSystem not initialized
	for _, c := range ecsmanager.World.Query(ecsmanager.WorldTags["monsters"]) {
		curPos := GetComponentType[*coords.LogicalPosition](c.Entity, PositionComponent)
		if pos.IsEqual(curPos) {
			return c.Entity.GetID()
		}
	}
	return 0
}

// FindEntityByID finds an entity pointer by its ID.
// Use this when you need the entity pointer for operations that require it.
// For component access, prefer GetComponentTypeByID and related helpers.
//
// Returns nil if the entity is not found.
func (em *EntityManager) FindEntityByID(entityID ecs.EntityID) *ecs.Entity {
	// Use ECS library's O(1) entitiesByID map lookup
	result := em.World.GetEntityByID(entityID)
	if result == nil {
		return nil
	}
	return result.Entity
}

// MoveEntity updates position component and position system atomically.
// This ensures that position is synchronized across:
// 1. Entity's PositionComponent
// 2. GlobalPositionSystem spatial grid
//
// Returns error if entity has no position component.
func (em *EntityManager) MoveEntity(
	entityID ecs.EntityID,
	entity *ecs.Entity,
	oldPos coords.LogicalPosition,
	newPos coords.LogicalPosition,
) error {
	// 1. Update component
	posComponent, ok := entity.GetComponentData(PositionComponent)
	if !ok {
		return nil // Silently skip entities without position component
	}

	posPtr := posComponent.(*coords.LogicalPosition)
	posPtr.X = newPos.X
	posPtr.Y = newPos.Y

	// 2. Update position system
	GlobalPositionSystem.RemoveEntity(entityID, oldPos)
	GlobalPositionSystem.AddEntity(entityID, newPos)

	return nil
}

// MoveSquadAndMembers moves squad and all member units to a new position.
// This atomically updates position across all three storage locations:
// 1. Squad position component
// 2. GlobalPositionSystem spatial grid
// 3. Unit member position components
//
// Returns error if squad has no position component.
func (em *EntityManager) MoveSquadAndMembers(
	squadID ecs.EntityID,
	squadEntity *ecs.Entity,
	unitIDs []ecs.EntityID,
	oldPos coords.LogicalPosition,
	newPos coords.LogicalPosition,
) error {
	// Move squad
	if err := em.MoveEntity(squadID, squadEntity, oldPos, newPos); err != nil {
		return err
	}

	// Move all unit members
	for _, unitID := range unitIDs {
		unitEntity := em.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		// Skip error - some units may not have position component
		em.MoveEntity(unitID, unitEntity, oldPos, newPos)
	}

	return nil
}

// CleanDisposeEntity removes an entity from both the ECS World and GlobalPositionSystem.
// This prevents memory leaks by ensuring entities are cleaned up from all systems.
//
// Call this instead of World.DisposeEntities() directly when the entity has a position.
// For entities without positions, World.DisposeEntities() is still safe to use directly.
//
// Usage:
//
//	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
//	if pos != nil {
//	    manager.CleanDisposeEntity(entity, pos)
//	} else {
//	    manager.World.DisposeEntities(entity)
//	}
func (em *EntityManager) CleanDisposeEntity(entity *ecs.Entity, position *coords.LogicalPosition) {
	if entity == nil {
		return
	}

	entityID := entity.GetID()

	// Remove from GlobalPositionSystem first
	if position != nil && GlobalPositionSystem != nil {
		GlobalPositionSystem.RemoveEntity(entityID, *position)
	}

	// Remove from ECS world
	em.World.DisposeEntities(entity)
}

// ========================================
// SUBSYSTEM SELF-REGISTRATION PATTERN
// ========================================

// subsystemRegistrars holds initialization functions for all ECS subsystems.
// Subsystems register themselves via init() functions in their own packages.
var subsystemRegistrars []func(*EntityManager)

// RegisterSubsystem adds a subsystem initialization function to the registry.
// This is called by subsystem packages in their init() functions.
//
// Example usage in a subsystem package:
//
//	func init() {
//	    common.RegisterSubsystem(func(em *common.EntityManager) {
//	        InitMyComponents(em)
//	        InitMyTags(em)
//	    })
//	}
func RegisterSubsystem(registrar func(*EntityManager)) {
	subsystemRegistrars = append(subsystemRegistrars, registrar)
}

// InitializeSubsystems calls all registered subsystem initialization functions.
// Call this after creating the EntityManager and setting World/WorldTags.
//
// This executes subsystem registrations in the order they were registered via init().
func InitializeSubsystems(em *EntityManager) {
	for _, registrar := range subsystemRegistrars {
		registrar(em)
	}
}
