package roster

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// SquadRosterComponent marks the SquadRoster component
var SquadRosterComponent *ecs.Component

// SquadRoster tracks squads owned by the player
type SquadRoster struct {
	OwnedSquads []ecs.EntityID // All squads owned by player
	MaxSquads   int            // Maximum squads player can own
}

// NewSquadRoster creates a new squad roster with a maximum capacity
func NewSquadRoster(maxSquads int) *SquadRoster {
	return &SquadRoster{
		OwnedSquads: make([]ecs.EntityID, 0),
		MaxSquads:   maxSquads,
	}
}

// CanAddSquad checks if roster has space for another squad
func (sr *SquadRoster) CanAddSquad() bool {
	return len(sr.OwnedSquads) < sr.MaxSquads
}

// AddSquad adds a squad to the roster
func (sr *SquadRoster) AddSquad(squadID ecs.EntityID) error {
	if !sr.CanAddSquad() {
		return fmt.Errorf("squad roster is full: %d/%d squads", len(sr.OwnedSquads), sr.MaxSquads)
	}

	for _, id := range sr.OwnedSquads {
		if id == squadID {
			return fmt.Errorf("squad %d already in roster", squadID)
		}
	}

	sr.OwnedSquads = append(sr.OwnedSquads, squadID)
	return nil
}

// RemoveSquad removes a squad from the roster by entity ID
func (sr *SquadRoster) RemoveSquad(squadID ecs.EntityID) bool {
	for i, id := range sr.OwnedSquads {
		if id == squadID {
			sr.OwnedSquads[i] = sr.OwnedSquads[len(sr.OwnedSquads)-1]
			sr.OwnedSquads = sr.OwnedSquads[:len(sr.OwnedSquads)-1]
			return true
		}
	}
	return false
}

// GetSquadCount returns current/max squad counts
func (sr *SquadRoster) GetSquadCount() (int, int) {
	return len(sr.OwnedSquads), sr.MaxSquads
}

// GetDeployedSquads returns IDs of all squads that are currently deployed
func (sr *SquadRoster) GetDeployedSquads(manager *common.EntityManager) []ecs.EntityID {
	return sr.filterSquadsByDeployment(manager, true)
}

// GetReserveSquads returns IDs of all squads that are in reserves (not deployed)
func (sr *SquadRoster) GetReserveSquads(manager *common.EntityManager) []ecs.EntityID {
	return sr.filterSquadsByDeployment(manager, false)
}

func (sr *SquadRoster) filterSquadsByDeployment(manager *common.EntityManager, deployed bool) []ecs.EntityID {
	result := make([]ecs.EntityID, 0)
	for _, squadID := range sr.OwnedSquads {
		squadData := common.GetComponentTypeByID[*squadcore.SquadData](manager, squadID, squadcore.SquadComponent)
		if squadData != nil && squadData.IsDeployed == deployed {
			result = append(result, squadID)
		}
	}
	return result
}

// GetPlayerSquadRoster retrieves player's squad roster from ECS
func GetPlayerSquadRoster(playerID ecs.EntityID, manager *common.EntityManager) *SquadRoster {
	return common.GetComponentTypeByID[*SquadRoster](manager, playerID, SquadRosterComponent)
}
