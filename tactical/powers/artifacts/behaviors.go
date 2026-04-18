// Package artifacts — major artifact behaviors.
//
// Each major artifact has exactly one behavior type in this file. Effect
// timing varies:
//   - Immediate: Activate mutates state now (TwinStrike, ChainOfCommand, EchoDrums).
//   - Reactive:  OnAttackComplete mutates state when an event fires (EngagementChains).
//   - Deferred:  Activate queues a pending effect; OnPostReset consumes it on the
//     target faction's next reset (DeadlockShackles, SaboteursHourglass).
//
// The deferred pattern is factored into requireCharge + activateWithPending +
// applyPendingEffects below.
package artifacts

import (
	"fmt"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterBehavior(&EngagementChainsBehavior{})
	RegisterBehavior(&SaboteursHourglassBehavior{})
	RegisterBehavior(&TwinStrikeBehavior{})
	RegisterBehavior(&DeadlockShacklesBehavior{})
	RegisterBehavior(&ChainOfCommandBehavior{})
	RegisterBehavior(&EchoDrumsBehavior{})
}

// ========================================
// Shared helpers (charge + pending flow)
// ========================================

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
	ctx.ChargeTracker.Pending.Add(behaviorKey, targetSquadID)
	ctx.ChargeTracker.UseCharge(behaviorKey, chargeType)
	return nil
}

// applyPendingEffects consumes queued effects for behaviorKey and invokes
// applyFn for each affected squad in squadIDs. When broadcast is true, every
// squad is affected (AOE). When false, only squads whose IDs appear as a
// queued TargetSquadID are affected.
func applyPendingEffects(
	ctx *BehaviorContext,
	behaviorKey string,
	squadIDs []ecs.EntityID,
	broadcast bool,
	applyFn func(actionState *combatstate.ActionStateData, squadID ecs.EntityID),
) {
	if ctx.ChargeTracker == nil {
		return
	}
	pendingEffects := ctx.ChargeTracker.Pending.Consume(behaviorKey)
	if len(pendingEffects) == 0 {
		return
	}

	var targetSet map[ecs.EntityID]bool
	if !broadcast {
		targetSet = make(map[ecs.EntityID]bool, len(pendingEffects))
		for _, pe := range pendingEffects {
			targetSet[pe.TargetSquadID] = true
		}
	}

	for _, sid := range squadIDs {
		if !broadcast && !targetSet[sid] {
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
// EngagementChainsBehavior — full move action after a kill
// ========================================

type EngagementChainsBehavior struct{ BaseBehavior }

func (EngagementChainsBehavior) BehaviorKey() string { return BehaviorEngagementChains }

func (EngagementChainsBehavior) OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
	if !result.TargetDestroyed || result.AttackerDestroyed {
		return
	}
	actionState := ctx.Cache.FindActionStateBySquadID(attackerID)
	if actionState == nil {
		return
	}
	squadSpeed := squadcore.GetSquadMovementSpeedOrDefault(attackerID, ctx.Manager)
	actionState.MovementRemaining = squadSpeed
	actionState.HasMoved = false
	ctx.Log(BehaviorEngagementChains, attackerID, fmt.Sprintf("gets full move action (speed %d)", squadSpeed))
}

// ========================================
// SaboteursHourglassBehavior — -2 movement to all enemy squads via pending effect
// ========================================

type SaboteursHourglassBehavior struct{ BaseBehavior }

func (SaboteursHourglassBehavior) BehaviorKey() string     { return BehaviorSaboteurWsHourglass }
func (SaboteursHourglassBehavior) IsPlayerActivated() bool { return true }

// OnPostReset applies -2 movement to ALL enemy squads (area-of-effect).
// Unlike targeted behaviors, this intentionally ignores pending effect targets.
func (SaboteursHourglassBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	applyPendingEffects(ctx, BehaviorSaboteurWsHourglass, squadIDs, true,
		func(actionState *combatstate.ActionStateData, sid ecs.EntityID) {
			actionState.MovementRemaining -= ArtifactBalance.SaboteursHourglass.MovementReduction
			if actionState.MovementRemaining < 0 {
				actionState.MovementRemaining = 0
			}
			ctx.Log(BehaviorSaboteurWsHourglass, sid, fmt.Sprintf("movement reduced to %d", actionState.MovementRemaining))
		})
}

func (SaboteursHourglassBehavior) Activate(ctx *BehaviorContext, _ ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorSaboteurWsHourglass, ChargeOncePerBattle, 0); err != nil {
		return err
	}
	ctx.Log(BehaviorSaboteurWsHourglass, 0, "activated")
	return nil
}

// ========================================
// TwinStrikeBehavior — grants bonus attack flag to a squad
// ========================================

type TwinStrikeBehavior struct{ BaseBehavior }

func (TwinStrikeBehavior) BehaviorKey() string            { return BehaviorTwinStrike }
func (TwinStrikeBehavior) IsPlayerActivated() bool        { return true }
func (TwinStrikeBehavior) TargetType() BehaviorTargetType { return TargetFriendly }

func (TwinStrikeBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorTwinStrike); err != nil {
		return err
	}
	actionState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if actionState == nil {
		return fmt.Errorf("squad %d has no action state", targetSquadID)
	}
	if !actionState.HasActed {
		return fmt.Errorf("squad %d has not attacked yet this turn", targetSquadID)
	}
	actionState.HasActed = false
	ctx.ChargeTracker.UseCharge(BehaviorTwinStrike, ChargeOncePerBattle)
	ctx.Log(BehaviorTwinStrike, targetSquadID, "activated")
	return nil
}

