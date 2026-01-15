package evaluation

// CompositionBonuses defines multipliers based on attack type diversity.
// Squads with diverse attack types (melee + ranged + magic) are more effective.
// Key is the count of unique attack types in the squad.
var CompositionBonuses = map[int]float64{
	1: 0.8, // Mono-composition penalty (vulnerable to counters)
	2: 1.1, // Dual-type bonus (good diversity)
	3: 1.2, // Triple-type bonus (excellent diversity)
	4: 1.3, // Quad-type bonus (optimal, rare)
}

// GetCompositionBonus returns the multiplier for a given count of unique attack types.
// Returns 1.0 (no modifier) for unexpected counts.
func GetCompositionBonus(uniqueAttackTypes int) float64 {
	if bonus, exists := CompositionBonuses[uniqueAttackTypes]; exists {
		return bonus
	}
	return 1.0
}
