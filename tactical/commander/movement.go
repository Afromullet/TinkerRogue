package commander

import (
	"fmt"
	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CommanderMovementSystem handles tile-by-tile commander movement on the overworld.
// Modeled after tactical/combat/combatmovementsystem.go.
type CommanderMovementSystem struct {
	manager   *common.EntityManager
	posSystem *common.PositionSystem
}

// NewCommanderMovementSystem creates a new commander movement system
func NewCommanderMovementSystem(manager *common.EntityManager, posSystem *common.PositionSystem) *CommanderMovementSystem {
	return &CommanderMovementSystem{
		manager:   manager,
		posSystem: posSystem,
	}
}

// MoveCommander moves a commander to the target position.
// Uses Chebyshev distance and decrements MovementRemaining.
func (cms *CommanderMovementSystem) MoveCommander(commanderID ecs.EntityID, targetPos coords.LogicalPosition) error {
	// Get action state
	actionState := GetCommanderActionState(commanderID, cms.manager)
	if actionState == nil {
		return fmt.Errorf("no action state for commander %d", commanderID)
	}

	if actionState.MovementRemaining <= 0 {
		return fmt.Errorf("commander has no movement remaining")
	}

	// Get current position
	commanderEntity := cms.manager.FindEntityByID(commanderID)
	if commanderEntity == nil {
		return fmt.Errorf("commander entity %d not found", commanderID)
	}

	currentPos := common.GetComponentType[*coords.LogicalPosition](commanderEntity, common.PositionComponent)
	if currentPos == nil {
		return fmt.Errorf("commander has no position")
	}

	// Calculate movement cost (Chebyshev distance)
	movementCost := currentPos.ChebyshevDistance(&targetPos)

	if actionState.MovementRemaining < movementCost {
		return fmt.Errorf("insufficient movement: need %d, have %d", movementCost, actionState.MovementRemaining)
	}

	if !cms.CanMoveTo(commanderID, targetPos) {
		return fmt.Errorf("cannot move to (%d,%d)", targetPos.X, targetPos.Y)
	}

	// Execute move
	if err := cms.manager.MoveEntity(commanderID, commanderEntity, *currentPos, targetPos); err != nil {
		return fmt.Errorf("failed to move commander: %w", err)
	}

	// Decrement movement
	actionState.MovementRemaining -= movementCost
	actionState.HasMoved = true

	return nil
}

// CanMoveTo checks if a commander can move to the target position.
func (cms *CommanderMovementSystem) CanMoveTo(commanderID ecs.EntityID, targetPos coords.LogicalPosition) bool {
	// Check map bounds and walkability
	if !core.IsTileWalkable(targetPos) {
		return false
	}

	// Check no other commander at tile
	occupyingCommander := GetCommanderAt(targetPos, cms.manager)
	if occupyingCommander != 0 && occupyingCommander != commanderID {
		return false
	}

	return true
}

// GetValidMovementTiles returns all tiles a commander can move to.
// Uses flood-fill within Chebyshev distance (same algorithm as combatmovementsystem.go).
func (cms *CommanderMovementSystem) GetValidMovementTiles(commanderID ecs.EntityID) []coords.LogicalPosition {
	commanderEntity := cms.manager.FindEntityByID(commanderID)
	if commanderEntity == nil {
		return nil
	}

	currentPos := common.GetComponentType[*coords.LogicalPosition](commanderEntity, common.PositionComponent)
	if currentPos == nil {
		return nil
	}

	actionState := GetCommanderActionState(commanderID, cms.manager)
	if actionState == nil {
		return nil
	}

	movementRange := actionState.MovementRemaining
	if movementRange <= 0 {
		return nil
	}

	validTiles := []coords.LogicalPosition{}

	for x := currentPos.X - movementRange; x <= currentPos.X+movementRange; x++ {
		for y := currentPos.Y - movementRange; y <= currentPos.Y+movementRange; y++ {
			testPos := coords.LogicalPosition{X: x, Y: y}

			// Check Chebyshev distance
			distance := currentPos.ChebyshevDistance(&testPos)
			if distance > movementRange || distance == 0 {
				continue
			}

			if cms.CanMoveTo(commanderID, testPos) {
				validTiles = append(validTiles, testPos)
			}
		}
	}

	return validTiles
}
