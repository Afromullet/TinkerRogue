package overworld

import (
	"fmt"

	"game_main/common"
	"game_main/world/coords"
	"game_main/world/encounter"

	"github.com/bytearena/ecs"
)

// EncounterParams describes a combat scenario generated from a threat node
type EncounterParams struct {
	ThreatNodeID     ecs.EntityID
	EnemyComposition []UnitTemplate // Unit types and counts
	Difficulty       int            // Derived from threat intensity
	EncounterName    string
	EncounterType    string
	Rewards          RewardTable
}

// UnitTemplate describes a unit to spawn in combat
type UnitTemplate struct {
	Type string // Template name (e.g., "Skeleton", "Bandit")
	Role string // Role: "Tank", "DPS", "Support"
}

// RewardTable defines rewards for defeating a threat
type RewardTable struct {
	Gold       int
	Experience int
	Items      []string // Future: item IDs
}

// TranslateThreatToEncounter generates combat parameters from a threat node
// This is the bridge between overworld strategic layer and tactical combat
func TranslateThreatToEncounter(
	manager *common.EntityManager,
	threatEntity *ecs.Entity,
) (*EncounterParams, error) {
	threatData := common.GetComponentType[*ThreatNodeData](threatEntity, ThreatNodeComponent)
	if threatData == nil {
		return nil, fmt.Errorf("entity is not a threat node")
	}

	// Generate enemy composition based on threat type + intensity
	enemies := GenerateEnemyComposition(threatData.ThreatType, threatData.Intensity)

	// Calculate rewards
	rewards := CalculateRewards(threatData.Intensity, threatData.ThreatType)

	// Create encounter name
	threatName := getThreatTypeName(threatData.ThreatType)
	encounterName := fmt.Sprintf("%s (Level %d)", threatName, threatData.Intensity)

	return &EncounterParams{
		ThreatNodeID:     threatData.ThreatID,
		EnemyComposition: enemies,
		Difficulty:       threatData.Intensity,
		EncounterName:    encounterName,
		EncounterType:    getThreatEncounterType(threatData.ThreatType),
		Rewards:          rewards,
	}, nil
}

// GenerateEnemyComposition creates enemy units based on threat
func GenerateEnemyComposition(threatType ThreatType, intensity int) []UnitTemplate {
	// Base composition
	baseUnits := GetBaseThreatUnits(threatType)

	// Scale by intensity
	squadCount := 1 + (intensity / 3)        // 1 squad at tier 1-2, 2 squads at 3-5, etc.
	unitsPerSquad := 5 + intensity           // More units at higher intensity

	// Cap at reasonable limits
	if squadCount > 4 {
		squadCount = 4 // Max 4 enemy squads
	}
	if unitsPerSquad > 9 {
		unitsPerSquad = 9 // Max squad capacity
	}

	var units []UnitTemplate
	for i := 0; i < squadCount; i++ {
		for j := 0; j < unitsPerSquad; j++ {
			// Pick unit type from base composition (cycle through)
			unitType := baseUnits[j%len(baseUnits)]
			units = append(units, unitType)
		}
	}

	return units
}

// GetBaseThreatUnits returns base unit types per threat
func GetBaseThreatUnits(threatType ThreatType) []UnitTemplate {
	switch threatType {
	case ThreatNecromancer:
		return []UnitTemplate{
			{Type: "Skeleton", Role: "Tank"},
			{Type: "Zombie", Role: "DPS"},
			{Type: "Wraith", Role: "Support"},
		}
	case ThreatBanditCamp:
		return []UnitTemplate{
			{Type: "Bandit", Role: "DPS"},
			{Type: "Archer", Role: "DPS"},
			{Type: "Thug", Role: "Tank"},
		}
	case ThreatCorruption:
		return []UnitTemplate{
			{Type: "CorruptedBeast", Role: "Tank"},
			{Type: "CorruptedSpirit", Role: "DPS"},
		}
	case ThreatBeastNest:
		return []UnitTemplate{
			{Type: "Wolf", Role: "DPS"},
			{Type: "Bear", Role: "Tank"},
			{Type: "Boar", Role: "Tank"},
		}
	case ThreatOrcWarband:
		return []UnitTemplate{
			{Type: "OrcWarrior", Role: "Tank"},
			{Type: "OrcBerserker", Role: "DPS"},
			{Type: "OrcShaman", Role: "Support"},
		}
	default:
		return []UnitTemplate{{Type: "Generic", Role: "DPS"}}
	}
}

