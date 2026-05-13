package combatlifecycle

import (
	"game_main/core/common"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// GetLivingUnitIDs returns the entity IDs of all living units across the given squads.
func GetLivingUnitIDs(manager *common.EntityManager, squadIDs []ecs.EntityID) []ecs.EntityID {
	var alive []ecs.EntityID
	squadcore.ForEachUnitWithAttrs(manager, squadIDs, func(unitID ecs.EntityID, attr *common.Attributes) bool {
		if attr.CurrentHealth > 0 {
			alive = append(alive, unitID)
		}
		return true
	})
	return alive
}

// CountDeadUnits counts how many units in the given squads have CurrentHealth <= 0.
func CountDeadUnits(manager *common.EntityManager, squadIDs []ecs.EntityID) int {
	dead := 0
	squadcore.ForEachUnitWithAttrs(manager, squadIDs, func(_ ecs.EntityID, attr *common.Attributes) bool {
		if attr.CurrentHealth <= 0 {
			dead++
		}
		return true
	})
	return dead
}
