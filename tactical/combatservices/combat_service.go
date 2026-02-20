package combatservices

import (
	"fmt"
	"game_main/common"
	"game_main/gear"
	"game_main/mind/ai"
	"game_main/mind/behavior"
	"game_main/mind/resolution"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/effects"
	"game_main/tactical/squads"
	"game_main/world/coords"

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

	// Artifact charge tracking (per-battle and per-round)
	chargeTracker *gear.ArtifactChargeTracker

	// Post-action callbacks (registered by GUI layer)
	onAttackComplete []OnAttackCompleteFunc
	onMoveComplete   []OnMoveCompleteFunc
	onTurnEnd        []OnTurnEndFunc
	postResetHooks   []PostResetHookFunc
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	cache := combat.NewCombatQueryCache(manager)
	battleRecorder := battlelog.NewBattleRecorder()
	combatActSystem := combat.NewCombatActionSystem(manager, cache)
	movementSystem := combat.NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	turnManager := combat.NewTurnManager(manager, cache)

	// Wire up battle recorder to combat action system
	combatActSystem.SetBattleRecorder(battleRecorder)

	cs := &CombatService{
		EntityManager:   manager,
		TurnManager:     turnManager,
		FactionManager:  combat.NewCombatFactionManager(manager, cache),
		MovementSystem:  movementSystem,
		CombatCache:     cache,
		CombatActSystem: combatActSystem,
		BattleRecorder:  battleRecorder,
		ThreatManager:   behavior.NewFactionThreatLevelManager(manager, cache),
		LayerEvaluators: make(map[ecs.EntityID]*behavior.CompositeThreatEvaluator),
	}

	// Wire system hooks to forward to registered callbacks
	combatActSystem.SetOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
		for _, fn := range cs.onAttackComplete {
			fn(attackerID, defenderID, result)
		}
	})

	movementSystem.SetOnMoveComplete(func(squadID ecs.EntityID) {
		for _, fn := range cs.onMoveComplete {
			fn(squadID)
		}
	})

	turnManager.SetOnTurnEnd(func(round int) {
		for _, fn := range cs.onTurnEnd {
			fn(round)
		}
	})

	// Wire post-reset hook to forward to registered callbacks
	turnManager.SetPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		for _, fn := range cs.postResetHooks {
			fn(factionID, squadIDs)
		}
	})

	// Register artifact behavior dispatch
	setupBehaviorDispatch(cs, manager, cache)

	return cs
}

// GetChargeTracker returns the artifact charge tracker for the current battle.
func (cs *CombatService) GetChargeTracker() *gear.ArtifactChargeTracker {
	return cs.chargeTracker
}

// InitializeCombat initializes combat with the given factions
// Also assigns any unassigned squads (from squad deployment) to the player faction.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. remove.
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	// Reset charge tracker for the new battle
	cs.chargeTracker = gear.NewArtifactChargeTracker()
	// Find player faction (has IsPlayerControlled = true)
	var playerFactionID ecs.EntityID
	for _, factionID := range factionIDs {
		// Use cached query for performance
		factionData := cs.CombatCache.FindFactionDataByID(factionID)
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

	// Apply minor artifact effects to all factions before combat initialization
	for _, factionID := range factionIDs {
		factionSquads := combat.GetSquadsForFaction(factionID, cs.EntityManager)
		gear.ApplyArtifactStatEffects(factionSquads, cs.EntityManager)
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
			fmt.Printf("WARNING: failed to assign squad %d to player faction: %v\n", squadID, err)
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
		factionData := cs.CombatCache.FindFactionDataByID(victorFaction)
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

// ================================
// Artifact Behavior Dispatch
// ================================

// setupBehaviorDispatch wires all registered artifact behaviors to the combat event system.
func setupBehaviorDispatch(cs *CombatService, manager *common.EntityManager, cache *combat.CombatQueryCache) {
	cs.RegisterPostResetHook(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		ctx := &gear.BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: cs.chargeTracker}
		for _, b := range gear.AllBehaviors() {
			b.OnPostReset(ctx, factionID, squadIDs)
		}
	})

	cs.RegisterOnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *squads.CombatResult) {
		ctx := &gear.BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: cs.chargeTracker}
		for _, b := range gear.AllBehaviors() {
			b.OnAttackComplete(ctx, attackerID, defenderID, result)
		}
	})

	cs.RegisterOnTurnEnd(func(round int) {
		if cs.chargeTracker != nil {
			cs.chargeTracker.RefreshRoundCharges()
		}
		ctx := &gear.BehaviorContext{Manager: manager, Cache: cache, ChargeTracker: cs.chargeTracker}
		for _, b := range gear.AllBehaviors() {
			b.OnTurnEnd(ctx, round)
		}
	})
}

