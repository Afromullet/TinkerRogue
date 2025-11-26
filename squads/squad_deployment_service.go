package squads

import (
	"fmt"
	"game_main/common"
	"game_main/coords"

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
	Success    bool
	SquadName  string
	Position   coords.LogicalPosition
	Error      string
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
	squadEntity := common.FindEntityByIDWithTag(sds.entityManager, squadID, SquadTag)
	if squadEntity == nil {
		result.Error = fmt.Sprintf("squad %d not found", squadID)
		return result
	}

	// Get squad data for name
	squadData := common.GetComponentType[*SquadData](squadEntity, SquadComponent)
	if squadData != nil {
		result.SquadName = squadData.Name
	}

	// Update position
	posPtr := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
	if posPtr == nil {
		result.Error = "squad has no position component"
		return result
	}

	posPtr.X = newPos.X
	posPtr.Y = newPos.Y

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

	// Query all squads
	squadsCleared := 0
	for _, queryResult := range sds.entityManager.World.Query(SquadTag) {
		entity := queryResult.Entity

		// Get position component
		posPtr := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if posPtr != nil {
			posPtr.X = 0
			posPtr.Y = 0
			squadsCleared++
		}
	}

	result.Success = true
	result.SquadsCleared = squadsCleared
	return result
}

// GetAllSquadPositions returns all squads with their current positions
func (sds *SquadDeploymentService) GetAllSquadPositions() map[ecs.EntityID]coords.LogicalPosition {
	positions := make(map[ecs.EntityID]coords.LogicalPosition)

	for _, queryResult := range sds.entityManager.World.Query(SquadTag) {
		entity := queryResult.Entity

		posPtr := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if posPtr != nil {
			positions[entity.GetID()] = *posPtr
		}
	}

	return positions
}
