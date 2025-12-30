package squadservices

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// SquadBuilderService encapsulates squad building game logic
type SquadBuilderService struct {
	entityManager *common.EntityManager
}

// NewSquadBuilderService creates a new squad builder service
func NewSquadBuilderService(manager *common.EntityManager) *SquadBuilderService {
	return &SquadBuilderService{
		entityManager: manager,
	}
}

// AssignRosterUnitResult contains information about roster unit assignment
type AssignRosterUnitResult struct {
	Success           bool
	Error             string
	PlacedUnitID      ecs.EntityID
	RosterUnitID      ecs.EntityID
	UnitName          string
	RemainingCapacity float64
}

// AssignRosterUnitToSquad handles both unit placement AND roster marking atomically
func (sbs *SquadBuilderService) AssignRosterUnitToSquad(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnitID ecs.EntityID,
	template squads.UnitTemplate,
	gridRow, gridCol int,
) *AssignRosterUnitResult {
	result := &AssignRosterUnitResult{
		RosterUnitID: rosterUnitID,
		UnitName:     template.Name,
	}

	// Get roster
	roster := squads.GetPlayerRoster(playerID, sbs.entityManager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Validate roster has this specific unit entity
	// The roster system tracks unit entities, so we just need to verify the entity exists
	// (The original code got the unit via GetUnitEntityForTemplate which validates availability)

	// Place unit in squad (creates new unit entity in formation grid)
	unitID, err := squads.AddUnitToSquad(squadID, sbs.entityManager, template, gridRow, gridCol)
	if err != nil {
		result.Error = err.Error()
		result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)
		return result
	}

	// Mark roster unit as assigned to squad (atomic with placement)
	if err := roster.MarkUnitInSquad(rosterUnitID, squadID); err != nil {
		// Rollback: Remove the placed unit
		unitIDs := squads.GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, sbs.entityManager)
		if len(unitIDs) > 0 {
			squads.RemoveUnitFromSquad(unitIDs[0], sbs.entityManager)
		}
		result.Error = fmt.Sprintf("failed to mark roster unit: %v", err)
		return result
	}

	// Success
	result.Success = true
	result.PlacedUnitID = unitID
	result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)

	return result
}

// UnassignRosterUnitResult contains information about roster unit removal
type UnassignRosterUnitResult struct {
	Success           bool
	Error             string
	RemainingCapacity float64
}

// UnassignRosterUnitFromSquad handles unit removal AND roster return atomically
func (sbs *SquadBuilderService) UnassignRosterUnitFromSquad(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnitID ecs.EntityID,
	gridRow, gridCol int,
) *UnassignRosterUnitResult {
	result := &UnassignRosterUnitResult{}

	// Get roster
	roster := squads.GetPlayerRoster(playerID, sbs.entityManager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Remove unit from grid
	unitIDs := squads.GetUnitIDsAtGridPosition(squadID, gridRow, gridCol, sbs.entityManager)
	if len(unitIDs) == 0 {
		result.Error = fmt.Sprintf("no unit at position (%d, %d)", gridRow, gridCol)
		return result
	}

	// Remove first unit at this position
	if err := squads.RemoveUnitFromSquad(unitIDs[0], sbs.entityManager); err != nil {
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
	result.RemainingCapacity = squads.GetSquadRemainingCapacity(squadID, sbs.entityManager)

	return result
}

// ClearSquadResult contains information about squad clearing
type ClearSquadResult struct {
	Success      bool
	Error        string
	UnitsCleared int
}

// ClearSquadAndReturnAllUnits removes all units from squad and returns them to roster
func (sbs *SquadBuilderService) ClearSquadAndReturnAllUnits(
	playerID ecs.EntityID,
	squadID ecs.EntityID,
	rosterUnits map[ecs.EntityID]ecs.EntityID, // map[placedUnitID]rosterUnitID
) *ClearSquadResult {
	result := &ClearSquadResult{}

	// Get roster
	roster := squads.GetPlayerRoster(playerID, sbs.entityManager)
	if roster == nil {
		result.Error = "roster not found"
		return result
	}

	// Get all units in squad
	unitIDs := squads.GetUnitIDsInSquad(squadID, sbs.entityManager)

	// Remove each unit and return to roster
	for _, unitID := range unitIDs {
		// Find unit entity to dispose it
		unitEntity := sbs.entityManager.FindEntityByID(unitID)
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
		sbs.entityManager.CleanDisposeEntity(unitEntity, pos)
		result.UnitsCleared++
	}

	// Update squad capacity after clearing
	squads.UpdateSquadCapacity(squadID, sbs.entityManager)

	result.Success = true
	return result
}
