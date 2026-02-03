package behavior

import (
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/tactical/squads"
	"game_main/templates"

	"github.com/bytearena/ecs"
)

// DEPRECATED: Use GetFlankingThreatRangeBonus() and related accessor functions instead.
// These constants are now loaded from aiconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
const (
	FlankingThreatRangeBonus   = 3
	IsolationThreshold         = 3
	RetreatSafeThreatThreshold = 10
)

// Max isolation distance for linear gradient calculation (internal constant)
const isolationMaxDistance = 8

// Hardcoded normalizer for engagement pressure (cosmetic, converts to 0-1 range)
const engagementPressureMax = 200

// DEPRECATED: Use GetSupportLayerParams() instead.
// These constants are now loaded from aiconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
const (
	SupportHealRadius           = 3
	BuffPriorityEngagementRange = 4
)

// Shared constants for role weights (roles differentiated by melee/support weights only)
const (
	sharedRangedWeight     = 0.5 // Moderate concern for ranged threats
	sharedPositionalWeight = 0.5 // Moderate positional awareness
)


// RoleThreatWeights defines how each role weighs different threat layers.
// Negative weights = attraction (e.g., tanks seek melee danger, support seeks wounded allies)
// Positive weights = avoidance (e.g., support avoids all danger)
type RoleThreatWeights struct {
	MeleeWeight      float64
	RangedWeight     float64
	SupportWeight    float64
	PositionalWeight float64
}

// DEPRECATED: Use GetRoleBehaviorWeights() instead.
// This map is now loaded from aiconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
// Note: RangedWeight and PositionalWeight now use shared constants.
var DefaultRoleWeights = map[squads.UnitRole]RoleThreatWeights{
	squads.RoleTank: {
		MeleeWeight:      -0.5,                   // Tanks SEEK melee danger (intercept enemies)
		RangedWeight:     sharedRangedWeight,     // Shared moderate concern
		SupportWeight:    0.2,                    // Stay near support for heals
		PositionalWeight: sharedPositionalWeight, // Shared moderate awareness
	},
	squads.RoleDPS: {
		MeleeWeight:      0.7,                    // Avoid melee danger
		RangedWeight:     sharedRangedWeight,     // Shared moderate concern
		SupportWeight:    0.1,                    // Low support priority
		PositionalWeight: sharedPositionalWeight, // Shared moderate awareness
	},
	squads.RoleSupport: {
		MeleeWeight:      1.0,                    // Strongly avoid melee danger
		RangedWeight:     sharedRangedWeight,     // Shared moderate concern
		SupportWeight:    -1.0,                   // SEEK high support value positions (wounded allies)
		PositionalWeight: sharedPositionalWeight, // Shared moderate awareness
	},
}

// GetRoleModifier returns threat multiplier for a role.
// Delegates to shared evaluation package.
func GetRoleModifier(role squads.UnitRole) float64 {
	return evaluation.GetRoleMultiplier(role)
}

// GetSquadRoleModifier returns threat modifier based on squad's primary role.
func GetSquadRoleModifier(squadID ecs.EntityID, manager *common.EntityManager) float64 {
	role := squads.GetSquadPrimaryRole(squadID, manager)
	return GetRoleModifier(role)
}

// --- Data-Driven Accessor Functions ---
// These functions retrieve AI behavior configuration from JSON templates.
// They replace direct constant access to enable designer-friendly tuning.

// GetFlankingThreatRangeBonus returns the flanking threat range bonus from config.
// Returns hardcoded default if template lookup fails.
func GetFlankingThreatRangeBonus() int {
	if templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus > 0 {
		return templates.AIConfigTemplate.ThreatCalculation.FlankingThreatRangeBonus
	}
	return FlankingThreatRangeBonus
}

// GetIsolationThreshold returns the isolation distance threshold from config.
// Units farther than this from allies start accumulating isolation risk.
// Returns hardcoded default if template lookup fails.
func GetIsolationThreshold() int {
	tc := templates.AIConfigTemplate.ThreatCalculation
	if tc.IsolationThreshold > 0 {
		return tc.IsolationThreshold
	}
	return IsolationThreshold
}

// GetIsolationMaxDistance returns the max distance for isolation risk calculation.
// At this distance, isolation risk is 1.0 (fully isolated).
func GetIsolationMaxDistance() int {
	return isolationMaxDistance
}

// GetEngagementPressureMax returns the normalizer for engagement pressure.
// This is a cosmetic value that converts raw pressure to 0-1 range.
func GetEngagementPressureMax() int {
	return engagementPressureMax
}

// GetRetreatSafeThreatThreshold returns the retreat safety threshold from config.
// Returns hardcoded default if template lookup fails.
func GetRetreatSafeThreatThreshold() int {
	if templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold > 0 {
		return templates.AIConfigTemplate.ThreatCalculation.RetreatSafeThreatThreshold
	}
	return RetreatSafeThreatThreshold
}

// GetRoleBehaviorWeights returns threat layer weights for a specific role from config.
// RangedWeight and PositionalWeight use shared constants (roles differentiated by melee/support).
// Falls back to hardcoded defaults if template lookup fails.
func GetRoleBehaviorWeights(role squads.UnitRole) RoleThreatWeights {
	roleStr := role.String()
	for _, rb := range templates.AIConfigTemplate.RoleBehaviors {
		if rb.Role == roleStr {
			return RoleThreatWeights{
				MeleeWeight:      rb.MeleeWeight,
				RangedWeight:     sharedRangedWeight,     // Hardcoded shared constant
				SupportWeight:    rb.SupportWeight,
				PositionalWeight: sharedPositionalWeight, // Hardcoded shared constant
			}
		}
	}
	// Fallback to hardcoded defaults
	if weights, exists := DefaultRoleWeights[role]; exists {
		return weights
	}
	return RoleThreatWeights{
		MeleeWeight:      0.5,
		RangedWeight:     sharedRangedWeight,
		SupportWeight:    0.5,
		PositionalWeight: sharedPositionalWeight,
	}
}

// GetPositionalRiskWeights returns positional risk evaluation weights.
// Returns equal weights (0.25 each) for balanced risk evaluation.
// These are hardcoded since the role's positionalWeight already controls overall importance.
func GetPositionalRiskWeights() (flanking, isolation, pressure, retreat float64) {
	return 0.25, 0.25, 0.25, 0.25
}

// GetSupportLayerParams returns support layer configuration parameters from config.
// Returns (healRadius, proximityRadius, buffRange).
// proximityRadius is derived as healRadius - 1.
// Falls back to hardcoded defaults if template lookup fails.
func GetSupportLayerParams() (healRadius, proximityRadius, buffRange int) {
	sl := templates.AIConfigTemplate.SupportLayer
	if sl.HealRadius > 0 && sl.BuffPriorityEngagementRange > 0 {
		// Derive proximityRadius from healRadius
		return sl.HealRadius, sl.HealRadius - 1, sl.BuffPriorityEngagementRange
	}
	return SupportHealRadius, SupportHealRadius - 1, BuffPriorityEngagementRange
}
