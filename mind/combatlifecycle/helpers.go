package combatlifecycle

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// ValidateEncounterEntity checks that an encounter entity exists and has OverworldEncounterData.
// Used by both OverworldCombatStarter and GarrisonDefenseStarter.
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

// ClampPowerTarget applies min/max bounds from the difficulty modifier to a raw power target.
// If raw is zero or negative, returns the minimum. If above max, returns the max.
func ClampPowerTarget(raw float64, mod templates.JSONEncounterDifficulty) float64 {
	if raw <= 0.0 {
		return mod.MinTargetPower
	}
	if raw > mod.MaxTargetPower {
		return mod.MaxTargetPower
	}
	return raw
}
