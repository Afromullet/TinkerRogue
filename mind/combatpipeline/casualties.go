package combatpipeline

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CountLivingUnitsInSquad returns the number of units in a squad with CurrentHealth > 0.
func CountLivingUnitsInSquad(manager *common.EntityManager, squadID ecs.EntityID) int {
	count := 0
	for _, unitID := range squads.GetUnitIDsInSquad(squadID, manager) {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth > 0 {
			count++
		}
	}
	return count
}

// GetLivingUnitIDs returns the entity IDs of all living units across the given squads.
func GetLivingUnitIDs(manager *common.EntityManager, squadIDs []ecs.EntityID) []ecs.EntityID {
	var alive []ecs.EntityID
	for _, squadID := range squadIDs {
		for _, unitID := range squads.GetUnitIDsInSquad(squadID, manager) {
			attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
			if attr != nil && attr.CurrentHealth > 0 {
				alive = append(alive, unitID)
			}
		}
	}
	return alive
}

// CountDeadUnits counts how many units in the given squads have CurrentHealth <= 0.
func CountDeadUnits(manager *common.EntityManager, squadIDs []ecs.EntityID) int {
	dead := 0
	for _, squadID := range squadIDs {
		for _, unitID := range squads.GetUnitIDsInSquad(squadID, manager) {
			entity := manager.FindEntityByID(unitID)
			if entity == nil {
				continue
			}
			attr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
			if attr != nil && attr.CurrentHealth <= 0 {
				dead++
			}
		}
	}
	return dead
}
