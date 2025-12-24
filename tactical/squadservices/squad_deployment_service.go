package squadservices

import (
	"game_main/common"
	"game_main/coords"
	"game_main/tactical/squads"

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