// ========================================
// DeadlockShacklesBehavior — skip enemy squad's entire activation
// ========================================

type DeadlockShacklesBehavior struct{ BaseBehavior }

func (DeadlockShacklesBehavior) BehaviorKey() string            { return BehaviorDeadlockShackles }
func (DeadlockShacklesBehavior) IsPlayerActivated() bool        { return true }
func (DeadlockShacklesBehavior) TargetType() BehaviorTargetType { return TargetEnemy }

func (DeadlockShacklesBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorDeadlockShackles, ChargeOncePerBattle, targetSquadID); err != nil {
		return err
	}
	ctx.Log(BehaviorDeadlockShackles, targetSquadID, "activated")
	return nil
}

func (DeadlockShacklesBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	applyPendingEffects(ctx, BehaviorDeadlockShackles, squadIDs, false,
		func(_ *combatstate.ActionStateData, sid ecs.EntityID) {
			ctx.SetSquadLocked(sid)
			ctx.Log(BehaviorDeadlockShackles, sid, "squad fully locked this turn")
		})
}

// ========================================
// ChainOfCommandBehavior — pass full action to adjacent squad
// ========================================

type ChainOfCommandBehavior struct{ BaseBehavior }

func (ChainOfCommandBehavior) BehaviorKey() string            { return BehaviorChainOfCommand }
func (ChainOfCommandBehavior) IsPlayerActivated() bool        { return true }
func (ChainOfCommandBehavior) TargetType() BehaviorTargetType { return TargetFriendly }

func (ChainOfCommandBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorChainOfCommand); err != nil {
		return err
	}

	// Find source squad: the faction squad that has this behavior
	targetFaction := combatstate.GetSquadFaction(targetSquadID, ctx.Manager)
	if targetFaction == 0 {
		return fmt.Errorf("target squad %d is not in combat", targetSquadID)
	}
	factionSquads := combatstate.GetActiveSquadsForFaction(targetFaction, ctx.Manager)
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
	ctx.SetSquadLocked(sourceSquadID)

	// Fully reset the target
	squadSpeed := squadcore.GetSquadMovementSpeedOrDefault(targetSquadID, ctx.Manager)
	ctx.ResetSquadActions(targetSquadID, squadSpeed)

	ctx.ChargeTracker.UseCharge(BehaviorChainOfCommand, ChargeOncePerRound)
	ctx.Log(BehaviorChainOfCommand, sourceSquadID, fmt.Sprintf("passes full action to squad %d", targetSquadID))
	return nil
}

// ========================================
// EchoDrumsBehavior — bonus movement phase after full move+attack
// ========================================

type EchoDrumsBehavior struct{ BaseBehavior }

func (EchoDrumsBehavior) BehaviorKey() string            { return BehaviorEchoDrums }
func (EchoDrumsBehavior) IsPlayerActivated() bool        { return true }
func (EchoDrumsBehavior) TargetType() BehaviorTargetType { return TargetFriendly }

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
	squadSpeed := squadcore.GetSquadMovementSpeedOrDefault(targetSquadID, ctx.Manager)
	actionState.HasMoved = false
	actionState.MovementRemaining = squadSpeed
	ctx.ChargeTracker.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
	ctx.Log(BehaviorEchoDrums, targetSquadID, fmt.Sprintf("gets bonus movement phase (speed %d)", squadSpeed))
	return nil
}
