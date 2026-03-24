package combatcore

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// Counterattack penalties - applied to units attacking back after being hit
const (
	counterattackDamageMultiplier = 0.5 // 50% damage on counterattack
	counterattackHitPenalty       = 20  // -20% hit chance on counterattack
)

// processAttackWithModifiers is the unified attack processing function
func processAttackWithModifiers(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, modifiers DamageModifiers, manager *common.EntityManager) int {

	for _, defenderID := range targetIDs {
		attackIndex++

		// Calculate damage with modifiers
		damage, event := calculateDamage(attackerID, defenderID, modifiers, manager)

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

		// Apply damage
		recordDamageToUnit(defenderID, damage, result, manager)

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// ProcessAttackOnTargets applies damage to all targets and creates combat events
// Returns the updated attack index
func ProcessAttackOnTargets(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	modifiers := DamageModifiers{
		HitPenalty:       0,
		DamageMultiplier: 1.0,
		IsCounterattack:  false,
	}
	return processAttackWithModifiers(attackerID, targetIDs, result, log, attackIndex, modifiers, manager)
}

// ProcessCounterattackOnTargets applies counterattack damage with penalties
func ProcessCounterattackOnTargets(attackerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	modifiers := DamageModifiers{
		HitPenalty:       counterattackHitPenalty,
		DamageMultiplier: counterattackDamageMultiplier,
		IsCounterattack:  true,
	}
	return processAttackWithModifiers(attackerID, targetIDs, result, log, attackIndex, modifiers, manager)
}

// ProcessHealOnTargets iterates heal targets, calculates healing, and records events.
// Returns updated attackIndex.
func ProcessHealOnTargets(healerID ecs.EntityID, targetIDs []ecs.EntityID, result *CombatResult,
	log *CombatLog, attackIndex int, manager *common.EntityManager) int {

	for _, targetID := range targetIDs {
		attackIndex++

		healAmount, event := calculateHealing(healerID, targetID, manager)
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
