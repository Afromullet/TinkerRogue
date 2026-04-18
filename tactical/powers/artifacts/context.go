package artifacts

import (
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// BehaviorContext bundles runtime dependencies for artifact behavior hooks.
// It embeds powercore.PowerContext by value so the shared fields (Manager,
// Cache, RoundNumber, Logger) live in one place; ChargeTracker is the
// artifact-specific extension. Value embedding keeps zero-value contexts
// usable in tests without additional nil checks.
type BehaviorContext struct {
	powercore.PowerContext
	ChargeTracker *ArtifactChargeTracker
}

// NewBehaviorContext wraps an existing PowerContext with artifact-specific state.
// A nil power argument is treated as a zero-valued PowerContext.
func NewBehaviorContext(power *powercore.PowerContext, chargeTracker *ArtifactChargeTracker) *BehaviorContext {
	ctx := &BehaviorContext{ChargeTracker: chargeTracker}
	if power != nil {
		ctx.PowerContext = *power
	}
	return ctx
}

// SetSquadLocked fully locks a squad so it cannot move or act this turn.
func (ctx *BehaviorContext) SetSquadLocked(squadID ecs.EntityID) {
	actionState := ctx.Cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		return
	}
	actionState.HasActed = true
	actionState.HasMoved = true
	actionState.MovementRemaining = 0
}

// ResetSquadActions fully resets a squad's action state with the given movement speed.
func (ctx *BehaviorContext) ResetSquadActions(squadID ecs.EntityID, speed int) {
	actionState := ctx.Cache.FindActionStateBySquadID(squadID)
	if actionState == nil {
		return
	}
	actionState.HasActed = false
	actionState.HasMoved = false
	actionState.MovementRemaining = speed
}
