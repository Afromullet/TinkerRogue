package combatservices

import (
	"fmt"
	"game_main/common"
	"game_main/world/coords"
	"game_main/tactical/ai"
	"game_main/tactical/behavior"
	"game_main/tactical/combat"
	"game_main/tactical/combat/battlelog"
	"game_main/tactical/squads"

	"github.com/bytearena/ecs"
)

// Type aliases for GUI layer convenience
type (
	AIController  = ai.AIController
	QueuedAttack  = ai.QueuedAttack
)

// CombatService encapsulates all combat game logic and system ownership
type CombatService struct {
	EntityManager   *common.EntityManager
	TurnManager     *combat.TurnManager
	FactionManager  *combat.FactionManager
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
}

// NewCombatService creates a new combat service
func NewCombatService(manager *common.EntityManager) *CombatService {
	cache := combat.NewCombatQueryCache(manager)
	battleRecorder := battlelog.NewBattleRecorder()
	combatActSystem := combat.NewCombatActionSystem(manager)

	// Wire up battle recorder to combat action system
	combatActSystem.SetBattleRecorder(battleRecorder)

	return &CombatService{
		EntityManager:   manager,
		TurnManager:     combat.NewTurnManager(manager),
		FactionManager:  combat.NewFactionManager(manager),
		MovementSystem:  combat.NewMovementSystem(manager, common.GlobalPositionSystem),
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
	// These are squads deployed via SquadDeploymentMode that have positions but no CombatFactionComponent
	if playerFactionID != 0 {
		cs.assignDeployedSquadsToPlayerFaction(playerFactionID)
	}

	return cs.TurnManager.InitializeCombat(factionIDs)
}

// assignDeployedSquadsToPlayerFaction finds all squads with positions but no CombatFactionComponent
// and assigns them to the player faction. These are squads that were deployed via SquadDeploymentMode.
// TODO: Assinging unassigned squads to the player faction is a temporary fix. Squads will have to be assigned to the
// Correct Faction. There can be multiple players
func (cs *CombatService) assignDeployedSquadsToPlayerFaction(playerFactionID ecs.EntityID) {
	for _, result := range cs.EntityManager.World.Query(squads.SquadTag) {
		squadEntity := result.Entity
		squadID := squadEntity.GetID()

		// Check if squad already has a faction (skip if it does)
		combatFaction := common.GetComponentType[*combat.CombatFactionData](squadEntity, combat.CombatFactionComponent)
		if combatFaction != nil {
			continue // Already assigned to a faction
		}

		// Check if squad has a position (deployed squads have positions)
		position := common.GetComponentType[*coords.LogicalPosition](squadEntity, common.PositionComponent)
		if position == nil {
			continue // No position, not a deployed squad
		}

		// Squad is unassigned and deployed - add it to player faction
		fm := combat.NewFactionManager(cs.EntityManager)
		if err := fm.AddSquadToFaction(playerFactionID, squadID, *position); err != nil {
			// Log error but continue with other squads
			continue
		}
	}
}

// GetAliveSquadsInFaction returns all alive squads for a faction
func (cs *CombatService) GetAliveSquadsInFaction(factionID ecs.EntityID) []ecs.EntityID {
	squadIDs := cs.FactionManager.GetFactionSquads(factionID)
	result := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, cs.EntityManager) {
			result = append(result, squadID)
		}
	}
	return result
}

// VictoryCheckResult contains battle outcome information.
type VictoryCheckResult struct {
	BattleOver       bool
	VictorFaction    ecs.EntityID
	VictorName       string
	DefeatedFactions []ecs.EntityID
	RoundsCompleted  int
}

// CheckVictoryCondition checks if battle has ended
// TODO: Add actual victory conditions
func (cs *CombatService) CheckVictoryCondition() *VictoryCheckResult {
	result := &VictoryCheckResult{
		RoundsCompleted: cs.TurnManager.GetCurrentRound(),
	}

	// Count alive squads per faction
	aliveByFaction := make(map[ecs.EntityID]int)

	for _, queryResult := range cs.EntityManager.World.Query(combat.FactionTag) {
		entity := queryResult.Entity
		factionID := entity.GetID()
		aliveByFaction[factionID] = 0
	}

	// Count squads
	for _, queryResult := range cs.EntityManager.World.Query(squads.SquadTag) {
		entity := queryResult.Entity
		squadData := common.GetComponentType[*squads.SquadData](entity, squads.SquadComponent)
		if squadData != nil && !squads.IsSquadDestroyed(entity.GetID(), cs.EntityManager) {
			factionData := common.GetComponentTypeByID[*combat.FactionData](
				cs.EntityManager, entity.GetID(), combat.FactionComponent)
			if factionData != nil {
				aliveByFaction[factionData.FactionID]++
			}
		}
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

		// Get faction data to include player name
		factionData := cs.CombatCache.FindFactionDataByID(victorFaction, cs.EntityManager)
		if factionData != nil {
			if factionData.PlayerID > 0 {
				// Player victory - include player name
				result.VictorName = fmt.Sprintf("%s (%s)", factionData.Name, factionData.PlayerName)
			} else {
				// AI victory
				result.VictorName = factionData.Name
			}
		} else {
			result.VictorName = "Unknown"
		}
	}

	return result
}

// getSquadNameByID is a helper to get squad name from ID
func getSquadNameByID(squadID ecs.EntityID, manager *common.EntityManager) string {
	squadData := common.GetComponentTypeByID[*squads.SquadData](manager, squadID, squads.SquadComponent)
	if squadData != nil {
		return squadData.Name
	}
	return "Unknown"
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
