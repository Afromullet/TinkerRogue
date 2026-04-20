package raid

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		initRaidComponents(em)
		initRaidTags(em)
	})
}

func initRaidComponents(manager *common.EntityManager) {
	RaidStateComponent = manager.World.NewComponent()
	FloorStateComponent = manager.World.NewComponent()
	RoomDataComponent = manager.World.NewComponent()
	AlertDataComponent = manager.World.NewComponent()
	GarrisonSquadComponent = manager.World.NewComponent()
	DeploymentComponent = manager.World.NewComponent()
}

func initRaidTags(manager *common.EntityManager) {
	RaidStateTag = ecs.BuildTag(RaidStateComponent)
	FloorStateTag = ecs.BuildTag(FloorStateComponent)
	RoomDataTag = ecs.BuildTag(RoomDataComponent)
	AlertDataTag = ecs.BuildTag(AlertDataComponent)
	GarrisonSquadTag = ecs.BuildTag(GarrisonSquadComponent)
	DeploymentTag = ecs.BuildTag(DeploymentComponent)

	manager.WorldTags["raidstate"] = RaidStateTag
	manager.WorldTags["floorstate"] = FloorStateTag
	manager.WorldTags["roomdata"] = RoomDataTag
	manager.WorldTags["alertdata"] = AlertDataTag
	manager.WorldTags["garrisonsquad"] = GarrisonSquadTag
	manager.WorldTags["deployment"] = DeploymentTag
}
