package core

import (
	"game_main/templates"
)

// --- Data-Driven Accessor Functions ---
// These functions retrieve overworld configuration from JSON templates.
// They replace direct constant access to enable designer-friendly tuning.

// GetThreatGrowthRate returns the default threat growth rate from config.
// Falls back to hardcoded default if template lookup fails.
func GetThreatGrowthRate() float64 {
	if templates.OverworldConfigTemplate.ThreatGrowth.DefaultGrowthRate > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.DefaultGrowthRate
	}
	return DefaultGrowthRate
}

// GetContainmentSlowdown returns the containment slowdown multiplier from config.
// Falls back to hardcoded default if template lookup fails.
func GetContainmentSlowdown() float64 {
	if templates.OverworldConfigTemplate.ThreatGrowth.ContainmentSlowdown > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.ContainmentSlowdown
	}
	return ContainmentSlowdown
}

// GetMaxThreatIntensity returns the maximum threat intensity from config.
// Falls back to hardcoded default if template lookup fails.
func GetMaxThreatIntensity() int {
	if templates.OverworldConfigTemplate.ThreatGrowth.MaxThreatIntensity > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.MaxThreatIntensity
	}
	return MaxThreatIntensity
}

// GetChildNodeSpawnThreshold returns the child node spawn threshold from config.
// Falls back to hardcoded default if template lookup fails.
func GetChildNodeSpawnThreshold() int {
	if templates.OverworldConfigTemplate.ThreatGrowth.ChildNodeSpawnThreshold > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.ChildNodeSpawnThreshold
	}
	return ChildNodeSpawnThreshold
}

// GetPlayerContainmentRadius returns the player containment radius from config.
// Falls back to hardcoded default if template lookup fails.
func GetPlayerContainmentRadius() int {
	if templates.OverworldConfigTemplate.ThreatGrowth.PlayerContainmentRadius > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.PlayerContainmentRadius
	}
	return PlayerContainmentRadius
}

// GetMaxChildNodeSpawnAttempts returns the max child node spawn attempts from config.
// Falls back to hardcoded default if template lookup fails.
func GetMaxChildNodeSpawnAttempts() int {
	if templates.OverworldConfigTemplate.ThreatGrowth.MaxChildNodeSpawnAttempts > 0 {
		return templates.OverworldConfigTemplate.ThreatGrowth.MaxChildNodeSpawnAttempts
	}
	return MaxChildNodeSpawnAttempts
}

// GetDefaultIntentTickDuration returns the default intent tick duration from config.
// Falls back to hardcoded default if template lookup fails.
func GetDefaultIntentTickDuration() int {
	if templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration > 0 {
		return templates.OverworldConfigTemplate.FactionAI.DefaultIntentTickDuration
	}
	return DefaultIntentTickDuration
}

// GetExpansionStrengthThreshold returns the expansion strength threshold from config.
// Falls back to hardcoded default if template lookup fails.
func GetExpansionStrengthThreshold() int {
	if templates.OverworldConfigTemplate.FactionAI.ExpansionStrengthThreshold > 0 {
		return templates.OverworldConfigTemplate.FactionAI.ExpansionStrengthThreshold
	}
	return ExpansionStrengthThreshold
}

// GetExpansionTerritoryLimit returns the expansion territory limit from config.
// Falls back to hardcoded default if template lookup fails.
func GetExpansionTerritoryLimit() int {
	if templates.OverworldConfigTemplate.FactionAI.ExpansionTerritoryLimit > 0 {
		return templates.OverworldConfigTemplate.FactionAI.ExpansionTerritoryLimit
	}
	return ExpansionTerritoryLimit
}

// GetFortificationWeakThreshold returns the fortification weak threshold from config.
// Falls back to hardcoded default if template lookup fails.
func GetFortificationWeakThreshold() int {
	if templates.OverworldConfigTemplate.FactionAI.FortificationWeakThreshold > 0 {
		return templates.OverworldConfigTemplate.FactionAI.FortificationWeakThreshold
	}
	return FortificationWeakThreshold
}

