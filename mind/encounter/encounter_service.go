package encounter

import (
	"fmt"
	"time"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/overworld/core"
	"game_main/tactical/combat"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// EncounterService coordinates encounter lifecycle and tracks history.
// This is an HONEST coordinator - it doesn't own everything, but it provides:
// - Encounter flow coordination (validation, state tracking, mode transitions)
// - Encounter state tracking (activeEncounter)
// - History recording (last 10 encounters)
// - Analytics (win rate, last encounter)
//
// What it DOESN'T do (handled by other systems):
// - Create encounter entities (TriggerCombatFromThreat handles this)
// - Setup combat (SetupBalancedEncounter in this package handles this)
// - Resolve combat outcomes (CombatService does this)
// - Mark threats defeated (CombatService does this)
type EncounterService struct {
	manager         *common.EntityManager
	modeCoordinator CombatTransitionHandler

	// Current encounter tracking
	activeEncounter *ActiveEncounter

	// History tracking
	history    []*CompletedEncounter
	maxHistory int

	// postCombatCallback is called after ExitCombat finishes processing.
	// Registered/unregistered by external systems (e.g., RaidRunner) to receive combat results.
	postCombatCallback func(combat.CombatExitReason, *combat.EncounterOutcome)
}

// NewEncounterService creates a new encounter coordinator
func NewEncounterService(
	manager *common.EntityManager,
	modeCoordinator CombatTransitionHandler,
) *EncounterService {
	return &EncounterService{
		manager:         manager,
		modeCoordinator: modeCoordinator,
		activeEncounter: nil,
		history:         make([]*CompletedEncounter, 0, 10),
		maxHistory:      10, // Keep last 10 encounters
	}
}

// RecordEncounterCompletion records the encounter outcome to history.
// This does NOT handle resolution - CombatService handles that.
// This just tracks what happened for analytics/debugging.
func (es *EncounterService) RecordEncounterCompletion(
	reason combat.CombatExitReason,
	victorFaction ecs.EntityID,
	victorName string,
	roundsCompleted int,
) {
	if es.activeEncounter == nil {
		fmt.Println("WARNING: RecordEncounterCompletion called with no active encounter")
		return
	}

	// Create history record
	completed := &CompletedEncounter{
		EncounterID:     es.activeEncounter.EncounterID,
		ThreatID:        es.activeEncounter.ThreatID,
		ThreatName:      es.activeEncounter.ThreatName,
		PlayerPosition:  es.activeEncounter.PlayerPosition,
		StartTime:       es.activeEncounter.StartTime,
		EndTime:         time.Now(),
		Duration:        time.Since(es.activeEncounter.StartTime),
		Outcome:         reason,
		RoundsCompleted: roundsCompleted,
		VictorFaction:   victorFaction,
		VictorName:      victorName,
	}

	es.addToHistory(completed)

	// Restore player to original position (before they were teleported to encounter)
	if pos := es.modeCoordinator.GetPlayerPosition(); pos != nil {
		originalPos := es.activeEncounter.OriginalPlayerPosition
		*pos = originalPos
	}

	// Clear active encounter
	es.activeEncounter = nil

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

// RegisterPostCombatListener sets a callback to receive combat results after ExitCombat completes.
// Only one listener is supported at a time (last registration wins).
func (es *EncounterService) RegisterPostCombatListener(fn func(combat.CombatExitReason, *combat.EncounterOutcome)) {
	es.postCombatCallback = fn
}

// UnregisterPostCombatListener removes the post-combat listener.
func (es *EncounterService) UnregisterPostCombatListener() {
	es.postCombatCallback = nil
}

// ExitCombat is the single unified exit point for all combat endings.
// All paths (victory, defeat, flee) MUST use this method.
// Handles resolution, history recording, cleanup, and listener notification.
func (es *EncounterService) ExitCombat(
	reason combat.CombatExitReason,
	result *combat.EncounterOutcome,
	combatCleaner combat.CombatCleaner,
) {
	if es.activeEncounter == nil {
		return
	}

	// Snapshot encounter state before RecordEncounterCompletion clears it
	encounter := es.activeEncounter
	enemySquadIDs := encounter.EnemySquadIDs
	combatType := encounter.Type
	defendedNodeID := encounter.DefendedNodeID

	// Step 1: Resolve combat outcome based on type + reason
	switch reason {
	case combat.ExitVictory, combat.ExitDefeat:
		if combatType != combat.CombatTypeRaid {
			es.resolveEncounterOutcome(encounter, result.IsPlayerVictory)
		}
	case combat.ExitFlee:
		es.restoreEncounterSprite(encounter.EncounterID)
		_, encounterData := es.getEncounterData(encounter.EncounterID)
		if encounterData != nil && encounterData.ThreatNodeID != 0 {
			resolver := &FleeResolver{ThreatNodeID: encounterData.ThreatNodeID}
			combatlifecycle.ExecuteResolution(es.manager, resolver)
		}
	}

	// Step 2: Mark encounter defeated on victory (non-raid)
	if result.IsPlayerVictory && combatType != combat.CombatTypeRaid {
		es.markEncounterDefeated(encounter.EncounterID)
	}

	// Step 3: Record history + restore player position
	es.RecordEncounterCompletion(reason, result.VictorFaction,
		result.VictorName, result.RoundsCompleted)

	// Step 4: Clean up all combat entities
	if combatCleaner != nil {
		if combatType == combat.CombatTypeGarrisonDefense && result.IsPlayerVictory {
			es.returnGarrisonSquadsToNode(defendedNodeID)
		}
		combatCleaner.CleanupCombat(enemySquadIDs)
	}

	// Step 5: Notify external listeners (e.g., RaidRunner)
	if es.postCombatCallback != nil {
		es.postCombatCallback(reason, result)
	}
}

// resolveEncounterOutcome dispatches to the correct resolver based on combat type.
func (es *EncounterService) resolveEncounterOutcome(encounter *ActiveEncounter, isPlayerVictory bool) {
	_, encounterData := es.getEncounterData(encounter.EncounterID)
	if encounterData == nil {
		fmt.Printf("WARNING: Encounter entity %d not found during resolution\n", encounter.EncounterID)
		return
	}

	switch encounter.Type {
	case combat.CombatTypeGarrisonDefense:
		resolver := &GarrisonDefenseResolver{
			PlayerVictory:        isPlayerVictory,
			DefendedNodeID:       encounter.DefendedNodeID,
			AttackingFactionType: encounterData.AttackingFactionType,
		}
		combatlifecycle.ExecuteResolution(es.manager, resolver)
	case combat.CombatTypeOverworld:
		if encounterData.ThreatNodeID != 0 {
			resolver := &OverworldCombatResolver{
				ThreatNodeID:   encounterData.ThreatNodeID,
				PlayerVictory:  isPlayerVictory,
				PlayerEntityID: encounter.PlayerEntityID,
				PlayerSquadIDs: es.getAllPlayerSquadIDs(),
				EnemySquadIDs:  encounter.EnemySquadIDs,
			}
			combatlifecycle.ExecuteResolution(es.manager, resolver)
		}
	// CombatTypeDebug: no resolution needed
	}
}

// markEncounterDefeated marks the encounter as defeated and hides its sprite permanently.
func (es *EncounterService) markEncounterDefeated(encounterID ecs.EntityID) {
	entity, encounterData := es.getEncounterData(encounterID)
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
func (es *EncounterService) restoreEncounterSprite(encounterID ecs.EntityID) {
	entity, encounterData := es.getEncounterData(encounterID)
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
func (es *EncounterService) TransitionToCombat(setup *combat.CombatSetup) error {
	if es.IsEncounterActive() {
		return fmt.Errorf("encounter already in progress")
	}

	originalPlayerPos, err := es.beginCombatTransition(setup.EncounterID, setup.CombatPosition)
	if err != nil {
		return err
	}

	// Set post-combat return mode if specified (e.g., "raid")
	if setup.PostCombatReturnMode != "" && es.modeCoordinator != nil {
		es.modeCoordinator.SetPostCombatReturnMode(setup.PostCombatReturnMode)
	}

	playerEntityID := ecs.EntityID(0)
	if es.modeCoordinator != nil {
		playerEntityID = es.modeCoordinator.GetPlayerEntityID()
	}

	es.activeEncounter = &ActiveEncounter{
		EncounterID:            setup.EncounterID,
		ThreatID:               setup.ThreatID,
		ThreatName:             setup.ThreatName,
		PlayerPosition:         setup.CombatPosition,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		EnemySquadIDs:          setup.EnemySquadIDs,
		RosterOwnerID:          setup.RosterOwnerID,
		PlayerEntityID:         playerEntityID,
		PlayerFactionID:        setup.PlayerFactionID,
		EnemyFactionID:         setup.EnemyFactionID,
		Type:                   setup.Type,
		DefendedNodeID:         setup.DefendedNodeID,
	}

	return nil
}

// === PRIVATE HELPER METHODS ===

// beginCombatTransition saves the player's original position, sets up battle state,
// moves the player camera to combatPos, and enters combat mode.
// Returns the saved original position for use in ActiveEncounter.
func (es *EncounterService) beginCombatTransition(encounterID ecs.EntityID, combatPos coords.LogicalPosition) (coords.LogicalPosition, error) {
	// Save player's original position before teleporting to encounter
	pos := es.modeCoordinator.GetPlayerPosition()
	if pos == nil {
		return coords.LogicalPosition{}, fmt.Errorf("player position unavailable for combat transition")
	}
	originalPlayerPos := *pos

	if es.modeCoordinator != nil {
		// Setup tactical state for GUI handoff to CombatMode
		es.modeCoordinator.SetTriggeredEncounterID(encounterID)
		es.modeCoordinator.ResetTacticalState()

		// Move player camera to encounter position so map zooms correctly
		if pos := es.modeCoordinator.GetPlayerPosition(); pos != nil {
			*pos = combatPos
		}

		// Transition to combat mode
		if err := es.modeCoordinator.EnterCombatMode(); err != nil {
			return originalPlayerPos, fmt.Errorf("failed to enter combat mode: %w", err)
		}
	}

	return originalPlayerPos, nil
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

// addToHistory adds a completed encounter to the history
func (es *EncounterService) addToHistory(completed *CompletedEncounter) {
	es.history = append(es.history, completed)

	// Trim history if exceeds max
	if len(es.history) > es.maxHistory {
		es.history = es.history[len(es.history)-es.maxHistory:]
	}
}
