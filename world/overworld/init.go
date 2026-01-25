package overworld

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// init registers the overworld subsystem with the ECS component registry.
// This allows the overworld package to self-register its components without
// game_main needing to know about overworld internals.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		InitOverworldComponents(em)
		InitOverworldTags(em)
	})
}

// InitOverworldComponents registers all overworld-related components with the ECS manager.
func InitOverworldComponents(manager *common.EntityManager) {
	ThreatNodeComponent = manager.World.NewComponent()
	OverworldFactionComponent = manager.World.NewComponent()
	TickStateComponent = manager.World.NewComponent()
	InfluenceComponent = manager.World.NewComponent()
	TerritoryComponent = manager.World.NewComponent()
	StrategicIntentComponent = manager.World.NewComponent()
	VictoryStateComponent = manager.World.NewComponent()
	PlayerResourcesComponent = manager.World.NewComponent()
	InfluenceCacheComponent = manager.World.NewComponent()
}

// InitOverworldTags creates tags for querying overworld-related entities.
func InitOverworldTags(manager *common.EntityManager) {
	ThreatNodeTag = ecs.BuildTag(ThreatNodeComponent)
	OverworldFactionTag = ecs.BuildTag(OverworldFactionComponent)
	TickStateTag = ecs.BuildTag(TickStateComponent)
	VictoryStateTag = ecs.BuildTag(VictoryStateComponent)
	PlayerResourcesTag = ecs.BuildTag(PlayerResourcesComponent)
	InfluenceCacheTag = ecs.BuildTag(InfluenceCacheComponent)

	// Register tags in WorldTags for easier lookup
	manager.WorldTags["threatnode"] = ThreatNodeTag
	manager.WorldTags["overworldfaction"] = OverworldFactionTag
	manager.WorldTags["tickstate"] = TickStateTag
	manager.WorldTags["victorystate"] = VictoryStateTag
	manager.WorldTags["playerresources"] = PlayerResourcesTag
	manager.WorldTags["influencecache"] = InfluenceCacheTag
}
