package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

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
	threatData := common.GetComponentType[*core.ThreatNodeData](threatEntity, core.ThreatNodeComponent)
	if threatData == nil {
		return nil, fmt.Errorf("entity is not a threat node")
	}

	// Get the encounter that was assigned to this node when it was created
	selectedEncounter := core.GetNodeRegistry().GetEncounterByID(threatData.EncounterID)
	if selectedEncounter == nil {
		return nil, fmt.Errorf("encounter %s not found for threat node", threatData.EncounterID)
	}

	// Create encounter name using the selected encounter's name
	encounterName := fmt.Sprintf("%s (Level %d)",
		getEncounterDisplayName(selectedEncounter, threatData.ThreatType),
		threatData.Intensity)

	return &encounterParams{
		ThreatNodeID:  threatData.ThreatID,
		Difficulty:    threatData.Intensity,
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
func getEncounterDisplayName(encounter *core.EncounterDefinition, threatType core.ThreatType) string {
	if encounter != nil && encounter.EncounterTypeName != "" {
		return encounter.EncounterTypeName
	}
	// Fallback to threat type display name
	return threatType.String()
}

// TriggerCombatFromThreat initiates combat when player engages a threat
// This function bridges the overworld -> combat transition
// Returns the created encounter ID
func TriggerCombatFromThreat(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
	playerPos coords.LogicalPosition,
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
