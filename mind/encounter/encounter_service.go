package encounter

import (
	"fmt"
	"time"

	"game_main/common"
	"game_main/gui/framework"
	"game_main/overworld/core"
	owencounter "game_main/overworld/overworldencounter"
	"game_main/tactical/combat"
	"game_main/tactical/combatresolution"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ModeCoordinator defines the interface for switching game modes and accessing state
type ModeCoordinator interface {
	GetBattleMapState() *framework.BattleMapState
	EnterBattleMap(mode string) error
	GetPlayerData() *common.PlayerData
}

// EncounterOutcome represents the result of an encounter
type EncounterOutcome int

const (
	Victory EncounterOutcome = iota // Player won
	Defeat                          // Player lost
	Fled                            // Player fled (not implemented yet)
)

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
	Outcome         EncounterOutcome
	RoundsCompleted int
	VictorFaction   ecs.EntityID
	VictorName      string
}

// EncounterService coordinates encounter lifecycle and tracks history.
// This is an HONEST coordinator - it doesn't own everything, but it provides:
// - Encounter flow coordination (validation, state tracking, mode transitions)
// - Encounter state tracking (activeEncounter)
// - History recording (last 10 encounters)
// - Analytics (win rate, last encounter)
//
// What it DOESN'T do (handled by other systems):
// - Create encounter entities (caller does this via overworld package)
// - Setup combat (CombatService does this via SetupBalancedEncounter)
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
// Caller must create the encounter entity first (via overworld.TriggerCombatFromThreat).
//
// This method:
// 1. Validates no encounter is active
// 2. Validates encounter entity exists
// 3. Spawns enemies and hides encounter sprite
// 4. Tracks encounter context (including enemy squad IDs)
// 5. Sets BattleMapState for combat mode handoff
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
	encounterEntity := es.manager.FindEntityByID(encounterID)
	if encounterEntity == nil {
		return fmt.Errorf("encounter entity %d not found", encounterID)
	}

	encounterData := common.GetComponentType[*core.OverworldEncounterData](encounterEntity, core.OverworldEncounterComponent)
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
	enemySquadIDs, err := SetupBalancedEncounter(es.manager, playerEntityID, playerPos, encounterData, encounterID)
	if err != nil {
		// Rollback sprite hiding on spawn failure
		if renderable != nil {
			renderable.Visible = true
		}
		return fmt.Errorf("failed to spawn enemies: %w", err)
	}
	fmt.Printf("Spawned %d enemy squads: %v\n", len(enemySquadIDs), enemySquadIDs)

	// Save player's original position before teleporting to encounter
	originalPlayerPos := coords.LogicalPosition{X: 50, Y: 40} // Default if PlayerData unavailable
	if es.modeCoordinator != nil {
		playerData := es.modeCoordinator.GetPlayerData()
		if playerData != nil && playerData.Pos != nil {
			originalPlayerPos = *playerData.Pos
			fmt.Printf("Saved original player position (%d,%d) before teleporting to encounter\n",
				originalPlayerPos.X, originalPlayerPos.Y)
		}
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

	// Set BattleMapState for GUI handoff to CombatMode
	if es.modeCoordinator != nil {
		battleMapState := es.modeCoordinator.GetBattleMapState()
		battleMapState.TriggeredEncounterID = encounterID
		battleMapState.Reset() // Reset UI state for new encounter

		// Move player camera to encounter position so map zooms correctly
		playerData := es.modeCoordinator.GetPlayerData()
		if playerData != nil && playerData.Pos != nil {
			*playerData.Pos = playerPos
			fmt.Printf("Updated player position to encounter location (%d,%d)\n", playerPos.X, playerPos.Y)
		}
	}

	// Transition to combat mode
	if es.modeCoordinator != nil {
		if err := es.modeCoordinator.EnterBattleMap("combat"); err != nil {
			// Rollback on failure
			es.activeEncounter = nil
			if renderable != nil {
				renderable.Visible = true
			}
			return fmt.Errorf("failed to enter combat mode: %w", err)
		}
	}

	fmt.Printf("EncounterService: Encounter %d started, entering combat\n", encounterID)
	return nil
}

