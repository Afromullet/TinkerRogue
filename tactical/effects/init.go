package effects

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		ActiveEffectsComponent = em.World.NewComponent()
		ActiveEffectsTag = ecs.BuildTag(ActiveEffectsComponent)
	})
}
