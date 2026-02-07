package combat

import (
	"fmt"
	"game_main/common"

	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

type CombatMovementSystem struct {
	manager     *common.EntityManager
	posSystem   *common.PositionSystem // For O(1) collision detection
	combatCache *CombatQueryCache
}

// Constructor
func NewMovementSystem(manager *common.EntityManager, posSystem *common.PositionSystem, cache *CombatQueryCache) *CombatMovementSystem {
	return &CombatMovementSystem{
		manager:     manager,
		posSystem:   posSystem,
		combatCache: cache,
	}
}

// GetSquadMovementSpeed delegates to squads package for consistent speed calculation
// Squad moves at the speed of its slowest alive unit with MovementSpeedComponent
func (ms *CombatMovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {
	speed := squads.GetSquadMovementSpeed(squadID, ms.manager)
	// If squads returns 0 (no units or no movement components), use default
	if speed == 0 {
		return DefaultMovementSpeed
	}
	return speed
}

func (ms *CombatMovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
	// Check if tile is occupied using PositionSystem
	occupyingID := ms.posSystem.GetEntityIDAt(targetPos)
	if occupyingID == 0 {
		return true // Empty tile - can move
	}

	// Check if occupied by a squad (not terrain/item)
	if !isSquad(occupyingID, ms.manager) {
		return false // Occupied by terrain/obstacle
	}

	// Squads cannot occupy the same square as another squad, even friendlies
	return false
}

func (ms *CombatMovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {

	if !canSquadMove(ms.combatCache, squadID, ms.manager) {
		return fmt.Errorf("squad has no movement remaining")
	}

	currentPos, err := GetSquadMapPosition(squadID, ms.manager)
	if err != nil {
		return fmt.Errorf("cannot get current position: %w", err)
	}

	movementCost := currentPos.ChebyshevDistance(&targetPos)

	// Check if squad has enough movement (using cached query for performance)
	actionStateEntity := ms.combatCache.FindActionStateEntity(squadID, ms.manager)
	if actionStateEntity == nil {
		return fmt.Errorf("no action state for squad")
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	if actionState.MovementRemaining < movementCost {
		return fmt.Errorf("insufficient movement: need %d, have %d", movementCost, actionState.MovementRemaining)
	}

	if !ms.CanMoveTo(squadID, targetPos) {
		return fmt.Errorf("cannot move to %v", targetPos)
	}

	// Get squad entity for movement
	squadEntity := ms.manager.FindEntityByID(squadID)
	if squadEntity == nil {
		return fmt.Errorf("squad entity not found")
	}

	// Get unit IDs for atomic squad+members movement
	unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

	// Move squad AND members atomically (updates both PositionComponent and PositionSystem)
	err = ms.manager.MoveSquadAndMembers(squadID, squadEntity, unitIDs, currentPos, targetPos)
	if err != nil {
		return fmt.Errorf("failed to move squad and members: %w", err)
	}

	decrementMovementRemaining(ms.combatCache, squadID, movementCost, ms.manager)
	markSquadAsMoved(ms.combatCache, squadID, ms.manager)

	return nil
}

func (ms *CombatMovementSystem) GetValidMovementTiles(squadID ecs.EntityID) []coords.LogicalPosition {
	currentPos, err := GetSquadMapPosition(squadID, ms.manager)
	if err != nil {
		return []coords.LogicalPosition{}
	}

	// Get remaining movement (using cached query for performance)
	actionStateEntity := ms.combatCache.FindActionStateEntity(squadID, ms.manager)
	if actionStateEntity == nil {
		return []coords.LogicalPosition{}
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	movementRange := actionState.MovementRemaining

	if movementRange <= 0 {
		return []coords.LogicalPosition{}
	}

	// Simple flood-fill for valid tiles movement with Chebyshev distance
	validTiles := []coords.LogicalPosition{}

	for x := currentPos.X - movementRange; x <= currentPos.X+movementRange; x++ {
		for y := currentPos.Y - movementRange; y <= currentPos.Y+movementRange; y++ {
			testPos := coords.LogicalPosition{X: x, Y: y}

			// Check if within Chebyshev distance
			distance := currentPos.ChebyshevDistance(&testPos)
			if distance > movementRange {
				continue
			}

			// Check if can move to this tile
			if ms.CanMoveTo(squadID, testPos) {
				validTiles = append(validTiles, testPos)
			}
		}
	}

	return validTiles
}
