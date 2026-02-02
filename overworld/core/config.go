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

// GetExpansionStrengthThreshold returns the expansion strength threshold from config.
func GetExpansionStrengthThreshold() int {
	return templates.OverworldConfigTemplate.FactionAI.ExpansionStrengthThreshold
}

// GetExpansionTerritoryLimit returns the expansion territory limit from config.
func GetExpansionTerritoryLimit() int {
	return templates.OverworldConfigTemplate.FactionAI.ExpansionTerritoryLimit

}

// GetFortificationWeakThreshold returns the fortification weak threshold from config.
func GetFortificationWeakThreshold() int {
	return templates.OverworldConfigTemplate.FactionAI.FortificationWeakThreshold
}

// GetFortificationStrengthGain returns the fortification strength gain from config.
func GetFortificationStrengthGain() int {
	return templates.OverworldConfigTemplate.FactionAI.FortificationStrengthGain
}

// GetRaidStrengthThreshold returns the raid strength threshold from config.
func GetRaidStrengthThreshold() int {
	return templates.OverworldConfigTemplate.FactionAI.RaidStrengthThreshold
}

// GetRetreatCriticalStrength returns the retreat critical strength from config.
func GetRetreatCriticalStrength() int {
	return templates.OverworldConfigTemplate.FactionAI.RetreatCriticalStrength
}

// GetMaxTerritorySize returns the max territory size from config.
func GetMaxTerritorySize() int {
	return templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize
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
func GetThreatTypeParamsFromConfig(threatType ThreatType) ThreatTypeParams {
	threatStr := threatType.String()

	for _, tt := range templates.OverworldConfigTemplate.ThreatTypes {
		if tt.ThreatType == threatStr {
			return ThreatTypeParams{
				BaseGrowthRate:   tt.BaseGrowthRate,
				BaseRadius:       tt.BaseRadius,
				PrimaryEffect:    stringToInfluenceEffect(tt.PrimaryEffect),
				CanSpawnChildren: tt.CanSpawnChildren,
				MaxIntensity:     tt.MaxIntensity,
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
// Returns (strongBonus, smallTerritoryBonus, maxTerritoryPenalty, cultistMod, orcMod, beastMod).
func GetExpansionScoringParams() (strongBonus, smallTerritoryBonus, maxTerritoryPenalty,
	cultistMod, orcMod, beastMod float64) {
	exp := templates.OverworldConfigTemplate.FactionScoring.Expansion
	return exp.StrongBonus, exp.SmallTerritoryBonus, exp.MaxTerritoryPenalty,
		exp.CultistModifier, exp.OrcModifier, exp.BeastModifier
}

// GetFortificationScoringParams returns fortification scoring parameters from config.
// Returns (weakBonus, baseValue, necromancerMod).
func GetFortificationScoringParams() (weakBonus, baseValue, necromancerMod float64) {
	fort := templates.OverworldConfigTemplate.FactionScoring.Fortification
	return fort.WeakBonus, fort.BaseValue, fort.NecromancerModifier
}

// GetRaidingScoringParams returns raiding scoring parameters from config.
// Returns (banditMod, orcMod, strongBonus, strongThreshold).
func GetRaidingScoringParams() (banditMod, orcMod, strongBonus float64, strongThreshold int) {
	raid := templates.OverworldConfigTemplate.FactionScoring.Raiding
	return raid.BanditModifier, raid.OrcModifier, raid.StrongBonus, raid.StrongThreshold
}

// GetRetreatScoringParams returns retreat scoring parameters from config.
// Returns (criticalWeakBonus, smallTerritoryPenalty, minTerritorySize).
func GetRetreatScoringParams() (criticalWeakBonus, smallTerritoryPenalty float64, minTerritorySize int) {
	retreat := templates.OverworldConfigTemplate.FactionScoring.Retreat
	return retreat.CriticalWeakBonus, retreat.SmallTerritoryPenalty, retreat.MinTerritorySize
}
