package ai

import (
	"game_main/common"
	"game_main/tactical/behavior"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// QueuedAttack represents an attack that should be animated
type QueuedAttack struct {
	AttackerID ecs.EntityID
	DefenderID ecs.EntityID
}

// AIController orchestrates AI decision-making for computer-controlled factions
type AIController struct {
	entityManager     *common.EntityManager
	turnManager       *combat.TurnManager
	movementSystem    *combat.CombatMovementSystem
	combatActSystem   *combat.CombatActionSystem
	combatCache       *combat.CombatQueryCache
	threatManager     *behavior.FactionThreatLevelManager
	layerEvaluators   map[ecs.EntityID]*behavior.CompositeThreatEvaluator

	// Attack queue for animations (populated during AI turn)
	attackQueue []QueuedAttack
}

// NewAIController creates a new AI controller
func NewAIController(
	entityManager *common.EntityManager,
	turnManager *combat.TurnManager,
	movementSystem *combat.CombatMovementSystem,
	combatActSystem *combat.CombatActionSystem,
	combatCache *combat.CombatQueryCache,
	threatManager *behavior.FactionThreatLevelManager,
	layerEvaluators map[ecs.EntityID]*behavior.CompositeThreatEvaluator,
) *AIController {
	return &AIController{
		entityManager:   entityManager,
		turnManager:     turnManager,
		movementSystem:  movementSystem,
		combatActSystem: combatActSystem,
		combatCache:     combatCache,
		threatManager:   threatManager,
		layerEvaluators: layerEvaluators,
		attackQueue:     make([]QueuedAttack, 0),
	}
}

// DecideFactionTurn executes AI turn for a faction
// Returns true if any actions were executed, false if faction has no actions
func (aic *AIController) DecideFactionTurn(factionID ecs.EntityID) bool {
	// Clear attack queue from previous turn
	aic.attackQueue = aic.attackQueue[:0]

	// Update threat layers at start of AI turn
	currentRound := aic.turnManager.GetCurrentRound()
	aic.updateThreatLayers(currentRound)

	// Get all alive squads in faction
	squadIDs := combat.GetSquadsForFaction(factionID, aic.entityManager)
	aliveSquads := []ecs.EntityID{}
	for _, squadID := range squadIDs {
		if !squads.IsSquadDestroyed(squadID, aic.entityManager) {
			aliveSquads = append(aliveSquads, squadID)
		}
	}

	if len(aliveSquads) == 0 {
		return false
	}

	actionExecuted := false

	// Process each squad - execute ALL available actions per squad
	for _, squadID := range aliveSquads {
		// Keep executing actions for this squad until it has no actions remaining
		for {
			// Get current action state
			actionState := aic.combatCache.FindActionStateBySquadID(squadID, aic.entityManager)
			if actionState == nil {
				break
			}

			// Stop if squad has no actions remaining
			if actionState.HasMoved && actionState.HasActed {
				break
			}

			// Create action context
			ctx := NewActionContext(squadID, aic)

			// Decide and execute best action
			// If no valid action found, stop processing this squad
			if !aic.executeSquadAction(ctx) {
				break
			}

			actionExecuted = true

			// Mark threat layers dirty after each action (positions changed)
			for _, evaluator := range aic.layerEvaluators {
				evaluator.MarkDirty()
			}
		}
	}

	return actionExecuted
}

// updateThreatLayers updates all threat layers
func (aic *AIController) updateThreatLayers(currentRound int) {
	// Update base threat data first
	aic.threatManager.UpdateAllFactions()

	// Then update composite layers
	for _, evaluator := range aic.layerEvaluators {
		evaluator.Update(currentRound)
	}
}

// executeSquadAction decides and executes the best action for a squad
// Returns true if an action was executed
func (aic *AIController) executeSquadAction(ctx ActionContext) bool {
	// Create action evaluator
	evaluator := NewActionEvaluator(ctx)

	// Generate all possible actions with scores
	actions := evaluator.EvaluateAllActions()

	if len(actions) == 0 {
		return false
	}

	// Select best action
	bestAction := SelectBestAction(actions)
	if bestAction == nil {
		return false
	}

	// Execute the action
	return bestAction.Action.Execute(aic.entityManager, aic.movementSystem, aic.combatActSystem)
}

// SelectBestAction picks the highest-scoring action
func SelectBestAction(actions []ScoredAction) *ScoredAction {
	if len(actions) == 0 {
		return nil
	}

	bestAction := &actions[0]
	for i := range actions {
		if actions[i].Score > bestAction.Score {
			bestAction = &actions[i]
		}
	}

	return bestAction
}

// ActionContext provides context for action evaluation
type ActionContext struct {
	SquadID     ecs.EntityID
	FactionID   ecs.EntityID
	ActionState *combat.ActionStateData

	// Threat evaluation
	ThreatEval *behavior.CompositeThreatEvaluator

	// Systems access
	Manager        *common.EntityManager
	MovementSystem *combat.CombatMovementSystem // For validating movement tiles
	AIController   *AIController                // Reference to AI controller for attack queueing

	// Cached squad info
	SquadRole   squads.UnitRole
	CurrentPos  coords.LogicalPosition
	SquadHealth float64 // Average HP percentage (0-1)
}

// NewActionContext creates a new action context for a squad
func NewActionContext(
	squadID ecs.EntityID,
	aic *AIController,
) ActionContext {
	factionID := combat.GetSquadFaction(squadID, aic.entityManager)

	// Get or create threat evaluator for faction
	evaluator := aic.getThreatEvaluator(factionID)

	ctx := ActionContext{
		SquadID:        squadID,
		FactionID:      factionID,
		ActionState:    aic.combatCache.FindActionStateBySquadID(squadID, aic.entityManager),
		ThreatEval:     evaluator,
		Manager:        aic.entityManager,
		MovementSystem: aic.movementSystem, // For validating movement tiles
		AIController:   aic,                // Pass reference for attack queueing
		SquadRole:      squads.GetSquadPrimaryRole(squadID, aic.entityManager),
	}

	// Get current position
	if pos, err := combat.GetSquadMapPosition(squadID, aic.entityManager); err == nil {
		ctx.CurrentPos = pos
	}

	// Calculate squad health using centralized function
	ctx.SquadHealth = squads.GetSquadHealthPercent(squadID, aic.entityManager)

	return ctx
}

// getThreatEvaluator returns evaluator for faction (uses existing or creates new)
func (aic *AIController) getThreatEvaluator(factionID ecs.EntityID) *behavior.CompositeThreatEvaluator {
	if evaluator, exists := aic.layerEvaluators[factionID]; exists {
		return evaluator
	}

	// Create new evaluator for this faction
	evaluator := behavior.NewCompositeThreatEvaluator(
		factionID,
		aic.entityManager,
		aic.combatCache,
		aic.threatManager,
	)
	aic.layerEvaluators[factionID] = evaluator
	return evaluator
}

// QueueAttack adds an attack to the animation queue
func (aic *AIController) QueueAttack(attackerID, defenderID ecs.EntityID) {
	aic.attackQueue = append(aic.attackQueue, QueuedAttack{
		AttackerID: attackerID,
		DefenderID: defenderID,
	})
}

// GetQueuedAttacks returns all queued attacks for animation
func (aic *AIController) GetQueuedAttacks() []QueuedAttack {
	return aic.attackQueue
}

// HasQueuedAttacks returns true if there are attacks waiting for animation
func (aic *AIController) HasQueuedAttacks() bool {
	return len(aic.attackQueue) > 0
}

// ClearAttackQueue clears all queued attacks
func (aic *AIController) ClearAttackQueue() {
	aic.attackQueue = aic.attackQueue[:0]
}

// NOTE: getSquadPrimaryRole and calculateSquadHealthPercent have been moved to
// squads.GetSquadPrimaryRole() and squads.GetSquadHealthPercent() respectively.
// These centralized functions eliminate code duplication across ai and behavior packages.
