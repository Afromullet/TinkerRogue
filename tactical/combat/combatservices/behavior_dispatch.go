package combatservices

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/powers/artifacts"

	"github.com/bytearena/ecs"
)

// setupBehaviorDispatch wires all registered artifact behaviors to the combat event system.
func setupBehaviorDispatch(cs *CombatService, manager *common.EntityManager, cache *combatcore.CombatQueryCache) {
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
}
