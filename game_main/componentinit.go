package main

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// InitializeECS sets up the Entity Component System.
// It creates the ECS manager, registers all components, and builds tags for querying.
// This is the central initialization point for all ECS-related setup.
//
// Subsystems (gear, squads, combat) self-register via init() functions.
// After setting up the core manager and tags, call common.InitializeSubsystems()
// to trigger all subsystem registrations.
func InitializeECS(ecsmanager *common.EntityManager) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()

	// Register core components
	registerCoreComponents(manager)

	// Build tags for core systems
	buildCoreTags(tags)

	// Assign to entity manager (must be done before initializing subsystems)
	ecsmanager.WorldTags = tags
	ecsmanager.World = manager

	// Initialize all registered subsystems (gear, squads, combat, etc.)
	// Subsystems register themselves via init() functions in their packages
	common.InitializeSubsystems(ecsmanager)
}

// registerCoreComponents registers all core game components.
// Subsystem components (gear, squads, combat, rendering) are registered by their own packages.
func registerCoreComponents(manager *ecs.Manager) {
	common.PositionComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.PlayerComponent = manager.NewComponent()
	common.MonsterComponent = manager.NewComponent()
	common.PlayerResourcesComponent = manager.NewComponent()
}

// buildCoreTags creates tags for querying core entity types.
// Subsystem-specific tags (gear, squads, combat, rendering) are built by the subsystems themselves.
func buildCoreTags(tags map[string]ecs.Tag) {
	// Initialize utility tag for "all entities" queries (empty component set)
	common.AllEntitiesTag = ecs.BuildTag()
	tags["all"] = common.AllEntitiesTag

	// Note: All subsystem tags (gear, squads, combat, rendering) are now built
	// by their respective packages during InitializeSubsystems()
}
