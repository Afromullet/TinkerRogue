package combatpipeline

import (
	"game_main/common"
	"game_main/tactical/combat"
)

// Type aliases for backward compatibility — existing callers in mind/ packages
// can continue using combatpipeline.CombatSetup etc. without changes.
type CombatStarter = combat.CombatStarter
type CombatSetup = combat.CombatSetup
type CombatTransitioner = combat.CombatTransitioner
type CombatStartRollback = combat.CombatStartRollback
type CombatStartResult = combat.CombatStartResult

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner CombatTransitioner,
	manager *common.EntityManager,
	starter CombatStarter,
) (*CombatStartResult, error) {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return nil, err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(CombatStartRollback); ok {
			rb.Rollback()
		}
		return nil, err
	}
	return &CombatStartResult{
		PlayerFactionID: setup.PlayerFactionID,
		EnemyFactionID:  setup.EnemyFactionID,
		EnemySquadIDs:   setup.EnemySquadIDs,
	}, nil
}
