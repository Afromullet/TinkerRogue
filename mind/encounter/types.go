package encounter

import (
	"time"

	"game_main/mind/combatlifecycle"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// Squad type identifiers for composition control
const (
	SquadTypeMelee   = "melee"
	SquadTypeRanged  = "ranged"
	SquadTypeMagic   = "magic"
	SquadTypeSupport = "support"
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

// Combat resolution constants
const (
	EnemiesPerIntensityPoint = 5 // Every 5 enemies killed = 1 intensity reduction
	DefeatIntensityGrowth    = 1 // Threat grows by 1 intensity on player defeat
)

// ModeCoordinator defines the narrow interface for encounter→combat mode transitions.
// Decoupled from GUI types — uses only primitives and coords.
type ModeCoordinator interface {
	SetPostCombatReturnMode(mode string)
	SetTriggeredEncounterID(id ecs.EntityID)
	ResetTacticalState()
	EnterCombatMode() error
	GetPlayerEntityID() ecs.EntityID
	GetPlayerPosition() *coords.LogicalPosition
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
	RosterOwnerID  ecs.EntityID // Commander entity (owns squad roster)
	PlayerEntityID ecs.EntityID // Player entity (owns resource stockpile)

	// Combat type (overworld, garrison defense, raid, debug)
	Type           combatlifecycle.CombatType
	DefendedNodeID ecs.EntityID // Node being defended (0 if not garrison defense)

	// SkipServiceResolution is true when resolution is handled by an external callback
	// (e.g., RaidRunner) rather than EncounterService itself.
	SkipServiceResolution bool
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
	Outcome         combatlifecycle.CombatExitReason
	RoundsCompleted int
	VictorFaction   ecs.EntityID
	VictorName      string
}

// SpawnResult holds the output of SpawnCombatEntities.
type SpawnResult struct {
	EnemySquadIDs   []ecs.EntityID
	PlayerFactionID ecs.EntityID
	EnemyFactionID  ecs.EntityID
}

// EnemySquadSpec describes a single enemy squad to create.
type EnemySquadSpec struct {
	SquadID  ecs.EntityID           // ID of created squad (filled after generation)
	Position coords.LogicalPosition // Where to spawn the squad
	Power    float64                // Target power level
	Type     string                 // Squad archetype (melee, ranged, magic)
	Name     string                 // Squad display name
}
