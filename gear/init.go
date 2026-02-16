package gear

import (
	"game_main/common"

	"github.com/bytearena/ecs"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		EquipmentComponent = em.World.NewComponent()
		EquipmentTag = ecs.BuildTag(EquipmentComponent)
		em.WorldTags["equipment"] = EquipmentTag
		ArtifactInventoryComponent = em.World.NewComponent()
	})
}
