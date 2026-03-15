package combatlifecycle

import (
	"game_main/common"
	"game_main/tactical/combat"
)

// ExecuteCombatStart is THE single entry point for all combat initiation.
func ExecuteCombatStart(
	transitioner combat.CombatTransitioner,
	manager *common.EntityManager,
	starter combat.CombatStarter,
) (*combat.CombatStartResult, error) {
	setup, err := starter.Prepare(manager)
	if err != nil {
		return nil, err
	}
	if err := transitioner.TransitionToCombat(setup); err != nil {
		// Let the starter undo any side effects from Prepare (e.g., hidden sprites)
		if rb, ok := starter.(combat.CombatStartRollback); ok {
			rb.Rollback()
		}
		return nil, err
	}
	return &combat.CombatStartResult{
		PlayerFactionID: setup.PlayerFactionID,
		EnemyFactionID:  setup.EnemyFactionID,
		EnemySquadIDs:   setup.EnemySquadIDs,
	}, nil
}
