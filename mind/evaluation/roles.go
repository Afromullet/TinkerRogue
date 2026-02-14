package evaluation

import (
	"game_main/tactical/squads"
	"game_main/templates"
)

// Scaling constants for power calculations.
// These are internal implementation details, not designer-tunable parameters.
// They convert raw stat values to comparable power scores.
const (
	RoleScalingFactor          = 10.0  // Base multiplier for role value
	CoverScalingFactor         = 100.0 // Scale cover value (0.0-0.5) to comparable range (0-50)
	CoverBeneficiaryMultiplier = 2.5   // Average units protected per cover provider
)

// --- Data-Driven Accessor Functions ---
// These functions retrieve role and scaling configuration from JSON templates.
// They replace direct map access to enable designer-friendly tuning.

// GetRoleMultiplierFromConfig returns the role multiplier from JSON config.
// Falls back to default values if not found in config.
func GetRoleMultiplierFromConfig(role squads.UnitRole) float64 {
	roleStr := role.String()
	for _, rm := range templates.PowerConfigTemplate.RoleMultipliers {
		if rm.Role == roleStr {
			return rm.Multiplier
		}
	}
	// Fallback to default values
	switch role {
	case squads.RoleTank:
		return 1.2
	case squads.RoleDPS:
		return 1.5
	case squads.RoleSupport:
		return 1.0
	default:
		return 1.0
	}
}

// GetAbilityPowerValue returns the power value for a specific ability from JSON config.
// Falls back to default values if not found in config.
func GetAbilityPowerValue(ability squads.AbilityType) float64 {
	abilityStr := ability.String()
	for _, av := range templates.PowerConfigTemplate.AbilityValues {
		if av.Ability == abilityStr {
			return av.Power
		}
	}
	// Fallback to default values
	switch ability {
	case squads.AbilityRally:
		return 15.0
	case squads.AbilityHeal:
		return 20.0
	case squads.AbilityBattleCry:
		return 12.0
	case squads.AbilityFireball:
		return 18.0
	case squads.AbilityNone:
		return 0.0
	default:
		return 0.0
	}
}

// GetCompositionBonusFromConfig returns the composition bonus from JSON config.
// Falls back to default values if not found in config.
func GetCompositionBonusFromConfig(uniqueAttackTypes int) float64 {
	for _, cb := range templates.PowerConfigTemplate.CompositionBonuses {
		if cb.UniqueTypes == uniqueAttackTypes {
			return cb.Bonus
		}
	}
	// Fallback to default values
	switch uniqueAttackTypes {
	case 1:
		return 0.8 // Mono-composition penalty
	case 2:
		return 1.1 // Dual-type bonus
	case 3:
		return 1.2 // Triple-type bonus
	case 4:
		return 1.3 // Quad-type bonus
	default:
		return 1.0
	}
}
