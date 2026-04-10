package artifacts

import (
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"

	"github.com/bytearena/ecs"
)

// ArtifactDispatcher encapsulates artifact behavior dispatch for combat lifecycle events.
// Created per-battle with a reference to the charge tracker.
type ArtifactDispatcher struct {
	manager       *common.EntityManager
	cache         *combatstate.CombatQueryCache
	chargeTracker *ArtifactChargeTracker
}

// NewArtifactDispatcher creates a dispatcher for the current battle.
func NewArtifactDispatcher(manager *common.EntityManager, cache *combatstate.CombatQueryCache) *ArtifactDispatcher {
	return &ArtifactDispatcher{manager: manager, cache: cache}
}

// SetChargeTracker updates the charge tracker (called when a new battle starts).
func (d *ArtifactDispatcher) SetChargeTracker(ct *ArtifactChargeTracker) {
	d.chargeTracker = ct
}

func (d *ArtifactDispatcher) makeBehaviorContext() *BehaviorContext {
	return NewBehaviorContext(d.manager, d.cache, d.chargeTracker)
}

// DispatchPostReset fires OnPostReset for all registered behaviors (broadcast).
func (d *ArtifactDispatcher) DispatchPostReset(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	ctx := d.makeBehaviorContext()
	for _, b := range AllBehaviors() {
		b.OnPostReset(ctx, factionID, squadIDs)
	}
}

// DispatchOnAttackComplete fires OnAttackComplete for behaviors equipped on the attacker.
func (d *ArtifactDispatcher) DispatchOnAttackComplete(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
	ctx := d.makeBehaviorContext()
	for _, b := range GetEquippedBehaviors(attackerID, d.manager) {
		b.OnAttackComplete(ctx, attackerID, defenderID, result)
	}
}

// DispatchOnTurnEnd fires OnTurnEnd for all behaviors and refreshes round charges.
func (d *ArtifactDispatcher) DispatchOnTurnEnd(round int) {
	if d.chargeTracker != nil {
		d.chargeTracker.RefreshRoundCharges()
	}
	ctx := d.makeBehaviorContext()
	for _, b := range AllBehaviors() {
		b.OnTurnEnd(ctx, round)
	}
}
