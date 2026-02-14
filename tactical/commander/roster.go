package commander

import (
	"fmt"
	"game_main/common"

	"github.com/bytearena/ecs"
)

// AddCommander adds a commander to the roster
func (r *CommanderRosterData) AddCommander(commanderID ecs.EntityID) error {
	if len(r.CommanderIDs) >= r.MaxCommanders {
		return fmt.Errorf("commander roster full: %d/%d", len(r.CommanderIDs), r.MaxCommanders)
	}

	for _, id := range r.CommanderIDs {
		if id == commanderID {
			return fmt.Errorf("commander %d already in roster", commanderID)
		}
	}

	r.CommanderIDs = append(r.CommanderIDs, commanderID)
	return nil
}

// RemoveCommander removes a commander from the roster
func (r *CommanderRosterData) RemoveCommander(commanderID ecs.EntityID) bool {
	for i, id := range r.CommanderIDs {
		if id == commanderID {
			r.CommanderIDs[i] = r.CommanderIDs[len(r.CommanderIDs)-1]
			r.CommanderIDs = r.CommanderIDs[:len(r.CommanderIDs)-1]
			return true
		}
	}
	return false
}

// GetCommanderCount returns current/max commander counts
func (r *CommanderRosterData) GetCommanderCount() (int, int) {
	return len(r.CommanderIDs), r.MaxCommanders
}

// GetPlayerCommanderRoster retrieves the player's commander roster from ECS
func GetPlayerCommanderRoster(playerID ecs.EntityID, manager *common.EntityManager) *CommanderRosterData {
	return common.GetComponentTypeByID[*CommanderRosterData](manager, playerID, CommanderRosterComponent)
}
