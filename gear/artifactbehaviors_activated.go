// Package gear — activated behaviors (artifactbehaviors_activated.go)
//
// "Activated" behaviors are triggered via direct player activation (Activate method)
// or react to combat events (OnAttackComplete). They are grouped here because their
// primary effect path is through explicit activation rather than passive turn hooks.
package gear

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func init() {
	RegisterBehavior(&StandDownBehavior{})
	RegisterBehavior(&DeadlockShacklesBehavior{})
	RegisterBehavior(&AnthemPerseveranceBehavior{})
	RegisterBehavior(&RallyingHornBehavior{})
	RegisterBehavior(&ChainOfCommandBehavior{})
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
// effect and consume a charge on activation (StandDown, Deadlock, Saboteur).
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
// StandDownBehavior — force enemy squad to skip attack
// ========================================

type StandDownBehavior struct{ BaseBehavior }

func (StandDownBehavior) BehaviorKey() string      { return BehaviorStandDown }
func (StandDownBehavior) IsPlayerActivated() bool   { return true }

func (StandDownBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := activateWithPending(ctx, BehaviorStandDown, ChargeOncePerBattle, targetSquadID); err != nil {
		return err
	}
	fmt.Printf("[GEAR] Stand Down Orders activated targeting squad %d\n", targetSquadID)
	return nil
}

func (StandDownBehavior) OnPostReset(ctx *BehaviorContext, factionID ecs.EntityID, squadIDs []ecs.EntityID) {
	applyPendingToTargets(ctx, BehaviorStandDown, squadIDs, func(actionState *combat.ActionStateData, sid ecs.EntityID) {
		actionState.HasActed = true
		fmt.Printf("[GEAR] Stand Down Orders: squad %d cannot attack this turn\n", sid)
	})
}

// ========================================
// DeadlockShacklesBehavior — skip enemy squad's entire activation
// ========================================

type DeadlockShacklesBehavior struct{ BaseBehavior }

func (DeadlockShacklesBehavior) BehaviorKey() string    { return BehaviorDeadlockShackles }
func (DeadlockShacklesBehavior) IsPlayerActivated() bool { return true }

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
// AnthemPerseveranceBehavior — grant friendly squad bonus attack
// ========================================

type AnthemPerseveranceBehavior struct{ BaseBehavior }

func (AnthemPerseveranceBehavior) BehaviorKey() string    { return BehaviorAnthemPerseverance }
func (AnthemPerseveranceBehavior) IsPlayerActivated() bool { return true }

func (AnthemPerseveranceBehavior) Activate(ctx *BehaviorContext, targetSquadID ecs.EntityID) error {
	if err := requireCharge(ctx, BehaviorAnthemPerseverance); err != nil {
		return err
	}
	actionState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if actionState == nil {
		return fmt.Errorf("squad %d has no action state", targetSquadID)
	}
	if !actionState.HasActed {
		return fmt.Errorf("squad %d has not acted yet (anthem requires acted squad)", targetSquadID)
	}
	actionState.HasActed = false
	ctx.ChargeTracker.UseCharge(BehaviorAnthemPerseverance, ChargeOncePerBattle)
	fmt.Printf("[GEAR] Anthem of Perseverance: squad %d can attack again\n", targetSquadID)
	return nil
}

// ========================================
// RallyingHornBehavior — bonus activation when faction is attacked
// ========================================

type RallyingHornBehavior struct{ BaseBehavior }

func (RallyingHornBehavior) BehaviorKey() string { return BehaviorRallyingHorn }

func (RallyingHornBehavior) OnAttackComplete(ctx *BehaviorContext, attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
	if result.AttackerDestroyed {
		return
	}
	defenderFaction := ctx.GetSquadFaction(defenderID)
	if defenderFaction == 0 {
		return
	}
	factionSquads := ctx.GetFactionSquads(defenderFaction)
	if !HasBehaviorInFaction(factionSquads, BehaviorRallyingHorn, ctx.Manager) {
		return
	}
	if ctx.ChargeTracker == nil || !ctx.ChargeTracker.IsAvailable(BehaviorRallyingHorn) {
		return
	}
	// Find first acted friendly squad in defender's faction and reset it
	for _, sid := range factionSquads {
		as := ctx.Cache.FindActionStateBySquadID(sid)
		if as == nil || !as.HasActed {
			continue
		}
		squadSpeed := ctx.GetSquadSpeed(sid)
		as.HasActed = false
		as.HasMoved = false
		as.MovementRemaining = squadSpeed
		ctx.ChargeTracker.UseCharge(BehaviorRallyingHorn, ChargeOncePerBattle)
		fmt.Printf("[GEAR] Rallying War Horn: squad %d gets bonus activation (speed %d)\n", sid, squadSpeed)
		return
	}
}

// ========================================
// ChainOfCommandBehavior — pass attack to adjacent squad
// ========================================

type ChainOfCommandBehavior struct{ BaseBehavior }

func (ChainOfCommandBehavior) BehaviorKey() string    { return BehaviorChainOfCommand }
func (ChainOfCommandBehavior) IsPlayerActivated() bool { return true }

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
		return fmt.Errorf("cannot pass attack to self")
	}

	// Validate source hasn't attacked
	sourceState := ctx.Cache.FindActionStateBySquadID(sourceSquadID)
	if sourceState == nil {
		return fmt.Errorf("source squad %d has no action state", sourceSquadID)
	}
	if sourceState.HasActed {
		return fmt.Errorf("source squad %d has already acted", sourceSquadID)
	}

	// Validate target has attacked
	targetState := ctx.Cache.FindActionStateBySquadID(targetSquadID)
	if targetState == nil {
		return fmt.Errorf("target squad %d has no action state", targetSquadID)
	}
	if !targetState.HasActed {
		return fmt.Errorf("target squad %d has not acted yet", targetSquadID)
	}

	// Validate adjacency (Chebyshev distance <= 2)
	sourcePos, err := getSquadPosition(sourceSquadID, ctx.Manager)
	if err != nil {
		return fmt.Errorf("source squad position: %w", err)
	}
	targetPos, err := getSquadPosition(targetSquadID, ctx.Manager)
	if err != nil {
		return fmt.Errorf("target squad position: %w", err)
	}
	if sourcePos.ChebyshevDistance(&targetPos) > 2 {
		return fmt.Errorf("target squad %d is too far away (max distance 2)", targetSquadID)
	}

	sourceState.HasActed = true
	targetState.HasActed = false
	ctx.ChargeTracker.UseCharge(BehaviorChainOfCommand, ChargeOncePerRound)
	fmt.Printf("[GEAR] Chain of Command: squad %d passes attack to squad %d\n", sourceSquadID, targetSquadID)
	return nil
}

// getSquadPosition returns the logical position of a squad.
func getSquadPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d not found", squadID)
	}
	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if pos == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d has no position", squadID)
	}
	return *pos, nil
}
