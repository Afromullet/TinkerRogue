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

// --- Difficulty-Adjusted Overworld Accessors ---

// GetContainmentSlowdown returns the containment slowdown scaled by difficulty.
func GetContainmentSlowdown() float64 {
	base := templates.OverworldConfigTemplate.ThreatGrowth.ContainmentSlowdown
	return base * templates.GlobalDifficulty.Overworld().ContainmentSlowdownScale
}

// GetMaxThreatIntensity returns the max threat intensity adjusted by difficulty offset.
func GetMaxThreatIntensity() int {
	base := templates.OverworldConfigTemplate.ThreatGrowth.MaxThreatIntensity
	result := base + templates.GlobalDifficulty.Overworld().MaxThreatIntensityOffset
	if result < 1 {
		return 1
	}
	return result
}

// GetExpansionThreatSpawnChance returns the expansion spawn chance scaled by difficulty, clamped 0-100.
func GetExpansionThreatSpawnChance() int {
	base := templates.OverworldConfigTemplate.SpawnProbabilities.ExpansionThreatSpawnChance
	result := int(float64(base) * templates.GlobalDifficulty.Overworld().SpawnChanceScale)
	if result < 0 {
		return 0
	}
	if result > 100 {
		return 100
	}
	return result
}

// GetFortifyThreatSpawnChance returns the fortify spawn chance scaled by difficulty, clamped 0-100.
func GetFortifyThreatSpawnChance() int {
	base := templates.OverworldConfigTemplate.SpawnProbabilities.FortifyThreatSpawnChance
	result := int(float64(base) * templates.GlobalDifficulty.Overworld().SpawnChanceScale)
	if result < 0 {
		return 0
	}
	if result > 100 {
		return 100
	}
	return result
}

// GetFortificationStrengthGain returns the fortification strength gain scaled by difficulty, min 1.
func GetFortificationStrengthGain() int {
	result := int(templates.GlobalDifficulty.Overworld().FortificationStrengthGainScale)
	if result < 1 {
		return 1
	}
	return result
}

// GetRaidBaseIntensity returns the raid base intensity scaled by difficulty.
func GetRaidBaseIntensity() int {
	base := templates.OverworldConfigTemplate.FactionScoringControl.RaidBaseIntensity
	return int(float64(base) * templates.GlobalDifficulty.Overworld().RaidIntensityScale)
}

// --- Threat Type Config ---

// GetThreatTypeParamsFromConfig returns parameters for a threat type.
// Delegates to NodeRegistry (not a simple template forward).
func GetThreatTypeParamsFromConfig(threatType ThreatType) ThreatTypeParams {
	return GetNodeRegistry().GetOverworldParams(threatType)
}

