package combatlifecycle

import (
	"game_main/core/common"
)

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner CombatTransitioner,
	manager *common.EntityManager,
	starter CombatStarter,
) error {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(CombatStartRollback); ok {
			rb.Rollback()
		}
		return err
	}
	return nil
}
