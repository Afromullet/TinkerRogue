package combatcore

import (
	"game_main/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// Counterattack penalties - applied to units attacking back after being hit
const (
	counterattackDamageMultiplier = 0.5 // 50% damage on counterattack
	counterattackHitPenalty       = 20  // -20% hit chance on counterattack
)

// processAttack is the unified attack processing function.
// defenderSquadID is needed for perk target override hooks (0 if unknown).
// callbacks may be nil if no perks are active.
func processAttack(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, modifiers DamageModifiers,
	callbacks *PerkCallbacks, manager *common.EntityManager) int {

	// Determine attacker's squad ID for perk hooks
	attackerSquadID := ecs.EntityID(0)
	memberData := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, attackerID, squadcore.SquadMemberComponent)
	if memberData != nil {
		attackerSquadID = memberData.SquadID
	}

	// Run target override hooks (attacker perks like Cleave, Precision Strike)
	if callbacks != nil && callbacks.TargetOverride != nil && defenderSquadID != 0 {
		targetIDs = callbacks.TargetOverride(attackerID, defenderSquadID, targetIDs, manager)
	}

	for _, defenderID := range targetIDs {
		attackIndex++

		// Make a copy of modifiers for this specific target (perks may modify per-target)
		targetModifiers := modifiers

		// Run attacker damage mod hooks BEFORE damage calculation
		if callbacks != nil && callbacks.AttackerDamageMod != nil {
			callbacks.AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &targetModifiers, manager)
		}

		// Run defender damage mod hooks BEFORE damage calculation
		if callbacks != nil && callbacks.DefenderDamageMod != nil {
			callbacks.DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &targetModifiers, manager)
		}

		// Calculate damage
		damage, event := combatmath.CalculateDamage(attackerID, defenderID, targetModifiers, callbacks, manager)

		// Add targeting info
		defenderPos := common.GetComponentTypeByID[*squadcore.GridPositionData](manager, defenderID, squadcore.GridPositionComponent)
		event.AttackIndex = attackIndex
		if defenderPos != nil {
			event.TargetInfo.TargetRow = defenderPos.AnchorRow
			event.TargetInfo.TargetCol = defenderPos.AnchorCol
		}

		// Set target mode
		targetData := common.GetComponentTypeByID[*squadcore.TargetRowData](manager, attackerID, squadcore.TargetRowComponent)
		if targetData != nil {
			event.TargetInfo.TargetMode = targetData.AttackType.String()
		}

		// Run damage redirect hooks (Guardian Protocol)
		if callbacks != nil && callbacks.DamageRedirect != nil && damage > 0 {
			reducedDmg, redirectTarget, redirectAmt := callbacks.DamageRedirect(defenderID, defenderSquadID, damage, manager)
			if redirectTarget != 0 && redirectAmt > 0 {
				damage = reducedDmg
				combatmath.RecordDamageToUnit(redirectTarget, redirectAmt, result, manager)
			}
		}

		// Apply damage
		combatmath.RecordDamageToUnit(defenderID, damage, result, manager)

		// Run post-damage hooks (attacker side: Bloodlust, Disruption)
		if callbacks != nil && callbacks.PostDamage != nil {
			callbacks.PostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, event.WasKilled, manager)
		}

		// Run defender post-damage hooks (Grudge Bearer tracking)
		if callbacks != nil && callbacks.DefenderPostDamage != nil {
			callbacks.DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, event.WasKilled, manager)
		}

		// Death override check (Resolute)
		if event.WasKilled && callbacks != nil && callbacks.DeathOverride != nil {
			defenderMember := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, defenderID, squadcore.SquadMemberComponent)
			defSquadID := defenderSquadID
			if defenderMember != nil {
				defSquadID = defenderMember.SquadID
			}
			if callbacks.DeathOverride(defenderID, defSquadID, manager) {
				// Prevent death: adjust recorded damage so unit survives at 1 HP
				attr := common.GetComponentTypeByID[*common.Attributes](manager, defenderID, common.AttributeComponent)
				if attr != nil {
					totalRecorded := result.DamageByUnit[defenderID]
					maxAllowedDamage := attr.CurrentHealth - 1
					if maxAllowedDamage < 0 {
						maxAllowedDamage = 0
					}
					if totalRecorded > maxAllowedDamage {
						result.DamageByUnit[defenderID] = maxAllowedDamage
					}
					for i, killedID := range result.UnitsKilled {
						if killedID == defenderID {
							result.UnitsKilled = append(result.UnitsKilled[:i], result.UnitsKilled[i+1:]...)
							break
						}
					}
					event.WasKilled = false
					event.DefenderHPAfter = 1
				}
			}
		}

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// ProcessAttackOnTargets applies damage to all targets and creates combat events.
// callbacks may be nil if no perks are active.
func ProcessAttackOnTargets(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, callbacks *PerkCallbacks, manager *common.EntityManager) int {

	modifiers := DamageModifiers{
		HitPenalty:       0,
		DamageMultiplier: 1.0,
		IsCounterattack:  false,
	}
	return processAttack(attackerID, defenderSquadID, targetIDs, result, log, attackIndex, modifiers, callbacks, manager)
}

// ProcessCounterattackOnTargets applies counterattack damage with penalties.
// callbacks may be nil if no perks are active.
func ProcessCounterattackOnTargets(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, modifiers DamageModifiers, callbacks *PerkCallbacks, manager *common.EntityManager) int {

	return processAttack(attackerID, defenderSquadID, targetIDs, result, log, attackIndex, modifiers, callbacks, manager)
}

// ProcessHealOnTargets iterates heal targets, calculates healing, and records events.
// Returns updated attackIndex.
func ProcessHealOnTargets(healerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	for _, targetID := range targetIDs {
		attackIndex++

		healAmount, event := combatmath.CalculateHealing(healerID, targetID, manager)
		event.AttackIndex = attackIndex

		if healAmount > 0 {
			// Record healing (accumulated per unit)
			result.HealingByUnit[targetID] += healAmount
		}

		log.HealEvents = append(log.HealEvents, *event)
		log.TotalHealing += healAmount
	}

	return attackIndex
}
