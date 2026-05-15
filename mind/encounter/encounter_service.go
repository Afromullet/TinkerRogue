package encounter

import (
	"fmt"
	"time"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	rstr "game_main/tactical/squads/roster"

	"github.com/bytearena/ecs"
)

// maxHistoryEntries caps the size of the recent-encounter history ring.
const maxHistoryEntries = 10

// EncounterService coordinates encounter lifecycle and tracks history.
// This is an HONEST coordinator - it doesn't own everything, but it provides:
// - Encounter flow coordination (validation, state tracking, mode transitions)
// - Encounter state tracking (activeEncounter)
// - History recording (last maxHistoryEntries encounters)
// - Analytics (win rate, last encounter)
//
// What it DOESN'T do (handled by other systems):
// - Create encounter entities (TriggerCombatFromThreat handles this)
// - Setup combat (SetupBalancedEncounter in this package handles this)
// - Mark threats defeated (handled via resolvers in ExitCombat)
type EncounterService struct {
	manager         *common.EntityManager
	modeCoordinator ModeCoordinator

	// Current encounter tracking
	activeEncounter *ActiveEncounter

	// History tracking
	history    []*CompletedEncounter
	maxHistory int

	// postCombatCallback is called after ExitCombat finishes processing.
	// Registered/unregistered by external systems (e.g., RaidRunner) to receive combat results.
	// resolution is the output of the setup.Resolver (nil if no resolver ran).
	//
	// SINGLE-SUBSCRIBER BY DESIGN. Do not add a second consumer without first promoting
	// this field to a slice — last-write-wins semantics would silently drop one of them.
	// All combat types flow through this channel, so subscribers must filter for the
	// combat types they care about (see RaidRunner's raidEntityID + RaidActive guard).
	postCombatCallback func(reason combatlifecycle.CombatExitReason, outcome *combatlifecycle.EncounterOutcome, resolution *combatlifecycle.ResolutionResult)
}

// NewEncounterService creates a new encounter coordinator
func NewEncounterService(
	manager *common.EntityManager,
	modeCoordinator ModeCoordinator,
) *EncounterService {
	return &EncounterService{
		manager:         manager,
		modeCoordinator: modeCoordinator,
		activeEncounter: nil,
		history:         make([]*CompletedEncounter, 0, maxHistoryEntries),
		maxHistory:      maxHistoryEntries,
	}
}

// === QUERY METHODS ===

// IsEncounterActive returns true if an encounter is currently in progress
func (es *EncounterService) IsEncounterActive() bool {
	return es.activeEncounter != nil
}

// GetCurrentEncounterID returns the currently active encounter ID (0 if none)
func (es *EncounterService) GetCurrentEncounterID() ecs.EntityID {
	if es.activeEncounter != nil {
		return es.activeEncounter.Setup.EncounterID
	}
	return 0
}

// GetRosterOwnerID returns the commander entity ID for the current encounter.
// Returns 0 if no encounter is active.
func (es *EncounterService) GetRosterOwnerID() ecs.EntityID {
	if es.activeEncounter != nil {
		return es.activeEncounter.Setup.RosterOwnerID
	}
	return 0
}

// GetEnemySquadIDs returns the enemy squad IDs for the current encounter (for cleanup coordination)
func (es *EncounterService) GetEnemySquadIDs() []ecs.EntityID {
	if es.activeEncounter != nil {
		return es.activeEncounter.Setup.EnemySquadIDs
	}
	return nil
}

// SetPostCombatCallback sets a callback to receive combat results after ExitCombat completes.
// Only one callback is supported at a time (last call wins).
// The callback receives the exit reason, outcome, and the resolution result (nil if no resolver ran).
func (es *EncounterService) SetPostCombatCallback(fn func(combatlifecycle.CombatExitReason, *combatlifecycle.EncounterOutcome, *combatlifecycle.ResolutionResult)) {
	es.postCombatCallback = fn
}

// ClearPostCombatCallback removes the post-combat callback.
func (es *EncounterService) ClearPostCombatCallback() {
	es.postCombatCallback = nil
}

