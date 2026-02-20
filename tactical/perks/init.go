package perks

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		SquadPerkComponent = em.World.NewComponent()
		UnitPerkComponent = em.World.NewComponent()
		CommanderPerkComponent = em.World.NewComponent()
		PerkUnlockComponent = em.World.NewComponent()

		SquadPerkTag = ecs.BuildTag(SquadPerkComponent)
		UnitPerkTag = ecs.BuildTag(UnitPerkComponent)
	})
}
