package combatservices

import (
	"fmt"
	"game_main/core/common"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/effects"
	"game_main/tactical/powers/perks"
	"game_main/tactical/powers/powercore"
	"game_main/tactical/squads/squadcore"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	EntityManager   *common.EntityManager
	TurnManager     *combatcore.TurnManager
	FactionManager  *combatstate.CombatFactionManager
	MovementSystem  *combatcore.CombatMovementSystem
	CombatCache     *combatstate.CombatQueryCache
	CombatActSystem *combatcore.CombatActionSystem

	// Battle recording for export
	BattleRecorder *battlelog.BattleRecorder

	// Threat evaluation system (injected via SetThreatProvider/SetThreatEvaluatorFactory)
	threatProvider  ThreatProvider
	layerEvaluators map[ecs.EntityID]ThreatLayerEvaluator
	evalFactory     func(factionID ecs.EntityID) ThreatLayerEvaluator

	// AI decision-making (set via SetAIController due to import cycle: ai -> combatservices)
	aiController AITurnController

	// Artifact charge tracking (per-battle and per-round)
	chargeTracker      *artifacts.ArtifactChargeTracker
	artifactDispatcher *artifacts.ArtifactDispatcher

	// Power system dispatchers (artifacts + perks) and the pipeline that
	// fans lifecycle events out to them in a declarative, registration-order
	// sequence. Replaces four near-duplicate Fire* bodies that previously
	// hard-coded the "artifacts → perks → GUI" ordering.
	perkDispatcher *perks.SquadPerkDispatcher
	powerPipeline  *powercore.PowerPipeline

	// Optional GUI callbacks for UI updates (cache invalidation, visualization refresh).
	// Set by CombatMode via Set* methods. Invoked as the last pipeline subscriber.
	onAttackCompleteGUI func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)

	// Exit state — set as combat ends (by turn flow on victory/defeat/flee), consumed on mode exit.
	fleeRequested       bool
	cachedVictoryResult *VictoryCheckResult
	onMoveCompleteGUI   func(squadID ecs.EntityID)
	onTurnEndGUI        func(round int)
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	cache := combatstate.NewCombatQueryCache(manager)
	battleRecorder := battlelog.NewBattleRecorder()
	combatActSystem := combatcore.NewCombatActionSystem(manager, cache)
	movementSystem := combatcore.NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	turnManager := combatcore.NewTurnManager(manager, cache)

	// Wire up battle recorder to combat action system
	combatActSystem.SetBattleRecorder(battleRecorder)

	// Charge tracker is created once and lives for the lifetime of the service;
	// per-battle reset happens via tracker.Reset() in InitializeCombat so the
	// dispatcher's bindings on the PowerPipeline stay valid across battles.
	chargeTracker := artifacts.NewArtifactChargeTracker()

	cs := &CombatService{
		EntityManager:      manager,
		TurnManager:        turnManager,
		FactionManager:     combatstate.NewCombatFactionManager(manager, cache),
		MovementSystem:     movementSystem,
		CombatCache:        cache,
		CombatActSystem:    combatActSystem,
		BattleRecorder:     battleRecorder,
		layerEvaluators:    make(map[ecs.EntityID]ThreatLayerEvaluator),
		chargeTracker:      chargeTracker,
		artifactDispatcher: artifacts.NewArtifactDispatcher(manager, cache, chargeTracker),
		powerPipeline:      &powercore.PowerPipeline{},
	}

	// Set up shared logger and construct the perk dispatcher.
	setupPowerDispatch(cs, manager, cache)

	// Register pipeline subscribers in the order they must fire. This is the
	// one place execution order is declared — adding a new power system is a
	// single On* call below, not edits to four Fire* method bodies.
	//
	// Order rationale:
	//   1. Artifact behaviors (e.g. Deadlock Shackles must lock before perk TurnStart)
	//   2. Perk hooks (TurnStart, state tracking)
	//   3. GUI callbacks (cache invalidation, visuals) — always last, nil-safe
	cs.powerPipeline.OnPostReset(cs.artifactDispatcher.DispatchPostReset)
	cs.powerPipeline.OnPostReset(func(factionID ecs.EntityID, squadIDs []ecs.EntityID) {
		if cs.perkDispatcher != nil {
			cs.perkDispatcher.DispatchTurnStart(squadIDs, cs.TurnManager.GetCurrentRound(), cs.EntityManager)
		}
	})

	cs.powerPipeline.OnAttackComplete(cs.artifactDispatcher.DispatchOnAttackComplete)
	cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
		if cs.perkDispatcher != nil {
			cs.perkDispatcher.DispatchAttackTracking(attackerID, defenderID, cs.EntityManager)
		}
	})
	cs.powerPipeline.OnAttackComplete(func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult) {
		if cs.onAttackCompleteGUI != nil {
			cs.onAttackCompleteGUI(attackerID, defenderID, result)
		}
	})

	cs.powerPipeline.OnTurnEnd(cs.artifactDispatcher.DispatchOnTurnEnd)
	cs.powerPipeline.OnTurnEnd(func(round int) {
		if cs.perkDispatcher != nil {
			cs.perkDispatcher.DispatchRoundEnd(cs.EntityManager)
		}
	})
	cs.powerPipeline.OnTurnEnd(func(round int) {
		if cs.onTurnEndGUI != nil {
			cs.onTurnEndGUI(round)
		}
	})

	cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) {
		if cs.perkDispatcher != nil {
			cs.perkDispatcher.DispatchMoveTracking(squadID, cs.EntityManager)
		}
	})
	cs.powerPipeline.OnMoveComplete(func(squadID ecs.EntityID) {
		if cs.onMoveCompleteGUI != nil {
			cs.onMoveCompleteGUI(squadID)
		}
	})

	// Subsystem hooks forward into the pipeline directly — no intermediate wrapper methods.
	combatActSystem.SetOnAttackComplete(cs.powerPipeline.FireAttackComplete)
	movementSystem.SetOnMoveComplete(cs.powerPipeline.FireMoveComplete)
	turnManager.SetOnTurnEnd(cs.powerPipeline.FireTurnEnd)
	turnManager.SetPostResetHook(cs.powerPipeline.FirePostReset)

	return cs
}

