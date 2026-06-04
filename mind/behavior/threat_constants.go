package behavior

import (
	"game_main/tactical/squads/unitdefs"
	"game_main/templates"
)

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

// GetIsolationMaxDistance returns the distance at which isolation risk saturates to 1.0.
// No difficulty offset applies; it is a pure gradient parameter.
func GetIsolationMaxDistance() int {
	base := 8 // Default isolation max distance
	if v := templates.AIConfigTemplate.ThreatCalculation.IsolationMaxDistance; v > 0 {
		base = v
	}
	return base
}

// GetEngagementPressureMax returns the combined melee+ranged threat that normalizes to 1.0.
// Combined threat above this value clamps every position to maximum engagement pressure, so
// raising it spreads the gradient out while lowering it saturates the layer sooner.
func GetEngagementPressureMax() int {
	base := 200 // Default engagement pressure normalizer
	if v := templates.AIConfigTemplate.ThreatCalculation.EngagementPressureMax; v > 0 {
		base = v
	}
	return base
}

// GetRoleBehaviorWeights returns threat layer weights for a specific role from config.
// RangedWeight and PositionalWeight use shared config values (roles differentiated by melee/support).
// Role behaviors are validated at boot (every role is required), so the lookup normally
// succeeds; the neutral fallback only guards a missing/malformed config rather than mirroring
// the JSON values, which would silently rot.
func GetRoleBehaviorWeights(role unitdefs.UnitRole) RoleThreatWeights {
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
	// Neutral fallback (config not loaded). Source of truth is aiconfig.json.
	return RoleThreatWeights{
		MeleeWeight:      0.5,
		RangedWeight:     rangedWeight,
		SupportWeight:    0.5,
		PositionalWeight: positionalWeight,
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

// getPositionalRiskWeights returns the weights for combining the four positional risk
// dimensions. When the config section is absent (all zero) it falls back to equal weights,
// which reproduces the historic simple average.
func getPositionalRiskWeights() templates.PositionalRiskWeightsConfig {
	w := templates.AIConfigTemplate.PositionalRiskWeights
	if w.Flanking == 0 && w.Isolation == 0 && w.EngagementPressure == 0 && w.Retreat == 0 {
		return templates.PositionalRiskWeightsConfig{Flanking: 1, Isolation: 1, EngagementPressure: 1, Retreat: 1}
	}
	return w
}
