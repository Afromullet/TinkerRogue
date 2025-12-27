package squadcommands

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// MoveSquadCommand moves a squad to a new position in combat
// Captures old position and ActionState for undo
type MoveSquadCommand struct {
	entityManager  *common.EntityManager
	movementSystem *combat.CombatMovementSystem
	squadID        ecs.EntityID
	newPosition    coords.LogicalPosition

	// Captured state for undo
	oldPosition          coords.LogicalPosition
	squadName            string
	oldMovementRemaining int
	oldHasMoved          bool
}

// NewMoveSquadCommand creates a new move squad command
func NewMoveSquadCommand(
	manager *common.EntityManager,
	movementSystem *combat.CombatMovementSystem,
	squadID ecs.EntityID,
	newPosition coords.LogicalPosition,
) *MoveSquadCommand {
	return &MoveSquadCommand{
		entityManager:  manager,
		movementSystem: movementSystem,
		squadID:        squadID,
		newPosition:    newPosition,
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

	// Capture old ActionState (CRITICAL for undo)
	actionStateEntity := combat.FindActionStateEntity(cmd.squadID, cmd.entityManager)
	if actionStateEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
		if actionState != nil {
			cmd.oldMovementRemaining = actionState.MovementRemaining
			cmd.oldHasMoved = actionState.HasMoved
		}
	}

	// NEW: Check if multi-turn movement needed
	distance := cmd.oldPosition.ChebyshevDistance(&cmd.newPosition)

	if distance <= cmd.oldMovementRemaining {
		// Single-turn movement - execute immediately (EXISTING BEHAVIOR)
		err = cmd.movementSystem.MoveSquad(cmd.squadID, cmd.newPosition)
		if err != nil {
			return fmt.Errorf("movement system failed: %w", err)
		}
	} else {
		// Multi-turn movement - queue step commands
		stepCommands := createMoveStepCommands(
			cmd.entityManager,
			cmd.movementSystem,
			cmd.squadID,
			cmd.oldPosition,
			cmd.newPosition,
			cmd.oldMovementRemaining,
		)

		for _, stepCmd := range stepCommands {
			err := QueueCommand(cmd.entityManager, cmd.squadID, stepCmd)
			if err != nil {
				return fmt.Errorf("failed to queue move step: %w", err)
			}
		}
	}

	return nil
}

// Undo restores the squad to its old position
func (cmd *MoveSquadCommand) Undo() error {
	// NEW: Clear any remaining queued commands
	ClearCommandQueue(cmd.entityManager, cmd.squadID)

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
	// Use EntityManager directly (no validation for undo - we're restoring known-good state)
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

	// Restore ActionState (CRITICAL - undo must restore full state)
	actionStateEntity := combat.FindActionStateEntity(cmd.squadID, cmd.entityManager)
	if actionStateEntity != nil {
		actionState := common.GetComponentType[*combat.ActionStateData](actionStateEntity, combat.ActionStateComponent)
		if actionState != nil {
			actionState.MovementRemaining = cmd.oldMovementRemaining
			actionState.HasMoved = cmd.oldHasMoved
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

// Helper: Create step commands for multi-turn movement
func createMoveStepCommands(
	manager *common.EntityManager,
	moveSys *combat.CombatMovementSystem,
	squadID ecs.EntityID,
	start, end coords.LogicalPosition,
	movementPerTurn int,
) []SquadCommand {
	path := createStraightPath(start, end, movementPerTurn)

	stepCommands := make([]SquadCommand, len(path))
	for i, destination := range path {
		stepCommands[i] = NewMoveStepCommand(manager, moveSys, squadID, destination)
	}

	return stepCommands
}

func createStraightPath(from, to coords.LogicalPosition, maxStepSize int) []coords.LogicalPosition {
	path := []coords.LogicalPosition{}
	current := from

	for current != to {
		dx := to.X - current.X
		dy := to.Y - current.Y

		// Clamp to maxStepSize
		if abs(dx) > maxStepSize {
			if dx > 0 {
				dx = maxStepSize
			} else {
				dx = -maxStepSize
			}
		}

		if abs(dy) > maxStepSize {
			if dy > 0 {
				dy = maxStepSize
			} else {
				dy = -maxStepSize
			}
		}

		next := coords.LogicalPosition{X: current.X + dx, Y: current.Y + dy}
		path = append(path, next)
		current = next
	}

	return path
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
