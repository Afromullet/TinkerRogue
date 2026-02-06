package encounter

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Squad type identifiers for composition control
const (
	SquadTypeMelee  = "melee"
	SquadTypeRanged = "ranged"
	SquadTypeMagic  = "magic"
)

// Position generation constants
const (
	EnemySpacingDistance = 10 // Distance from player for enemy squad spawns
	PlayerMinDistance    = 3  // Minimum distance for player squad positioning
	PlayerMaxDistance    = 4  // Maximum distance for player squad positioning
)

// Squad creation constants (non-difficulty-dependent)
const (
	PowerThreshold          = 0.95 // Stop adding units at 95% target power
	LeadershipAttributeBase = 20   // Base leadership for squad leaders
)

// Power profile constant
const (
	DefaultPowerProfile = "Balanced" // Default power calculation profile
)

// EncounterSpec describes what to create for an encounter.
// Pure data structure - no combat references.
// This allows encounter generation to be decoupled from combat setup.
type EncounterSpec struct {
	PlayerSquadIDs []ecs.EntityID   // Player's deployed squads
	EnemySquads    []EnemySquadSpec // Enemy squads to create
	Difficulty     int              // Encounter difficulty level
	EncounterType  string           // Type of encounter (goblin, bandit, etc.)
	PlayerStartPos coords.LogicalPosition
}

// EnemySquadSpec describes a single enemy squad to create.
type EnemySquadSpec struct {
	SquadID  ecs.EntityID           // ID of created squad (filled after generation)
	Position coords.LogicalPosition // Where to spawn the squad
	Power    float64                // Target power level
	Type     string                 // Squad archetype (melee, ranged, magic)
	Name     string                 // Squad display name
}
