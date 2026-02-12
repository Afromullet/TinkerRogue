package encounter

import (
	"fmt"
	"math"
	"time"

	"game_main/common"
	"game_main/overworld/core"
	"game_main/overworld/garrison"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
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

// StartEncounter coordinates encounter initialization and spawns enemies.
// Caller must create the encounter entity first (via TriggerCombatFromThreat).
//
// This method:
// 1. Validates no encounter is active
// 2. Validates encounter entity exists
// 3. Spawns enemies and hides encounter sprite
// 4. Tracks encounter context (including enemy squad IDs)
// 5. Sets TacticalState for combat mode handoff
// 6. Transitions to combat mode
func (es *EncounterService) StartEncounter(
	encounterID ecs.EntityID,
	threatID ecs.EntityID,
	threatName string,
	playerPos coords.LogicalPosition,
	playerEntityID ecs.EntityID,
) error {
	// Validate no active encounter
	if es.IsEncounterActive() {
		return fmt.Errorf("encounter already in progress")
	}

	if encounterID == 0 {
		return fmt.Errorf("invalid encounter ID: 0")
	}

	// Validate encounter entity exists
	encounterEntity, encounterData := es.getEncounterData(encounterID)
	if encounterEntity == nil {
		return fmt.Errorf("encounter entity %d not found", encounterID)
	}
	if encounterData == nil {
		return fmt.Errorf("encounter %d missing core.OverworldEncounterData", encounterID)
	}

	fmt.Printf("EncounterService: Starting encounter %d (%s)\n", encounterID, threatName)

	// Hide encounter sprite during combat
	renderable := common.GetComponentType[*rendering.Renderable](
		encounterEntity,
		rendering.RenderableComponent,
	)
	if renderable != nil {
		renderable.Visible = false
		fmt.Println("Hiding overworld encounter sprite during combat")
	}

	// Spawn enemies using balanced encounter system
	fmt.Println("Starting combat encounter - spawning entities")
	enemySquadIDs, err := SpawnCombatEntities(es.manager, playerEntityID, playerPos, encounterData, encounterID)
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return fmt.Errorf("failed to spawn enemies: %w", err)
	}
	fmt.Printf("Spawned %d enemy squads: %v\n", len(enemySquadIDs), enemySquadIDs)

	// Handle mode transition (save position, set battle state, enter combat)
	originalPlayerPos, err := es.beginCombatTransition(encounterID, playerPos)
	if err != nil {
		if renderable != nil {
			renderable.Visible = true
		}
		return err
	}

	// Track active encounter with combat data
	es.activeEncounter = &ActiveEncounter{
		EncounterID:            encounterID,
		ThreatID:               threatID,
		ThreatName:             threatName,
		PlayerPosition:         playerPos,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		EnemySquadIDs:          enemySquadIDs,
		PlayerEntityID:         playerEntityID,
	}

	fmt.Printf("EncounterService: Encounter %d started, entering combat\n", encounterID)
	return nil
}

