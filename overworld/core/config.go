package core

import (
	"game_main/templates"
)

// --- Data-Driven Accessor Functions ---
// These functions retrieve overworld configuration from JSON templates.
// They replace direct constant access to enable designer-friendly tuning.

// GetContainmentSlowdown returns the containment slowdown multiplier from config.
func GetContainmentSlowdown() float64 {
	return templates.OverworldConfigTemplate.ThreatGrowth.ContainmentSlowdown
}

// GetChildNodeSpawnThreshold returns the child node spawn threshold from config.
func GetChildNodeSpawnThreshold() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.ChildNodeSpawnThreshold
}

// GetMaxChildNodeSpawnAttempts returns the max child node spawn attempts from config.
func GetMaxChildNodeSpawnAttempts() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.MaxChildNodeSpawnAttempts
}

// GetDefaultIntentTickDuration returns the default intent tick duration from config.
func GetDefaultIntentTickDuration() int {
	return templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration
}

// GetExpansionTerritoryLimit returns the expansion territory limit from config.
func GetExpansionTerritoryLimit() int {
	return templates.OverworldConfigTemplate.FactionAI.ExpansionTerritoryLimit
}

// GetFortificationStrengthGain returns the fortification strength gain from config.
func GetFortificationStrengthGain() int {
	return templates.OverworldConfigTemplate.FactionAI.FortificationStrengthGain
}

// GetMaxTerritorySize returns the max territory size from config.
func GetMaxTerritorySize() int {
	return templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize
}

// --- Unified Strength Thresholds ---

// GetWeakThreshold returns the weak strength threshold from config.
func GetWeakThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Weak
}

// GetStrongThreshold returns the strong strength threshold from config.
func GetStrongThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Strong
}

// GetCriticalThreshold returns the critical strength threshold from config.
func GetCriticalThreshold() int {
	return templates.OverworldConfigTemplate.StrengthThresholds.Critical
}

// --- Faction Archetype System ---

// FactionArchetype defines strategic archetype and aggression level
type FactionArchetype struct {
	Strategy   string
	Aggression float64
}

// FactionBonuses defines behavior bonuses derived from archetype
type FactionBonuses struct {
	ExpansionBonus     float64
	FortificationBonus float64
	RaidingBonus       float64
	RetreatPenalty     float64
}

// strategyBonuses maps archetype strategies to behavior bonuses
var strategyBonuses = map[string]FactionBonuses{
	"Expansionist": {ExpansionBonus: 3.0, FortificationBonus: 0.0, RaidingBonus: 1.0, RetreatPenalty: 0.0},
	"Aggressor":    {ExpansionBonus: 2.0, FortificationBonus: 0.0, RaidingBonus: 4.0, RetreatPenalty: 0.0},
	"Raider":       {ExpansionBonus: 0.0, FortificationBonus: 0.0, RaidingBonus: 5.0, RetreatPenalty: -2.0},
	"Defensive":    {ExpansionBonus: 0.0, FortificationBonus: 2.0, RaidingBonus: 0.0, RetreatPenalty: 2.0},
	"Territorial":  {ExpansionBonus: -1.0, FortificationBonus: 1.0, RaidingBonus: 0.0, RetreatPenalty: -3.0},
}

// GetFactionArchetype returns archetype config for a faction type.
func GetFactionArchetype(factionType FactionType) FactionArchetype {
	factionName := factionType.String()
	if a, ok := templates.OverworldConfigTemplate.FactionArchetypes[factionName]; ok {
		return FactionArchetype{
			Strategy:   a.Strategy,
			Aggression: a.Aggression,
		}
	}
	// Default: neutral archetype
	return FactionArchetype{Strategy: "Defensive", Aggression: 0.5}
}

// GetFactionBonuses returns behavior bonuses for a faction type based on its archetype.
func GetFactionBonuses(factionType FactionType) FactionBonuses {
	archetype := GetFactionArchetype(factionType)
	if bonuses, ok := strategyBonuses[archetype.Strategy]; ok {
		return bonuses
	}
	return FactionBonuses{}
}

// GetFactionAggression returns the aggression level for a faction type.
func GetFactionAggression(factionType FactionType) float64 {
	return GetFactionArchetype(factionType).Aggression
}

// --- Victory Conditions ---

// GetHighIntensityThreshold returns the high intensity threshold from config.
func GetHighIntensityThreshold() int {
	return templates.OverworldConfigTemplate.VictoryConditions.HighIntensityThreshold
}

