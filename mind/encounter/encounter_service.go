package encounter

import (
	"fmt"
	"time"

	"game_main/common"
	"game_main/mind/combatpipeline"
	"game_main/overworld/core"
	"game_main/visual/rendering"
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
	modeCoordinator ModeCoordinator

	// Current encounter tracking
	activeEncounter *ActiveEncounter

	// History tracking
	history    []*CompletedEncounter
	maxHistory int

	// PostCombatCallback is called after ExitCombat finishes processing.
	// Set by external systems (e.g., RaidRunner) to receive combat results.
	PostCombatCallback func(CombatExitReason, *CombatResult)
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
	reason CombatExitReason,
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
	if pos := es.getPlayerPosition(); pos != nil {
		originalPos := es.activeEncounter.OriginalPlayerPosition
		*pos = originalPos
		fmt.Printf("Restored player position to original location (%d,%d)\n",
			originalPos.X, originalPos.Y)
	}

	// Clear active encounter
	es.activeEncounter = nil

	fmt.Printf("EncounterService: Recorded %s after %d rounds (%.1fs)\n",
		reason, roundsCompleted, completed.Duration.Seconds())
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

// EndEncounter marks an encounter as defeated if the player won and applies combat resolution.
// This should be called after combat concludes but before cleanup.
func (es *EncounterService) EndEncounter(
	isPlayerVictory bool,
	victorFaction ecs.EntityID,
	victorName string,
	roundsCompleted int,
	defeatedFactions []ecs.EntityID,
) {
	// Only process if we have a tracked encounter
	if es.activeEncounter == nil {
		fmt.Println("WARNING: EndEncounter called with no active encounter")
		return
	}

	encounterID := es.activeEncounter.EncounterID

	entity, encounterData := es.getEncounterData(encounterID)
	if entity == nil || encounterData == nil {
		fmt.Printf("WARNING: Encounter entity %d not found or missing data during EndEncounter\n", encounterID)
		return
	}

	// Handle resolution via unified pipeline
	if es.activeEncounter.IsGarrisonDefense {
		resolver := &GarrisonDefenseResolver{
			PlayerVictory:        isPlayerVictory,
			DefendedNodeID:       es.activeEncounter.DefendedNodeID,
			AttackingFactionType: encounterData.AttackingFactionType,
		}
		combatpipeline.ExecuteResolution(es.manager, resolver)
	} else if encounterData.ThreatNodeID != 0 {
		resolver := &OverworldCombatResolver{
			ThreatNodeID:   encounterData.ThreatNodeID,
			PlayerVictory:  isPlayerVictory,
			PlayerEntityID: es.activeEncounter.PlayerEntityID,
			PlayerSquadIDs: es.getAllPlayerSquadIDs(),
			EnemySquadIDs:  es.activeEncounter.EnemySquadIDs,
		}
		combatpipeline.ExecuteResolution(es.manager, resolver)
	}

	// Only mark as defeated if player won
	if isPlayerVictory {
		// Mark encounter as defeated and hide permanently
		encounterData.IsDefeated = true

		// Hide encounter sprite permanently on overworld map
		renderable := common.GetComponentType[*rendering.Renderable](
			entity,
			rendering.RenderableComponent,
		)
		if renderable != nil {
			renderable.Visible = false
		}

		fmt.Printf("Marked encounter '%s' as defeated\n", encounterData.Name)
	}
}

// RestoreEncounterSprite restores the encounter sprite visibility when fleeing combat.
// This allows the player to re-engage with the encounter later.
func (es *EncounterService) RestoreEncounterSprite() {
	if es.activeEncounter == nil {
		return
	}

	entity, encounterData := es.getEncounterData(es.activeEncounter.EncounterID)
	if entity == nil || encounterData == nil || encounterData.IsDefeated {
		return
	}

	renderable := common.GetComponentType[*rendering.Renderable](
		entity,
		rendering.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = true
		fmt.Println("Restoring overworld encounter sprite after fleeing")
	}
}

// ExitCombat is the single unified exit point for all combat endings.
// All paths (victory, defeat, flee) MUST use this method.
func (es *EncounterService) ExitCombat(
	reason CombatExitReason,
	result *CombatResult,
	combatCleaner CombatCleaner,
) {
	if es.activeEncounter == nil {
		return
	}

	// Capture before RecordEncounterCompletion clears activeEncounter
	enemySquadIDs := es.activeEncounter.EnemySquadIDs
	isGarrisonDefense := es.activeEncounter.IsGarrisonDefense
	defendedNodeID := es.activeEncounter.DefendedNodeID

	// Step 1: Resolve combat outcome to overworld
	switch reason {
	case ExitVictory, ExitDefeat:
		if !es.activeEncounter.IsRaidCombat {
			es.EndEncounter(result.IsPlayerVictory, result.VictorFaction,
				result.VictorName, result.RoundsCompleted, result.DefeatedFactions)
		}
	case ExitFlee:
		es.RestoreEncounterSprite()
		// Flee resolver
		_, encounterData := es.getEncounterData(es.activeEncounter.EncounterID)
		if encounterData != nil && encounterData.ThreatNodeID != 0 {
			resolver := &FleeResolver{ThreatNodeID: encounterData.ThreatNodeID}
			combatpipeline.ExecuteResolution(es.manager, resolver)
		}
	}

	// Step 2: Record history + restore player position
	es.RecordEncounterCompletion(reason, result.VictorFaction,
		result.VictorName, result.RoundsCompleted)

	// Step 3: Clean up all combat entities
	if combatCleaner != nil {
		// For garrison defense victories, garrison squads need special handling
		// (returned to garrison instead of disposed)
		if isGarrisonDefense && result.IsPlayerVictory {
			es.returnGarrisonSquadsToNode(defendedNodeID)
		}
		combatCleaner.CleanupCombat(enemySquadIDs)
	}

	// Step 4: Notify external listeners (e.g., RaidRunner)
	if es.PostCombatCallback != nil {
		es.PostCombatCallback(reason, result)
	}
}

// TransitionToCombat performs the shared combat mode transition.
// Called by combatpipeline.ExecuteCombatStart after Prepare() succeeds.
// Satisfies combatpipeline.CombatTransitioner via structural typing.
func (es *EncounterService) TransitionToCombat(setup *combatpipeline.CombatSetup) error {
	if es.IsEncounterActive() {
		return fmt.Errorf("encounter already in progress")
	}

	originalPlayerPos, err := es.beginCombatTransition(setup.EncounterID, setup.CombatPosition)
	if err != nil {
		return err
	}

	// Set post-combat return mode if specified (e.g., "raid")
	if setup.PostCombatReturnMode != "" && es.modeCoordinator != nil {
		tacticalState := es.modeCoordinator.GetTacticalState()
		tacticalState.PostCombatReturnMode = setup.PostCombatReturnMode
	}

	playerEntityID := ecs.EntityID(0)
	if es.modeCoordinator != nil {
		if pd := es.modeCoordinator.GetPlayerData(); pd != nil {
			playerEntityID = pd.PlayerEntityID
		}
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
		IsGarrisonDefense:      setup.IsGarrisonDefense,
		DefendedNodeID:         setup.DefendedNodeID,
		IsRaidCombat:           setup.IsRaidCombat,
	}

	return nil
}

// === PRIVATE HELPER METHODS ===

// beginCombatTransition saves the player's original position, sets up battle state,
// moves the player camera to combatPos, and enters combat mode.
// Returns the saved original position for use in ActiveEncounter.
func (es *EncounterService) beginCombatTransition(encounterID ecs.EntityID, combatPos coords.LogicalPosition) (coords.LogicalPosition, error) {
	// Save player's original position before teleporting to encounter
	originalPlayerPos := coords.LogicalPosition{X: 50, Y: 40} // Default if PlayerData unavailable
	if pos := es.getPlayerPosition(); pos != nil {
		originalPlayerPos = *pos
	}

	if es.modeCoordinator != nil {
		// Setup tactical state for GUI handoff to CombatMode
		tacticalState := es.modeCoordinator.GetTacticalState()
		tacticalState.TriggeredEncounterID = encounterID
		tacticalState.Reset()

		// Move player camera to encounter position so map zooms correctly
		if pos := es.getPlayerPosition(); pos != nil {
			*pos = combatPos
		}

		// Transition to combat mode
		if err := es.modeCoordinator.EnterTactical("combat"); err != nil {
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

// getPlayerPosition returns the player's current position, or nil if unavailable.
// This centralizes the nil-check pattern for modeCoordinator -> PlayerData -> Pos.
func (es *EncounterService) getPlayerPosition() *coords.LogicalPosition {
	if es.modeCoordinator == nil {
		return nil
	}
	playerData := es.modeCoordinator.GetPlayerData()
	if playerData == nil || playerData.Pos == nil {
		return nil
	}
	return playerData.Pos
}

// addToHistory adds a completed encounter to the history
func (es *EncounterService) addToHistory(completed *CompletedEncounter) {
	es.history = append(es.history, completed)

	// Trim history if exceeds max
	if len(es.history) > es.maxHistory {
		es.history = es.history[len(es.history)-es.maxHistory:]
	}
}
