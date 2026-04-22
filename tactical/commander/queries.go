package commander

import (
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/squads/roster"

	"github.com/bytearena/ecs"
)

// GetCommanderAt returns the commander entity ID at the given position, or 0 if none.
// Uses PositionSystem for O(1) spatial lookup instead of iterating all commanders.
func GetCommanderAt(pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, id := range entityIDs {
		if manager.HasComponent(id, CommanderComponent) {
			return id
		}
	}
	return 0
}

// GetCommanderEntity finds commander entity by ID. Returns nil if not found or not a commander.
func GetCommanderEntity(commanderID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	entity := manager.FindEntityByID(commanderID)
	if entity != nil && entity.HasComponent(CommanderComponent) {
		return entity
	}
	return nil
}

// GetCommanderData retrieves CommanderData for a commander entity
func GetCommanderData(commanderID ecs.EntityID, manager *common.EntityManager) *CommanderData {
	return common.GetComponentTypeByID[*CommanderData](manager, commanderID, CommanderComponent)
}

// IsCommander checks if an entity is a commander
func IsCommander(entityID ecs.EntityID, manager *common.EntityManager) bool {
	return manager.HasComponent(entityID, CommanderComponent)
}

// GetAllCommanders returns all commander IDs from the player's roster
func GetAllCommanders(playerID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	roster := GetPlayerCommanderRoster(playerID, manager)
	if roster == nil {
		return nil
	}
	return roster.CommanderIDs
}

// FindCommanderForSquad returns the commander entity ID that owns the given squad,
// or 0 if no commander's roster contains it. Used to resolve "whose progression
// library applies to this squad" — since progression is per-commander and the
// squad→commander link lives implicitly inside each commander's SquadRoster.
func FindCommanderForSquad(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	for _, result := range manager.World.Query(CommanderTag) {
		commanderID := result.Entity.GetID()
		sr := common.GetComponentType[*roster.SquadRoster](result.Entity, roster.SquadRosterComponent)
		if sr == nil {
			continue
		}
		for _, id := range sr.OwnedSquads {
			if id == squadID {
				return commanderID
			}
		}
	}
	return 0
}
