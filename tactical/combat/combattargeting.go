package combat

import (
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/tactical/unitdefs"

	"github.com/bytearena/ecs"
)

// CanUnitAttack checks if a unit is alive, can act, and within attack range
func CanUnitAttack(attackerID ecs.EntityID, squadDistance int, manager *common.EntityManager) bool {
	entity := manager.FindEntityByID(attackerID)
	if entity == nil {
		return false
	}

	// Check if unit is alive and can act
	attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
	if attr == nil || attr.CurrentHealth <= 0 || !attr.CanAct {
		return false
	}

	// Check if unit has attack range component and is within range
	if !entity.HasComponent(squads.AttackRangeComponent) {
		return false
	}

	rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
	return rangeData != nil && rangeData.Range >= squadDistance
}

// SelectTargetUnits determines targets based on attack type (public for GUI and internal use)
func SelectTargetUnits(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	entity := manager.FindEntityByID(attackerID)
	if entity == nil {
		return []ecs.EntityID{}
	}

	// Check if attacker has targeting component
	if !entity.HasComponent(squads.TargetRowComponent) {
		return []ecs.EntityID{}
	}

	targetData := common.GetComponentType[*squads.TargetRowData](entity, squads.TargetRowComponent)
	if targetData == nil {
		return []ecs.EntityID{}
	}

	switch targetData.AttackType {
	case unitdefs.AttackTypeMeleeRow:
		return selectMeleeRowTargets(attackerID, defenderSquadID, manager)
	case unitdefs.AttackTypeMeleeColumn:
		return selectMeleeColumnTargets(attackerID, defenderSquadID, manager)
	case unitdefs.AttackTypeRanged:
		return selectRangedTargets(attackerID, defenderSquadID, manager)
	case unitdefs.AttackTypeMagic:
		return selectMagicTargets(defenderSquadID, targetData.TargetCells, manager)
	case unitdefs.AttackTypeHeal:
		// Heal targets own squad - defenderSquadID is passed as the healer's squad by the caller
		return selectHealTargets(defenderSquadID, targetData.TargetCells, manager)
	default:
		return []ecs.EntityID{}
	}
}

// selectMeleeRowTargets targets front row (row 0), piercing to next row if empty
// Always targets all units in the row (up to 3)
func selectMeleeRowTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	// Try each row starting from front (0 -> 1 -> 2)
	for row := 0; row <= 2; row++ {
		targets := getUnitsInRow(defenderSquadID, row, manager)

		if len(targets) > 0 {
			return targets // Return all units in the row
		}
	}

	return []ecs.EntityID{}
}

// selectMeleeColumnTargets targets column directly across from attacker, piercing to next column if empty
// Always targets all units in the column (piercing attack)
func selectMeleeColumnTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerPos := common.GetComponentTypeByID[*squads.GridPositionData](manager, attackerID, squads.GridPositionComponent)
	if attackerPos == nil {
		return []ecs.EntityID{}
	}

	attackerCol := attackerPos.AnchorCol

	// Try columns starting from attacker's column, wrapping around
	// Example: attackerCol=1 -> try columns 1, 2, 0
	for offset := 0; offset < 3; offset++ {
		col := (attackerCol + offset) % 3
		targets := getUnitsInColumn(defenderSquadID, col, manager)

		if len(targets) > 0 {
			return targets // Return all units in the column
		}
	}

	return []ecs.EntityID{}
}

// selectRangedTargets targets same row as attacker (all units), with fallback logic
func selectRangedTargets(attackerID, defenderSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	attackerPos := common.GetComponentTypeByID[*squads.GridPositionData](manager, attackerID, squads.GridPositionComponent)
	if attackerPos == nil {
		return []ecs.EntityID{}
	}

	attackerRow := attackerPos.AnchorRow

	// Try same row as attacker - return ALL units in row
	targets := getUnitsInRow(defenderSquadID, attackerRow, manager)
	if len(targets) > 0 {
		return targets
	}

	// Fallback: lowest armor, furthest row, leftmost column tiebreaker
	return selectLowestArmorTarget(defenderSquadID, manager)
}

// selectMagicTargets uses cell-based patterns WITHOUT pierce-through
func selectMagicTargets(defenderSquadID ecs.EntityID, targetCells [][2]int, manager *common.EntityManager) []ecs.EntityID {
	var targets []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for _, cell := range targetCells {
		row, col := cell[0], cell[1]

		// Get units at exact cell (no pierce)
		cellTargets := squads.GetUnitIDsAtGridPosition(defenderSquadID, row, col, manager)

		for _, unitID := range cellTargets {
			if !seen[unitID] {
				targets = append(targets, unitID)
				seen[unitID] = true
			}
		}
	}

	return targets
}

