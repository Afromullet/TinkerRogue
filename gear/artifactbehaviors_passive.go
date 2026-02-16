// Package gear — event-driven behaviors (artifactbehaviors_passive.go)
//
// Behaviors in this file fire via event hooks (OnPostReset, OnAttackComplete) rather
// than direct player activation. Some (SaboteursHourglass, DoubleTime) do have
// IsPlayerActivated() = true because the player triggers them, but their effects
// are applied through hooks or immediate state mutation, not through a separate
// "activated" flow.
package gear

import (
	"fmt"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterBehavior(&VanguardMovementBehavior{})
	RegisterBehavior(&EngagementChainsBehavior{})
	RegisterBehavior(&MomentumStandardBehavior{})
	RegisterBehavior(&EchoDrumsBehavior{})
	RegisterBehavior(&SaboteursHourglassBehavior{})
	RegisterBehavior(&DoubleTimeBehavior{})
}

// ========================================
// VanguardMovementBehavior — +2 movement to first squad each turn
// ========================================

type VanguardMovementBehavior struct{ BaseBehavior }

func (VanguardMovementBehavior) BehaviorKey() string { return BehaviorVanguardMovement }

func (VanguardMovementBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	if len(squadIDs) == 0 {
		return
	}
	if !HasSpecificArtifactInFaction(squadIDs, "vanguards_oath", ctx.Manager) {
		return
	}
	actionState := ctx.Cache.FindActionStateBySquadID(squadIDs[0])
	if actionState != nil {
		actionState.MovementRemaining += 2
		fmt.Printf("[GEAR] Vanguard's Oath: +2 movement to squad %d (now %d)\n",
			squadIDs[0], actionState.MovementRemaining)
	}
}

// ========================================
// EngagementChainsBehavior — bonus 1-tile move after a kill
// ========================================

type EngagementChainsBehavior struct{ BaseBehavior }

func (EngagementChainsBehavior) BehaviorKey() string { return BehaviorEngagementChains }

func (EngagementChainsBehavior) OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
	if !result.TargetDestroyed || result.AttackerDestroyed {
		return
	}
	if !HasArtifactBehavior(attackerID, BehaviorEngagementChains, ctx.Manager) {
		return
	}
	actionState := ctx.Cache.FindActionStateBySquadID(attackerID)
	if actionState == nil {
		return
	}
	actionState.MovementRemaining = 1
	actionState.HasMoved = false
	fmt.Printf("[GEAR] Forced Engagement Chains: squad %d gets 1 bonus movement tile\n", attackerID)
}

// ========================================
// MomentumStandardBehavior — +1 movement to next friendly squad after a kill
// ========================================

type MomentumStandardBehavior struct{ BaseBehavior }

func (MomentumStandardBehavior) BehaviorKey() string { return BehaviorMomentumStandard }

func (MomentumStandardBehavior) OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
	if !result.TargetDestroyed {
		return
	}
	attackerFaction := ctx.GetSquadFaction(attackerID)
	if attackerFaction == 0 {
		return
	}
	factionSquads := ctx.GetFactionSquads(attackerFaction)
	if !HasBehaviorInFaction(factionSquads, BehaviorMomentumStandard, ctx.Manager) {
		return
	}
	if ctx.ChargeTracker == nil || !ctx.ChargeTracker.IsAvailable(BehaviorMomentumStandard) {
		return
	}
	for _, sid := range factionSquads {
		if sid == attackerID {
			continue
		}
		as := ctx.Cache.FindActionStateBySquadID(sid)
		if as != nil && !as.HasActed {
			as.MovementRemaining += 1
			ctx.ChargeTracker.UseCharge(BehaviorMomentumStandard, ChargeOncePerRound)
			fmt.Printf("[GEAR] Momentum Standard: +1 movement to squad %d (now %d)\n", sid, as.MovementRemaining)
			return
		}
	}
}

// ========================================
// EchoDrumsBehavior — bonus movement phase after full move+attack
// ========================================

type EchoDrumsBehavior struct{ BaseBehavior }

func (EchoDrumsBehavior) BehaviorKey() string { return BehaviorEchoDrums }

func (EchoDrumsBehavior) OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
	if result.AttackerDestroyed {
		return
	}
	actionState := ctx.Cache.FindActionStateBySquadID(attackerID)
	if actionState == nil {
		return
	}
	if !actionState.HasMoved || !actionState.HasActed {
		return
	}
	attackerFaction := ctx.GetSquadFaction(attackerID)
	if attackerFaction == 0 {
		return
	}
	factionSquads := ctx.GetFactionSquads(attackerFaction)
	if !HasBehaviorInFaction(factionSquads, BehaviorEchoDrums, ctx.Manager) {
		return
	}
	if ctx.ChargeTracker == nil || !ctx.ChargeTracker.IsAvailable(BehaviorEchoDrums) {
		return
	}
	squadSpeed := ctx.GetSquadSpeed(attackerID)
	actionState.HasMoved = false
	actionState.MovementRemaining = squadSpeed
	ctx.ChargeTracker.UseCharge(BehaviorEchoDrums, ChargeOncePerRound)
	fmt.Printf("[GEAR] Echo Drums: squad %d gets bonus movement phase (speed %d)\n", attackerID, squadSpeed)
}

// ========================================
// SaboteursHourglassBehavior — -2 movement to enemy squads via pending effect
// ========================================

type SaboteursHourglassBehavior struct{ BaseBehavior }

func (SaboteursHourglassBehavior) BehaviorKey() string { return BehaviorSaboteurWsHourglass }

func (SaboteursHourglassBehavior) IsPlayerActivated() bool { return true }

func (SaboteursHourglassBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	if ctx.ChargeTracker == nil {
		return
	}
	pendingEffects := ctx.ChargeTracker.ConsumePendingEffects(BehaviorSaboteurWsHourglass)
	if len(pendingEffects) == 0 {
		return
	}
	for _, sid := range squadIDs {
		actionState := ctx.Cache.FindActionStateBySquadID(sid)
		if actionState == nil {
			continue
		}
		actionState.MovementRemaining -= 2
		if actionState.MovementRemaining < 0 {
			actionState.MovementRemaining = 0
		}
		fmt.Printf("[GEAR] Saboteur's Hourglass: squad %d movement reduced to %d\n", sid, actionState.MovementRemaining)
	}
}

func (SaboteursHourglassBehavior) Activate(ctx *BehaviorContext, _ ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorSaboteurWsHourglass, ChargeOncePerBattle, 0); err != nil {
		return err
	}
	fmt.Println("[GEAR] Saboteur's Hourglass activated")
	return nil
}

// ========================================
// DoubleTimeBehavior — grants double attack flag to a squad
// ========================================

type DoubleTimeBehavior struct{ BaseBehavior }

func (DoubleTimeBehavior) BehaviorKey() string { return BehaviorDoubleTime }

func (DoubleTimeBehavior) IsPlayerActivated() bool { return true }

func (DoubleTimeBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorDoubleTime); err != nil {
		return err
	}
	actionState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if actionState == nil {
		return fmt.Errorf("squad %d has no action state", targetSquadID)
	}
	if actionState.HasActed {
		return fmt.Errorf("squad %d has already acted this turn", targetSquadID)
	}
	actionState.DoubleTimeActive = true
	ctx.ChargeTracker.UseCharge(BehaviorDoubleTime, ChargeOncePerBattle)
	fmt.Printf("[GEAR] Double Time Drums activated on squad %d\n", targetSquadID)
	return nil
}
