package overworldencounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

//TODO think of a better name for this package

// EncounterParams describes a combat scenario generated from a threat node
type EncounterParams struct {
	ThreatNodeID  ecs.EntityID
	Difficulty    int // Derived from threat intensity
	EncounterName string
	EncounterType string
	Rewards       RewardTable
}

// RewardTable defines rewards for defeating a threat
type RewardTable struct {
	Gold       int
	Experience int
	Items      []string // Future: item IDs
}

// TranslateThreatToEncounter generates combat parameters from a threat node.
// This creates the encounter metadata (name, difficulty, type, rewards).
// Enemy composition is generated later by SetupBalancedEncounter() using
// power-based balancing to match player strength.
func TranslateThreatToEncounter(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
) (*EncounterParams, error) {
	threatData := common.GetComponentType[*core.ThreatNodeData](threatEntity, core.ThreatNodeComponent)
	if threatData == nil {
		return nil, fmt.Errorf("entity is not a threat node")
	}

	// Get the encounter that was assigned to this node when it was created
	selectedEncounter := core.GetNodeRegistry().GetEncounterByID(threatData.EncounterID)
	if selectedEncounter == nil {
		return nil, fmt.Errorf("encounter %s not found for threat node", threatData.EncounterID)
	}

	// Calculate rewards based on this node's specific encounter
	rewards := CalculateRewards(threatData.Intensity, selectedEncounter)

	// Create encounter name using the selected encounter's name
	encounterName := fmt.Sprintf("%s (Level %d)",
		getEncounterDisplayName(selectedEncounter, threatData.ThreatType),
		threatData.Intensity)

	return &EncounterParams{
		ThreatNodeID:  threatData.ThreatID,
		Difficulty:    threatData.Intensity,
		EncounterName: encounterName,
		EncounterType: selectedEncounter.EncounterTypeID,
		Rewards:       rewards,
	}, nil
}

// CalculateRewards determines loot from defeating a threat.
// Reward multiplier is now derived from intensity instead of hardcoded per-type values.
// Formula: 1.0 + (intensity × 0.1) gives 1.1x-1.5x for intensity 1-5.
// Uses the selected encounter's drop table for items.
func CalculateRewards(intensity int, encounter *core.EncounterDefinition) RewardTable {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	// Intensity-based multiplier (replaces hardcoded type-specific values)
	// Higher intensity threats give proportionally better rewards
	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	// Generate item drops based on selected encounter and intensity
	items := GenerateItemDrops(intensity, encounter)

	return RewardTable{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		Items:      items,
	}
}

// GenerateItemDrops creates item rewards based on selected encounter and intensity
func GenerateItemDrops(intensity int, encounter *core.EncounterDefinition) []string {
	items := []string{}

	// Higher intensity threats drop more items
	numDrops := 0
	if intensity >= 5 {
		numDrops = 3 // High-tier threats (level 5) drop 3 items
	} else if intensity >= 4 {
		numDrops = 2 // Mid-tier threats (level 4) drop 2 items
	} else if intensity >= 2 {
		numDrops = 1 // Low-tier threats (level 2-3) drop 1 item
	}
	// Intensity 1 drops nothing (no guaranteed drops)

	// Random chance for bonus drop
	if common.RandomInt(100) < core.GetBonusItemDropChance() {
		numDrops++
	}

	// Generate items from the selected encounter's drop table
	for i := 0; i < numDrops; i++ {
		item := generateItemFromEncounter(encounter, intensity)
		if item != "" {
			items = append(items, item)
		}
	}

	return items
}

// HighTierIntensityThreshold is the minimum intensity for high-tier drops
const HighTierIntensityThreshold = 5

// generateItemFromEncounter returns an item name from the encounter's drop table.
// Uses the selected encounter's specific item pools.
func generateItemFromEncounter(encounter *core.EncounterDefinition, intensity int) string {
	if encounter == nil {
		return "Unknown Item"
	}

	basic := encounter.BasicItems
	highTier := encounter.HighTierItems

	if len(basic) == 0 {
		return "Unknown Item"
	}

	options := make([]string, len(basic))
	copy(options, basic)

	// High-tier drops available at max intensity (level 5)
	if intensity >= HighTierIntensityThreshold && len(highTier) > 0 {
		options = append(options, highTier...)
	}

	return options[common.RandomInt(len(options))]
}

// CreateOverworldEncounter creates an encounter entity from threat parameters
func CreateOverworldEncounter(
	manager *common.EntityManager,
	params *EncounterParams,
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

// SelectRandomEncounterForThreat randomly selects an encounter from the threat's faction pool.
// Returns the encounter type ID and the full encounter definition.
// Exported for use by encounter service for reward calculation.
func SelectRandomEncounterForThreat(threatType core.ThreatType) (string, *core.EncounterDefinition) {
	// Get the node definition to find the faction
	node := core.GetNodeRegistry().GetNodeByType(threatType)
	if node == nil || node.FactionID == "" {
		return threatType.EncounterTypeID(), nil
	}

	// Get all encounters for this faction
	encounters := core.GetNodeRegistry().GetEncountersByFaction(node.FactionID)
	if len(encounters) == 0 {
		return threatType.EncounterTypeID(), nil
	}

	// Randomly select one encounter from the faction's pool
	selectedEncounter := encounters[common.RandomInt(len(encounters))]
	return selectedEncounter.EncounterTypeID, selectedEncounter
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
// This function bridges the overworld → combat transition
// Returns the created encounter ID
func TriggerCombatFromThreat(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
	playerPos coords.LogicalPosition,
) (ecs.EntityID, error) {
	// 1. Translate threat to encounter
	encounterParams, err := TranslateThreatToEncounter(manager, threatEntity)
	if err != nil {
		return 0, fmt.Errorf("failed to translate threat: %w", err)
	}

	// 2. Create encounter entity
	encounterID, err := CreateOverworldEncounter(manager, encounterParams)
	if err != nil {
		return 0, fmt.Errorf("failed to create encounter: %w", err)
	}

	fmt.Printf("Combat triggered from threat %d: %s (Encounter ID: %d)\n",
		encounterParams.ThreatNodeID, encounterParams.EncounterName, encounterID)

	// 3. Combat system will call SetupBalancedEncounter with the encounter data
	// This happens in the combat mode transition (handled by GUI/mode coordinator)
	// The encounter ID is stored and passed to combat lifecycle for resolution

	return encounterID, nil
}
