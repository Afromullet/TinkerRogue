package guicombat

import (
	"fmt"

	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatservices"
	"game_main/tactical/squads"
	"game_main/world/coords"
	"game_main/world/encounter"

	"github.com/bytearena/ecs"
)

// CombatLifecycleManager handles combat initialization and cleanup operations.
// Extracted from CombatMode to separate lifecycle concerns from UI management.
type CombatLifecycleManager struct {
	ecsManager     *common.EntityManager
	queries        *framework.GUIQueries
	combatService  *combatservices.CombatService
	logManager     *CombatLogManager
	combatLogArea  *widgets.CachedTextAreaWrapper
	battleRecorder *battlelog.BattleRecorder
}

// NewCombatLifecycleManager creates a new combat lifecycle manager
func NewCombatLifecycleManager(
	ecsManager *common.EntityManager,
	queries *framework.GUIQueries,
	combatService *combatservices.CombatService,
	logManager *CombatLogManager,
	combatLogArea *widgets.CachedTextAreaWrapper,
) *CombatLifecycleManager {
	return &CombatLifecycleManager{
		ecsManager:    ecsManager,
		queries:       queries,
		combatService: combatService,
		logManager:    logManager,
		combatLogArea: combatLogArea,
	}
}

// SetBattleRecorder sets the battle recorder for export functionality
func (clm *CombatLifecycleManager) SetBattleRecorder(recorder *battlelog.BattleRecorder) {
	clm.battleRecorder = recorder
}

// SetupEncounter initializes a combat encounter by spawning entities
// Returns the encounter ID and any error encountered
func (clm *CombatLifecycleManager) SetupEncounter(
	encounterID ecs.EntityID,
	playerStartPos coords.LogicalPosition,
) (ecs.EntityID, error) {
	clm.logManager.UpdateTextArea(clm.combatLogArea, "Fresh combat encounter - spawning entities")

	// Log encounter info if available
	if encounterID != 0 {
		entity := clm.ecsManager.FindEntityByID(encounterID)
		if entity != nil {
			encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
				entity,
				encounter.OverworldEncounterComponent,
			)
			if encounterData != nil {
				clm.logManager.UpdateTextArea(clm.combatLogArea,
					fmt.Sprintf("Encounter: %s (Level %d)", encounterData.Name, encounterData.Level))
			}
		}
	}

	// Call SetupGameplayFactions to create combat entities
	if err := combat.SetupGameplayFactions(clm.ecsManager, playerStartPos); err != nil {
		clm.logManager.UpdateTextArea(clm.combatLogArea, fmt.Sprintf("Error spawning combat entities: %v", err))
		return 0, fmt.Errorf("failed to setup gameplay factions: %w", err)
	}

	return encounterID, nil
}

// InitializeCombatFactions collects all factions and initializes combat
// Returns the list of faction IDs and any error encountered
func (clm *CombatLifecycleManager) InitializeCombatFactions() ([]ecs.EntityID, error) {
	// Collect all factions using query service
	factionIDs := clm.queries.GetAllFactions()

	// Initialize combat with all factions
	if len(factionIDs) > 0 {
		if err := clm.combatService.InitializeCombat(factionIDs); err != nil {
			clm.logManager.UpdateTextArea(clm.combatLogArea, fmt.Sprintf("Error initializing combat: %v", err))
			return nil, err
		}

		// Log initial faction
		currentFactionID := clm.combatService.TurnManager.GetCurrentFaction()
		factionData := clm.queries.CombatCache.FindFactionDataByID(currentFactionID, clm.queries.ECSManager)
		factionName := "Unknown"
		if factionData != nil {
			factionName = factionData.Name
		}
		clm.logManager.UpdateTextArea(clm.combatLogArea, fmt.Sprintf("Round 1: %s goes first!", factionName))
	} else {
		clm.logManager.UpdateTextArea(clm.combatLogArea, "No factions found - combat cannot start")
	}

	return factionIDs, nil
}

// StartBattleRecording initializes battle recording if enabled
func (clm *CombatLifecycleManager) StartBattleRecording(round int) {
	if config.ENABLE_COMBAT_LOG_EXPORT && clm.battleRecorder != nil {
		clm.battleRecorder.SetEnabled(true)
		clm.battleRecorder.Start()
		clm.battleRecorder.SetCurrentRound(round)
	}
}

