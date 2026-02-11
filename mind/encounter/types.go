package encounter

import (
	"time"

	"game_main/common"
	"game_main/gui/framework"
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

// ModeCoordinator defines the interface for switching game modes and accessing state
type ModeCoordinator interface {
	GetBattleMapState() *framework.BattleMapState
	EnterBattleMap(mode string) error
	GetPlayerData() *common.PlayerData
}

// CombatExitReason describes why combat ended
type CombatExitReason int

const (
	ExitVictory CombatExitReason = iota
	ExitDefeat
	ExitFlee
)

// String returns a human-readable name for the exit reason
func (r CombatExitReason) String() string {
	switch r {
	case ExitVictory:
		return "Victory"
	case ExitDefeat:
		return "Defeat"
	case ExitFlee:
		return "Fled"
	default:
		return "Unknown"
	}
}

// CombatResult captures the combat outcome for the exit pipeline.
// Built by the GUI layer from CombatService.CheckVictoryCondition().
type CombatResult struct {
	IsPlayerVictory  bool
	VictorFaction    ecs.EntityID
	VictorName       string
	RoundsCompleted  int
	DefeatedFactions []ecs.EntityID
}

// CombatCleaner handles entity disposal when exiting combat.
// Implemented by CombatService (satisfies via Go structural typing, no import needed).
type CombatCleaner interface {
	CleanupCombat(enemySquadIDs []ecs.EntityID)
}

// ActiveEncounter holds context for the currently active encounter
type ActiveEncounter struct {
	// Core identification
	EncounterID ecs.EntityID
	ThreatID    ecs.EntityID
	ThreatName  string

	// Positioning
	PlayerPosition         coords.LogicalPosition // Encounter location (where combat happens)
	OriginalPlayerPosition coords.LogicalPosition // Player's original location (to restore after combat)

	// Timing
	StartTime time.Time

	// Combat tracking (for cleanup coordination)
	EnemySquadIDs  []ecs.EntityID
	PlayerEntityID ecs.EntityID

	// Garrison defense tracking
	IsGarrisonDefense bool         // True if defending a garrisoned node
	DefendedNodeID    ecs.EntityID // Node being defended (0 if not garrison defense)
}

// CompletedEncounter represents a finished encounter for history tracking
type CompletedEncounter struct {
	EncounterID    ecs.EntityID
	ThreatID       ecs.EntityID
	ThreatName     string
	PlayerPosition coords.LogicalPosition

	// Timing
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Outcome
	Outcome         CombatExitReason
	RoundsCompleted int
	VictorFaction   ecs.EntityID
	VictorName      string
}

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
