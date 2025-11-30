package squads

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// CheckAndTriggerAbilities - ✅ Works with ecs.EntityID
func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	// Find leader via query (not stored reference)
	leaderID := GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return // No leader, no abilities
	}

	if !ecsmanager.HasComponentByIDWithTag(leaderID, SquadMemberTag, AbilitySlotComponent) {
		return
	}

	if !ecsmanager.HasComponentByIDWithTag(leaderID, SquadMemberTag, CooldownTrackerComponent) {
		return
	}

	abilityData := common.GetComponentTypeByID[*AbilitySlotData](ecsmanager, leaderID, AbilitySlotComponent)
	cooldownData := common.GetComponentTypeByID[*CooldownTrackerData](ecsmanager, leaderID, CooldownTrackerComponent)

	// Check each ability slot
	for i := 0; i < 4; i++ {
		slot := &abilityData.Slots[i]

		if !slot.IsEquipped || cooldownData.Cooldowns[i] > 0 || slot.HasTriggered {
			continue
		}

		// Evaluate trigger condition
		triggered := evaluateTrigger(slot, squadID, ecsmanager)
		if !triggered {
			continue
		}

		// Execute ability
		executeAbility(slot, squadID, ecsmanager)

		// Set cooldown
		cooldownData.Cooldowns[i] = cooldownData.MaxCooldowns[i]

		// Mark as triggered
		slot.HasTriggered = true
	}

	// Tick down cooldowns
	for i := 0; i < 4; i++ {
		if cooldownData.Cooldowns[i] > 0 {
			cooldownData.Cooldowns[i]--
		}
	}
}

// evaluateTrigger checks if a condition is met
func evaluateTrigger(slot *AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return false
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	switch slot.TriggerType {
	case TriggerSquadHPBelow:
		avgHP := calculateAverageHP(squadID, ecsmanager)
		return avgHP < slot.Threshold

	case TriggerTurnCount:
		return squadData.TurnCount == int(slot.Threshold)

	case TriggerCombatStart:
		return squadData.TurnCount == 1

	case TriggerEnemyCount:
		enemyCount := countEnemySquads(ecsmanager)
		return float64(enemyCount) >= slot.Threshold

	case TriggerMoraleBelow:
		return float64(squadData.Morale) < slot.Threshold

	default:
		return false
	}
}

// calculateAverageHP computes the squad's average HP as a percentage (0.0 - 1.0)
func calculateAverageHP(squadID ecs.EntityID, ecsmanager *common.EntityManager) float64 {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	totalHP := 0
	totalMaxHP := 0

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(ecsmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		totalHP += attr.CurrentHealth
		totalMaxHP += attr.MaxHealth
	}

	if totalMaxHP == 0 {
		return 0.0
	}

	return float64(totalHP) / float64(totalMaxHP)
}

// countEnemySquads counts the number of enemy squads on the map
func countEnemySquads(ecsmanager *common.EntityManager) int {
	count := 0
	for _, result := range ecsmanager.World.Query(SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

		// Assume enemy squads don't have "Player" prefix (adjust based on your naming)
		if len(squadData.Name) > 0 && squadData.Name[0] != 'P' {
			count++
		}
	}
	return count
}

// executeAbility triggers the ability effect
// Data-driven approach: reads ability params, applies effects
func executeAbility(slot *AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	params := GetAbilityParams(slot.AbilityType)

	switch slot.AbilityType {
	case AbilityRally:
		applyRallyEffect(squadID, params, ecsmanager)
	case AbilityHeal:
		applyHealEffect(squadID, params, ecsmanager)
	case AbilityBattleCry:
		applyBattleCryEffect(squadID, params, ecsmanager)
	case AbilityFireball:
		applyFireballEffect(squadID, params, ecsmanager)
	}
}

// --- Ability Implementations (Data-Driven) ---

