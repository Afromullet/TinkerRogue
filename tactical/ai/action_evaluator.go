package ai

import (
	"fmt"
	"game_main/common"
	"game_main/tactical/combat"
	"game_main/tactical/squads"
	"game_main/tactical/squadcommands"
	"game_main/world/coords"

	"github.com/bytearena/ecs"
)

// ScoredAction represents a possible action with its utility score
type ScoredAction struct {
	Action      SquadAction
	Score       float64
	Description string
}

// SquadAction interface for executable actions
type SquadAction interface {
	Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool
	GetDescription() string
}

// MoveAction represents a movement decision
type MoveAction struct {
	squadID ecs.EntityID
	target  coords.LogicalPosition
}

func (ma *MoveAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	cmd := squadcommands.NewMoveSquadCommand(
		manager,
		movementSystem,
		ma.squadID,
		ma.target,
	)

	err := cmd.Execute()
	return err == nil
}

func (ma *MoveAction) GetDescription() string {
	return fmt.Sprintf("Move to %v", ma.target)
}

// AttackAction represents an attack decision
type AttackAction struct {
	attackerID ecs.EntityID
	targetID   ecs.EntityID
}

func (aa *AttackAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	cmd := squadcommands.NewAttackCommand(
		manager,
		combatActSystem,
		aa.attackerID,
		aa.targetID,
	)

	err := cmd.Execute()
	return err == nil
}

func (aa *AttackAction) GetDescription() string {
	return fmt.Sprintf("Attack squad %d", aa.targetID)
}

// WaitAction represents doing nothing (skip turn)
type WaitAction struct {
	squadID ecs.EntityID
}

func (wa *WaitAction) Execute(manager *common.EntityManager, movementSystem *combat.CombatMovementSystem, combatActSystem *combat.CombatActionSystem) bool {
	// Just mark the squad as having acted
	// This ensures we don't get stuck in infinite loops
	return true
}

func (wa *WaitAction) GetDescription() string {
	return "Wait"
}

// ActionEvaluator generates and scores possible actions
type ActionEvaluator struct {
	ctx ActionContext
}

// NewActionEvaluator creates a new action evaluator
func NewActionEvaluator(ctx ActionContext) *ActionEvaluator {
	return &ActionEvaluator{ctx: ctx}
}

// EvaluateAllActions generates all possible actions and scores them
func (ae *ActionEvaluator) EvaluateAllActions() []ScoredAction {
	var actions []ScoredAction

	// Generate movement actions if haven't moved
	if !ae.ctx.ActionState.HasMoved {
		actions = append(actions, ae.evaluateMovement()...)
	}

	// Generate attack actions if haven't acted
	if !ae.ctx.ActionState.HasActed {
		actions = append(actions, ae.evaluateAttacks()...)
	}

	// Always have wait as fallback
	actions = append(actions, ScoredAction{
		Action:      &WaitAction{squadID: ae.ctx.SquadID},
		Score:       0.0, // Lowest priority
		Description: "Wait (fallback)",
	})

	return actions
}

// evaluateMovement generates and scores movement actions
func (ae *ActionEvaluator) evaluateMovement() []ScoredAction {
	var actions []ScoredAction

	// Get valid movement tiles
	validTiles := ae.getValidMovementTiles()

	if len(validTiles) == 0 {
		return actions
	}

	// Score each position based on role
	for _, pos := range validTiles {
		score := ae.scoreMovementPosition(pos)

		actions = append(actions, ScoredAction{
			Action:      &MoveAction{squadID: ae.ctx.SquadID, target: pos},
			Score:       score,
			Description: fmt.Sprintf("Move to %v (role-aware)", pos),
		})
	}

	return actions
}

// getValidMovementTiles returns all tiles the squad can move to
func (ae *ActionEvaluator) getValidMovementTiles() []coords.LogicalPosition {
	// This is a simplified version - in reality, you'd call the movement system
	// For now, return positions within movement range
	var tiles []coords.LogicalPosition

	moveSpeed := squads.GetSquadMovementSpeed(ae.ctx.SquadID, ae.ctx.Manager)

	for dx := -moveSpeed; dx <= moveSpeed; dx++ {
		for dy := -moveSpeed; dy <= moveSpeed; dy++ {
			pos := coords.LogicalPosition{
				X: ae.ctx.CurrentPos.X + dx,
				Y: ae.ctx.CurrentPos.Y + dy,
			}

			distance := ae.ctx.CurrentPos.ChebyshevDistance(&pos)
			if distance > 0 && distance <= moveSpeed {
				tiles = append(tiles, pos)
			}
		}
	}

	return tiles
}