// MarkEncounterDefeated marks an encounter as defeated if the player won
func (clm *CombatLifecycleManager) MarkEncounterDefeated(encounterID ecs.EntityID) {
	// Only mark if we have a tracked encounter
	if encounterID == 0 {
		return
	}

	// Check victory condition
	victor := clm.combatService.CheckVictoryCondition()

	// Only mark as defeated if a player faction won
	if victor.VictorFaction != 0 {
		factionData := clm.queries.CombatCache.FindFactionDataByID(victor.VictorFaction, clm.queries.ECSManager)
		if factionData != nil && factionData.IsPlayerControlled {
			// Player won - mark encounter as defeated
			entity := clm.ecsManager.FindEntityByID(encounterID)
			if entity != nil {
				encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
					entity,
					encounter.OverworldEncounterComponent,
				)
				if encounterData != nil {
					encounterData.IsDefeated = true
					fmt.Printf("Marked encounter '%s' as defeated\n", encounterData.Name)
					clm.logManager.UpdateTextArea(clm.combatLogArea,
						fmt.Sprintf("Encounter '%s' defeated!", encounterData.Name))
				}
			}
		}
	}
}

// CleanupCombatEntities removes ALL combat entities when returning to exploration
// This is idempotent and safe to call multiple times
func (clm *CombatLifecycleManager) CleanupCombatEntities() {
	fmt.Println("Cleaning up combat entities")

	// Step 1: Collect IDs of combat squads (those being removed)
	combatSquadIDs := make(map[ecs.EntityID]bool)
	for _, result := range clm.ecsManager.World.Query(squads.SquadTag) {
		entity := result.Entity
		// Combat squads have CombatFactionComponent
		if entity.HasComponent(combat.CombatFactionComponent) {
			combatSquadIDs[entity.GetID()] = true
		}
	}

	fmt.Printf("Found %d combat squads to remove\n", len(combatSquadIDs))

	// Step 2: Remove all faction entities
	for _, result := range clm.ecsManager.World.Query(combat.FactionTag) {
		entity := result.Entity
		clm.ecsManager.World.DisposeEntities(entity)
	}

	// Step 3: Remove ONLY combat squads (those with CombatFactionComponent)
	// This preserves exploration squads which don't have this component
	for _, result := range clm.ecsManager.World.Query(squads.SquadTag) {
		entity := result.Entity

		// CRITICAL: Only remove squads that belong to factions (combat squads)
		// Exploration squads don't have CombatFactionComponent and should be preserved
		if !entity.HasComponent(combat.CombatFactionComponent) {
			fmt.Printf("Preserving exploration squad: %d\n", entity.GetID())
			continue // Skip exploration squads
		}

		fmt.Printf("Removing combat squad: %d\n", entity.GetID())

		// Remove from position system
		if entity.HasComponent(common.PositionComponent) {
			posData := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			if posData != nil {
				common.GlobalPositionSystem.RemoveEntity(entity.GetID(), *posData)
			}
		}

		// Dispose entity
		clm.ecsManager.World.DisposeEntities(entity)
	}

	// Step 4: Remove ONLY unit entities that belong to combat squads
	// This preserves units in exploration squads
	for _, result := range clm.ecsManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity

		// Get unit's squad ID
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData != nil {
			// Only remove if this unit belongs to a combat squad
			if combatSquadIDs[memberData.SquadID] {
				fmt.Printf("Removing combat unit from squad %d\n", memberData.SquadID)
				clm.ecsManager.World.DisposeEntities(entity)
			} else {
				fmt.Printf("Preserving exploration unit from squad %d\n", memberData.SquadID)
			}
		}
	}

	// Step 5: Remove all action state entities
	for _, result := range clm.ecsManager.World.Query(combat.ActionStateTag) {
		entity := result.Entity
		clm.ecsManager.World.DisposeEntities(entity)
	}

	// Step 6: Remove turn state entity
	for _, result := range clm.ecsManager.World.Query(combat.TurnStateTag) {
		entity := result.Entity
		clm.ecsManager.World.DisposeEntities(entity)
	}

	// Step 7: Clear all caches
	clm.queries.MarkAllSquadsDirty()

	fmt.Println("Combat entities cleanup complete")
}

// ExportBattleLog exports the battle log to disk if recording is enabled
func (clm *CombatLifecycleManager) ExportBattleLog() error {
	if !config.ENABLE_COMBAT_LOG_EXPORT || clm.battleRecorder == nil || !clm.battleRecorder.IsEnabled() {
		return nil
	}

	victor := clm.combatService.CheckVictoryCondition()
	victoryInfo := &battlelog.VictoryInfo{
		RoundsCompleted: victor.RoundsCompleted,
		VictorFaction:   victor.VictorFaction,
		VictorName:      victor.VictorName,
	}
	record := clm.battleRecorder.Finalize(victoryInfo)
	if err := battlelog.ExportBattleJSON(record, config.COMBAT_LOG_EXPORT_DIR); err != nil {
		return fmt.Errorf("failed to export combat log: %w", err)
	}

	clm.battleRecorder.Clear()
	clm.battleRecorder.SetEnabled(false)
	return nil
}
