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
	IsolationSafeDistance      = 2
	IsolationModerateDistance  = 3
	IsolationHighDistance      = 6
	EngagementPressureMax      = 200
	RetreatSafeThreatThreshold = 10
)

// DEPRECATED: Use GetSupportLayerParams() instead.
// These constants are now loaded from aiconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
const (
	SupportHealRadius           = 3
	SupportAllyProximityRadius  = 2
	BuffPriorityEngagementRange = 4
)

// Threat calculation constants
const (
	// Movement defaults
	DefaultSquadMovement = 3   // Base movement when no data available
	MaxSpeedSentinel     = 999 // Sentinel for finding minimum speed

	// Reference target for expected damage calculations
	// Represents a medium-difficulty enemy unit with typical stats
	ReferenceTargetStrength  = 10 // → 12 resistance (10/4 + 5*2)
	ReferenceTargetDexterity = 15 // → 5% dodge (15/3)
	ReferenceTargetArmor     = 5  // → Contributes to resistance
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
var DefaultRoleWeights = map[squads.UnitRole]RoleThreatWeights{
	squads.RoleTank: {
		MeleeWeight:      -0.5, // Tanks SEEK melee danger (intercept enemies)
		RangedWeight:     0.3,  // Moderate concern for ranged
		SupportWeight:    0.2,  // Stay near support for heals
		PositionalWeight: 0.5,  // High concern for isolation
	},
	squads.RoleDPS: {
		MeleeWeight:      0.7, // Avoid melee danger
		RangedWeight:     0.5, // Moderate concern for ranged
		SupportWeight:    0.1, // Low support priority
		PositionalWeight: 0.6, // High concern for flanking
	},
	squads.RoleSupport: {
		MeleeWeight:      1.0,  // Strongly avoid melee danger
		RangedWeight:     0.8,  // Strongly avoid ranged pressure
		SupportWeight:    -1.0, // SEEK high support value positions (wounded allies)
		PositionalWeight: 0.4,  // Moderate positional awareness
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

// GetIsolationDistances returns isolation distance thresholds from config.
// Returns (safe, moderate, high) distances. Falls back to hardcoded defaults if template lookup fails.
func GetIsolationDistances() (safe, moderate, high int) {
	tc := templates.AIConfigTemplate.ThreatCalculation
	if tc.IsolationSafeDistance > 0 && tc.IsolationModerateDistance > 0 && tc.IsolationHighDistance > 0 {
		return tc.IsolationSafeDistance, tc.IsolationModerateDistance, tc.IsolationHighDistance
	}
	return IsolationSafeDistance, IsolationModerateDistance, IsolationHighDistance
}

// GetEngagementPressureMax returns the max engagement pressure threshold from config.
// Returns hardcoded default if template lookup fails.
func GetEngagementPressureMax() int {
	if templates.AIConfigTemplate.ThreatCalculation.EngagementPressureMax > 0 {
		return templates.AIConfigTemplate.ThreatCalculation.EngagementPressureMax
	}
	return EngagementPressureMax
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
// Falls back to hardcoded defaults if template lookup fails.
func GetRoleBehaviorWeights(role squads.UnitRole) RoleThreatWeights {
	roleStr := role.String()
	for _, rb := range templates.AIConfigTemplate.RoleBehaviors {
		if rb.Role == roleStr {
			return RoleThreatWeights{
				MeleeWeight:      rb.MeleeWeight,
				RangedWeight:     rb.RangedWeight,
				SupportWeight:    rb.SupportWeight,
				PositionalWeight: rb.PositionalWeight,
			}
		}
	}
	// Fallback to hardcoded defaults
	if weights, exists := DefaultRoleWeights[role]; exists {
		return weights
	}
	return RoleThreatWeights{
		MeleeWeight:      0.5,
		RangedWeight:     0.5,
		SupportWeight:    0.5,
		PositionalWeight: 0.5,
	}
}

// GetPositionalRiskWeights returns positional risk evaluation weights from config.
// Returns (flanking, isolation, pressure, retreat) weights.
// Falls back to equal weights (0.25 each) if template lookup fails.
func GetPositionalRiskWeights() (flanking, isolation, pressure, retreat float64) {
	pr := templates.AIConfigTemplate.PositionalRisk
	if pr.FlankingWeight > 0 || pr.IsolationWeight > 0 || pr.PressureWeight > 0 || pr.RetreatWeight > 0 {
		return pr.FlankingWeight, pr.IsolationWeight, pr.PressureWeight, pr.RetreatWeight
	}
	return 0.4, 0.3, 0.2, 0.1
}

// GetSupportLayerParams returns support layer configuration parameters from config.
// Returns (healRadius, proximityRadius, buffRange).
// Falls back to hardcoded defaults if template lookup fails.
func GetSupportLayerParams() (healRadius, proximityRadius, buffRange int) {
	sl := templates.AIConfigTemplate.SupportLayer
	if sl.HealRadius > 0 && sl.AllyProximityRadius > 0 && sl.BuffPriorityEngagementRange > 0 {
		return sl.HealRadius, sl.AllyProximityRadius, sl.BuffPriorityEngagementRange
	}
	return SupportHealRadius, SupportAllyProximityRadius, BuffPriorityEngagementRange
}
