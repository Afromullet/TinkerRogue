package encounter

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CalculateSquadPower computes the power rating for a full squad
// Aggregates unit power scores and applies squad-level modifiers
func CalculateSquadPower(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) float64 {
	squadEntity := squads.GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return 0.0
	}

	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return 0.0
	}

	// Sum unit power scores
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	if len(unitIDs) == 0 {
		return 0.0
	}

	totalUnitPower := 0.0
	for _, unitID := range unitIDs {
		unitPower := CalculateUnitPower(unitID, manager, config)
		totalUnitPower += unitPower
	}

	// Apply squad-level modifiers
	basePower := totalUnitPower

	// Morale modifier (0-100 morale → 0.8x to 1.2x multiplier)
	moraleBonus := 1.0 + (float64(squadData.Morale) * config.MoraleMultiplier)
	basePower *= moraleBonus

	// Composition bonus (attack type diversity)
	compositionMod := calculateCompositionBonus(squadID, manager, config)
	basePower *= compositionMod

	// Health penalty (low HP squads are less effective)
	basePower *= calculateHealthMultiplier(
		squads.GetSquadHealthPercent(squadID, manager),
		config.HealthPenalty,
	)

	return basePower
}

// calculateCompositionBonus evaluates attack type diversity
func calculateCompositionBonus(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	config *EvaluationConfigData,
) float64 {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
	attackTypes := make(map[squads.AttackType]bool)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		targetRowData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
		if targetRowData != nil {
			attackTypes[targetRowData.AttackType] = true
		}
	}

	uniqueTypes := len(attackTypes)
	if bonus, exists := config.CompositionBonus[uniqueTypes]; exists {
		return bonus
	}

	return 1.0 // Default no bonus
}
