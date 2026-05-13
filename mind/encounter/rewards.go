package encounter

import (
	"game_main/mind/combatlifecycle"
	"game_main/templates"
)

// CalculateIntensityReward determines loot from defeating a threat.
// Reward multiplier is derived from intensity:
//
//	1.0 + (intensity * IntensityMultiplierStep)
//
// With the default step of 0.1, this yields 1.1x–1.5x for intensity 1–5.
// All tunables live in templates.GameConfig.Encounter, loaded from
// gamedata/gameconfig.json.
func CalculateIntensityReward(intensity int) combatlifecycle.Reward {
	cfg := templates.GameConfig.Encounter

	baseGold := cfg.BaseGold + (intensity * cfg.GoldPerIntensity)
	baseXP := cfg.BaseXP + (intensity * cfg.XPPerIntensity)
	basePoints := cfg.BasePoints + (intensity * cfg.PointsPerIntensity)

	typeMultiplier := 1.0 + (float64(intensity) * cfg.IntensityMultiplierStep)

	return combatlifecycle.Reward{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		ArcanaPts:  int(float64(basePoints) * typeMultiplier),
		SkillPts:   int(float64(basePoints) * typeMultiplier),
	}
}
