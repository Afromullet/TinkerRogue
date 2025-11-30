package squadservices

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// SquadDeploymentService encapsulates all squad deployment game logic
type SquadDeploymentService struct {
	entityManager *common.EntityManager
}

// NewSquadDeploymentService creates a new squad deployment service
func NewSquadDeploymentService(manager *common.EntityManager) *SquadDeploymentService {
	return &SquadDeploymentService{
		entityManager: manager,
	}
}

// PlaceSquadResult contains information about squad placement
type PlaceSquadResult struct {
	Success   bool
	SquadName string
	Position  coords.LogicalPosition
	Error     string
}

// PlaceSquadAtPosition places a squad at a specific map position
func (sds *SquadDeploymentService) PlaceSquadAtPosition(
	squadID ecs.EntityID,
	newPos coords.LogicalPosition,
) *PlaceSquadResult {
	result := &PlaceSquadResult{
		Position: newPos,
	}

	// Find the squad entity
	squadEntity := common.FindEntityByIDWithTag(sds.entityManager, squadID, squads.SquadTag)
	if squadEntity == nil {
		result.Error = fmt.Sprintf("squad %d not found", squadID)
		return result
	}

	// Get squad data for name
	squadData := common.GetComponentType[*squads.SquadData](squadEntity, squads.SquadComponent)
	if squadData != nil {
		result.SquadName = squadData.Name
	}

	// Get current position
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		result.Error = "squad has no position component"
		return result
	}

	// Move entity atomically (updates both component and GlobalPositionSystem)
	oldPos := *posPtr
	err := sds.entityManager.MoveEntity(squadID, squadEntity, oldPos, newPos)
	if err != nil {
		result.Error = fmt.Sprintf("failed to move squad: %v", err)
		return result
	}

	result.Success = true
	return result
}

// ClearAllSquadsResult contains information about clearing positions
type ClearAllSquadsResult struct {
	Success       bool
	SquadsCleared int
	Error         string
}

// ClearAllSquadPositions resets all squad positions to (0, 0)
func (sds *SquadDeploymentService) ClearAllSquadPositions() *ClearAllSquadsResult {
	result := &ClearAllSquadsResult{}

	// Target position for clearing
	clearPos := coords.LogicalPosition{X: 0, Y: 0}

	// Query all squads
	squadsCleared := 0
	for _, queryResult := range sds.entityManager.World.Query(squads.SquadTag) {
		entity := queryResult.Entity
		squadID := entity.GetID()

		// Get current position
		posPtr := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if posPtr != nil {
			// Move entity atomically (updates both component and GlobalPositionSystem)
			oldPos := *posPtr
			err := sds.entityManager.MoveEntity(squadID, entity, oldPos, clearPos)
			if err == nil {
				squadsCleared++
			}
			// If error occurs, skip this squad and continue with others
		}
	}

	result.Success = true
	result.SquadsCleared = squadsCleared
	return result
}

// GetAllSquadPositions returns all squads with their current positions
func (sds *SquadDeploymentService) GetAllSquadPositions() map[ecs.EntityID]coords.LogicalPosition {
	positions := make(map[ecs.EntityID]coords.LogicalPosition)

	for _, queryResult := range sds.entityManager.World.Query(squads.SquadTag) {
		entity := queryResult.Entity

		posPtr := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if posPtr != nil {
			positions[entity.GetID()] = *posPtr
		}
	}

	return positions
}
