package main

import (
	"game_main/common"
	"game_main/gear"
	"game_main/monsters"
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
	registerCreatureComponents(manager, tags)

	// Build tags for core systems
	buildCoreTags(tags)

	// Assign to entity manager
	ecsmanager.WorldTags = tags
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
}

// buildCoreTags creates tags for querying core entity types.
func buildCoreTags(tags map[string]ecs.Tag) {
	renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(common.UserMsgComponent)
	tags["messengers"] = messengers
}

// registerItemComponents registers item/gear system components.
// Delegates to gear package for its own component setup.
func registerItemComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {
	gear.InitializeItemComponents(manager, tags)
}

// registerCreatureComponents registers creature/monster components and builds their tag.
func registerCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {
	monsters.CreatureComponent = manager.NewComponent()

	creatures := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent, common.AttributeComponent)
	tags["monsters"] = creatures
}
