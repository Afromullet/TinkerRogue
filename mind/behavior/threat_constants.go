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

// getSharedRangedWeight returns the shared ranged threat weight from config.
// Falls back to default if not configured.
func getSharedRangedWeight() float64 {
	if templates.AIConfigTemplate.SharedRangedWeight > 0 {
		return templates.AIConfigTemplate.SharedRangedWeight
	}
	return defaultSharedRangedWeight
}

// getSharedPositionalWeight returns the shared positional awareness weight from config.
// Falls back to default if not configured.
func getSharedPositionalWeight() float64 {
	if templates.AIConfigTemplate.SharedPositionalWeight > 0 {
		return templates.AIConfigTemplate.SharedPositionalWeight
	}
	return defaultSharedPositionalWeight
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

// GetFlankingThreatRangeBonus returns the flanking threat range bonus from config.
// Returns default value if template lookup fails.
func GetFlankingThreatRangeBonus() int {
	if templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus > 0 {
		return templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus
	}
	return 3 // Default flanking threat range bonus
}

// GetIsolationThreshold returns the isolation distance threshold from config.
// Units farther than this from allies start accumulating isolation risk.
// Returns default value if template lookup fails.
func GetIsolationThreshold() int {
	tc := templates.AIConfigTemplate.ThreatCalculation
	if tc.IsolationThreshold > 0 {
		return tc.IsolationThreshold
	}
	return 3 // Default isolation threshold
}

// GetRetreatSafeThreatThreshold returns the retreat safety threshold from config.
// Returns default value if template lookup fails.
func GetRetreatSafeThreatThreshold() int {
	if templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold > 0 {
		return templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold
	}
	return 10 // Default retreat safe threat threshold
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
			MeleeWeight:      -0.5,           // Tanks SEEK melee danger
			RangedWeight:     rangedWeight,   // Shared config value
			SupportWeight:    0.2,            // Stay near support for heals
			PositionalWeight: positionalWeight,
		}
	case squads.RoleDPS:
		return RoleThreatWeights{
			MeleeWeight:      0.7,            // Avoid melee danger
			RangedWeight:     rangedWeight,   // Shared config value
			SupportWeight:    0.1,            // Low support priority
			PositionalWeight: positionalWeight,
		}
	case squads.RoleSupport:
		return RoleThreatWeights{
			MeleeWeight:      1.0,            // Strongly avoid melee danger
			RangedWeight:     rangedWeight,   // Shared config value
			SupportWeight:    -1.0,           // SEEK high support value positions
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
