package encounter

import (
	"fmt"
	"math/rand"
	"time"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// rewardTable defines rewards for defeating a threat
type rewardTable struct {
	Gold       int
	Experience int
	Items      []string // Future: item IDs
}

// highTierIntensityThreshold is the minimum intensity for high-tier drops
const highTierIntensityThreshold = 5

// calculateRewards determines loot from defeating a threat.
// Reward multiplier is now derived from intensity instead of hardcoded per-type values.
// Formula: 1.0 + (intensity x 0.1) gives 1.1x-1.5x for intensity 1-5.
// Uses the selected encounter's drop table for items.
func calculateRewards(intensity int, encounter *core.EncounterDefinition) rewardTable {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	// Intensity-based multiplier (replaces hardcoded type-specific values)
	// Higher intensity threats give proportionally better rewards
	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	// Generate item drops based on selected encounter and intensity
	items := generateItemDrops(intensity, encounter)

	return rewardTable{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		Items:      items,
	}
}

// generateItemDrops creates item rewards based on selected encounter and intensity
func generateItemDrops(intensity int, encounter *core.EncounterDefinition) []string {
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
	if common.RandomInt(100) < templates.OverworldConfigTemplate.SpawnProbabilities.BonusItemDropChance {
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
	if intensity >= highTierIntensityThreshold && len(highTier) > 0 {
		options = append(options, highTier...)
	}

	return options[common.RandomInt(len(options))]
}

// grantRewards distributes rewards to all surviving units across all player squads
func grantRewards(manager *common.EntityManager, outcome *combatOutcome, rewards rewardTable) {
	if len(outcome.PlayerSquadIDs) == 0 {
		return
	}

	// Grant gold to the player
	if rewards.Gold > 0 && outcome.PlayerEntityID != 0 {
		resources := common.GetResourceStockpile(outcome.PlayerEntityID, manager)
		if resources != nil {
			common.AddGold(resources, rewards.Gold)
			fmt.Printf("Granted %d gold to player %d\n", rewards.Gold, outcome.PlayerEntityID)
		}
	}

	// Distribute experience across all alive units in all squads
	if rewards.Experience > 0 {
		grantExperience(manager, outcome.PlayerSquadIDs, rewards.Experience)
	}
}

// grantExperience distributes XP evenly across all alive units in all squads
func grantExperience(manager *common.EntityManager, squadIDs []ecs.EntityID, totalXP int) {
	// Collect all alive unit IDs across all squads
	var aliveUnitIDs []ecs.EntityID
	for _, squadID := range squadIDs {
		unitIDs := squads.GetUnitIDsInSquad(squadID, manager)
		for _, unitID := range unitIDs {
			unitEntity := manager.FindEntityByID(unitID)
			if unitEntity == nil {
				continue
			}
			attr := common.GetComponentType[*common.Attributes](unitEntity, common.AttributeComponent)
			if attr != nil && attr.CurrentHealth > 0 {
				aliveUnitIDs = append(aliveUnitIDs, unitID)
			}
		}
	}

	if len(aliveUnitIDs) == 0 {
		return
	}

	xpPerUnit := totalXP / len(aliveUnitIDs)
	if xpPerUnit <= 0 {
		xpPerUnit = 1 // Minimum 1 XP per unit
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, unitID := range aliveUnitIDs {
		squads.AwardExperience(unitID, xpPerUnit, manager, rng)
	}

	fmt.Printf("Granted %d XP each to %d alive units (total %d XP)\n",
		xpPerUnit, len(aliveUnitIDs), totalXP)
}
