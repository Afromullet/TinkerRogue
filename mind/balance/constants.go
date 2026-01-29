// Package balance provides centralized balance constants for the tactical combat system.
// All gameplay-affecting constants should be defined here for easy tuning.
package balance

// ========================================
// THREAT ASSESSMENT CONSTANTS
// ========================================

// Positional risk constants used by AI threat evaluation.
const (
	// FlankingThreatRangeBonus extends threat range for flanking calculations.
	FlankingThreatRangeBonus = 3

	// Isolation risk thresholds (in tiles from nearest ally).
	IsolationSafeDistance     = 2 // 0-2 tiles = no isolation risk
	IsolationModerateDistance = 3 // 3-5 tiles = moderate risk
	IsolationHighDistance     = 6 // 6+ tiles = high isolation risk

	// EngagementPressureMax is the normalization value for expected damage per position.
	EngagementPressureMax = 200.0

	// RetreatSafeThreatThreshold defines threat values below which positions are safe escape routes.
	RetreatSafeThreatThreshold = 10.0
)

// Support layer constants for AI support unit behavior.
const (
	// SupportHealRadius is the default healing support radius in tiles.
	SupportHealRadius = 3

	// SupportAllyProximityRadius is the radius for ally proximity tracking.
	SupportAllyProximityRadius = 2

	// BuffPriorityEngagementRange is the distance within which buffs are prioritized.
	BuffPriorityEngagementRange = 4
)

// ========================================
// ENCOUNTER DIFFICULTY CONSTANTS
// ========================================

// EncounterDifficultyModifier defines how encounter level scales enemy power.
type EncounterDifficultyModifier struct {
	PowerMultiplier float64 // Multiply player power by this
	MinSquads       int     // Minimum enemy squads
	MaxSquads       int     // Maximum enemy squads
}

// DifficultyTable maps encounter level to difficulty modifiers.
// Used by encounter generation to scale enemy power and squad counts.
var DifficultyTable = map[int]EncounterDifficultyModifier{
	1: {PowerMultiplier: 0.7, MinSquads: 2, MaxSquads: 3}, // Easy
	2: {PowerMultiplier: 0.9, MinSquads: 3, MaxSquads: 4}, // Moderate
	3: {PowerMultiplier: 1.0, MinSquads: 3, MaxSquads: 5}, // Fair fight
	4: {PowerMultiplier: 1.2, MinSquads: 4, MaxSquads: 6}, // Hard
	5: {PowerMultiplier: 1.5, MinSquads: 5, MaxSquads: 7}, // Boss-tier
}

// ========================================
// SQUAD COMPOSITION PREFERENCES
// ========================================

// Squad type identifiers for composition control.
const (
	SquadTypeMelee  = "melee"
	SquadTypeRanged = "ranged"
	SquadTypeMagic  = "magic"
)

// SquadPreferences maps encounter types to preferred squad compositions.
// Key is encounter type string, value is ordered list of squad types.
var SquadPreferences = map[string][]string{
	"goblin_basic": {SquadTypeMelee, SquadTypeMelee, SquadTypeRanged},
	"bandit_basic": {SquadTypeMelee, SquadTypeRanged, SquadTypeRanged},
	"beast_basic":  {SquadTypeMelee, SquadTypeMelee, SquadTypeMelee},
	"orc_basic":    {SquadTypeMelee, SquadTypeRanged, SquadTypeMagic},
}
