package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/powercore"

	"github.com/bytearena/ecs"
)

// SquadPerkDispatcher implements combattypes.PerkDispatcher by iterating
// equipped PerkBehavior implementations for each squad.
//
// The dispatcher holds a single PowerLogger that is threaded into every
// HookContext it builds. Previously a package-global (perkLogger) fulfilled
// this role; the logger now lives on the dispatcher so each combat instance
// can wire its own without mutating global state.
type SquadPerkDispatcher struct {
	logger powercore.PowerLogger
}

// SetLogger injects the PowerLogger used by perk activations.
func (d *SquadPerkDispatcher) SetLogger(logger powercore.PowerLogger) {
	d.logger = logger
}

// combatCtx returns a HookContext with the attacker/defender identifiers
// common to every damage-pipeline hook and CoverMod. Five dispatch methods
// use this exact shape; centralising the literal keeps them in sync.
func combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID) HookContext {
	return HookContext{
		AttackerID: attackerID, DefenderID: defenderID,
		AttackerSquadID: attackerSquadID, DefenderSquadID: defenderSquadID,
	}
}

// run is the shared dispatch primitive. Callers provide a HookContext
// populated with the per-hook identifiers they need (AttackerID, UnitID,
// etc.); run fills in the dispatcher-owned fields (PowerContext, RoundState)
// and iterates equipped perks on ownerSquadID. Returning false from hook
// terminates iteration early — used by CounterMod, DeathOverride, and
// DamageRedirect.
//
// The PowerContext and RoundState fields on the passed-in ctx are always
// overwritten; callers should leave them zero-valued.
func (d *SquadPerkDispatcher) run(ownerSquadID ecs.EntityID, manager *common.EntityManager, ctx HookContext, hook func(*HookContext, PerkBehavior) bool) {
	roundState := GetRoundState(ownerSquadID, manager)
	if roundState == nil {
		return
	}
	ctx.PowerContext = powercore.PowerContext{Manager: manager, Logger: d.logger}
	ctx.RoundState = roundState
	forEachPerkBehavior(ownerSquadID, manager, func(b PerkBehavior) bool {
		return hook(&ctx, b)
	})
}

// --- Damage pipeline hooks ---

func (d *SquadPerkDispatcher) AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combattypes.DamageModifiers, manager *common.EntityManager) {
	d.run(attackerSquadID, manager, combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID),
		func(ctx *HookContext, b PerkBehavior) bool {
			b.AttackerDamageMod(ctx, modifiers)
			return true
		})
}

func (d *SquadPerkDispatcher) DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combattypes.DamageModifiers, manager *common.EntityManager) {
	d.run(defenderSquadID, manager, combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID),
		func(ctx *HookContext, b PerkBehavior) bool {
			b.DefenderDamageMod(ctx, modifiers)
			return true
		})
}

func (d *SquadPerkDispatcher) AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	d.run(attackerSquadID, manager, combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID),
		func(ctx *HookContext, b PerkBehavior) bool {
			b.AttackerPostDamage(ctx, damageDealt, wasKill)
			return true
		})
}

func (d *SquadPerkDispatcher) DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	d.run(defenderSquadID, manager, combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID),
		func(ctx *HookContext, b PerkBehavior) bool {
			b.DefenderPostDamage(ctx, damageDealt, wasKill)
			return true
		})
}

// --- Targeting, counter, cover, and lifecycle hooks ---

func (d *SquadPerkDispatcher) TargetOverride(attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	d.run(attackerSquadID, manager, HookContext{
		AttackerID:      attackerID,
		AttackerSquadID: attackerSquadID,
		DefenderSquadID: defenderSquadID,
	}, func(ctx *HookContext, b PerkBehavior) bool {
		targets = b.TargetOverride(ctx, targets)
		return true
	})
	return targets
}

func (d *SquadPerkDispatcher) CounterMod(defenderSquadID, attackerID ecs.EntityID,
	modifiers *combattypes.DamageModifiers, manager *common.EntityManager) bool {
	skip := false
	d.run(defenderSquadID, manager, HookContext{
		AttackerID:      attackerID,
		DefenderSquadID: defenderSquadID,
		SquadID:         defenderSquadID,
	}, func(ctx *HookContext, b PerkBehavior) bool {
		if b.CounterMod(ctx, modifiers) {
			skip = true
			return false
		}
		return true
	})
	return skip
}

func (d *SquadPerkDispatcher) CoverMod(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combattypes.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	d.run(defenderSquadID, manager, combatCtx(attackerID, defenderID, attackerSquadID, defenderSquadID),
		func(ctx *HookContext, b PerkBehavior) bool {
			b.DefenderCoverMod(ctx, coverBreakdown)
			return true
		})
}

func (d *SquadPerkDispatcher) DeathOverride(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool {
	prevented := false
	d.run(squadID, manager, HookContext{
		UnitID:  unitID,
		SquadID: squadID,
	}, func(ctx *HookContext, b PerkBehavior) bool {
		if b.DeathOverride(ctx) {
			prevented = true
			return false
		}
		return true
	})
	return prevented
}

func (d *SquadPerkDispatcher) DamageRedirect(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	reducedDmg, redirectTarget, redirectAmt := damageAmount, ecs.EntityID(0), 0
	d.run(defenderSquadID, manager, HookContext{
		DefenderID:      defenderID,
		DefenderSquadID: defenderSquadID,
		UnitID:          defenderID,
		SquadID:         defenderSquadID,
		DamageAmount:    damageAmount,
	}, func(ctx *HookContext, b PerkBehavior) bool {
		rd, rt, ra := b.DamageRedirect(ctx)
		if rt != 0 {
			reducedDmg, redirectTarget, redirectAmt = rd, rt, ra
			return false
		}
		return true
	})
	return reducedDmg, redirectTarget, redirectAmt
}

// ========================================
// Lifecycle dispatch methods
// ========================================

// DispatchTurnStart resets per-turn state and runs TurnStart hooks for all squads in a faction.
func (d *SquadPerkDispatcher) DispatchTurnStart(squadIDs []ecs.EntityID, roundNumber int, manager *common.EntityManager) {
	for _, squadID := range squadIDs {
		roundState := GetRoundState(squadID, manager)
		if roundState == nil {
			continue
		}
		ResetPerkRoundStateTurn(roundState)
		RunTurnStartHooks(squadID, roundNumber, roundState, manager, d.logger)
	}
}

// DispatchRoundEnd resets per-round state for all squads with perks.
func (d *SquadPerkDispatcher) DispatchRoundEnd(manager *common.EntityManager) {
	for _, result := range manager.World.Query(PerkSlotTag) {
		squadID := result.Entity.GetID()
		roundState := GetRoundState(squadID, manager)
		if roundState != nil {
			ResetPerkRoundStateRound(roundState)
		}
	}
}

// DispatchAttackTracking updates perk round state after an attack resolves.
func (d *SquadPerkDispatcher) DispatchAttackTracking(attackerID, defenderID ecs.EntityID, manager *common.EntityManager) {
	attackerState := GetRoundState(attackerID, manager)
	if attackerState != nil {
		attackerState.AttackedThisTurn = true
	}

	defenderState := GetRoundState(defenderID, manager)
	if defenderState != nil {
		defenderState.WasAttackedThisTurn = true
	}
}

// DispatchMoveTracking updates perk round state after a squad moves.
func (d *SquadPerkDispatcher) DispatchMoveTracking(squadID ecs.EntityID, manager *common.EntityManager) {
	roundState := GetRoundState(squadID, manager)
	if roundState != nil {
		roundState.MovedThisTurn = true
		roundState.TurnsStationary = 0
	}
}
