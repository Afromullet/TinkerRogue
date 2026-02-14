package encounter

import (
	"fmt"
	"math/rand"
	"time"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// rewardTable defines rewards for defeating a threat
type rewardTable struct {
	Gold       int
	Experience int
}

// calculateRewards determines loot from defeating a threat.
// Reward multiplier is now derived from intensity instead of hardcoded per-type values.
// Formula: 1.0 + (intensity x 0.1) gives 1.1x-1.5x for intensity 1-5.
func calculateRewards(intensity int, encounter *core.EncounterDefinition) rewardTable {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)

	// Intensity-based multiplier (replaces hardcoded type-specific values)
	// Higher intensity threats give proportionally better rewards
	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	return rewardTable{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
	}
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
