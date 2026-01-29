package guicombat

import (
	"fmt"

	"game_main/common"
	"game_main/config"
	"game_main/gui/framework"
	"game_main/gui/widgets"
	"game_main/mind/encounter"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatresolution"
	"game_main/tactical/combatservices"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/overworld"

	"github.com/bytearena/ecs"
)

// TODO, this can probably be moved outside of the guicombat package
// CombatLifecycleManager handles combat initialization and cleanup operations.
// Extracted from CombatMode to separate lifecycle concerns from UI management.
type CombatLifecycleManager struct {
	ecsManager     *common.EntityManager
	queries        *framework.GUIQueries
	combatService  *combatservices.CombatService
	logManager     *CombatLogManager
	combatLogArea  *widgets.CachedTextAreaWrapper
	battleRecorder *battlelog.BattleRecorder
	playerEntityID ecs.EntityID // Player entity ID from PlayerData
	// Track enemy squads created during combat for explicit cleanup
	enemySquadIDs []ecs.EntityID
}

// NewCombatLifecycleManager creates a new combat lifecycle manager
func NewCombatLifecycleManager(
	ecsManager *common.EntityManager,
	queries *framework.GUIQueries,
	combatService *combatservices.CombatService,
	logManager *CombatLogManager,
	combatLogArea *widgets.CachedTextAreaWrapper,
	playerEntityID ecs.EntityID,
) *CombatLifecycleManager {
	return &CombatLifecycleManager{
		ecsManager:     ecsManager,
		queries:        queries,
		combatService:  combatService,
		logManager:     logManager,
		combatLogArea:  combatLogArea,
		playerEntityID: playerEntityID,
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

	// Clear previous enemy squad tracking
	clm.enemySquadIDs = []ecs.EntityID{}

	// Extract encounter data to pass to balanced spawner
	var encounterData *encounter.OverworldEncounterData
	if encounterID != 0 {
		entity := clm.ecsManager.FindEntityByID(encounterID)
		if entity != nil {
			encounterData = common.GetComponentType[*encounter.OverworldEncounterData](
				entity,
				encounter.OverworldEncounterComponent,
			)
			if encounterData != nil {
				clm.logManager.UpdateTextArea(clm.combatLogArea,
					fmt.Sprintf("Encounter: %s (Level %d)", encounterData.Name, encounterData.Level))
			}

			// Hide encounter sprite during combat
			renderable := common.GetComponentType[*rendering.Renderable](
				entity,
				rendering.RenderableComponent,
			)
			if renderable != nil {
				renderable.Visible = false
				fmt.Println("Hiding overworld encounter sprite during combat")
			}
		}
	}

	// Call SetupBalancedEncounter for power-based enemy spawning
	enemySquadIDs, err := encounter.SetupBalancedEncounter(clm.ecsManager, clm.playerEntityID, playerStartPos, encounterData)
	if err != nil {
		clm.logManager.UpdateTextArea(clm.combatLogArea, fmt.Sprintf("Error spawning combat entities: %v", err))
		return 0, fmt.Errorf("failed to setup balanced encounter: %w", err)
	}

	// Store enemy squad IDs for cleanup
	clm.enemySquadIDs = enemySquadIDs
	fmt.Printf("Tracking %d enemy squads for cleanup: %v\n", len(clm.enemySquadIDs), clm.enemySquadIDs)

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
// and applies combat resolution to overworld threats
func (clm *CombatLifecycleManager) MarkEncounterDefeated(encounterID ecs.EntityID) {
	// Only mark if we have a tracked encounter
	if encounterID == 0 {
		return
	}

	// Check victory condition
	victor := clm.combatService.CheckVictoryCondition()

	// Get encounter data
	entity := clm.ecsManager.FindEntityByID(encounterID)
	if entity == nil {
		return
	}

	encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
		entity,
		encounter.OverworldEncounterComponent,
	)
	if encounterData == nil {
		return
	}

	// Apply combat resolution to overworld if this came from a threat
	if encounterData.ThreatNodeID != 0 {
		clm.resolveCombatToOverworld(encounterData.ThreatNodeID, victor)
	}

	// Only mark as defeated if a player faction won
	//TODO, we need to remove it rather than just hiding it
	if victor.VictorFaction != 0 {
		factionData := clm.queries.CombatCache.FindFactionDataByID(victor.VictorFaction, clm.queries.ECSManager)
		if factionData != nil && factionData.IsPlayerControlled {
			// Player won - mark encounter as defeated and hide permanently
			//TODO, we need to remove this rather than hide sprite.
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
			clm.logManager.UpdateTextArea(clm.combatLogArea,
				fmt.Sprintf("Encounter '%s' defeated!", encounterData.Name))
		}
	}
}

// RestoreEncounterSprite restores the encounter sprite visibility when fleeing combat
// This allows the player to re-engage with the encounter later
func (clm *CombatLifecycleManager) RestoreEncounterSprite(encounterID ecs.EntityID) {
	if encounterID == 0 {
		return
	}

	entity := clm.ecsManager.FindEntityByID(encounterID)
	if entity == nil {
		return
	}

	// Only restore if not already defeated
	encounterData := common.GetComponentType[*encounter.OverworldEncounterData](
		entity,
		encounter.OverworldEncounterComponent,
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

// CleanupCombatEntities removes ALL combat entities when returning to exploration
func (clm *CombatLifecycleManager) CleanupCombatEntities() {
	fmt.Println("=== Combat Cleanup Starting ===")

	// Get player info
	//Todo, this can just be one function
	playerID := clm.getPlayerEntityID()
	playerSquadIDs := clm.getPlayerSquadIDs(playerID)
	playerPos := clm.getPlayerPosition(playerID)

	// Move player squads back to player position and remove combat components
	// TODO, we need a better way to handle this. We can just move the squads back to the roster
	clm.resetPlayerSquads(playerSquadIDs, playerPos)

	// Build set of enemy squad IDs for unit filtering
	enemySquadSet := make(map[ecs.EntityID]bool)
	for _, id := range clm.enemySquadIDs {
		enemySquadSet[id] = true
	}

	// Dispose all combat entities in one pass
	clm.disposeEntitiesByTagHelper(combat.FactionTag, "factions")
	clm.disposeEntitiesByTagHelper(combat.ActionStateTag, "action states")
	clm.disposeEntitiesByTagHelper(combat.TurnStateTag, "turn states")
	clm.disposeEnemySquadsHelper()
	clm.disposeEnemyUnitsHelper(enemySquadSet)

	// Clear tracking
	clm.enemySquadIDs = []ecs.EntityID{}
	fmt.Println("=== Combat Cleanup Complete ===")
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

// getPlayerEntityID returns the stored player entity ID
func (clm *CombatLifecycleManager) getPlayerEntityID() ecs.EntityID {
	return clm.playerEntityID
}

// Helper: Get player squad IDs from roster
func (clm *CombatLifecycleManager) getPlayerSquadIDs(playerID ecs.EntityID) map[ecs.EntityID]bool {
	playerSquadIDs := make(map[ecs.EntityID]bool)
	if playerID == 0 {
		return playerSquadIDs
	}

	roster := squads.GetPlayerSquadRoster(playerID, clm.ecsManager)
	if roster != nil {
		for _, squadID := range roster.OwnedSquads {
			playerSquadIDs[squadID] = true
		}
	}
	return playerSquadIDs
}

// Helper: Get player position
func (clm *CombatLifecycleManager) getPlayerPosition(playerID ecs.EntityID) coords.LogicalPosition {
	defaultPos := coords.LogicalPosition{X: 0, Y: 0}
	if playerID == 0 {
		return defaultPos
	}

	playerEntity := clm.ecsManager.FindEntityByID(playerID)
	if playerEntity == nil {
		return defaultPos
	}

	if pos := common.GetComponentType[*coords.LogicalPosition](playerEntity, common.PositionComponent); pos != nil {
		return *pos
	}
	return defaultPos
}

// Helper: Reset player squads to player position
func (clm *CombatLifecycleManager) resetPlayerSquads(playerSquadIDs map[ecs.EntityID]bool, playerPos coords.LogicalPosition) {
	movedCount := 0
	for _, result := range clm.ecsManager.World.Query(squads.SquadTag) {
		entity := result.Entity
		squadID := entity.GetID()

		if !playerSquadIDs[squadID] {
			continue
		}

		// Move squad back to player
		if squadPos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent); squadPos != nil {
			clm.ecsManager.MoveEntity(squadID, entity, *squadPos, playerPos)
			movedCount++
		}

		// Remove combat component
		if entity.HasComponent(combat.FactionMembershipComponent) {
			entity.RemoveComponent(combat.FactionMembershipComponent)
		}
	}
	fmt.Printf("Moved %d player squads back to player\n", movedCount)
}

// Helper: Dispose entities by tag
func (clm *CombatLifecycleManager) disposeEntitiesByTagHelper(tag ecs.Tag, name string) {
	count := 0
	for _, result := range clm.ecsManager.World.Query(tag) {
		clm.ecsManager.World.DisposeEntities(result.Entity)
		count++
	}
	fmt.Printf("Disposed %d %s\n", count, name)
}

// Helper: Dispose enemy squads
func (clm *CombatLifecycleManager) disposeEnemySquadsHelper() {
	for _, squadID := range clm.enemySquadIDs {
		if entity := clm.ecsManager.FindEntityByID(squadID); entity != nil {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			clm.ecsManager.CleanDisposeEntity(entity, pos)
		}
	}
	fmt.Printf("Disposed %d enemy squads\n", len(clm.enemySquadIDs))
}

// Helper: Dispose units belonging to enemy squads
func (clm *CombatLifecycleManager) disposeEnemyUnitsHelper(enemySquadSet map[ecs.EntityID]bool) {
	count := 0
	for _, result := range clm.ecsManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)

		if memberData != nil && enemySquadSet[memberData.SquadID] {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			clm.ecsManager.CleanDisposeEntity(entity, pos)
			count++
		}
	}
	fmt.Printf("Disposed %d enemy units\n", count)
}

// resolveCombatToOverworld applies combat outcome to overworld threat state
func (clm *CombatLifecycleManager) resolveCombatToOverworld(
	threatNodeID ecs.EntityID,
	victor *combatservices.VictoryCheckResult,
) {
	// Calculate casualties
	playerUnitsLost, enemyUnitsKilled := clm.calculateCasualties(victor)

	// Determine outcome type
	playerVictory := false
	playerRetreat := false

	if victor.VictorFaction != 0 {
		factionData := clm.queries.CombatCache.FindFactionDataByID(victor.VictorFaction, clm.queries.ECSManager)
		if factionData != nil {
			playerVictory = factionData.IsPlayerControlled
		}
	}

	// TODO: Track retreat status (currently not implemented in combat system)
	// For now, assume no retreat - either win or lose

	// Get player squad ID (use first deployed squad)
	playerSquadID := clm.getFirstPlayerSquadID()

	// Calculate rewards from threat
	threatEntity := clm.ecsManager.FindEntityByID(threatNodeID)
	if threatEntity == nil {
		fmt.Printf("WARNING: Threat node %d not found for resolution\n", threatNodeID)
		return
	}

	threatData := common.GetComponentType[*overworld.ThreatNodeData](threatEntity, overworld.ThreatNodeComponent)
	if threatData == nil {
		fmt.Printf("WARNING: Entity %d is not a threat node\n", threatNodeID)
		return
	}

	rewards := overworld.CalculateRewards(threatData.Intensity, threatData.ThreatType)

	// Create combat outcome
	outcome := combatresolution.CreateCombatOutcome(
		threatNodeID,
		playerVictory,
		playerRetreat,
		playerSquadID,
		playerUnitsLost,
		enemyUnitsKilled,
		rewards,
	)

	// Apply to overworld
	if err := combatresolution.ResolveCombatToOverworld(clm.ecsManager, outcome); err != nil {
		fmt.Printf("ERROR resolving combat to overworld: %v\n", err)
		clm.logManager.UpdateTextArea(clm.combatLogArea,
			fmt.Sprintf("Warning: Failed to update overworld state: %v", err))
	} else {
		fmt.Printf("Combat resolved to overworld: %d enemy killed, %d player lost\n",
			enemyUnitsKilled, playerUnitsLost)
	}
}

// calculateCasualties counts units killed in combat
func (clm *CombatLifecycleManager) calculateCasualties(
	victor *combatservices.VictoryCheckResult,
) (playerUnitsLost int, enemyUnitsKilled int) {
	// Count destroyed units by faction
	playerFactionID := ecs.EntityID(0)
	enemyFactionID := ecs.EntityID(0)

	// Find player and enemy factions
	for _, result := range clm.ecsManager.World.Query(combat.FactionTag) {
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
	for _, result := range clm.ecsManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData == nil {
			continue
		}

		// Get squad to check faction membership
		squadEntity := clm.ecsManager.FindEntityByID(memberData.SquadID)
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
func (clm *CombatLifecycleManager) getFirstPlayerSquadID() ecs.EntityID {
	roster := squads.GetPlayerSquadRoster(clm.playerEntityID, clm.ecsManager)
	if roster != nil && len(roster.OwnedSquads) > 0 {
		return roster.OwnedSquads[0]
	}
	return 0
}
