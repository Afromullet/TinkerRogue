// Package generation builds enemy squads and spawn positions for overworld encounters.
// It owns the power-budget math, squad-composition rules, and position-placement helpers.
// Encounter orchestration (lifecycle, triggers, starters) lives in the parent encounter package
// and imports this package.
package spawning

import (
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// Squad type identifiers for composition control.
const (
	SquadTypeMelee   = "melee"
	SquadTypeRanged  = "ranged"
	SquadTypeMagic   = "magic"
	SquadTypeSupport = "support"
)

// Position generation constants.
const (
	EnemySpacingDistance = 10 // Distance from player for enemy squad spawns
	PlayerMinDistance    = 3  // Minimum distance for player squad positioning
	PlayerMaxDistance    = 4  // Maximum distance for player squad positioning
)

// Squad creation constants (non-difficulty-dependent).
const (
	PowerThreshold          = 0.95 // Stop adding units at 95% target power
	LeadershipAttributeBase = 20   // Base leadership for squad leaders
)

// DefaultPowerProfile selects the default weighted power config.
const DefaultPowerProfile = "Balanced"

// enemySquadSpec describes a single enemy squad to create. Used internally to
// stage results before flattening into parallel id/position slices.
type enemySquadSpec struct {
	SquadID  ecs.EntityID
	Position coords.LogicalPosition
	Power    float64
	Type     string
	Name     string
}