// RecordEncounterCompletion records the encounter outcome to history.
// This does NOT handle resolution - CombatService handles that.
// This just tracks what happened for analytics/debugging.
func (es *EncounterService) RecordEncounterCompletion(
	isPlayerVictory bool,
	victorFaction ecs.EntityID,
	victorName string,
	roundsCompleted int,
) {
	if es.activeEncounter == nil {
		fmt.Println("WARNING: RecordEncounterCompletion called with no active encounter")
		return
	}

	// Determine outcome
	var outcome EncounterOutcome
	if isPlayerVictory {
		outcome = Victory
	} else {
		outcome = Defeat
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
		Outcome:         outcome,
		RoundsCompleted: roundsCompleted,
		VictorFaction:   victorFaction,
		VictorName:      victorName,
	}

	es.addToHistory(completed)

	// Restore player to original position (before they were teleported to encounter)
	if es.modeCoordinator != nil {
		playerData := es.modeCoordinator.GetPlayerData()
		if playerData != nil && playerData.Pos != nil {
			originalPos := es.activeEncounter.OriginalPlayerPosition
			*playerData.Pos = originalPos
			fmt.Printf("Restored player position to original location (%d,%d)\n",
				originalPos.X, originalPos.Y)
		}
	}

	// Clear active encounter
	es.activeEncounter = nil

	fmt.Printf("EncounterService: Recorded %s after %d rounds (%.1fs)\n",
		outcome, roundsCompleted, completed.Duration.Seconds())
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

	// Get encounter entity
	entity := es.manager.FindEntityByID(encounterID)
	if entity == nil {
		fmt.Printf("WARNING: Encounter entity %d not found during EndEncounter\n", encounterID)
		return
	}

	encounterData := common.GetComponentType[*core.OverworldEncounterData](
		entity,
		core.OverworldEncounterComponent,
	)
	if encounterData == nil {
		fmt.Printf("WARNING: Encounter %d missing core.OverworldEncounterData\n", encounterID)
		return
	}

	// Apply combat resolution to overworld if this came from a threat
	if encounterData.ThreatNodeID != 0 {
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

// resolveCombatToOverworld applies combat outcome to overworld threat state
func (es *EncounterService) resolveCombatToOverworld(
	threatNodeID ecs.EntityID,
	playerVictory bool,
	victorFaction ecs.EntityID,
	defeatedFactions []ecs.EntityID,
	roundsCompleted int,
) {
	if es.activeEncounter == nil {
		return
	}

	// Calculate casualties
	playerUnitsLost, enemyUnitsKilled := es.calculateCasualties(victorFaction, defeatedFactions)

	// Get player squad ID (use first from roster)
	playerSquadID := es.getFirstPlayerSquadID()

	// Calculate rewards from threat
	threatEntity := es.manager.FindEntityByID(threatNodeID)
	if threatEntity == nil {
		fmt.Printf("WARNING: Threat node %d not found for resolution\n", threatNodeID)
		return
	}

	threatData := common.GetComponentType[*core.ThreatNodeData](threatEntity, core.ThreatNodeComponent)
	if threatData == nil {
		fmt.Printf("WARNING: Entity %d is not a threat node\n", threatNodeID)
		return
	}

	rewards := owencounter.CalculateRewards(threatData.Intensity, threatData.ThreatType)

	// Create combat outcome
	outcome := combatresolution.CreateCombatOutcome(
		threatNodeID,
		playerVictory,
		false, // playerRetreat - not implemented yet
		playerSquadID,
		playerUnitsLost,
		enemyUnitsKilled,
		rewards,
	)

	// Apply to overworld
	if err := combatresolution.ResolveCombatToOverworld(es.manager, outcome); err != nil {
		fmt.Printf("ERROR resolving combat to overworld: %v\n", err)
	} else {
		fmt.Printf("Combat resolved to overworld: %d enemy killed, %d player lost\n",
			enemyUnitsKilled, playerUnitsLost)
	}
}

// calculateCasualties counts units killed in combat
func (es *EncounterService) calculateCasualties(
	victorFaction ecs.EntityID,
	defeatedFactions []ecs.EntityID,
) (playerUnitsLost int, enemyUnitsKilled int) {
	if es.activeEncounter == nil {
		return 0, 0
	}

	// Find player and enemy factions from combat
	// Note: We rely on faction data that still exists during this call
	playerFactionID := ecs.EntityID(0)
	enemyFactionID := ecs.EntityID(0)

	// Find player and enemy factions
	// The factions still exist at this point (cleaned up after this call)
	for _, result := range es.manager.World.Query(combat.FactionTag) {
		entity := result.Entity
		factionData := common.GetComponentType[*combat.FactionData](entity, combat.CombatFactionComponent)
		if factionData != nil {
			if factionData.IsPlayerControlled {
				playerFactionID = entity.GetID()
			} else {
				enemyFactionID = entity.GetID()
			}
		}
	}

	// Count dead units in each faction
	for _, result := range es.manager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData == nil {
			continue
		}

		// Get squad to check faction membership
		squadEntity := es.manager.FindEntityByID(memberData.SquadID)
		if squadEntity == nil {
			continue
		}

		squadFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.FactionMembershipComponent)
		if squadFaction == nil {
			continue
		}

		// Check if unit is dead
		unitAttr := common.GetComponentType[*common.Attributes](entity, common.AttributeComponent)
		if unitAttr != nil && unitAttr.CurrentHealth <= 0 {
			if squadFaction.FactionID == playerFactionID {
				playerUnitsLost++
			} else if squadFaction.FactionID == enemyFactionID {
				enemyUnitsKilled++
			}
		}
	}

	return playerUnitsLost, enemyUnitsKilled
}

// getFirstPlayerSquadID returns the first player squad ID found
func (es *EncounterService) getFirstPlayerSquadID() ecs.EntityID {
	if es.activeEncounter == nil {
		return 0
	}

	roster := squads.GetPlayerSquadRoster(es.activeEncounter.PlayerEntityID, es.manager)
	if roster != nil && len(roster.OwnedSquads) > 0 {
		return roster.OwnedSquads[0]
	}
	return 0
}

// RestoreEncounterSprite restores the encounter sprite visibility when fleeing combat.
// This allows the player to re-engage with the encounter later.
func (es *EncounterService) RestoreEncounterSprite() {
	if es.activeEncounter == nil {
		return
	}

	encounterID := es.activeEncounter.EncounterID
	entity := es.manager.FindEntityByID(encounterID)
	if entity == nil {
		return
	}

	// Only restore if not already defeated
	encounterData := common.GetComponentType[*core.OverworldEncounterData](
		entity,
		core.OverworldEncounterComponent,
	)
	if encounterData != nil && !encounterData.IsDefeated {
		renderable := common.GetComponentType[*rendering.Renderable](
			entity,
			rendering.RenderableComponent,
		)
		if renderable != nil {
			renderable.Visible = true
			fmt.Println("Restoring overworld encounter sprite after fleeing")
		}
	}
}

// === PRIVATE HELPER METHODS ===

// addToHistory adds a completed encounter to the history
func (es *EncounterService) addToHistory(completed *CompletedEncounter) {
	es.history = append(es.history, completed)

	// Trim history if exceeds max
	if len(es.history) > es.maxHistory {
		es.history = es.history[len(es.history)-es.maxHistory:]
	}
}
