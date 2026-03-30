package combatcore

import (
	"game_main/common"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"

	"github.com/bytearena/ecs"
)

// rollD100Check is a unified dice roll for hit/crit/dodge checks.
// Returns the roll (1-100) and whether it passed (roll <= threshold).
func rollD100Check(threshold int) (roll int, passed bool) {
	roll = common.GetDiceRoll(100)
	passed = roll <= threshold
	return
}

// calculateDamage handles the full damage pipeline: hit roll, dodge roll, base damage, crit, resistance, cover.
// callbacks may be nil if no perks are active.
func calculateDamage(attackerID, defenderID ecs.EntityID, modifiers DamageModifiers,
	callbacks *PerkCallbacks, squadmanager *common.EntityManager) (int, *AttackEvent) {
	attackerAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, attackerID, common.AttributeComponent)
	defenderAttr := common.GetComponentTypeByID[*common.Attributes](squadmanager, defenderID, common.AttributeComponent)

	event := &AttackEvent{
		AttackerID:      attackerID,
		DefenderID:      defenderID,
		IsCounterattack: modifiers.IsCounterattack,
	}

	if defenderAttr != nil {
		event.DefenderHPBefore = defenderAttr.CurrentHealth
	}

	if attackerAttr == nil || defenderAttr == nil {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Hit roll with optional penalty
	baseHitThreshold := attackerAttr.GetHitRate()
	hitThreshold := baseHitThreshold - modifiers.HitPenalty
	if hitThreshold < 0 {
		hitThreshold = 0
	}

	hitRoll, didHit := rollD100Check(hitThreshold)
	event.HitResult.HitRoll = hitRoll
	event.HitResult.HitThreshold = hitThreshold

	if !didHit {
		event.HitResult.Type = HitTypeMiss
		return 0, event
	}

	// Dodge roll
	dodgeThreshold := defenderAttr.GetDodgeChance()
	dodgeRoll, wasDodged := rollD100Check(dodgeThreshold)
	event.HitResult.DodgeRoll = dodgeRoll
	event.HitResult.DodgeThreshold = dodgeThreshold

	if wasDodged {
		event.HitResult.Type = HitTypeDodge
		return 0, event
	}

	// Get attacker's attack type to determine damage formula
	attackerTargetData := common.GetComponentTypeByID[*squadcore.TargetRowData](squadmanager, attackerID, squadcore.TargetRowComponent)

	// Calculate base damage based on attack type
	var baseDamage int
	var resistance int

	if attackerTargetData != nil && attackerTargetData.AttackType == unitdefs.AttackTypeMagic {
		baseDamage = attackerAttr.GetMagicDamage()
		resistance = defenderAttr.GetMagicDefense()
	} else {
		baseDamage = attackerAttr.GetPhysicalDamage()
		resistance = defenderAttr.GetPhysicalResistance()
	}

	event.BaseDamage = baseDamage
	event.CritMultiplier = 1.0

	// Crit roll (supports SkipCrit and CritBonus from perks)
	if modifiers.SkipCrit {
		if modifiers.IsCounterattack {
			event.HitResult.Type = HitTypeCounterattack
		} else {
			event.HitResult.Type = HitTypeNormal
		}
	} else {
		critThreshold := attackerAttr.GetCritChance() + modifiers.CritBonus
		critRoll, wasCrit := rollD100Check(critThreshold)
		event.HitResult.CritRoll = critRoll
		event.HitResult.CritThreshold = critThreshold

		if wasCrit {
			baseDamage = int(float64(baseDamage) * 1.5)
			event.CritMultiplier = 1.5
			event.HitResult.Type = HitTypeCritical
		} else {
			if modifiers.IsCounterattack {
				event.HitResult.Type = HitTypeCounterattack
			} else {
				event.HitResult.Type = HitTypeNormal
			}
		}
	}

	// Apply damage multiplier
	baseDamage = int(float64(baseDamage) * modifiers.DamageMultiplier)
	if baseDamage < 1 {
		baseDamage = 1
	}

	// Apply resistance
	event.ResistanceAmount = resistance
	totalDamage := baseDamage - resistance
	if totalDamage < 1 {
		totalDamage = 1
	}

	// Apply cover
	coverBreakdown := CalculateCoverBreakdown(defenderID, squadmanager)

	// Run perk cover mod hooks
	if callbacks != nil && callbacks.CoverMod != nil {
		callbacks.CoverMod(attackerID, defenderID, &coverBreakdown, squadmanager)
	}

	// Apply perk cover bonus
	if modifiers.CoverBonus > 0 {
		coverBreakdown.TotalReduction += modifiers.CoverBonus
		if coverBreakdown.TotalReduction > 1.0 {
			coverBreakdown.TotalReduction = 1.0
		}
	}

	event.CoverReduction = coverBreakdown

	if coverBreakdown.TotalReduction > 0.0 {
		totalDamage = int(float64(totalDamage) * (1.0 - coverBreakdown.TotalReduction))
		if totalDamage < 1 {
			totalDamage = 1
		}
	}

	event.FinalDamage = totalDamage
	event.DefenderHPAfter = defenderAttr.CurrentHealth - totalDamage
	if event.DefenderHPAfter <= 0 {
		event.WasKilled = true
	}

	return totalDamage, event
}

