package combatservices

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/perks"

	"github.com/bytearena/ecs"
)

// setupPowerDispatch wires both artifact behaviors and perk hooks into the combat
// event system. Artifact dispatch registers first so that artifact effects (e.g.,
// Deadlock Shackles locking a squad) resolve before perk turn-start hooks.
//
// Execution order per event:
//
//	PostReset:       artifacts.OnPostReset → perks.TurnStart
//	OnAttackComplete: artifacts.OnAttackComplete → perks state tracking
//	OnTurnEnd:       artifacts charge refresh + OnTurnEnd → perks round reset
//	OnMoveComplete:  perks movement tracking (no artifact hook)
func setupPowerDispatch(cs *CombatService, manager *common.EntityManager, cache *combatcore.CombatQueryCache) {

	// ==========================================
	// Phase 1: Artifact behavior dispatch
	// ==========================================

	makeBehaviorContext := func() *artifacts.BehaviorContext {
		return artifacts.NewBehaviorContext(manager, cache, cs.chargeTracker)
	}

	// OnPostReset remains broadcast: pending-effect behaviors (DeadlockShackles,
	// SaboteursHourglass) fire on the enemy faction's reset, not the owning
	// squad's. The pending-effect queue already gates correctness.
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		ctx := makeBehaviorContext()
		for _, b := range artifacts.AllBehaviors() {
			b.OnPostReset(ctx, factionID, squadIDs)
		}
	})

	// Squad-scoped: only run behaviors equipped on the attacker.
	// If a future behavior needs to trigger on defender's artifacts,
	// add a second loop over GetEquippedBehaviors(defenderID, manager).
	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult) {
		ctx := makeBehaviorContext()
		for _, b := range artifacts.GetEquippedBehaviors(attackerID, manager) {
			b.OnAttackComplete(ctx, attackerID, defenderID, result)
		}
	})

	// OnTurnEnd remains broadcast: no behaviors override it currently.
	// RefreshRoundCharges is global (not per-squad).
	cs.RegisterOnTurnEnd(func(round int) {
		if cs.chargeTracker != nil {
			cs.chargeTracker.RefreshRoundCharges()
		}
		ctx := makeBehaviorContext()
		for _, b := range artifacts.AllBehaviors() {
			b.OnTurnEnd(ctx, round)
		}
	})

	// ==========================================
	// Phase 2: Perk hook dispatch
	// ==========================================

	// Wire perk activation logger for combat feedback.
	perks.SetPerkLogger(func(perkID string, squadID ecs.EntityID, message string) {
		fmt.Printf("[PERK] %s: %s (squad %d)\n", perkID, message, squadID)
	})

	// Wire perk dispatcher into the damage pipeline.
	cs.CombatActSystem.SetPerkDispatcher(&perks.SquadPerkDispatcher{})

	// Register perk turn-start hooks on post-reset (runs when a faction's turn starts).
	// Fires AFTER artifact PostReset hooks above.
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		roundNumber := cs.TurnManager.GetCurrentRound()

		for _, squadID := range squadIDs {
			roundState := perks.GetRoundState(squadID, manager)
			if roundState == nil {
				continue
			}
			perks.ResetPerkRoundStateTurn(roundState)
			perks.RunTurnStartHooks(squadID, roundNumber, roundState, manager)
		}
	})

	// Register perk round-end hooks on turn end (runs when round advances).
	// Fires AFTER artifact OnTurnEnd hooks above.
	cs.RegisterOnTurnEnd(func(round int) {
		for _, result := range manager.World.Query(perks.PerkSlotTag) {
			squadID := result.Entity.GetID()
			roundState := perks.GetRoundState(squadID, manager)
			if roundState != nil {
				perks.ResetPerkRoundStateRound(roundState)
			}
		}
	})

	// Register perk combat tracking via attack complete hook.
	// Fires AFTER artifact OnAttackComplete hooks above.
	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult) {
		attackerState := perks.GetRoundState(attackerID, manager)
		if attackerState != nil {
			attackerState.AttackedThisTurn = true
		}

		defenderState := perks.GetRoundState(defenderID, manager)
		if defenderState != nil {
			defenderState.WasAttackedThisTurn = true
		}
	})

	// Register movement tracking for perk state (no artifact hook for this event).
	cs.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
		roundState := perks.GetRoundState(squadID, manager)
		if roundState != nil {
			roundState.MovedThisTurn = true
			roundState.TurnsStationary = 0
		}
	})
}
