package encounter

import "game_main/mind/combatlifecycle"

// CalculateIntensityReward determines loot from defeating a threat.
// Reward multiplier is derived from intensity: 1.0 + (intensity x 0.1) gives 1.1x-1.5x for intensity 1-5.
func CalculateIntensityReward(intensity int) combatlifecycle.Reward {
	baseGold := 100 + (intensity * 50)
	baseXP := 50 + (intensity * 25)
	basePoints := 1 + intensity

	typeMultiplier := 1.0 + (float64(intensity) * 0.1)

	return combatlifecycle.Reward{
		Gold:       int(float64(baseGold) * typeMultiplier),
		Experience: int(float64(baseXP) * typeMultiplier),
		ArcanaPts:  int(float64(basePoints) * typeMultiplier),
		SkillPts:   int(float64(basePoints) * typeMultiplier),
	}
}
