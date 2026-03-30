package combatservices

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/perks"

	"github.com/bytearena/ecs"
)

// setupPerkDispatch wires perk hook runners into the combat pipeline.
func setupPerkDispatch(cs *CombatService, manager *common.EntityManager) {
	// Direct function assignment — no closure wrappers needed because
	// Run* functions now internalize PerkRoundState lookup.
	callbacks := &combatcore.PerkCallbacks{
		AttackerDamageMod:  perks.RunAttackerDamageModHooks,
		DefenderDamageMod:  perks.RunDefenderDamageModHooks,
		CoverMod:           perks.RunCoverModHooks,
		TargetOverride:     perks.RunTargetOverrideHooks,
		PostDamage:         perks.RunAttackerPostDamageHooks,
		DefenderPostDamage: perks.RunDefenderPostDamageHooks,
		DeathOverride:      perks.RunDeathOverrideHooks,
		CounterMod:         perks.RunCounterModHooks,
		DamageRedirect:     perks.RunDamageRedirectHooks,
	}

	cs.CombatActSystem.SetPerkCallbacks(callbacks)

	// Register perk turn-start hooks on post-reset (runs when a faction's turn starts)
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		roundNumber := cs.TurnManager.GetCurrentRound()

		for _, squadID := range squadIDs {
			roundState := perks.GetRoundState(squadID, manager)
			if roundState == nil {
				continue
			}
			roundState.ResetPerTurn()
			perks.RunTurnStartHooks(squadID, roundNumber, roundState, manager)
		}
	})

	// Register perk round-end hooks on turn end (runs when round advances)
	cs.RegisterOnTurnEnd(func(round int) {
		for _, result := range manager.World.Query(perks.PerkSlotTag) {
			squadID := result.Entity.GetID()
			roundState := perks.GetRoundState(squadID, manager)
			if roundState != nil {
				roundState.ResetPerRound()
			}
		}
	})

	// Register perk combat tracking via attack complete hook
	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult) {
		attackerState := perks.GetRoundState(attackerID, manager)
		if attackerState != nil {
			attackerState.AttackedThisTurn = true
		}

		defenderState := perks.GetRoundState(defenderID, manager)
		if defenderState != nil {
			defenderState.WasAttackedLastTurn = true
		}
	})

	// Register movement tracking for perk state
	cs.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
		roundState := perks.GetRoundState(squadID, manager)
		if roundState != nil {
			roundState.MovedThisTurn = true
			roundState.TurnsStationary = 0
		}
	})
}
