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

	// AllEntitiesTag queries all entities in the ECS world (empty component set).
	// Used by utility functions that need to work with any entity regardless of components.
	AllEntitiesTag ecs.Tag

	// GlobalPositionSystem provides O(1) position-based entity lookup.
	// Initialized during game setup. Replaces O(n) linear search from trackers.
	GlobalPositionSystem *systems.PositionSystem
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

// GetAllEntities returns all entity IDs currently managed by the EntityManager.
func (em *EntityManager) GetAllEntities() []ecs.EntityID {
	var entityIDs []ecs.EntityID
	for _, result := range em.World.Query(AllEntitiesTag) {
		entityIDs = append(entityIDs, result.Entity.GetID())
	}
	return entityIDs
}

// HasComponent checks if an entity has a specific component.
// Returns false if the entity ID is invalid or the component is not found.
func (em *EntityManager) HasComponent(entityID ecs.EntityID, component *ecs.Component) bool {
	for _, result := range em.World.Query(AllEntitiesTag) {
		if result.Entity.GetID() == entityID {
			_, ok := result.Entity.GetComponentData(component)
			return ok
		}
	}
	return false
}

// HasComponentByIDWithTag checks if an entity (queried by tag) has a specific component.
// Returns false if the entity ID is invalid or the component is not found.
func (em *EntityManager) HasComponentByIDWithTag(entityID ecs.EntityID, tag ecs.Tag, component *ecs.Component) bool {
	for _, result := range em.World.Query(tag) {
		if result.Entity.GetID() == entityID {
			_, ok := result.Entity.GetComponentData(component)
			return ok
		}
	}
	return false
}

// GetComponent retrieves component data from an entity by its ID.
// Returns the component data and a boolean indicating if the component was found.
// Returns (nil, false) if the entity ID is invalid or the component is not found.
func (em *EntityManager) GetComponent(entityID ecs.EntityID, component *ecs.Component) (interface{}, bool) {
	for _, result := range em.World.Query(AllEntitiesTag) {
		if result.Entity.GetID() == entityID {
			return result.Entity.GetComponentData(component)
		}
	}
	return nil, false
}

// GetComponentType retrieves a component of type T from an entity pointer.
// Used internally when entity is already available from a query result.
// For new code, prefer GetComponentTypeByID.
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

// GetComponentTypeByID retrieves a component of type T from an entity by ID.
// Returns zero value if entity or component not found.
func GetComponentTypeByID[T any](manager *EntityManager, entityID ecs.EntityID, component *ecs.Component) T {
	defer func() {
		if r := recover(); r != nil {
			// ERROR HANDLING IN FUTURE
		}
	}()

	for _, result := range manager.World.Query(AllEntitiesTag) {
		if result.Entity.GetID() == entityID {
			if c, ok := result.Entity.GetComponentData(component); ok {
				return c.(T)
			}
			var nilValue T
			return nilValue
		}
	}

	var nilValue T
	return nilValue
}

// GetComponentTypeByIDWithTag retrieves a component by entity ID within a specific tag query.
// Returns zero value if entity or component not found.
func GetComponentTypeByIDWithTag[T any](manager *EntityManager, entityID ecs.EntityID, tag ecs.Tag, component *ecs.Component) T {
	defer func() {
		if r := recover(); r != nil {
			// ERROR HANDLING IN FUTURE
		}
	}()

	for _, result := range manager.World.Query(tag) {
		if result.Entity.GetID() == entityID {
			if c, ok := result.Entity.GetComponentData(component); ok {
				return c.(T)
			}
			var nilValue T
			return nilValue
		}
	}

	var nilValue T
	return nilValue
}

// GetAttributes returns the Attributes component from an entity.
// Used for entities obtained from query results.
func GetAttributes(e *ecs.Entity) *Attributes {
	return GetComponentType[*Attributes](e, AttributeComponent)
}

// GetAttributesByID returns the Attributes component by entity ID.
// Use this when you only have an EntityID (not the entity pointer).
// Returns nil if entity not found.
func GetAttributesByID(manager *EntityManager, entityID ecs.EntityID) *Attributes {
	return GetComponentTypeByID[*Attributes](manager, entityID, AttributeComponent)
}

// GetPosition returns the Position component from an entity.
// Used for entities obtained from query results.
func GetPosition(e *ecs.Entity) *coords.LogicalPosition {
	return GetComponentType[*coords.LogicalPosition](e, PositionComponent)
}