// RallyEffect: Temporary damage buff to own squad
func applyRallyEffect(squadID ecs.EntityID, params AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(ecsmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth > 0 {
			attr.Strength += params.StrengthBonus
			// TODO: Track buff duration (requires turn/buff system)
		}
	}

	fmt.Printf("[ABILITY] Rally! +%d damage for %d turns\n", params.StrengthBonus, params.Duration)
}

// HealEffect: Restore HP to own squad
func applyHealEffect(squadID ecs.EntityID, params AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)

	healed := 0
	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(ecsmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth <= 0 {
			continue
		}

		// Cap at max HP
		attr.CurrentHealth += params.HealAmount
		if attr.CurrentHealth > attr.MaxHealth {
			attr.CurrentHealth = attr.MaxHealth
		}
		healed++
	}

	fmt.Printf("[ABILITY] Healing Aura! %d units restored %d HP\n", healed, params.HealAmount)
}

// BattleCryEffect: First turn buff (morale + damage)
func applyBattleCryEffect(squadID ecs.EntityID, params AbilityParams, ecsmanager *common.EntityManager) {
	squadEntity := GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

	// Boost morale
	squadData.Morale += params.MoraleBonus

	// Boost damage
	unitIDs := GetUnitIDsInSquad(squadID, ecsmanager)
	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(ecsmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth > 0 {
			attr.Strength += params.StrengthBonus
		}
	}

	fmt.Printf("[ABILITY] Battle Cry! Morale and damage increased!\n")
}

// FireballEffect: AOE damage to enemy squad
func applyFireballEffect(squadID ecs.EntityID, params AbilityParams, ecsmanager *common.EntityManager) {
	// Find first enemy squad (simplified targeting)
	var targetSquadID ecs.EntityID
	for _, result := range ecsmanager.World.Query(SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)

		if squadData.SquadID != squadID {
			targetSquadID = squadData.SquadID
			break
		}
	}

	if targetSquadID == 0 {
		return // No targets
	}

	unitIDs := GetUnitIDsInSquad(targetSquadID, ecsmanager)
	killed := 0

	for _, unitID := range unitIDs {
		attr := common.GetAttributesByIDWithTag(ecsmanager, unitID, SquadMemberTag)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.CurrentHealth -= params.BaseDamage
		if attr.CurrentHealth <= 0 {
			killed++
		}
	}

	fmt.Printf("[ABILITY] Fireball! %d damage dealt, %d units killed\n", params.BaseDamage, killed)
}

// EquipAbilityToLeader - ✅ Accepts ecs.EntityID (native type)
func EquipAbilityToLeader(
	leaderEntityID ecs.EntityID,
	slotIndex int,
	abilityType AbilityType,
	triggerType TriggerType,
	threshold float64,
	ecsmanager *common.EntityManager,
) error {

	if slotIndex < 0 || slotIndex >= 4 {
		return fmt.Errorf("invalid slot %d", slotIndex)
	}

	if !ecsmanager.HasComponentByIDWithTag(leaderEntityID, SquadMemberTag, LeaderComponent) {
		return fmt.Errorf("entity is not a leader")
	}

	// Get ability params
	params := GetAbilityParams(abilityType)

	// Update ability slot
	abilityData := common.GetComponentTypeByID[*AbilitySlotData](ecsmanager, leaderEntityID, AbilitySlotComponent)
	if abilityData == nil {
		return fmt.Errorf("leader entity not found")
	}

	abilityData.Slots[slotIndex] = AbilitySlot{
		AbilityType:  abilityType,
		TriggerType:  triggerType,
		Threshold:    threshold,
		HasTriggered: false,
		IsEquipped:   true,
	}

	// Update cooldown tracker
	cooldownData := common.GetComponentTypeByID[*CooldownTrackerData](ecsmanager, leaderEntityID, CooldownTrackerComponent)
	if cooldownData != nil {
		cooldownData.MaxCooldowns[slotIndex] = params.BaseCooldown
		cooldownData.Cooldowns[slotIndex] = 0
	}

	return nil
}