// recordDamageToUnit records damage in the combat result without modifying HP (pure calculation)
func recordDamageToUnit(unitID ecs.EntityID, damage int, result *CombatResult, squadmanager *common.EntityManager) {
	// Accumulate damage (in case unit is hit multiple times)
	result.DamageByUnit[unitID] += damage

	// Check if unit would be killed (prediction based on current HP)
	attr := common.GetComponentTypeByID[*common.Attributes](squadmanager, unitID, common.AttributeComponent)
	if attr != nil {
		totalDamageTaken := result.DamageByUnit[unitID]
		if attr.CurrentHealth-totalDamageTaken <= 0 {
			// Only add to UnitsKilled once
			alreadyMarked := false
			for _, killedID := range result.UnitsKilled {
				if killedID == unitID {
					alreadyMarked = true
					break
				}
			}
			if !alreadyMarked {
				result.UnitsKilled = append(result.UnitsKilled, unitID)
			}
		}
	}
}

// ApplyRecordedDamage applies all recorded damage from result.DamageByUnit to actual unit HP.
// This is called during orchestration phase after all combat calculations are complete.
func ApplyRecordedDamage(result *CombatResult, squadmanager *common.EntityManager) {
	for unitID, damage := range result.DamageByUnit {
		attr := common.GetComponentTypeByID[*common.Attributes](squadmanager, unitID, common.AttributeComponent)
		if attr == nil {
			continue
		}
		attr.CurrentHealth -= damage
	}
}

// ApplyRecordedHealing applies all recorded healing from result.HealingByUnit to actual unit HP.
// Called AFTER ApplyRecordedDamage so healers can offset damage taken in the same round.
func ApplyRecordedHealing(result *CombatResult, manager *common.EntityManager) {
	for unitID, healing := range result.HealingByUnit {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr == nil {
			continue
		}
		attr.CurrentHealth += healing
		maxHP := attr.GetMaxHealth()
		if attr.CurrentHealth > maxHP {
			attr.CurrentHealth = maxHP
		}
	}
}

// sumDamageMap totals all damage values in a damage map
func sumDamageMap(damageMap map[ecs.EntityID]int) int {
	total := 0
	for _, dmg := range damageMap {
		total += dmg
	}
	return total
}

// calculateSquadStatus summarizes squad health for combat log
func calculateSquadStatus(squadID ecs.EntityID, manager *common.EntityManager) SquadStatus {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
	aliveCount := 0

	for _, unitID := range unitIDs {
		if squadcore.GetAliveUnitAttributes(unitID, manager) != nil {
			aliveCount++
		}
	}

	hpPercent := squadcore.GetSquadHealthPercent(squadID, manager)
	avgHP := int(hpPercent * 100)

	return SquadStatus{
		AliveUnits: aliveCount,
		TotalUnits: len(unitIDs),
		AverageHP:  avgHP,
	}
}

// calculateHealing computes guaranteed healing (no hit/dodge/crit rolls).
// Uses GetHealingAmount() (Magic * 2), capped at MaxHealth - CurrentHealth.
func calculateHealing(healerID, targetID ecs.EntityID, manager *common.EntityManager) (int, *HealEvent) {
	healerAttr := common.GetComponentTypeByID[*common.Attributes](manager, healerID, common.AttributeComponent)
	targetAttr := common.GetComponentTypeByID[*common.Attributes](manager, targetID, common.AttributeComponent)

	event := &HealEvent{
		HealerID: healerID,
		TargetID: targetID,
	}

	if healerAttr == nil || targetAttr == nil {
		return 0, event
	}

	event.TargetHPBefore = targetAttr.CurrentHealth

	healAmount := healerAttr.GetHealingAmount()

	// Cap at missing HP
	missingHP := targetAttr.GetMaxHealth() - targetAttr.CurrentHealth
	if healAmount > missingHP {
		healAmount = missingHP
	}
	if healAmount < 0 {
		healAmount = 0
	}

	event.HealAmount = healAmount
	event.TargetHPAfter = targetAttr.CurrentHealth + healAmount

	return healAmount, event
}
