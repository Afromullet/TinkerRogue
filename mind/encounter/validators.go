package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// ValidateEncounterEntity checks that an encounter entity exists and has OverworldEncounterData.
// Used by OverworldCombatStarter and GarrisonDefenseStarter.
func ValidateEncounterEntity(
	manager *common.EntityManager,
	encounterID ecs.EntityID,
) (*ecs.Entity, *core.OverworldEncounterData, error) {
	if encounterID == 0 {
		return nil, nil, fmt.Errorf("invalid encounter ID: 0")
	}

	entity := manager.FindEntityByID(encounterID)
	if entity == nil {
		return nil, nil, fmt.Errorf("encounter entity %d not found", encounterID)
	}

	data := common.GetComponentType[*core.OverworldEncounterData](entity, core.OverworldEncounterComponent)
	if data == nil {
		return nil, nil, fmt.Errorf("encounter %d missing OverworldEncounterData", encounterID)
	}

	return entity, data, nil
}
