package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"

	"github.com/bytearena/ecs"
)

// ========================================
// SQUAD RELATED
// ========================================

func CreateEmptySquad(squadmanager *SquadECSManager,
	squadName string) {

	squadEntity := squadmanager.Manager.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:   squadID,
		Name:      squadName,
		Morale:    100,
		TurnCount: 0,
		MaxUnits:  9,
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

}

// gridRow and gridCol are the row and col we want to anchor the unit at
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *SquadECSManager,
	unit UnitTemplate,
	gridRow, gridCol int) error {

	// Validate position using the provided parameters, not unit template values
	if gridRow < 0 || gridRow > 2 || gridCol < 0 || gridCol > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", gridRow, gridCol)
	}

	// Check if position occupied
	existingUnitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, squadmanager)
	if len(existingUnitIDs) > 0 {
		return fmt.Errorf("grid position (%d, %d) already occupied", gridRow, gridCol)
	}

	// Create unit entity (adds GridPositionComponent with default 0,0)
	unitEntity, err := CreateUnitEntity(squadmanager, unit)
	if err != nil {
		return fmt.Errorf("invalid unit for %s: %w", unit.Name, err)
	}

	// Add SquadMemberComponent to link unit to squad
	unitEntity.AddComponent(SquadMemberComponent, &SquadMemberData{
		SquadID: squadID,
	})

	// Update GridPositionComponent with actual grid position
	gridPos := common.GetComponentType[*GridPositionData](unitEntity, GridPositionComponent)
	gridPos.AnchorRow = gridRow
	gridPos.AnchorCol = gridCol

	return nil
}

// RemoveUnitFromSquad - ✅ Accepts ecs.EntityID (native type)
func RemoveUnitFromSquad(unitEntityID ecs.EntityID, ecsmanager *common.EntityManager) error {
	unitEntity := FindUnitByID(unitEntityID, ecsmanager)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	// In bytearena/ecs, we can't remove components
	// Workaround: Set SquadID to 0 to mark as "removed"
	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	memberData.SquadID = 0

	return nil
}
