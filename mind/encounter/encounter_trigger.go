package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"

	"github.com/bytearena/ecs"
)

// encounterParams describes a combat scenario generated from a threat node
type encounterParams struct {
	ThreatNodeID  ecs.EntityID
	Difficulty    int // Derived from threat intensity
	EncounterName string
	EncounterType string
}

// translateThreatToEncounter generates combat parameters from a threat node.
// This creates the encounter metadata (name, difficulty, type).
// Enemy composition is generated later by SetupBalancedEncounter() using
// power-based balancing to match player strength.
func translateThreatToEncounter(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
) (*encounterParams, error) {
	nodeData := common.GetComponentType[*core.OverworldNodeData](threatEntity, core.OverworldNodeComponent)
	if nodeData == nil {
		return nil, fmt.Errorf("entity is not an overworld node")
	}

	// Get the encounter that was assigned to this node when it was created
	selectedEncounter := core.GetNodeRegistry().GetEncounterByID(nodeData.EncounterID)
	if selectedEncounter == nil {
		return nil, fmt.Errorf("encounter %s not found for node", nodeData.EncounterID)
	}

	// Create encounter name using the selected encounter's name
	encounterName := fmt.Sprintf("%s (Level %d)",
		getEncounterDisplayName(selectedEncounter, nodeData.NodeTypeID),
		nodeData.Intensity)

	return &encounterParams{
		ThreatNodeID:  nodeData.NodeID,
		Difficulty:    nodeData.Intensity,
		EncounterName: encounterName,
		EncounterType: selectedEncounter.EncounterTypeID,
	}, nil
}

// createOverworldEncounter creates an encounter entity from threat parameters
func createOverworldEncounter(
	manager *common.EntityManager,
	params *encounterParams,
) (ecs.EntityID, error) {
	entity := manager.World.NewEntity()

	encounterData := &core.OverworldEncounterData{
		Name:          params.EncounterName,
		Level:         params.Difficulty,
		EncounterType: params.EncounterType,
		IsDefeated:    false,
		ThreatNodeID:  params.ThreatNodeID, // Store threat link for resolution
	}

	entity.AddComponent(core.OverworldEncounterComponent, encounterData)

	return entity.GetID(), nil
}

// getEncounterDisplayName returns the display name for an encounter.
// Falls back to threat type name if encounter is nil.
func getEncounterDisplayName(encounter *core.EncounterDefinition, nodeTypeID string) string {
	if encounter != nil && encounter.EncounterTypeName != "" {
		return encounter.EncounterTypeName
	}
	// Fallback to node display name from registry
	nodeDef := core.GetNodeRegistry().GetNodeByID(nodeTypeID)
	if nodeDef != nil {
		return nodeDef.DisplayName
	}
	return nodeTypeID
}

// TriggerCombatFromThreat initiates combat when player engages a threat
// This function bridges the overworld -> combat transition
// Returns the created encounter ID
func TriggerCombatFromThreat(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
) (ecs.EntityID, error) {
	// 1. Translate threat to encounter
	params, err := translateThreatToEncounter(manager, threatEntity)
	if err != nil {
		return 0, fmt.Errorf("failed to translate threat: %w", err)
	}

	// 2. Create encounter entity
	encounterID, err := createOverworldEncounter(manager, params)
	if err != nil {
		return 0, fmt.Errorf("failed to create encounter: %w", err)
	}

	fmt.Printf("Combat triggered from threat %d: %s (Encounter ID: %d)\n",
		params.ThreatNodeID, params.EncounterName, encounterID)

	// 3. Combat system will call SetupBalancedEncounter with the encounter data
	// This happens in the combat mode transition (handled by GUI/mode coordinator)
	// The encounter ID is stored and passed to combat lifecycle for resolution

	return encounterID, nil
}

// TriggerGarrisonDefense creates an encounter entity for a garrison defense scenario.
// The attacking faction generates enemies via power budget. The garrison squads defend.
func TriggerGarrisonDefense(
	manager *common.EntityManager,
	targetNodeID ecs.EntityID,
	attackingFactionType core.FactionType,
	attackingStrength int,
) (ecs.EntityID, error) {
	entity := manager.World.NewEntity()

	encounterData := &core.OverworldEncounterData{
		Name:                 fmt.Sprintf("%s Raid on Garrison", attackingFactionType.String()),
		Level:                1 + (attackingStrength / 20),
		EncounterType:        string(core.MapFactionToThreatType(attackingFactionType)),
		IsDefeated:           false,
		ThreatNodeID:         targetNodeID,
		IsGarrisonDefense:    true,
		AttackingFactionType: attackingFactionType,
	}

	entity.AddComponent(core.OverworldEncounterComponent, encounterData)

	encounterID := entity.GetID()
	fmt.Printf("Garrison defense encounter created: ID %d, node %d, attacker %s\n",
		encounterID, targetNodeID, attackingFactionType.String())

	return encounterID, nil
}
