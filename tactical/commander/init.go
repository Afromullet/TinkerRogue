package commander

import (
	"game_main/core/common"
	"game_main/tactical/powers/progression"
	"game_main/templates"

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

// SeedStarters appends the starter perk and spell lists from
// templates.GameConfig.Commander to the commander's already-attached
// ProgressionData. Call immediately after CreateCommander when the caller
// wants the default starter library; skip it to leave the commander's
// library empty. No-op if the commander has no ProgressionComponent.
func SeedStarters(commanderID ecs.EntityID, manager *common.EntityManager) {
	data := progression.GetProgression(commanderID, manager)
	if data == nil {
		return
	}
	cfg := templates.GameConfig.Commander
	data.UnlockedPerkIDs = append(data.UnlockedPerkIDs, cfg.StartingPerks...)
	data.UnlockedSpellIDs = append(data.UnlockedSpellIDs, cfg.StartingSpells...)
}
