package encounter

import (
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// EncounterSpec describes what to create for an encounter.
// Pure data structure - no combat references.
// This allows encounter generation to be decoupled from combat setup.
type EncounterSpec struct {
	PlayerSquadIDs []ecs.EntityID      // Player's deployed squads
	EnemySquads    []EnemySquadSpec    // Enemy squads to create
	Difficulty     int                 // Encounter difficulty level
	EncounterType  string              // Type of encounter (goblin, bandit, etc.)
	PlayerStartPos coords.LogicalPosition
}

// EnemySquadSpec describes a single enemy squad to create.
type EnemySquadSpec struct {
	SquadID  ecs.EntityID          // ID of created squad (filled after generation)
	Position coords.LogicalPosition // Where to spawn the squad
	Power    float64               // Target power level
	Type     string                // Squad archetype (melee, ranged, magic)
	Name     string                // Squad display name
}
