package roster

import "game_main/core/common"

// init registers the roster subsystem with the ECS component registry.
func init() {
	common.RegisterSubsystem(func(em *common.EntityManager) {
		UnitRosterComponent = em.World.NewComponent()
		SquadRosterComponent = em.World.NewComponent()
	})
}
