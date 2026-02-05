package victory

import (
	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/threat"
)

// CheckPlayerDefeat checks if player has lost and returns structured result.
// Single source of truth for defeat determination - runs checks once and caches results.
// Thresholds are loaded from config for designer-friendly tuning.
func CheckPlayerDefeat(manager *common.EntityManager) *core.DefeatCheckResult {
	result := &core.DefeatCheckResult{
		IsDefeated:   false,
		DefeatReason: core.DefeatNone,
	}

	// Get thresholds from config
	maxInfluence := core.GetMaxThreatInfluence()
	highIntensityThreshold := core.GetHighIntensityThreshold()
	maxHighIntensityThreats := core.GetMaxHighIntensityThreats()

	// Check threat influence (run query once and cache result)
	result.TotalInfluence = GetTotalThreatInfluence(manager)
	if result.TotalInfluence > maxInfluence {
		result.IsDefeated = true
		result.DefeatReason = core.DefeatByInfluence
		result.DefeatMessage = core.FormatEventString("Defeat! Overwhelmed by threat influence (%.1f)", result.TotalInfluence)
		return result
	}

	// Check high-intensity threats (run query once and cache result)
	result.HighIntensityCount = threat.CountHighIntensityThreats(manager, highIntensityThreshold)
	if result.HighIntensityCount >= maxHighIntensityThreats {
		result.IsDefeated = true
		result.DefeatReason = core.DefeatByHighIntensityThreats
		result.DefeatMessage = core.FormatEventString("Defeat! Too many powerful threats (%d tier-%d+ threats)",
			result.HighIntensityCount, highIntensityThreshold)
		return result
	}

	return result
}

// GetTotalThreatInfluence calculates combined threat pressure
func GetTotalThreatInfluence(manager *common.EntityManager) float64 {
	total := 0.0

	for _, result := range manager.World.Query(core.ThreatNodeTag) {
		threatData := common.GetComponentType[*core.ThreatNodeData](result.Entity, core.ThreatNodeComponent)
		influenceData := common.GetComponentType[*core.InfluenceData](result.Entity, core.InfluenceComponent)

		if threatData != nil && influenceData != nil {
			// Influence value = intensity × radius × strength
			influence := float64(threatData.Intensity) * float64(influenceData.Radius) * influenceData.EffectStrength
			total += influence
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
