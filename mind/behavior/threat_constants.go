package behavior

import (
	"game_main/common"
	"game_main/mind/evaluation"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Re-export balance constants for backward compatibility within behavior package.
// New code should import from tactical/balance directly.
const (
	FlankingThreatRangeBonus   = 3
	IsolationSafeDistance      = 2
	IsolationModerateDistance  = 3
	IsolationHighDistance      = 6
	EngagementPressureMax      = 200
	RetreatSafeThreatThreshold = 10
)

// Support layer constants (re-exported from balance).
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

// DefaultRoleWeights defines standard weights for each role
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
