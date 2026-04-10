package combatlifecycle

import (
	"game_main/common"
	"game_main/tactical/combat/combattypes"
)

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner combattypes.CombatTransitioner,
	manager *common.EntityManager,
	starter combattypes.CombatStarter,
) error {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(combattypes.CombatStartRollback); ok {
			rb.Rollback()
		}
		return err
	}
	return nil
}
