package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Validation Helpers
// These functions consolidate common validation patterns used across commands

// validateSquadExists checks if a squad entity exists
// Returns error if squad ID is invalid or squad doesn't exist
func validateSquadExists(squadID ecs.EntityID, manager *common.EntityManager) error {
	if squadID == 0 {
		return fmt.Errorf("invalid squad ID")
	}

	squadEntity := squads.GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return fmt.Errorf("squad does not exist")
	}

	return nil
}

// validateGridPosition checks if grid coordinates are within valid bounds (0-2)
// Returns error if position is outside the 3x3 squad grid
func validateGridPosition(row, col int) error {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return fmt.Errorf("invalid grid position (%d, %d)", row, col)
	}
	return nil
}

// validateUnitInSquad checks if a unit exists and belongs to the specified squad
// Returns error if unit doesn't exist, isn't a squad member, or belongs to a different squad
func validateUnitInSquad(unitID, squadID ecs.EntityID, manager *common.EntityManager) error {
	if unitID == 0 {
		return fmt.Errorf("invalid unit ID")
	}

	if !manager.HasComponent(unitID, squads.SquadMemberComponent) {
		return fmt.Errorf("unit is not in a squad")
	}

	memberData := common.GetComponentTypeByID[*squads.SquadMemberData](manager, unitID, squads.SquadMemberComponent)
	if memberData == nil || memberData.SquadID != squadID {
		return fmt.Errorf("unit is not in this squad")
	}

	return nil
}

// validateGridPositionNotOccupied checks if a grid position is occupied by other units
// excludeUnitID allows excluding a specific unit (useful when moving units)
func validateGridPositionNotOccupied(squadID ecs.EntityID, row, col int, manager *common.EntityManager, excludeUnitID ecs.EntityID) error {
	existingUnits := squads.GetUnitIDsAtGridPosition(squadID, row, col, manager)
	for _, existingID := range existingUnits {
		if existingID != excludeUnitID {
			return fmt.Errorf("grid position (%d, %d) is already occupied", row, col)
		}
	}
	return nil
}

// Component Retrieval Helpers
// These functions simplify getting entities and components with error handling

// getSquadOrError retrieves a squad entity or returns an error
// Consolidates the pattern of getting and checking squad existence
func getSquadOrError(squadID ecs.EntityID, manager *common.EntityManager) (*ecs.Entity, error) {
	if squadID == 0 {
		return nil, fmt.Errorf("invalid squad ID")
	}

	squadEntity := squads.GetSquadEntity(squadID, manager)
	if squadEntity == nil {
		return nil, fmt.Errorf("squad not found")
	}

	return squadEntity, nil
}

// getSquadDataOrError retrieves squad data component or returns an error
func getSquadDataOrError(squadEntity *ecs.Entity) (*squads.SquadData, error) {
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData == nil {
		return nil, fmt.Errorf("squad has no data component")
	}
	return squadData, nil
}

// getGridPositionOrError retrieves a unit's grid position or returns an error
func getGridPositionOrError(unitID ecs.EntityID, manager *common.EntityManager) (*squads.GridPositionData, error) {
	gridPos := common.GetComponentTypeByID[*squads.GridPositionData](manager, unitID, squads.GridPositionComponent)
	if gridPos == nil {
		return nil, fmt.Errorf("unit has no grid position")
	}
	return gridPos, nil
}
