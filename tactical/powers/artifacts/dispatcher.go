package artifacts

import (
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// ArtifactDispatcher encapsulates artifact behavior dispatch for combat lifecycle events.
//
// The charge tracker is required at construction. A previous design exposed
// SetChargeTracker, which allowed the dispatcher to silently no-op turn-end
// charge refreshes if a caller forgot the wiring. Per-battle state is now
// reset in place via tracker.Reset() rather than by swapping the tracker.
type ArtifactDispatcher struct {
	manager       *common.EntityManager
	cache         *combatstate.CombatQueryCache
	chargeTracker *ArtifactChargeTracker
	logger        powercore.PowerLogger
}

// NewArtifactDispatcher constructs a dispatcher. chargeTracker must not be nil —
// a nil tracker would silently disable charge refreshes and every charge-gated
// activation. Panics on nil to surface wiring bugs at startup.
func NewArtifactDispatcher(manager *common.EntityManager, cache *combatstate.CombatQueryCache, chargeTracker *ArtifactChargeTracker) *ArtifactDispatcher {
	if chargeTracker == nil {
		panic("artifacts.NewArtifactDispatcher: chargeTracker must not be nil")
	}
	return &ArtifactDispatcher{
		manager:       manager,
		cache:         cache,
		chargeTracker: chargeTracker,
	}
}

// SetLogger injects the PowerLogger used by behavior activations.
func (d *ArtifactDispatcher) SetLogger(logger powercore.PowerLogger) {
	d.logger = logger
}

// ChargeTracker returns the tracker this dispatcher holds. Exposed so
// CombatService can reset per-battle state without replacing the dispatcher
// (which would invalidate the PowerPipeline subscriber bindings).
func (d *ArtifactDispatcher) ChargeTracker() *ArtifactChargeTracker {
	return d.chargeTracker
}

func (d *ArtifactDispatcher) makeBehaviorContext(round int) *BehaviorContext {
	return NewBehaviorContext(
		powercore.NewPowerContext(d.manager, d.cache, round, d.logger),
		d.chargeTracker,
	)
}

// DispatchPostReset fires OnPostReset for equipped behaviors and behaviors with pending effects.
// Only calls behaviors that are relevant: either equipped on a squad in this faction,
// or have pending effects queued from a previous activation (e.g., Deadlock Shackles).
func (d *ArtifactDispatcher) DispatchPostReset(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	ctx := d.makeBehaviorContext(0)
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

// DispatchOnAttackComplete fires OnAttackComplete for behaviors equipped on the attacker.
func (d *ArtifactDispatcher) DispatchOnAttackComplete(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
	ctx := d.makeBehaviorContext(0)
	for _, b := range GetEquippedBehaviors(attackerID, d.manager) {
		b.OnAttackComplete(ctx, attackerID, defenderID, result)
	}
}

// DispatchOnTurnEnd fires OnTurnEnd for equipped behaviors and refreshes round charges.
func (d *ArtifactDispatcher) DispatchOnTurnEnd(round int) {
	d.chargeTracker.RefreshRoundCharges()
	ctx := d.makeBehaviorContext(round)
	for _, b := range AllBehaviors() {
		b.OnTurnEnd(ctx, round)
	}
}
