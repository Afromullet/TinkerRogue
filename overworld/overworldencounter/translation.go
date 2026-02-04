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

	// Calculate rewards
	rewards := CalculateRewards(threatData.Intensity, threatData.ThreatType)

	// Create encounter name
	encounterName := fmt.Sprintf("%s (Level %d)", threatData.ThreatType.String(), threatData.Intensity)

	return &EncounterParams{
		ThreatNodeID:  threatData.ThreatID,
		Difficulty:    threatData.Intensity,
		EncounterName: encounterName,
		EncounterType: getThreatEncounterType(threatData.ThreatType),
		Rewards:       rewards,
	}, nil
}

// CalculateRewards determines loot from defeating a threat.
// Reward multiplier is now derived from intensity instead of hardcoded per-type values.
// Formula: 1.0 + (intensity × 0.1) gives 1.1x-1.5x for intensity 1-5.
func CalculateRewards(intensity int, threatType core.ThreatType) RewardTable {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	// Intensity-based multiplier (replaces hardcoded type-specific values)
	// Higher intensity threats give proportionally better rewards
	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	// Generate item drops based on threat type and intensity
	items := GenerateItemDrops(intensity, threatType)

	return RewardTable{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		Items:      items,
	}
}

// GenerateItemDrops creates item rewards based on threat type and intensity
func GenerateItemDrops(intensity int, threatType core.ThreatType) []string {
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

	// Generate items based on threat type
	for i := 0; i < numDrops; i++ {
		item := generateItemByType(threatType, intensity)
		if item != "" {
			items = append(items, item)
		}
	}

	return items
}

// HighTierIntensityThreshold is the minimum intensity for high-tier drops
const HighTierIntensityThreshold = 5

// generateItemByType returns an item name based on threat type.
// Uses ThreatRegistry for data-driven item drop tables.
func generateItemByType(threatType core.ThreatType, intensity int) string {
	basic, highTier := core.GetThreatRegistry().GetItemDropTable(threatType)

	if len(basic) == 0 {
		return "Unknown Item"
	}

	options := make([]string, len(basic))
	copy(options, basic)

	// High-tier drops available at max intensity (level 5)
	if intensity >= HighTierIntensityThreshold {
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

// getThreatEncounterType maps threat type to encounter type string.
// Returns the JSON encounter type ID that matches encounterdata.json.
func getThreatEncounterType(threatType core.ThreatType) string {
	return threatType.EncounterTypeID()
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
