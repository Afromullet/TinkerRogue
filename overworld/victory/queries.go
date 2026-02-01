package victory

import (
	"game_main/common"
	"game_main/world/overworld/core"
	"game_main/world/overworld/threat"
)

// High-intensity threat thresholds for defeat conditions
const (
	HighIntensityThreshold  = 4     // Intensity level considered "high" (4-5 are high with max 5)
	MaxHighIntensityThreats = 10    // Maximum allowed before defeat
	MaxThreatInfluence      = 100.0 // Maximum total influence before defeat
)

// CheckPlayerDefeat checks if player has lost and returns structured result.
// Single source of truth for defeat determination - runs checks once and caches results.
// Replaces the duplicate logic that was split between IsPlayerDefeated() and GetDefeatReason().
func CheckPlayerDefeat(manager *common.EntityManager) *core.DefeatCheckResult {
	result := &core.DefeatCheckResult{
		IsDefeated:   false,
		DefeatReason: core.DefeatNone,
	}

	// Check threat influence (run query once and cache result)
	result.TotalInfluence = GetTotalThreatInfluence(manager)
	if result.TotalInfluence > MaxThreatInfluence {
		result.IsDefeated = true
		result.DefeatReason = core.DefeatByInfluence
		result.DefeatMessage = core.FormatEventString("Defeat! Overwhelmed by threat influence (%.1f)", result.TotalInfluence)
		return result
	}

	// Check high-intensity threats (run query once and cache result)
	result.HighIntensityCount = threat.CountHighIntensityThreats(manager, HighIntensityThreshold)
	if result.HighIntensityCount >= MaxHighIntensityThreats {
		result.IsDefeated = true
		result.DefeatReason = core.DefeatByHighIntensityThreats
		result.DefeatMessage = core.FormatEventString("Defeat! Too many powerful threats (%d tier-%d+ threats)",
			result.HighIntensityCount, HighIntensityThreshold)
		return result
	}

	// Check squad loss
	if HasPlayerLostAllSquads(manager) {
		result.IsDefeated = true
		result.DefeatReason = core.DefeatBySquadLoss
		result.DefeatMessage = "Defeat! All squads destroyed"
		return result
	}

	return result
}

// IsPlayerDefeated checks if player has lost.
// DEPRECATED: Use CheckPlayerDefeat() instead for better performance (avoids duplicate checks).
func IsPlayerDefeated(manager *common.EntityManager) bool {
	return CheckPlayerDefeat(manager).IsDefeated
}

// HasPlayerLostAllSquads checks if player has any surviving squads
func HasPlayerLostAllSquads(manager *common.EntityManager) bool {
	// If no squad checker is injected, assume player hasn't lost
	// (squad-based defeat is optional feature)
	squadChecker := core.GetSquadChecker()
	if squadChecker == nil {
		return false
	}

	// Invert the checker result: HasActiveSquads=false means all squads lost
	return !squadChecker.HasActiveSquads(manager)
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
