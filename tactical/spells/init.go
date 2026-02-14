package spells

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

// init registers the spell subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		ManaComponent = em.World.NewComponent()
		SpellBookComponent = em.World.NewComponent()

		ManaTag = ecs.BuildTag(ManaComponent)
		SpellBookTag = ecs.BuildTag(SpellBookComponent)

		em.WorldTags["mana"] = ManaTag
		em.WorldTags["spellbook"] = SpellBookTag
	})
}