// ExitCombat is the single unified exit point for all combat endings.
// All paths (victory, defeat, flee) MUST use this method.
// Snapshots the active encounter, delegates orchestration to
// combatlifecycle.ExecuteCombatExit, records history during the orchestration's
// history hook, then fires the post-combat callback for external listeners.
func (es *EncounterService) ExitCombat(
	reason combatlifecycle.CombatExitReason,
	result *combatlifecycle.EncounterOutcome,
	teardown combatlifecycle.CombatTeardown,
) {
	if es.activeEncounter == nil {
		return
	}

	// Snapshot per-encounter state before orchestration. PlayerSquadIDs is read
	// from the roster up front so the resolver sees a stable list even after
	// activeEncounter is cleared by the history hook.
	enc := *es.activeEncounter
	ctx := combatlifecycle.ResolutionContext{
		Setup:                  enc.Setup,
		Reason:                 reason,
		Outcome:                result,
		PlayerEntityID:         enc.PlayerEntityID,
		PlayerSquadIDs:         es.collectPlayerSquadIDs(enc.Setup.RosterOwnerID),
		OriginalPlayerPosition: enc.OriginalPlayerPosition,
	}

	hooks := NewEncounterExitHooks(es.manager, es.modeCoordinator)

	onHistory := func(resolution *combatlifecycle.ResolutionResult) {
		completed := &CompletedEncounter{
			EncounterID:     enc.Setup.EncounterID,
			ThreatID:        enc.Setup.ThreatID,
			ThreatName:      enc.Setup.ThreatName,
			PlayerPosition:  enc.Setup.CombatPosition,
			StartTime:       enc.StartTime,
			EndTime:         time.Now(),
			Duration:        time.Since(enc.StartTime),
			Outcome:         reason,
			RoundsCompleted: result.RoundsCompleted,
			VictorFaction:   result.VictorFaction,
			VictorName:      result.VictorName,
		}
		es.history = append(es.history, completed)
		if len(es.history) > es.maxHistory {
			es.history = es.history[len(es.history)-es.maxHistory:]
		}
		es.activeEncounter = nil
	}

	resolution := combatlifecycle.ExecuteCombatExit(es.manager, ctx, hooks, teardown, onHistory)

	if es.postCombatCallback != nil {
		es.postCombatCallback(reason, result, resolution)
	}
}

// TransitionToCombat performs the shared combat mode transition.
// Called by combatlifecycle.ExecuteCombatStart after Prepare() succeeds.
// Satisfies combat.CombatTransitioner via structural typing.
func (es *EncounterService) TransitionToCombat(setup *combatlifecycle.CombatSetup) error {
	if es.IsEncounterActive() {
		return fmt.Errorf("encounter already in progress")
	}

	if es.modeCoordinator == nil {
		return fmt.Errorf("mode coordinator unavailable for combat transition")
	}

	// Save player's original position before teleporting to encounter
	pos := es.modeCoordinator.GetPlayerPosition()
	if pos == nil {
		return fmt.Errorf("player position unavailable for combat transition")
	}
	originalPlayerPos := *pos

	// Setup tactical state for GUI handoff to CombatMode
	es.modeCoordinator.SetTriggeredEncounterID(setup.EncounterID)
	es.modeCoordinator.ResetTacticalState()

	// Set post-combat return mode if specified (e.g., "raid").
	// Derived from setup.Type rather than a separate field.
	if mode := setup.PostCombatReturnMode(); mode != "" {
		es.modeCoordinator.SetPostCombatReturnMode(mode)
	}

	// Move player camera to encounter position so map zooms correctly
	*pos = setup.CombatPosition

	// Transition to combat mode
	if err := es.modeCoordinator.EnterCombatMode(); err != nil {
		return fmt.Errorf("failed to enter combat mode: %w", err)
	}

	playerEntityID := es.modeCoordinator.GetPlayerEntityID()

	es.activeEncounter = &ActiveEncounter{
		Setup:                  *setup,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		PlayerEntityID:         playerEntityID,
	}

	return nil
}

// collectPlayerSquadIDs reads the commander's owned-squad list from the roster.
// Used to populate ResolutionContext.PlayerSquadIDs in ExitCombat before the
// activeEncounter snapshot is cleared. Returns nil when the roster is missing
// or empty (e.g., garrison defense, debug encounters).
func (es *EncounterService) collectPlayerSquadIDs(rosterOwnerID ecs.EntityID) []ecs.EntityID {
	roster := rstr.GetPlayerSquadRoster(rosterOwnerID, es.manager)
	if roster != nil && len(roster.OwnedSquads) > 0 {
		return roster.OwnedSquads
	}
	return nil
}

