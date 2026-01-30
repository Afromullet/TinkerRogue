package combatservices

import (
	"fmt"
	"game_main/common"
	"game_main/mind/ai"
	"game_main/mind/behavior"
	"game_main/mind/encounter"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combatresolution"
	"game_main/tactical/squads"
	"game_main/visual/rendering"
	"game_main/world/coords"
	"game_main/world/overworld"

	"github.com/bytearena/ecs"
)

// Type aliases for GUI layer convenience
type (
	AIController = ai.AIController
	QueuedAttack = ai.QueuedAttack
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	EntityManager   *common.EntityManager
	TurnManager     *combat.TurnManager
	FactionManager  *combat.CombatFactionManager
	MovementSystem  *combat.CombatMovementSystem
	CombatCache     *combat.CombatQueryCache
	CombatActSystem *combat.CombatActionSystem

	// Battle recording for export
	BattleRecorder *battlelog.BattleRecorder

	// Threat evaluation system
	ThreatManager   *behavior.FactionThreatLevelManager
	LayerEvaluators map[ecs.EntityID]*behavior.CompositeThreatEvaluator

	// AI decision-making
	aiController *ai.AIController

	// Combat lifecycle tracking
	currentEncounterID ecs.EntityID
	enemySquadIDs      []ecs.EntityID // Track enemy squads for cleanup
	playerEntityID     ecs.EntityID   // Player entity ID for squad management
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	cache := combat.NewCombatQueryCache(manager)
	battleRecorder := battlelog.NewBattleRecorder()
	combatActSystem := combat.NewCombatActionSystem(manager, cache)

	// Wire up battle recorder to combat action system
	combatActSystem.SetBattleRecorder(battleRecorder)

	return &CombatService{
		EntityManager:   manager,
		TurnManager:     combat.NewTurnManager(manager, cache),
		FactionManager:  combat.NewCombatFactionManager(manager, cache),
		MovementSystem:  combat.NewMovementSystem(manager, common.GlobalPositionSystem, cache),
		CombatCache:     cache,
		CombatActSystem: combatActSystem,
		BattleRecorder:  battleRecorder,
		ThreatManager:   behavior.NewFactionThreatLevelManager(manager, cache),
		LayerEvaluators: make(map[ecs.EntityID]*behavior.CompositeThreatEvaluator),
	}
}

// InitializeCombat initializes combat with the given factions
// Also assigns any unassigned squads (from squad deployment) to the player faction.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. remove.
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	// Find player faction (has IsPlayerControlled = true)
	var playerFactionID ecs.EntityID
	for _, factionID := range factionIDs {
		// Use cached query for performance
		factionData := cs.CombatCache.FindFactionDataByID(factionID, cs.EntityManager)
		if factionData != nil && factionData.IsPlayerControlled {
			playerFactionID = factionID
			break
		}
	}

	// Assign any unassigned squads to player faction
	// These are squads deployed via SquadDeploymentMode that have positions but no FactionMembershipComponent
	if playerFactionID != 0 {
		cs.assignDeployedSquadsToPlayerFaction(playerFactionID)
	}

	return cs.TurnManager.InitializeCombat(factionIDs)
}

// assignDeployedSquadsToPlayerFaction finds all squads with positions but no FactionMembershipComponent
// and assigns them to the player faction. These are squads that were deployed via SquadDeploymentMode.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. Squads will have to be assigned to the
// Correct Faction. There can be multiple players
func (cs *CombatService) assignDeployedSquadsToPlayerFaction(playerFactionID ecs.EntityID) {
	for _, result := range cs.EntityManager.World.Query(squads.SquadTag) {
		squadEntity := result.Entity
		squadID := squadEntity.GetID()

		// Check if squad already has a faction (skip if it does)
		combatFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.FactionMembershipComponent)
		if combatFaction != nil {
			continue // Already assigned to a faction
		}

		// Check if squad has a position (deployed squads have positions)
		position := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if position == nil {
			continue // No position, not a deployed squad
		}

		// Squad is unassigned and deployed - add it to player faction
		if err := cs.FactionManager.AddSquadToFaction(playerFactionID, squadID, *position); err != nil {
			// Log error but continue with other squads
			continue
		}
	}
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	return combat.GetActiveSquadsForFaction(factionID, cs.EntityManager)
}

// VictoryCheckResult contains battle outcome information.
type VictoryCheckResult struct {
	BattleOver       bool
	VictorFaction    ecs.EntityID
	VictorName       string
	IsPlayerVictory  bool // True if a player-controlled faction won
	DefeatedFactions []ecs.EntityID
	RoundsCompleted  int
}