// ================================
// Combat Lifecycle Methods
// ================================

// CleanupCombat removes ALL combat entities when returning to exploration
// Enemy squads must be provided by the encounter service for cleanup
func (cs *CombatService) CleanupCombat(enemySquadIDs []ecs.EntityID) {
	fmt.Println("=== Combat Cleanup Starting ===")

	// Clear registered callbacks (they reference GUI state that's being torn down)
	cs.ClearCallbacks()

	// Remove all active effects from player units before leaving combat
	cs.cleanupEffects()

	// Remove player squads from map and remove combat components
	// Uses faction-based filtering instead of roster lookup
	cs.resetPlayerSquadsToOverworld()

	// Build set of enemy squad IDs for unit filtering
	enemySquadSet := make(map[ecs.EntityID]bool)
	for _, id := range enemySquadIDs {
		enemySquadSet[id] = true
	}

	// Dispose all combat entities in one pass
	cs.disposeEntitiesByTag(combat.FactionTag, "factions")
	cs.disposeEntitiesByTag(combat.ActionStateTag, "action states")
	cs.disposeEntitiesByTag(combat.TurnStateTag, "turn states")
	cs.disposeEnemySquads(enemySquadIDs)
	cs.disposeEnemyUnits(enemySquadSet)

	fmt.Println("=== Combat Cleanup Complete ===")
}

// cleanupEffects removes all active effects from all player squad units.
// This ensures no stale buffs/debuffs persist between battles.
func (cs *CombatService) cleanupEffects() {
	cleaned := 0
	for _, result := range cs.EntityManager.World.Query(squads.SquadMemberTag) {
		unitID := result.Entity.GetID()
		if effects.HasActiveEffects(unitID, cs.EntityManager) {
			effects.RemoveAllEffects(unitID, cs.EntityManager)
			cleaned++
		}
	}
	if cleaned > 0 {
		fmt.Printf("Cleaned up effects from %d units\n", cleaned)
	}
}

// ================================
// Helper Methods
// ================================

// resetPlayerSquadsToOverworld removes player squads from the map after combat.
// Player squads should only exist in the roster, not on the map.
// Uses faction membership to identify player squads, then delegates stripping to resolution.
func (cs *CombatService) resetPlayerSquadsToOverworld() {
	var playerSquadIDs []ecs.EntityID
	for _, result := range cs.EntityManager.World.Query(squads.SquadTag) {
		entity := result.Entity
		factionData := common.GetComponentType[*combat.CombatFactionData](entity, combat.FactionMembershipComponent)
		if factionData == nil {
			continue
		}
		factionEntity := cs.EntityManager.FindEntityByID(factionData.FactionID)
		if factionEntity == nil {
			continue
		}
		faction := common.GetComponentType[*combat.FactionData](factionEntity, combat.CombatFactionComponent)
		if faction == nil || !faction.IsPlayerControlled {
			continue
		}
		playerSquadIDs = append(playerSquadIDs, entity.GetID())
	}
	resolution.StripCombatComponents(cs.EntityManager, playerSquadIDs)
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
func (cs *CombatService) disposeEnemySquads(enemySquadIDs []ecs.EntityID) {
	for _, squadID := range enemySquadIDs {
		if entity := cs.EntityManager.FindEntityByID(squadID); entity != nil {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			cs.EntityManager.CleanDisposeEntity(entity, pos)
		}
	}
	fmt.Printf("Disposed %d enemy squads\n", len(enemySquadIDs))
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
