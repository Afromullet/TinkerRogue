package combatservices

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/perks"

	"github.com/bytearena/ecs"
)

// setupPerkDispatch wires perk hook runners into the combat pipeline.
// This is the integration layer that connects the perks package to combatcore
// without creating circular imports.
func setupPerkDispatch(cs *CombatService, manager *common.EntityManager) {
	callbacks := &combatcore.PerkCallbacks{
		AttackerDamageMod: func(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
			modifiers *combatcore.DamageModifiers, mgr *common.EntityManager) {
			roundState := perks.GetRoundState(attackerSquadID, mgr)
			if roundState != nil {
				perks.RunDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, mgr)
			}
		},

		DefenderDamageMod: func(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
			modifiers *combatcore.DamageModifiers, mgr *common.EntityManager) {
			roundState := perks.GetRoundState(defenderSquadID, mgr)
			if roundState != nil {
				perks.RunDefenderDamageModHooks(attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, roundState, mgr)
			}
			// Also check Reckless Assault vulnerability on the defender
			// (attacker's perk giving defender a damage bonus)
			attackerState := perks.GetRoundState(attackerSquadID, mgr)
			if attackerState != nil && attackerState.RecklessVulnerable {
				// The defender benefits from +20% damage because the attacker has Reckless Assault vulnerability
				// This is already handled inside the recklessAssaultDamageMod behavior
			}
		},

		CoverMod: func(attackerID, defenderID ecs.EntityID,
			cover *combatcore.CoverBreakdown, mgr *common.EntityManager) {
			perks.RunCoverModHooks(attackerID, defenderID, cover, mgr)
		},

		TargetOverride: func(attackerID, defenderSquadID ecs.EntityID,
			targets []ecs.EntityID, mgr *common.EntityManager) []ecs.EntityID {
			return perks.RunTargetOverrideHooks(attackerID, defenderSquadID, targets, mgr)
		},

		PostDamage: func(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
			damage int, wasKill bool, mgr *common.EntityManager) {
			roundState := perks.GetRoundState(attackerSquadID, mgr)
			if roundState != nil {
				perks.RunPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, wasKill, roundState, mgr)
			}
		},

		DefenderPostDamage: func(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
			damage int, wasKill bool, mgr *common.EntityManager) {
			roundState := perks.GetRoundState(defenderSquadID, mgr)
			if roundState != nil {
				perks.RunDefenderPostDamageHooks(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, wasKill, roundState, mgr)
			}
		},

		DeathOverride: func(unitID, squadID ecs.EntityID, mgr *common.EntityManager) bool {
			roundState := perks.GetRoundState(squadID, mgr)
			return perks.RunDeathOverrideHooks(unitID, squadID, roundState, mgr)
		},

		CounterMod: func(defenderSquadID, attackerID ecs.EntityID,
			modifiers *combatcore.DamageModifiers, mgr *common.EntityManager) bool {
			roundState := perks.GetRoundState(defenderSquadID, mgr)
			return perks.RunCounterModHooks(defenderSquadID, attackerID, modifiers, roundState, mgr)
		},
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
			// Reset per-turn state
			roundState.ResetPerTurn()
			// Run TurnStart hooks (Field Medic, Resolute snapshot, Counterpunch, Deadshot)
			perks.RunTurnStartHooks(squadID, roundNumber, roundState, manager)
		}
	})

	// Register perk round-end hooks on turn end (runs when round advances)
	cs.RegisterOnTurnEnd(func(round int) {
		// Reset per-round state for all squads with perks
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
		// Mark attacker's round state that they have attacked
		attackerState := perks.GetRoundState(attackerID, manager)
		if attackerState != nil {
			attackerState.AttackedThisTurn = true
		}

		// Mark defender as having been attacked (for Counterpunch tracking)
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
			// Moving resets Fortify counter
			roundState.TurnsStationary = 0
		}
	})
}
