package commander

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// init registers the commander subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		initCommanderComponents(em)
		initCommanderTags(em)
		CommanderView = em.World.CreateView(CommanderTag)
	})
}

func initCommanderComponents(manager *common.EntityManager) {
	CommanderComponent = manager.World.NewComponent()
	CommanderActionStateComponent = manager.World.NewComponent()
	OverworldTurnStateComponent = manager.World.NewComponent()
	CommanderRosterComponent = manager.World.NewComponent()
}

func initCommanderTags(manager *common.EntityManager) {
	CommanderTag = ecs.BuildTag(CommanderComponent)
	CommanderActionTag = ecs.BuildTag(CommanderActionStateComponent)
	OverworldTurnTag = ecs.BuildTag(OverworldTurnStateComponent)

	manager.WorldTags["commander"] = CommanderTag
	manager.WorldTags["commanderaction"] = CommanderActionTag
	manager.WorldTags["overworldturn"] = OverworldTurnTag
}