// scoreMovementPosition scores a movement position based on role and threat
func (ae *ActionEvaluator) scoreMovementPosition(pos coords.LogicalPosition) float64 {
	// Use threat evaluator to get role-weighted threat
	// Lower threat = better position (we want to minimize threat)
	// So we invert it for scoring
	threat := ae.ctx.ThreatEval.GetRoleWeightedThreat(ae.ctx.SquadID, pos)

	// Convert threat to score (lower threat = higher score)
	// Base score of 50, reduced by threat
	baseScore := 50.0
	score := baseScore - threat

	// Bonus for staying near allies (avoid isolation)
	supportLayer := ae.ctx.ThreatEval.GetSupportLayer()
	allyProximity := supportLayer.GetAllyProximityAt(pos)
	score += float64(allyProximity) * 5.0

	return score
}

// evaluateAttacks generates and scores attack actions
func (ae *ActionEvaluator) evaluateAttacks() []ScoredAction {
	var actions []ScoredAction

	// Get all enemy squads in range
	targets := ae.getAttackableTargets()

	for _, targetID := range targets {
		score := ae.scoreAttackTarget(targetID)

		actions = append(actions, ScoredAction{
			Action:      &AttackAction{attackerID: ae.ctx.SquadID, targetID: targetID},
			Score:       score,
			Description: fmt.Sprintf("Attack squad %d", targetID),
		})
	}

	return actions
}

// getAttackableTargets returns all enemy squads we can attack
func (ae *ActionEvaluator) getAttackableTargets() []ecs.EntityID {
	var targets []ecs.EntityID

	// Get all enemy factions
	allFactions := combat.GetAllFactions(ae.ctx.Manager)

	for _, factionID := range allFactions {
		if factionID == ae.ctx.FactionID {
			continue // Skip own faction
		}

		squadIDs := combat.GetSquadsForFaction(factionID, ae.ctx.Manager)

		for _, squadID := range squadIDs {
			if squads.IsSquadDestroyed(squadID, ae.ctx.Manager) {
				continue
			}

			// Check if squad is in attack range
			if ae.isInAttackRange(squadID) {
				targets = append(targets, squadID)
			}
		}
	}

	return targets
}

// isInAttackRange checks if target is in attack range
func (ae *ActionEvaluator) isInAttackRange(targetID ecs.EntityID) bool {
	targetPos, err := combat.GetSquadMapPosition(targetID, ae.ctx.Manager)
	if err != nil {
		return false
	}

	distance := ae.ctx.CurrentPos.ChebyshevDistance(&targetPos)

	// Get max attack range from squad's units
	maxRange := ae.getMaxAttackRange()

	return distance <= maxRange
}

// getMaxAttackRange returns maximum attack range of squad's units
func (ae *ActionEvaluator) getMaxAttackRange() int {
	unitIDs := squads.GetUnitIDsInSquad(ae.ctx.SquadID, ae.ctx.Manager)

	maxRange := 1 // Default melee range

	for _, unitID := range unitIDs {
		entity := ae.ctx.Manager.FindEntityByID(unitID)
		if entity == nil {
			continue
		}

		rangeData := common.GetComponentType[*squads.AttackRangeData](entity, squads.AttackRangeComponent)
		if rangeData != nil {
			if rangeData.Range > maxRange {
				maxRange = rangeData.Range
			}
		}
	}

	return maxRange
}

// scoreAttackTarget scores an attack target
func (ae *ActionEvaluator) scoreAttackTarget(targetID ecs.EntityID) float64 {
	baseScore := 50.0

	// Prioritize wounded targets (focus fire)
	targetHealth := squads.GetSquadHealthPercent(targetID, ae.ctx.Manager)
	baseScore += (1.0 - targetHealth) * 20.0 // Up to +20 for very wounded targets

	// Prioritize high-threat targets
	targetRole := squads.GetSquadPrimaryRole(targetID, ae.ctx.Manager)
	if targetRole == squads.RoleDPS {
		baseScore += 15.0 // DPS are high priority
	} else if targetRole == squads.RoleSupport {
		baseScore += 10.0 // Support are medium priority
	}

	// Bonus for role counters (DPS good vs Support, Tank good vs DPS)
	if ae.ctx.SquadRole == squads.RoleDPS && targetRole == squads.RoleSupport {
		baseScore += 10.0
	} else if ae.ctx.SquadRole == squads.RoleTank && targetRole == squads.RoleDPS {
		baseScore += 10.0
	}

	return baseScore
}
