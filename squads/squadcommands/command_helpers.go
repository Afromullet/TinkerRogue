package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/squads"

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

	if !manager.HasComponentByIDWithTag(unitID, squads.SquadMemberTag, squads.SquadMemberComponent) {
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

// getUnitEntityOrError retrieves a unit entity or returns an error
func getUnitEntityOrError(unitID ecs.EntityID, manager *common.EntityManager) (*ecs.Entity, error) {
	if unitID == 0 {
		return nil, fmt.Errorf("invalid unit ID")
	}

	unitEntity := common.FindEntityByIDWithTag(manager, unitID, squads.SquadMemberTag)
	if unitEntity == nil {
		return nil, fmt.Errorf("unit not found")
	}

	return unitEntity, nil
}

// getGridPositionOrError retrieves a unit's grid position or returns an error
func getGridPositionOrError(unitID ecs.EntityID, manager *common.EntityManager) (*squads.GridPositionData, error) {
	gridPos := common.GetComponentTypeByID[*squads.GridPositionData](manager, unitID, squads.GridPositionComponent)
	if gridPos == nil {
		return nil, fmt.Errorf("unit has no grid position")
	}
	return gridPos, nil
}

// getUnitName retrieves the name of a unit entity, with fallback to default
// Returns "Unit" if no name component exists or if retrieval fails
func getUnitName(unitID ecs.EntityID, manager *common.EntityManager) string {
	if nameComp, ok := manager.GetComponent(unitID, common.NameComponent); ok {
		if name := nameComp.(*common.Name); name != nil {
			return name.NameStr
		}
	}
	return "Unit"
}

// Leader Component Helpers
// These functions manage the set of components that make up a squad leader

// addLeaderComponents adds all leader-related components to a unit entity
// Includes LeaderComponent, AbilitySlotComponent, and CooldownTrackerComponent
func addLeaderComponents(entity *ecs.Entity) {
	// Add leader component with default values
	entity.AddComponent(squads.LeaderComponent, &squads.LeaderData{
		Leadership: 10,
		Experience: 0,
	})

	// Add ability slots
	entity.AddComponent(squads.AbilitySlotComponent, &squads.AbilitySlotData{
		Slots: [4]squads.AbilitySlot{},
	})

	// Add cooldown tracker
	entity.AddComponent(squads.CooldownTrackerComponent, &squads.CooldownTrackerData{
		Cooldowns:    [4]int{0, 0, 0, 0},
		MaxCooldowns: [4]int{0, 0, 0, 0},
	})
}

// removeLeaderComponents removes all leader-related components from a unit entity
// Removes LeaderComponent, AbilitySlotComponent, and CooldownTrackerComponent if they exist
func removeLeaderComponents(entity *ecs.Entity) {
	if entity.HasComponent(squads.LeaderComponent) {
		entity.RemoveComponent(squads.LeaderComponent)
	}
	if entity.HasComponent(squads.AbilitySlotComponent) {
		entity.RemoveComponent(squads.AbilitySlotComponent)
	}
	if entity.HasComponent(squads.CooldownTrackerComponent) {
		entity.RemoveComponent(squads.CooldownTrackerComponent)
	}
}
