package combatservices

import (
	"fmt"
	"game_main/core/common"
	"game_main/core/coords"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/combat/combattypes"
	"game_main/tactical/powers/artifacts"
	"game_main/tactical/powers/effects"
	"game_main/tactical/powers/perks"
	"game_main/tactical/powers/powercore"
	"game_main/tactical/squads/squadcore"

	"github.com/bytearena/ecs"
)

// CombatService owns the combat system fields, GUI callbacks, AI/threat
// injection, and lifecycle methods. Power-pipeline wiring and per-battle
// artifact/perk state are owned by Powers (*PowerOrchestrator); flee/victory
// state lives on exit (*CombatExitController).
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

	// Power pipeline + artifact/perk dispatchers + charge tracker live here.
	Powers *PowerOrchestrator

	// Optional GUI callbacks for UI updates. Set externally via SetOn*GUI; the
	// orchestrator's pipeline subscribers read fields off this struct lazily so
	// callbacks installed after WirePipeline still fire.
	guiHooks *GUIHooks

	// Exit state — flee flag + cached victory result + CheckVictoryCondition.
	exit *CombatExitController

	// Shared PowerLogger used by service teardown logging, abilities, gear,
	// artifacts, and perks. Stored here as a convenience; identical to
	// cs.Powers.Logger().
	logger powercore.PowerLogger
}

// NewCombatService creates a fully wired combat service. Pipeline subscribers
// and the perk dispatcher are constructed via PowerOrchestrator; the shared
// PowerLogger is set up by setupPowerDispatch which also routes ability and
// gear messages through it.
func NewCombatService(manager *common.EntityManager) *CombatService {
	cache := combatstate.NewCombatQueryCache(manager)
	battleRecorder := battlelog.NewBattleRecorder()
	combatActSystem := combatcore.NewCombatActionSystem(manager, cache)
	movementSystem := combatcore.NewMovementSystem(manager, common.GlobalPositionSystem, cache)
	turnManager := combatcore.NewTurnManager(manager, cache)

	combatActSystem.SetBattleRecorder(battleRecorder)

	cs := &CombatService{
		EntityManager:   manager,
		TurnManager:     turnManager,
		FactionManager:  combatstate.NewCombatFactionManager(manager, cache),
		MovementSystem:  movementSystem,
		CombatCache:     cache,
		CombatActSystem: combatActSystem,
		BattleRecorder:  battleRecorder,
		layerEvaluators: make(map[ecs.EntityID]ThreatLayerEvaluator),
		Powers:          NewPowerOrchestrator(manager, cache),
		guiHooks:        &GUIHooks{},
		exit:            NewCombatExitController(manager, turnManager, cache),
	}

	// Install the shared logger (creates perkDispatcher inside the orchestrator)
	// then wire pipeline subscribers in declared order.
	setupPowerDispatch(cs)
	cs.Powers.WirePipeline(manager, turnManager, combatActSystem, movementSystem, cs.guiHooks)

	return cs
}

// ========================================
// Power System Dispatch (Artifacts → Perks)
// ========================================

// SetOnAttackCompleteGUI sets the GUI callback for attack-complete events.
func (cs *CombatService) SetOnAttackCompleteGUI(fn func(attackerID, defenderID ecs.EntityID, result *combattypes.CombatResult)) {
	cs.guiHooks.OnAttackComplete = fn
}

// SetOnMoveCompleteGUI sets the GUI callback for move-complete events.
func (cs *CombatService) SetOnMoveCompleteGUI(fn func(squadID ecs.EntityID)) {
	cs.guiHooks.OnMoveComplete = fn
}

// SetOnTurnEndGUI sets the GUI callback for turn-end events.
func (cs *CombatService) SetOnTurnEndGUI(fn func(round int)) {
	cs.guiHooks.OnTurnEnd = fn
}

// GetChargeTracker returns the artifact charge tracker for the current battle.
func (cs *CombatService) GetChargeTracker() *artifacts.ArtifactChargeTracker {
	return cs.Powers.ChargeTracker()
}

// InitializeCombat initializes combat with the given factions. Squad-to-faction
// enrollment happens upstream in the encounter/raid starters via
// EnrollSquadsAtPositions; this method only runs the per-faction artifact and
// perk setup that depends on faction membership being complete.
func (cs *CombatService) InitializeCombat(factionIDs []ecs.EntityID) error {
	// Clear all charge/pending state for the new battle. The tracker instance
	// is shared with the ArtifactDispatcher; resetting in place preserves the
	// pipeline subscriber bindings rather than swapping the tracker out.
	cs.Powers.ChargeTracker().Reset()

	// Apply minor artifact effects to all factions before combat initialization
	for _, factionID := range factionIDs {
		factionSquads := combatstate.GetSquadsForFaction(factionID, cs.EntityManager)
		artifacts.ApplyArtifactStatEffects(factionSquads, cs.EntityManager)

		// Initialize perk round state for all squads with perks
		perks.InitializePerkRoundStatesForFaction(factionSquads, cs.EntityManager)
	}

	return cs.TurnManager.InitializeCombat(factionIDs)
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	return combatstate.GetActiveSquadsForFaction(factionID, cs.EntityManager)
}

// Exit returns the CombatExitController, which owns flee/victory state and the
// CheckVictoryCondition computation. Most callers should use the delegating
// methods below; new code can use cs.Exit() directly.
func (cs *CombatService) Exit() *CombatExitController { return cs.exit }

