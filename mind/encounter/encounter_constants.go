package encounter

// Encounter type identifiers
type EncounterType string

const (
	EncounterGoblinBasic EncounterType = "goblin_basic"
	EncounterBanditBasic EncounterType = "bandit_basic"
	EncounterBeastBasic  EncounterType = "beast_basic"
	EncounterOrcBasic    EncounterType = "orc_basic"
)

// EncounterDifficultyModifier defines how encounter level scales enemy power
type EncounterDifficultyModifier struct {
	PowerMultiplier float64 // Multiply player power by this (e.g., 0.7 for easier, 1.5 for harder)
	SquadCount      int     // Fixed number of enemy squads
}

// EncounterDifficultyTable maps encounter level to difficulty modifiers.
//
// DEPRECATED: This map is now loaded from assets/gamedata/encounterdata.json.
// Use GetDifficultyModifier(level) from encounter_config.go instead.
// This map is kept temporarily for validation and backward compatibility.
var EncounterDifficultyTable = map[int]EncounterDifficultyModifier{
	1: {PowerMultiplier: 0.7, SquadCount: 2}, // Level 1: Easy, 70% power, 2 squads
	2: {PowerMultiplier: 0.9, SquadCount: 3}, // Level 2: Moderate, 90% power, 3 squads
	3: {PowerMultiplier: 1.0, SquadCount: 4}, // Level 3: Fair fight, equal power, 4 squads
	4: {PowerMultiplier: 1.2, SquadCount: 5}, // Level 4: Hard, 120% power, 5 squads
	5: {PowerMultiplier: 1.5, SquadCount: 6}, // Level 5: Boss-tier, 150% power, 6 squads
}

// Squad type identifiers for composition control
const (
	SquadTypeMelee  = "melee"
	SquadTypeRanged = "ranged"
	SquadTypeMagic  = "magic"
)

// EncounterSquadPreferences maps encounter types to preferred squad compositions.
// Each encounter favors certain unit types (melee, ranged, or magic).
//
// DEPRECATED: This map is now loaded from assets/gamedata/encounterdata.json.
// Use GetSquadPreferences(encounterType) from encounter_config.go instead.
// This map is kept temporarily for validation and backward compatibility.
var EncounterSquadPreferences = map[string][]string{
	string(EncounterGoblinBasic): {SquadTypeMelee, SquadTypeMelee, SquadTypeRanged},  // Goblins: 2 melee + 1 ranged
	string(EncounterBanditBasic): {SquadTypeMelee, SquadTypeRanged, SquadTypeRanged}, // Bandits: 1 melee + 2 ranged
	string(EncounterBeastBasic):  {SquadTypeMelee, SquadTypeMelee, SquadTypeMelee},   // Beasts: 3 melee (swarm)
	string(EncounterOrcBasic):    {SquadTypeMelee, SquadTypeRanged, SquadTypeMagic},  // Orcs: balanced composition
}