// ========================================
// Power System Dispatch (Artifacts → Perks)
// ========================================

// SetOnAttackCompleteGUI sets the GUI callback for attack-complete events.
func (cs *CombatService) SetOnAttackCompleteGUI(fn func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)) {
	cs.onAttackCompleteGUI = fn
}

// SetOnMoveCompleteGUI sets the GUI callback for move-complete events.
func (cs *CombatService) SetOnMoveCompleteGUI(fn func(squadID ecs.EntityID)) {
	cs.onMoveCompleteGUI = fn
}

// SetOnTurnEndGUI sets the GUI callback for turn-end events.
func (cs *CombatService) SetOnTurnEndGUI(fn func(round int)) {
	cs.onTurnEndGUI = fn
}

// GetChargeTracker returns the artifact charge tracker for the current battle.
func (cs *CombatService) GetChargeTracker() *artifacts.ArtifactChargeTracker {
	return cs.chargeTracker
}

// InitializeCombat initializes combat with the given factions.
// Also assigns any unassigned deployed squads to the player faction as a safety net.
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	// Clear all charge/pending state for the new battle. The tracker instance
	// is shared with the ArtifactDispatcher; resetting in place preserves the
	// pipeline subscriber bindings rather than swapping the tracker out.
	cs.chargeTracker.Reset()
	// Find player faction (has IsPlayerControlled = true)
	var playerFactionID ecs.EntityID
	for _, factionID := range factionIDs {
		factionData := cs.CombatCache.FindFactionDataByID(factionID)
		if factionData != nil && factionData.IsPlayerControlled {
			playerFactionID = factionID
			break
		}
	}

	// Safety net: assign any deployed squads that somehow lack faction membership.
	// Starters should enroll all squads via EnrollSquadInFaction, so this should
	// rarely fire. If it does, it indicates a bug in the starter.
	if playerFactionID != 0 {
		cs.assignDeployedSquadsToPlayerFaction(playerFactionID)
	}

	// Apply minor artifact effects to all factions before combat initialization
	for _, factionID := range factionIDs {
		factionSquads := combatstate.GetSquadsForFaction(factionID, cs.EntityManager)
		artifacts.ApplyArtifactStatEffects(factionSquads, cs.EntityManager)

		// Initialize perk round state for all squads with perks
		perks.InitializePerkRoundStatesForFaction(factionSquads, cs.EntityManager)
	}

	return cs.TurnManager.InitializeCombat(factionIDs)
}

