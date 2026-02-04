package evaluation

import (
	"game_main/templates"
)

// PowerConfig holds configurable weights for power calculations.
// Used by both encounter generation and AI threat assessment for consistent evaluation.
// Pure data component - no logic.
//
// Only top-level category weights are configurable. Sub-calculations use fixed
// formulas internally (e.g., offensive = damage * hitRate * critMultiplier).
type PowerConfig struct {
	ProfileName string // "Balanced" (only profile in use)

	// Unit-level weights (0.0-1.0 range, sum should equal 1.0)
	OffensiveWeight float64 // Weight for offensive stats (damage output)
	DefensiveWeight float64 // Weight for defensive stats (survivability)
	UtilityWeight   float64 // Weight for utility (role, abilities, cover)

	// Squad-level modifiers
	HealthPenalty float64 // Exponent for health-based power scaling (e.g., 2.0 = squared)
}

// PowerProfile defines named configuration profiles.
// Currently only Balanced is used. Other profiles can be added via powerconfig.json.
type PowerProfile string

const (
	ProfileBalanced PowerProfile = "Balanced" // Equal weight offensive/defensive (only profile in use)
)

// Power calculation constants
const (
	// Default category weights for balanced profile
	DefaultOffensiveWeight = 0.4
	DefaultDefensiveWeight = 0.4
	DefaultUtilityWeight   = 0.2

	// Squad-level modifier defaults
	// Note: MoraleMultiplier is 0 (dormant) - morale system not yet implemented
	DefaultMoraleMultiplier = 0.0
)

// --- Data-Driven Accessor Functions ---
// These functions retrieve power evaluation configuration from JSON templates.
// They replace direct constant/map access to enable designer-friendly tuning.

// GetPowerConfigByProfile returns power configuration for the specified profile name.
// Converts JSON profile to runtime PowerConfig struct.
// Falls back to default balanced profile if not found.
func GetPowerConfigByProfile(profileName string) *PowerConfig {
	// Try to find profile in loaded config
	for _, profile := range templates.PowerConfigTemplate.Profiles {
		if profile.Name == profileName {
			return &PowerConfig{
				ProfileName:     profile.Name,
				OffensiveWeight: profile.OffensiveWeight,
				DefensiveWeight: profile.DefensiveWeight,
				UtilityWeight:   profile.UtilityWeight,
				HealthPenalty:   profile.HealthPenalty,
			}
		}
	}
	// Fallback to default balanced profile
	return &PowerConfig{
		ProfileName:     "Balanced",
		OffensiveWeight: DefaultOffensiveWeight,
		DefensiveWeight: DefaultDefensiveWeight,
		UtilityWeight:   DefaultUtilityWeight,
		HealthPenalty:   2.0,
	}
}
