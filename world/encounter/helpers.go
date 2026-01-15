package encounter

import (
	"game_main/common"
	"game_main/tactical/evaluation"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// calculateHealthMultiplier applies health penalty with minimum floor
func calculateHealthMultiplier(healthPercent, healthPenalty float64) float64 {
	multiplier := healthPercent * healthPenalty
	if multiplier < MinimumHealthMultiplier {
		return MinimumHealthMultiplier
	}
	return multiplier
}

// applyDeploymentWeight applies deployment weight based on squad status
func applyDeploymentWeight(power float64, isDeployed bool, config *EvaluationConfigData) float64 {
	if isDeployed {
		return power * config.DeployedWeight
	}
	return power * config.ReserveWeight
}

// calculateRoleValue computes power contribution from unit role.
// Uses shared role multipliers from evaluation package.
func calculateRoleValue(roleData *squads.UnitRoleData) float64 {
	roleMultiplier := evaluation.GetRoleMultiplier(roleData.Role)
	return roleMultiplier * RoleScalingFactor
}

// calculateAbilityValue computes power contribution from leader abilities
func calculateAbilityValue(entity *ecs.Entity) float64 {
	if !entity.HasComponent(squads.LeaderComponent) {
		return 0.0
	}
	return calculateLeaderAbilityValue(entity)
}

// calculateCoverValue computes power contribution from cover provision
func calculateCoverValue(entity *ecs.Entity) float64 {
	if !entity.HasComponent(squads.CoverComponent) {
		return 0.0
	}
	coverData := common.GetComponentType[*squads.CoverData](entity, squads.CoverComponent)
	if coverData == nil {
		return 0.0
	}
	return coverData.CoverValue * CoverScalingFactor * CoverBeneficiaryMultiplier
}

// calculateLeaderAbilityValue sums power value of equipped abilities
func calculateLeaderAbilityValue(entity *ecs.Entity) float64 {
	if !entity.HasComponent(squads.AbilitySlotComponent) {
		return 0.0
	}

	abilitySlots := common.GetComponentType[*squads.AbilitySlotData](entity, squads.AbilitySlotComponent)
	if abilitySlots == nil {
		return 0.0
	}

	totalValue := 0.0
	for _, slot := range abilitySlots.Slots {
		if slot.IsEquipped {
			totalValue += AbilityPowerValues[slot.AbilityType]
		}
	}

	return totalValue
}
