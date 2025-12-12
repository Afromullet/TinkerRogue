package combat

import (
	"fmt"
	"game_main/common"
	"game_main/coords"
	"game_main/squads"

	"github.com/bytearena/ecs"
)

// findTurnStateEntity finds the single TurnStateData entity
func findTurnStateEntity(manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(TurnStateTag) {
		return result.Entity // Only one should exist
	}
	return nil
}

// GetSquadFaction returns the faction ID for a squad in combat.
// Returns 0 if squad is not in combat (doesn't have CombatFactionComponent).
func GetSquadFaction(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	combatFaction := common.GetComponentTypeByIDWithTag[*CombatFactionData](
		manager, squadID, squads.SquadTag, CombatFactionComponent)
	if combatFaction == nil {
		return 0
	}

	return combatFaction.FactionID
}

// findActionStateEntity finds ActionStateData for a squad (internal version)
// DEPRECATED: Use CombatQueryCache.FindActionStateEntity for better performance (50-200x faster)
// This version scans all entities and should only be used during initialization
func FindActionStateEntity(squadID ecs.EntityID, manager *common.EntityManager) *ecs.Entity {
	for _, result := range manager.World.Query(ActionStateTag) {
		actionState := common.GetComponentType[*ActionStateData](result.Entity, ActionStateComponent)
		if actionState.SquadID == squadID {
			return result.Entity
		}
	}
	return nil
}

// getFactionOwner returns the faction that owns a squad
func getFactionOwner(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	return GetSquadFaction(squadID, manager)
}

// getSquadMapPosition returns the current map position of a squad
func getSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
	pos := common.GetComponentTypeByID[*coords.LogicalPosition](manager, squadID, common.PositionComponent)
	if pos == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d has no position", squadID)
	}
	return *pos, nil
}

// GetSquadsForFaction returns all squads owned by a faction
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	var squadIDs []ecs.EntityID

	// Query all squads and filter by CombatFactionComponent
	for _, result := range manager.World.Query(squads.SquadTag) {
		combatFaction := common.GetComponentType[*CombatFactionData](result.Entity, CombatFactionComponent)
		if combatFaction != nil && combatFaction.FactionID == factionID {
			squadIDs = append(squadIDs, result.Entity.GetID())
		}
	}

	return squadIDs
}

// GetSquadAtPosition returns the squad entity ID at the given position
// Returns 0 if no squad at position or squad is destroyed
func GetSquadAtPosition(pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID {
	// Query all squads and filter by combat squads only
	for _, result := range manager.World.Query(squads.SquadTag) {
		// Only consider squads in combat
		combatFaction := common.GetComponentType[*CombatFactionData](result.Entity, CombatFactionComponent)
		if combatFaction == nil {
			continue
		}

		squadPos := common.GetComponentType[*coords.LogicalPosition](result.Entity, common.PositionComponent)
		if squadPos != nil && squadPos.X == pos.X && squadPos.Y == pos.Y {
			squadID := result.Entity.GetID()
			if !squads.IsSquadDestroyed(squadID, manager) {
				return squadID
			}
		}
	}
	return 0
}

// isSquad checks if an entity ID represents a squad in combat
func isSquad(entityID ecs.EntityID, manager *common.EntityManager) bool {
	// Check if squad has CombatFactionComponent (is in combat)
	combatFaction := common.GetComponentTypeByIDWithTag[*CombatFactionData](
		manager, entityID, squads.SquadTag, CombatFactionComponent)
	return combatFaction != nil
}

// ========================================
// ACTION STATE HELPERS
// ========================================

// canSquadAct checks if a squad can perform an action this turn (uses cached query for O(k) performance)
func canSquadAct(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) bool {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return false
	}
	return !actionState.HasActed
}

// canSquadMove checks if a squad can still move this turn (uses cached query for O(k) performance)
func canSquadMove(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) bool {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return false
	}
	return actionState.MovementRemaining > 0
}

// markSquadAsActed marks a squad as having used its combat action (uses cached query for O(k) performance)
func markSquadAsActed(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return
	}
	actionState.HasActed = true
}

// markSquadAsMoved marks a squad as having used movement (uses cached query for O(k) performance)
func markSquadAsMoved(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return
	}
	actionState.HasMoved = true
}

// decrementMovementRemaining reduces squad's remaining movement (uses cached query for O(k) performance)
func decrementMovementRemaining(cache *CombatQueryCache, squadID ecs.EntityID, amount int, manager *common.EntityManager) {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return
	}
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
	squad := common.FindEntityByIDWithTag(manager, squadID, squads.SquadTag)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Get position before removal
	position := common.GetComponentType[*coords.LogicalPosition](squad, common.PositionComponent)
	if position != nil {
		// Remove from PositionSystem spatial grid
		common.GlobalPositionSystem.RemoveEntity(squadID, *position)
	}

	// Remove CombatFactionComponent (squad exits combat)
	squad.RemoveComponent(CombatFactionComponent)

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
