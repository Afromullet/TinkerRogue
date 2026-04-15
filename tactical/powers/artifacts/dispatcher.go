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

// DispatchPostReset fires OnPostReset for equipped behaviors and behaviors with pending effects.
// Only calls behaviors that are relevant: either equipped on a squad in this faction,
// or have pending effects queued from a previous activation (e.g., Deadlock Shackles).
func (d *ArtifactDispatcher) DispatchPostReset(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	ctx := d.makeBehaviorContext()
	fired := make(map[string]bool)

	// Fire behaviors equipped on squads in this faction
	for _, squadID := range squadIDs {
		for _, b := range GetEquippedBehaviors(squadID, d.manager) {
			key := b.BehaviorKey()
			if !fired[key] {
				fired[key] = true
				b.OnPostReset(ctx, factionID, squadIDs)
			}
		}
	}

	// Fire behaviors with pending effects (these target enemy squads, not the equipping squad)
	if d.chargeTracker != nil {
		for _, key := range d.chargeTracker.PendingBehaviorKeys() {
			if !fired[key] {
				b := GetBehavior(key)
				if b != nil {
					fired[key] = true
					b.OnPostReset(ctx, factionID, squadIDs)
				}
			}
		}
	}
}

// DispatchOnAttackComplete fires OnAttackComplete for behaviors equipped on the attacker.
func (d *ArtifactDispatcher) DispatchOnAttackComplete(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
	ctx := d.makeBehaviorContext()
	for _, b := range GetEquippedBehaviors(attackerID, d.manager) {
		b.OnAttackComplete(ctx, attackerID, defenderID, result)
	}
}

// DispatchOnTurnEnd fires OnTurnEnd for equipped behaviors and refreshes round charges.
func (d *ArtifactDispatcher) DispatchOnTurnEnd(round int) {
	if d.chargeTracker != nil {
		d.chargeTracker.RefreshRoundCharges()
	}
	ctx := d.makeBehaviorContext()
	for _, b := range AllBehaviors() {
		b.OnTurnEnd(ctx, round)
	}
}
