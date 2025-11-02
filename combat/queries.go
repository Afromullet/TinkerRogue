package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"
	"math/rand/v2"

	"github.com/bytearena/ecs"
)

func findEntityByID(entityID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	// Search all entities (O(n))
	for _, result := range manager.World.Query(ecs.BuildTag()) {
		if result.Entity.GetID() == entityID {
			return result.Entity
		}
	}
	return nil
}

// findFactionByID finds a faction entity by faction ID
func findFactionByID(factionID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(FactionTag) {
		faction := result.Entity
		factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
		if factionData.FactionID == factionID {
			return faction
		}
	}
	return nil
}

// findTurnStateEntity finds the single TurnStateData entity
func findTurnStateEntity(manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(TurnStateTag) {
		return result.Entity // Only one should exist
	}
	return nil
}

// findMapPositionEntity finds MapPositionData for a squad
func findMapPositionEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
		if mapPos.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// findActionStateEntity finds ActionStateData for a squad
func findActionStateEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(ActionStateTag) {
		actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
		if actionState.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// FindSquadByID finds a squad entity by its ID
// Note: This function doesn't exist in the codebase yet - add it to squads/squadqueries.go
// or use it from combat/queries.go
func FindSquadByID(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	// Import squads package: "game_main/squads"
	for _, result := range manager.World.Query(squads.SquadTag) {
		if result.Entity.GetID() == squadID {
			return result.Entity
		}
	}
	return nil
}

// getFactionOwner returns the faction that owns a squad
func getFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	mapPosEntity := findMapPositionEntity(squadID, manager)
	if mapPosEntity == nil {
		return 0
	}

	mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
	return mapPos.FactionID
}

// getSquadMapPosition returns the current map position of a squad
func getSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
	mapPosEntity := findMapPositionEntity(squadID, manager)
	if mapPosEntity == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d not on map", squadID)
	}

	mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
	return mapPos.Position, nil
}

// GetSquadsForFaction returns all squads owned by a faction
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	var squadIDs []ecs.EntityID

	for _, result := range manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)
		if mapPos.FactionID == factionID {
			squadIDs = append(squadIDs, mapPos.SquadID)
		}
	}

	return squadIDs
}

// isSquad checks if an entity ID represents a squad
func isSquad(entityID ecs.EntityID, manager *common.EntityManager) bool {
	mapPosEntity := findMapPositionEntity(entityID, manager)
	return mapPosEntity != nil
}

// ========================================
// ACTION STATE HELPERS
// ========================================

// canSquadAct checks if a squad can perform an action this turn
func canSquadAct(squadID ecs.EntityID, manager *common.EntityManager) bool {
	actionStateEntity := findActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return false
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	return !actionState.HasActed
}

// canSquadMove checks if a squad can still move this turn
func canSquadMove(squadID ecs.EntityID, manager *common.EntityManager) bool {
	actionStateEntity := findActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return false
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	return actionState.MovementRemaining > 0
}

// markSquadAsActed marks a squad as having used its combat action
func markSquadAsActed(squadID ecs.EntityID, manager *common.EntityManager) {
	actionStateEntity := findActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	actionState.HasActed = true
}

// markSquadAsMoved marks a squad as having used movement
func markSquadAsMoved(squadID ecs.EntityID, manager *common.EntityManager) {
	actionStateEntity := findActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	actionState.HasMoved = true
}

// decrementMovementRemaining reduces squad's remaining movement
func decrementMovementRemaining(squadID ecs.EntityID, amount int, manager *common.EntityManager) {
	actionStateEntity := findActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	actionState.MovementRemaining -= amount
	if actionState.MovementRemaining < 0 {
		actionState.MovementRemaining = 0
	}
}

// ========================================
// COMBAT STATE HELPERS
// ========================================

// removeSquadFromMap removes a squad from the combat map
func removeSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
	// Find and remove MapPositionData
	mapPosEntity := findMapPositionEntity(squadID, manager)
	if mapPosEntity == nil {
		return fmt.Errorf("squad %d not on map", squadID)
	}

	// Get position before removal
	mapPos := common.GetComponentType[*MapPositionData](mapPosEntity, MapPositionComponent)
	position := mapPos.Position

	// Remove from ECS
	manager.World.DisposeEntities(mapPosEntity)

	// Remove from PositionSystem spatial grid
	common.GlobalPositionSystem.RemoveEntity(squadID, position)

	return nil
}

// combatActive checks if combat is currently ongoing
func combatActive(manager *common.EntityManager) bool {
	turnStateEntity := findTurnStateEntity(manager)
	if turnStateEntity == nil {
		return false
	}

	turnState := common.GetComponentType[*TurnStateData](turnStateEntity, TurnStateComponent)
	return turnState.CombatActive
}

// ========================================
// UTILITY HELPERS
// ========================================

// shuffleFactionOrder randomizes faction turn order using Fisher-Yates
func shuffleFactionOrder(factionIDs []ecs.EntityID) {
	for i := len(factionIDs) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		factionIDs[i], factionIDs[j] = factionIDs[j], factionIDs[i]
	}
}

// logCombatResult logs combat result for debugging/UI
func logCombatResult(result *squads.CombatResult) {
	// TODO: Implement event system for UI
	fmt.Printf("Combat result: %d damage, %d kills\n", result.TotalDamage, len(result.UnitsKilled))
}

// contains checks if a slice contains a position
func contains(positions []coords.LogicalPosition, pos coords.LogicalPosition) bool {
	for _, p := range positions {
		if p.X == pos.X && p.Y == pos.Y {
			return true
		}
	}
	return false
}

// containsEntity checks if a slice contains an entity ID
func containsEntity(entities []ecs.EntityID, entityID ecs.EntityID) bool {
	for _, e := range entities {
		if e == entityID {
			return true
		}
	}
	return false
}
