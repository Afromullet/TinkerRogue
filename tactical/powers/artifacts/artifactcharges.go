package artifacts

// ChargeType determines when an artifact charge refreshes.
type ChargeType int

const (
	ChargeOncePerBattle ChargeType = iota
	ChargeOncePerRound
)

// ArtifactChargeTracker tracks per-battle and per-round charge usage for major
// artifacts. Deferred-effect mechanics live on the embedded Pending queue —
// see pending_effects.go for the two-phase pattern documentation. Callers
// reach the queue directly via tracker.Pending (no forwarding methods).
type ArtifactChargeTracker struct {
	usedThisBattle map[string]bool
	usedThisRound  map[string]bool
	Pending        *PendingEffectQueue
}

// NewArtifactChargeTracker creates a new tracker with initialized state.
func NewArtifactChargeTracker() *ArtifactChargeTracker {
	return &ArtifactChargeTracker{
		usedThisBattle: make(map[string]bool),
		usedThisRound:  make(map[string]bool),
		Pending:        NewPendingEffectQueue(),
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

// Reset clears all charge tracking state (battle charges, round charges, pending effects).
func (ct *ArtifactChargeTracker) Reset() {
	ct.usedThisBattle = make(map[string]bool)
	ct.usedThisRound = make(map[string]bool)
	ct.Pending.Reset()
}
