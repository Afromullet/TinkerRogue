package overworld

import (
	"game_main/common"
)

// High-intensity threat thresholds for defeat conditions
const (
	HighIntensityThreshold  = 8     // Intensity level considered "high"
	MaxHighIntensityThreats = 10    // Maximum allowed before defeat
	MaxThreatInfluence      = 100.0 // Maximum total influence before defeat
)

// CheckPlayerDefeat checks if player has lost and returns structured result.
// Single source of truth for defeat determination - runs checks once and caches results.
// Replaces the duplicate logic that was split between IsPlayerDefeated() and GetDefeatReason().
func CheckPlayerDefeat(manager *common.EntityManager) *DefeatCheckResult {
	result := &DefeatCheckResult{
		IsDefeated:   false,
		DefeatReason: DefeatNone,
	}

	// Check threat influence (run query once and cache result)
	result.TotalInfluence = GetTotalThreatInfluence(manager)
	if result.TotalInfluence > MaxThreatInfluence {
		result.IsDefeated = true
		result.DefeatReason = DefeatByInfluence
		result.DefeatMessage = formatEventString("Defeat! Overwhelmed by threat influence (%.1f)", result.TotalInfluence)
		return result
	}

	// Check high-intensity threats (run query once and cache result)
	result.HighIntensityCount = CountHighIntensityThreats(manager, HighIntensityThreshold)
	if result.HighIntensityCount >= MaxHighIntensityThreats {
		result.IsDefeated = true
		result.DefeatReason = DefeatByHighIntensityThreats
		result.DefeatMessage = formatEventString("Defeat! Too many powerful threats (%d tier-%d+ threats)",
			result.HighIntensityCount, HighIntensityThreshold)
		return result
	}

	// Check squad loss
	if HasPlayerLostAllSquads(manager) {
		result.IsDefeated = true
		result.DefeatReason = DefeatBySquadLoss
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
	if squadChecker == nil {
		return false
	}

	// Invert the checker result: HasActiveSquads=false means all squads lost
	return !squadChecker.HasActiveSquads(manager)
}

// HasPlayerEliminatedAllThreats checks if all threats are gone
func HasPlayerEliminatedAllThreats(manager *common.EntityManager) bool {
	threatCount := CountThreatNodes(manager)
	return threatCount == 0
}

// HasPlayerDefeatedFactionType checks if specific faction type is eliminated
func HasPlayerDefeatedFactionType(manager *common.EntityManager, factionType FactionType) bool {
	for _, result := range manager.World.Query(OverworldFactionTag) {
		factionData := common.GetComponentType[*OverworldFactionData](result.Entity, OverworldFactionComponent)
		if factionData != nil && factionData.FactionType == factionType {
			return false // Faction still exists
		}
	}
	return true // No factions of this type found
}

// GetTotalThreatInfluence calculates combined threat pressure
func GetTotalThreatInfluence(manager *common.EntityManager) float64 {
	total := 0.0

	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		influenceData := common.GetComponentType[*InfluenceData](result.Entity, InfluenceComponent)

		if threatData != nil && influenceData != nil {
			// Influence value = intensity × radius × strength
			influence := float64(threatData.Intensity) * float64(influenceData.Radius) * influenceData.EffectStrength
			total += influence
		}
	}

	return total
}

// GetVictoryState retrieves singleton victory state
func GetVictoryState(manager *common.EntityManager) *VictoryStateData {
	for _, result := range manager.World.Query(VictoryStateTag) {
		return common.GetComponentType[*VictoryStateData](result.Entity, VictoryStateComponent)
	}
	return nil
}
