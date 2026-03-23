package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/effects"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CheckAndTriggerAbilities checks and triggers leader abilities for a squad
func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	leaderID := squads.GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return
	}

	leaderEntity := ecsmanager.FindEntityByID(leaderID)
	if leaderEntity == nil {
		return
	}

	if !leaderEntity.HasComponent(squads.AbilitySlotComponent) {
		return
	}

	if !leaderEntity.HasComponent(squads.CooldownTrackerComponent) {
		return
	}

	abilityData := common.GetComponentType[*squads.AbilitySlotData](leaderEntity, squads.AbilitySlotComponent)
	cooldownData := common.GetComponentType[*squads.CooldownTrackerData](leaderEntity, squads.CooldownTrackerComponent)

	if abilityData == nil || cooldownData == nil {
		return
	}

	for i := 0; i < 4; i++ {
		slot := &abilityData.Slots[i]

		if !slot.IsEquipped || cooldownData.Cooldowns[i] > 0 || slot.HasTriggered {
			continue
		}

		triggered := evaluateTrigger(slot, squadID, ecsmanager)
		if !triggered {
			continue
		}

		executeAbility(slot, squadID, ecsmanager)

		cooldownData.Cooldowns[i] = cooldownData.MaxCooldowns[i]
		slot.HasTriggered = true
	}

	for i := 0; i < 4; i++ {
		if cooldownData.Cooldowns[i] > 0 {
			cooldownData.Cooldowns[i]--
		}
	}
}

func evaluateTrigger(slot *squads.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	squadEntity := squads.GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return false
	}

	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

	switch slot.TriggerType {
	case squads.TriggerSquadHPBelow:
		avgHP := squads.GetSquadHealthPercent(squadID, ecsmanager)
		return avgHP < slot.Threshold

	case squads.TriggerTurnCount:
		return squadData.TurnCount == int(slot.Threshold)

	case squads.TriggerCombatStart:
		return squadData.TurnCount == 1

	case squads.TriggerEnemyCount:
		enemyCount := countEnemySquads(ecsmanager)
		return float64(enemyCount) >= slot.Threshold

	case squads.TriggerMoraleBelow:
		return float64(squadData.Morale) < slot.Threshold

	default:
		return false
	}
}

func countEnemySquads(ecsmanager *common.EntityManager) int {
	count := 0
	for _, result := range ecsmanager.World.Query(squads.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

		if len(squadData.Name) > 0 && squadData.Name[0] != 'P' {
			count++
		}
	}
	return count
}

func executeAbility(slot *squads.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	params := squads.GetAbilityParams(slot.AbilityType)

	switch slot.AbilityType {
	case squads.AbilityRally:
		applyRallyEffect(squadID, params, ecsmanager)
	case squads.AbilityHeal:
		applyHealEffect(squadID, params, ecsmanager)
	case squads.AbilityBattleCry:
		applyBattleCryEffect(squadID, params, ecsmanager)
	case squads.AbilityFireball:
		applyFireballEffect(squadID, params, ecsmanager)
	}
}

func applyRallyEffect(squadID ecs.EntityID, params squads.AbilityParams, ecsmanager *common.EntityManager) {
	effect := effects.ActiveEffect{
		Name:           "Rally",
		Source:         effects.SourceAbility,
		Stat:           effects.StatStrength,
		Modifier:       params.StrengthBonus,
		RemainingTurns: params.Duration,
	}
	unitIDs := squads.GetUnitIDsInSquad(squadID, ecsmanager)
	effects.ApplyEffectToUnits(unitIDs, effect, ecsmanager)

	fmt.Printf("[ABILITY] Rally! +%d damage for %d turns\n", params.StrengthBonus, params.Duration)
}

func applyHealEffect(squadID ecs.EntityID, params squads.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := squads.GetUnitIDsInSquad(squadID, ecsmanager)

	healed := 0
	for _, unitID := range unitIDs {
		entity := ecsmanager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		if attr.CurrentHealth <= 0 {
			continue
		}

		attr.CurrentHealth += params.HealAmount
		if attr.CurrentHealth > attr.GetMaxHealth() {
			attr.CurrentHealth = attr.GetMaxHealth()
		}
		healed++
	}

	fmt.Printf("[ABILITY] Healing Aura! %d units restored %d HP\n", healed, params.HealAmount)
}

func applyBattleCryEffect(squadID ecs.EntityID, params squads.AbilityParams, ecsmanager *common.EntityManager) {
	squadEntity := squads.GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

	squadData.Morale += params.MoraleBonus

	effect := effects.ActiveEffect{
		Name:           "Battle Cry",
		Source:         effects.SourceAbility,
		Stat:           effects.StatStrength,
		Modifier:       params.StrengthBonus,
		RemainingTurns: params.Duration,
	}
	unitIDs := squads.GetUnitIDsInSquad(squadID, ecsmanager)
	effects.ApplyEffectToUnits(unitIDs, effect, ecsmanager)

	fmt.Printf("[ABILITY] Battle Cry! Morale and damage increased!\n")
}

func applyFireballEffect(squadID ecs.EntityID, params squads.AbilityParams, ecsmanager *common.EntityManager) {
	var targetSquadID ecs.EntityID
	for _, result := range ecsmanager.World.Query(squads.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)

		if squadData.SquadID != squadID {
			targetSquadID = squadData.SquadID
			break
		}
	}

	if targetSquadID == 0 {
		return
	}

	unitIDs := squads.GetUnitIDsInSquad(targetSquadID, ecsmanager)
	killed := 0

	for _, unitID := range unitIDs {
		entity := ecsmanager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
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
