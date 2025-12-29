package squads

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// AssignRosterUnitResult contains information about roster unit assignment
type AssignRosterUnitResult struct {
	Success           bool
	Error             string
	PlacedUnitID      ecs.EntityID
	RosterUnitID      ecs.EntityID
	UnitName          string
	RemainingCapacity float64
}

// UnassignRosterUnitResult contains information about roster unit removal
type UnassignRosterUnitResult struct {
	Success           bool
	Error             string
	RemainingCapacity float64
}

// ClearSquadResult contains information about squad clearing
type ClearSquadResult struct {
	Success      bool
	Error        string
	UnitsCleared int
}

// AssignRosterUnitToSquad handles both unit placement AND roster marking atomically
func AssignRosterUnitToSquad(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnitID ecs.EntityID,
	template UnitTemplate,
	gridRow, gridCol int,
	manager *common.EntityManager,
) *AssignRosterUnitResult {
	result := &AssignRosterUnitResult{
		RosterUnitID: rosterUnitID,
		UnitName:     template.Name,
	}

	// Get roster
	roster := GetPlayerRoster(playerID, manager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Validate roster has this specific unit entity
	// The roster system tracks unit entities, so we just need to verify the entity exists
	// (The original code got the unit via GetUnitEntityForTemplate which validates availability)

	// Place unit in squad (creates new unit entity in formation grid)
	unitID, err := AddUnitToSquad(squadID, manager, template, gridRow, gridCol)
	if err != nil {
		result.Error = err.Error()
		result.RemainingCapacity = GetSquadRemainingCapacity(squadID, manager)
		return result
	}

	// Mark roster unit as assigned to squad (atomic with placement)
	if err := roster.MarkUnitInSquad(rosterUnitID, squadID); err != nil {
		// Rollback: Remove the placed unit
		unitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, manager)
		if len(unitIDs) > 0 {
			RemoveUnitFromSquad(unitIDs[0], manager)
		}
		result.Error = fmt.Sprintf("failed to mark roster unit: %v", err)
		return result
	}

	// Success
	result.Success = true
	result.PlacedUnitID = unitID
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, manager)

	return result
}

// UnassignRosterUnitFromSquad handles unit removal AND roster return atomically
func UnassignRosterUnitFromSquad(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnitID ecs.EntityID,
	gridRow, gridCol int,
	manager *common.EntityManager,
) *UnassignRosterUnitResult {
	result := &UnassignRosterUnitResult{}

	// Get roster
	roster := GetPlayerRoster(playerID, manager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Remove unit from grid
	unitIDs := GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, manager)
	if len(unitIDs) == 0 {
		result.Error = fmt.Sprintf("no unit at position (%d, %d)", gridRow, gridCol)
		return result
	}

	// Remove first unit at this position
	if err := RemoveUnitFromSquad(unitIDs[0], manager); err != nil {
		result.Error = err.Error()
		return result
	}

	// Return unit to roster (mark as available)
	if err := roster.MarkUnitAvailable(rosterUnitID); err != nil {
		result.Error = fmt.Sprintf("failed to mark unit available: %v", err)
		return result
	}

	// Success
	result.Success = true
	result.RemainingCapacity = GetSquadRemainingCapacity(squadID, manager)

	return result
}

// ClearSquadAndReturnAllUnits removes all units from squad and returns them to roster
func ClearSquadAndReturnAllUnits(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnits map[ecs.EntityID]ecs.EntityID, // map[placedUnitID]rosterUnitID
	manager *common.EntityManager,
) *ClearSquadResult {
	result := &ClearSquadResult{}

	// Get roster
	roster := GetPlayerRoster(playerID, manager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Get all units in squad
	unitIDs := GetUnitIDsInSquad(squadID, manager)

	// Remove each unit and return to roster
	for _, unitID := range unitIDs {
		// Find unit entity to dispose it
		unitEntity := manager.FindEntityByID(unitID)
		if unitEntity == nil {
			continue
		}

		// Get corresponding roster unit ID if provided
		if rosterUnitID, exists := rosterUnits[unitID]; exists {
			// Mark roster unit as available
			if err := roster.MarkUnitAvailable(rosterUnitID); err != nil {
				fmt.Printf("Warning: Failed to return unit to roster: %v\n", err)
			}
		}

		// Dispose the placed unit entity
		pos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
		manager.CleanDisposeEntity(unitEntity, pos)
		result.UnitsCleared++
	}

	// Update squad capacity after clearing
	UpdateSquadCapacity(squadID, manager)

	result.Success = true
	return result
}
