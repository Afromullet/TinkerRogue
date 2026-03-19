package encounter

import (
	"fmt"
	"time"

	"game_main/common"
	"game_main/mind/combatlifecycle"
	"game_main/overworld/core"
	"game_main/tactical/combat"

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
	postCombatCallback func(combat.CombatExitReason, *combat.EncounterOutcome)
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

// SetPostCombatCallback sets a callback to receive combat results after ExitCombat completes.
// Only one callback is supported at a time (last call wins).
func (es *EncounterService) SetPostCombatCallback(fn func(combat.CombatExitReason, *combat.EncounterOutcome)) {
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
	reason combat.CombatExitReason,
	result *combat.EncounterOutcome,
	combatCleaner combat.CombatCleaner,
) {
	if es.activeEncounter == nil {
		return
	}

	// Snapshot encounter before RecordEncounterCompletion clears activeEncounter.
	// All cleanup steps reference this snapshot so new fields don't get missed.
	enc := *es.activeEncounter

	// Step 1: Resolve combat outcome based on type + reason
	switch reason {
	case combat.ExitVictory, combat.ExitDefeat:
		if enc.Type != combat.CombatTypeRaid {
			es.resolveEncounterOutcome(&enc, result.IsPlayerVictory)
		}
	case combat.ExitFlee:
		es.restoreEncounterSprite(enc.EncounterID)
		_, encounterData := es.getEncounterData(enc.EncounterID)
		if encounterData != nil && encounterData.ThreatNodeID != 0 {
			resolver := &FleeResolver{ThreatNodeID: encounterData.ThreatNodeID}
			combatlifecycle.ExecuteResolution(es.manager, resolver)
		}
	}

	// Step 2: Mark encounter defeated on victory (non-raid)
	if result.IsPlayerVictory && enc.Type != combat.CombatTypeRaid {
		es.markEncounterDefeated(enc.EncounterID)
	}

	// Step 3: Restore player to original position (before they were teleported to encounter)
	if es.modeCoordinator != nil {
		if pos := es.modeCoordinator.GetPlayerPosition(); pos != nil {
			*pos = enc.OriginalPlayerPosition
		}
	}

	// Step 4: Record history
	es.RecordEncounterCompletion(reason, result.VictorFaction,
		result.VictorName, result.RoundsCompleted)

	// Step 5: Clean up all combat entities
	if combatCleaner != nil {
		if enc.Type == combat.CombatTypeGarrisonDefense && result.IsPlayerVictory {
			es.returnGarrisonSquadsToNode(enc.DefendedNodeID)
		}
		combatCleaner.CleanupCombat(enc.EnemySquadIDs)
	}

	// Step 6: Notify external listeners (e.g., RaidRunner)
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
		EncounterID:            setup.EncounterID,
		ThreatID:               setup.ThreatID,
		ThreatName:             setup.ThreatName,
		PlayerPosition:         setup.CombatPosition,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		EnemySquadIDs:          setup.EnemySquadIDs,
		RosterOwnerID:          setup.RosterOwnerID,
		PlayerEntityID:         playerEntityID,
		Type:                   setup.Type,
		DefendedNodeID:         setup.DefendedNodeID,
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

// addToHistory adds a completed encounter to the history
func (es *EncounterService) addToHistory(completed *CompletedEncounter) {
	es.history = append(es.history, completed)

	// Trim history if exceeds max
	if len(es.history) > es.maxHistory {
		es.history = es.history[len(es.history)-es.maxHistory:]
	}
}
