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

	// Wire artifact activation logger for combat feedback.
	artifacts.SetArtifactLogger(func(behaviorKey string, squadID ecs.EntityID, message string) {
		fmt.Printf("[GEAR] %s: %s (squad %d)\n", behaviorKey, message, squadID)
	})

	// OnPostReset remains broadcast: pending-effect behaviors (DeadlockShackles,
	// SaboteursHourglass) fire on the enemy faction's reset, not the owning
	// squad's. The pending-effect queue already gates correctness.
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		cs.artifactDispatcher.DispatchPostReset(factionID, squadIDs)
	})

	// Squad-scoped: only run behaviors equipped on the attacker.
	// If a future behavior needs to trigger on defender's artifacts,
	// add a second loop over GetEquippedBehaviors(defenderID, manager).
	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult) {
		cs.artifactDispatcher.DispatchOnAttackComplete(attackerID, defenderID, result)
	})

	// OnTurnEnd remains broadcast: charge refresh + behavior hooks.
	cs.RegisterOnTurnEnd(func(round int) {
		cs.artifactDispatcher.DispatchOnTurnEnd(round)
	})

	// ==========================================
	// Phase 2: Perk hook dispatch
	// ==========================================

	// Wire perk activation logger for combat feedback.
	perks.SetPerkLogger(func(perkID perks.PerkID, squadID ecs.EntityID, message string) {
		fmt.Printf("[PERK] %s: %s (squad %d)\n", perkID, message, squadID)
	})

	// Wire perk dispatcher into the damage pipeline and lifecycle hooks.
	perkDispatcher := &perks.SquadPerkDispatcher{}
	cs.CombatActSystem.SetPerkDispatcher(perkDispatcher)

	// Register perk turn-start hooks on post-reset (runs when a faction's turn starts).
	// Fires AFTER artifact PostReset hooks above.
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		perkDispatcher.DispatchTurnStart(squadIDs, cs.TurnManager.GetCurrentRound(), manager)
	})

	// Register perk round-end hooks on turn end (runs when round advances).
	// Fires AFTER artifact OnTurnEnd hooks above.
	cs.RegisterOnTurnEnd(func(round int) {
		perkDispatcher.DispatchRoundEnd(manager)
	})

	// Register perk combat tracking via attack complete hook.
	// Fires AFTER artifact OnAttackComplete hooks above.
	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combatcore.CombatResult) {
		perkDispatcher.DispatchAttackTracking(attackerID, defenderID, manager)
	})

	// Register movement tracking for perk state (no artifact hook for this event).
	cs.RegisterOnMoveComplete(func(squadID ecs.EntityID) {
		perkDispatcher.DispatchMoveTracking(squadID, manager)
	})
}