// MarkFleeRequested delegates to the exit controller. Kept on CombatService
// for caller compatibility.
func (cs *CombatService) MarkFleeRequested() { cs.exit.MarkFleeRequested() }

// IsFleeRequested delegates to the exit controller.
func (cs *CombatService) IsFleeRequested() bool { return cs.exit.IsFleeRequested() }

// CacheVictoryResult delegates to the exit controller.
func (cs *CombatService) CacheVictoryResult(result *VictoryCheckResult) {
	cs.exit.CacheVictoryResult(result)
}

// GetExitResult delegates to the exit controller.
func (cs *CombatService) GetExitResult() *VictoryCheckResult { return cs.exit.GetExitResult() }

// ClearExitState delegates to the exit controller.
func (cs *CombatService) ClearExitState() { cs.exit.ClearExitState() }

// CheckVictoryCondition delegates to the exit controller.
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	return cs.exit.CheckVictoryCondition()
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
func (cs *CombatService) UpdateThreatLayers() {
	// Update base threat data first
	if cs.threatProvider != nil {
		cs.threatProvider.UpdateAllFactions()
	}

	// Then update composite layers
	for _, evaluator := range cs.layerEvaluators {
		evaluator.Update()
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

// TeardownCombat disposes tactical-side combat entities when returning to exploration.
// Enemy squads must be provided by the encounter service for disposal.
// Player squads are identified via faction membership and stripped of combat-only
// state (faction membership, perk round state, positions, IsDeployed) before
// faction entities are disposed.
// Satisfies combatlifecycle.CombatTeardown via structural typing.
func (cs *CombatService) TeardownCombat(enemySquadIDs []ecs.EntityID) {
	cs.logger.Log("service", 0, "=== Combat Teardown Starting ===")

	// Remove all active effects from player units before leaving combat
	cs.cleanupEffects()

	// Strip combat state from player squads via direct calls into the owning
	// packages. Must happen BEFORE faction entities are disposed:
	// collectPlayerSquadIDs reads FactionMembershipComponent to identify
	// player-controlled squads, and RemoveCombatMembership then removes it.
	for _, squadID := range cs.collectPlayerSquadIDs() {
		entity := cs.EntityManager.FindEntityByID(squadID)
		if entity == nil {
			continue
		}
		combatstate.RemoveCombatMembership(entity)
		perks.RemovePerkRoundState(entity)
		squadcore.ResetSquadDeployment(cs.EntityManager, entity)
	}

	// Build set of enemy squad IDs for unit filtering
	enemySquadSet := make(map[ecs.EntityID]bool)
	for _, id := range enemySquadIDs {
		enemySquadSet[id] = true
	}

	// Dispose all combat entities in one pass
	cs.disposeMatching(combatstate.FactionTag, "factions", nil)
	cs.disposeMatching(combatstate.ActionStateTag, "action states", nil)
	cs.disposeMatching(combatstate.TurnStateTag, "turn states", nil)
	cs.disposeEnemySquads(enemySquadIDs)
	cs.disposeMatching(squadcore.SquadMemberTag, "enemy units", func(entity *ecs.Entity) bool {
		memberData := common.GetComponentType[*squadcore.SquadMemberData](entity, squadcore.SquadMemberComponent)
		return memberData != nil && enemySquadSet[memberData.SquadID]
	})

	cs.logger.Log("service", 0, "=== Combat Teardown Complete ===")
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
		cs.logger.Log("service", 0, fmt.Sprintf("Cleaned up effects from %d units", cleaned))
	}
}

// ================================
// Helper Methods
// ================================

// collectPlayerSquadIDs returns all squads whose faction membership is player-controlled.
// Used internally by TeardownCombat to identify which squads to strip combat-only
// state from (faction membership, perk round state, positions, IsDeployed)
// before faction entities are disposed.
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

// disposeMatching disposes every entity carrying tag that also passes predicate
// (or all of them if predicate is nil). Entities with a PositionComponent are
// removed from the position system before being disposed. Logs the count under
// label.
func (cs *CombatService) disposeMatching(tag ecs.Tag, label string, predicate func(*ecs.Entity) bool) {
	count := 0
	for _, result := range cs.EntityManager.World.Query(tag) {
		entity := result.Entity
		if predicate != nil && !predicate(entity) {
			continue
		}
		pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
		if pos != nil {
			cs.EntityManager.CleanDisposeEntity(entity, pos)
		} else {
			cs.EntityManager.World.DisposeEntities(entity)
		}
		count++
	}
	cs.logger.Log("service", 0, fmt.Sprintf("Disposed %d %s", count, label))
}

// disposeEnemySquads disposes all tracked enemy squads by ID list (rather than
// querying), since their IDs are supplied by the encounter service.
func (cs *CombatService) disposeEnemySquads(enemySquadIDs []ecs.EntityID) {
	for _, squadID := range enemySquadIDs {
		if entity := cs.EntityManager.FindEntityByID(squadID); entity != nil {
			pos := common.GetComponentType[*coords.LogicalPosition](entity, common.PositionComponent)
			cs.EntityManager.CleanDisposeEntity(entity, pos)
		}
	}
	cs.logger.Log("service", 0, fmt.Sprintf("Disposed %d enemy squads", len(enemySquadIDs)))
}
