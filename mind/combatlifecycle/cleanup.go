package combatlifecycle

import (
	"fmt"

	"game_main/common"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// ApplyHPRecovery restores a percentage of max HP to all living units in a squad.
func ApplyHPRecovery(manager *common.EntityManager, squadID ecs.EntityID, hpPercent int) {
	for _, unitID := range squadcore.GetUnitIDsInSquad(squadID, manager) {
		attr := common.GetComponentTypeByID[*common.Attributes](manager, unitID, common.AttributeComponent)
		if attr != nil && attr.CurrentHealth > 0 {
			heal := attr.GetMaxHealth() * hpPercent / 100
			attr.CurrentHealth += heal
			if attr.CurrentHealth > attr.GetMaxHealth() {
				attr.CurrentHealth = attr.GetMaxHealth()
			}
		}
	}
}

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
		if entity.HasComponent(combatcore.FactionMembershipComponent) {
			entity.RemoveComponent(combatcore.FactionMembershipComponent)
		}

		// Atomically remove squad position from both component and position system
		manager.UnregisterEntityPosition(entity)

		// Atomically remove all unit positions
		unitIDs := squadcore.GetUnitIDsInSquad(squadID, manager)
		for _, unitID := range unitIDs {
			unitEntity := manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			manager.UnregisterEntityPosition(unitEntity)
		}

		// Reset deployment flag
		squadData := common.GetComponentType[*squadcore.SquadData](entity, squadcore.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = false
		}

		fmt.Printf("Stripped combat components from squad %d\n", squadID)
	}
}
