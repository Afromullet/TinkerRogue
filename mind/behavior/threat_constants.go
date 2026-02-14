package behavior

import (
	"game_main/tactical/squads"
	"game_main/templates"
)

// Max isolation distance for linear gradient calculation (internal constant)
const isolationMaxDistance = 8

// Hardcoded normalizer for engagement pressure (cosmetic, converts to 0-1 range)
const engagementPressureMax = 200

// Default values for shared weights (used when config not loaded)
const (
	defaultSharedRangedWeight     = 0.5
	defaultSharedPositionalWeight = 0.5
)

// getSharedRangedWeight returns the shared ranged threat weight from config,
// scaled by the active difficulty multiplier.
func getSharedRangedWeight() float64 {
	base := defaultSharedRangedWeight
	if templates.AIConfigTemplate.SharedRangedWeight > 0 {
		base = templates.AIConfigTemplate.SharedRangedWeight
	}
	return base * templates.GlobalDifficulty.AI().SharedRangedWeightScale
}

// getSharedPositionalWeight returns the shared positional awareness weight from config,
// scaled by the active difficulty multiplier.
func getSharedPositionalWeight() float64 {
	base := defaultSharedPositionalWeight
	if templates.AIConfigTemplate.SharedPositionalWeight > 0 {
		base = templates.AIConfigTemplate.SharedPositionalWeight
	}
	return base * templates.GlobalDifficulty.AI().SharedPositionalWeightScale
}

// RoleThreatWeights defines how each role weighs different threat layers.
// Negative weights = attraction (e.g., tanks seek melee danger, support seeks wounded allies)
// Positive weights = avoidance (e.g., support avoids all danger)
type RoleThreatWeights struct {
	MeleeWeight      float64
	RangedWeight     float64
	SupportWeight    float64
	PositionalWeight float64
}

// --- Data-Driven Accessor Functions ---
// These functions retrieve AI behavior configuration from JSON templates.
// They replace direct constant access to enable designer-friendly tuning.

// GetFlankingThreatRangeBonus returns the flanking threat range bonus from config,
// adjusted by the active difficulty offset.
func GetFlankingThreatRangeBonus() int {
	base := 3 // Default flanking threat range bonus
	if templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus > 0 {
		base = templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus
	}
	result := base + templates.GlobalDifficulty.AI().FlankingRangeBonusOffset
	if result < 1 {
		return 1
	}
	return result
}

// GetIsolationThreshold returns the isolation distance threshold from config,
// adjusted by the active difficulty offset.
func GetIsolationThreshold() int {
	base := 3 // Default isolation threshold
	tc := templates.AIConfigTemplate.ThreatCalculation
	if tc.IsolationThreshold > 0 {
		base = tc.IsolationThreshold
	}
	result := base + templates.GlobalDifficulty.AI().IsolationThresholdOffset
	if result < 1 {
		return 1
	}
	return result
}

// GetRetreatSafeThreatThreshold returns the retreat safety threshold from config,
// adjusted by the active difficulty offset.
func GetRetreatSafeThreatThreshold() int {
	base := 10 // Default retreat safe threat threshold
	if templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold > 0 {
		base = templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold
	}
	result := base + templates.GlobalDifficulty.AI().RetreatSafeThresholdOffset
	if result < 1 {
		return 1
	}
	return result
}

// GetRoleBehaviorWeights returns threat layer weights for a specific role from config.
// RangedWeight and PositionalWeight use shared config values (roles differentiated by melee/support).
// Falls back to default values if template lookup fails.
func GetRoleBehaviorWeights(role squads.UnitRole) RoleThreatWeights {
	rangedWeight := getSharedRangedWeight()
	positionalWeight := getSharedPositionalWeight()

	roleStr := role.String()
	for _, rb := range templates.AIConfigTemplate.RoleBehaviors {
		if rb.Role == roleStr {
			return RoleThreatWeights{
				MeleeWeight:      rb.MeleeWeight,
				RangedWeight:     rangedWeight,
				SupportWeight:    rb.SupportWeight,
				PositionalWeight: positionalWeight,
			}
		}
	}
	// Fallback to default values by role
	switch role {
	case squads.RoleTank:
		return RoleThreatWeights{
			MeleeWeight:      -0.5,         // Tanks SEEK melee danger
			RangedWeight:     rangedWeight, // Shared config value
			SupportWeight:    0.2,          // Stay near support for heals
			PositionalWeight: positionalWeight,
		}
	case squads.RoleDPS:
		return RoleThreatWeights{
			MeleeWeight:      0.7,          // Avoid melee danger
			RangedWeight:     rangedWeight, // Shared config value
			SupportWeight:    0.1,          // Low support priority
			PositionalWeight: positionalWeight,
		}
	case squads.RoleSupport:
		return RoleThreatWeights{
			MeleeWeight:      1.0,          // Strongly avoid melee danger
			RangedWeight:     rangedWeight, // Shared config value
			SupportWeight:    -1.0,         // SEEK high support value positions
			PositionalWeight: positionalWeight,
		}
	default:
		return RoleThreatWeights{
			MeleeWeight:      0.5,
			RangedWeight:     rangedWeight,
			SupportWeight:    0.5,
			PositionalWeight: positionalWeight,
		}
	}
}

// GetSupportLayerParams returns support layer configuration parameters from config.
// Returns (healRadius, proximityRadius).
// proximityRadius is derived as healRadius - 1.
// Falls back to default values if template lookup fails.
func GetSupportLayerParams() (healRadius, proximityRadius int) {
	sl := templates.AIConfigTemplate.SupportLayer
	if sl.HealRadius > 0 {
		return sl.HealRadius, sl.HealRadius - 1
	}
	const defaultHealRadius = 3
	return defaultHealRadius, defaultHealRadius - 1
}