// GetFortificationStrengthGain returns the fortification strength gain from config.
// Falls back to hardcoded default if template lookup fails.
func GetFortificationStrengthGain() int {
	if templates.OverworldConfigTemplate.FactionAI.FortificationStrengthGain > 0 {
		return templates.OverworldConfigTemplate.FactionAI.FortificationStrengthGain
	}
	return FortificationStrengthGain
}

// GetRaidStrengthThreshold returns the raid strength threshold from config.
// Falls back to hardcoded default if template lookup fails.
func GetRaidStrengthThreshold() int {
	if templates.OverworldConfigTemplate.FactionAI.RaidStrengthThreshold > 0 {
		return templates.OverworldConfigTemplate.FactionAI.RaidStrengthThreshold
	}
	return RaidStrengthThreshold
}

// GetRaidProximityRange returns the raid proximity range from config.
// Falls back to hardcoded default if template lookup fails.
func GetRaidProximityRange() int {
	if templates.OverworldConfigTemplate.FactionAI.RaidProximityRange > 0 {
		return templates.OverworldConfigTemplate.FactionAI.RaidProximityRange
	}
	return RaidProximityRange
}

// GetRetreatCriticalStrength returns the retreat critical strength from config.
// Falls back to hardcoded default if template lookup fails.
func GetRetreatCriticalStrength() int {
	if templates.OverworldConfigTemplate.FactionAI.RetreatCriticalStrength > 0 {
		return templates.OverworldConfigTemplate.FactionAI.RetreatCriticalStrength
	}
	return RetreatCriticalStrength
}

// GetMaxTerritorySize returns the max territory size from config.
// Falls back to hardcoded default if template lookup fails.
func GetMaxTerritorySize() int {
	if templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize > 0 {
		return templates.OverworldConfigTemplate.FactionAI.MaxTerritorySize
	}
	return MaxTerritorySize
}

// GetExpansionThreatSpawnChance returns the expansion threat spawn chance from config.
// Falls back to hardcoded default if template lookup fails.
func GetExpansionThreatSpawnChance() int {
	if templates.OverworldConfigTemplate.SpawnProbabilities.ExpansionThreatSpawnChance >= 0 {
		return templates.OverworldConfigTemplate.SpawnProbabilities.ExpansionThreatSpawnChance
	}
	return ExpansionThreatSpawnChance
}

// GetFortifyThreatSpawnChance returns the fortify threat spawn chance from config.
// Falls back to hardcoded default if template lookup fails.
func GetFortifyThreatSpawnChance() int {
	if templates.OverworldConfigTemplate.SpawnProbabilities.FortifyThreatSpawnChance >= 0 {
		return templates.OverworldConfigTemplate.SpawnProbabilities.FortifyThreatSpawnChance
	}
	return FortifyThreatSpawnChance
}

// GetBonusItemDropChance returns the bonus item drop chance from config.
// Falls back to hardcoded default if template lookup fails.
func GetBonusItemDropChance() int {
	if templates.OverworldConfigTemplate.SpawnProbabilities.BonusItemDropChance >= 0 {
		return templates.OverworldConfigTemplate.SpawnProbabilities.BonusItemDropChance
	}
	return BonusItemDropChance
}

// GetDefaultMapWidth returns the default map width from config.
// Falls back to hardcoded default if template lookup fails.
func GetDefaultMapWidth() int {
	if templates.OverworldConfigTemplate.MapDimensions.DefaultMapWidth > 0 {
		return templates.OverworldConfigTemplate.MapDimensions.DefaultMapWidth
	}
	return DefaultMapWidth
}

// GetDefaultMapHeight returns the default map height from config.
// Falls back to hardcoded default if template lookup fails.
func GetDefaultMapHeight() int {
	if templates.OverworldConfigTemplate.MapDimensions.DefaultMapHeight > 0 {
		return templates.OverworldConfigTemplate.MapDimensions.DefaultMapHeight
	}
	return DefaultMapHeight
}

// GetThreatTypeParamsFromConfig returns parameters for each threat type from config.
// Falls back to hardcoded defaults if template lookup fails.
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