// GetMaxHighIntensityThreats returns the max high intensity threats from config.
func GetMaxHighIntensityThreats() int {
	return templates.OverworldConfigTemplate.VictoryConditions.MaxHighIntensityThreats
}

// GetMaxThreatInfluence returns the max threat influence threshold from config.
func GetMaxThreatInfluence() float64 {
	return templates.OverworldConfigTemplate.VictoryConditions.MaxThreatInfluence
}

// --- Faction Scoring Control ---

// GetIdleScoreThreshold returns the idle score threshold from config.
func GetIdleScoreThreshold() float64 {
	return templates.OverworldConfigTemplate.FactionScoringControl.IdleScoreThreshold
}

// GetRaidBaseIntensity returns the raid base intensity from config.
func GetRaidBaseIntensity() int {
	return templates.OverworldConfigTemplate.FactionScoringControl.RaidBaseIntensity
}

// GetRaidIntensityScale returns the raid intensity scale from config.
func GetRaidIntensityScale() float64 {
	return templates.OverworldConfigTemplate.FactionScoringControl.RaidIntensityScale
}

// --- Global Threat Config ---

// GetMaxThreatIntensity returns the global max threat intensity from config.
func GetMaxThreatIntensity() int {
	return templates.OverworldConfigTemplate.ThreatGrowth.MaxThreatIntensity
}

// GetExpansionThreatSpawnChance returns the expansion threat spawn chance from config.
func GetExpansionThreatSpawnChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.ExpansionThreatSpawnChance
}

// GetFortifyThreatSpawnChance returns the fortify threat spawn chance from config.
// Falls back to hardcoded default if template lookup fails.
func GetFortifyThreatSpawnChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.FortifyThreatSpawnChance
}

// GetBonusItemDropChance returns the bonus item drop chance from config.
func GetBonusItemDropChance() int {
	return templates.OverworldConfigTemplate.SpawnProbabilities.BonusItemDropChance
}

// GetThreatTypeParamsFromConfig returns parameters for each threat type from config.
// Falls back to hardcoded defaults if template lookup fails.
// Note: MaxIntensity is now a global config value - use GetMaxThreatIntensity().
func GetThreatTypeParamsFromConfig(threatType ThreatType) ThreatTypeParams {
	threatStr := threatType.String()

	for _, tt := range templates.OverworldConfigTemplate.ThreatTypes {
		if tt.ThreatType == threatStr {
			return ThreatTypeParams{
				BaseGrowthRate:   tt.BaseGrowthRate,
				BaseRadius:       tt.BaseRadius,
				PrimaryEffect:    stringToInfluenceEffect(tt.PrimaryEffect),
				CanSpawnChildren: tt.CanSpawnChildren,
			}
		}
	}

	// Fallback to hardcoded defaults
	return GetThreatTypeParams(threatType)
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

// GetExpansionScoringParams returns expansion scoring parameters from config.
// Returns (strongBonus, smallTerritoryBonus, maxTerritoryPenalty).
func GetExpansionScoringParams() (strongBonus, smallTerritoryBonus, maxTerritoryPenalty float64) {
	exp := templates.OverworldConfigTemplate.FactionScoring.Expansion
	return exp.StrongBonus, exp.SmallTerritoryBonus, exp.MaxTerritoryPenalty
}

// GetFortificationScoringParams returns fortification scoring parameters from config.
// Returns (weakBonus, baseValue).
func GetFortificationScoringParams() (weakBonus, baseValue float64) {
	fort := templates.OverworldConfigTemplate.FactionScoring.Fortification
	return fort.WeakBonus, fort.BaseValue
}

// GetRaidingScoringParams returns raiding scoring parameters from config.
// Returns strongBonus.
func GetRaidingScoringParams() float64 {
	return templates.OverworldConfigTemplate.FactionScoring.Raiding.StrongBonus
}

// GetRetreatScoringParams returns retreat scoring parameters from config.
// Returns (criticalWeakBonus, smallTerritoryPenalty, minTerritorySize).
func GetRetreatScoringParams() (criticalWeakBonus, smallTerritoryPenalty float64, minTerritorySize int) {
	retreat := templates.OverworldConfigTemplate.FactionScoring.Retreat
	return retreat.CriticalWeakBonus, retreat.SmallTerritoryPenalty, retreat.MinTerritorySize
}
