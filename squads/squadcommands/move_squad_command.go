package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// MoveSquadCommand moves a squad to a new position in combat
// Captures old position for undo
type MoveSquadCommand struct {
	entityManager *common.EntityManager
	squadID       ecs.EntityID
	newPosition   coords.LogicalPosition

	// Captured state for undo
	oldPosition coords.LogicalPosition
	squadName   string
}

// NewMoveSquadCommand creates a new move squad command
func NewMoveSquadCommand(
	manager *common.EntityManager,
	squadID ecs.EntityID,
	newPosition coords.LogicalPosition,
) *MoveSquadCommand {
	return &MoveSquadCommand{
		entityManager: manager,
		squadID:       squadID,
		newPosition:   newPosition,
	}
}

// Validate checks if the squad can be moved
func (cmd *MoveSquadCommand) Validate() error {
	// Check if squad exists
	if err := validateSquadExists(cmd.squadID, cmd.entityManager); err != nil {
		return err
	}

	// Validate position is within reasonable bounds (optional - could check map bounds)
	if cmd.newPosition.X < 0 || cmd.newPosition.Y < 0 {
		return fmt.Errorf("invalid position (%d, %d)", cmd.newPosition.X, cmd.newPosition.Y)
	}

	return nil
}

// Execute moves the squad to the new position
func (cmd *MoveSquadCommand) Execute() error {
	// Get squad entity
	squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
	if err != nil {
		return err
	}

	// Get squad name for description
	squadData, err := getSquadDataOrError(squadEntity)
	if err == nil {
		cmd.squadName = squadData.Name
	} else {
		cmd.squadName = "Unknown Squad"
	}

	// Capture old position
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		return fmt.Errorf("squad has no position component")
	}
	cmd.oldPosition = *posPtr

	// Move squad and all members atomically
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	if err := cmd.entityManager.MoveSquadAndMembers(
		cmd.squadID,
		squadEntity,
		unitIDs,
		cmd.oldPosition,
		cmd.newPosition,
	); err != nil {
		return fmt.Errorf("failed to move squad: %w", err)
	}

	return nil
}

// Undo restores the squad to its old position
func (cmd *MoveSquadCommand) Undo() error {
	// Get squad entity
	squadEntity, err := getSquadOrError(cmd.squadID, cmd.entityManager)
	if err != nil {
		return err
	}

	// Get current position before undo
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		return fmt.Errorf("squad has no position component")
	}
	currentPos := *posPtr

	// Move squad and all members back to old position atomically
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	if err := cmd.entityManager.MoveSquadAndMembers(
		cmd.squadID,
		squadEntity,
		unitIDs,
		currentPos,
		cmd.oldPosition,
	); err != nil {
		return fmt.Errorf("failed to undo squad move: %w", err)
	}

	return nil
}

// Description returns a human-readable description
func (cmd *MoveSquadCommand) Description() string {
	return fmt.Sprintf("Move %s from (%d, %d) to (%d, %d)",
		cmd.squadName,
		cmd.oldPosition.X, cmd.oldPosition.Y,
		cmd.newPosition.X, cmd.newPosition.Y)
}
