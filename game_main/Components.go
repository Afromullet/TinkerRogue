package main

import (
	"game_main/common"
	"game_main/equipment"
	"game_main/worldmap"

	"github.com/bytearena/ecs"
)

/*
 */

var (
	CreatureComponent *ecs.Component
)

// The ECS library returns pointers to the struct when querying it for components, so the Position methods take a pointer as input
// Other than that, there's no reason for using pointers for the functions below.

// This gets called so often that it might as well be a function
func GetCreature(e *ecs.Entity) *Creature {
	return common.GetComponentType[*Creature](e, CreatureComponent)
}

// todo Will be refactored. Don't get distracted by this at the moment.
// ALl of the initialziation will have to be handled differently - since
func InitializeECS(ecsmanager *common.EntityManager) {
	tags := make(map[string]ecs.Tag)
	manager := ecs.NewManager()
	common.PositionComponent = manager.NewComponent()
	common.RenderableComponent = manager.NewComponent()

	common.NameComponent = manager.NewComponent()

	equipment.InventoryComponent = manager.NewComponent()

	common.AttributeComponent = manager.NewComponent()
	common.UsrMsg = manager.NewComponent()

	equipment.WeaponComponent = manager.NewComponent()
	equipment.RangedWeaponComponent = manager.NewComponent()
	equipment.ArmorComponent = manager.NewComponent()

	renderables := ecs.BuildTag(common.RenderableComponent, common.PositionComponent)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(common.UsrMsg)
	tags["messengers"] = messengers

	InitializeMovementComponents(manager, tags)
	equipment.InitializeItemComponents(manager, tags)
	InitializeCreatureComponents(manager, tags)

	ecsmanager.WorldTags = tags
	ecsmanager.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	CreatureComponent = manager.NewComponent()

	approachAndAttack = manager.NewComponent()
	distanceRangeAttack = manager.NewComponent()

	creatures := ecs.BuildTag(CreatureComponent, common.PositionComponent, common.AttributeComponent)
	tags["monsters"] = creatures

}

// Creates a slice of Positions from p to other. Uses AStar to build the path
func BuildPath(gm *worldmap.GameMap, start *common.Position, other *common.Position) []common.Position {

	astar := AStar{}
	return astar.GetPath(*gm, start, other, false)

}
