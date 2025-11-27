package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// FindFactionByID finds a faction entity by faction ID (public version)
func FindFactionByID(factionID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(FactionTag) {
		faction := result.Entity
		factionData := common.GetComponentType[*FactionData](faction, FactionComponent)
		if factionData.FactionID == factionID {
			return faction
		}
	}
	return nil
}

// FindFactionDataByID returns FactionData for a faction ID (public version)
// Returns nil if faction not found
func FindFactionDataByID(factionID ecs.EntityID, manager *common.EntityManager) *FactionData {
	entity := FindFactionByID(factionID, manager)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*FactionData](entity, FactionComponent)
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

// FindMapPositionBySquadID returns MapPositionData for a squad (public version)
// Returns nil if squad not found on map
func FindMapPositionBySquadID(squadID ecs.EntityID, manager *common.EntityManager) *MapPositionData {
	entity := findMapPositionEntity(squadID, manager)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*MapPositionData](entity, MapPositionComponent)
}

// FindMapPositionByFactionID returns all MapPositionData for squads in a faction
// Returns empty slice if no squads found for faction
func FindMapPositionByFactionID(factionID ecs.EntityID, manager *common.EntityManager) []*MapPositionData {
	result := make([]*MapPositionData, 0)
	for _, queryResult := range manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](queryResult.Entity, MapPositionComponent)
		if mapPos.FactionID == factionID {
			result = append(result, mapPos)
		}
	}
	return result
}

// findActionStateEntity finds ActionStateData for a squad
func FindActionStateEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(ActionStateTag) {
		actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
		if actionState.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// FindActionStateBySquadID returns ActionStateData for a squad (public version)
// Returns nil if squad's action state not found
func FindActionStateBySquadID(squadID ecs.EntityID, manager *common.EntityManager) *ActionStateData {
	entity := FindActionStateEntity(squadID, manager)
	if entity == nil {
		return nil
	}
	return common.GetComponentType[*ActionStateData](entity, ActionStateComponent)
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

	// Use consolidated query function
	mapPositions := FindMapPositionByFactionID(factionID, manager)
	for _, mapPos := range mapPositions {
		squadIDs = append(squadIDs, mapPos.SquadID)
	}

	return squadIDs
}

// GetSquadAtPosition returns the squad entity ID at the given position
// Returns 0 if no squad at position or squad is destroyed
func GetSquadAtPosition(pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID {
	for _, result := range manager.World.Query(MapPositionTag) {
		mapPos := common.GetComponentType[*MapPositionData](result.Entity, MapPositionComponent)

		if mapPos.Position.X == pos.X && mapPos.Position.Y == pos.Y {
			if !squads.IsSquadDestroyed(mapPos.SquadID, manager) {
				return mapPos.SquadID
			}
		}
	}
	return 0
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
	actionStateEntity := FindActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return false
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	return !actionState.HasActed
}

// canSquadMove checks if a squad can still move this turn
func canSquadMove(squadID ecs.EntityID, manager *common.EntityManager) bool {
	actionStateEntity := FindActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return false
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	return actionState.MovementRemaining > 0
}

// markSquadAsActed marks a squad as having used its combat action
func markSquadAsActed(squadID ecs.EntityID, manager *common.EntityManager) {
	actionStateEntity := FindActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	actionState.HasActed = true
}

// markSquadAsMoved marks a squad as having used movement
func markSquadAsMoved(squadID ecs.EntityID, manager *common.EntityManager) {
	actionStateEntity := FindActionStateEntity(squadID, manager)
	if actionStateEntity == nil {
		return
	}

	actionState := common.GetComponentType[*ActionStateData](actionStateEntity, ActionStateComponent)
	actionState.HasMoved = true
}

// decrementMovementRemaining reduces squad's remaining movement
func decrementMovementRemaining(squadID ecs.EntityID, amount int, manager *common.EntityManager) {
	actionStateEntity := FindActionStateEntity(squadID, manager)
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
		j := common.RandomInt(i + 1)
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
