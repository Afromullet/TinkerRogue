package encounter

import (
	"fmt"
	"time"

	"game_main/core/common"
	"game_main/mind/combatlifecycle"
	"game_main/campaign/overworld/core"

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
		return es.activeEncounter.EncounterID
	}
	return 0
}

// GetRosterOwnerID returns the commander entity ID for the current encounter.
// Returns 0 if no encounter is active.
func (es *EncounterService) GetRosterOwnerID() ecs.EntityID {
	if es.activeEncounter != nil {
		return es.activeEncounter.RosterOwnerID
	}
	return 0
}

// GetEnemySquadIDs returns the enemy squad IDs for the current encounter (for cleanup coordination)
func (es *EncounterService) GetEnemySquadIDs() []ecs.EntityID {
	if es.activeEncounter != nil {
		return es.activeEncounter.EnemySquadIDs
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
// Handles resolution, history recording, cleanup, and listener notification.
func (es *EncounterService) ExitCombat(
	reason combatlifecycle.CombatExitReason,
	result *combatlifecycle.EncounterOutcome,
	teardown combatlifecycle.CombatTeardown,
) {
	if es.activeEncounter == nil {
		return
	}

	// Snapshot encounter before we clear activeEncounter.
	// All cleanup steps reference this snapshot so new fields don't get missed.
	enc := *es.activeEncounter

	// Single lookup of the encounter entity + data; reused by all steps below.
	encounterEntity, encounterData := es.getEncounterData(enc.EncounterID)

	// Step 1: Resolve combat outcome via the setup's resolver (built eagerly by the starter).
	// One dispatch path for all combat types. Nil resolver means no resolution (debug encounters).
	var resolution *combatlifecycle.ResolutionResult
	if enc.Resolver != nil {
		ctx := combatlifecycle.ResolutionContext{
			Reason:         reason,
			PlayerVictory:  result.IsPlayerVictory,
			PlayerEntityID: enc.PlayerEntityID,
			PlayerSquadIDs: es.getAllPlayerSquadIDs(),
		}
		resolution = combatlifecycle.ExecuteResolution(es.manager, enc.Resolver, ctx)
	}

	// Step 1b: Flee restores the encounter sprite (overworld-only effect; no-op otherwise).
	if reason == combatlifecycle.ExitFlee {
		restoreEncounterSprite(encounterEntity, encounterData)
	}

	// Step 2: Mark encounter defeated on overworld victory.
	// markEncounterDefeated is null-safe: it no-ops for raid (no OverworldEncounterData on the raid entity).
	if result.IsPlayerVictory && enc.Type != combatlifecycle.CombatTypeRaid {
		markEncounterDefeated(encounterEntity, encounterData)
	}

	// Step 3: Restore player to original position (before they were teleported to encounter)
	if es.modeCoordinator != nil {
		if pos := es.modeCoordinator.GetPlayerPosition(); pos != nil {
			*pos = enc.OriginalPlayerPosition
		}
	}

	// Step 4: Record history from the snapshot, then clear activeEncounter.
	completed := &CompletedEncounter{
		EncounterID:     enc.EncounterID,
		ThreatID:        enc.ThreatID,
		ThreatName:      enc.ThreatName,
		PlayerPosition:  enc.CombatPosition,
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

	// Step 5: Tear down all combat entities.
	// TeardownCombat internally strips combat-only state from player squads
	// (faction membership, perk round state, positions, IsDeployed) — the
	// caller does not need to follow up.
	if teardown != nil {
		if enc.Type == combatlifecycle.CombatTypeGarrisonDefense && result.IsPlayerVictory {
			es.returnGarrisonSquadsToNode(enc.DefendedNodeID)
		}
		teardown.TeardownCombat(enc.EnemySquadIDs)
	}

	// Step 6: Notify external listeners (e.g., RaidRunner)
	if es.postCombatCallback != nil {
		es.postCombatCallback(reason, result, resolution)
	}
}

// markEncounterDefeated marks the encounter as defeated and hides its sprite permanently.
func markEncounterDefeated(entity *ecs.Entity, encounterData *core.OverworldEncounterData) {
	if entity == nil || encounterData == nil {
		return
	}

	encounterData.IsDefeated = true

	renderable := common.GetComponentType[*common.Renderable](entity, common.RenderableComponent)
	if renderable != nil {
		renderable.Visible = false
	}
}

// restoreEncounterSprite restores the encounter sprite visibility when fleeing combat.
func restoreEncounterSprite(entity *ecs.Entity, encounterData *core.OverworldEncounterData) {
	if entity == nil || encounterData == nil || encounterData.IsDefeated {
		return
	}

	renderable := common.GetComponentType[*common.Renderable](entity, common.RenderableComponent)
	if renderable != nil {
		renderable.Visible = true
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

	// Set post-combat return mode if specified (e.g., "raid")
	if setup.PostCombatReturnMode != "" {
		es.modeCoordinator.SetPostCombatReturnMode(setup.PostCombatReturnMode)
	}

	// Move player camera to encounter position so map zooms correctly
	*pos = setup.CombatPosition

	// Transition to combat mode
	if err := es.modeCoordinator.EnterCombatMode(); err != nil {
		return fmt.Errorf("failed to enter combat mode: %w", err)
	}

	playerEntityID := es.modeCoordinator.GetPlayerEntityID()

	es.activeEncounter = &ActiveEncounter{
		CombatSetup:            *setup,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		PlayerEntityID:         playerEntityID,
	}

	return nil
}

// getEncounterData looks up an encounter entity and its OverworldEncounterData.
// Returns (nil, nil) if either the entity or the component is missing.
func (es *EncounterService) getEncounterData(encounterID ecs.EntityID) (*ecs.Entity, *core.OverworldEncounterData) {
	entity := es.manager.FindEntityByID(encounterID)
	if entity == nil {
		return nil, nil
	}
	data := common.GetComponentType[*core.OverworldEncounterData](entity, core.OverworldEncounterComponent)
	return entity, data
}

