// Package gear — activated behaviors (artifactbehaviors_activated.go)
//
// "Activated" behaviors are triggered via direct player activation (Activate method)
// or react to combat events (OnAttackComplete). They are grouped here because their
// primary effect path is through explicit activation rather than passive turn hooks.
package gear

import (
	"fmt"
	"game_main/tactical/combat"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterBehavior(&DeadlockShacklesBehavior{})
	RegisterBehavior(&ChainOfCommandBehavior{})
	RegisterBehavior(&EchoDrumsBehavior{})
}

// requireCharge checks that the behavior's charge is available, returning a
// standard error if not.
func requireCharge(ctx *BehaviorContext, behaviorKey string) error {
	if !ctx.ChargeTracker.IsAvailable(behaviorKey) {
		return fmt.Errorf("%s already used this battle", behaviorKey)
	}
	return nil
}

// activateWithPending is the common pattern for behaviors that queue a pending
// effect and consume a charge on activation (Deadlock, Saboteur).
func activateWithPending(ctx *BehaviorContext, behaviorKey string, chargeType ChargeType, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, behaviorKey); err != nil {
		return err
	}
	ctx.ChargeTracker.AddPendingEffect(behaviorKey, targetSquadID)
	ctx.ChargeTracker.UseCharge(behaviorKey, chargeType)
	return nil
}

// applyPendingToTargets consumes pending effects for the given behavior key,
// builds a target set, and calls applyFn for each matching squad's action state.
func applyPendingToTargets(
	ctx *BehaviorContext,
	behaviorKey string,
	squadIDs []ecs.EntityID,
	applyFn func(actionState *combat.ActionStateData, squadID ecs.EntityID),
) {
	if ctx.ChargeTracker == nil {
		return
	}
	pendingEffects := ctx.ChargeTracker.ConsumePendingEffects(behaviorKey)
	if len(pendingEffects) == 0 {
		return
	}
	targetSet := make(map[ecs.EntityID]bool, len(pendingEffects))
	for _, pe := range pendingEffects {
		targetSet[pe.TargetSquadID] = true
	}
	for _, sid := range squadIDs {
		if !targetSet[sid] {
			continue
		}
		actionState := ctx.Cache.FindActionStateBySquadID(sid)
		if actionState == nil {
			continue
		}
		applyFn(actionState, sid)
	}
}

// ========================================
// DeadlockShacklesBehavior — skip enemy squad's entire activation
// ========================================

type DeadlockShacklesBehavior struct{ BaseBehavior }

func (DeadlockShacklesBehavior) BehaviorKey() string     { return BehaviorDeadlockShackles }
func (DeadlockShacklesBehavior) IsPlayerActivated() bool { return true }
func (DeadlockShacklesBehavior) TargetType() int         { return TargetEnemy }

func (DeadlockShacklesBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorDeadlockShackles, ChargeOncePerBattle, targetSquadID); err != nil {
		return err
	}
	fmt.Printf("[GEAR] Deadlock Shackles activated targeting squad %d\n", targetSquadID)
	return nil
}

func (DeadlockShacklesBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	applyPendingToTargets(ctx, BehaviorDeadlockShackles, squadIDs, func(actionState *combat.ActionStateData, sid ecs.EntityID) {
		actionState.HasActed = true
		actionState.HasMoved = true
		actionState.MovementRemaining = 0
		fmt.Printf("[GEAR] Deadlock Shackles: squad %d fully locked this turn\n", sid)
	})
}

// ========================================
// ChainOfCommandBehavior — pass attack to adjacent squad
// ========================================

type ChainOfCommandBehavior struct{ BaseBehavior }

func (ChainOfCommandBehavior) BehaviorKey() string     { return BehaviorChainOfCommand }
func (ChainOfCommandBehavior) IsPlayerActivated() bool { return true }
func (ChainOfCommandBehavior) TargetType() int         { return TargetFriendly }

func (ChainOfCommandBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorChainOfCommand); err != nil {
		return err
	}

	// Find source squad: the faction squad that has this behavior
	targetFaction := ctx.GetSquadFaction(targetSquadID)
	if targetFaction == 0 {
		return fmt.Errorf("target squad %d is not in combat", targetSquadID)
	}
	factionSquads := ctx.GetFactionSquads(targetFaction)
	sourceSquadID := GetFactionSquadWithBehavior(factionSquads, BehaviorChainOfCommand, ctx.Manager)
	if sourceSquadID == 0 {
		return fmt.Errorf("no squad with %s behavior in faction", BehaviorChainOfCommand)
	}
	if sourceSquadID == targetSquadID {
		return fmt.Errorf("cannot pass action to self")
	}

	// Validate source is fresh (hasn't moved or acted)
	sourceState := ctx.Cache.FindActionStateBySquadID(sourceSquadID)
	if sourceState == nil {
		return fmt.Errorf("source squad %d has no action state", sourceSquadID)
	}
	if sourceState.HasActed {
		return fmt.Errorf("source squad %d has already acted", sourceSquadID)
	}
	if sourceState.HasMoved {
		return fmt.Errorf("source squad %d has already moved", sourceSquadID)
	}

	// Validate target has acted
	targetState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if targetState == nil {
		return fmt.Errorf("target squad %d has no action state", targetSquadID)
	}
	if !targetState.HasActed {
		return fmt.Errorf("target squad %d has not acted yet", targetSquadID)
	}

	// Fully spend the source
	sourceState.HasActed = true
	sourceState.HasMoved = true
	sourceState.MovementRemaining = 0

	// Fully reset the target
	squadSpeed := ctx.GetSquadSpeed(targetSquadID)
	targetState.HasActed = false
	targetState.HasMoved = false
	targetState.MovementRemaining = squadSpeed

	ctx.ChargeTracker.UseCharge(BehaviorChainOfCommand, ChargeOncePerRound)
	fmt.Printf("[GEAR] Chain of Command: squad %d passes full action to squad %d\n", sourceSquadID, targetSquadID)
	return nil
}

// ========================================
// EchoDrumsBehavior — bonus movement phase after full move+attack
// ========================================

type EchoDrumsBehavior struct{ BaseBehavior }

func (EchoDrumsBehavior) BehaviorKey() string     { return BehaviorEchoDrums }
func (EchoDrumsBehavior) IsPlayerActivated() bool { return true }
func (EchoDrumsBehavior) TargetType() int         { return TargetFriendly }

func (EchoDrumsBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorEchoDrums); err != nil {
		return err
	}
	actionState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if actionState == nil {
		return fmt.Errorf("squad %d has no action state", targetSquadID)
	}
	if !actionState.HasMoved || !actionState.HasActed {
		return fmt.Errorf("squad %d must have moved and attacked first", targetSquadID)
	}
	squadSpeed := ctx.GetSquadSpeed(targetSquadID)
	actionState.HasMoved = false
	actionState.MovementRemaining = squadSpeed
	ctx.ChargeTracker.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
	fmt.Printf("[GEAR] Echo Drums: squad %d gets bonus movement phase (speed %d)\n", targetSquadID, squadSpeed)
	return nil
}
