package combatcore

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/powers/effects"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// CheckAndTriggerAbilities checks and triggers leader abilities for a squad
func CheckAndTriggerAbilities(squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	leaderID := squadcore.GetLeaderID(squadID, ecsmanager)
	if leaderID == 0 {
		return
	}

	leaderEntity := ecsmanager.FindEntityByID(leaderID)
	if leaderEntity == nil {
		return
	}

	if !leaderEntity.HasComponent(squadcore.AbilitySlotComponent) {
		return
	}

	if !leaderEntity.HasComponent(squadcore.CooldownTrackerComponent) {
		return
	}

	abilityData := common.GetComponentType[*squadcore.AbilitySlotData](leaderEntity, squadcore.AbilitySlotComponent)
	cooldownData := common.GetComponentType[*squadcore.CooldownTrackerData](leaderEntity, squadcore.CooldownTrackerComponent)

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

func evaluateTrigger(slot *squadcore.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) bool {
	squadEntity := squadcore.GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return false
	}

	squadData := common.GetComponentType[*squadcore.SquadData](squadEntity, squadcore.SquadComponent)

	switch slot.TriggerType {
	case squadcore.TriggerSquadHPBelow:
		avgHP := squadcore.GetSquadHealthPercent(squadID, ecsmanager)
		return avgHP < slot.Threshold

	case squadcore.TriggerTurnCount:
		return squadData.TurnCount == int(slot.Threshold)

	case squadcore.TriggerCombatStart:
		return squadData.TurnCount == 1

	case squadcore.TriggerEnemyCount:
		enemyCount := countEnemySquads(squadID, ecsmanager)
		return float64(enemyCount) >= slot.Threshold

	case squadcore.TriggerMoraleBelow:
		return float64(squadData.Morale) < slot.Threshold

	default:
		return false
	}
}

func countEnemySquads(squadID ecs.EntityID, ecsmanager *common.EntityManager) int {
	myFaction := combatstate.GetSquadFaction(squadID, ecsmanager)
	if myFaction == 0 {
		return 0
	}

	count := 0
	for _, result := range ecsmanager.World.Query(squadcore.SquadTag) {
		otherID := result.Entity.GetID()
		otherFaction := combatstate.GetSquadFaction(otherID, ecsmanager)
		if otherFaction != 0 && otherFaction != myFaction {
			count++
		}
	}
	return count
}

func executeAbility(slot *squadcore.AbilitySlot, squadID ecs.EntityID, ecsmanager *common.EntityManager) {
	params := squadcore.GetAbilityParams(slot.AbilityType)

	switch slot.AbilityType {
	case squadcore.AbilityRally:
		applyRallyEffect(squadID, params, ecsmanager)
	case squadcore.AbilityHeal:
		applyHealEffect(squadID, params, ecsmanager)
	case squadcore.AbilityBattleCry:
		applyBattleCryEffect(squadID, params, ecsmanager)
	case squadcore.AbilityFireball:
		applyFireballEffect(squadID, params, ecsmanager)
	}
}

func applyRallyEffect(squadID ecs.EntityID, params squadcore.AbilityParams, ecsmanager *common.EntityManager) {
	effect := effects.ActiveEffect{
		Name:           "Rally",
		Source:         effects.SourceAbility,
		Stat:           effects.StatStrength,
		Modifier:       params.StrengthBonus,
		RemainingTurns: params.Duration,
	}
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, ecsmanager)
	effects.ApplyEffectToUnits(unitIDs, effect, ecsmanager)

	fmt.Printf("[ABILITY] Rally! +%d damage for %d turns\n", params.StrengthBonus, params.Duration)
}

func applyHealEffect(squadID ecs.EntityID, params squadcore.AbilityParams, ecsmanager *common.EntityManager) {
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, ecsmanager)

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

func applyBattleCryEffect(squadID ecs.EntityID, params squadcore.AbilityParams, ecsmanager *common.EntityManager) {
	squadEntity := squadcore.GetSquadEntity(squadID, ecsmanager)
	if squadEntity == nil {
		return
	}

	squadData := common.GetComponentType[*squadcore.SquadData](squadEntity, squadcore.SquadComponent)

	squadData.Morale += params.MoraleBonus

	effect := effects.ActiveEffect{
		Name:           "Battle Cry",
		Source:         effects.SourceAbility,
		Stat:           effects.StatStrength,
		Modifier:       params.StrengthBonus,
		RemainingTurns: params.Duration,
	}
	unitIDs := squadcore.GetUnitIDsInSquad(squadID, ecsmanager)
	effects.ApplyEffectToUnits(unitIDs, effect, ecsmanager)

	fmt.Printf("[ABILITY] Battle Cry! Morale and damage increased!\n")
}

func applyFireballEffect(squadID ecs.EntityID, params squadcore.AbilityParams, ecsmanager *common.EntityManager) {
	var targetSquadID ecs.EntityID
	for _, result := range ecsmanager.World.Query(squadcore.SquadTag) {
		squadEntity := result.Entity
		squadData := common.GetComponentType[*squadcore.SquadData](squadEntity, squadcore.SquadComponent)

		if squadData.SquadID != squadID {
			targetSquadID = squadData.SquadID
			break
		}
	}

	if targetSquadID == 0 {
		return
	}

	unitIDs := squadcore.GetUnitIDsInSquad(targetSquadID, ecsmanager)
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
