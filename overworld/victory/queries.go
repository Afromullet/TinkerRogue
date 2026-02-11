package victory

import (
	"fmt"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/threat"
	"game_main/templates"
)

// CheckPlayerDefeat checks if player has lost.
// Returns (isDefeated, defeatMessage). Empty message means not defeated.
func CheckPlayerDefeat(manager *common.EntityManager) (bool, string) {
	// Check threat influence
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > templates.OverworldConfigTemplate.VictoryConditions.MaxThreatInfluence {
		return true, fmt.Sprintf("Defeat! Overwhelmed by threat influence (%.1f)", totalInfluence)
	}

	// Check high-intensity threats
	highIntensityThreshold := templates.OverworldConfigTemplate.VictoryConditions.HighIntensityThreshold
	highCount := threat.CountHighIntensityThreats(manager, highIntensityThreshold)
	if highCount >= templates.OverworldConfigTemplate.VictoryConditions.MaxHighIntensityThreats {
		return true, fmt.Sprintf("Defeat! Too many powerful threats (%d tier-%d+ threats)",
			highCount, highIntensityThreshold)
	}

	return false, ""
}

// GetTotalThreatInfluence calculates combined threat pressure as sum of intensities.
// Uses unified OverworldNodeComponent, filters by threat category.
func GetTotalThreatInfluence(manager *common.EntityManager) float64 {
	total := 0.0

	for _, result := range core.OverworldNodeView.Get() {
		nodeData := common.GetComponentType[*core.OverworldNodeData](result.Entity, core.OverworldNodeComponent)
		if nodeData != nil && nodeData.Category == core.NodeCategoryThreat {
			total += float64(nodeData.Intensity)
		}
	}

	return total
}

// GetVictoryState retrieves singleton victory state
func GetVictoryState(manager *common.EntityManager) *core.VictoryStateData {
	for _, result := range manager.World.Query(core.VictoryStateTag) {
		return common.GetComponentType[*core.VictoryStateData](result.Entity, core.VictoryStateComponent)
	}
	return nil
}
