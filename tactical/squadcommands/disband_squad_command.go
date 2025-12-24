package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// DisbandSquadCommand disbands a squad and returns all units to the roster
// Captures full squad state for undo
type DisbandSquadCommand struct {
	entityManager *common.EntityManager
	playerID      ecs.EntityID
	squadID       ecs.EntityID

	// Captured state for undo
	savedSquadData *squads.SquadData
	savedPosition  *coords.LogicalPosition
	savedUnits     []CapturedUnitState
}

// NewDisbandSquadCommand creates a new disband squad command
func NewDisbandSquadCommand(
	manager *common.EntityManager,
	playerID ecs.EntityID,
	squadID ecs.EntityID,
) *DisbandSquadCommand {
	return &DisbandSquadCommand{
		entityManager: manager,
		playerID:      playerID,
		squadID:       squadID,
	}
}

// Validate checks if the squad can be disbanded
func (cmd *DisbandSquadCommand) Validate() error {
	// Check if squad exists
	if err := validateSquadExists(cmd.squadID, cmd.entityManager); err != nil {
		return err
	}

	// Note: Additional validation could check if squad is in combat or deployed
	// For now, we allow disbanding any squad
	// Future: Add checks like:
	// if combat.IsSquadInCombat(cmd.squadID, cmd.entityManager) {
	//     return fmt.Errorf("cannot disband squad in combat")
	// }

	return nil
}

// Execute disbands the squad and returns units to roster
func (cmd *DisbandSquadCommand) Execute() error {
	// Capture state for undo BEFORE making changes
	if err := cmd.captureState(); err != nil {
		return fmt.Errorf("failed to capture squad state: %w", err)
	}

	// Get player roster
	roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
	if roster == nil {
		return fmt.Errorf("player has no roster")
	}

	// Return all units to roster
	for _, unitState := range cmd.savedUnits {
		// Mark unit as available in roster
		if err := roster.MarkUnitAvailable(unitState.UnitID); err != nil {
			// If marking available fails, the unit might not have been in roster
			// This is OK - we'll continue with other units
			continue
		}
	}

	// Remove all units from squad (dispose entities)
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil {
			// Get position component for cleanup
			pos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			// Use CleanDisposeEntity to remove from both ECS World and GlobalPositionSystem
			cmd.entityManager.CleanDisposeEntity(unitEntity, pos)
		}
	}

	// Remove squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity != nil {
		// Get position component for cleanup
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		// Use CleanDisposeEntity to remove from both ECS World and GlobalPositionSystem
		cmd.entityManager.CleanDisposeEntity(squadEntity, pos)
	}

	return nil
}

// Undo recreates the squad from saved state
func (cmd *DisbandSquadCommand) Undo() error {
	if cmd.savedSquadData == nil {
		return fmt.Errorf("no saved state available for undo")
	}

	// Create new squad entity with saved ID
	squadEntity := cmd.entityManager.World.NewEntity()
	newSquadID := squadEntity.GetID()

	// Restore squad data (using new squad ID)
	restoredSquadData := &squads.SquadData{
		SquadID:       newSquadID,
		Formation:     cmd.savedSquadData.Formation,
		Name:          cmd.savedSquadData.Name,
		Morale:        cmd.savedSquadData.Morale,
		SquadLevel:    cmd.savedSquadData.SquadLevel,
		TurnCount:     cmd.savedSquadData.TurnCount,
		MaxUnits:      cmd.savedSquadData.MaxUnits,
		UsedCapacity:  cmd.savedSquadData.UsedCapacity,
		TotalCapacity: cmd.savedSquadData.TotalCapacity,
	}

	squadEntity.AddComponent(squads.SquadComponent, restoredSquadData)

	// Restore squad position
	if cmd.savedPosition != nil {
		squadEntity.AddComponent(common.PositionComponent, cmd.savedPosition)
		common.GlobalPositionSystem.AddEntity(newSquadID, *cmd.savedPosition)
	} else {
		squadEntity.AddComponent(common.PositionComponent, &coords.LogicalPosition{})
	}

	// Get player roster
	roster := squads.GetPlayerRoster(cmd.playerID, cmd.entityManager)
	if roster == nil {
		return fmt.Errorf("player has no roster")
	}

	// Recreate all units
	for i := range cmd.savedUnits {
		unitState := &cmd.savedUnits[i]

		// Restore unit from captured state
		newUnitID, err := RestoreUnitToSquad(unitState, newSquadID, cmd.entityManager)
		if err != nil {
			return fmt.Errorf("failed to recreate unit %s: %w", unitState.Template.Name, err)
		}

		// Mark unit as in squad in roster
		if err := roster.MarkUnitInSquad(newUnitID, newSquadID); err != nil {
			// If marking fails, continue with other units
			continue
		}
	}

	// Update squad capacity
	squads.UpdateSquadCapacity(newSquadID, cmd.entityManager)

	// Update the command's squad ID to the new one (for potential re-undo)
	cmd.squadID = newSquadID

	return nil
}

// Description returns a human-readable description
func (cmd *DisbandSquadCommand) Description() string {
	if cmd.savedSquadData != nil {
		return fmt.Sprintf("Disband squad '%s'", cmd.savedSquadData.Name)
	}
	return "Disband squad"
}

// captureState saves all squad state before disbanding
func (cmd *DisbandSquadCommand) captureState() error {
	// Get squad entity
	squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
	if err != nil {
		return err
	}

	// Save squad data
	squadData, err := getSquadDataOrError(squadEntity)
	if err != nil {
		return err
	}

	cmd.savedSquadData = &squads.SquadData{
		SquadID:       squadData.SquadID,
		Formation:     squadData.Formation,
		Name:          squadData.Name,
		Morale:        squadData.Morale,
		SquadLevel:    squadData.SquadLevel,
		TurnCount:     squadData.TurnCount,
		MaxUnits:      squadData.MaxUnits,
		UsedCapacity:  squadData.UsedCapacity,
		TotalCapacity: squadData.TotalCapacity,
	}

	// Save squad position
	if squadEntity.HasComponent(common.PositionComponent) {
		pos := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if pos != nil {
			cmd.savedPosition = &coords.LogicalPosition{
				X: pos.X,
				Y: pos.Y,
			}
		}
	}

	// Capture all unit states using shared helper
	capturedUnits, err := CaptureAllUnitsInSquad(cmd.squadID, cmd.entityManager)
	if err != nil {
		return fmt.Errorf("failed to capture unit states: %w", err)
	}
	cmd.savedUnits = capturedUnits

	return nil
}
