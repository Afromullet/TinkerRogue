package squads

import (
	"fmt"
	"game_main/common"

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
// Returns error if roster is full
func (sr *SquadRoster) AddSquad(squadID ecs.EntityID) error {
	if !sr.CanAddSquad() {
		return fmt.Errorf("squad roster is full: %d/%d squads", len(sr.OwnedSquads), sr.MaxSquads)
	}

	// Check if squad already in roster
	for _, id := range sr.OwnedSquads {
		if id == squadID {
			return fmt.Errorf("squad %d already in roster", squadID)
		}
	}

	sr.OwnedSquads = append(sr.OwnedSquads, squadID)
	return nil
}

// RemoveSquad removes a squad from the roster by entity ID
// Returns true if squad was found and removed
func (sr *SquadRoster) RemoveSquad(squadID ecs.EntityID) bool {
	for i, id := range sr.OwnedSquads {
		if id == squadID {
			// Remove squad ID (swap with last and truncate)
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
	deployed := make([]ecs.EntityID, 0)
	for _, squadID := range sr.OwnedSquads {
		squadData := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
		if squadData != nil && squadData.IsDeployed {
			deployed = append(deployed, squadID)
		}
	}
	return deployed
}

// GetReserveSquads returns IDs of all squads that are in reserves (not deployed)
func (sr *SquadRoster) GetReserveSquads(manager *common.EntityManager) []ecs.EntityID {
	reserves := make([]ecs.EntityID, 0)
	for _, squadID := range sr.OwnedSquads {
		squadData := common.GetComponentTypeByID[*SquadData](manager, squadID, SquadComponent)
		if squadData != nil && !squadData.IsDeployed {
			reserves = append(reserves, squadID)
		}
	}
	return reserves
}

// GetPlayerSquadRoster retrieves player's squad roster from ECS
// Returns nil if player has no roster component
func GetPlayerSquadRoster(playerID ecs.EntityID, manager *common.EntityManager) *SquadRoster {
	return common.GetComponentTypeByID[*SquadRoster](manager, playerID, SquadRosterComponent)
}
