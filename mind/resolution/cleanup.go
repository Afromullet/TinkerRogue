package resolution

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// StripCombatComponents removes combat-related state from the given squads
// and their units: FactionMembership, Position, and resets IsDeployed.
// Callers decide WHICH squads to strip (by ID list).
func StripCombatComponents(manager *common.EntityManager, squadIDs []ecs.EntityID) {
	for _, squadID := range squadIDs {
		entity := manager.FindEntityByID(squadID)
		if entity == nil {
			continue
		}

		// Remove faction membership
		if entity.HasComponent(combat.FactionMembershipComponent) {
			entity.RemoveComponent(combat.FactionMembershipComponent)
		}

		// Remove squad position
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if pos != nil {
			common.GlobalPositionSystem.RemoveEntity(squadID, *pos)
			entity.RemoveComponent(common.PositionComponent)
		}

		// Remove all unit positions
		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		for _, unitID := range unitIDs {
			unitEntity := manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			unitPos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
			if unitPos != nil {
				common.GlobalPositionSystem.RemoveEntity(unitID, *unitPos)
				unitEntity.RemoveComponent(common.PositionComponent)
			}
		}

		// Reset deployment flag
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = false
		}

		fmt.Printf("Stripped combat components from squad %d\n", squadID)
	}
}
