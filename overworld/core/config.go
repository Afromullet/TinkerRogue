package core

import (
	"game_main/templates"
)

// --- Threat Growth Config ---

// GetContainmentSlowdown returns the containment slowdown multiplier.
func GetContainmentSlowdown() float64 {
	return templates.OverworldConfigTemplate.ThreatGrowth.ContainmentSlowdown
}

// GetChildNodeSpawnThreshold returns the child node spawn threshold.
func GetChildNodeSpawnThreshold() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.ChildNodeSpawnThreshold
}

// GetMaxChildNodeSpawnAttempts returns the max child node spawn attempts.
func GetMaxChildNodeSpawnAttempts() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.MaxChildNodeSpawnAttempts
}

// GetMaxThreatIntensity returns the global max threat intensity.
func GetMaxThreatIntensity() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.MaxThreatIntensity
}

// --- Faction AI Config ---

// GetDefaultIntentTickDuration returns the default intent tick duration.
func GetDefaultIntentTickDuration() int {
	return templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration
}

// GetExpansionTerritoryLimit returns the expansion territory limit.
func GetExpansionTerritoryLimit() int {
	return templates.OverworldConfigTemplate.FactionAI.ExpansionTerritoryLimit
}

// GetFortificationStrengthGain returns the fortification strength gain.
func GetFortificationStrengthGain() int {
	return templates.OverworldConfigTemplate.FactionAI.FortificationStrengthGain
}

// GetMaxTerritorySize returns the max territory size.
func GetMaxTerritorySize() int {
	return templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize
}

// --- Strength Thresholds ---

// GetWeakThreshold returns the weak strength threshold.
func GetWeakThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Weak
}

// GetStrongThreshold returns the strong strength threshold.
func GetStrongThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Strong
}

// GetCriticalThreshold returns the critical strength threshold.
func GetCriticalThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Critical
}

// --- Victory Conditions ---

// GetHighIntensityThreshold returns the high intensity threshold.
func GetHighIntensityThreshold() int {
	return templates.OverworldConfigTemplate.VictoryConditions.HighIntensityThreshold
}

// GetMaxHighIntensityThreats returns the max high intensity threats.
func GetMaxHighIntensityThreats() int {
	return templates.OverworldConfigTemplate.VictoryConditions.MaxHighIntensityThreats
}

// GetMaxThreatInfluence returns the max threat influence threshold.
func GetMaxThreatInfluence() float64 {
	return templates.OverworldConfigTemplate.VictoryConditions.MaxThreatInfluence
}

// --- Faction Scoring Control ---

// GetIdleScoreThreshold returns the idle score threshold.
func GetIdleScoreThreshold() float64 {
	return templates.OverworldConfigTemplate.FactionScoringControl.IdleScoreThreshold
}

// GetRaidBaseIntensity returns the raid base intensity.
func GetRaidBaseIntensity() int {
	return templates.OverworldConfigTemplate.FactionScoringControl.RaidBaseIntensity
}

// GetRaidIntensityScale returns the raid intensity scale.
func GetRaidIntensityScale() float64 {
	return templates.OverworldConfigTemplate.FactionScoringControl.RaidIntensityScale
}

// --- Spawn Probabilities ---

// GetExpansionThreatSpawnChance returns the expansion threat spawn chance.
func GetExpansionThreatSpawnChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.ExpansionThreatSpawnChance
}

// GetFortifyThreatSpawnChance returns the fortify threat spawn chance.
func GetFortifyThreatSpawnChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.FortifyThreatSpawnChance
}

// GetBonusItemDropChance returns the bonus item drop chance.
func GetBonusItemDropChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.BonusItemDropChance
}

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
func GetRaidingScoringParams() float64 {
	return templates.OverworldConfigTemplate.FactionScoring.Raiding.StrongBonus
}

// GetRetreatScoringParams returns retreat scoring parameters.
func GetRetreatScoringParams() (criticalWeakBonus, smallTerritoryPenalty float64, minTerritorySize int) {
	retreat := templates.OverworldConfigTemplate.FactionScoring.Retreat
	return retreat.CriticalWeakBonus, retreat.SmallTerritoryPenalty, retreat.MinTerritorySize
}

// --- Strategy Bonuses ---

// GetStrategyBonuses returns the strategy bonuses map from config.
func GetStrategyBonuses() map[string]templates.StrategyBonusConfig {
	return templates.OverworldConfigTemplate.StrategyBonuses
}

// --- Player Node Config ---

// GetMaxPlacementRange returns the max distance from player/existing nodes for placement.
func GetMaxPlacementRange() int {
	return templates.OverworldConfigTemplate.PlayerNodes.MaxPlacementRange
}

// GetMaxPlayerNodes returns the max number of player nodes allowed.
func GetMaxPlayerNodes() int {
	return templates.OverworldConfigTemplate.PlayerNodes.MaxNodes
}

// --- Threat Type Config ---

// GetThreatTypeParamsFromConfig returns parameters for a threat type.
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
