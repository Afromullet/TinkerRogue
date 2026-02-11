package core

import (
	"game_main/templates"
)

// --- Faction Scoring Params (grouped for convenience) ---

// GetExpansionScoringParams returns expansion scoring parameters.
func GetExpansionScoringParams() (strongBonus, smallTerritoryBonus, maxTerritoryPenalty float64) {
	exp := templates.OverworldConfigTemplate.FactionScoring.Expansion
	return exp.StrongBonus, exp.SmallTerritoryBonus, exp.MaxTerritoryPenalty
}

// GetFortificationScoringParams returns fortification scoring parameters.
func GetFortificationScoringParams() (weakBonus, baseValue float64) {
	fort := templates.OverworldConfigTemplate.FactionScoring.Fortification
	return fort.WeakBonus, fort.BaseValue
}

// GetRaidingScoringParams returns raiding scoring parameters.
func GetRaidingScoringParams() (strongBonus float64, veryStrongOffset int) {
	raid := templates.OverworldConfigTemplate.FactionScoring.Raiding
	return raid.StrongBonus, raid.VeryStrongOffset
}

// GetRetreatScoringParams returns retreat scoring parameters.
func GetRetreatScoringParams() (criticalWeakBonus, smallTerritoryPenalty float64, minTerritorySize int) {
	retreat := templates.OverworldConfigTemplate.FactionScoring.Retreat
	return retreat.CriticalWeakBonus, retreat.SmallTerritoryPenalty, retreat.MinTerritorySize
}

// --- Threat Type Config ---

// GetThreatTypeParamsFromConfig returns parameters for a threat type.
// Delegates to NodeRegistry (not a simple template forward).
func GetThreatTypeParamsFromConfig(threatType ThreatType) ThreatTypeParams {
	return GetNodeRegistry().GetOverworldParams(threatType)
}

// stringToInfluenceEffect converts string to InfluenceEffect enum
func stringToInfluenceEffect(s string) InfluenceEffect {
	switch s {
	case "SpawnBoost":
		return InfluenceSpawnBoost
	case "ResourceDrain":
		return InfluenceResourceDrain
	case "TerrainCorruption":
		return InfluenceTerrainCorruption
	case "CombatDebuff":
		return InfluenceCombatDebuff
	default:
		return InfluenceSpawnBoost
	}
}
