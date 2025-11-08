// Package gui provides UI and mode system for the game
package gui

import (
	"game_main/combat"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// FindAllSquads returns all squad entity IDs in the game.
// Uses efficient ECS query pattern with SquadComponent tag.
func FindAllSquads(ecsManager *common.EntityManager) []ecs.EntityID {
	allSquads := make([]ecs.EntityID, 0)

	// Iterate through all entities
	entityIDs := ecsManager.GetAllEntities()
	for _, entityID := range entityIDs {
		// Check if entity has SquadData component
		if ecsManager.HasComponent(entityID, squads.SquadComponent) {
			allSquads = append(allSquads, entityID)
		}
	}

	return allSquads
}

// GetSquadName returns the name of a squad by its ID.
// Returns "Unknown Squad" if squad not found.
// Uses ECS query for lookup.
func GetSquadName(ecsManager *common.EntityManager, squadID ecs.EntityID) string {
	for _, result := range ecsManager.World.Query(ecsManager.Tags["squad"]) {
		squadData := common.GetComponentType[*squads.SquadData](
			result.Entity, squads.SquadComponent)
		if squadData.SquadID == squadID {
			return squadData.Name
		}
	}
	return "Unknown Squad"
}

// GetSquadAtPosition returns the squad entity ID at the given position.
// Returns 0 if no squad at position or squad is destroyed.
// Uses MapPositionComponent to look up positions.
func GetSquadAtPosition(ecsManager *common.EntityManager, pos coords.LogicalPosition) ecs.EntityID {
	for _, result := range ecsManager.World.Query(ecsManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](
			result.Entity, combat.MapPositionComponent)

		if mapPos.Position.X == pos.X && mapPos.Position.Y == pos.Y {
			if !squads.IsSquadDestroyed(mapPos.SquadID, ecsManager) {
				return mapPos.SquadID
			}
		}
	}
	return 0
}

// FindSquadsByFaction returns all squad IDs belonging to a faction.
// Returns empty slice if no squads found for faction.
// Filters out destroyed squads.
func FindSquadsByFaction(ecsManager *common.EntityManager, factionID ecs.EntityID) []ecs.EntityID {
	result := make([]ecs.EntityID, 0)

	for _, queryResult := range ecsManager.World.Query(ecsManager.Tags["mapposition"]) {
		mapPos := common.GetComponentType[*combat.MapPositionData](
			queryResult.Entity, combat.MapPositionComponent)

		if mapPos.FactionID == factionID {
			if !squads.IsSquadDestroyed(mapPos.SquadID, ecsManager) {
				result = append(result, mapPos.SquadID)
			}
		}
	}

	return result
}
