package behavior

import (
	"game_main/common"
	"game_main/mind/balance"
	"game_main/mind/evaluation"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Re-export balance constants for backward compatibility within behavior package.
// New code should import from tactical/balance directly.
const (
	FlankingThreatRangeBonus   = balance.FlankingThreatRangeBonus
	IsolationSafeDistance      = balance.IsolationSafeDistance
	IsolationModerateDistance  = balance.IsolationModerateDistance
	IsolationHighDistance      = balance.IsolationHighDistance
	EngagementPressureMax      = balance.EngagementPressureMax
	RetreatSafeThreatThreshold = balance.RetreatSafeThreatThreshold
)

// Support layer constants (re-exported from balance).
const (
	SupportHealRadius           = balance.SupportHealRadius
	SupportAllyProximityRadius  = balance.SupportAllyProximityRadius
	BuffPriorityEngagementRange = balance.BuffPriorityEngagementRange
)

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
