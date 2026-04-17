package combatlifecycle

import (
	"game_main/templates"
)

// ClampPowerTarget applies min/max bounds from the difficulty modifier to a raw power target.
// If raw is zero or negative, returns the minimum. If above max, returns the max.
func ClampPowerTarget(raw float64, mod templates.JSONEncounterDifficulty) float64 {
	if raw <= 0.0 {
		return mod.MinTargetPower
	}
	if raw > mod.MaxTargetPower {
		return mod.MaxTargetPower
	}
	return raw
}
