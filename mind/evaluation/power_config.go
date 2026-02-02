package evaluation

import (
	"game_main/tactical/squads"
)

// PowerConfig holds configurable weights for power calculations.
// Used by both encounter generation and AI threat assessment for consistent evaluation.
// Pure data component - no logic.
type PowerConfig struct {
	ProfileName string // "Offensive", "Defensive", "Balanced"

	// Unit-level weights (0.0-1.0 range, sum should equal 1.0 for each category)
	OffensiveWeight float64 // Weight for offensive stats (damage, hit, crit)
	DefensiveWeight float64 // Weight for defensive stats (HP, resistance, dodge)
	UtilityWeight   float64 // Weight for utility (role, abilities, cover)

	// Offensive sub-weights (should sum to 1.0)
	DamageWeight   float64 // Physical/magic damage output
	AccuracyWeight float64 // Hit rate and crit chance

	// Defensive sub-weights (should sum to 1.0)
	HealthWeight     float64 // Max HP and current HP
	ResistanceWeight float64 // Physical/magic resistance
	AvoidanceWeight  float64 // Dodge chance

	// Utility sub-weights (should sum to 1.0)
	RoleWeight    float64 // Role multiplier importance
	AbilityWeight float64 // Leader ability value
	CoverWeight   float64 // Cover provision value

	// Squad-level modifiers
	FormationBonus   float64 // Bonus per formation type
	MoraleMultiplier float64 // Morale impact (0.01 per morale point)
	HealthPenalty    float64 // Penalty multiplier for low HP squads

	// Roster-level modifiers
	DeployedWeight float64 // Weight for deployed squads (default 1.0)
	ReserveWeight  float64 // Weight for reserve squads (default 0.3)
}

// PowerProfile defines named configuration profiles.
// These are data-driven and can be loaded from JSON in the future.
type PowerProfile string

const (
	ProfileBalanced  PowerProfile = "Balanced"  // Equal weight offensive/defensive
	ProfileOffensive PowerProfile = "Offensive" // Prioritize damage output
	ProfileDefensive PowerProfile = "Defensive" // Prioritize survivability
	ProfileUtility   PowerProfile = "Utility"   // Prioritize support/abilities
)

// Power calculation constants
const (
	// Default category weights for balanced profile
	DefaultOffensiveWeight = 0.4
	DefaultDefensiveWeight = 0.4
	DefaultUtilityWeight   = 0.2

	// Squad-level modifier defaults
	DefaultMoraleMultiplier = 0.002 // +0.2% power per morale point
	MinimumHealthMultiplier = 0.1   // Minimum 10% power even at low HP

	// Deployment weights
	DefaultDeployedWeight = 1.0 // Full weight for deployed squads
	DefaultReserveWeight  = 0.3 // 30% weight for reserves
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

		// Offensive sub-weights (sum to 1.0)
		DamageWeight:   0.6,
		AccuracyWeight: 0.4,

		// Defensive sub-weights (sum to 1.0)
		HealthWeight:     0.5,
		ResistanceWeight: 0.3,
		AvoidanceWeight:  0.2,

		// Utility sub-weights (sum to 1.0)
		RoleWeight:    0.5,
		AbilityWeight: 0.3,
		CoverWeight:   0.2,

		// Squad modifiers
		FormationBonus:   1.0, // No bonus by default
		MoraleMultiplier: DefaultMoraleMultiplier,
		HealthPenalty:    2.0, // HP% squared penalty (50% HP = 0.25x power)

		// Roster modifiers
		DeployedWeight: DefaultDeployedWeight,
		ReserveWeight:  DefaultReserveWeight,
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
