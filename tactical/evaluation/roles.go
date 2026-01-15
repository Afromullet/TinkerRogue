package evaluation

import "game_main/tactical/squads"

// RoleMultipliers defines scoring multipliers for unit roles.
// Used by both threat evaluation (behavior) and power calculation (encounter).
// These values affect game balance - change with care.
var RoleMultipliers = map[squads.UnitRole]float64{
	squads.RoleTank:    1.2, // High survivability value
	squads.RoleDPS:     1.5, // High damage output value
	squads.RoleSupport: 1.0, // Baseline utility value
}

// LeaderBonus is the multiplier applied to leader units.
// Leaders provide tactical bonuses and are high-value targets.
const LeaderBonus = 1.3

// GetRoleMultiplier returns the scoring multiplier for a unit role.
// Returns 1.0 (baseline) for unknown roles.
func GetRoleMultiplier(role squads.UnitRole) float64 {
	if mult, exists := RoleMultipliers[role]; exists {
		return mult
	}
	return 1.0
}