// CalculateRewards determines loot from defeating a threat
func CalculateRewards(intensity int, threatType ThreatType) RewardTable {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	// Type-specific bonuses
	typeMultiplier := 1.0
	switch threatType {
	case ThreatNecromancer:
		typeMultiplier = 1.5 // Higher rewards for harder threats
	case ThreatOrcWarband:
		typeMultiplier = 1.3
	case ThreatBanditCamp:
		typeMultiplier = 1.2
	case ThreatCorruption:
		typeMultiplier = 1.1
	case ThreatBeastNest:
		typeMultiplier = 1.0
	}

	// Generate item drops based on threat type and intensity
	items := GenerateItemDrops(intensity, threatType)

	return RewardTable{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		Items:      items,
	}
}

// GenerateItemDrops creates item rewards based on threat type and intensity
func GenerateItemDrops(intensity int, threatType ThreatType) []string {
	items := []string{}

	// Higher intensity threats drop more items
	numDrops := 0
	if intensity >= 8 {
		numDrops = 3 // High-tier threats drop 3 items
	} else if intensity >= 5 {
		numDrops = 2 // Mid-tier threats drop 2 items
	} else if intensity >= 3 {
		numDrops = 1 // Low-tier threats drop 1 item
	}
	// Intensity 1-2 drops nothing (no guaranteed drops)

	// Random chance for bonus drop
	if common.RandomInt(100) < 30 {
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

// generateItemByType returns an item name based on threat type
func generateItemByType(threatType ThreatType, intensity int) string {
	switch threatType {
	case ThreatNecromancer:
		// Necromancer drops: Dark items, scrolls, bones
		options := []string{
			"Dark Essence",
			"Necromantic Scroll",
			"Bone Fragment",
			"Soul Gem",
			"Cursed Tome",
		}
		if intensity >= 7 {
			options = append(options, "Lich Phylactery", "Staff of Undeath")
		}
		return options[common.RandomInt(len(options))]

	case ThreatBanditCamp:
		// Bandit drops: Weapons, gold, equipment
		options := []string{
			"Rusty Sword",
			"Leather Armor",
			"Iron Dagger",
			"Stolen Goods",
			"Lockpicks",
		}
		if intensity >= 7 {
			options = append(options, "Masterwork Blade", "Bandit King's Crown")
		}
		return options[common.RandomInt(len(options))]

	case ThreatOrcWarband:
		// Orc drops: Heavy weapons, crude armor
		options := []string{
			"Orcish Axe",
			"Crude Shield",
			"War Paint",
			"Tusk Trophy",
			"Bone Club",
		}
		if intensity >= 7 {
			options = append(options, "Warlord's Greataxe", "Berserker Totem")
		}
		return options[common.RandomInt(len(options))]

	case ThreatCorruption:
		// Corruption drops: Tainted items, essence
		options := []string{
			"Tainted Crystal",
			"Corrupted Seed",
			"Void Essence",
			"Shadow Fragment",
			"Blighted Herb",
		}
		if intensity >= 7 {
			options = append(options, "Heart of Corruption", "Void Shard")
		}
		return options[common.RandomInt(len(options))]

	case ThreatBeastNest:
		// Beast drops: Pelts, claws, natural materials
		options := []string{
			"Beast Pelt",
			"Sharp Claw",
			"Fang",
			"Beast Horn",
			"Hide Scraps",
		}
		if intensity >= 7 {
			options = append(options, "Alpha Pelt", "Primal Essence")
		}
		return options[common.RandomInt(len(options))]

	default:
		return "Unknown Item"
	}
}

// CreateOverworldEncounter creates an encounter entity from threat parameters
func CreateOverworldEncounter(
	manager *common.EntityManager,
	params *EncounterParams,
) (ecs.EntityID, error) {
	entity := manager.World.NewEntity()

	encounterData := &encounter.OverworldEncounterData{
		Name:          params.EncounterName,
		Level:         params.Difficulty,
		EncounterType: params.EncounterType,
		IsDefeated:    false,
		ThreatNodeID:  params.ThreatNodeID, // Store threat link for resolution
	}

	entity.AddComponent(encounter.OverworldEncounterComponent, encounterData)

	return entity.GetID(), nil
}

// getThreatEncounterType maps threat type to encounter type string
func getThreatEncounterType(threatType ThreatType) string {
	switch threatType {
	case ThreatNecromancer:
		return "undead"
	case ThreatBanditCamp:
		return "humanoid"
	case ThreatCorruption:
		return "corruption"
	case ThreatBeastNest:
		return "beast"
	case ThreatOrcWarband:
		return "orc"
	default:
		return "generic"
	}
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
