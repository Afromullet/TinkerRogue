package combatcore

import (
	"game_main/core/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// counterattackDamageMultiplier returns the damage multiplier for counterattacks from the balance config.
func counterattackDamageMultiplier() float64 {
	return combatmath.CombatBalance.Counterattack.DamageMultiplier
}

// counterattackHitPenalty returns the hit penalty for counterattacks from the balance config.
func counterattackHitPenalty() int {
	return combatmath.CombatBalance.Counterattack.HitPenalty
}

// processAttack is the unified attack processing function. Each target follows
// the same ordered pipeline; the helpers below name each step so the loop body
// reads as orchestration rather than mechanics.
//
//	1. resolveAttackerSquadID    – look up the attacker's squad once per call
//	2. runTargetOverride         – perk-driven target swap (Cleave, etc.)
//	3. applyDamageMods           – per-target attacker/defender modifier hooks
//	4. combatmath.CalculateDamage – core damage roll
//	5. enrichTargetingInfo       – attach grid position + attack mode to the event
//	6. applyDamageWithRedirect   – Guardian Protocol redirect + record damage
//	7. applyDeathOverride        – Resolute prevents lethal damage
//	8. runPostDamageHooks        – Attacker/Defender post-damage hooks
//
// A nil dispatcher is normalized to NoopPerkDispatcher so every helper can call
// the dispatcher unconditionally.
func processAttack(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, modifiers combattypes.DamageModifiers,
	dispatcher combattypes.PerkDispatcher, manager *common.EntityManager) int {

	if dispatcher == nil {
		dispatcher = combattypes.NoopPerkDispatcher{}
	}

	attackerSquadID := resolveAttackerSquadID(attackerID, manager)
	targetIDs = runTargetOverride(dispatcher, attackerID, defenderSquadID, targetIDs, manager)

	for _, defenderID := range targetIDs {
		attackIndex++

		targetModifiers := applyDamageMods(dispatcher, attackerID, defenderID, attackerSquadID, defenderSquadID, modifiers, manager)
		damage, event := combatmath.CalculateDamage(attackerID, defenderID, targetModifiers, dispatcher, manager)
		enrichTargetingInfo(event, attackerID, defenderID, attackIndex, manager)
		damage = applyDamageWithRedirect(dispatcher, defenderID, defenderSquadID, damage, result, manager)
		applyDeathOverride(dispatcher, event, defenderID, defenderSquadID, result, manager)
		runPostDamageHooks(dispatcher, attackerID, defenderID, attackerSquadID, defenderSquadID, damage, event.WasKilled, manager)

		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// resolveAttackerSquadID looks up the squad an attacker belongs to. Returns 0
// if the attacker is not a squad member (e.g., a free-standing entity in a test).
func resolveAttackerSquadID(attackerID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	memberData := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, attackerID, squadcore.SquadMemberComponent)
	if memberData == nil {
		return 0
	}
	return memberData.SquadID
}

// runTargetOverride lets attacker perks (Cleave, Precision Strike) mutate the
// target list before damage rolls. Skips when defenderSquadID==0 because the
// hook can't resolve which squad's perks apply.
func runTargetOverride(dispatcher combattypes.PerkDispatcher, attackerID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	if defenderSquadID == 0 {
		return targetIDs
	}
	return dispatcher.TargetOverride(attackerID, defenderSquadID, targetIDs, manager)
}

// applyDamageMods runs both AttackerDamageMod and DefenderDamageMod hooks on a
// copy of modifiers and returns the modified copy. The caller-provided
// modifiers are not mutated so the next target sees a fresh starting state.
func applyDamageMods(dispatcher combattypes.PerkDispatcher,
	attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	modifiers combattypes.DamageModifiers, manager *common.EntityManager) combattypes.DamageModifiers {
	out := modifiers
	dispatcher.AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &out, manager)
	dispatcher.DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &out, manager)
	return out
}

// enrichTargetingInfo attaches the defender's grid position and the attacker's
// attack-mode tag to a calculated event. Mutates event in place.
func enrichTargetingInfo(event *combattypes.AttackEvent,
	attackerID, defenderID ecs.EntityID, attackIndex int, manager *common.EntityManager) {
	event.AttackIndex = attackIndex
	if defenderPos := common.GetComponentTypeByID[*squadcore.GridPositionData](manager, defenderID, squadcore.GridPositionComponent); defenderPos != nil {
		event.TargetInfo.TargetRow = defenderPos.AnchorRow
		event.TargetInfo.TargetCol = defenderPos.AnchorCol
	}
	if targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](manager, attackerID, squadcore.TargetRowComponent); targetData != nil {
		event.TargetInfo.TargetMode = targetData.AttackType.String()
	}
}

