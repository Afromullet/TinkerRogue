package overworld

import (
	"game_main/common"
)

// IsPlayerDefeated checks if player has lost
func IsPlayerDefeated(manager *common.EntityManager) bool {
	// Player loses if threat influence is too high
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > 100.0 {
		return true
	}

	// Player loses if too many high-intensity threats exist
	highIntensityCount := 0
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil && threatData.Intensity >= 8 {
			highIntensityCount++
		}
	}

	if highIntensityCount >= 10 {
		return true // 10+ tier-8 threats = overwhelming
	}

	// Player loses if all squads destroyed
	if HasPlayerLostAllSquads(manager) {
		return true
	}

	return false
}

// GetDefeatReason returns a human-readable defeat reason
func GetDefeatReason(manager *common.EntityManager) string {
	totalInfluence := GetTotalThreatInfluence(manager)
	if totalInfluence > 100.0 {
		return formatEventString("Defeat! Overwhelmed by threat influence (%.1f)", totalInfluence)
	}

	highIntensityCount := 0
	for _, result := range manager.World.Query(ThreatNodeTag) {
		threatData := common.GetComponentType[*ThreatNodeData](result.Entity, ThreatNodeComponent)
		if threatData != nil && threatData.Intensity >= 8 {
			highIntensityCount++
		}
	}

	if highIntensityCount >= 10 {
		return formatEventString("Defeat! Too many powerful threats (%d tier-8+ threats)", highIntensityCount)
	}

	if HasPlayerLostAllSquads(manager) {
		return "Defeat! All squads destroyed"
	}

	return "Defeat! Unknown reason"
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