// GetPositionByID returns the Position component by entity ID.
// Use this when you only have an EntityID (not the entity pointer).
// Returns nil if entity not found.
func GetPositionByID(manager *EntityManager, entityID ecs.EntityID) *coords.LogicalPosition {
	return GetComponentTypeByID[*coords.LogicalPosition](manager, entityID, PositionComponent)
}

// GetAttributesByIDWithTag returns the Attributes component by entity ID within a tag query.
// Returns nil if entity not found or doesn't have the component.
func GetAttributesByIDWithTag(manager *EntityManager, entityID ecs.EntityID, tag ecs.Tag) *Attributes {
	return GetComponentTypeByIDWithTag[*Attributes](manager, entityID, tag, AttributeComponent)
}

// GetPositionByIDWithTag returns the Position component by entity ID within a tag query.
// Returns nil if entity not found or doesn't have the component.
func GetPositionByIDWithTag(manager *EntityManager, entityID ecs.EntityID, tag ecs.Tag) *coords.LogicalPosition {
	return GetComponentTypeByIDWithTag[*coords.LogicalPosition](manager, entityID, tag, PositionComponent)
}

// GetCreatureAtPosition finds the first monster entity ID at the specified position.
// Returns 0 if no creature is found at that position.
// Uses O(1) PositionSystem lookup instead of O(n) linear search.
func GetCreatureAtPosition(ecsmnager *EntityManager, pos *coords.LogicalPosition) ecs.EntityID {
	// Use new O(1) PositionSystem if available
	if GlobalPositionSystem != nil {
		entityID := GlobalPositionSystem.GetEntityIDAt(*pos)
		if entityID == 0 {
			return 0
		}

		// Verify it's a monster
		for _, result := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
			if result.Entity.GetID() == entityID {
				return entityID
			}
		}
		return 0
	}

	// Fallback to old O(n) search if PositionSystem not initialized
	for _, c := range ecsmnager.World.Query(ecsmnager.WorldTags["monsters"]) {
		curPos := GetPosition(c.Entity)
		if pos.IsEqual(curPos) {
			return c.Entity.GetID()
		}
	}
	return 0
}

// FindEntityIDWithTag finds an entity ID within a specific tag query.
// This replaces the old FindEntityByIDWithTag function.
// Returns 0 if the entity is not found.
//
// Usage:
//   unitID := FindEntityIDWithTag(manager, unitEntityID, SquadMemberTag)
//   if unitID != 0 { ... }
func FindEntityIDWithTag(manager *EntityManager, entityID ecs.EntityID, tag ecs.Tag) ecs.EntityID {
	for _, result := range manager.World.Query(tag) {
		if result.Entity.GetID() == entityID {
			return entityID
		}
	}
	return 0
}

// FindEntityByID finds an entity pointer by its ID, searching all entities.
// Use this when you need the entity pointer for operations that require it.
// For component access, prefer GetComponentTypeByID and related helpers.
//
// Returns nil if the entity is not found.
//
// Usage:
//   entity := FindEntityByID(manager, entityID)
//   if entity != nil {
//       entity.AddComponent(ComponentType, data)
//   }
func FindEntityByID(manager *EntityManager, entityID ecs.EntityID) *ecs.Entity {
	for _, result := range manager.World.Query(AllEntitiesTag) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}

// FindEntityByIDInManager finds an entity pointer by its ID using ecs.Manager directly.
// This is for packages that work with ecs.Manager instead of EntityManager.
// Returns nil if the entity is not found.
//
// Usage:
//   entity := FindEntityByIDInManager(ecsManager, entityID)
//   if entity != nil {
//       // use entity
//   }
func FindEntityByIDInManager(manager *ecs.Manager, entityID ecs.EntityID) *ecs.Entity {
	for _, result := range manager.Query(AllEntitiesTag) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}

// FindEntityByIDWithTag finds an entity pointer within a specific tag query.
// ⚠️ DEPRECATED: This function should only be used when you need the entity pointer
// for operations like AddComponent that require it. For all other cases, use the
// EntityID-based helpers like GetComponentTypeByIDWithTag, GetAttributesByIDWithTag, etc.
//
// Returns nil if the entity is not found within the tag query.
//
// Limited valid use cases:
//   - entity.AddComponent() - ECS library requires entity pointer
//
// Usage:
//   entity := FindEntityByIDWithTag(manager, entityID, Tag)
//   if entity != nil {
//       entity.AddComponent(ComponentType, data)
//   }
func FindEntityByIDWithTag(manager *EntityManager, entityID ecs.EntityID, tag ecs.Tag) *ecs.Entity {
	for _, result := range manager.World.Query(tag) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}
