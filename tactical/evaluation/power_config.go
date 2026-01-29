package evaluation

import "game_main/tactical/squads"

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
	FormationBonus   float64         // Bonus per formation type
	MoraleMultiplier float64         // Morale impact (0.01 per morale point)
	CompositionBonus map[int]float64 // Bonus by unique attack type count (1→0.8, 2→1.1, 3→1.2, 4→1.3)
	HealthPenalty    float64         // Penalty multiplier for low HP squads

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

	// Scaling factors for power calculations
	RoleScalingFactor          = 10.0  // Base multiplier for role value
	DodgeScalingFactor         = 100.0 // Scale dodge to 0-40 range
	CoverScalingFactor         = 100.0 // Scale cover value percentage
	CoverBeneficiaryMultiplier = 2.5   // Avg units protected per cover provider

	// Deployment weights
	DefaultDeployedWeight = 1.0 // Full weight for deployed squads
	DefaultReserveWeight  = 0.3 // 30% weight for reserves
)

// GetDefaultConfig returns a default balanced configuration.
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
		FormationBonus:   1.0,                   // No bonus by default
		MoraleMultiplier: DefaultMoraleMultiplier,
		CompositionBonus: CompositionBonuses,    // Use shared composition bonus
		HealthPenalty:    2.0,                   // HP% multiplier (0.5 HP = 1.0 power, 1.0 HP = 2.0 power)

		// Roster modifiers
		DeployedWeight: DefaultDeployedWeight,
		ReserveWeight:  DefaultReserveWeight,
	}
}

// GetOffensiveConfig returns an offensive-focused configuration.
func GetOffensiveConfig() *PowerConfig {
	config := GetDefaultConfig()
	config.ProfileName = string(ProfileOffensive)
	config.OffensiveWeight = 0.6
	config.DefensiveWeight = 0.25
	config.UtilityWeight = 0.15
	config.DamageWeight = 0.7
	config.AccuracyWeight = 0.3
	return config
}

// GetDefensiveConfig returns a defensive-focused configuration.
func GetDefensiveConfig() *PowerConfig {
	config := GetDefaultConfig()
	config.ProfileName = string(ProfileDefensive)
	config.OffensiveWeight = 0.25
	config.DefensiveWeight = 0.6
	config.UtilityWeight = 0.15
	config.HealthWeight = 0.6
	config.ResistanceWeight = 0.25
	config.AvoidanceWeight = 0.15
	return config
}

// GetConfigByProfile returns a pre-configured evaluation profile.
func GetConfigByProfile(profile PowerProfile) *PowerConfig {
	switch profile {
	case ProfileOffensive:
		return GetOffensiveConfig()
	case ProfileDefensive:
		return GetDefensiveConfig()
	case ProfileUtility:
		return GetDefaultConfig() // Future: add utility profile
	default:
		return GetDefaultConfig()
	}
}

// AbilityPowerValues maps leader abilities to power ratings.
var AbilityPowerValues = map[squads.AbilityType]float64{
	squads.AbilityRally:     15.0, // +5 Strength for 3 turns = sustained damage
	squads.AbilityHeal:      20.0, // 10 HP heal = high value
	squads.AbilityBattleCry: 12.0, // +3 Strength + morale once per combat
	squads.AbilityFireball:  18.0, // 15 direct damage AoE
	squads.AbilityNone:      0.0,  // No ability
}
