package artifacts

import "github.com/bytearena/ecs"

// ChargeType determines when an artifact charge refreshes.
type ChargeType int

const (
	ChargeOncePerBattle ChargeType = iota
	ChargeOncePerRound
)

// ArtifactChargeTracker tracks per-battle and per-round charge usage for major
// artifacts and owns the pending-effect queue used by deferred behaviors.
//
// Pending-effect mechanics are encapsulated in PendingEffectQueue — see
// pending_effects.go for the two-phase pattern documentation. The tracker
// delegates its pending-effect methods to the queue so existing callers keep
// working while the mechanics live in a single named file.
type ArtifactChargeTracker struct {
	usedThisBattle map[string]bool
	usedThisRound  map[string]bool
	pending        *PendingEffectQueue
}

// NewArtifactChargeTracker creates a new tracker with initialized state.
func NewArtifactChargeTracker() *ArtifactChargeTracker {
	return &ArtifactChargeTracker{
		usedThisBattle: make(map[string]bool),
		usedThisRound:  make(map[string]bool),
		pending:        NewPendingEffectQueue(),
	}
}

// UseCharge marks a behavior as used for the given charge type.
func (ct *ArtifactChargeTracker) UseCharge(behavior string, chargeType ChargeType) {
	switch chargeType {
	case ChargeOncePerBattle:
		ct.usedThisBattle[behavior] = true
	case ChargeOncePerRound:
		ct.usedThisRound[behavior] = true
	}
}

// IsAvailable returns true if the behavior has not been used this battle or this round.
func (ct *ArtifactChargeTracker) IsAvailable(behavior string) bool {
	return !ct.usedThisBattle[behavior] && !ct.usedThisRound[behavior]
}

// RefreshRoundCharges clears per-round charges. Called at the end of each turn.
func (ct *ArtifactChargeTracker) RefreshRoundCharges() {
	ct.usedThisRound = make(map[string]bool)
}

// --- Pending-effect API (delegates to PendingEffectQueue) ---

// AddPendingEffect queues a deferred artifact effect for later consumption.
func (ct *ArtifactChargeTracker) AddPendingEffect(behavior string, targetSquadID ecs.EntityID) {
	ct.pending.Add(behavior, targetSquadID)
}

// ConsumePendingEffects removes and returns all pending effects matching the given behavior.
func (ct *ArtifactChargeTracker) ConsumePendingEffects(behavior string) []PendingArtifactEffect {
	return ct.pending.Consume(behavior)
}

// HasPendingEffects returns true if there are any pending effects queued.
func (ct *ArtifactChargeTracker) HasPendingEffects() bool {
	return ct.pending.Has()
}

// PendingBehaviorKeys returns the unique behavior keys that have pending effects.
func (ct *ArtifactChargeTracker) PendingBehaviorKeys() []string {
	return ct.pending.Keys()
}

// PendingEffectCount returns the number of pending effects queued.
func (ct *ArtifactChargeTracker) PendingEffectCount() int {
	return ct.pending.Count()
}

// Reset clears all charge tracking state (battle charges, round charges, pending effects).
func (ct *ArtifactChargeTracker) Reset() {
	ct.usedThisBattle = make(map[string]bool)
	ct.usedThisRound = make(map[string]bool)
	ct.pending.Reset()
}