// CheckVictoryCondition checks if battle has ended
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: cs.TurnManager.GetCurrentRound(),
	}

	// Count alive squads per faction using existing helper
	aliveByFaction := make(map[ecs.EntityID]int)

	// Get all factions
	allFactions := combat.GetAllFactions(cs.EntityManager)
	for _, factionID := range allFactions {
		// Use existing GetActiveSquadsForFaction which filters destroyed squads
		activeSquads := combat.GetActiveSquadsForFaction(factionID, cs.EntityManager)
		aliveByFaction[factionID] = len(activeSquads)
	}

	// Check victory: only one faction with alive squads
	factionsWithSquads := 0
	var victorFaction ecs.EntityID
	for factionID, count := range aliveByFaction {
		if count > 0 {
			factionsWithSquads++
			victorFaction = factionID
		} else {
			result.DefeatedFactions = append(result.DefeatedFactions, factionID)
		}
	}

	if factionsWithSquads <= 1 {
		result.BattleOver = true
		result.VictorFaction = victorFaction

		// Get faction data to determine victor name and if player won
		factionData := cs.CombatCache.FindFactionDataByID(victorFaction, cs.EntityManager)
		if factionData != nil {
			// Set player victory flag (SINGLE SOURCE OF TRUTH)
			result.IsPlayerVictory = factionData.IsPlayerControlled

			if factionData.PlayerID > 0 {
				// Player victory - include player name
				result.VictorName = fmt.Sprintf("%s (%s)", factionData.Name, factionData.PlayerName)
			} else {
				// AI victory
				result.VictorName = factionData.Name
			}
		} else {
			result.VictorName = "Unknown"
			result.IsPlayerVictory = false
		}
	}

	return result
}

// GetThreatEvaluator returns composite evaluator for a faction (lazy initialization)
func (cs *CombatService) GetThreatEvaluator(factionID ecs.EntityID) *behavior.CompositeThreatEvaluator {
	if evaluator, exists := cs.LayerEvaluators[factionID]; exists {
		return evaluator
	}

	// Create new evaluator for this faction
	evaluator := behavior.NewCompositeThreatEvaluator(
		factionID,
		cs.EntityManager,
		cs.CombatCache,
		cs.ThreatManager,
	)
	cs.LayerEvaluators[factionID] = evaluator
	return evaluator
}

// UpdateThreatLayers updates all threat layers at start of AI turn
func (cs *CombatService) UpdateThreatLayers(currentRound int) {
	// Update base threat data first
	cs.ThreatManager.UpdateAllFactions()

	// Then update composite layers
	for _, evaluator := range cs.LayerEvaluators {
		evaluator.Update(currentRound)
	}
}

// GetAIController returns the AI controller (lazy initialization)
func (cs *CombatService) GetAIController() *ai.AIController {
	if cs.aiController == nil {
		cs.aiController = ai.NewAIController(
			cs.EntityManager,
			cs.TurnManager,
			cs.MovementSystem,
			cs.CombatActSystem,
			cs.CombatCache,
			cs.ThreatManager,
			cs.LayerEvaluators,
		)
	}
	return cs.aiController
}

// SetPlayerEntity sets the player entity ID for combat lifecycle management
func (cs *CombatService) SetPlayerEntity(playerID ecs.EntityID) {
	cs.playerEntityID = playerID
}

// ================================
// Combat Lifecycle Methods
// ================================

