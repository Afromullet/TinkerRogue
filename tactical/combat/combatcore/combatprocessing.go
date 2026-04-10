package combatcore

import (
	"game_main/common"
	"game_main/tactical/combat/combatmath"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// counterattackDamageMultiplier returns the damage multiplier for counterattacks from the balance config.
func counterattackDamageMultiplier() float64 {
	return CombatBalance.Counterattack.DamageMultiplier
}

// counterattackHitPenalty returns the hit penalty for counterattacks from the balance config.
func counterattackHitPenalty() int {
	return CombatBalance.Counterattack.HitPenalty
}

// processAttack is the unified attack processing function.
// defenderSquadID is needed for perk target override hooks (0 if unknown).
func processAttack(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, modifiers combattypes.DamageModifiers,
	dispatcher combattypes.PerkDispatcher, manager *common.EntityManager) int {

	// Determine attacker's squad ID for perk hooks
	attackerSquadID := ecs.EntityID(0)
	memberData := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, attackerID, squadcore.SquadMemberComponent)
	if memberData != nil {
		attackerSquadID = memberData.SquadID
	}

	// Run target override hooks (attacker perks like Cleave, Precision Strike)
	if dispatcher != nil && defenderSquadID != 0 {
		targetIDs = dispatcher.TargetOverride(attackerID, defenderSquadID, targetIDs, manager)
	}

	for _, defenderID := range targetIDs {
		attackIndex++

		// Make a copy of modifiers for this specific target (perks may modify per-target)
		targetModifiers := modifiers

		// Run attacker damage mod hooks BEFORE damage calculation
		if dispatcher != nil {
			dispatcher.AttackerDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &targetModifiers, manager)
		}

		// Run defender damage mod hooks BEFORE damage calculation
		if dispatcher != nil {
			dispatcher.DefenderDamageMod(attackerID, defenderID, attackerSquadID, defenderSquadID, &targetModifiers, manager)
		}

		// Calculate damage
		damage, event := combatmath.CalculateDamage(attackerID, defenderID, targetModifiers, dispatcher, manager)

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
		if dispatcher != nil && damage > 0 {
			reducedDmg, redirectTarget, redirectAmt := dispatcher.DamageRedirect(defenderID, defenderSquadID, damage, manager)
			if redirectTarget != 0 && redirectAmt > 0 {
				damage = reducedDmg
				combatmath.RecordDamageToUnit(redirectTarget, redirectAmt, result, manager)
			}
		}

		// Apply damage
		combatmath.RecordDamageToUnit(defenderID, damage, result, manager)

		// Death override check (Resolute) — must run BEFORE PostDamage hooks
		// so that prevented deaths are not counted as kills by perks like Bloodlust.
		if event.WasKilled && dispatcher != nil {
			defenderMember := common.GetComponentTypeByID[*squadcore.SquadMemberData](manager, defenderID, squadcore.SquadMemberComponent)
			defSquadID := defenderSquadID
			if defenderMember != nil {
				defSquadID = defenderMember.SquadID
			}
			if dispatcher.DeathOverride(defenderID, defSquadID, manager) {
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

		// Run post-damage hooks AFTER death override so WasKilled reflects prevented deaths
		if dispatcher != nil {
			dispatcher.AttackerPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, event.WasKilled, manager)
		}

		// Run defender post-damage hooks (Grudge Bearer tracking)
		if dispatcher != nil {
			dispatcher.DefenderPostDamage(attackerID, defenderID, attackerSquadID, defenderSquadID, damage, event.WasKilled, manager)
		}

		// Store event
		log.AttackEvents = append(log.AttackEvents, *event)
	}

	return attackIndex
}

// ProcessAttackOnTargets applies damage to all targets and creates combat events.
func ProcessAttackOnTargets(attackerID ecs.EntityID, defenderSquadID ecs.EntityID,
	targetIDs []ecs.EntityID, result *combattypes.CombatResult,
	log *combattypes.CombatLog, attackIndex int, dispatcher combattypes.PerkDispatcher, manager *common.EntityManager) int {

	modifiers := combattypes.DamageModifiers{
		HitPenalty:       0,
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
			result.HealingByUnit[targetID] += healAmount
		}

		log.HealEvents = append(log.HealEvents, *event)
		log.TotalHealing += healAmount
	}

	return attackIndex
}
