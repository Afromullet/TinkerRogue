package combatcore

// Re-exports from combatstate for backward compatibility.

import (
	"game_main/common"
	"game_main/tactical/combat/combatstate"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

func IsCombatActive(manager *common.EntityManager) bool {
	return combatstate.IsCombatActive(manager)
}

func GetSquadFaction(squadID ecs.EntityID, manager *common.EntityManager) ecs.EntityID {
	return combatstate.GetSquadFaction(squadID, manager)
}

func GetSquadMapPosition(squadID ecs.EntityID, manager *common.EntityManager) (coords.LogicalPosition, error) {
	return combatstate.GetSquadMapPosition(squadID, manager)
}

func GetAllFactions(manager *common.EntityManager) []ecs.EntityID {
	return combatstate.GetAllFactions(manager)
}

func GetSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	return combatstate.GetSquadsForFaction(factionID, manager)
}

func GetActiveSquadsForFaction(factionID ecs.EntityID, manager *common.EntityManager) []ecs.EntityID {
	return combatstate.GetActiveSquadsForFaction(factionID, manager)
}

func GetSquadAtPosition(pos coords.LogicalPosition, manager *common.EntityManager) ecs.EntityID {
	return combatstate.GetSquadAtPosition(pos, manager)
}

func CreateActionStateForSquad(manager *common.EntityManager, squadID ecs.EntityID) {
	combatstate.CreateActionStateForSquad(manager, squadID)
}

func MarkSquadAsActed(cache *CombatQueryCache, squadID ecs.EntityID, manager *common.EntityManager) {
	combatstate.MarkSquadAsActed(cache, squadID, manager)
}

func RemoveSquadFromMap(squadID ecs.EntityID, manager *common.EntityManager) error {
	return combatstate.RemoveSquadFromMap(squadID, manager)
}

func NewCombatQueryCache(manager *common.EntityManager) *CombatQueryCache {
	return combatstate.NewCombatQueryCache(manager)
}

func NewCombatFactionManager(manager *common.EntityManager, cache *CombatQueryCache) *CombatFactionManager {
	return combatstate.NewCombatFactionManager(manager, cache)
}
