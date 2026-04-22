package ai

import (
	"game_main/core/common"
	"game_main/mind/behavior"
	"game_main/tactical/combat/combatcore"
	"game_main/tactical/combat/combatservices"
	"game_main/tactical/combat/combatstate"
	"game_main/tactical/squads/squadcore"
	"game_main/tactical/squads/unitdefs"
	"game_main/core/coords"

	"github.com/bytearena/ecs"
)

// AIController orchestrates AI decision-making for computer-controlled factions
type AIController struct {
	entityManager   *common.EntityManager
	turnManager     *combatcore.TurnManager
	movementSystem  *combatcore.CombatMovementSystem
	combatActSystem *combatcore.CombatActionSystem
	combatCache     *combatstate.CombatQueryCache
	threatManager   *behavior.FactionThreatLevelManager
	layerEvaluators map[ecs.EntityID]*behavior.CompositeThreatEvaluator

	// Attack queue for animations (populated during AI turn)
	attackQueue []combatservices.QueuedAttack
}

// CombatAISetup holds all components created by SetupCombatAI.
// Callers inject these into CombatService via its setter methods.
type CombatAISetup struct {
	Controller     combatservices.AITurnController
	ThreatProvider combatservices.ThreatProvider
	EvalFactory    func(factionID ecs.EntityID) combatservices.ThreatLayerEvaluator
}

// SetupCombatAI creates the full AI + threat evaluation stack.
// Returns interfaces that should be injected into CombatService.
// This keeps mind/behavior out of both tactical/combatservices and gui/guicombat.
func SetupCombatAI(
	entityManager *common.EntityManager,
	turnManager *combatcore.TurnManager,
	movementSystem *combatcore.CombatMovementSystem,
	combatActSystem *combatcore.CombatActionSystem,
	combatCache *combatstate.CombatQueryCache,
) *CombatAISetup {
	threatMgr := behavior.NewFactionThreatLevelManager(entityManager, combatCache)
	layerEvaluators := make(map[ecs.EntityID]*behavior.CompositeThreatEvaluator)

	aic := &AIController{
		entityManager:   entityManager,
		turnManager:     turnManager,
		movementSystem:  movementSystem,
		combatActSystem: combatActSystem,
		combatCache:     combatCache,
		threatManager:   threatMgr,
		layerEvaluators: layerEvaluators,
		attackQueue:     make([]combatservices.QueuedAttack, 0),
	}

	evalFactory := func(factionID ecs.EntityID) combatservices.ThreatLayerEvaluator {
		if eval, exists := layerEvaluators[factionID]; exists {
			return eval
		}
		eval := behavior.NewCompositeThreatEvaluator(factionID, entityManager, combatCache, threatMgr)
		layerEvaluators[factionID] = eval
		return eval
	}

	return &CombatAISetup{
		Controller:     aic,
		ThreatProvider: threatMgr,
		EvalFactory:    evalFactory,
	}
}

// DecideFactionTurn executes AI turn for a faction
// Returns true if any actions were executed, false if faction has no actions
func (aic *AIController) DecideFactionTurn(factionID ecs.EntityID) bool {
	// Clear attack queue from previous turn
	aic.attackQueue = aic.attackQueue[:0]

	// TODO: AI spell casting - enemy commanders don't cast spells yet.
	// When implemented: check if faction has a commander with mana/spells,
	// evaluate spell value vs saving mana, pick target, call spells.ExecuteSpellCast.

	// Update threat layers at start of AI turn
	aic.updateThreatLayers()

	// Get all alive squads in faction
	aliveSquads := combatstate.GetActiveSquadsForFaction(factionID, aic.entityManager)

	if len(aliveSquads) == 0 {
		return false
	}

	actionExecuted := false

	// Process each squad - execute ALL available actions per squad
	for _, squadID := range aliveSquads {
		// Keep executing actions for this squad until it has no actions remaining
		for {
			// Get current action state
			actionState := aic.combatCache.FindActionStateBySquadID(squadID)
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
		}
	}

	return actionExecuted
}

// updateThreatLayers updates all threat layers
func (aic *AIController) updateThreatLayers() {
	// Update base threat data first
	aic.threatManager.UpdateAllFactions()

	// Then update composite layers
	for _, evaluator := range aic.layerEvaluators {
		evaluator.Update()
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
	return bestAction.Action.Execute(aic.entityManager, aic.movementSystem, aic.combatActSystem, aic.combatCache)
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
	ActionState *combatstate.ActionStateData

	// Threat evaluation
	ThreatEval *behavior.CompositeThreatEvaluator

	// Systems access
	Manager        *common.EntityManager
	MovementSystem *combatcore.CombatMovementSystem // For validating movement tiles
	AIController   *AIController                    // Reference to AI controller for attack queueing

	// Cached squad info
	SquadRole  unitdefs.UnitRole
	CurrentPos coords.LogicalPosition
}

// NewActionContext creates a new action context for a squad
func NewActionContext(
	squadID ecs.EntityID,
	aic *AIController,
) ActionContext {
	factionID := combatstate.GetSquadFaction(squadID, aic.entityManager)

	// Get or create threat evaluator for faction
	evaluator := aic.getThreatEvaluator(factionID)

	ctx := ActionContext{
		SquadID:        squadID,
		FactionID:      factionID,
		ActionState:    aic.combatCache.FindActionStateBySquadID(squadID),
		ThreatEval:     evaluator,
		Manager:        aic.entityManager,
		MovementSystem: aic.movementSystem, // For validating movement tiles
		AIController:   aic,                // Pass reference for attack queueing
		SquadRole:      squadcore.GetSquadPrimaryRole(squadID, aic.entityManager),
	}

	// Get current position
	if pos, err := combatstate.GetSquadMapPosition(squadID, aic.entityManager); err == nil {
		ctx.CurrentPos = pos
	}

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
	aic.attackQueue = append(aic.attackQueue, combatservices.QueuedAttack{
		AttackerID: attackerID,
		DefenderID: defenderID,
	})
}

// GetQueuedAttacks returns all queued attacks for animation
func (aic *AIController) GetQueuedAttacks() []combatservices.QueuedAttack {
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