// StartGarrisonDefense initiates combat where the player's garrison squads defend a node
// against an attacking faction. The garrison squads become the player's combat forces,
// and attacker squads are generated from power budget.
func (es *EncounterService) StartGarrisonDefense(
	encounterID ecs.EntityID,
	targetNodeID ecs.EntityID,
	playerEntityID ecs.EntityID,
) error {
	if es.IsEncounterActive() {
		return fmt.Errorf("encounter already in progress")
	}

	if encounterID == 0 {
		return fmt.Errorf("invalid encounter ID: 0")
	}

	// Validate encounter entity
	encounterEntity, encounterData := es.getEncounterData(encounterID)
	if encounterEntity == nil || encounterData == nil {
		return fmt.Errorf("encounter entity %d not found or missing data", encounterID)
	}

	// Get garrison data
	garrisonData := garrison.GetGarrisonAtNode(es.manager, targetNodeID)
	if garrisonData == nil || len(garrisonData.SquadIDs) == 0 {
		return fmt.Errorf("no garrison at node %d", targetNodeID)
	}

	// Get node position for combat
	nodeEntity := es.manager.FindEntityByID(targetNodeID)
	if nodeEntity == nil {
		return fmt.Errorf("node entity %d not found", targetNodeID)
	}
	nodePos := common.GetComponentType[*coords.LogicalPosition](nodeEntity, common.PositionComponent)
	if nodePos == nil {
		return fmt.Errorf("node %d has no position", targetNodeID)
	}

	fmt.Printf("EncounterService: Starting garrison defense at node %d\n", targetNodeID)

	// Create factions
	cache := combat.NewCombatQueryCache(es.manager)
	fm := combat.NewCombatFactionManager(es.manager, cache)
	playerFactionID := fm.CreateFactionWithPlayer("Garrison Defense", 1, "Player 1", encounterID)
	enemyFactionID := fm.CreateFactionWithPlayer("Attacking Forces", 0, "", encounterID)

	// Add garrison squads to player faction (they defend)
	garrisonPositions := generatePositionsAroundPoint(*nodePos, len(garrisonData.SquadIDs), -math.Pi/2, math.Pi/2, PlayerMinDistance, PlayerMaxDistance)
	for i, squadID := range garrisonData.SquadIDs {
		pos := garrisonPositions[i]
		if err := fm.AddSquadToFaction(playerFactionID, squadID, pos); err != nil {
			return fmt.Errorf("failed to add garrison squad %d: %w", squadID, err)
		}
		ensureUnitPositions(es.manager, squadID, pos)
		combat.CreateActionStateForSquad(es.manager, squadID)

		// Mark squad as deployed for combat
		squadData := common.GetComponentTypeByID[*squads.SquadData](es.manager, squadID, squads.SquadComponent)
		if squadData != nil {
			squadData.IsDeployed = true
		}
	}

	// Generate attacker squads from power budget
	spec, err := GenerateEncounterSpec(es.manager, playerEntityID, *nodePos, encounterData)
	if err != nil {
		return fmt.Errorf("failed to generate attacker spec: %w", err)
	}

	enemySquadIDs := make([]ecs.EntityID, 0, len(spec.EnemySquads))
	for i, enemySpec := range spec.EnemySquads {
		if err := fm.AddSquadToFaction(enemyFactionID, enemySpec.SquadID, enemySpec.Position); err != nil {
			return fmt.Errorf("failed to add enemy squad %d: %w", i, err)
		}
		combat.CreateActionStateForSquad(es.manager, enemySpec.SquadID)
		enemySquadIDs = append(enemySquadIDs, enemySpec.SquadID)
	}

	// Handle mode transition (save position, set battle state, enter combat)
	originalPlayerPos, err := es.beginCombatTransition(encounterID, *nodePos)
	if err != nil {
		return err
	}

	// Track active encounter
	es.activeEncounter = &ActiveEncounter{
		EncounterID:            encounterID,
		ThreatID:               targetNodeID,
		ThreatName:             encounterData.Name,
		PlayerPosition:         *nodePos,
		OriginalPlayerPosition: originalPlayerPos,
		StartTime:              time.Now(),
		EnemySquadIDs:          enemySquadIDs,
		PlayerEntityID:         playerEntityID,
		IsGarrisonDefense:      true,
		DefendedNodeID:         targetNodeID,
	}

	fmt.Printf("EncounterService: Garrison defense started at node %d\n", targetNodeID)
	return nil
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

	// Handle garrison defense resolution
	if es.activeEncounter != nil && es.activeEncounter.IsGarrisonDefense {
		es.resolveGarrisonDefense(isPlayerVictory, encounterData)
	} else if encounterData.ThreatNodeID != 0 {
		// Apply standard combat resolution to overworld
		es.resolveCombatToOverworld(
			encounterData.ThreatNodeID,
			isPlayerVictory,
			victorFaction,
			defeatedFactions,
			roundsCompleted,
		)
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
		es.EndEncounter(result.IsPlayerVictory, result.VictorFaction,
			result.VictorName, result.RoundsCompleted, result.DefeatedFactions)
	case ExitFlee:
		es.RestoreEncounterSprite()
		es.resolveFleeToOverworld()
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
