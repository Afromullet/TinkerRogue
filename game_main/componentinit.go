package main

import (
	"game_main/common"
	"game_main/gear"

	"game_main/rendering"

	"github.com/bytearena/ecs"
)

// InitializeECS sets up the Entity Component System.
// It creates the ECS manager, registers all components, and builds tags for querying.
// This is the central initialization point for all ECS-related setup.
func InitializeECS(ecsmanager *common.EntityManager) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()

	// Register core components
	registerCoreComponents(manager)

	// Register subsystem components and build their tags
	registerItemComponents(manager, tags)

	// Build tags for core systems
	buildCoreTags(tags)

	// Assign to entity manager
	ecsmanager.Tags = tags
	ecsmanager.World = manager
}

// registerCoreComponents registers all core game components.
func registerCoreComponents(manager *ecs.Manager) {
	common.PositionComponent = manager.NewComponent()
	rendering.RenderableComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()
	gear.InventoryComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.UserMsgComponent = manager.NewComponent()
	common.PlayerComponent = manager.NewComponent()
	common.MonsterComponent = manager.NewComponent()
}

// buildCoreTags creates tags for querying core entity types.
func buildCoreTags(tags map[string]ecs.Tag) {
	// Initialize utility tag for "all entities" queries (empty component set)
	common.AllEntitiesTag = ecs.BuildTag()
	tags["all"] = common.AllEntitiesTag

	// Initialize rendering tags
	rendering.RenderablesTag = ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
	tags["renderables"] = rendering.RenderablesTag

	rendering.MessengersTag = ecs.BuildTag(common.UserMsgComponent)
	tags["messengers"] = rendering.MessengersTag

	// Initialize gear tags
	gear.ItemsTag = ecs.BuildTag(gear.ItemComponent, common.PositionComponent)
	tags["items"] = gear.ItemsTag

	gear.MonstersTag = ecs.BuildTag(common.MonsterComponent)
	tags["monsters"] = gear.MonstersTag
}

// registerItemComponents registers item/gear system components.
// Delegates to gear package for its own component setup.
func registerItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {
	gear.InitializeItemComponents(manager, tags)
}
