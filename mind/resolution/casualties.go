package resolution

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CasualtyReport tracks units lost on each side of combat.
type CasualtyReport struct {
	PlayerUnitsLost  int
	EnemyUnitsKilled int
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
