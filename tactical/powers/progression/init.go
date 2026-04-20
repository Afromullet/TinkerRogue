package progression

import (
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		ProgressionComponent = em.World.NewComponent()
		ProgressionTag = ecs.BuildTag(ProgressionComponent)
		em.WorldTags["progression"] = ProgressionTag
	})
}
