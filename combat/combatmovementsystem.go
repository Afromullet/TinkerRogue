package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"
	"game_main/systems"

	"github.com/bytearena/ecs"
)

type CombatMovementSystem struct {
	manager     *common.EntityManager
	posSystem   *systems.PositionSystem // For O(1) collision detection
	combatCache *CombatQueryCache       
}

// Constructor
func NewMovementSystem(manager *common.EntityManager, posSystem *systems.PositionSystem) *CombatMovementSystem {
	return &CombatMovementSystem{
		manager:     manager,
		posSystem:   posSystem,
		combatCache: NewCombatQueryCache(manager),
	}
}

// The squad movement speed is the movement speed of the slowest unit in the squad
func (ms *CombatMovementSystem) GetSquadMovementSpeed(squadID ecs.EntityID) int {

	unitIDs := squads.GetUnitIDsInSquad(squadID, ms.manager)

	//TODO: This makes no sense. This needs better error handling. If there are no unit IDs, this should throw an error
	if len(unitIDs) == 0 {
		return 3
	}

	minSpeed := 999
	for _, unitID := range unitIDs {

		entity := common.FindEntityByID(ms.manager, unitID)
		if entity == nil {
			continue
		}

		attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if attr == nil {
			continue
		}

		speed := attr.GetMovementSpeed()

		if speed < minSpeed {
			minSpeed = speed
		}
	}

	//TODO: This makes no sense. This needs better error handling. If there are no unit IDs, this should throw an error
	if minSpeed == 999 {
		return 3 // Default if no valid units
	}

	return minSpeed // Squad moves at slowest unit's speed
}

func (ms *CombatMovementSystem) CanMoveTo(squadID ecs.EntityID, targetPos coords.LogicalPosition) bool {
	//Check if tile is occupied using PositionSystem
	occupyingID := ms.posSystem.GetEntityIDAt(targetPos)
	if occupyingID == 0 {
		return true // Empty tile - can move
	}

	//Check if occupied by a squad (not terrain/item)
	if !isSquad(occupyingID, ms.manager) {
		return false // Occupied by terrain/obstacle
	}

	//If occupied by squad, check if it's friendly
	occupyingFaction := getFactionOwner(occupyingID, ms.manager)
	squadFaction := getFactionOwner(squadID, ms.manager)

	// Can pass through friendlies, NOT enemies
	//TODO. It can pass through friendlies, but should not be able to occupy the same square
	return occupyingFaction == squadFaction
}

func (ms *CombatMovementSystem) MoveSquad(squadID ecs.EntityID, targetPos coords.LogicalPosition) error {

	if !canSquadMove(ms.combatCache, squadID, ms.manager) {
		return fmt.Errorf("squad has no movement remaining")
	}

	currentPos, err := ms.GetSquadPosition(squadID)
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
	squadEntity := common.FindEntityByIDWithTag(ms.manager, squadID, squads.SquadTag)
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
	currentPos, err := ms.GetSquadPosition(squadID)
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

// GetSquadPosition returns the position of a squad
func (ms *CombatMovementSystem) GetSquadPosition(squadID ecs.EntityID) (coords.LogicalPosition, error) {
	entity := common.FindEntityByID(ms.manager, squadID)
	if entity == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d not found", squadID)
	}

	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if pos == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d has no position component", squadID)
	}
	return *pos, nil
}
