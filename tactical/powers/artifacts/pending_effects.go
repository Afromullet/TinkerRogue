package artifacts

import "github.com/bytearena/ecs"

// PendingEffectQueue implements the two-phase deferred-effect pattern used by
// artifact behaviors whose outcome must land on a *later* dispatch cycle than
// the one that triggered them.
//
// Phase 1 — Queue (during Activate or OnAttackComplete):
//
//	The player activates an artifact (e.g. Deadlock Shackles, Saboteur's
//	Hourglass) that targets an enemy squad. We don't apply the effect now
//	because the target has already had its turn this round; the intent is
//	to affect the target's *next* reset. The activation calls
//	queue.Add(behaviorKey, targetSquadID) and consumes its charge.
//
// Phase 2 — Consume (in the next DispatchPostReset for the target's faction):
//
//	When the target faction's squads are reset, the ArtifactDispatcher iterates
//	queue.Keys() and invokes OnPostReset on each behavior with a pending
//	effect. The behavior calls queue.Consume(behaviorKey) to drain its queued
//	(behaviorKey, targetSquadID) pairs and apply the effect to those targets.
//
// Broadcast effects (e.g. "reduce movement on every enemy squad", Saboteur's
// Hourglass) set BroadcastEffect = true and ignore TargetSquadID. Targeted
// effects set BroadcastEffect = false and store the affected squad in
// TargetSquadID. The two modes never mix within a single behavior's queued
// effects — each behavior is either always-broadcast or always-targeted.
//
// This file isolates the queue mechanics from ArtifactChargeTracker so the
// two-phase flow is named and documented in one place. ArtifactChargeTracker
// composes a queue and delegates its pending-effect methods, preserving the
// public API that tests and behaviors already use.
type PendingEffectQueue struct {
	effects []PendingArtifactEffect
}

// PendingArtifactEffect is a single queued deferred effect:
// "behavior X should apply its OnPostReset work to squad Y on the next
// post-reset for Y's faction." When BroadcastEffect is true, TargetSquadID
// is meaningless and the effect applies to every squad in the consuming
// faction.
type PendingArtifactEffect struct {
	Behavior        string
	TargetSquadID   ecs.EntityID
	BroadcastEffect bool
}

// NewPendingEffectQueue returns an empty queue.
func NewPendingEffectQueue() *PendingEffectQueue {
	return &PendingEffectQueue{}
}

// Add queues a deferred effect targeted at a specific squad.
func (q *PendingEffectQueue) Add(behavior string, targetSquadID ecs.EntityID) {
	q.effects = append(q.effects, PendingArtifactEffect{
		Behavior:      behavior,
		TargetSquadID: targetSquadID,
	})
}

// AddBroadcast queues a deferred effect that should fire against every squad
// in the consuming faction (AOE), not a specific target.
func (q *PendingEffectQueue) AddBroadcast(behavior string) {
	q.effects = append(q.effects, PendingArtifactEffect{
		Behavior:        behavior,
		BroadcastEffect: true,
	})
}

// Consume removes and returns all pending effects matching the given behavior.
func (q *PendingEffectQueue) Consume(behavior string) []PendingArtifactEffect {
	var matched []PendingArtifactEffect
	var remaining []PendingArtifactEffect
	for _, pe := range q.effects {
		if pe.Behavior == behavior {
			matched = append(matched, pe)
		} else {
			remaining = append(remaining, pe)
		}
	}
	q.effects = remaining
	return matched
}

// Has returns true if any effects are queued.
func (q *PendingEffectQueue) Has() bool {
	return len(q.effects) > 0
}

// Keys returns the unique behavior keys that have pending effects.
func (q *PendingEffectQueue) Keys() []string {
	seen := make(map[string]bool)
	var keys []string
	for _, pe := range q.effects {
		if !seen[pe.Behavior] {
			seen[pe.Behavior] = true
			keys = append(keys, pe.Behavior)
		}
	}
	return keys
}

// Count returns the total number of queued effects.
func (q *PendingEffectQueue) Count() int {
	return len(q.effects)
}

// Reset clears the queue. Called at battle start via ArtifactChargeTracker.Reset.
func (q *PendingEffectQueue) Reset() {
	q.effects = nil
}
