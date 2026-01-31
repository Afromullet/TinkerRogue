package evaluation

import (
	"game_main/tactical/squads"
	"game_main/templates"
)

// DEPRECATED: Use GetRoleMultiplierFromConfig() instead.
// This map is now loaded from powerconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
var RoleMultipliers = map[squads.UnitRole]float64{
	squads.RoleTank:    1.2, // High survivability value
	squads.RoleDPS:     1.5, // High damage output value
	squads.RoleSupport: 1.0, // Baseline utility value
}

// DEPRECATED: Use GetLeaderBonusFromConfig() instead.
// This constant is now loaded from powerconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
const LeaderBonus = 1.3

// DEPRECATED: Use GetScalingConstants() instead.
// These constants are now loaded from powerconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
const (
	RoleScalingFactor          = 10.0  // Base multiplier for role value
	DodgeScalingFactor         = 100.0 // Scale dodge to 0-40 range
	CoverScalingFactor         = 100.0 // Scale cover value percentage
	CoverBeneficiaryMultiplier = 2.5   // Avg units protected per cover provider
)

// DEPRECATED: Use GetRoleMultiplierFromConfig() instead.
// This function uses hardcoded values. The new function loads from powerconfig.json.
// Kept for fallback purposes only.
func GetRoleMultiplier(role squads.UnitRole) float64 {
	if mult, exists := RoleMultipliers[role]; exists {
		return mult
	}
	return 1.0
}

// DEPRECATED: Use GetCompositionBonusFromConfig() instead.
// This map is now loaded from powerconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
var CompositionBonuses = map[int]float64{
	1: 0.8, // Mono-composition penalty (vulnerable to counters)
	2: 1.1, // Dual-type bonus (good diversity)
	3: 1.2, // Triple-type bonus (excellent diversity)
	4: 1.3, // Quad-type bonus (optimal, rare)
}

// DEPRECATED: Use GetCompositionBonusFromConfig() instead.
// This function uses hardcoded values. The new function loads from powerconfig.json.
// Kept for fallback purposes only.
func GetCompositionBonus(uniqueAttackTypes int) float64 {
	if bonus, exists := CompositionBonuses[uniqueAttackTypes]; exists {
		return bonus
	}
	return 1.0
}

// --- Data-Driven Accessor Functions ---
// These functions retrieve role and scaling configuration from JSON templates.
// They replace direct map access to enable designer-friendly tuning.

// GetRoleMultiplierFromConfig returns the role multiplier from JSON config.
// Falls back to hardcoded map if not found in config.
func GetRoleMultiplierFromConfig(role squads.UnitRole) float64 {
	roleStr := role.String()
	for _, rm := range templates.PowerConfigTemplate.RoleMultipliers {
		if rm.Role == roleStr {
			return rm.Multiplier
		}
	}
	// Fallback to hardcoded map
	return GetRoleMultiplier(role)
}

// GetLeaderBonusFromConfig returns the leader bonus multiplier from JSON config.
// Falls back to hardcoded constant if not found in config.
func GetLeaderBonusFromConfig() float64 {
	if templates.PowerConfigTemplate.ScalingConstants.LeaderBonus > 0 {
		return templates.PowerConfigTemplate.ScalingConstants.LeaderBonus
	}
	return LeaderBonus
}

// GetAbilityPowerValue returns the power value for a specific ability from JSON config.
// Falls back to hardcoded map if not found in config.
func GetAbilityPowerValue(ability squads.AbilityType) float64 {
	abilityStr := ability.String()
	for _, av := range templates.PowerConfigTemplate.AbilityValues {
		if av.Ability == abilityStr {
			return av.Power
		}
	}
	// Fallback to hardcoded map
	if power, exists := AbilityPowerValues[ability]; exists {
		return power
	}
	return 0.0
}

// GetCompositionBonusFromConfig returns the composition bonus from JSON config.
// Falls back to hardcoded map if not found in config.
func GetCompositionBonusFromConfig(uniqueAttackTypes int) float64 {
	for _, cb := range templates.PowerConfigTemplate.CompositionBonuses {
		if cb.UniqueTypes == uniqueAttackTypes {
			return cb.Bonus
		}
	}
	// Fallback to hardcoded map
	return GetCompositionBonus(uniqueAttackTypes)
}

// ScalingConstants holds scaling factors for power calculations.
type ScalingConstants struct {
	RoleScaling                float64
	DodgeScaling               float64
	CoverScaling               float64
	CoverBeneficiaryMultiplier float64
	LeaderBonus                float64
}

// GetScalingConstants returns all scaling constants from JSON config.
// Falls back to hardcoded constants if not found in config.
func GetScalingConstants() ScalingConstants {
	sc := templates.PowerConfigTemplate.ScalingConstants
	if sc.RoleScaling > 0 && sc.DodgeScaling > 0 && sc.CoverScaling > 0 &&
		sc.CoverBeneficiaryMultiplier > 0 && sc.LeaderBonus > 0 {
		return ScalingConstants{
			RoleScaling:                sc.RoleScaling,
			DodgeScaling:               sc.DodgeScaling,
			CoverScaling:               sc.CoverScaling,
			CoverBeneficiaryMultiplier: sc.CoverBeneficiaryMultiplier,
			LeaderBonus:                sc.LeaderBonus,
		}
	}
	// Fallback to hardcoded constants
	return ScalingConstants{
		RoleScaling:                RoleScalingFactor,
		DodgeScaling:               DodgeScalingFactor,
		CoverScaling:               CoverScalingFactor,
		CoverBeneficiaryMultiplier: CoverBeneficiaryMultiplier,
		LeaderBonus:                LeaderBonus,
	}
}
