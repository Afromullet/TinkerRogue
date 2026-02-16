package gear

import "github.com/bytearena/ecs"

// ChargeType determines when an artifact charge refreshes.
type ChargeType int

const (
	ChargeOncePerBattle ChargeType = iota
	ChargeOncePerRound
)

// PendingArtifactEffect represents a deferred artifact effect to be consumed later
// (e.g., Saboteur's Hourglass applies at the start of the enemy's next turn).
type PendingArtifactEffect struct {
	Behavior      string
	TargetSquadID ecs.EntityID
}

// ArtifactChargeTracker tracks per-battle and per-round charge usage for major artifacts.
type ArtifactChargeTracker struct {
	usedThisBattle map[string]bool
	usedThisRound  map[string]bool
	pendingEffects []PendingArtifactEffect
}

// NewArtifactChargeTracker creates a new tracker with initialized maps.
func NewArtifactChargeTracker() *ArtifactChargeTracker {
	return &ArtifactChargeTracker{
		usedThisBattle: make(map[string]bool),
		usedThisRound:  make(map[string]bool),
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

// AddPendingEffect queues a deferred artifact effect for later consumption.
func (ct *ArtifactChargeTracker) AddPendingEffect(behavior string, targetSquadID ecs.EntityID) {
	ct.pendingEffects = append(ct.pendingEffects, PendingArtifactEffect{
		Behavior:      behavior,
		TargetSquadID: targetSquadID,
	})
}

// ConsumePendingEffects removes and returns all pending effects matching the given behavior.
func (ct *ArtifactChargeTracker) ConsumePendingEffects(behavior string) []PendingArtifactEffect {
	var matched []PendingArtifactEffect
	var remaining []PendingArtifactEffect
	for _, pe := range ct.pendingEffects {
		if pe.Behavior == behavior {
			matched = append(matched, pe)
		} else {
			remaining = append(remaining, pe)
		}
	}
	ct.pendingEffects = remaining
	return matched
}

// HasPendingEffects returns true if there are any pending effects queued.
func (ct *ArtifactChargeTracker) HasPendingEffects() bool {
	return len(ct.pendingEffects) > 0
}

// PendingEffectCount returns the number of pending effects queued.
func (ct *ArtifactChargeTracker) PendingEffectCount() int {
	return len(ct.pendingEffects)
}

// Reset clears all charge tracking state (battle charges, round charges, pending effects).
func (ct *ArtifactChargeTracker) Reset() {
	ct.usedThisBattle = make(map[string]bool)
	ct.usedThisRound = make(map[string]bool)
	ct.pendingEffects = nil
}
