package behavior

import (
	"game_main/common"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Role threat multipliers (shared across all threat systems)
const (
	ThreatMultiplierDPS     = 1.5 // DPS units deal highest threat
	ThreatMultiplierTank    = 1.2 // Tanks provide high durability threat
	ThreatMultiplierSupport = 1.0 // Support provides utility baseline
)

// Positional risk constants
const (
	// Flanking threat extends this many tiles beyond movement speed
	FlankingThreatRangeBonus = 3

	// Isolation risk thresholds (in tiles from nearest ally)
	IsolationSafeDistance     = 2 // 0-2 tiles = no isolation risk
	IsolationModerateDistance = 3 // 3-5 tiles = moderate risk
	IsolationHighDistance     = 6 // 6+ tiles = high isolation risk

	// Engagement pressure normalization (max expected damage per position)
	EngagementPressureMax = 200.0

	// Retreat quality threshold (threat values below this are safe escape routes)
	RetreatSafeThreatThreshold = 10.0
)

// Support layer constants
const (
	// Default healing support radius (in tiles)
	SupportHealRadius = 3

	// Ally proximity tracking radius (in tiles)
	SupportAllyProximityRadius = 2

	// Buff priority engagement distance thresholds
	BuffPriorityEngagementRange = 4 // Within 4 tiles = prioritize buffs
)

// RoleModifiers defines threat multipliers for unit roles
var RoleModifiers = map[squads.UnitRole]float64{
	squads.RoleTank:    ThreatMultiplierTank,
	squads.RoleDPS:     ThreatMultiplierDPS,
	squads.RoleSupport: ThreatMultiplierSupport,
}

// GetRoleModifier returns threat multiplier for a role
func GetRoleModifier(role squads.UnitRole) float64 {
	if modifier, exists := RoleModifiers[role]; exists {
		return modifier
	}
	return ThreatMultiplierSupport // Default to baseline
}

// GetSquadRoleModifier returns threat modifier based on squad's primary role
func GetSquadRoleModifier(squadID ecs.EntityID, manager *common.EntityManager) float64 {
	role := getSquadPrimaryRole(squadID, manager)
	return GetRoleModifier(role)
}
