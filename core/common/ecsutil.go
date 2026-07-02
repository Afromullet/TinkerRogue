// Package common provides core ECS utilities and shared components for the roguelike game.
// It includes the EntityManager wrapper, common components like Position and Attributes,
// and utility functions for type-safe component access and entity management.
package common

import (
	"errors"
	"fmt"

	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

var (
	PositionComponent  *ecs.Component
	NameComponent      *ecs.Component
	AttributeComponent *ecs.Component

	// AllEntitiesTag queries all entities in the ECS world (empty component set).
	// Used by utility functions that need to work with any entity regardless of components.
	AllEntitiesTag ecs.Tag

	// GlobalPositionSystem is a deliberate global mutable singleton: the spatial
	// grid is shared process-wide and initialized once at boot. The trade-off is
	// that tests touching it cannot run in parallel and init order matters. This is
	// acceptable for a solo project; see docs/.../ECS_BEST_PRACTICES.md if injection
	// ever becomes necessary (parallel tests, snapshot/replay).
	GlobalPositionSystem *PositionSystem

	RenderableComponent *ecs.Component
	RenderablesTag      ecs.Tag
)

// EntityManager wraps the ECS library's manager and provides centralized entity and tag management.
type EntityManager struct {
	World     *ecs.Manager
	WorldTags map[string]ecs.Tag
}

// NewEntityManager creates an EntityManager with an empty ECS world and tag map.
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
// Returns zero value if entity is nil or component not found.
// Panics on type assertion failure (indicates a bug — wrong type parameter).
func GetComponentType[T any](entity *ecs.Entity, component *ecs.Component) T {
	if entity == nil {
		var zero T
		return zero
	}

	if c, ok := entity.GetComponentData(component); ok {
		return c.(T)
	}

	var zero T
	return zero
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

// ErrNoPositionComponent is returned (wrapped) by MoveEntity when the target
// entity has no PositionComponent. Single-entity movers (squads, commanders)
// treat it as a real error; bulk movers like MoveSquadAndMembers treat it as an
// expected skip for members that are not positioned yet (e.g. reserves). Test for
// it with errors.Is.
var ErrNoPositionComponent = errors.New("entity has no position component")

// MoveEntity updates position component and position system atomically.
// This ensures that position is synchronized across:
// 1. Entity's PositionComponent
// 2. GlobalPositionSystem spatial grid
//
// Returns an error wrapping ErrNoPositionComponent if the entity has no
// PositionComponent (a real misuse for single-entity callers, not a silent no-op).
//
// Upsert semantics: this is intentionally NOT the same operation as
// PositionSystem.MoveEntity (a strict grid-only move that errors if the entity
// is not at oldPos). The RemoveEntity error here is deliberately ignored because
// callers such as combatlifecycle.EnsureUnitPositions move entities that hold a
// stale PositionComponent but are not yet registered in the spatial grid; the
// AddEntity below then registers them at newPos. That is why this function
// re-implements remove+add instead of delegating to PositionSystem.MoveEntity.
func (em *EntityManager) MoveEntity(
	entityID ecs.EntityID,
	entity *ecs.Entity,
	oldPos coords.LogicalPosition,
	newPos coords.LogicalPosition,
) error {
	// 1. Update component
	posComponent, ok := entity.GetComponentData(PositionComponent)
	if !ok {
		return fmt.Errorf("MoveEntity: entity %d: %w", entityID, ErrNoPositionComponent)
	}

	posPtr := posComponent.(*coords.LogicalPosition)
	posPtr.X = newPos.X
	posPtr.Y = newPos.Y

	// 2. Update position system (upsert; see doc comment for why the remove error
	// is intentionally ignored).
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
// Returns an error if the squad cannot be moved, or a joined error describing any
// member units that failed to move (the move continues for the remaining members
// rather than aborting half-way).
func (em *EntityManager) MoveSquadAndMembers(
	squadID ecs.EntityID,
	squadEntity *ecs.Entity,
	unitIDs []ecs.EntityID,
	oldPos coords.LogicalPosition,
	newPos coords.LogicalPosition,
) error {
	// Move squad
	if err := em.MoveEntity(squadID, squadEntity, oldPos, newPos); err != nil {
		return fmt.Errorf("failed to move squad %d: %w", squadID, err)
	}

	// Move all unit members, collecting (not swallowing) real errors. A member
	// without a PositionComponent is an expected skip (e.g. units not positioned
	// before the squad is enrolled), not a failure.
	var errs []error
	for _, unitID := range unitIDs {
		unitEntity := em.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		if err := em.MoveEntity(unitID, unitEntity, oldPos, newPos); err != nil {
			if errors.Is(err, ErrNoPositionComponent) {
				continue
			}
			errs = append(errs, fmt.Errorf("unit %d: %w", unitID, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// RegisterEntityPosition atomically adds a PositionComponent and registers the entity
// with the GlobalPositionSystem. Use this for initial entity creation when the entity
// needs a tracked world position. This prevents desync between the two position stores.
//
// For moving an existing entity, use MoveEntity instead.
// For removing position, use UnregisterEntityPosition or CleanDisposeEntity.
func (em *EntityManager) RegisterEntityPosition(entity *ecs.Entity, pos coords.LogicalPosition) {
	posPtr := new(coords.LogicalPosition)
	*posPtr = pos
	entity.AddComponent(PositionComponent, posPtr)
	if GlobalPositionSystem != nil {
		GlobalPositionSystem.AddEntity(entity.GetID(), pos)
	}
}

// UnregisterEntityPosition atomically removes an entity from the GlobalPositionSystem
// and removes its PositionComponent. Use this when stripping position from an entity
// that will continue to exist (e.g., post-combat cleanup).
//
// For full entity disposal (position removal + ECS disposal), use CleanDisposeEntity.
// No-op if the entity has no PositionComponent.
func (em *EntityManager) UnregisterEntityPosition(entity *ecs.Entity) {
	pos := GetComponentType[*coords.LogicalPosition](entity, PositionComponent)
	if pos == nil {
		return
	}
	if GlobalPositionSystem != nil {
		GlobalPositionSystem.RemoveEntity(entity.GetID(), *pos)
	}
	entity.RemoveComponent(PositionComponent)
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
// Init-order invariant: registrars run in the order their packages' init()
// functions ran, which Go does not guarantee across packages. A registrar must
// therefore NOT depend on another subsystem's components or tags being created
// yet — it may only create its own. Cross-subsystem wiring belongs in a later
// explicit setup step, not in a registrar.
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

// GetEntityName retrieves the name of an entity. Returns defaultName if not found.
func GetEntityName(manager *EntityManager, entityID ecs.EntityID, defaultName string) string {
	if name := GetComponentTypeByID[*Name](manager, entityID, NameComponent); name != nil {
		return name.NameStr
	}
	return defaultName
}
