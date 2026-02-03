package encounter

// EncounterDifficultyModifier defines how encounter level scales enemy power
type EncounterDifficultyModifier struct {
	PowerMultiplier float64 // Multiply player power by this (e.g., 0.7 for easier, 1.5 for harder)
	SquadCount      int     // Fixed number of enemy squads
}

// Squad type identifiers for composition control
const (
	SquadTypeMelee  = "melee"
	SquadTypeRanged = "ranged"
	SquadTypeMagic  = "magic"
)
