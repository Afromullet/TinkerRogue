package perks

import (
	"game_main/common"
	"game_main/tactical/combat/combatcore"

	"github.com/bytearena/ecs"
)

// SquadPerkDispatcher implements combattypes.PerkDispatcher by iterating
// equipped PerkBehavior implementations for each squad.
type SquadPerkDispatcher struct{}

func (d *SquadPerkDispatcher) AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	ctx := buildCombatContext(attackerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkBehavior(attackerSquadID, manager, func(behavior PerkBehavior) bool {
		behavior.AttackerDamageMod(ctx, modifiers)
		return true
	})
}

func (d *SquadPerkDispatcher) DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) {
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkBehavior(defenderSquadID, manager, func(behavior PerkBehavior) bool {
		behavior.DefenderDamageMod(ctx, modifiers)
		return true
	})
}

func (d *SquadPerkDispatcher) AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	ctx := buildCombatContext(attackerSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkBehavior(attackerSquadID, manager, func(behavior PerkBehavior) bool {
		behavior.AttackerPostDamage(ctx, damageDealt, wasKill)
		return true
	})
}

func (d *SquadPerkDispatcher) DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damageDealt int, wasKill bool, manager *common.EntityManager) {
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkBehavior(defenderSquadID, manager, func(behavior PerkBehavior) bool {
		behavior.DefenderPostDamage(ctx, damageDealt, wasKill)
		return true
	})
}

func (d *SquadPerkDispatcher) TargetOverride(attackerID, defenderSquadID ecs.EntityID,
	targets []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	ctx := buildHookContext(attackerSquadID, manager)
	if ctx == nil {
		return targets
	}
	ctx.AttackerID = attackerID
	ctx.AttackerSquadID = attackerSquadID
	ctx.DefenderSquadID = defenderSquadID
	forEachPerkBehavior(attackerSquadID, manager, func(behavior PerkBehavior) bool {
		targets = behavior.TargetOverride(ctx, targets)
		return true
	})
	return targets
}

func (d *SquadPerkDispatcher) CounterMod(defenderSquadID, attackerID ecs.EntityID,
	modifiers *combatcore.DamageModifiers, manager *common.EntityManager) bool {
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return false
	}
	ctx.DefenderSquadID = defenderSquadID
	ctx.AttackerID = attackerID
	ctx.SquadID = defenderSquadID
	skip := false
	forEachPerkBehavior(defenderSquadID, manager, func(behavior PerkBehavior) bool {
		if behavior.CounterMod(ctx, modifiers) {
			skip = true
			return false
		}
		return true
	})
	return skip
}

func (d *SquadPerkDispatcher) CoverMod(attackerID, defenderID ecs.EntityID,
	coverBreakdown *combatcore.CoverBreakdown, manager *common.EntityManager) {
	attackerSquadID := getSquadIDForUnit(attackerID, manager)
	defenderSquadID := getSquadIDForUnit(defenderID, manager)
	ctx := buildCombatContext(defenderSquadID, attackerID, defenderID, attackerSquadID, defenderSquadID, manager)
	if ctx == nil {
		return
	}
	forEachPerkBehavior(defenderSquadID, manager, func(behavior PerkBehavior) bool {
		behavior.DefenderCoverMod(ctx, coverBreakdown)
		return true
	})
}

func (d *SquadPerkDispatcher) DeathOverride(unitID, squadID ecs.EntityID, manager *common.EntityManager) bool {
	ctx := buildHookContext(squadID, manager)
	if ctx == nil {
		return false
	}
	ctx.UnitID = unitID
	ctx.SquadID = squadID
	prevented := false
	forEachPerkBehavior(squadID, manager, func(behavior PerkBehavior) bool {
		if behavior.DeathOverride(ctx) {
			prevented = true
			return false
		}
		return true
	})
	return prevented
}

func (d *SquadPerkDispatcher) DamageRedirect(defenderID, defenderSquadID ecs.EntityID,
	damageAmount int, manager *common.EntityManager) (int, ecs.EntityID, int) {
	ctx := buildHookContext(defenderSquadID, manager)
	if ctx == nil {
		return damageAmount, 0, 0
	}
	ctx.UnitID = defenderID
	ctx.SquadID = defenderSquadID
	ctx.DefenderID = defenderID
	ctx.DefenderSquadID = defenderSquadID
	ctx.DamageAmount = damageAmount
	reducedDmg, redirectTarget, redirectAmt := damageAmount, ecs.EntityID(0), 0
	forEachPerkBehavior(defenderSquadID, manager, func(behavior PerkBehavior) bool {
		rd, rt, ra := behavior.DamageRedirect(ctx)
		if rt != 0 {
			reducedDmg, redirectTarget, redirectAmt = rd, rt, ra
			return false
		}
		return true
	})
	return reducedDmg, redirectTarget, redirectAmt
}
