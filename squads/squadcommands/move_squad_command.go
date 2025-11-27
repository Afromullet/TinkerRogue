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
	if cmd.squadID == 0 {
		return fmt.Errorf("invalid squad ID")
	}

	// Check if squad exists
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad does not exist")
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
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Get squad name for description
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData != nil {
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

	// Move squad to new position
	posPtr.X = cmd.newPosition.X
	posPtr.Y = cmd.newPosition.Y

	// Update position in global position system
	common.GlobalPositionSystem.RemoveEntity(cmd.squadID, cmd.oldPosition)
	common.GlobalPositionSystem.AddEntity(cmd.squadID, cmd.newPosition)

	// Update all unit positions to match squad position
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil && unitEntity.HasComponent(common.PositionComponent) {
			unitPosPtr := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if unitPosPtr != nil {
				unitPosPtr.X = cmd.newPosition.X
				unitPosPtr.Y = cmd.newPosition.Y
			}
		}
	}

	return nil
}

// Undo restores the squad to its old position
func (cmd *MoveSquadCommand) Undo() error {
	// Get squad entity
	squadEntity := squads.GetSquadEntity(cmd.squadID, cmd.entityManager)
	if squadEntity == nil {
		return fmt.Errorf("squad not found")
	}

	// Restore old position
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		return fmt.Errorf("squad has no position component")
	}

	// Move squad back to old position
	currentPos := *posPtr
	posPtr.X = cmd.oldPosition.X
	posPtr.Y = cmd.oldPosition.Y

	// Update position in global position system
	common.GlobalPositionSystem.RemoveEntity(cmd.squadID, currentPos)
	common.GlobalPositionSystem.AddEntity(cmd.squadID, cmd.oldPosition)

	// Update all unit positions to match squad position
	unitIDs := squads.GetUnitIDsInSquad(cmd.squadID, cmd.entityManager)
	for _, unitID := range unitIDs {
		unitEntity := common.FindEntityByIDWithTag(cmd.entityManager, unitID, squads.SquadMemberTag)
		if unitEntity != nil && unitEntity.HasComponent(common.PositionComponent) {
			unitPosPtr := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if unitPosPtr != nil {
				unitPosPtr.X = cmd.oldPosition.X
				unitPosPtr.Y = cmd.oldPosition.Y
			}
		}
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
