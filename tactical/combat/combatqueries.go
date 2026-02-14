package combat

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// GetSquadFaction returns the faction ID for a squad in combat.
// Returns 0 if squad is not in combat (doesn't have FactionMembershipComponent).
func GetSquadFaction(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return 0
	}

	combatFaction := common.GetComponentType[*CombatFactionData](entity, FactionMembershipComponent)
	if combatFaction == nil {
		return 0
	}

	return combatFaction.FactionID
}

// GetSquadMapPosition returns the current map position of a squad
func GetSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
	entity := manager.FindEntityByID(squadID)
	if entity == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d not found", squadID)
	}

	pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
	if pos == nil {
		return coords.LogicalPosition{}, fmt.Errorf("squad %d has no position", squadID)
	}
	return *pos, nil
}

// GetAllFactions returns all factions in the combat system.
// Uses package-level ecs.View for zero-allocation reads.
func GetAllFactions(manager *common.EntityManager) []ecs.EntityID {
	var factionIDs []ecs.EntityID

	for _, result := range factionView.Get() {
		factionIDs = append(factionIDs, result.Entity.GetID())
	}

	return factionIDs
}

// GetSquadsForFaction returns all squads owned by a faction.
// Uses package-level ecs.View for zero-allocation reads.
func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	var squadIDs []ecs.EntityID

	for _, result := range combatSquadView.Get() {
		combatFaction := common.GetComponentType[*CombatFactionData](result.Entity, FactionMembershipComponent)
		if combatFaction != nil && combatFaction.FactionID == factionID {
			squadIDs = append(squadIDs, result.Entity.GetID())
		}
	}

	return squadIDs
}

// GetActiveSquadsForFaction returns all non-destroyed squads owned by a faction.
// This filters out destroyed squads, eliminating the need for callers to check IsSquadDestroyed.
func GetActiveSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	all := GetSquadsForFaction(factionID, manager)
	active := make([]ecs.EntityID, 0, len(all))
	for _, squadID := range all {
		if !squads.IsSquadDestroyed(squadID, manager) {
			active = append(active, squadID)
		}
	}
	return active
}

// GetSquadAtPosition returns the squad entity ID at the given position.
// Uses PositionSystem for O(1) lookup instead of iterating all squads.
// Returns 0 if no combat squad at position or squad is destroyed.
func GetSquadAtPosition(pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID {
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(pos)
	for _, entityID := range entityIDs {
		if !isSquad(entityID, manager) {
			continue
		}
		if !squads.IsSquadDestroyed(entityID, manager) {
			return entityID
		}
	}
	return 0
}

// isSquad checks if an entity ID represents a squad in combat
func isSquad(entityID ecs.EntityID, manager *common.EntityManager) bool {
	// Check if squad has FactionMembershipComponent (is in combat)
	// Use direct entity lookup instead of tag-based iteration (O(1) vs O(n))
	entity := manager.FindEntityByID(entityID)
	if entity == nil {
		return false
	}

	combatFaction := common.GetComponentType[*CombatFactionData](entity, FactionMembershipComponent)
	return combatFaction != nil
}

// ========================================
// ACTION STATE HELPERS
// ========================================

// CreateActionStateForSquad creates an ActionStateData entity for a squad.
// MovementRemaining is initialized from the squad's actual speed.
func CreateActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) {
	movementSpeed := squads.GetSquadMovementSpeed(squadID, manager)
	if movementSpeed == 0 {
		movementSpeed = DefaultMovementSpeed
	}

	actionEntity := manager.World.NewEntity()
	actionEntity.AddComponent(ActionStateComponent, &ActionStateData{
		SquadID:           squadID,
		HasMoved:          false,
		HasActed:          false,
		MovementRemaining: movementSpeed,
	})
}

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
	// Squad can move if it has movement points remaining
	return actionState.MovementRemaining > 0
}

// markSquadAsActed marks a squad as having used its combat action
func markSquadAsActed(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return
	}
	actionState.HasActed = true
}

// markSquadAsMoved marks a squad as having used movement
func markSquadAsMoved(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) {
	actionState := cache.FindActionStateBySquadID(squadID, manager)
	if actionState == nil {
		return
	}
	actionState.HasMoved = true
}

// decrementMovementRemaining reduces squad's remaining movement
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

// RemoveSquadFromMap removes a destroyed squad from the combat map and disposes all entities.
// This performs complete cleanup: removes from position system, disposes all units, and disposes the squad.
func RemoveSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
	squad := manager.FindEntityByID(squadID)
	if squad == nil {
		return fmt.Errorf("squad %d not found", squadID)
	}

	// Get position before removal (for position system cleanup)
	position := common.GetComponentType[*coords.LogicalPosition](squad, common.PositionComponent)
	if position != nil {
		// Remove from PositionSystem spatial grid
		common.GlobalPositionSystem.RemoveEntity(squadID, *position)
	}

	// Remove FactionMembershipComponent (squad exits combat) before disposal
	squad.RemoveComponent(FactionMembershipComponent)

	// Dispose the squad and all its units (dead or alive) from the ECS world
	squads.DisposeSquadAndUnits(squadID, manager)

	return nil
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