// assignDeployedSquadsToPlayerFaction finds all squads with positions but no FactionMembershipComponent
// and assigns them to the player faction. This is a safety net for squads deployed via SquadDeploymentMode
// that weren't enrolled by the CombatStarter. Logs a warning when triggered.
func (cs *CombatService) assignDeployedSquadsToPlayerFaction(playerFactionID ecs.EntityID) {
	for _, result := range cs.EntityManager.World.Query(squadcore.SquadTag) {
		squadEntity := result.Entity
		squadID := squadEntity.GetID()

		combatFaction := common.GetComponentType[*combatstate.CombatFactionData](squadEntity, combatstate.FactionMembershipComponent)
		if combatFaction != nil {
			continue
		}

		position := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if position == nil {
			continue
		}

		fmt.Printf("WARNING: squad %d has position but no faction — starter should have enrolled it\n", squadID)
		if err := cs.FactionManager.AddSquadToFaction(playerFactionID, squadID, *position); err != nil {
			fmt.Printf("WARNING: failed to assign squad %d to player faction: %v\n", squadID, err)
		}
	}
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	return combatstate.GetActiveSquadsForFaction(factionID, cs.EntityManager)
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

// MarkFleeRequested records that the player chose to flee.
// Consumed by GetExitResult/IsFleeRequested when combat mode exits.
func (cs *CombatService) MarkFleeRequested() {
	cs.fleeRequested = true
}

// IsFleeRequested reports whether the player requested to flee.
func (cs *CombatService) IsFleeRequested() bool {
	return cs.fleeRequested
}

// CacheVictoryResult stores the victory/flee outcome so the mode exit can
// consume it without re-running CheckVictoryCondition.
func (cs *CombatService) CacheVictoryResult(result *VictoryCheckResult) {
	cs.cachedVictoryResult = result
}

// GetExitResult returns the cached victory/flee outcome, falling back to a fresh
// CheckVictoryCondition if nothing was cached (e.g., abnormal exits).
func (cs *CombatService) GetExitResult() *VictoryCheckResult {
	if cs.cachedVictoryResult != nil {
		return cs.cachedVictoryResult
	}
	return cs.CheckVictoryCondition()
}

// ClearExitState resets the flee flag and cached victory result. Called after
// the mode exit has consumed them, so the next combat starts clean.
func (cs *CombatService) ClearExitState() {
	cs.fleeRequested = false
	cs.cachedVictoryResult = nil
}

// CheckVictoryCondition checks if battle has ended
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: cs.TurnManager.GetCurrentRound(),
	}

	// Count alive squads per faction using existing helper
	aliveByFaction := make(map[ecs.EntityID]int)

	// Get all factions
	allFactions := combatstate.GetAllFactions(cs.EntityManager)
	for _, factionID := range allFactions {
		// Use existing GetActiveSquadsForFaction which filters destroyed squads
		activeSquads := combatstate.GetActiveSquadsForFaction(factionID, cs.EntityManager)
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

// GetThreatEvaluator returns composite evaluator for a faction (lazy initialization).
// Requires SetThreatEvaluatorFactory to have been called first.
func (cs *CombatService) GetThreatEvaluator(factionID ecs.EntityID) ThreatLayerEvaluator {
	if eval, exists := cs.layerEvaluators[factionID]; exists {
		return eval
	}
	if cs.evalFactory == nil {
		return nil
	}
	eval := cs.evalFactory(factionID)
	cs.layerEvaluators[factionID] = eval
	return eval
}

// GetThreatProvider returns the threat provider (must be injected via SetThreatProvider).
func (cs *CombatService) GetThreatProvider() ThreatProvider {
	return cs.threatProvider
}

// SetThreatProvider injects the threat data provider.
// Must be set externally because the concrete type lives in mind/behavior.
func (cs *CombatService) SetThreatProvider(tp ThreatProvider) {
	cs.threatProvider = tp
}

// SetThreatEvaluatorFactory injects a factory for creating per-faction threat evaluators.
// Must be set externally because the concrete type lives in mind/behavior.
func (cs *CombatService) SetThreatEvaluatorFactory(fn func(factionID ecs.EntityID) ThreatLayerEvaluator) {
	cs.evalFactory = fn
}

// UpdateThreatLayers updates all threat layers at start of AI turn
func (cs *CombatService) UpdateThreatLayers(currentRound int) {
	// Update base threat data first
	if cs.threatProvider != nil {
		cs.threatProvider.UpdateAllFactions()
	}

	// Then update composite layers
	for _, evaluator := range cs.layerEvaluators {
		evaluator.Update(currentRound)
	}
}

// GetAIController returns the AI controller (must be injected via SetAIController)
func (cs *CombatService) GetAIController() AITurnController {
	return cs.aiController
}

// SetAIController injects the AI turn controller.
// Must be set externally due to import cycle (ai -> combatservices).
func (cs *CombatService) SetAIController(ctrl AITurnController) {
	cs.aiController = ctrl
}

// ================================
// Combat Lifecycle Methods
// ================================

// CleanupCombat removes tactical-side combat entities when returning to exploration.
// Enemy squads must be provided by the encounter service for cleanup.
// Returns the player squad IDs that were in combat so the caller (EncounterService) can
// finish cross-cutting cleanup (stripping FactionMembership, PerkRoundState, etc. via
// combatlifecycle.StripCombatComponents) without tactical/combat depending on mind/.
func (cs *CombatService) CleanupCombat(enemySquadIDs []ecs.EntityID) []ecs.EntityID {
	fmt.Println("=== Combat Cleanup Starting ===")

	// Remove all active effects from player units before leaving combat
	cs.cleanupEffects()

	// Identify player squads from faction membership so the caller can strip their combat components.
	playerSquadIDs := cs.collectPlayerSquadIDs()

	// Build set of enemy squad IDs for unit filtering
	enemySquadSet := make(map[ecs.EntityID]bool)
	for _, id := range enemySquadIDs {
		enemySquadSet[id] = true
	}

	// Dispose all combat entities in one pass
	cs.disposeEntitiesByTag(combatstate.FactionTag, "factions")
	cs.disposeEntitiesByTag(combatstate.ActionStateTag, "action states")
	cs.disposeEntitiesByTag(combatstate.TurnStateTag, "turn states")
	cs.disposeEnemySquads(enemySquadIDs)
	cs.disposeEnemyUnits(enemySquadSet)

	fmt.Println("=== Combat Cleanup Complete ===")
	return playerSquadIDs
}

// cleanupEffects removes all active effects from all player squad units.
// This ensures no stale buffs/debuffs persist between battles.
func (cs *CombatService) cleanupEffects() {
	cleaned := 0
	for _, result := range cs.EntityManager.World.Query(squadcore.SquadMemberTag) {
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

// collectPlayerSquadIDs returns all squads whose faction membership is player-controlled.
// Caller (EncounterService) uses this list to strip cross-cutting squad components via
// combatlifecycle.StripCombatComponents — keeps tactical/combat free of mind/ imports.
func (cs *CombatService) collectPlayerSquadIDs() []ecs.EntityID {
	var playerSquadIDs []ecs.EntityID
	for _, result := range cs.EntityManager.World.Query(squadcore.SquadTag) {
		entity := result.Entity
		factionData := common.GetComponentType[*combatstate.CombatFactionData](entity, combatstate.FactionMembershipComponent)
		if factionData == nil {
			continue
		}
		factionEntity := cs.EntityManager.FindEntityByID(factionData.FactionID)
		if factionEntity == nil {
			continue
		}
		faction := common.GetComponentType[*combatstate.FactionData](factionEntity, combatstate.CombatFactionComponent)
		if faction == nil || !faction.IsPlayerControlled {
			continue
		}
		playerSquadIDs = append(playerSquadIDs, entity.GetID())
	}
	return playerSquadIDs
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
	for _, result := range cs.EntityManager.World.Query(squadcore.SquadMemberTag) {
		entity := result.Entity
		memberData := common.GetComponentType[*squadcore.SquadMemberData](entity, squadcore.SquadMemberComponent)

		if memberData != nil && enemySquadSet[memberData.SquadID] {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			cs.EntityManager.CleanDisposeEntity(entity, pos)
			count++
		}
	}
	fmt.Printf("Disposed %d enemy units\n", count)
}
