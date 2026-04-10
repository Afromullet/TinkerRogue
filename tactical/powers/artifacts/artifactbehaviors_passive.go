// Package gear — event-driven behaviors (artifactbehaviors_passive.go)
//
// Behaviors in this file fire via event hooks (OnPostReset, OnAttackComplete) rather
// than direct player activation. Some (SaboteursHourglass, TwinStrike) do have
// IsPlayerActivated() = true because the player triggers them, but their effects
// are applied through hooks or immediate state mutation, not through a separate
// "activated" flow.
package artifacts

import (
	"fmt"
	"game_main/tactical/combat/combattypes"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterBehavior(&EngagementChainsBehavior{})
	RegisterBehavior(&SaboteursHourglassBehavior{})
	RegisterBehavior(&TwinStrikeBehavior{})
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
	actionState := ctx.GetActionState(attackerID)
	if actionState == nil {
		return
	}
	squadSpeed := ctx.GetSquadSpeed(attackerID)
	actionState.MovementRemaining = squadSpeed
	actionState.HasMoved = false
	logArtifactActivation(BehaviorEngagementChains, attackerID, fmt.Sprintf("gets full move action (speed %d)", squadSpeed))
}

// ========================================
// SaboteursHourglassBehavior — -2 movement to enemy squads via pending effect
// ========================================

type SaboteursHourglassBehavior struct{ BaseBehavior }

func (SaboteursHourglassBehavior) BehaviorKey() string { return BehaviorSaboteurWsHourglass }

func (SaboteursHourglassBehavior) IsPlayerActivated() bool { return true }

// OnPostReset applies -2 movement to ALL enemy squads (area-of-effect).
// Unlike targeted behaviors, this intentionally ignores pending effect targets.
func (SaboteursHourglassBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	if ctx.ChargeTracker == nil {
		return
	}
	pendingEffects := ctx.ChargeTracker.ConsumePendingEffects(BehaviorSaboteurWsHourglass)
	if len(pendingEffects) == 0 {
		return
	}
	for _, sid := range squadIDs {
		actionState := ctx.GetActionState(sid)
		if actionState == nil {
			continue
		}
		actionState.MovementRemaining -= ArtifactBalance.SaboteursHourglass.MovementReduction
		if actionState.MovementRemaining < 0 {
			actionState.MovementRemaining = 0
		}
		logArtifactActivation(BehaviorSaboteurWsHourglass, sid, fmt.Sprintf("movement reduced to %d", actionState.MovementRemaining))
	}
}

func (SaboteursHourglassBehavior) Activate(ctx *BehaviorContext, _ ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorSaboteurWsHourglass, ChargeOncePerBattle, 0); err != nil {
		return err
	}
	logArtifactActivation(BehaviorSaboteurWsHourglass, 0, "activated")
	return nil
}

// ========================================
// TwinStrikeBehavior — grants bonus attack flag to a squad
// ========================================

type TwinStrikeBehavior struct{ BaseBehavior }

func (TwinStrikeBehavior) BehaviorKey() string     { return BehaviorTwinStrike }
func (TwinStrikeBehavior) IsPlayerActivated() bool { return true }
func (TwinStrikeBehavior) TargetType() BehaviorTargetType { return TargetFriendly }

func (TwinStrikeBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorTwinStrike); err != nil {
		return err
	}
	actionState := ctx.GetActionState(targetSquadID)
	if actionState == nil {
		return fmt.Errorf("squad %d has no action state", targetSquadID)
	}
	if !actionState.HasActed {
		return fmt.Errorf("squad %d has not attacked yet this turn", targetSquadID)
	}
	actionState.HasActed = false
	ctx.ChargeTracker.UseCharge(BehaviorTwinStrike, ChargeOncePerBattle)
	logArtifactActivation(BehaviorTwinStrike, targetSquadID, "activated")
	return nil
}
