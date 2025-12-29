package behavior

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// AttackTypeFilter represents attack types to filter for
type AttackTypeFilter []squads.AttackType

var (
	// MeleeAttackTypes includes all melee attack types
	MeleeAttackTypes = AttackTypeFilter{squads.AttackTypeMeleeRow, squads.AttackTypeMeleeColumn}

	// RangedAttackTypes includes ranged and magic attack types
	RangedAttackTypes = AttackTypeFilter{squads.AttackTypeRanged, squads.AttackTypeMagic}
)

// hasUnitsWithAttackType checks if squad has any units matching attack types
func hasUnitsWithAttackType(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	attackTypes AttackTypeFilter,
) bool {
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		targetRow := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
		if targetRow == nil {
			continue
		}

		// Check if attack type matches
		for _, attackType := range attackTypes {
			if targetRow.AttackType == attackType {
				return true
			}
		}
	}

	return false
}

// getMaxRangeForAttackTypes returns maximum attack range among matching units
func getMaxRangeForAttackTypes(
	squadID ecs.EntityID,
	manager *common.EntityManager,
	attackTypes AttackTypeFilter,
	defaultRange int,
) int {
	maxRange := defaultRange
	unitIDs := squads.GetUnitIDsInSquad(squadID, manager)

	for _, unitID := range unitIDs {
		entity := manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		targetRow := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
		if targetRow == nil {
			continue
		}

		// Check if attack type matches
		matches := false
		for _, attackType := range attackTypes {
			if targetRow.AttackType == attackType {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}

		// Get range
		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		if rangeData != nil && rangeData.Range > maxRange {
			maxRange = rangeData.Range
		}
	}

	return maxRange
}
