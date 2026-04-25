package encounter

import (
	"fmt"

	"game_main/campaign/overworld/core"
	"game_main/campaign/overworld/ids"
	"game_main/core/common"

	"github.com/bytearena/ecs"
)

// createEncounterEntity creates an encounter entity from OverworldEncounterData.
// This is the single place where encounter entities are born.
func createEncounterEntity(manager *common.EntityManager, data *core.OverworldEncounterData) ecs.EntityID {
	entity := manager.World.NewEntity()
	entity.AddComponent(core.OverworldEncounterComponent, data)
	return entity.GetID()
}

// getEncounterDisplayName returns the display name for an encounter.
// Falls back to threat type name if encounter is nil.
func getEncounterDisplayName(encounter *core.EncounterDefinition, nodeTypeID ids.NodeTypeID) string {
	if encounter != nil && encounter.EncounterTypeName != "" {
		return encounter.EncounterTypeName
	}
	// Fallback to node display name from registry
	nodeDef := core.GetNodeRegistry().GetNodeByID(nodeTypeID)
	if nodeDef != nil {
		return nodeDef.DisplayName
	}
	return string(nodeTypeID)
}

// TriggerRandomEncounter creates a debug encounter entity directly, bypassing threat-node lookup.
// ThreatNodeID 0 means EndEncounter skips overworld resolution (no side effects).
// Empty EncounterType falls back to generateRandomComposition.
func TriggerRandomEncounter(manager *common.EntityManager, difficulty int) (ecs.EntityID, error) {
	return createEncounterEntity(manager, &core.OverworldEncounterData{
		Name:          fmt.Sprintf("Random Encounter (Level %d)", difficulty),
		Level:         difficulty,
		EncounterType: "",
		IsDefeated:    false,
		ThreatNodeID:  0,
	}), nil
}

// TriggerCombatFromThreat initiates combat when player engages a threat.
// Bridges the overworld -> combat transition. Returns the created encounter ID.
func TriggerCombatFromThreat(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
) (ecs.EntityID, error) {
	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return 0, fmt.Errorf("entity is not an overworld node")
	}

	selectedEncounter := core.GetNodeRegistry().GetEncounterByID(nodeData.EncounterID)
	if selectedEncounter == nil {
		return 0, fmt.Errorf("encounter %s not found for node", nodeData.EncounterID)
	}

	return createEncounterEntity(manager, &core.OverworldEncounterData{
		Name: fmt.Sprintf("%s (Level %d)",
			getEncounterDisplayName(selectedEncounter, nodeData.NodeTypeID),
			nodeData.Intensity),
		Level:         nodeData.Intensity,
		EncounterType: selectedEncounter.EncounterTypeID,
		IsDefeated:    false,
		ThreatNodeID:  nodeData.NodeID,
	}), nil
}

// TriggerGarrisonDefense creates an encounter entity for a garrison defense scenario.
// The attacking faction generates enemies via power budget. The garrison squads defend.
func TriggerGarrisonDefense(
	manager *common.EntityManager,
	targetNodeID ecs.EntityID,
	attackingFactionType core.FactionType,
	attackingStrength int,
) (ecs.EntityID, error) {
	encounterData := &core.OverworldEncounterData{
		Name:                 fmt.Sprintf("%s Raid on Garrison", attackingFactionType.String()),
		Level:                1 + (attackingStrength / 20),
		EncounterType:        ids.EncounterTypeID(core.MapFactionToThreatType(attackingFactionType)),
		IsDefeated:           false,
		ThreatNodeID:         targetNodeID,
		IsGarrisonDefense:    true,
		AttackingFactionType: attackingFactionType,
	}

	return createEncounterEntity(manager, encounterData), nil
}
