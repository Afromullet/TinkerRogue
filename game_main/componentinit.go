package main

import (
	"game_main/common"
	"game_main/gear"
	"game_main/tactical/combat"
	"game_main/tactical/squads"

	"game_main/visual/rendering"

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

	// Assign to entity manager (must be done before initializing subsystems)
	ecsmanager.WorldTags = tags
	ecsmanager.World = manager

	// Initialize subsystems that need the full EntityManager
	registerSquadComponents(ecsmanager)
	registerCombatComponents(ecsmanager)
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
	common.PlayerResourcesComponent = manager.NewComponent()
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

// registerSquadComponents registers squad system components and tags.
// Must be called after EntityManager.World and WorldTags are assigned.
func registerSquadComponents(ecsmanager *common.EntityManager) {
	squads.InitSquadComponents(ecsmanager)
	squads.InitSquadTags(ecsmanager)
}

// registerCombatComponents registers combat system components and tags.
// Must be called after EntityManager.World and WorldTags are assigned.
func registerCombatComponents(ecsmanager *common.EntityManager) {
	combat.InitCombatComponents(ecsmanager)
	combat.InitCombatTags(ecsmanager)
}
