package evaluation

import (
	"game_main/tactical/squads"
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

// DEPRECATED: Use GetPowerConfigByProfile("Balanced") instead.
// This function is kept for fallback purposes only.
// Power configurations are now loaded from powerconfig.json for designer-friendly tuning.
func GetDefaultConfig() *PowerConfig {
	return &PowerConfig{
		ProfileName: string(ProfileBalanced),

		// Category weights (sum to 1.0)
		OffensiveWeight: DefaultOffensiveWeight,
		DefensiveWeight: DefaultDefensiveWeight,
		UtilityWeight:   DefaultUtilityWeight,

		// Squad modifiers
		HealthPenalty: 2.0, // HP% squared penalty (50% HP = 0.25x power)
	}
}

// DEPRECATED: Use GetAbilityPowerValue() instead.
// This map is now loaded from powerconfig.json for designer-friendly tuning.
// Kept for fallback purposes only.
var AbilityPowerValues = map[squads.AbilityType]float64{
	squads.AbilityRally:     15.0, // +5 Strength for 3 turns = sustained damage
	squads.AbilityHeal:      20.0, // 10 HP heal = high value
	squads.AbilityBattleCry: 12.0, // +3 Strength + morale once per combat
	squads.AbilityFireball:  18.0, // 15 direct damage AoE
	squads.AbilityNone:      0.0,  // No ability
}

// --- Data-Driven Accessor Functions ---
// These functions retrieve power evaluation configuration from JSON templates.
// They replace direct constant/map access to enable designer-friendly tuning.

// GetPowerConfigByProfile returns power configuration for the specified profile name.
// Converts JSON profile to runtime PowerConfig struct.
// Falls back to hardcoded defaults if profile not found.
