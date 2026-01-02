package encounter

import (
	"game_main/common"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// CheckEncounterAtPosition returns encounter ID if one exists at position
func CheckEncounterAtPosition(
	manager *common.EntityManager,
	position coords.LogicalPosition,
) ecs.EntityID {
	// Use GlobalPositionSystem for O(1) lookup
	entityIDs := common.GlobalPositionSystem.GetAllEntityIDsAt(position)

	// Check if any entity is an encounter
	for _, entityID := range entityIDs {
		entity := manager.FindEntityByID(entityID)
		if entity == nil {
			continue
		}

		// Check if this entity has encounter component
		if entity.HasComponent(OverworldEncounterComponent) {
			encounterData := common.GetComponentType[*OverworldEncounterData](
				entity,
				OverworldEncounterComponent,
			)

			// Only trigger if not defeated
			if encounterData != nil && !encounterData.IsDefeated {
				return entityID
			}
		}
	}

	return 0 // No encounter found
}
