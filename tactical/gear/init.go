package gear

import (
	"game_main/common"
)

func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		EquipmentComponent = em.World.NewComponent()
		ArtifactInventoryComponent = em.World.NewComponent()
	})
}
