package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// MoveStepCommand represents a single movement step (one turn)
// Used internally by MoveSquadCommand for multi-turn movement
// Does NOT have independent undo - parent command handles undo
type MoveStepCommand struct {
	entityManager  *common.EntityManager
	movementSystem *combat.CombatMovementSystem
	squadID        ecs.EntityID
	destination    coords.LogicalPosition
}

func NewMoveStepCommand(
	manager *common.EntityManager,
	moveSys *combat.CombatMovementSystem,
	squadID ecs.EntityID,
	destination coords.LogicalPosition,
) *MoveStepCommand {
	return &MoveStepCommand{
		entityManager:  manager,
		movementSystem: moveSys,
		squadID:        squadID,
		destination:    destination,
	}
}

func (cmd *MoveStepCommand) Validate() error {
	// Check if squad exists (same pattern as MoveSquadCommand)
	if err := validateSquadExists(cmd.squadID, cmd.entityManager); err != nil {
		return err
	}

	// Validate position is within reasonable bounds
	if cmd.destination.X < 0 || cmd.destination.Y < 0 {
		return fmt.Errorf("invalid position (%d, %d)", cmd.destination.X, cmd.destination.Y)
	}

	return nil
}

func (cmd *MoveStepCommand) Execute() error {
	// Delegate to CombatMovementSystem (REUSE EXISTING)
	return cmd.movementSystem.MoveSquad(cmd.squadID, cmd.destination)
}

func (cmd *MoveStepCommand) Undo() error {
	// No-op: Parent MoveSquadCommand handles undo
	// Step commands are just micro-operations within a larger command
	return nil
}

func (cmd *MoveStepCommand) Description() string {
	return fmt.Sprintf("Move step to (%d,%d)", cmd.destination.X, cmd.destination.Y)
}
