package encounter

import (
	"game_main/tactical/evaluation"
	"game_main/tactical/squads"
)

// EvaluationConfigData holds configurable weights for power calculations
// Pure data component - no logic
type EvaluationConfigData struct {
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
	LeaderBonus      float64         // Leader presence multiplier (1.2-1.5)
	CompositionBonus map[int]float64 // Bonus by unique attack type count (1→0.8, 2→1.1, 3→1.2, 4→1.3)
	HealthPenalty    float64         // Penalty multiplier for low HP squads

	// Roster-level modifiers
	DeployedWeight float64 // Weight for deployed squads (default 1.0)
	ReserveWeight  float64 // Weight for reserve squads (default 0.3)
}

// EvaluationProfile defines named configuration profiles
// These are data-driven and can be loaded from JSON in the future
type EvaluationProfile string

const (
	ProfileBalanced  EvaluationProfile = "Balanced"  // Equal weight offensive/defensive
	ProfileOffensive EvaluationProfile = "Offensive" // Prioritize damage output
	ProfileDefensive EvaluationProfile = "Defensive" // Prioritize survivability
	ProfileUtility   EvaluationProfile = "Utility"   // Prioritize support/abilities
)

// GetDefaultConfig returns a default balanced configuration
func GetDefaultConfig() *EvaluationConfigData {
	return &EvaluationConfigData{
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
		FormationBonus:   1.0,                     // No bonus by default
		MoraleMultiplier: DefaultMoraleMultiplier, // +0.2% power per morale point (20% at 100 morale)
		LeaderBonus:      evaluation.LeaderBonus,  // Shared leader bonus from evaluation package
		CompositionBonus: map[int]float64{ // Bonus by unique attack type count
			1: 0.8, // Monoculture penalty
			2: 1.1, // Mixed composition bonus
			3: 1.2, // Diverse composition bonus
			4: 1.3, // Maximum diversity bonus
		},
		HealthPenalty: 2.0, // HP% multiplier (0.5 HP = 1.0 power, 1.0 HP = 2.0 power)

		// Roster modifiers
		DeployedWeight: DefaultDeployedWeight, // Full weight for deployed squads
		ReserveWeight:  DefaultReserveWeight,  // 30% weight for reserves (they exist but not immediately available)
	}
}

// GetOffensiveConfig returns an offensive-focused configuration
func GetOffensiveConfig() *EvaluationConfigData {
	config := GetDefaultConfig()
	config.ProfileName = string(ProfileOffensive)
	config.OffensiveWeight = 0.6
	config.DefensiveWeight = 0.25
	config.UtilityWeight = 0.15
	config.DamageWeight = 0.7
	config.AccuracyWeight = 0.3
	return config
}

// GetDefensiveConfig returns a defensive-focused configuration
func GetDefensiveConfig() *EvaluationConfigData {
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

// RolePowerModifiers aliases shared role multipliers for backward compatibility.
// These match the AI threat system for consistency (both use evaluation.RoleMultipliers).
var RolePowerModifiers = evaluation.RoleMultipliers

// AbilityPowerValues maps leader abilities to power ratings
var AbilityPowerValues = map[squads.AbilityType]float64{
	squads.AbilityRally:     15.0, // +5 Strength for 3 turns = sustained damage
	squads.AbilityHeal:      20.0, // 10 HP heal = high value
	squads.AbilityBattleCry: 12.0, // +3 Strength + morale once per combat
	squads.AbilityFireball:  18.0, // 15 direct damage AoE
	squads.AbilityNone:      0.0,  // No ability
}

// GetConfigByProfile returns a pre-configured evaluation profile
func GetConfigByProfile(profile EvaluationProfile) *EvaluationConfigData {
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
