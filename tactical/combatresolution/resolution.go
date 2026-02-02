package combatresolution

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	owencounter "game_main/overworld/overworldencounter"
	"game_main/overworld/threat"

	"github.com/bytearena/ecs"
)

// CombatOutcome describes result of a tactical battle
type CombatOutcome struct {
	ThreatNodeID  ecs.EntityID
	PlayerVictory bool
	PlayerRetreat bool
	PlayerSquadID ecs.EntityID
	Casualties    CasualtyReport
	RewardsEarned owencounter.RewardTable
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
			GrantRewards(manager, outcome.PlayerSquadID, outcome.RewardsEarned)

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
			GrantRewards(manager, outcome.PlayerSquadID, partialRewards)

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
			params := core.GetThreatTypeParams(threatData.ThreatType)
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

// GrantRewards applies rewards to player
func GrantRewards(manager *common.EntityManager, squadID ecs.EntityID, rewards owencounter.RewardTable) {

	//todo
}

// CreateCombatOutcome creates outcome from combat state
// Helper function to construct outcome from combat results
func CreateCombatOutcome(
	threatNodeID ecs.EntityID,
	playerWon bool,
	playerRetreated bool,
	playerSquadID ecs.EntityID,
	playerUnitsLost int,
	enemyUnitsKilled int,
	rewards owencounter.RewardTable,
) *CombatOutcome {
	return &CombatOutcome{
		ThreatNodeID:  threatNodeID,
		PlayerVictory: playerWon,
		PlayerRetreat: playerRetreated,
		PlayerSquadID: playerSquadID,
		Casualties: CasualtyReport{
			PlayerUnitsLost:  playerUnitsLost,
			EnemyUnitsKilled: enemyUnitsKilled,
		},
		RewardsEarned: rewards,
	}
}
