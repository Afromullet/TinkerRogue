package victory

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/threat"
)

// CheckPlayerDefeat checks if player has lost.
// Returns (isDefeated, defeatMessage). Empty message means not defeated.
func CheckPlayerDefeat(manager *common.EntityManager) (bool, string) {
	// Check threat influence
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > core.GetMaxThreatInfluence() {
		return true, core.FormatEventString("Defeat! Overwhelmed by threat influence (%.1f)", totalInfluence)
	}

	// Check high-intensity threats
	highIntensityThreshold := core.GetHighIntensityThreshold()
	highCount := threat.CountHighIntensityThreats(manager, highIntensityThreshold)
	if highCount >= core.GetMaxHighIntensityThreats() {
		return true, core.FormatEventString("Defeat! Too many powerful threats (%d tier-%d+ threats)",
			highCount, highIntensityThreshold)
	}

	return false, ""
}

// GetTotalThreatInfluence calculates combined threat pressure as sum of intensities
func GetTotalThreatInfluence(manager *common.EntityManager) float64 {
	total := 0.0

	for _, result := range manager.World.Query(core.ThreatNodeTag) {
		threatData := common.GetComponentType[*core.ThreatNodeData](result.Entity, core.ThreatNodeComponent)
		if threatData != nil {
			total += float64(threatData.Intensity)
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