// applyDamageWithRedirect runs the Guardian Protocol redirect hook (when damage
// is positive), records any redirected portion against the redirect target,
// then records the (possibly reduced) primary damage against defenderID.
// Returns the final damage applied to the primary target.
func applyDamageWithRedirect(dispatcher combattypes.PerkDispatcher,
	defenderID, defenderSquadID ecs.EntityID, damage int,
	result *combattypes.CombatResult, manager *common.EntityManager) int {
	if damage > 0 {
		reducedDmg, redirectTarget, redirectAmt := dispatcher.DamageRedirect(defenderID, defenderSquadID, damage, manager)
		if redirectTarget != 0 && redirectAmt > 0 {
			damage = reducedDmg
			combatmath.RecordDamageToUnit(redirectTarget, redirectAmt, result, manager)
		}
	}
	combatmath.RecordDamageToUnit(defenderID, damage, result, manager)
	return damage
}

// applyDeathOverride lets the Resolute perk cancel a killing blow. Must run
// BEFORE post-damage hooks so prevented deaths aren't counted as kills by
// perks like Bloodlust. Mutates event.WasKilled and result.DamageByUnit /
// result.UnitsKilled on success.
func applyDeathOverride(dispatcher combattypes.PerkDispatcher, event *combattypes.AttackEvent,
	defenderID, defenderSquadID ecs.EntityID,
	result *combattypes.CombatResult, manager *common.EntityManager) {
	if !event.WasKilled {
		return
	}
	defenderMember := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, defenderID, squadcore.SquadMemberComponent)
	defSquadID := defenderSquadID
	if defenderMember != nil {
		defSquadID = defenderMember.SquadID
	}
	if !dispatcher.DeathOverride(defenderID, defSquadID, manager) {
		return
	}

	attr := common.GetComponentTypeByID[*common.Attributes](manager, defenderID, common.AttributeComponent)
	if attr == nil {
		return
	}
	maxAllowedDamage := attr.CurrentHealth - 1
	if maxAllowedDamage < 0 {
		maxAllowedDamage = 0
	}
	if result.Damage.DamageByUnit[defenderID] > maxAllowedDamage {
		result.Damage.DamageByUnit[defenderID] = maxAllowedDamage
	}
	for i, killedID := range result.Damage.UnitsKilled {
		if killedID == defenderID {
			result.Damage.UnitsKilled = append(result.Damage.UnitsKilled[:i], result.Damage.UnitsKilled[i+1:]...)
			break
		}
	}
	event.WasKilled = false
	event.DefenderHPAfter = 1
}

// runPostDamageHooks fires the AttackerPostDamage + DefenderPostDamage hooks.
// Runs after applyDeathOverride so wasKill reflects prevented deaths.
func runPostDamageHooks(dispatcher combattypes.PerkDispatcher,
	attackerID, defenderID, attackerSquadID, defenderSquadID ecs.EntityID,
	damage int, wasKill bool, manager *common.EntityManager) {
	dispatcher.AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, wasKill, manager)
	dispatcher.DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, wasKill, manager)
}

// ProcessAttackOnTargets applies damage to all targets and creates combat events.
func ProcessAttackOnTargets(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, dispatcher combattypes.PerkDispatcher, manager *common.EntityManager) int {

	modifiers := combattypes.DamageModifiers{
		HitModifier:      0,
		DamageMultiplier: 1.0,
		IsCounterattack:  false,
	}
	return processAttack(attackerID, defenderSquadID, targetIDs, result, log, attackIndex, modifiers, dispatcher, manager)
}

// ProcessCounterattackOnTargets applies counterattack damage with penalties.
func ProcessCounterattackOnTargets(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, modifiers combattypes.DamageModifiers, dispatcher combattypes.PerkDispatcher, manager *common.EntityManager) int {

	return processAttack(attackerID, defenderSquadID, targetIDs, result, log, attackIndex, modifiers, dispatcher, manager)
}

// ProcessHealOnTargets iterates heal targets, calculates healing, and records events.
// Returns updated attackIndex.
func ProcessHealOnTargets(healerID ecs.EntityID, targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, manager *common.EntityManager) int {

	for _, targetID := range targetIDs {
		attackIndex++

		healAmount, event := combatmath.CalculateHealing(healerID, targetID, manager)
		event.AttackIndex = attackIndex

		if healAmount > 0 {
			// Record healing (accumulated per unit)
			result.Damage.HealingByUnit[targetID] += healAmount
		}

		log.HealEvents = append(log.HealEvents, *event)
		log.TotalHealing += healAmount
	}

	return attackIndex
}
