package encounter

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/threat"

	"github.com/bytearena/ecs"
)

// combatOutcome describes result of a tactical battle (victory or defeat only;
// retreat is handled separately by resolveFleeToOverworld)
type combatOutcome struct {
	ThreatNodeID   ecs.EntityID
	PlayerVictory  bool
	PlayerEntityID ecs.EntityID
	PlayerSquadIDs []ecs.EntityID
	Casualties     casualtyReport
	RewardsEarned  rewardTable
}

// casualtyReport tracks units lost in combat
type casualtyReport struct {
	PlayerUnitsLost  int
	EnemyUnitsKilled int
}

// applyCombatOutcome applies combat outcome to overworld state
// This is the feedback loop from tactical combat back to strategic layer
func applyCombatOutcome(
	manager *common.EntityManager,
	outcome *combatOutcome,
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
		damageDealt := calculateThreatDamage(outcome.Casualties.EnemyUnitsKilled)
		oldIntensity := threatData.Intensity
		threatData.Intensity -= damageDealt

		if threatData.Intensity <= 0 {
			// Destroy threat node completely
			threat.DestroyThreatNode(manager, threatEntity)

			// Grant full rewards
			grantRewards(manager, outcome, outcome.RewardsEarned)

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
			partialRewards := rewardTable{
				Gold:       outcome.RewardsEarned.Gold / 2,
				Experience: outcome.RewardsEarned.Experience / 2,
			}
			grantRewards(manager, outcome, partialRewards)

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
			influenceData.BaseMagnitude = core.CalculateBaseMagnitude(threatData.Intensity)
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

// calculateThreatDamage converts enemy casualties to threat intensity damage
// Every 5 enemies killed = 1 intensity reduction
// TODO, thi will require mroe thought
func calculateThreatDamage(enemiesKilled int) int {
	return enemiesKilled / 5
}

// createCombatOutcome creates outcome from combat state
// Helper function to construct outcome from combat results
func createCombatOutcome(
	threatNodeID ecs.EntityID,
	playerWon bool,
	playerEntityID ecs.EntityID,
	playerSquadIDs []ecs.EntityID,
	playerUnitsLost int,
	enemyUnitsKilled int,
	rewards rewardTable,
) *combatOutcome {
	return &combatOutcome{
		ThreatNodeID:   threatNodeID,
		PlayerVictory:  playerWon,
		PlayerEntityID: playerEntityID,
		PlayerSquadIDs: playerSquadIDs,
		Casualties: casualtyReport{
			PlayerUnitsLost:  playerUnitsLost,
			EnemyUnitsKilled: enemyUnitsKilled,
		},
		RewardsEarned: rewards,
	}
}
