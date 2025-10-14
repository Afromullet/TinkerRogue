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

func CreateEmptySquad(squadmanager *common.EntityManager,
	squadName string) {

	squadEntity := squadmanager.World.NewEntity()
	squadID := squadEntity.GetID()

	squadEntity.AddComponent(SquadComponent, &SquadData{
		SquadID:       squadID,
		Name:          squadName,
		Morale:        100,
		TurnCount:     0,
		MaxUnits:      9,
		UsedCapacity:  0.0,
		TotalCapacity: 6, // Default capacity (no leader yet)
	})

	squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})

}

// gridRow and gridCol are the row and col we want to anchor the unit at
func AddUnitToSquad(
	squadID ecs.EntityID,
	squadmanager *common.EntityManager,
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

	// Check capacity before adding unit
	unitCapacityCost := unit.Attributes.GetCapacityCost()
	if !CanAddUnitToSquad(squadID, unitCapacityCost, squadmanager) {
		remaining := GetSquadRemainingCapacity(squadID, squadmanager)
		return fmt.Errorf("insufficient squad capacity: need %.2f, have %.2f remaining (unit %s costs %.2f)",
			unitCapacityCost, remaining, unit.Name, unitCapacityCost)
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

	// Update squad capacity tracking
	UpdateSquadCapacity(squadID, squadmanager)

	return nil
}

// RemoveUnitFromSquad - ✅ Accepts ecs.EntityID (native type)
func RemoveUnitFromSquad(unitEntityID ecs.EntityID, squadmanager *common.EntityManager) error {
	unitEntity := FindUnitByID(unitEntityID, squadmanager)
	if unitEntity == nil {
		return fmt.Errorf("unit entity not found")
	}

	if !unitEntity.HasComponent(SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	// Get the squad ID before removing to update capacity
	memberData := common.GetComponentType[*SquadMemberData](unitEntity, SquadMemberComponent)
	squadID := memberData.SquadID

	// In bytearena/ecs, we can't remove components
	// Workaround: Set SquadID to 0 to mark as "removed"
	memberData.SquadID = 0

	// Update squad capacity tracking after removal
	UpdateSquadCapacity(squadID, squadmanager)

	return nil
}
