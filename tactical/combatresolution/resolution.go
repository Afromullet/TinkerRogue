package combatresolution

import (
	"fmt"
	"math/rand"
	"time"

	"game_main/common"
	"game_main/overworld/core"
	owencounter "game_main/overworld/overworldencounter"
	"game_main/overworld/threat"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// CombatOutcome describes result of a tactical battle
type CombatOutcome struct {
	ThreatNodeID   ecs.EntityID
	PlayerVictory  bool
	PlayerRetreat  bool
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID
	Casualties     CasualtyReport
	RewardsEarned  owencounter.RewardTable
}

// CasualtyReport tracks units lost in combat
type CasualtyReport struct {
	PlayerUnitsLost  int
	EnemyUnitsKilled int
}

// ResolveCombatToOverworld applies combat outcome to overworld state
// This is the feedback loop from tactical combat back to strategic layer
func ResolveCombatToOverworld(
	manager *common.EntityManager,
	outcome *CombatOutcome,
) error {
	// Find threat node
	threatEntity := manager.FindEntityByID(outcome.ThreatNodeID)
	if threatEntity == nil {
		return fmt.Errorf("threat node %d not found", outcome.ThreatNodeID)
	}

	threatData := common.GetComponentType[*core.ThreatNodeData](threatEntity, core.ThreatNodeComponent)
	if threatData == nil {
		return fmt.Errorf("entity is not a threat node")
	}

	// Get current tick for event logging
	tickState := core.GetTickState(manager)
	currentTick := int64(0)
	if tickState != nil {
		currentTick = tickState.CurrentTick
	}

	if outcome.PlayerVictory {
		// Player won - reduce or destroy threat
		damageDealt := CalculateThreatDamage(outcome.Casualties.EnemyUnitsKilled)
		oldIntensity := threatData.Intensity
		threatData.Intensity -= damageDealt

		if threatData.Intensity <= 0 {
			// Destroy threat node completely
			threat.DestroyThreatNode(manager, threatEntity)

			// Grant full rewards
			GrantRewards(manager, outcome, outcome.RewardsEarned)

			// Log combat resolution event
			core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
				fmt.Sprintf("Combat victory - Threat %d destroyed", outcome.ThreatNodeID),
				map[string]interface{}{
					"victory":            true,
					"intensity_reduced":  oldIntensity,
					"rewards_gold":       outcome.RewardsEarned.Gold,
					"rewards_xp":         outcome.RewardsEarned.Experience,
					"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
					"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
				})

			fmt.Printf("Threat %d destroyed! Rewards: %d gold, %d XP\n",
				outcome.ThreatNodeID, outcome.RewardsEarned.Gold, outcome.RewardsEarned.Experience)
		} else {
			// Weakened but not destroyed - partial rewards
			partialRewards := owencounter.RewardTable{
				Gold:       outcome.RewardsEarned.Gold / 2,
				Experience: outcome.RewardsEarned.Experience / 2,
			}
			GrantRewards(manager, outcome, partialRewards)

			// Reset growth progress (player setback the threat)
			threatData.GrowthProgress = 0.0

			// Log combat resolution event
			core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
				fmt.Sprintf("Combat victory - Threat %d weakened to intensity %d", outcome.ThreatNodeID, threatData.Intensity),
				map[string]interface{}{
					"victory":            true,
					"intensity_reduced":  damageDealt,
					"new_intensity":      threatData.Intensity,
					"rewards_gold":       partialRewards.Gold,
					"rewards_xp":         partialRewards.Experience,
					"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
					"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
				})

			fmt.Printf("Threat %d weakened to intensity %d. Partial rewards: %d gold, %d XP\n",
				outcome.ThreatNodeID, threatData.Intensity, partialRewards.Gold, partialRewards.Experience)
		}
	} else if outcome.PlayerRetreat {
		// Player fled - no change to threat, no rewards
		// Log combat resolution event
		core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
			fmt.Sprintf("Retreated from threat %d", outcome.ThreatNodeID),
			map[string]interface{}{
				"victory":            false,
				"retreat":            true,
				"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
				"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
			})

		fmt.Printf("Retreated from threat %d (no changes)\n", outcome.ThreatNodeID)
	} else {
		// Player defeat - threat grows stronger
		oldIntensity := threatData.Intensity
		threatData.Intensity += 1
		threatData.GrowthProgress = 0.0

		// Update influence radius
		influenceData := common.GetComponentType[*core.InfluenceData](threatEntity, core.InfluenceComponent)
		if influenceData != nil {
			params := core.GetThreatTypeParamsFromConfig(threatData.ThreatType)
			influenceData.Radius = params.BaseRadius + threatData.Intensity
			influenceData.EffectStrength = float64(threatData.Intensity) * 0.1
		}

		// Log combat resolution event
		core.LogEvent(core.EventCombatResolved, currentTick, outcome.ThreatNodeID,
			fmt.Sprintf("Combat defeat - Threat %d grew to intensity %d", outcome.ThreatNodeID, threatData.Intensity),
			map[string]interface{}{
				"victory":            false,
				"old_intensity":      oldIntensity,
				"new_intensity":      threatData.Intensity,
				"player_units_lost":  outcome.Casualties.PlayerUnitsLost,
				"enemy_units_killed": outcome.Casualties.EnemyUnitsKilled,
			})

		fmt.Printf("Defeated by threat %d! Threat grew to intensity %d\n",
			outcome.ThreatNodeID, threatData.Intensity)

		// TODO: Player suffers additional penalties (resource loss, morale, etc.)
	}

	return nil
}

// CalculateThreatDamage converts enemy casualties to threat intensity damage
// Every 5 enemies killed = 1 intensity reduction
func CalculateThreatDamage(enemiesKilled int) int {
	return enemiesKilled / 5
}

// GrantRewards distributes rewards to all surviving units across all player squads
func GrantRewards(manager *common.EntityManager, outcome *CombatOutcome, rewards owencounter.RewardTable) {
	if len(outcome.PlayerSquadIDs) == 0 {
		return
	}

	// Grant gold to the player
	if rewards.Gold > 0 && outcome.PlayerEntityID != 0 {
		resources := common.GetPlayerResources(outcome.PlayerEntityID, manager)
		if resources != nil {
			resources.AddGold(rewards.Gold)
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

// CreateCombatOutcome creates outcome from combat state
// Helper function to construct outcome from combat results
func CreateCombatOutcome(
	threatNodeID ecs.EntityID,
	playerWon bool,
	playerRetreated bool,
	playerEntityID ecs.EntityID,
	playerSquadIDs []ecs.EntityID,
	playerUnitsLost int,
	enemyUnitsKilled int,
	rewards owencounter.RewardTable,
) *CombatOutcome {
	return &CombatOutcome{
		ThreatNodeID:   threatNodeID,
		PlayerVictory:  playerWon,
		PlayerRetreat:  playerRetreated,
		PlayerEntityID: playerEntityID,
		PlayerSquadIDs: playerSquadIDs,
		Casualties: CasualtyReport{
			PlayerUnitsLost:  playerUnitsLost,
			EnemyUnitsKilled: enemyUnitsKilled,
		},
		RewardsEarned: rewards,
	}
}
