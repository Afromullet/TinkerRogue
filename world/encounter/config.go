package encounter

import (
	"game_main/tactical/evaluation"
	"game_main/tactical/squads"
)

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
