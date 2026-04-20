package artifacts

import (
	"game_main/core/common"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		EquipmentComponent = em.World.NewComponent()
		ArtifactInventoryComponent = em.World.NewComponent()
	})
}