// getUnitsInLine returns all ALIVE units in a specific row or column
// If isRow is true, lineIndex specifies the row; otherwise it specifies the column
func getUnitsInLine(squadID ecs.EntityID, lineIndex int, isRow bool, manager *common.EntityManager) []ecs.EntityID {
	var units []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	// Iterate through the perpendicular dimension
	for i := 0; i <= 2; i++ {
		var cellUnits []ecs.EntityID
		if isRow {
			// lineIndex is row, i is column
			cellUnits = squads.GetUnitIDsAtGridPosition(squadID, lineIndex, i, manager)
		} else {
			// lineIndex is column, i is row
			cellUnits = squads.GetUnitIDsAtGridPosition(squadID, i, lineIndex, manager)
		}

		for _, unitID := range cellUnits {
			if !seen[unitID] {
				// Use helper to check if unit is alive
				if squads.GetAliveUnitAttributes(unitID, manager) != nil {
					units = append(units, unitID)
					seen[unitID] = true
				}
			}
		}
	}

	return units
}

// getUnitsInRow returns all ALIVE units in a specific row
func getUnitsInRow(squadID ecs.EntityID, row int, manager *common.EntityManager) []ecs.EntityID {
	return getUnitsInLine(squadID, row, true, manager)
}

// getUnitsInColumn returns all ALIVE units in a specific column
func getUnitsInColumn(squadID ecs.EntityID, col int, manager *common.EntityManager) []ecs.EntityID {
	return getUnitsInLine(squadID, col, false, manager)
}

// selectLowestArmorTarget selects lowest armor target, furthest row on tie, leftmost column on further tie
func selectLowestArmorTarget(squadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	allUnits := squads.GetUnitIDsInSquad(squadID, manager)

	if len(allUnits) == 0 {
		return []ecs.EntityID{}
	}

	// Find unit with lowest armor
	var bestTarget ecs.EntityID
	lowestArmor := int(^uint(0) >> 1) // Max int
	furthestRow := -1
	leftmostCol := 3 // Start with invalid column (max is 2)

	for _, unitID := range allUnits {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr == nil || attr.CurrentHealth <= 0 {
			continue
		}

		armor := attr.GetPhysicalResistance()
		pos := common.GetComponentTypeByID[*squads.GridPositionData](manager, unitID, squads.GridPositionComponent)
		if pos == nil {
			continue
		}

		row := pos.AnchorRow
		col := pos.AnchorCol

		// Select if:
		// 1. Lower armor, OR
		// 2. Same armor AND further row, OR
		// 3. Same armor AND same row AND more left column
		if armor < lowestArmor ||
			(armor == lowestArmor && row > furthestRow) ||
			(armor == lowestArmor && row == furthestRow && col < leftmostCol) {
			lowestArmor = armor
			furthestRow = row
			leftmostCol = col
			bestTarget = unitID
		}
	}

	if bestTarget == 0 {
		return []ecs.EntityID{}
	}

	return []ecs.EntityID{bestTarget}
}

// IsHealUnit checks if a unit has AttackTypeHeal
func IsHealUnit(unitID ecs.EntityID, manager *common.EntityManager) bool {
	targetData := common.GetComponentTypeByID[*squads.TargetRowData](manager, unitID, squads.TargetRowComponent)
	return targetData != nil && targetData.AttackType == unitdefs.AttackTypeHeal
}

// selectHealTargets finds alive friendly units at targetCells that are below max HP.
// Same cell-based logic as selectMagicTargets but targets the healer's own squad.
func selectHealTargets(healerSquadID ecs.EntityID, targetCells [][2]int, manager *common.EntityManager) []ecs.EntityID {
	var targets []ecs.EntityID
	seen := make(map[ecs.EntityID]bool)

	for _, cell := range targetCells {
		row, col := cell[0], cell[1]

		cellTargets := squads.GetUnitIDsAtGridPosition(healerSquadID, row, col, manager)

		for _, unitID := range cellTargets {
			if seen[unitID] {
				continue
			}
			seen[unitID] = true

			// Only heal alive units that are missing HP
			attr := squads.GetAliveUnitAttributes(unitID, manager)
			if attr == nil {
				continue
			}
			if attr.CurrentHealth < attr.GetMaxHealth() {
				targets = append(targets, unitID)
			}
		}
	}

	return targets
}

// SelectHealTargets is the public wrapper that reads target cells from the healer's component
// and calls selectHealTargets on the healer's own squad.
func SelectHealTargets(healerID, healerSquadID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	targetData := common.GetComponentTypeByID[*squads.TargetRowData](manager, healerID, squads.TargetRowComponent)
	if targetData == nil {
		return []ecs.EntityID{}
	}

	return selectHealTargets(healerSquadID, targetData.TargetCells, manager)
}
