package perks

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// init registers the perk subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		PerkSlotComponent = em.World.NewComponent()
		PerkRoundStateComponent = em.World.NewComponent()

		PerkSlotTag = ecs.BuildTag(PerkSlotComponent)

		em.WorldTags["perkslot"] = PerkSlotTag
	})
}

// RemovePerkRoundState removes PerkRoundStateComponent from the entity if
// present. PerkRoundStateComponent tracks per-round perk activations during a
// single combat encounter and must be stripped when the squad leaves combat.
// Called from combat-exit orchestration (combatlifecycle.cleanup and
// CombatService.TeardownCombat).
func RemovePerkRoundState(entity *ecs.Entity) {
	if entity == nil {
		return
	}
	if entity.HasComponent(PerkRoundStateComponent) {
		entity.RemoveComponent(PerkRoundStateComponent)
	}
}
