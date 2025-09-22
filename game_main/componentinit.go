package main

import (
	"game_main/common"
	"game_main/gear"
	"game_main/monsters"
	"game_main/rendering"

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
	rendering.RenderableComponent = manager.NewComponent()
	common.NameComponent = manager.NewComponent()
	gear.InventoryComponent = manager.NewComponent()
	common.AttributeComponent = manager.NewComponent()
	common.UserMsgComponent = manager.NewComponent()
	gear.MeleeWeaponComponent = manager.NewComponent()
	gear.RangedWeaponComponent = manager.NewComponent()
	gear.ArmorComponent = manager.NewComponent()

	renderables := ecs.BuildTag(rendering.RenderableComponent, common.PositionComponent)
	tags["renderables"] = renderables

	messengers := ecs.BuildTag(common.UserMsgComponent)
	tags["messengers"] = messengers

	gear.InitializeItemComponents(manager, tags)

	InitializeCreatureComponents(manager, tags)

	ecsmanager.WorldTags = tags
	ecsmanager.World = manager
}

func InitializeCreatureComponents(manager *ecs.Manager, tags map[string]ecs.Tag) {

	monsters.CreatureComponent = manager.NewComponent()

	creatures := ecs.BuildTag(monsters.CreatureComponent, common.PositionComponent, common.AttributeComponent)
	tags["monsters"] = creatures

}
