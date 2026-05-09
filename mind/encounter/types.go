package encounter

import (
	"time"

	"game_main/mind/combatlifecycle"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
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

// ActiveEncounter holds context for the currently active encounter.
// CombatSetup fields (EncounterID, ThreatID, ThreatName, EnemySquadIDs, RosterOwnerID,
// Type, DefendedNodeID, SkipServiceResolution, BuildResolver, etc.) are promoted.
// CombatPosition (promoted) is the encounter location where combat takes place.
type ActiveEncounter struct {
	combatlifecycle.CombatSetup

	// Player's original location before teleporting to the encounter (restored after combat).
	OriginalPlayerPosition coords.LogicalPosition

	// Timing
	StartTime time.Time

	// PlayerEntityID is the player entity (owns resource stockpile).
	// Populated by TransitionToCombat after the mode coordinator provides it.
	PlayerEntityID ecs.EntityID
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

// EncounterController is the GUI's command/query port onto EncounterService.
// EncounterService structurally satisfies this interface — pass it directly.
// Not a callback mechanism; for post-combat listener hooks see
// EncounterService.SetPostCombatCallback.
type EncounterController interface {
	ExitCombat(reason combatlifecycle.CombatExitReason, result *combatlifecycle.EncounterOutcome, teardown combatlifecycle.CombatTeardown)
	GetRosterOwnerID() ecs.EntityID
	GetCurrentEncounterID() ecs.EntityID
}