// StartEncounter initializes a combat encounter by spawning entities.
// Returns the encounter ID and any error encountered.
func (cs *CombatService) StartEncounter(encounterID ecs.EntityID, playerStartPos coords.LogicalPosition) (ecs.EntityID, error) {
	fmt.Println("Starting combat encounter - spawning entities")

	// Clear previous enemy squad tracking
	cs.enemySquadIDs = []ecs.EntityID{}
	cs.currentEncounterID = encounterID

	// Extract encounter data to pass to balanced spawner
	var encounterData *encounter.OverworldEncounterData
	if encounterID != 0 {
		entity := cs.EntityManager.FindEntityByID(encounterID)
		if entity != nil {
			encounterData = common.GetComponentType[*encounter.OverworldEncounterData](
				entity,
				encounter.OverworldEncounterComponent,
			)
			if encounterData != nil {
				fmt.Printf("Encounter: %s (Level %d)\n", encounterData.Name, encounterData.Level)
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
	enemySquadIDs, err := encounter.SetupBalancedEncounter(cs.EntityManager, cs.playerEntityID, playerStartPos, encounterData, encounterID)
	if err != nil {
		return 0, fmt.Errorf("failed to setup balanced encounter: %w", err)
	}

	// Store enemy squad IDs for cleanup
	cs.enemySquadIDs = enemySquadIDs
	fmt.Printf("Tracking %d enemy squads for cleanup: %v\n", len(cs.enemySquadIDs), cs.enemySquadIDs)

	return encounterID, nil
}

// GetCurrentEncounterID returns the current encounter ID
func (cs *CombatService) GetCurrentEncounterID() ecs.EntityID {
	return cs.currentEncounterID
}

// EndEncounter marks an encounter as defeated if the player won and applies combat resolution to overworld threats.
// Accepts a pre-calculated VictoryCheckResult to avoid redundant checks.
func (cs *CombatService) EndEncounter(victor *VictoryCheckResult) {
	// Only mark if we have a tracked encounter
	if cs.currentEncounterID == 0 {
		return
	}

	// Use the provided victory result (already calculated by caller)

	// Get encounter data
	entity := cs.EntityManager.FindEntityByID(cs.currentEncounterID)
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
		cs.resolveCombatToOverworld(encounterData.ThreatNodeID, victor)
	}

	// Only mark as defeated if a player faction won (uses single source of truth)
	if victor.IsPlayerVictory {
		// Player won - mark encounter as defeated and hide permanently
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
func (cs *CombatService) RestoreEncounterSprite() {
	if cs.currentEncounterID == 0 {
		return
	}

	entity := cs.EntityManager.FindEntityByID(cs.currentEncounterID)
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

// CleanupCombat removes ALL combat entities when returning to exploration
func (cs *CombatService) CleanupCombat() {
	fmt.Println("=== Combat Cleanup Starting ===")

	// Remove player squads from map and remove combat components
	// NEW: Uses faction-based filtering instead of roster lookup
	cs.resetPlayerSquadsToOverworld()

	// Build set of enemy squad IDs for unit filtering
	enemySquadSet := make(map[ecs.EntityID]bool)
	for _, id := range cs.enemySquadIDs {
		enemySquadSet[id] = true
	}

	// Dispose all combat entities in one pass
	cs.disposeEntitiesByTag(combat.FactionTag, "factions")
	cs.disposeEntitiesByTag(combat.ActionStateTag, "action states")
	cs.disposeEntitiesByTag(combat.TurnStateTag, "turn states")
	cs.disposeEnemySquads()
	cs.disposeEnemyUnits(enemySquadSet)

	// Clear tracking
	cs.enemySquadIDs = []ecs.EntityID{}
	cs.currentEncounterID = 0
	fmt.Println("=== Combat Cleanup Complete ===")
}

// ================================
// Combat Resolution to Overworld
// ================================

// resolveCombatToOverworld applies combat outcome to overworld threat state
func (cs *CombatService) resolveCombatToOverworld(threatNodeID ecs.EntityID, victor *VictoryCheckResult) {
	// Calculate casualties
	playerUnitsLost, enemyUnitsKilled := cs.calculateCasualties(victor)

	// Get player victory from single source of truth
	playerVictory := victor.IsPlayerVictory

	// Get player squad ID (use first deployed squad)
	playerSquadID := cs.getFirstPlayerSquadID()

	// Calculate rewards from threat
	threatEntity := cs.EntityManager.FindEntityByID(threatNodeID)
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
		false, // playerRetreat - not implemented yet
		playerSquadID,
		playerUnitsLost,
		enemyUnitsKilled,
		rewards,
	)

	// Apply to overworld
	if err := combatresolution.ResolveCombatToOverworld(cs.EntityManager, outcome); err != nil {
		fmt.Printf("ERROR resolving combat to overworld: %v\n", err)
	} else {
		fmt.Printf("Combat resolved to overworld: %d enemy killed, %d player lost\n",
			enemyUnitsKilled, playerUnitsLost)
	}
}

// calculateCasualties counts units killed in combat
func (cs *CombatService) calculateCasualties(victor *VictoryCheckResult) (playerUnitsLost int, enemyUnitsKilled int) {
	// Count destroyed units by faction
	playerFactionID := ecs.EntityID(0)
	enemyFactionID := ecs.EntityID(0)

	// Find player and enemy factions
	for _, result := range cs.EntityManager.World.Query(combat.FactionTag) {
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
	for _, result := range cs.EntityManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)
		if memberData == nil {
			continue
		}

		// Get squad to check faction membership
		squadEntity := cs.EntityManager.FindEntityByID(memberData.SquadID)
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

// ================================
// Helper Methods
// ================================

// getFirstPlayerSquadID returns the first player squad ID found
func (cs *CombatService) getFirstPlayerSquadID() ecs.EntityID {
	roster := squads.GetPlayerSquadRoster(cs.playerEntityID, cs.EntityManager)
	if roster != nil && len(roster.OwnedSquads) > 0 {
		return roster.OwnedSquads[0]
	}
	return 0
}

// resetPlayerSquadsToOverworld removes player squads from the map after combat
// Player squads should only exist in the roster, not on the map
// Uses faction membership to identify player squads (more robust than roster lookup)
func (cs *CombatService) resetPlayerSquadsToOverworld() {
	removedCount := 0
	skippedCount := 0

	for _, result := range cs.EntityManager.World.Query(squads.SquadTag) {
		entity := result.Entity
		squadID := entity.GetID()

		// Get faction membership - squads without this aren't in combat
		factionData := common.GetComponentType[*combat.CombatFactionData](entity, combat.FactionMembershipComponent)
		if factionData == nil {
			skippedCount++
			continue // Not in combat, skip
		}

		// Verify it's a player-controlled faction
		factionEntity := cs.EntityManager.FindEntityByID(factionData.FactionID)
		if factionEntity == nil {
			fmt.Printf("WARNING: Squad %d has invalid faction ID %d\n", squadID, factionData.FactionID)
			continue
		}

		faction := common.GetComponentType[*combat.FactionData](factionEntity, combat.CombatFactionComponent)
		if faction == nil {
			fmt.Printf("WARNING: Faction entity %d has no FactionData component\n", factionData.FactionID)
			continue
		}

		if !faction.IsPlayerControlled {
			// Enemy squad - should NOT be touched here, will be disposed separately
			skippedCount++
			continue
		}

		// This is a player squad - remove its position and its units' positions
		squadPos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if squadPos != nil {
			fmt.Printf("Removing player squad %d from map position (%d,%d)\n",
				squadID, squadPos.X, squadPos.Y)

			// Remove squad from position system and remove component
			common.GlobalPositionSystem.RemoveEntity(squadID, *squadPos)
			entity.RemoveComponent(common.PositionComponent)

			// Remove positions from all units in this squad
			unitIDs := squads.GetUnitIDsInSquad(squadID, cs.EntityManager)
			for _, unitID := range unitIDs {
				unitEntity := cs.EntityManager.FindEntityByID(unitID)
				if unitEntity == nil {
					continue
				}
				unitPos := common.GetComponentType[*coords.LogicalPosition](unitEntity, common.PositionComponent)
				if unitPos != nil {
					common.GlobalPositionSystem.RemoveEntity(unitID, *unitPos)
					unitEntity.RemoveComponent(common.PositionComponent)
				}
			}

			removedCount++
		}

		// Remove combat component (no longer in faction during overworld)
		if entity.HasComponent(combat.FactionMembershipComponent) {
			entity.RemoveComponent(combat.FactionMembershipComponent)
		}
	}

	fmt.Printf("Removed %d player squads from map (skipped %d non-player squads)\n", removedCount, skippedCount)
}

// disposeEntitiesByTag disposes all entities with a given tag
func (cs *CombatService) disposeEntitiesByTag(tag ecs.Tag, name string) {
	count := 0
	for _, result := range cs.EntityManager.World.Query(tag) {
		cs.EntityManager.World.DisposeEntities(result.Entity)
		count++
	}
	fmt.Printf("Disposed %d %s\n", count, name)
}

// disposeEnemySquads disposes all tracked enemy squads
func (cs *CombatService) disposeEnemySquads() {
	for _, squadID := range cs.enemySquadIDs {
		if entity := cs.EntityManager.FindEntityByID(squadID); entity != nil {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			cs.EntityManager.CleanDisposeEntity(entity, pos)
		}
	}
	fmt.Printf("Disposed %d enemy squads\n", len(cs.enemySquadIDs))
}

// disposeEnemyUnits disposes all units belonging to enemy squads
func (cs *CombatService) disposeEnemyUnits(enemySquadSet map[ecs.EntityID]bool) {
	count := 0
	for _, result := range cs.EntityManager.World.Query(squads.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squads.SquadMemberData](entity, squads.SquadMemberComponent)

		if memberData != nil && enemySquadSet[memberData.SquadID] {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			cs.EntityManager.CleanDisposeEntity(entity, pos)
			count++
		}
	}
	fmt.Printf("Disposed %d enemy units\n", count)
}
