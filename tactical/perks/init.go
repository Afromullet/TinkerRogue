package perks

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// init registers the perk subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		PerkSlotComponent = em.World.NewComponent()
		PerkRoundStateComponent = em.World.NewComponent()
		PerkUnlockComponent = em.World.NewComponent()

		PerkSlotTag = ecs.BuildTag(PerkSlotComponent)

		em.WorldTags["perkslot"] = PerkSlotTag
	})
}
