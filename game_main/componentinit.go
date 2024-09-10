package main

import (
	"game_main/actionmanager"
	"game_main/common"
	"game_main/equipment"
	"game_main/monsters"

	"github.com/bytearena/ecs"
)

/*
 */

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

	monsters.InitializeMovementComponents(manager, tags)
	equipment.InitializeItemComponents(manager, tags)
	actionmanager.ActionComponent = manager.NewComponent()

	InitializeCreatureComponents(manager, tags)

	ecsmanager.WorldTags = tags
	ecsmanager.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	monsters.CreatureComponent = manager.NewComponent()

	monsters.ApproachAndAttackComp = manager.NewComponent()
	monsters.DistanceRangeAttackComp = manager.NewComponent()

	creatures := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent, common.AttributeComponent)
	tags["monsters"] = creatures

}
