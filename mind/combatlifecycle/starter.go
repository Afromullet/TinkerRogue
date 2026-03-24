package combatlifecycle

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
)

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner combatcore.CombatTransitioner,
	manager *common.EntityManager,
	starter combatcore.CombatStarter,
) error {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(combatcore.CombatStartRollback); ok {
			rb.Rollback()
		}
		return err
	}
	return nil
}
